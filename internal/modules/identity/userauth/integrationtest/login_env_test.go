package integrationtest

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/dujiao-next/internal/config"
	"github.com/dujiao-next/internal/constants"
	emailverificationdomain "github.com/dujiao-next/internal/modules/identity/emailverification/domain"
	emailverificationstore "github.com/dujiao-next/internal/modules/identity/emailverification/infrastructure/gormstore"
	externalidentitydomain "github.com/dujiao-next/internal/modules/identity/externalidentity/domain"
	externalidentitystore "github.com/dujiao-next/internal/modules/identity/externalidentity/infrastructure/gormstore"
	"github.com/dujiao-next/internal/modules/identity/jwttoken"
	userdomain "github.com/dujiao-next/internal/modules/identity/user/domain"
	userstore "github.com/dujiao-next/internal/modules/identity/user/infrastructure/gormstore"
	userauthapp "github.com/dujiao-next/internal/modules/identity/userauth/application"
	userauthhttp "github.com/dujiao-next/internal/modules/identity/userauth/transport/http"
	settingsapp "github.com/dujiao-next/internal/modules/settings/application"
	"github.com/dujiao-next/internal/shared/mailbrand"
	"github.com/dujiao-next/internal/testkit/memorysettings"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type recordedEmail struct {
	to      string
	code    string
	purpose string
}

type stubVerifyEmailSender struct {
	sent []recordedEmail
}

func (s *stubVerifyEmailSender) SendVerifyCode(toEmail, code, purpose, _ string, _ mailbrand.Brand) error {
	s.sent = append(s.sent, recordedEmail{to: toEmail, code: code, purpose: purpose})
	return nil
}

type stubLoginHistory struct {
	entries []userauthapp.LoginHistoryEntry
}

func (s *stubLoginHistory) ListRecentUserLogins(uint, int) ([]userauthapp.LoginHistoryEntry, error) {
	return s.entries, nil
}

func setupLoginEnvService(t *testing.T) (*userauthapp.Service, *stubVerifyEmailSender, *userstore.Store, *emailverificationstore.Store) {
	t.Helper()
	dsn := fmt.Sprintf("file:login_env_%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	if err := db.AutoMigrate(&userdomain.User{}, &externalidentitydomain.Identity{}, &emailverificationdomain.Code{}); err != nil {
		t.Fatalf("migrate failed: %v", err)
	}
	uStore := userstore.New(db)
	eStore := externalidentitystore.New(db)
	cStore := emailverificationstore.New(db)
	settingSvc := settingsapp.NewService(memorysettings.New())
	sender := &stubVerifyEmailSender{}
	cfg := &config.Config{
		UserJWT: config.JWTConfig{
			SecretKey:             "test-user-jwt-secret-key-32-bytes!!",
			ExpireHours:           24,
			RememberMeExpireHours: 168,
		},
		Security: config.SecurityConfig{
			PasswordPolicy: config.PasswordPolicyConfig{MinLength: 8},
		},
		Email: config.EmailConfig{
			VerifyCode: config.VerifyCodeConfig{
				ExpireMinutes:       10,
				SendIntervalSeconds: 60,
				MaxAttempts:         5,
				Length:              6,
			},
		},
	}
	svc := userauthapp.NewService(cfg, uStore, eStore, cStore, settingSvc, sender, nil)
	return svc, sender, uStore, cStore
}

func seedActiveUser(t *testing.T, uStore *userstore.Store, email, password string, withTOTP bool) *userdomain.User {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("bcrypt failed: %v", err)
	}
	now := time.Now()
	u := &userdomain.User{
		Email:           email,
		PasswordHash:    string(hash),
		DisplayName:     "Tester",
		Status:          constants.UserStatusActive,
		EmailVerifiedAt: &now,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if withTOTP {
		u.TOTPEnabledAt = &now
	}
	if err := uStore.Create(u); err != nil {
		t.Fatalf("create user failed: %v", err)
	}
	return u
}

func TestLoginEnvVerificationAndTrustedDeviceLifecycle(t *testing.T) {
	svc, sender, uStore, _ := setupLoginEnvService(t)
	user := seedActiveUser(t, uStore, "alice@example.com", "Password123!", false)
	chromeUA := "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 Chrome/131.0.0.0 Safari/537.36"
	firefoxUA := "Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15; rv:132.0) Gecko/20100101 Firefox/132.0"

	// 1. 新环境登录必须要求邮箱验证码，不签发访问令牌
	res, err := svc.LoginWithContext(context.Background(), userauthapp.LoginContextInput{
		Email:     "alice@example.com",
		Password:  "Password123!",
		ClientIP:  "198.51.100.10",
		UserAgent: chromeUA,
		Locale:    "zh-CN",
	})
	if err != nil {
		t.Fatalf("LoginWithContext failed: %v", err)
	}
	if !res.RequiresEmailVerify || res.Token != "" || res.ChallengeToken == "" {
		t.Fatalf("expected email verification challenge without access token, got %#v", res)
	}
	if len(sender.sent) != 1 || sender.sent[0].purpose != constants.VerifyPurposeLoginEnv {
		t.Fatalf("expected 1 login_env email, got %#v", sender.sent)
	}
	// Challenge token 的 typ 必须是 login_email_challenge，不能冒充 access token
	if parsed, parseErr := svc.ParseUserJWT(res.ChallengeToken); parseErr == nil && jwttoken.IsAccessType(parsed.Typ) {
		t.Fatalf("login email challenge token must not pass IsAccessType")
	}

	// 2. 核验邮箱验证码后签发正式 JWT 与受信任设备 Cookie 令牌
	verified, err := svc.VerifyLoginEmailChallenge(context.Background(), res.ChallengeToken, sender.sent[0].code, chromeUA)
	if err != nil {
		t.Fatalf("VerifyLoginEmailChallenge failed: %v", err)
	}
	if verified.Token == "" || verified.TrustedDeviceToken == "" {
		t.Fatalf("expected access token and trusted device token, got %#v", verified)
	}

	// 3. 同一 challenge 重放必须失败
	if _, err := svc.VerifyLoginEmailChallenge(context.Background(), res.ChallengeToken, sender.sent[0].code, chromeUA); !errors.Is(err, userauthapp.ErrLoginEmailChallengeInvalid) {
		t.Fatalf("replayed challenge got %v, want ErrLoginEmailChallengeInvalid", err)
	}

	// 4. 携带同一浏览器的受信任设备凭证（即便 IP 切换如开关代理）直接密码登录，不再发邮件
	res2, err := svc.LoginWithContext(context.Background(), userauthapp.LoginContextInput{
		Email:              "alice@example.com",
		Password:           "Password123!",
		ClientIP:           "203.0.113.88",
		UserAgent:          chromeUA,
		TrustedDeviceToken: verified.TrustedDeviceToken,
	})
	if err != nil {
		t.Fatalf("trusted device login failed: %v", err)
	}
	if res2.RequiresEmailVerify || res2.Token == "" {
		t.Fatalf("trusted device should sign in directly, got %#v", res2)
	}
	if len(sender.sent) != 1 {
		t.Fatalf("trusted device must not send extra emails, sent=%d", len(sender.sent))
	}

	// 5. 换用 Firefox（环境变化），旧 Chrome 设备凭证不再匹配 → 再次触发邮箱验证
	// 先模拟上次发送时间超过 60s 冷却以便发新码
	res3, err := svc.LoginWithContext(context.Background(), userauthapp.LoginContextInput{
		Email:              "alice@example.com",
		Password:           "Password123!",
		ClientIP:           "198.51.100.10",
		UserAgent:          firefoxUA,
		TrustedDeviceToken: verified.TrustedDeviceToken,
	})
	if err != nil && !errors.Is(err, userauthapp.ErrVerifyCodeTooFrequent) {
		t.Fatalf("unexpected error on browser switch: %v", err)
	}
	if err == nil && !res3.RequiresEmailVerify {
		t.Fatalf("different browser family must require email verification")
	}

	// 6. 改密后 TokenVersion 递增，原 Chrome 信任设备立即失效
	if err := svc.ChangePassword(user.ID, "Password123!", "NewPassword456!"); err != nil {
		t.Fatalf("ChangePassword failed: %v", err)
	}
	reloaded, _ := uStore.GetByID(user.ID)
	if svc.VerifyTrustedDeviceToken(verified.TrustedDeviceToken, reloaded, chromeUA) {
		t.Fatalf("trusted device token must be invalidated after password change")
	}
}

func TestLoginEnvSmoothTransitionForExistingHistoryAndTOTPPriority(t *testing.T) {
	svc, sender, uStore, _ := setupLoginEnvService(t)
	_ = seedActiveUser(t, uStore, "legacy@example.com", "Password123!", false)
	_ = seedActiveUser(t, uStore, "totp@example.com", "Password123!", true)
	chromeUA := "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 Chrome/131.0.0.0 Safari/537.36"

	// 1. 存量老用户无 Cookie，但 30 天内存在同 /24 网段 + 同浏览器家族历史成功登录 → 平滑放行并补发设备 Cookie
	svc.SetLoginHistoryReader(&stubLoginHistory{
		entries: []userauthapp.LoginHistoryEntry{
			{
				Status:    constants.LoginLogStatusSuccess,
				ClientIP:  "198.51.100.12",
				UserAgent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 Chrome/129.0.0.0 Safari/537.36",
				CreatedAt: time.Now().Add(-48 * time.Hour),
			},
		},
	})
	res, err := svc.LoginWithContext(context.Background(), userauthapp.LoginContextInput{
		Email:     "legacy@example.com",
		Password:  "Password123!",
		ClientIP:  "198.51.100.99",
		UserAgent: chromeUA,
	})
	if err != nil {
		t.Fatalf("legacy smooth login failed: %v", err)
	}
	if res.RequiresEmailVerify || res.Token == "" || res.TrustedDeviceToken == "" {
		t.Fatalf("expected smooth sign-in with issued trusted device token, got %#v", res)
	}
	if len(sender.sent) != 0 {
		t.Fatalf("smooth transition must not send email, sent=%d", len(sender.sent))
	}

	// 2. 开启 TOTP 的账号在新环境登录 → 直接走 TOTP，不额外发邮箱验证码
	totpRes, err := svc.LoginWithContext(context.Background(), userauthapp.LoginContextInput{
		Email:     "totp@example.com",
		Password:  "Password123!",
		ClientIP:  "203.0.113.5",
		UserAgent: chromeUA,
	})
	if err != nil {
		t.Fatalf("totp login failed: %v", err)
	}
	if !totpRes.RequiresTOTP || totpRes.RequiresEmailVerify {
		t.Fatalf("totp user should enter TOTP directly without email verify, got %#v", totpRes)
	}
	if len(sender.sent) != 0 {
		t.Fatalf("totp user must not receive login_env email, sent=%d", len(sender.sent))
	}
}

func TestAccountLevelBruteForceLockoutAndHourlyEmailQuota(t *testing.T) {
	svc, _, uStore, cStore := setupLoginEnvService(t)
	_ = seedActiveUser(t, uStore, "victim@example.com", "StrongPass99!", false)

	// 模拟代理池换不同 IP 对同一邮箱连续输错 10 次密码
	for i := 0; i < 10; i++ {
		_, err := svc.LoginWithContext(context.Background(), userauthapp.LoginContextInput{
			Email:    "victim@example.com",
			Password: fmt.Sprintf("WrongPass%d", i),
			ClientIP: fmt.Sprintf("198.51.100.%d", i+1),
		})
		if !errors.Is(err, userauthapp.ErrInvalidCredentials) {
			t.Fatalf("attempt %d got %v want ErrInvalidCredentials", i+1, err)
		}
	}
	// 第 11 次即便换新 IP 且密码正确也必须被账号保护锁拦截
	_, err := svc.LoginWithContext(context.Background(), userauthapp.LoginContextInput{
		Email:    "victim@example.com",
		Password: "StrongPass99!",
		ClientIP: "203.0.113.200",
	})
	if !errors.Is(err, userauthapp.ErrAccountLocked) {
		t.Fatalf("expected ErrAccountLocked after 10 failures across IPs, got %v", err)
	}

	// 同一邮箱 1 小时内已有 8 条发信记录 → 第 9 封必须被配额拦截
	now := time.Now()
	for i := 0; i < 8; i++ {
		if err := cStore.Create(&emailverificationdomain.Code{
			Email:     "quota@example.com",
			Purpose:   constants.VerifyPurposeRegister,
			Code:      fmt.Sprintf("10000%d", i),
			ExpiresAt: now.Add(10 * time.Minute),
			SentAt:    now.Add(-time.Duration(i+2) * time.Minute),
			CreatedAt: now.Add(-time.Duration(i+2) * time.Minute),
		}); err != nil {
			t.Fatalf("seed code failed: %v", err)
		}
	}
	err = svc.SendVerifyCode(context.Background(), "quota@example.com", constants.VerifyPurposeEmailAuth, "zh-CN", "")
	if !errors.Is(err, userauthapp.ErrVerifyCodeTooFrequent) {
		t.Fatalf("expected hourly email quota to return ErrVerifyCodeTooFrequent, got %v", err)
	}
}

type registerNoEmailSettings struct{}

func (registerNoEmailSettings) GetRegistrationEnabled(bool) (bool, error)      { return true, nil }
func (registerNoEmailSettings) GetEmailVerificationEnabled(bool) (bool, error) { return false, nil }

type registerNoEmailAuth struct{}

func (registerNoEmailAuth) Register(context.Context, string, string, string, string, bool, bool) (*userdomain.User, string, time.Time, error) {
	return &userdomain.User{ID: 1, Email: "u@example.com"}, "tok", time.Now().Add(time.Hour), nil
}
func (registerNoEmailAuth) LoginStep1(context.Context, string, string, bool) (*userauthhttp.AuthLoginResult, error) {
	return nil, nil
}

func TestUserRegisterRequiresCaptchaWhenEmailVerificationDisabled(t *testing.T) {
	gin.SetMode(gin.TestMode)
	captcha := &emailAuthCaptcha{}
	handler := userauthhttp.NewUserLoginHandler(registerNoEmailSettings{}, registerNoEmailAuth{}, captcha, nil)
	router := gin.New()
	router.POST("/auth/register", handler.UserRegister)

	req := httptest.NewRequest(http.MethodPost, "/auth/register", strings.NewReader(`{"email":"u@example.com","password":"Password123!","agreement_accepted":true}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	if captcha.scene != constants.CaptchaSceneRegisterSendCode {
		t.Fatalf("register without email verification must invoke captcha scene %q, got %q", constants.CaptchaSceneRegisterSendCode, captcha.scene)
	}
}
