package integrationtest

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/dujiao-next/internal/constants"
	captchahttp "github.com/dujiao-next/internal/modules/captcha/transport/http"
	userauthhttp "github.com/dujiao-next/internal/modules/identity/userauth/transport/http"
	"github.com/gin-gonic/gin"
)

type emailAuthHTTPSettings struct{}

func (emailAuthHTTPSettings) GetEmailVerificationEnabled(bool) (bool, error) { return false, nil }
func (emailAuthHTTPSettings) GetRegistrationEnabled(bool) (bool, error)      { return false, nil }

type emailAuthHTTPPort struct {
	purpose string
	result  *userauthhttp.EmailAuthFlowResult
	err     error
}

func (s *emailAuthHTTPPort) SendVerifyCode(_ context.Context, _, purpose, _, _ string) error {
	s.purpose = purpose
	return nil
}
func (s *emailAuthHTTPPort) VerifyEmailAuth(context.Context, string, string) (*userauthhttp.EmailAuthFlowResult, error) {
	return s.result, s.err
}

type emailAuthCaptcha struct{ scene string }

func (s *emailAuthCaptcha) Verify(scene string, _ captchahttp.CaptchaPayloadRequest, _ string) error {
	s.scene = scene
	return nil
}

func TestEmailAuthHTTPUsesExistingCaptchaRateLimitAndNoStore(t *testing.T) {
	gin.SetMode(gin.TestMode)
	port := &emailAuthHTTPPort{result: &userauthhttp.EmailAuthFlowResult{NextStep: "register", RegistrationCode: "eauth_opaque-ticket", ExpiresAt: time.Now().Add(time.Minute).Format(time.RFC3339)}}
	captcha := &emailAuthCaptcha{}
	handler := userauthhttp.NewUserVerifyHandler(emailAuthHTTPSettings{}, captcha, port)
	router := gin.New()
	limited := 0
	userauthhttp.RegisterUserVerifyAuthRoutes(router.Group("/auth"), handler, func(c *gin.Context) { limited++; c.Next() })
	call := func(path, body string) *httptest.ResponseRecorder {
		request := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
		request.Header.Set("Content-Type", "application/json")
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)
		return response
	}
	response := call("/auth/send-verify-code", `{"email":"owner@example.com","purpose":"email_auth"}`)
	if !strings.Contains(response.Body.String(), `"sent":true`) || port.purpose != constants.VerifyPurposeEmailAuth || captcha.scene != constants.CaptchaSceneRegisterSendCode || limited != 1 {
		t.Fatalf("email-auth send policy changed: %s", response.Body.String())
	}
	response = call("/auth/email/verify", `{"email":"owner@example.com","code":"123456"}`)
	if response.Header().Get("Cache-Control") != "no-store" || !strings.Contains(response.Body.String(), `"next_step":"register"`) || limited != 2 {
		t.Fatalf("verified continuation = %s", response.Body.String())
	}
	port.err = userauthhttp.ErrVerifyCodeInvalid
	response = call("/auth/email/verify", `{"email":"owner@example.com","code":"invalid"}`)
	var result struct {
		StatusCode int             `json:"status_code"`
		Data       json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil || result.StatusCode == 0 || strings.Contains(string(result.Data), "next_step") || strings.Contains(response.Body.String(), "eauth_") || response.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("failed proof disclosed continuation: %s", response.Body.String())
	}
	port.err = nil
	port.result = &userauthhttp.EmailAuthFlowResult{NextStep: "register", RegistrationCode: "eauth_expired", ExpiresAt: time.Now().Add(-time.Minute).Format(time.RFC3339)}
	response = call("/auth/email/verify", `{"email":"owner@example.com","code":"123456"}`)
	if strings.Contains(response.Body.String(), "eauth_expired") {
		t.Fatal("transport exposed expired continuation")
	}
}
