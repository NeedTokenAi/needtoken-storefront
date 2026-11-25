package application

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/dujiao-next/internal/telegramidentity"
	"math/big"
	"net/mail"
	"net/netip"
	"strings"
	"sync"
	"time"

	usercontract "github.com/dujiao-next/internal/modules/identity/user/contract"
	"github.com/dujiao-next/internal/shared/passwordpolicy"

	userdomain "github.com/dujiao-next/internal/modules/identity/user/domain"

	telegramauthapp "github.com/dujiao-next/internal/modules/identity/telegramauth/application"
	settingsapp "github.com/dujiao-next/internal/modules/settings/application"

	"github.com/dujiao-next/internal/cache"
	"github.com/dujiao-next/internal/config"
	"github.com/dujiao-next/internal/constants"
	"github.com/dujiao-next/internal/logger"
	emailverificationcontract "github.com/dujiao-next/internal/modules/identity/emailverification/contract"
	emailverificationdomain "github.com/dujiao-next/internal/modules/identity/emailverification/domain"
	externalidentitycontract "github.com/dujiao-next/internal/modules/identity/externalidentity/contract"
	googleauthapp "github.com/dujiao-next/internal/modules/identity/googleauth/application"
	"github.com/dujiao-next/internal/modules/identity/jwttoken"
	"github.com/dujiao-next/internal/modules/identity/userauth/challenge"
	notificationcontract "github.com/dujiao-next/internal/modules/notification/contract"
	"github.com/dujiao-next/internal/shared/mailbrand"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// Service 用户认证服务
type Service struct {
	cfg                   *config.Config
	userRepo              usercontract.Store
	userOAuthIdentityRepo externalidentitycontract.Store
	codeRepo              emailverificationcontract.Store
	settingService        *settingsapp.Service
	emailService          VerificationEmailSender
	emailBrandResolver    mailbrand.Resolver
	telegramAuthService   *telegramauthapp.Service
	googleAuthService     *googleauthapp.Service
	googleRedirectStore   GoogleRedirectStore
	memberLevelSvc        MemberLevelAssigner
	authUnitOfWork        AuthUnitOfWork
	inviteCodes           InviteCodeChecker
	loginHistory          LoginHistoryReader
	runtimeGuard          *authRuntimeGuard
}

type MemberLevelAssigner interface {
	AssignDefaultLevel(userID uint) error
}

// SetMemberLevelService 设置会员等级服务
func (s *Service) SetMemberLevelService(svc MemberLevelAssigner) {
	s.memberLevelSvc = svc
}

// SetGoogleAuthService injects the runtime-configurable Google credential verifier.
func (s *Service) SetGoogleAuthService(service *googleauthapp.Service) {
	s.googleAuthService = service
}

// SetGoogleRedirectStore injects the Redis-backed single-use state store used
// only by the redirect UX. Popup Google login remains independent of Redis.
func (s *Service) SetGoogleRedirectStore(store GoogleRedirectStore) {
	s.googleRedirectStore = store
}

// SetAuthUnitOfWork injects the transaction boundary shared by user accounts
// and external identities.
func (s *Service) SetAuthUnitOfWork(unitOfWork AuthUnitOfWork) {
	s.authUnitOfWork = unitOfWork
}

// SetEmailBrandResolver enables request-scoped storefront branding for
// verification emails.
func (s *Service) SetEmailBrandResolver(resolver mailbrand.Resolver) {
	s.emailBrandResolver = resolver
}

// NewService 创建用户认证服务
func NewService(
	cfg *config.Config,
	userRepo usercontract.Store,
	userOAuthIdentityRepo externalidentitycontract.Store,
	codeRepo emailverificationcontract.Store,
	settingService *settingsapp.Service,
	emailService VerificationEmailSender,
	telegramAuthService *telegramauthapp.Service,
) *Service {
	return &Service{
		cfg:                   cfg,
		userRepo:              userRepo,
		userOAuthIdentityRepo: userOAuthIdentityRepo,
		codeRepo:              codeRepo,
		settingService:        settingService,
		emailService:          emailService,
		telegramAuthService:   telegramAuthService,
		runtimeGuard:          newAuthRuntimeGuard(),
	}
}

// UserJWTClaims 用户 JWT 声明
type UserJWTClaims struct {
	UserID       uint   `json:"user_id"`
	Email        string `json:"email"`
	TokenVersion uint64 `json:"token_version"`
	Typ          string `json:"typ,omitempty"`
	// ResellerID 签发时所在的分销站点（主站为 0）。令牌只在同一站点有效，
	// 防止分销商在自己的域名上截获的令牌被拿到主站或其他分销站使用。
	ResellerID uint `json:"rid,omitempty"`
	jwt.RegisteredClaims
}

// BindUserTokenTenant 将访问令牌绑定到签发请求所在的分销站点；主站（resellerID=0）原样返回。
func (s *Service) BindUserTokenTenant(tokenString string, resellerID uint) (string, error) {
	if resellerID == 0 || tokenString == "" {
		return tokenString, nil
	}
	claims, err := s.ParseUserJWT(tokenString)
	if err != nil {
		return "", err
	}
	claims.ResellerID = resellerID
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(s.cfg.UserJWT.SecretKey))
}

// UserChallengeClaims 用户 2FA 挑战 token claims
//
// 注：Typ 字段同时占用与 UserJWTClaims 兼容的 typ 键，写入 "2fa_challenge"，
// 防止挑战 token 在被错误地解析为 UserJWTClaims 时通过中间件的 typ 校验。
type UserChallengeClaims struct {
	UserID      uint   `json:"user_id"`
	JTI         string `json:"jti"`
	Purpose     string `json:"purpose"`
	RememberMe  bool   `json:"remember_me"`
	LoginSource string `json:"login_source,omitempty"`
	Typ         string `json:"typ"`
	jwt.RegisteredClaims
}

// UserLoginResult 用户登录第一步结果
type UserLoginResult struct {
	RequiresTOTP        bool
	RequiresEmailVerify bool
	MaskedEmail         string
	TrustedDeviceToken  string
	User                *userdomain.User
	Token               string
	ExpiresAt           time.Time
	ChallengeToken      string
	ChallengeJTI        string
	ChallengeExpiresAt  time.Time
	// InviteRequired 表示第三方身份已验证但需补填邀请码才能建号，InviteTicket 用于提交补全。
	InviteRequired bool
	InviteTicket   string
}

const (
	// EmailChangeModeBindOnly 表示仅需校验新邮箱验证码（用于 Telegram 虚拟邮箱账号）
	EmailChangeModeBindOnly = "bind_only"
	// EmailChangeModeChangeWithOldAndNew 表示需要旧邮箱 + 新邮箱双验证码
	EmailChangeModeChangeWithOldAndNew = "change_with_old_and_new"
	// PasswordChangeModeSetWithoutOld 表示首次设置密码，不需要旧密码
	PasswordChangeModeSetWithoutOld = "set_without_old"
	// PasswordChangeModeChangeWithOld 表示修改密码，需要旧密码
	PasswordChangeModeChangeWithOld = "change_with_old"
)

// GenerateUserJWT 生成用户 JWT Token
func (s *Service) GenerateUserJWT(user *userdomain.User, expireHours int) (string, time.Time, error) {
	resolvedHours := expireHours
	if resolvedHours <= 0 {
		resolvedHours = resolveUserJWTExpireHours(s.cfg.UserJWT)
	}
	expiresAt := time.Now().Add(time.Duration(resolvedHours) * time.Hour)
	claims := UserJWTClaims{
		UserID:       user.ID,
		Email:        user.Email,
		TokenVersion: user.TokenVersion,
		Typ:          jwttoken.TypeAccess,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(s.cfg.UserJWT.SecretKey))
	if err != nil {
		return "", time.Time{}, err
	}
	return tokenString, expiresAt, nil
}

// ParseUserJWT 解析用户 JWT Token
func (s *Service) ParseUserJWT(tokenString string) (*UserJWTClaims, error) {
	parser := jwttoken.NewHS256Parser()
	claims := &UserJWTClaims{}
	token, err := parser.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(s.cfg.UserJWT.SecretKey), nil
	})
	if err != nil {
		return nil, err
	}
	if claims, ok := token.Claims.(*UserJWTClaims); ok && token.Valid {
		return claims, nil
	}
	return nil, errors.New("无效的 token")
}

// SendVerifyCode 发送邮箱验证码；inviteCode 仅在注册用途且开启邀请注册时校验（不核销）。
func (s *Service) SendVerifyCode(ctx context.Context, email, purpose, locale, inviteCode string) error {
	if s.emailService == nil {
		return ErrEmailServiceNotConfigured
	}
	normalized, err := normalizeEmail(email)
	if err != nil {
		return err
	}
	purpose = strings.ToLower(strings.TrimSpace(purpose))
	if !isVerifyPurposeSupported(purpose) {
		return ErrInvalidVerifyPurpose
	}
	if purpose == constants.VerifyPurposeEmailAuth {
		if normalized, err = normalizeUserSuppliedEmail(normalized); err != nil {
			return err
		}
		if address, err := mail.ParseAddress(normalized); err != nil || address.Address != normalized {
			return ErrInvalidEmail
		}
	}

	if purpose == constants.VerifyPurposeRegister {
		policy, err := s.loadRegistrationPolicy(inviteCode)
		if err != nil {
			return err
		}
		// 邀请注册模式下先校验邀请码，避免公开的发信接口被用来向任意邮箱发送注册邮件。
		if err := s.precheckInviteCode(policy); err != nil {
			return err
		}
		if err := settingsapp.CheckRegistrationEmailDomainAllowed(normalized, policy.EmailDomain); err != nil {
			return err
		}
		exist, err := s.userRepo.GetByEmail(normalized)
		if err != nil {
			return err
		}
		if exist != nil {
			return ErrEmailExists
		}
	}

	if purpose == constants.VerifyPurposeReset {
		user, err := s.userRepo.GetByEmail(normalized)
		if err != nil {
			return err
		}
		if user == nil {
			return ErrNotFound
		}
		if strings.TrimSpace(user.Locale) != "" {
			locale = user.Locale
		}
	}

	if purpose == constants.VerifyPurposeTelegramBind {
		// 公开发信入口也必须遵守 Telegram 开关，不能只在登录或绑定时校验。
		setting, err := s.settingService.GetTelegramAuthSetting(s.cfg.TelegramAuth)
		if err != nil {
			return err
		}
		if !setting.Enabled {
			return telegramauthapp.ErrTelegramAuthDisabled
		}
		user, err := s.userRepo.GetByEmail(normalized)
		if err != nil {
			return err
		}
		if user == nil {
			return ErrNotFound
		}
		if strings.TrimSpace(user.Locale) != "" {
			locale = user.Locale
		}
	}

	return s.sendVerifyCode(ctx, normalized, purpose, locale)
}

// EmailAuthResult discloses only the next credential step, after the inbox
// owner has consumed a purpose-bound code. It never grants an access session.
type EmailAuthResult struct {
	NextStep         string
	RegistrationCode string
	ExpiresAt        time.Time
}

func (s *Service) VerifyEmailAuth(ctx context.Context, email, code string) (*EmailAuthResult, error) {
	normalized, err := normalizeUserSuppliedEmail(email)
	if err != nil {
		return nil, err
	}
	if address, err := mail.ParseAddress(normalized); err != nil || address.Address != normalized {
		return nil, ErrInvalidEmail
	}
	if ctx != nil && ctx.Err() != nil {
		return nil, ctx.Err()
	}
	if _, err := s.verifyCode(normalized, constants.VerifyPurposeEmailAuth, code); err != nil {
		return nil, err
	}
	user, err := s.userRepo.GetByEmail(normalized)
	if err != nil {
		return nil, err
	}
	if user != nil {
		// A disabled account receives the same password step. Its native password
		// login still enforces status, verification and second-factor checks.
		return &EmailAuthResult{NextStep: "password"}, nil
	}
	policy, err := s.loadRegistrationPolicy("")
	if err != nil {
		return nil, err
	}
	if !policy.Enabled {
		return nil, ErrRegistrationDisabled
	}
	if err := settingsapp.CheckRegistrationEmailDomainAllowed(normalized, policy.EmailDomain); err != nil {
		return nil, err
	}
	// Invite/password checks remain in Register before consuming this ticket,
	// allowing the owner to correct their form without repeating inbox proof.
	secret := make([]byte, 32)
	if _, err := rand.Read(secret); err != nil {
		return nil, err
	}
	ticket := "eauth_" + base64.RawURLEncoding.EncodeToString(secret)
	now := time.Now()
	expiresAt := now.Add(time.Duration(constants.EmailAuthMaxExpireMinutes) * time.Minute)
	record := &emailverificationdomain.Code{Email: normalized, Purpose: constants.VerifyPurposeRegister, Code: ticket, ExpiresAt: expiresAt, SentAt: now, CreatedAt: now}
	if err := s.codeRepo.Create(record); err != nil {
		return nil, err
	}
	return &EmailAuthResult{NextStep: "register", RegistrationCode: ticket, ExpiresAt: expiresAt}, nil
}

// Register 用户注册
func (s *Service) Register(email, password, code, inviteCode string, agreementAccepted bool, emailVerificationEnabled bool) (*userdomain.User, string, time.Time, error) {
	if !agreementAccepted {
		return nil, "", time.Time{}, ErrAgreementRequired
	}
	normalized, err := normalizeUserSuppliedEmail(email)
	if err != nil {
		return nil, "", time.Time{}, err
	}
	policy, err := s.loadRegistrationPolicy(inviteCode)
	if err != nil {
		return nil, "", time.Time{}, err
	}
	if err := s.precheckInviteCode(policy); err != nil {
		return nil, "", time.Time{}, err
	}
	if err := settingsapp.CheckRegistrationEmailDomainAllowed(normalized, policy.EmailDomain); err != nil {
		return nil, "", time.Time{}, err
	}
	if err := passwordpolicy.Validate(s.cfg.Security.PasswordPolicy.ValidationPolicy(), password); err != nil {
		return nil, "", time.Time{}, err
	}

	exist, err := s.userRepo.GetByEmail(normalized)
	if err != nil {
		return nil, "", time.Time{}, err
	}
	if exist != nil {
		return nil, "", time.Time{}, ErrEmailExists
	}

	emailVerified := false
	if emailVerificationEnabled || strings.TrimSpace(code) != "" {
		if _, err := s.verifyCode(normalized, constants.VerifyPurposeRegister, code); err != nil {
			return nil, "", time.Time{}, err
		}
		emailVerified = true
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, "", time.Time{}, err
	}

	now := time.Now()
	var emailVerifiedAt *time.Time
	if emailVerified {
		emailVerifiedAt = &now
	}
	nickname := resolveNicknameFromEmail(normalized)
	user := &userdomain.User{
		Email:           normalized,
		PasswordHash:    string(hashedPassword),
		DisplayName:     nickname,
		Status:          constants.UserStatusActive,
		EmailVerifiedAt: emailVerifiedAt,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	if err := s.createUserInTransaction(context.Background(), user, policy); err != nil {
		return nil, "", time.Time{}, err
	}

	token, expiresAt, err := s.GenerateUserJWT(user, 0)
	if err != nil {
		return nil, "", time.Time{}, err
	}

	user.LastLoginAt = &now
	if err := s.userRepo.Update(user); err != nil {
		return nil, "", time.Time{}, err
	}
	_ = cache.SetUserAuthState(context.Background(), cache.BuildUserAuthState(user))

	// 分配默认会员等级（必须在最后一次 Update 之后，避免被覆盖）
	if s.memberLevelSvc != nil {
		_ = s.memberLevelSvc.AssignDefaultLevel(user.ID)
	}

	return user, token, expiresAt, nil
}

// LoginStep1 用户密码登录第一步：校验密码，根据是否启用 2FA 返回 challenge token 或正式 JWT。
func (s *Service) LoginStep1(email, password string, rememberMe bool) (*UserLoginResult, error) {
	normalized, err := normalizeEmail(email)
	if err != nil {
		return nil, err
	}
	user, err := s.userRepo.GetByEmail(normalized)
	if err != nil {
		return nil, err
	}
	if user == nil {
		_ = bcrypt.CompareHashAndPassword([]byte("$2a$10$dummyhashtopreventtimingattacksxxxxxxxxxxxxxxxxxx"), []byte(password))
		return nil, ErrInvalidCredentials
	}
	if strings.ToLower(user.Status) != constants.UserStatusActive {
		return nil, ErrUserDisabled
	}
	if user.EmailVerifiedAt == nil {
		emailVerificationEnabled := true
		if s.settingService != nil {
			enabled, err := s.settingService.GetEmailVerificationEnabled(true)
			if err != nil {
				return nil, err
			}
			emailVerificationEnabled = enabled
		}
		if emailVerificationEnabled {
			return nil, ErrEmailNotVerified
		}
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	if user.TOTPEnabledAt != nil {
		challenge, jti, expiresAt, err := s.IssueUserChallengeToken(user.ID, rememberMe)
		if err != nil {
			return nil, err
		}
		return &UserLoginResult{
			RequiresTOTP:       true,
			User:               user,
			ChallengeToken:     challenge,
			ChallengeJTI:       jti,
			ChallengeExpiresAt: expiresAt,
		}, nil
	}

	expireHours := resolveUserJWTExpireHours(s.cfg.UserJWT)
	if rememberMe {
		expireHours = resolveRememberMeExpireHours(s.cfg.UserJWT)
	}
	token, expiresAt, err := s.GenerateUserJWT(user, expireHours)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	user.LastLoginAt = &now
	if err := s.userRepo.Update(user); err != nil {
		return nil, err
	}
	_ = cache.SetUserAuthState(context.Background(), cache.BuildUserAuthState(user))

	return &UserLoginResult{
		RequiresTOTP: false,
		User:         user,
		Token:        token,
		ExpiresAt:    expiresAt,
	}, nil
}

// IssueUserChallengeToken 签发用户 2FA 挑战 token
func (s *Service) IssueUserChallengeToken(userID uint, rememberMe bool) (token, jti string, expiresAt time.Time, err error) {
	return s.IssueUserChallengeTokenForSource(userID, rememberMe, constants.LoginLogSourceWeb)
}

// IssueUserChallengeTokenForSource signs a 2FA challenge while retaining the
// original login provider for the completion audit log.
func (s *Service) IssueUserChallengeTokenForSource(userID uint, rememberMe bool, loginSource string) (token, jti string, expiresAt time.Time, err error) {
	jti = uuid.NewString()
	expiresAt = time.Now().Add(challenge.TTL)
	claims := UserChallengeClaims{
		UserID:      userID,
		JTI:         jti,
		Purpose:     challenge.PurposeTwoFactor,
		RememberMe:  rememberMe,
		LoginSource: normalizeLoginSource(loginSource),
		Typ:         jwttoken.TypeTwoFactorChallenge,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			ID:        jti,
		},
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := t.SignedString([]byte(s.cfg.UserJWT.SecretKey))
	if err != nil {
		return "", "", time.Time{}, err
	}
	return signed, jti, expiresAt, nil
}

func normalizeLoginSource(source string) string {
	switch strings.ToLower(strings.TrimSpace(source)) {
	case constants.LoginLogSourceGoogle:
		return constants.LoginLogSourceGoogle
	case constants.LoginLogSourceTelegram:
		return constants.LoginLogSourceTelegram
	default:
		return constants.LoginLogSourceWeb
	}
}

// ParseUserChallengeToken 解析并校验用户挑战 token
func (s *Service) ParseUserChallengeToken(tokenString string) (*UserChallengeClaims, error) {
	parser := jwttoken.NewHS256Parser()
	tok, err := parser.ParseWithClaims(tokenString, &UserChallengeClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(s.cfg.UserJWT.SecretKey), nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := tok.Claims.(*UserChallengeClaims)
	if !ok || !tok.Valid {
		return nil, errors.New("invalid challenge token")
	}
	if claims.Purpose != challenge.PurposeTwoFactor || claims.Typ != jwttoken.TypeTwoFactorChallenge {
		return nil, errors.New("invalid challenge purpose")
	}
	return claims, nil
}

// CompleteLoginAfter2FA 用户 2FA 验证通过后完成登录：发正式 JWT、更新 last_login
func (s *Service) CompleteLoginAfter2FA(userID uint, rememberMe bool) (*UserLoginResult, error) {
	user, err := s.userRepo.GetByID(userID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrNotFound
	}
	expireHours := resolveUserJWTExpireHours(s.cfg.UserJWT)
	if rememberMe {
		expireHours = resolveRememberMeExpireHours(s.cfg.UserJWT)
	}
	token, expiresAt, err := s.GenerateUserJWT(user, expireHours)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	user.LastLoginAt = &now
	if err := s.userRepo.Update(user); err != nil {
		return nil, err
	}
	_ = cache.SetUserAuthState(context.Background(), cache.BuildUserAuthState(user))
	return &UserLoginResult{RequiresTOTP: false, User: user, Token: token, ExpiresAt: expiresAt}, nil
}

func (s *Service) verifyCode(email, purpose, code string) (*emailverificationdomain.Code, error) {
	record, err := s.codeRepo.GetLatest(email, purpose)
	if err != nil {
		return nil, err
	}
	if record == nil {
		return nil, ErrVerifyCodeInvalid
	}
	if record.VerifiedAt != nil {
		return nil, ErrVerifyCodeInvalid
	}

	now := time.Now()
	if record.ExpiresAt.Before(now) {
		return nil, ErrVerifyCodeExpired
	}

	// 先原子占用一次尝试机会再比对：并发猜测请求无法共享同一个"剩余次数"。
	maxAttempts := resolveMaxAttempts(s.cfg.Email.VerifyCode)
	if (purpose == constants.VerifyPurposeEmailAuth || purpose == constants.VerifyPurposeLoginEnv) && maxAttempts > 5 {
		maxAttempts = 5
	}
	reserved, err := s.codeRepo.ReserveAttempt(record.ID, maxAttempts)
	if err != nil {
		return nil, err
	}
	if !reserved {
		return nil, ErrVerifyCodeAttemptsExceeded
	}

	if subtle.ConstantTimeCompare([]byte(strings.TrimSpace(record.Code)), []byte(strings.TrimSpace(code))) != 1 {
		return nil, ErrVerifyCodeInvalid
	}

	marked, err := s.codeRepo.MarkVerified(record.ID, now)
	if err != nil {
		return nil, err
	}
	if !marked {
		return nil, ErrVerifyCodeInvalid
	}
	return record, nil
}

func (s *Service) sendVerifyCode(ctx context.Context, email, purpose, locale string) error {
	unlock, locked := s.acquireVerifyCodeSendLock(ctx, email, purpose)
	if !locked {
		return ErrVerifyCodeTooFrequent
	}
	defer unlock()

	latest, err := s.codeRepo.GetLatest(email, purpose)
	if err != nil {
		return err
	}
	now := time.Now()
	if latest != nil {
		interval := time.Duration(resolveSendIntervalSeconds(s.cfg.Email.VerifyCode)) * time.Second
		if !latest.SentAt.IsZero() && now.Sub(latest.SentAt) < interval {
			return ErrVerifyCodeTooFrequent
		}
	}
	// 单邮箱小时/天发送配额，防止邮件轰炸与 SMTP 额度耗尽。
	if hourCount, err := s.codeRepo.CountSentSince(email, now.Add(-time.Hour)); err == nil && hourCount >= verifyCodeMaxPerHour {
		return ErrVerifyCodeTooFrequent
	}
	if dayCount, err := s.codeRepo.CountSentSince(email, now.Add(-24*time.Hour)); err == nil && dayCount >= verifyCodeMaxPerDay {
		return ErrVerifyCodeTooFrequent
	}

	codeLength := resolveCodeLength(s.cfg.Email.VerifyCode)
	if (purpose == constants.VerifyPurposeEmailAuth || purpose == constants.VerifyPurposeLoginEnv) && codeLength < 6 {
		codeLength = 6
	}
	code, err := randomNumericCode(codeLength)
	if err != nil {
		return err
	}
	brand := mailbrand.Brand{}
	if s.emailBrandResolver != nil {
		brand, err = s.emailBrandResolver.ResolveEmailBrand(ctx, mailbrand.Scope{})
		if err != nil {
			return err
		}
	}

	expireMinutes := resolveExpireMinutes(s.cfg.Email.VerifyCode)
	if (purpose == constants.VerifyPurposeEmailAuth || purpose == constants.VerifyPurposeLoginEnv) && expireMinutes > constants.EmailAuthMaxExpireMinutes {
		expireMinutes = constants.EmailAuthMaxExpireMinutes
	}
	record := &emailverificationdomain.Code{
		Email:     email,
		Purpose:   strings.ToLower(purpose),
		Code:      code,
		ExpiresAt: now.Add(time.Duration(expireMinutes) * time.Minute),
		SentAt:    now,
		CreatedAt: now,
	}
	// 先落库再发信：配合上方发送互斥锁，杜绝并发请求同时发出多封邮件。
	if err := s.codeRepo.Create(record); err != nil {
		return err
	}
	if err := s.emailService.SendVerifyCode(email, code, purpose, locale, brand); err != nil {
		// 未送达的验证码不能占用发送间隔，也不能顶替用户手里仍有效的上一封；删除失败只退回旧行为，不掩盖发信错误
		_ = s.codeRepo.Delete(record.ID)
		return err
	}
	return nil
}

func normalizeEmail(email string) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(email))
	if normalized == "" {
		return "", ErrInvalidEmail
	}
	if _, err := mail.ParseAddress(normalized); err != nil {
		return "", ErrInvalidEmail
	}
	return normalized, nil
}

// normalizeUserSuppliedEmail 用于注册 / 换绑等用户自填邮箱的入口，
// Telegram 占位邮箱仅由系统生成，不接受用户填写。
func normalizeUserSuppliedEmail(email string) (string, error) {
	normalized, err := normalizeEmail(email)
	if err != nil {
		return "", err
	}
	if telegramidentity.IsPlaceholderEmail(normalized) {
		return "", ErrInvalidEmail
	}
	return normalized, nil
}

// NormalizeEmail 统一邮箱格式
func NormalizeEmail(email string) (string, error) {
	return normalizeEmail(email)
}

// isVerifyPurposeSupported 判断公开发信入口（SendVerifyCode）是否接受该用途。
// change_email_old / change_email_new 不在其中：它们只能由登录态的 SendChangeEmailCode
// 在校验用户身份、换绑模式和新邮箱占用后发送，公开入口接受会让未登录者向任意邮箱发信。
func isVerifyPurposeSupported(purpose string) bool {
	switch strings.ToLower(strings.TrimSpace(purpose)) {
	case constants.VerifyPurposeRegister, constants.VerifyPurposeEmailAuth, constants.VerifyPurposeReset, constants.VerifyPurposeTelegramBind:
		return true
	default:
		return false
	}
}

func resolveUserJWTExpireHours(cfg config.JWTConfig) int {
	if cfg.ExpireHours <= 0 {
		return 24
	}
	return cfg.ExpireHours
}

func resolveRememberMeExpireHours(cfg config.JWTConfig) int {
	if cfg.RememberMeExpireHours <= 0 {
		return resolveUserJWTExpireHours(cfg)
	}
	return cfg.RememberMeExpireHours
}

func resolveNicknameFromEmail(email string) string {
	parts := strings.SplitN(email, "@", 2)
	if len(parts) == 2 && strings.TrimSpace(parts[0]) != "" {
		return userdomain.SanitizeDisplayName(parts[0])
	}
	return userdomain.SanitizeDisplayName(email)
}

func resolveExpireMinutes(cfg config.VerifyCodeConfig) int {
	if cfg.ExpireMinutes <= 0 {
		return 10
	}
	return cfg.ExpireMinutes
}

func resolveSendIntervalSeconds(cfg config.VerifyCodeConfig) int {
	if cfg.SendIntervalSeconds <= 0 {
		return 60
	}
	return cfg.SendIntervalSeconds
}

func resolveMaxAttempts(cfg config.VerifyCodeConfig) int {
	if cfg.MaxAttempts <= 0 {
		return 5
	}
	return cfg.MaxAttempts
}

func resolveCodeLength(cfg config.VerifyCodeConfig) int {
	if cfg.Length < 4 || cfg.Length > 10 {
		return 6
	}
	return cfg.Length
}

func randomNumericCode(length int) (string, error) {
	var b strings.Builder
	for i := 0; i < length; i++ {
		n, err := rand.Int(rand.Reader, big.NewInt(10))
		if err != nil {
			return "", err
		}
		b.WriteString(fmt.Sprintf("%d", n.Int64()))
	}
	return b.String(), nil
}

const (
	// TrustedDeviceCookieName 标记当前浏览器已通过身份核验（30 天有效，改密/TokenVersion 变更即失效）。
	TrustedDeviceCookieName = "nt_trusted_device"
	TrustedDeviceTTL        = 30 * 24 * time.Hour

	accountLoginFailWindow      = 15 * time.Minute
	accountLoginLockDuration    = 15 * time.Minute
	accountLoginMaxFailures     = 10
	accountLoginMaxLocalEntries = 20000

	verifyCodeMaxPerHour = 8
	verifyCodeMaxPerDay  = 20
	verifyCodeLockTTL    = 5 * time.Second
)

// LoginContextInput 携带登录请求上下文，用于执行环境变更检测与受信任设备签发。
type LoginContextInput struct {
	Email              string
	Password           string
	RememberMe         bool
	ClientIP           string
	UserAgent          string
	Locale             string
	TrustedDeviceToken string
}

// LoginEmailChallengeClaims 新环境登录邮箱二次验证挑战 token claims。
type LoginEmailChallengeClaims struct {
	UserID      uint   `json:"user_id"`
	JTI         string `json:"jti"`
	Purpose     string `json:"purpose"`
	RememberMe  bool   `json:"remember_me"`
	LoginSource string `json:"login_source,omitempty"`
	Typ         string `json:"typ"`
	jwt.RegisteredClaims
}

type trustedDevicePayload struct {
	UID uint   `json:"uid"`
	TV  uint64 `json:"tv"`
	Exp int64  `json:"exp"`
	Dev string `json:"dev"`
}

// SetLoginHistoryReader 注入历史登录记录查询端口（用于存量老用户平滑过渡）。
func (s *Service) SetLoginHistoryReader(reader LoginHistoryReader) {
	s.loginHistory = reader
}

// BuildDeviceFamilyFingerprint 提取稳定的 OS + 浏览器家族标识，
// 避免浏览器自动小版本升级误判环境变化，同时在切换设备或浏览器时触发二次验证。
func BuildDeviceFamilyFingerprint(userAgent string) string {
	ua := strings.ToLower(strings.TrimSpace(userAgent))
	if ua == "" {
		return "unknown|unknown"
	}

	osFamily := "other"
	switch {
	case strings.Contains(ua, "iphone") || strings.Contains(ua, "ipad") || strings.Contains(ua, "ios"):
		osFamily = "ios"
	case strings.Contains(ua, "android"):
		osFamily = "android"
	case strings.Contains(ua, "mac os x") || strings.Contains(ua, "macintosh"):
		osFamily = "macos"
	case strings.Contains(ua, "windows"):
		osFamily = "windows"
	case strings.Contains(ua, "linux"):
		osFamily = "linux"
	}

	browserFamily := "other"
	switch {
	case strings.Contains(ua, "edg/") || strings.Contains(ua, "edgios/") || strings.Contains(ua, "edga/"):
		browserFamily = "edge"
	case strings.Contains(ua, "opr/") || strings.Contains(ua, "opera"):
		browserFamily = "opera"
	case strings.Contains(ua, "firefox/") || strings.Contains(ua, "fxios/"):
		browserFamily = "firefox"
	case strings.Contains(ua, "micromessenger"):
		browserFamily = "wechat"
	case strings.Contains(ua, "telegram"):
		browserFamily = "telegram"
	case strings.Contains(ua, "chrome/") || strings.Contains(ua, "crios/") || strings.Contains(ua, "chromium"):
		browserFamily = "chrome"
	case strings.Contains(ua, "safari/"):
		browserFamily = "safari"
	}

	return osFamily + "|" + browserFamily
}

func (s *Service) trustedDeviceSigningKey() []byte {
	secret := ""
	if s != nil && s.cfg != nil {
		secret = s.cfg.UserJWT.SecretKey
	}
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte("nt-trusted-device-v1"))
	return mac.Sum(nil)
}

// IssueTrustedDeviceToken 为指定用户在当前浏览器签发 30 天受信任设备凭证。
// 凭证绑定 TokenVersion：修改或重置密码后所有旧设备自动失效。
func (s *Service) IssueTrustedDeviceToken(user *userdomain.User, userAgent string) string {
	if s == nil || user == nil || user.ID == 0 {
		return ""
	}
	payload := trustedDevicePayload{
		UID: user.ID,
		TV:  user.TokenVersion,
		Exp: time.Now().Add(TrustedDeviceTTL).Unix(),
		Dev: BuildDeviceFamilyFingerprint(userAgent),
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return ""
	}
	encoded := "v1." + base64.RawURLEncoding.EncodeToString(raw)
	mac := hmac.New(sha256.New, s.trustedDeviceSigningKey())
	_, _ = mac.Write([]byte(encoded))
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return encoded + "." + sig
}

// VerifyTrustedDeviceToken 校验受信任设备 Cookie 是否有效且与当前用户、版本、设备家族匹配。
func (s *Service) VerifyTrustedDeviceToken(token string, user *userdomain.User, userAgent string) bool {
	if s == nil || user == nil || user.ID == 0 {
		return false
	}
	trimmed := strings.TrimSpace(token)
	parts := strings.Split(trimmed, ".")
	if len(parts) != 3 || parts[0] != "v1" {
		return false
	}
	encoded := parts[0] + "." + parts[1]
	sigBytes, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return false
	}
	mac := hmac.New(sha256.New, s.trustedDeviceSigningKey())
	_, _ = mac.Write([]byte(encoded))
	expected := mac.Sum(nil)
	if subtle.ConstantTimeCompare(sigBytes, expected) != 1 {
		return false
	}
	rawPayload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return false
	}
	var payload trustedDevicePayload
	if err := json.Unmarshal(rawPayload, &payload); err != nil {
		return false
	}
	if payload.UID != user.ID || payload.TV != user.TokenVersion {
		return false
	}
	if time.Now().Unix() >= payload.Exp {
		return false
	}
	return payload.Dev == BuildDeviceFamilyFingerprint(userAgent)
}

func sameClientNetwork(a, b string) bool {
	a = strings.TrimSpace(a)
	b = strings.TrimSpace(b)
	if a == "" || b == "" {
		return false
	}
	if a == b {
		return true
	}
	addrA, errA := netip.ParseAddr(a)
	addrB, errB := netip.ParseAddr(b)
	if errA != nil || errB != nil {
		return false
	}
	addrA = addrA.Unmap()
	addrB = addrB.Unmap()
	if addrA.Is4() && addrB.Is4() {
		pA, _ := addrA.Prefix(24)
		pB, _ := addrB.Prefix(24)
		return pA == pB
	}
	if addrA.Is6() && addrB.Is6() {
		pA, _ := addrA.Prefix(48)
		pB, _ := addrB.Prefix(48)
		return pA == pB
	}
	return false
}

// matchesRecentLoginHistory 检查存量老用户在最近 30 天内是否已有同网络 + 同设备家族的成功登录记录。
func (s *Service) matchesRecentLoginHistory(userID uint, clientIP, userAgent string) bool {
	if s == nil || s.loginHistory == nil || userID == 0 {
		return false
	}
	records, err := s.loginHistory.ListRecentUserLogins(userID, 20)
	if err != nil || len(records) == 0 {
		return false
	}
	cutoff := time.Now().Add(-TrustedDeviceTTL)
	currentDev := BuildDeviceFamilyFingerprint(userAgent)
	for _, rec := range records {
		if rec.CreatedAt.Before(cutoff) {
			continue
		}
		if strings.ToLower(strings.TrimSpace(rec.Status)) != constants.LoginLogStatusSuccess {
			continue
		}
		if sameClientNetwork(rec.ClientIP, clientIP) && BuildDeviceFamilyFingerprint(rec.UserAgent) == currentDev {
			return true
		}
	}
	return false
}

// MaskEmail 将邮箱脱敏用于前端安全验证提示（如 q***e@gmail.com）。
func MaskEmail(email string) string {
	normalized := strings.TrimSpace(email)
	at := strings.LastIndex(normalized, "@")
	if at <= 0 || at >= len(normalized)-1 {
		return normalized
	}
	local := []rune(normalized[:at])
	domain := normalized[at:]
	if len(local) <= 1 {
		return string(local) + "***" + domain
	}
	if len(local) == 2 {
		return string(local[0]) + "***" + domain
	}
	return string(local[0]) + "***" + string(local[len(local)-1]) + domain
}

// IssueLoginEmailChallengeToken 签发新环境登录邮箱验证挑战 token（独立 typ，不可冒充访问令牌）。
func (s *Service) IssueLoginEmailChallengeToken(userID uint, rememberMe bool, loginSource string) (token, jti string, expiresAt time.Time, err error) {
	jti = uuid.NewString()
	expiresAt = time.Now().Add(challenge.LoginEmailTTL)
	claims := LoginEmailChallengeClaims{
		UserID:      userID,
		JTI:         jti,
		Purpose:     challenge.PurposeLoginEmail,
		RememberMe:  rememberMe,
		LoginSource: normalizeLoginSource(loginSource),
		Typ:         jwttoken.TypeLoginEmailChallenge,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			ID:        jti,
		},
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := t.SignedString([]byte(s.cfg.UserJWT.SecretKey))
	if err != nil {
		return "", "", time.Time{}, err
	}
	return signed, jti, expiresAt, nil
}

// ParseLoginEmailChallengeToken 解析并校验新环境邮箱验证挑战 token。
func (s *Service) ParseLoginEmailChallengeToken(tokenString string) (*LoginEmailChallengeClaims, error) {
	parser := jwttoken.NewHS256Parser()
	tok, err := parser.ParseWithClaims(tokenString, &LoginEmailChallengeClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(s.cfg.UserJWT.SecretKey), nil
	})
	if err != nil {
		return nil, ErrLoginEmailChallengeInvalid
	}
	claims, ok := tok.Claims.(*LoginEmailChallengeClaims)
	if !ok || !tok.Valid {
		return nil, ErrLoginEmailChallengeInvalid
	}
	if claims.Purpose != challenge.PurposeLoginEmail || claims.Typ != jwttoken.TypeLoginEmailChallenge || strings.TrimSpace(claims.JTI) == "" {
		return nil, ErrLoginEmailChallengeInvalid
	}
	return claims, nil
}

// LoginWithContext 执行带登录环境感知与账号级防爆破的密码登录。
func (s *Service) LoginWithContext(ctx context.Context, input LoginContextInput) (*UserLoginResult, error) {
	normalized, err := normalizeEmail(input.Email)
	if err != nil {
		return nil, err
	}
	if locked, retryAfter := s.checkAccountLoginLocked(ctx, normalized); locked {
		return nil, AccountLockedError{RetryAfterSeconds: retryAfter}
	}

	user, err := s.userRepo.GetByEmail(normalized)
	if err != nil {
		return nil, err
	}
	if user == nil {
		_ = bcrypt.CompareHashAndPassword([]byte("$2a$10$dummyhashtopreventtimingattacksxxxxxxxxxxxxxxxxxx"), []byte(input.Password))
		s.recordAccountLoginFailure(ctx, normalized)
		return nil, ErrInvalidCredentials
	}
	if strings.ToLower(user.Status) != constants.UserStatusActive {
		return nil, ErrUserDisabled
	}
	emailVerificationEnabled := true
	if s.settingService != nil {
		enabled, settingErr := s.settingService.GetEmailVerificationEnabled(true)
		if settingErr != nil {
			return nil, settingErr
		}
		emailVerificationEnabled = enabled
	}
	if user.EmailVerifiedAt == nil && emailVerificationEnabled {
		return nil, ErrEmailNotVerified
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(input.Password)); err != nil {
		s.recordAccountLoginFailure(ctx, normalized)
		return nil, ErrInvalidCredentials
	}

	s.clearAccountLoginFailures(ctx, normalized)

	// 已开启 TOTP 时直接走 TOTP 验证，避免新环境叠加两次验证码
	if user.TOTPEnabledAt != nil {
		challengeToken, jti, expiresAt, err := s.IssueUserChallengeToken(user.ID, input.RememberMe)
		if err != nil {
			return nil, err
		}
		return &UserLoginResult{
			RequiresTOTP:       true,
			User:               user,
			ChallengeToken:     challengeToken,
			ChallengeJTI:       jti,
			ChallengeExpiresAt: expiresAt,
		}, nil
	}

	trusted := s.VerifyTrustedDeviceToken(input.TrustedDeviceToken, user, input.UserAgent)
	if !trusted {
		trusted = s.matchesRecentLoginHistory(user.ID, input.ClientIP, input.UserAgent)
	}

	if !trusted && emailVerificationEnabled && s.emailService != nil {
		locale := strings.TrimSpace(input.Locale)
		if strings.TrimSpace(user.Locale) != "" {
			locale = user.Locale
		}
		sendErr := s.sendVerifyCode(ctx, user.Email, constants.VerifyPurposeLoginEnv, locale)
		if sendErr != nil {
			switch {
			case errors.Is(sendErr, ErrEmailServiceNotConfigured),
				errors.Is(sendErr, notificationcontract.ErrEmailServiceDisabled),
				errors.Is(sendErr, notificationcontract.ErrEmailNotConfigured):
				logger.Warnw("login_env_email_skipped_unconfigured", "user_id", user.ID, "error", sendErr)
			case errors.Is(sendErr, ErrVerifyCodeTooFrequent) && s.hasActiveVerifyCode(user.Email, constants.VerifyPurposeLoginEnv):
				challengeToken, jti, expiresAt, err := s.IssueLoginEmailChallengeToken(user.ID, input.RememberMe, constants.LoginLogSourceWeb)
				if err != nil {
					return nil, err
				}
				return &UserLoginResult{
					RequiresEmailVerify: true,
					User:                user,
					ChallengeToken:      challengeToken,
					ChallengeJTI:        jti,
					ChallengeExpiresAt:  expiresAt,
					MaskedEmail:         MaskEmail(user.Email),
				}, nil
			default:
				return nil, sendErr
			}
		} else {
			challengeToken, jti, expiresAt, err := s.IssueLoginEmailChallengeToken(user.ID, input.RememberMe, constants.LoginLogSourceWeb)
			if err != nil {
				return nil, err
			}
			return &UserLoginResult{
				RequiresEmailVerify: true,
				User:                user,
				ChallengeToken:      challengeToken,
				ChallengeJTI:        jti,
				ChallengeExpiresAt:  expiresAt,
				MaskedEmail:         MaskEmail(user.Email),
			}, nil
		}
	}

	return s.completeTrustedPasswordLogin(user, input.RememberMe, input.UserAgent)
}

func (s *Service) hasActiveVerifyCode(email, purpose string) bool {
	record, err := s.codeRepo.GetLatest(email, purpose)
	if err != nil || record == nil || record.VerifiedAt != nil {
		return false
	}
	return record.ExpiresAt.After(time.Now())
}

func (s *Service) completeTrustedPasswordLogin(user *userdomain.User, rememberMe bool, userAgent string) (*UserLoginResult, error) {
	expireHours := resolveUserJWTExpireHours(s.cfg.UserJWT)
	if rememberMe {
		expireHours = resolveRememberMeExpireHours(s.cfg.UserJWT)
	}
	token, expiresAt, err := s.GenerateUserJWT(user, expireHours)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	user.LastLoginAt = &now
	if err := s.userRepo.Update(user); err != nil {
		return nil, err
	}
	_ = cache.SetUserAuthState(context.Background(), cache.BuildUserAuthState(user))

	return &UserLoginResult{
		RequiresTOTP:       false,
		User:               user,
		Token:              token,
		ExpiresAt:          expiresAt,
		TrustedDeviceToken: s.IssueTrustedDeviceToken(user, userAgent),
	}, nil
}

// VerifyLoginEmailChallenge 核验新环境登录邮箱验证码并完成登录，同时签发 30 天受信任设备凭证。
func (s *Service) VerifyLoginEmailChallenge(ctx context.Context, challengeToken, code, userAgent string) (*UserLoginResult, error) {
	claims, err := s.ParseLoginEmailChallengeToken(challengeToken)
	if err != nil {
		return nil, ErrLoginEmailChallengeInvalid
	}
	if s.isLoginEmailChallengeRevoked(ctx, claims.JTI) {
		return nil, ErrLoginEmailChallengeInvalid
	}
	user, err := s.userRepo.GetByID(claims.UserID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrLoginEmailChallengeInvalid
	}
	if strings.ToLower(user.Status) != constants.UserStatusActive {
		return nil, ErrUserDisabled
	}

	if _, err := s.verifyCode(user.Email, constants.VerifyPurposeLoginEnv, code); err != nil {
		fails := s.bumpLoginEmailChallengeFails(ctx, claims.JTI)
		if fails >= challenge.MaxFailures {
			s.revokeLoginEmailChallenge(ctx, claims.JTI)
			return nil, ErrVerifyCodeAttemptsExceeded
		}
		return nil, err
	}

	s.revokeLoginEmailChallenge(ctx, claims.JTI)
	return s.completeTrustedPasswordLogin(user, claims.RememberMe, userAgent)
}

// ResendLoginEmailChallenge 为有效的新环境登录挑战重新发送安全验证码。
func (s *Service) ResendLoginEmailChallenge(ctx context.Context, challengeToken, locale string) error {
	claims, err := s.ParseLoginEmailChallengeToken(challengeToken)
	if err != nil {
		return ErrLoginEmailChallengeInvalid
	}
	if s.isLoginEmailChallengeRevoked(ctx, claims.JTI) {
		return ErrLoginEmailChallengeInvalid
	}
	user, err := s.userRepo.GetByID(claims.UserID)
	if err != nil {
		return err
	}
	if user == nil {
		return ErrLoginEmailChallengeInvalid
	}
	if strings.ToLower(user.Status) != constants.UserStatusActive {
		return ErrUserDisabled
	}
	resolvedLocale := strings.TrimSpace(locale)
	if strings.TrimSpace(user.Locale) != "" {
		resolvedLocale = user.Locale
	}
	return s.sendVerifyCode(ctx, user.Email, constants.VerifyPurposeLoginEnv, resolvedLocale)
}

type localFailState struct {
	count     int
	expiresAt time.Time
	lockedTo  time.Time
}

type authRuntimeGuard struct {
	mu           sync.Mutex
	accountFails map[string]localFailState
	revokedJTIs  map[string]time.Time
	jtiFails     map[string]localFailState
	sendLocks    map[string]time.Time
}

func newAuthRuntimeGuard() *authRuntimeGuard {
	return &authRuntimeGuard{
		accountFails: make(map[string]localFailState),
		revokedJTIs:  make(map[string]time.Time),
		jtiFails:     make(map[string]localFailState),
		sendLocks:    make(map[string]time.Time),
	}
}

func (s *Service) guard() *authRuntimeGuard {
	if s.runtimeGuard == nil {
		s.runtimeGuard = newAuthRuntimeGuard()
	}
	return s.runtimeGuard
}

func accountFailRedisKey(email string) string {
	return "auth:user:account_fail:" + strings.ToLower(strings.TrimSpace(email))
}

func accountLockRedisKey(email string) string {
	return "auth:user:account_lock:" + strings.ToLower(strings.TrimSpace(email))
}

func (s *Service) checkAccountLoginLocked(ctx context.Context, email string) (bool, int) {
	if rdb := cache.Client(); rdb != nil {
		if ttl, err := rdb.TTL(ctx, accountLockRedisKey(email)).Result(); err == nil && ttl > 0 {
			sec := int(ttl.Seconds())
			if sec < 1 {
				sec = 1
			}
			return true, sec
		}
	}
	g := s.guard()
	g.mu.Lock()
	defer g.mu.Unlock()
	now := time.Now()
	st, ok := g.accountFails[email]
	if !ok {
		return false, 0
	}
	if !st.lockedTo.IsZero() && now.Before(st.lockedTo) {
		sec := int(st.lockedTo.Sub(now).Seconds())
		if sec < 1 {
			sec = 1
		}
		return true, sec
	}
	if !st.expiresAt.IsZero() && !now.Before(st.expiresAt) {
		delete(g.accountFails, email)
	}
	return false, 0
}

func (s *Service) recordAccountLoginFailure(ctx context.Context, email string) {
	if rdb := cache.Client(); rdb != nil {
		failKey := accountFailRedisKey(email)
		cnt, err := rdb.Incr(ctx, failKey).Result()
		if err == nil {
			if cnt == 1 {
				_ = rdb.Expire(ctx, failKey, accountLoginFailWindow).Err()
			}
			if cnt >= accountLoginMaxFailures {
				_ = rdb.Set(ctx, accountLockRedisKey(email), "1", accountLoginLockDuration).Err()
				_ = rdb.Del(ctx, failKey).Err()
			}
			return
		}
	}
	g := s.guard()
	g.mu.Lock()
	defer g.mu.Unlock()
	now := time.Now()
	if len(g.accountFails) >= accountLoginMaxLocalEntries {
		for k, v := range g.accountFails {
			if (v.lockedTo.IsZero() || !now.Before(v.lockedTo)) && (v.expiresAt.IsZero() || !now.Before(v.expiresAt)) {
				delete(g.accountFails, k)
			}
		}
	}
	st := g.accountFails[email]
	if st.expiresAt.IsZero() || !now.Before(st.expiresAt) {
		st = localFailState{expiresAt: now.Add(accountLoginFailWindow)}
	}
	st.count++
	if st.count >= accountLoginMaxFailures {
		st.lockedTo = now.Add(accountLoginLockDuration)
		st.count = 0
	}
	g.accountFails[email] = st
}

func (s *Service) clearAccountLoginFailures(ctx context.Context, email string) {
	if rdb := cache.Client(); rdb != nil {
		_ = rdb.Del(ctx, accountFailRedisKey(email), accountLockRedisKey(email)).Err()
	}
	g := s.guard()
	g.mu.Lock()
	delete(g.accountFails, email)
	g.mu.Unlock()
}

func loginEmailRevokeKey(jti string) string {
	return "auth:user:login_email_challenge:" + jti + ":revoked"
}

func loginEmailFailKey(jti string) string {
	return "auth:user:login_email_challenge:" + jti + ":fails"
}

func (s *Service) isLoginEmailChallengeRevoked(ctx context.Context, jti string) bool {
	if rdb := cache.Client(); rdb != nil {
		if v, err := rdb.Exists(ctx, loginEmailRevokeKey(jti)).Result(); err == nil && v == 1 {
			return true
		}
	}
	g := s.guard()
	g.mu.Lock()
	defer g.mu.Unlock()
	exp, ok := g.revokedJTIs[jti]
	if !ok {
		return false
	}
	if time.Now().After(exp) {
		delete(g.revokedJTIs, jti)
		return false
	}
	return true
}

func (s *Service) bumpLoginEmailChallengeFails(ctx context.Context, jti string) int64 {
	if rdb := cache.Client(); rdb != nil {
		key := loginEmailFailKey(jti)
		if cnt, err := rdb.Incr(ctx, key).Result(); err == nil {
			if cnt == 1 {
				_ = rdb.Expire(ctx, key, challenge.LoginEmailTTL).Err()
			}
			return cnt
		}
	}
	g := s.guard()
	g.mu.Lock()
	defer g.mu.Unlock()
	now := time.Now()
	st := g.jtiFails[jti]
	if st.expiresAt.IsZero() || now.After(st.expiresAt) {
		st = localFailState{expiresAt: now.Add(challenge.LoginEmailTTL)}
	}
	st.count++
	g.jtiFails[jti] = st
	return int64(st.count)
}

func (s *Service) revokeLoginEmailChallenge(ctx context.Context, jti string) {
	if rdb := cache.Client(); rdb != nil {
		_ = rdb.Set(ctx, loginEmailRevokeKey(jti), "1", challenge.LoginEmailTTL).Err()
	}
	g := s.guard()
	g.mu.Lock()
	g.revokedJTIs[jti] = time.Now().Add(challenge.LoginEmailTTL)
	delete(g.jtiFails, jti)
	g.mu.Unlock()
}

// acquireVerifyCodeSendLock 防止同一邮箱并发请求绕过发送冷却瞬间发出多封验证码邮件。
func (s *Service) acquireVerifyCodeSendLock(ctx context.Context, email, purpose string) (func(), bool) {
	lockKey := fmt.Sprintf("auth:verify_code:lock:%s:%s", strings.ToLower(strings.TrimSpace(email)), strings.ToLower(strings.TrimSpace(purpose)))
	if rdb := cache.Client(); rdb != nil {
		ok, err := rdb.SetNX(ctx, lockKey, "1", verifyCodeLockTTL).Result()
		if err == nil {
			if !ok {
				return nil, false
			}
			return func() {
				_ = rdb.Del(context.Background(), lockKey).Err()
			}, true
		}
	}
	g := s.guard()
	g.mu.Lock()
	now := time.Now()
	if exp, exists := g.sendLocks[lockKey]; exists && now.Before(exp) {
		g.mu.Unlock()
		return nil, false
	}
	g.sendLocks[lockKey] = now.Add(verifyCodeLockTTL)
	g.mu.Unlock()
	return func() {
		g.mu.Lock()
		delete(g.sendLocks, lockKey)
		g.mu.Unlock()
	}, true
}
