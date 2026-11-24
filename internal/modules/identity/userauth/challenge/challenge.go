package challenge

import "time"

const (
	PurposeTwoFactor  = "user_2fa_challenge"
	PurposeLoginEmail = "user_login_email_challenge"
	TTL               = 5 * time.Minute
	LoginEmailTTL     = 10 * time.Minute
	MaxFailures       = 5
)

func FailureKey(jti string) string {
	return "2fa:user:challenge:" + jti + ":fails"
}

func RevocationKey(jti string) string {
	return "2fa:user:challenge:" + jti + ":revoked"
}
