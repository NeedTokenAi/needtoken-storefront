package integrationtest

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/dujiao-next/internal/config"
	"github.com/dujiao-next/internal/constants"
	emailverificationdomain "github.com/dujiao-next/internal/modules/identity/emailverification/domain"
	emailverificationstore "github.com/dujiao-next/internal/modules/identity/emailverification/infrastructure/gormstore"
	usercontract "github.com/dujiao-next/internal/modules/identity/user/contract"
	userdomain "github.com/dujiao-next/internal/modules/identity/user/domain"
	userstore "github.com/dujiao-next/internal/modules/identity/user/infrastructure/gormstore"
	userauthapp "github.com/dujiao-next/internal/modules/identity/userauth/application"
	settingsapp "github.com/dujiao-next/internal/modules/settings/application"
	"github.com/dujiao-next/internal/shared/mailbrand"
	"github.com/pquerna/otp/totp"
)

type emailAuthSender struct {
	codes    map[string]string
	purposes []string
}

type emailAuthLookupStore struct {
	usercontract.Store
	lookups int
}

func (s *emailAuthLookupStore) GetByEmail(email string) (*userdomain.User, error) {
	s.lookups++
	return s.Store.GetByEmail(email)
}

func TestEmailAuthDoesNotQueryAccountBeforeInboxProof(t *testing.T) {
	sender := &emailAuthSender{}
	_, settings, db := newRegistrationDomainPolicyAuthServiceWithSender(t, sender)
	users := &emailAuthLookupStore{Store: userstore.New(db)}
	service := userauthapp.NewService(&config.Config{}, users, nil, emailverificationstore.New(db), settings, sender, nil)
	email := "proof-first@example.com"
	code := emailAuthProof(t, service, sender, email)
	if users.lookups != 0 {
		t.Fatal("public mail request queried account existence")
	}
	if _, err := service.VerifyEmailAuth(context.Background(), email, "invalid-code"); !errors.Is(err, userauthapp.ErrVerifyCodeInvalid) || users.lookups != 0 {
		t.Fatalf("invalid proof queried account: calls=%d, %v", users.lookups, err)
	}
	if _, err := service.VerifyEmailAuth(context.Background(), email, code); err != nil || users.lookups != 1 {
		t.Fatalf("valid proof must precede one lookup: calls=%d, %v", users.lookups, err)
	}
}

func (s *emailAuthSender) SendVerifyCode(email, code, purpose, _ string, _ mailbrand.Brand) error {
	if s.codes == nil {
		s.codes = map[string]string{}
	}
	s.codes[email], s.purposes = code, append(s.purposes, purpose)
	return nil
}

func emailAuthProof(t *testing.T, service *userauthapp.Service, sender *emailAuthSender, email string) string {
	t.Helper()
	if err := service.SendVerifyCode(context.Background(), email, constants.VerifyPurposeEmailAuth, constants.LocaleZhCN, ""); err != nil {
		t.Fatal(err)
	}
	code := sender.codes[email]
	if len(code) < 6 {
		t.Fatal("email ownership code must contain at least six digits")
	}
	return code
}

func TestEmailAuthRequiresInboxProofAndTicketRemainsEmailBound(t *testing.T) {
	sender := &emailAuthSender{}
	service, _, db := newRegistrationDomainPolicyAuthServiceWithSender(t, sender)
	email := "new-owner@example.com"
	code := emailAuthProof(t, service, sender, email)
	if _, err := service.VerifyEmailAuth(context.Background(), "other@example.com", code); !errors.Is(err, userauthapp.ErrVerifyCodeInvalid) {
		t.Fatalf("cross-email proof accepted: %v", err)
	}
	result, err := service.VerifyEmailAuth(context.Background(), email, code)
	if err != nil || result.NextStep != "register" || !strings.HasPrefix(result.RegistrationCode, "eauth_") || len(result.RegistrationCode) < 48 || time.Until(result.ExpiresAt) > 10*time.Minute {
		t.Fatalf("registration continuation = %+v, %v", result, err)
	}
	if _, err := service.VerifyEmailAuth(context.Background(), email, code); !errors.Is(err, userauthapp.ErrVerifyCodeInvalid) {
		t.Fatalf("inbox code replay accepted: %v", err)
	}
	if _, err := service.ParseUserJWT(result.RegistrationCode); err == nil {
		t.Fatal("registration ticket must never authenticate an access session")
	}
	if _, _, _, err := service.Register("other@example.com", "Secret123", result.RegistrationCode, "", true, false); !errors.Is(err, userauthapp.ErrVerifyCodeInvalid) {
		t.Fatalf("ticket accepted for another email with global verification disabled: %v", err)
	}
	if _, _, _, err := service.Register(email, "Secret123", result.RegistrationCode, "", false, false); !errors.Is(err, userauthapp.ErrAgreementRequired) {
		t.Fatalf("ticket bypassed agreement: %v", err)
	}
	user, token, _, err := service.Register(email, "Secret123", result.RegistrationCode, "", true, false)
	if err != nil || user == nil || token == "" || user.EmailVerifiedAt == nil || user.TOTPEnabledAt != nil {
		t.Fatalf("verified registration = %+v, %v", user, err)
	}
	var ticket emailverificationdomain.Code
	if err := db.Where("email = ? AND purpose = ?", email, constants.VerifyPurposeRegister).First(&ticket).Error; err != nil || ticket.VerifiedAt == nil {
		t.Fatalf("register did not consume ticket: %+v, %v", ticket, err)
	}
}

func TestEmailAuthExistingAccountDoesNotBypassPasswordOrMFA(t *testing.T) {
	service, totpService, users, db := newUser2FATestServices(t)
	user := createActiveUser(t, users, "existing-owner@example.com", "Secret123")
	setup, err := totpService.Setup(user.ID)
	if err != nil {
		t.Fatal(err)
	}
	code, _ := totp.GenerateCode(setup.Secret, time.Now())
	if _, err := totpService.Enable(user.ID, code); err != nil {
		t.Fatal(err)
	}
	before, _ := users.GetByID(user.ID)
	now := time.Now()
	proof := emailverificationdomain.Code{Email: user.Email, Purpose: constants.VerifyPurposeEmailAuth, Code: "123456", SentAt: now, CreatedAt: now, ExpiresAt: now.Add(time.Minute)}
	if err := db.Create(&proof).Error; err != nil {
		t.Fatal(err)
	}
	result, err := service.VerifyEmailAuth(context.Background(), user.Email, "123456")
	if err != nil || result.NextStep != "password" || result.RegistrationCode != "" || !result.ExpiresAt.IsZero() {
		t.Fatalf("existing-account continuation = %+v, %v", result, err)
	}
	after, _ := users.GetByID(user.ID)
	if after.PasswordHash != before.PasswordHash || after.TokenVersion != before.TokenVersion || after.TOTPEnabledAt == nil {
		t.Fatal("inbox proof altered existing credentials or sessions")
	}
	login, err := service.LoginStep1(user.Email, "Secret123", false)
	if err != nil || !login.RequiresTOTP || login.Token != "" {
		t.Fatalf("existing MFA bypassed: %+v, %v", login, err)
	}
}

func TestEmailAuthSendDoesNotRevealRegistrationPolicyOrAccountState(t *testing.T) {
	sender := &emailAuthSender{}
	service, settings, db := newRegistrationDomainPolicyAuthServiceWithSender(t, sender)
	if _, err := settings.Update(constants.SettingKeyRegistrationConfig, map[string]interface{}{constants.SettingFieldRegistrationEnabled: false, constants.SettingFieldEmailDomainAllowlistEnabled: true, constants.SettingFieldAllowedEmailDomains: []interface{}{"qq.com"}}); err != nil {
		t.Fatal(err)
	}
	existing := userdomain.User{Email: "disabled@example.com", PasswordHash: "unchanged", Status: constants.UserStatusDisabled, TokenVersion: 7}
	if err := db.Create(&existing).Error; err != nil {
		t.Fatal(err)
	}
	for _, email := range []string{existing.Email, "absent@example.com"} {
		code := emailAuthProof(t, service, sender, email)
		result, err := service.VerifyEmailAuth(context.Background(), email, code)
		if email == existing.Email {
			if err != nil || result.NextStep != "password" {
				t.Fatalf("disabled account leaked status: %+v, %v", result, err)
			}
		} else if !errors.Is(err, userauthapp.ErrRegistrationDisabled) {
			t.Fatalf("new registration ignored disabled policy: %v", err)
		}
	}
	if len(sender.purposes) != 2 || sender.purposes[0] != constants.VerifyPurposeEmailAuth || sender.purposes[1] != constants.VerifyPurposeEmailAuth {
		t.Fatal("mail purpose differed by account existence")
	}
	if _, err := settings.Update(constants.SettingKeyRegistrationConfig, map[string]interface{}{constants.SettingFieldRegistrationEnabled: true, constants.SettingFieldEmailDomainAllowlistEnabled: true, constants.SettingFieldAllowedEmailDomains: []interface{}{"qq.com"}}); err != nil {
		t.Fatal(err)
	}
	code := emailAuthProof(t, service, sender, "blocked-domain@example.com")
	if _, err := service.VerifyEmailAuth(context.Background(), "blocked-domain@example.com", code); !errors.Is(err, settingsapp.ErrEmailDomainNotAllowed) {
		t.Fatalf("domain policy bypassed after proof: %v", err)
	}
}

func TestEmailAuthCodeExpiryPurposeRetryAndCancellationBoundaries(t *testing.T) {
	for _, scenario := range []string{"expired", "wrong purpose", "attempt limit", "cancelled", "expired ticket", "superseded"} {
		t.Run(scenario, func(t *testing.T) {
			sender := &emailAuthSender{}
			service, _, db := newRegistrationDomainPolicyAuthServiceWithSender(t, sender)
			email := "boundary@example.com"
			code := emailAuthProof(t, service, sender, email)
			switch scenario {
			case "superseded":
				newCode := "111111"
				if code == newCode {
					newCode = "222222"
				}
				now := time.Now()
				if err := db.Create(&emailverificationdomain.Code{Email: email, Purpose: constants.VerifyPurposeEmailAuth, Code: newCode, SentAt: now.Add(time.Millisecond), CreatedAt: now, ExpiresAt: now.Add(time.Minute)}).Error; err != nil {
					t.Fatal(err)
				}
				if _, err := service.VerifyEmailAuth(context.Background(), email, code); !errors.Is(err, userauthapp.ErrVerifyCodeInvalid) {
					t.Fatalf("superseded code accepted: %v", err)
				}
				if _, err := service.VerifyEmailAuth(context.Background(), email, newCode); err != nil {
					t.Fatalf("latest code rejected: %v", err)
				}
			case "expired":
				_ = db.Model(&emailverificationdomain.Code{}).Where("email = ?", email).Update("expires_at", time.Now().Add(-time.Minute)).Error
				if _, err := service.VerifyEmailAuth(context.Background(), email, code); !errors.Is(err, userauthapp.ErrVerifyCodeExpired) {
					t.Fatalf("expired code = %v", err)
				}
			case "wrong purpose":
				_ = db.Model(&emailverificationdomain.Code{}).Where("email = ?", email).Update("purpose", constants.VerifyPurposeReset).Error
				if _, err := service.VerifyEmailAuth(context.Background(), email, code); !errors.Is(err, userauthapp.ErrVerifyCodeInvalid) {
					t.Fatalf("wrong-purpose code = %v", err)
				}
			case "attempt limit":
				for i := 0; i < 5; i++ {
					_, _ = service.VerifyEmailAuth(context.Background(), email, "not-the-code")
				}
				if _, err := service.VerifyEmailAuth(context.Background(), email, code); !errors.Is(err, userauthapp.ErrVerifyCodeAttemptsExceeded) {
					t.Fatalf("exhausted code = %v", err)
				}
			case "cancelled":
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				if _, err := service.VerifyEmailAuth(ctx, email, code); !errors.Is(err, context.Canceled) {
					t.Fatalf("cancelled verification = %v", err)
				}
				if _, err := service.VerifyEmailAuth(context.Background(), email, code); err != nil {
					t.Fatalf("cancel consumed code: %v", err)
				}
			case "expired ticket":
				result, err := service.VerifyEmailAuth(context.Background(), email, code)
				if err != nil {
					t.Fatal(err)
				}
				_ = db.Model(&emailverificationdomain.Code{}).Where("email = ? AND purpose = ?", email, constants.VerifyPurposeRegister).Update("expires_at", time.Now().Add(-time.Minute)).Error
				if _, _, _, err := service.Register(email, "Secret123", result.RegistrationCode, "", true, false); !errors.Is(err, userauthapp.ErrVerifyCodeExpired) {
					t.Fatalf("expired ticket = %v", err)
				}
			}
		})
	}
}

func TestEmailAuthCodeHasOneConcurrentWinner(t *testing.T) {
	sender := &emailAuthSender{}
	service, _, db := newRegistrationDomainPolicyAuthServiceWithSender(t, sender)
	sqlDB, _ := db.DB()
	sqlDB.SetMaxOpenConns(1)
	email := "concurrent-owner@example.com"
	code := emailAuthProof(t, service, sender, email)
	var wait sync.WaitGroup
	results := make(chan error, 2)
	for i := 0; i < 2; i++ {
		wait.Add(1)
		go func() {
			defer wait.Done()
			_, err := service.VerifyEmailAuth(context.Background(), email, code)
			results <- err
		}()
	}
	wait.Wait()
	close(results)
	winners := 0
	for err := range results {
		if err == nil {
			winners++
		}
	}
	var tickets int64
	if err := db.Model(&emailverificationdomain.Code{}).Where("email = ? AND purpose = ?", email, constants.VerifyPurposeRegister).Count(&tickets).Error; err != nil || winners != 1 || tickets != 1 {
		t.Fatalf("replayed proof minted tickets: winners=%d tickets=%d err=%v", winners, tickets, err)
	}
}
