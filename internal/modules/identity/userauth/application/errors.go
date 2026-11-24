package application

import "errors"

var (
	ErrNotFound                     = errors.New("user not found")
	ErrInvalidCredentials           = errors.New("invalid credentials")
	ErrInvalidPassword              = errors.New("invalid password")
	ErrEmailExists                  = errors.New("email exists")
	ErrEmailNotVerified             = errors.New("email not verified")
	ErrUserDisabled                 = errors.New("user disabled")
	ErrInvalidEmail                 = errors.New("invalid email")
	ErrInvalidVerifyPurpose         = errors.New("invalid verify purpose")
	ErrAgreementRequired            = errors.New("agreement required")
	ErrVerifyCodeInvalid            = errors.New("verify code invalid")
	ErrVerifyCodeExpired            = errors.New("verify code expired")
	ErrVerifyCodeTooFrequent        = errors.New("verify code too frequent")
	ErrVerifyCodeAttemptsExceeded   = errors.New("verify code attempts exceeded")
	ErrEmailServiceNotConfigured    = errors.New("email service not configured")
	ErrUserOAuthIdentityExists      = errors.New("user oauth identity exists")
	ErrUserOAuthAlreadyBound        = errors.New("user oauth already bound")
	ErrUserOAuthNotBound            = errors.New("user oauth not bound")
	ErrTelegramUnbindRequiresEmail  = errors.New("telegram unbind requires real email")
	ErrGoogleAutoLinkForbidden      = errors.New("google email auto link forbidden")
	ErrGoogleUnbindLocked           = errors.New("google unbind would lock account")
	ErrGoogleRedirectUnavailable    = errors.New("google redirect state store unavailable")
	ErrGoogleRedirectSessionExpired = errors.New("google redirect session expired")
	ErrGoogleRedirectTenantMismatch = errors.New("google redirect tenant mismatch")
	ErrGoogleRedirectUserMismatch   = errors.New("google redirect user mismatch")
	ErrGoogleRedirectFlowInvalid    = errors.New("google redirect flow invalid")
	ErrProfileEmpty                 = errors.New("profile empty")
	ErrNicknameInvalid              = errors.New("nickname invalid")
	ErrEmailChangeInvalid           = errors.New("email change invalid")
	ErrEmailChangeExists            = errors.New("email change exists")
	ErrRegistrationDisabled         = errors.New("registration disabled")
	ErrInviteCodeRequired           = errors.New("invite code required")
	ErrInviteCodeInvalid            = errors.New("invite code invalid")
	ErrInviteTicketInvalid          = errors.New("invite ticket invalid")
	ErrLoginEmailChallengeInvalid   = errors.New("login email challenge invalid")
	ErrAccountLocked                = errors.New("account login temporarily locked")
)

// AccountLockedError 携带账号临时锁定剩余秒数。
type AccountLockedError struct {
	RetryAfterSeconds int
}

func (e AccountLockedError) Error() string { return ErrAccountLocked.Error() }
func (e AccountLockedError) Is(target error) bool {
	return target == ErrAccountLocked
}

var errExternalIdentityUnbindLocked = errors.New("external identity unbind would lock account")
var errGoogleLoginMappingChanged = errors.New("google identity mapping changed during login")
