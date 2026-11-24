package userauthhttp

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	userdomain "github.com/dujiao-next/internal/modules/identity/user/domain"

	"github.com/dujiao-next/internal/constants"
	"github.com/dujiao-next/internal/i18n"
	captcha "github.com/dujiao-next/internal/modules/captcha/contract"
	captchahttp "github.com/dujiao-next/internal/modules/captcha/transport/http"
	userpresenter "github.com/dujiao-next/internal/modules/identity/userauth/transport/presenter"
	"github.com/dujiao-next/internal/platform/http/ginutil"
	"github.com/dujiao-next/internal/platform/http/response"

	"github.com/gin-gonic/gin"
)

const (
	trustedDeviceCookieName   = "nt_trusted_device"
	trustedDeviceCookieMaxAge = 30 * 24 * 3600
)

var (
	ErrAgreementRequired          = errors.New("agreement required")
	ErrInvalidCredentials         = errors.New("invalid credentials")
	ErrEmailNotVerified           = errors.New("email not verified")
	ErrLoginEmailChallengeInvalid = errors.New("login email challenge invalid")
	ErrAccountLocked              = errors.New("account login temporarily locked")
)

// AccountLockedError 携带账号临时锁定的剩余秒数。
type AccountLockedError struct {
	RetryAfterSeconds int
}

func (e AccountLockedError) Error() string { return ErrAccountLocked.Error() }
func (e AccountLockedError) Is(target error) bool {
	return target == ErrAccountLocked
}

// LoginContextParams 登录环境校验输入。
type LoginContextParams struct {
	Email              string
	Password           string
	RememberMe         bool
	ClientIP           string
	UserAgent          string
	Locale             string
	TrustedDeviceToken string
}

// UserLoginEnvAuth 是可选的环境感知登录、受信任设备签发与邮箱二次验证端口。
type UserLoginEnvAuth interface {
	LoginStep1WithEnv(ctx context.Context, params LoginContextParams) (*AuthLoginResult, error)
	VerifyLoginEmail(ctx context.Context, challengeToken, code, userAgent string) (*AuthLoginResult, error)
	ResendLoginEmail(ctx context.Context, challengeToken, locale string) error
	IssueTrustedDeviceToken(user *userdomain.User, userAgent string) string
}

// TrustedDeviceIssuer 供注册/2FA 完成后签发受信任设备 Cookie。
type TrustedDeviceIssuer interface {
	IssueTrustedDeviceToken(user *userdomain.User, userAgent string) string
}

func isSecureCookieRequest(c *gin.Context) bool {
	if c == nil || c.Request == nil {
		return false
	}
	if c.Request.TLS != nil {
		return true
	}
	if strings.EqualFold(strings.TrimSpace(c.GetHeader("X-Forwarded-Proto")), "https") {
		return true
	}
	if strings.Contains(strings.ToLower(c.GetHeader("CF-Visitor")), `"https"`) {
		return true
	}
	return false
}

func setTrustedDeviceCookie(c *gin.Context, token string) {
	if c == nil || strings.TrimSpace(token) == "" {
		return
	}
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     trustedDeviceCookieName,
		Value:    token,
		Path:     "/",
		MaxAge:   trustedDeviceCookieMaxAge,
		HttpOnly: true,
		Secure:   isSecureCookieRequest(c),
		SameSite: http.SameSiteLaxMode,
	})
}

// UserLoginSettings 是注册端点所需的设置端口。
type UserLoginSettings interface {
	GetRegistrationEnabled(defaultValue bool) (bool, error)
	GetEmailVerificationEnabled(defaultValue bool) (bool, error)
}

// UserLoginAuth 是注册/登录端点所需的认证端口。
type UserLoginAuth interface {
	Register(ctx context.Context, email, password, code, inviteCode string, agreementAccepted, emailVerificationEnabled bool) (*userdomain.User, string, time.Time, error)
	LoginStep1(ctx context.Context, email, password string, rememberMe bool) (*AuthLoginResult, error)
}

// UserLoginHandler 处理公开的注册与登录 HTTP 请求。
type UserLoginHandler struct {
	settings UserLoginSettings
	auth     UserLoginAuth
	captcha  CaptchaVerifier
	recorder LoginRecorder
}

func NewUserLoginHandler(settings UserLoginSettings, auth UserLoginAuth, captcha CaptchaVerifier, recorder LoginRecorder) *UserLoginHandler {
	if settings == nil {
		panic("user login handler: settings is nil")
	}
	if auth == nil {
		panic("user login handler: auth is nil")
	}
	return &UserLoginHandler{settings: settings, auth: auth, captcha: captcha, recorder: recorder}
}

// UserRegisterRequest 注册请求。
type UserRegisterRequest struct {
	Email             string                            `json:"email" binding:"required"`
	Password          string                            `json:"password" binding:"required"`
	Code              string                            `json:"code"`
	InviteCode        string                            `json:"invite_code"`
	AgreementAccepted bool                              `json:"agreement_accepted"`
	CaptchaPayload    captchahttp.CaptchaPayloadRequest `json:"captcha_payload"`
}

// UserLoginRequest 登录请求。
type UserLoginRequest struct {
	Email          string                            `json:"email" binding:"required"`
	Password       string                            `json:"password" binding:"required"`
	RememberMe     bool                              `json:"remember_me"`
	CaptchaPayload captchahttp.CaptchaPayloadRequest `json:"captcha_payload"`
}

func (h *UserLoginHandler) recordLogin(c *gin.Context, email string, userID uint, status, failReason, source string) {
	if h != nil {
		recordLoginEvent(h.recorder, c, email, userID, status, failReason, source)
	}
}

// UserRegister 用户注册。
func (h *UserLoginHandler) UserRegister(c *gin.Context) {
	var req UserRegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ginutil.RespondBindError(c, err)
		return
	}

	registrationEnabled, err := h.settings.GetRegistrationEnabled(true)
	if err != nil {
		ginutil.RespondError(c, response.CodeInternal, "error.register_failed", err)
		return
	}
	if !registrationEnabled {
		ginutil.RespondError(c, response.CodeForbidden, "error.registration_disabled", nil)
		return
	}

	emailVerificationEnabled, err := h.settings.GetEmailVerificationEnabled(true)
	if err != nil {
		ginutil.RespondError(c, response.CodeInternal, "error.register_failed", err)
		return
	}

	// 若站点未开启邮箱验证，注册提交必须直接过人机核验，防止脚本无限批量注册。
	if !emailVerificationEnabled && h.captcha != nil {
		if captchaErr := h.captcha.Verify(constants.CaptchaSceneRegisterSendCode, req.CaptchaPayload, c.ClientIP()); captchaErr != nil {
			respondCaptchaError(c, captchaErr)
			return
		}
	}

	user, token, expiresAt, err := h.auth.Register(c.Request.Context(), req.Email, req.Password, req.Code, req.InviteCode, req.AgreementAccepted, emailVerificationEnabled)
	if err != nil {
		if respondInviteCodeError(c, err) {
			return
		}
		switch {
		case errors.Is(err, ErrInvalidEmail):
			ginutil.RespondError(c, response.CodeBadRequest, "error.email_invalid", nil)
		case errors.Is(err, ErrEmailExists):
			ginutil.RespondError(c, response.CodeBadRequest, "error.email_exists", nil)
		case errors.Is(err, ErrEmailDomainNotAllowed):
			ginutil.RespondError(c, response.CodeBadRequest, "error.email_domain_not_allowed", nil)
		case errors.Is(err, ErrVerifyCodeInvalid):
			ginutil.RespondError(c, response.CodeBadRequest, "error.verify_code_invalid", nil)
		case errors.Is(err, ErrVerifyCodeExpired):
			ginutil.RespondError(c, response.CodeBadRequest, "error.verify_code_expired", nil)
		case errors.Is(err, ErrVerifyCodeAttemptsExceeded):
			ginutil.RespondError(c, response.CodeBadRequest, "error.verify_code_attempts_exceeded", nil)
		case errors.Is(err, ErrAgreementRequired):
			ginutil.RespondError(c, response.CodeBadRequest, "error.agreement_required", nil)
		case errors.Is(err, ErrWeakPassword):
			respondWeakPassword(c, err)
		default:
			ginutil.RespondError(c, response.CodeInternal, "error.register_failed", err)
		}
		return
	}

	if issuer, ok := h.auth.(TrustedDeviceIssuer); ok && user != nil {
		setTrustedDeviceCookie(c, issuer.IssueTrustedDeviceToken(user, c.GetHeader("User-Agent")))
	}
	if user != nil {
		h.recordLogin(c, user.Email, user.ID, constants.LoginLogStatusSuccess, "", constants.LoginLogSourceWeb)
	}

	response.Success(c, gin.H{
		"user":       userpresenter.NewUserAuthBriefResp(user),
		"token":      token,
		"expires_at": expiresAt.Format("2006-01-02T15:04:05Z07:00"),
	})
}

// UserLogin 用户登录。
func (h *UserLoginHandler) UserLogin(c *gin.Context) {
	var req UserLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.recordLogin(c, req.Email, 0, constants.LoginLogStatusFailed, constants.LoginLogFailReasonBadRequest, constants.LoginLogSourceWeb)
		ginutil.RespondError(c, response.CodeBadRequest, "error.bad_request", err)
		return
	}

	if h.captcha != nil {
		if captchaErr := h.captcha.Verify(constants.CaptchaSceneLogin, req.CaptchaPayload, c.ClientIP()); captchaErr != nil {
			switch {
			case errors.Is(captchaErr, captcha.ErrRequired):
				h.recordLogin(c, req.Email, 0, constants.LoginLogStatusFailed, constants.LoginLogFailReasonCaptchaRequired, constants.LoginLogSourceWeb)
				ginutil.RespondError(c, response.CodeBadRequest, "error.captcha_required", nil)
				return
			case errors.Is(captchaErr, captcha.ErrInvalid):
				h.recordLogin(c, req.Email, 0, constants.LoginLogStatusFailed, constants.LoginLogFailReasonCaptchaInvalid, constants.LoginLogSourceWeb)
				ginutil.RespondError(c, response.CodeBadRequest, "error.captcha_invalid", nil)
				return
			case errors.Is(captchaErr, captcha.ErrConfigInvalid):
				h.recordLogin(c, req.Email, 0, constants.LoginLogStatusFailed, constants.LoginLogFailReasonCaptchaConfigInvalid, constants.LoginLogSourceWeb)
				ginutil.RespondError(c, response.CodeInternal, "error.captcha_config_invalid", captchaErr)
				return
			default:
				h.recordLogin(c, req.Email, 0, constants.LoginLogStatusFailed, constants.LoginLogFailReasonCaptchaVerifyFailed, constants.LoginLogSourceWeb)
				ginutil.RespondError(c, response.CodeInternal, "error.captcha_verify_failed", captchaErr)
				return
			}
		}
	}

	var (
		res *AuthLoginResult
		err error
	)
	if envAuth, ok := h.auth.(UserLoginEnvAuth); ok {
		trustedCookie, _ := c.Cookie(trustedDeviceCookieName)
		res, err = envAuth.LoginStep1WithEnv(c.Request.Context(), LoginContextParams{
			Email:              req.Email,
			Password:           req.Password,
			RememberMe:         req.RememberMe,
			ClientIP:           c.ClientIP(),
			UserAgent:          c.GetHeader("User-Agent"),
			Locale:             i18n.ResolveLocale(c),
			TrustedDeviceToken: trustedCookie,
		})
	} else {
		res, err = h.auth.LoginStep1(c.Request.Context(), req.Email, req.Password, req.RememberMe)
	}
	if err != nil {
		var lockedErr AccountLockedError
		switch {
		case errors.As(err, &lockedErr):
			h.recordLogin(c, req.Email, 0, constants.LoginLogStatusFailed, constants.LoginLogFailReasonAccountLocked, constants.LoginLogSourceWeb)
			retryAfter := lockedErr.RetryAfterSeconds
			if retryAfter < 1 {
				retryAfter = 60
			}
			msg := i18n.Sprintf(i18n.ResolveLocale(c), "error.login_account_locked", retryAfter)
			response.ErrorWithHTTPStatus(c, http.StatusTooManyRequests, response.CodeTooManyRequests, msg)
		case errors.Is(err, ErrInvalidEmail):
			h.recordLogin(c, req.Email, 0, constants.LoginLogStatusFailed, constants.LoginLogFailReasonInvalidEmail, constants.LoginLogSourceWeb)
			ginutil.RespondError(c, response.CodeBadRequest, "error.email_invalid", nil)
		case errors.Is(err, ErrInvalidCredentials):
			h.recordLogin(c, req.Email, 0, constants.LoginLogStatusFailed, constants.LoginLogFailReasonInvalidCredentials, constants.LoginLogSourceWeb)
			ginutil.RespondError(c, response.CodeUnauthorized, "error.login_invalid", nil)
		case errors.Is(err, ErrEmailNotVerified):
			h.recordLogin(c, req.Email, 0, constants.LoginLogStatusFailed, constants.LoginLogFailReasonEmailNotVerified, constants.LoginLogSourceWeb)
			ginutil.RespondError(c, response.CodeUnauthorized, "error.email_not_verified", nil)
		case errors.Is(err, ErrUserDisabled):
			h.recordLogin(c, req.Email, 0, constants.LoginLogStatusFailed, constants.LoginLogFailReasonUserDisabled, constants.LoginLogSourceWeb)
			ginutil.RespondError(c, response.CodeUnauthorized, "error.user_disabled", nil)
		case errors.Is(err, ErrVerifyCodeTooFrequent):
			h.recordLogin(c, req.Email, 0, constants.LoginLogStatusFailed, constants.LoginLogFailReasonTooManyAttempts, constants.LoginLogSourceWeb)
			ginutil.RespondError(c, response.CodeTooManyRequests, "error.verify_code_too_frequent", nil)
		default:
			h.recordLogin(c, req.Email, 0, constants.LoginLogStatusFailed, constants.LoginLogFailReasonInternalError, constants.LoginLogSourceWeb)
			ginutil.RespondError(c, response.CodeInternal, "error.login_failed", err)
		}
		return
	}

	if res.RequiresTOTP {
		h.recordLogin(c, res.User.Email, res.User.ID, constants.LoginLogStatusSuccess, constants.LoginLogPasswordOK2FAPending, constants.LoginLogSourceWeb)
		response.Success(c, gin.H{
			"requires_totp":         true,
			"requires_email_verify": false,
			"challenge_token":       res.ChallengeToken,
			"challenge_expires_at":  res.ChallengeExpiresAt.Format("2006-01-02T15:04:05Z07:00"),
		})
		return
	}

	if res.RequiresEmailVerify {
		c.Header("Cache-Control", "no-store")
		h.recordLogin(c, res.User.Email, res.User.ID, constants.LoginLogStatusSuccess, constants.LoginLogPasswordOKEmailPending, constants.LoginLogSourceWeb)
		response.Success(c, gin.H{
			"requires_totp":         false,
			"requires_email_verify": true,
			"challenge_token":       res.ChallengeToken,
			"challenge_expires_at":  res.ChallengeExpiresAt.Format("2006-01-02T15:04:05Z07:00"),
			"masked_email":          res.MaskedEmail,
		})
		return
	}

	setTrustedDeviceCookie(c, res.TrustedDeviceToken)
	h.recordLogin(c, res.User.Email, res.User.ID, constants.LoginLogStatusSuccess, "", constants.LoginLogSourceWeb)
	response.Success(c, gin.H{
		"requires_totp":         false,
		"requires_email_verify": false,
		"user":                  userpresenter.NewUserAuthBriefResp(res.User),
		"token":                 res.Token,
		"expires_at":            res.ExpiresAt.Format("2006-01-02T15:04:05Z07:00"),
	})
}

// VerifyLoginEmailRequest 新环境登录邮箱验证码核验请求。
type VerifyLoginEmailRequest struct {
	ChallengeToken string `json:"challenge_token" binding:"required"`
	Code           string `json:"code" binding:"required"`
}

// VerifyLoginEmail 完成新环境登录邮箱验证码验证并签发受信任设备 Cookie。
func (h *UserLoginHandler) VerifyLoginEmail(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	var req VerifyLoginEmailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ginutil.RespondBindError(c, err)
		return
	}
	envAuth, ok := h.auth.(UserLoginEnvAuth)
	if !ok {
		ginutil.RespondError(c, response.CodeInternal, "error.email_auth_unavailable", nil)
		return
	}
	res, err := envAuth.VerifyLoginEmail(c.Request.Context(), req.ChallengeToken, req.Code, c.GetHeader("User-Agent"))
	if err != nil {
		switch {
		case errors.Is(err, ErrLoginEmailChallengeInvalid):
			h.recordLogin(c, "", 0, constants.LoginLogStatusFailed, constants.LoginLogFailReasonChallengeInvalid, constants.LoginLogSourceWeb)
			ginutil.RespondError(c, response.CodeUnauthorized, "error.login_email_challenge_invalid", nil)
		case errors.Is(err, ErrVerifyCodeInvalid):
			h.recordLogin(c, "", 0, constants.LoginLogStatusFailed, constants.LoginLogFailReasonInvalidCredentials, constants.LoginLogSourceWeb)
			ginutil.RespondError(c, response.CodeBadRequest, "error.verify_code_invalid", nil)
		case errors.Is(err, ErrVerifyCodeExpired):
			h.recordLogin(c, "", 0, constants.LoginLogStatusFailed, constants.LoginLogFailReasonChallengeInvalid, constants.LoginLogSourceWeb)
			ginutil.RespondError(c, response.CodeBadRequest, "error.verify_code_expired", nil)
		case errors.Is(err, ErrVerifyCodeAttemptsExceeded):
			h.recordLogin(c, "", 0, constants.LoginLogStatusFailed, constants.LoginLogFailReasonTooManyAttempts, constants.LoginLogSourceWeb)
			ginutil.RespondError(c, response.CodeTooManyRequests, "error.verify_code_attempts_exceeded", nil)
		case errors.Is(err, ErrUserDisabled):
			h.recordLogin(c, "", 0, constants.LoginLogStatusFailed, constants.LoginLogFailReasonUserDisabled, constants.LoginLogSourceWeb)
			ginutil.RespondError(c, response.CodeUnauthorized, "error.user_disabled", nil)
		default:
			ginutil.RespondError(c, response.CodeInternal, "error.login_failed", err)
		}
		return
	}
	setTrustedDeviceCookie(c, res.TrustedDeviceToken)
	h.recordLogin(c, res.User.Email, res.User.ID, constants.LoginLogStatusSuccess, "", constants.LoginLogSourceWeb)
	response.Success(c, gin.H{
		"requires_totp":         false,
		"requires_email_verify": false,
		"user":                  userpresenter.NewUserAuthBriefResp(res.User),
		"token":                 res.Token,
		"expires_at":            res.ExpiresAt.Format("2006-01-02T15:04:05Z07:00"),
	})
}

// ResendLoginEmailRequest 重发新环境登录邮箱验证码请求。
type ResendLoginEmailRequest struct {
	ChallengeToken string `json:"challenge_token" binding:"required"`
}

// ResendLoginEmail 重发新环境登录安全验证码。
func (h *UserLoginHandler) ResendLoginEmail(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	var req ResendLoginEmailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ginutil.RespondBindError(c, err)
		return
	}
	envAuth, ok := h.auth.(UserLoginEnvAuth)
	if !ok {
		ginutil.RespondError(c, response.CodeInternal, "error.email_auth_unavailable", nil)
		return
	}
	if err := envAuth.ResendLoginEmail(c.Request.Context(), req.ChallengeToken, i18n.ResolveLocale(c)); err != nil {
		switch {
		case errors.Is(err, ErrLoginEmailChallengeInvalid):
			ginutil.RespondError(c, response.CodeUnauthorized, "error.login_email_challenge_invalid", nil)
		case errors.Is(err, ErrUserDisabled):
			ginutil.RespondError(c, response.CodeUnauthorized, "error.user_disabled", nil)
		case errors.Is(err, ErrVerifyCodeTooFrequent):
			ginutil.RespondError(c, response.CodeTooManyRequests, "error.verify_code_too_frequent", nil)
		case errors.Is(err, ErrEmailServiceDisabled),
			errors.Is(err, ErrEmailServiceNotConfigured):
			ginutil.RespondError(c, response.CodeInternal, "error.email_service_not_configured", err)
		default:
			ginutil.RespondError(c, response.CodeInternal, "error.send_verify_code_failed", err)
		}
		return
	}
	response.Success(c, gin.H{"sent": true})
}

var (
	ErrInviteCodeRequired  = errors.New("invite code required")
	ErrInviteCodeInvalid   = errors.New("invite code invalid")
	ErrInviteTicketInvalid = errors.New("invite ticket invalid")
)

// UserInviteService 是第三方登录补填邀请码端点所需的端口。
type UserInviteService interface {
	CompleteInviteRegistration(ctx context.Context, ticket, inviteCode string) (*AuthLoginResult, error)
}

// UserInviteHandler 处理第三方登录新用户补填邀请码。
type UserInviteHandler struct {
	service  UserInviteService
	recorder LoginRecorder
}

func NewUserInviteHandler(service UserInviteService, recorder LoginRecorder) *UserInviteHandler {
	if service == nil {
		panic("user invite handler: service is nil")
	}
	return &UserInviteHandler{service: service, recorder: recorder}
}

type completeInviteRegistrationRequest struct {
	Ticket     string `json:"invite_ticket" binding:"required"`
	InviteCode string `json:"invite_code" binding:"required"`
}

// respondInviteRequired 第三方身份已验证但需邀请码才能建号：返回票据，由前端弹窗补填后调用 /auth/invite/complete。
func respondInviteRequired(c *gin.Context, res *AuthLoginResult) bool {
	if res == nil || !res.InviteRequired {
		return false
	}
	response.Success(c, gin.H{
		"invite_required": true,
		"invite_ticket":   res.InviteTicket,
	})
	return true
}

// respondInviteCodeError 映射邀请码相关错误，未命中返回 false。
func respondInviteCodeError(c *gin.Context, err error) bool {
	switch {
	case errors.Is(err, ErrInviteCodeRequired):
		ginutil.RespondError(c, response.CodeBadRequest, "error.invite_code_required", nil)
	case errors.Is(err, ErrInviteCodeInvalid):
		ginutil.RespondError(c, response.CodeBadRequest, "error.invite_code_invalid", nil)
	case errors.Is(err, ErrInviteTicketInvalid):
		ginutil.RespondError(c, response.CodeBadRequest, "error.invite_ticket_invalid", nil)
	default:
		return false
	}
	return true
}

// CompleteInviteRegistration POST /auth/invite/complete
func (h *UserInviteHandler) CompleteInviteRegistration(c *gin.Context) {
	var req completeInviteRegistrationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ginutil.RespondBindError(c, err)
		return
	}
	res, err := h.service.CompleteInviteRegistration(c.Request.Context(), req.Ticket, req.InviteCode)
	if err != nil {
		recordLoginEvent(h.recorder, c, "", 0, constants.LoginLogStatusFailed, constants.LoginLogFailReasonBadRequest, constants.LoginLogSourceWeb)
		if respondInviteCodeError(c, err) {
			return
		}
		switch {
		case errors.Is(err, ErrRegistrationDisabled):
			ginutil.RespondError(c, response.CodeForbidden, "error.registration_disabled", nil)
		case errors.Is(err, ErrEmailDomainNotAllowed):
			ginutil.RespondError(c, response.CodeBadRequest, "error.email_domain_not_allowed", nil)
		case errors.Is(err, ErrUserDisabled):
			ginutil.RespondError(c, response.CodeUnauthorized, "error.user_disabled", nil)
		default:
			ginutil.RespondError(c, response.CodeInternal, "error.login_failed", err)
		}
		return
	}
	if res == nil || res.User == nil {
		ginutil.RespondError(c, response.CodeInternal, "error.login_failed", errors.New("invite registration returned an empty result"))
		return
	}
	if res.RequiresTOTP {
		response.Success(c, gin.H{
			"requires_totp":        true,
			"challenge_token":      res.ChallengeToken,
			"challenge_expires_at": res.ChallengeExpiresAt.Format("2006-01-02T15:04:05Z07:00"),
		})
		return
	}
	recordLoginEvent(h.recorder, c, res.User.Email, res.User.ID, constants.LoginLogStatusSuccess, "", constants.LoginLogSourceWeb)
	response.Success(c, gin.H{
		"requires_totp": false,
		"user":          userpresenter.NewUserAuthBriefResp(res.User),
		"token":         res.Token,
		"expires_at":    res.ExpiresAt.Format("2006-01-02T15:04:05Z07:00"),
	})
}
