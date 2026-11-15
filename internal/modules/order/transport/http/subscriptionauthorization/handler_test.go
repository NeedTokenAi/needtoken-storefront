package subscriptionauthorizationhttp

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	subscriptionauth "github.com/dujiao-next/internal/modules/order/application/subscriptionauthorization"
	"github.com/gin-gonic/gin"
)

func TestGuestAuthorizationRejectsMissingPrivateLookupAndCredentialBody(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := New(subscriptionauth.New(subscriptionauth.Config{}, nil, nil, nil, nil))
	router := gin.New()
	RegisterGuestReadRoutes(router, h)
	RegisterGuestWriteRoutes(router, h)
	for _, test := range []struct {
		body, authorization string
		want                int
	}{
		{`{"operation_id":"auth_one"}`, "", http.StatusNotFound},
		{`{"operation_id":"auth_one","credential":"must-not-enter-storefront"}`, "Guest " + base64.RawURLEncoding.EncodeToString([]byte("owner@example.test\nprivate-password")), http.StatusBadRequest},
	} {
		request := httptest.NewRequest(http.MethodPost, "/orders/ORDER-1/subscription-authorization/challenge", strings.NewReader(test.body))
		request.Header.Set("Content-Type", "application/json")
		if test.authorization != "" {
			request.Header.Set("Authorization", test.authorization)
		}
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)
		var envelope struct {
			StatusCode int `json:"status_code"`
		}
		if json.Unmarshal(response.Body.Bytes(), &envelope) != nil || envelope.StatusCode != test.want {
			t.Fatalf("response=%d body=%s", response.Code, response.Body.String())
		}
		if strings.Contains(response.Body.String(), "must-not-enter-storefront") || response.Header().Get("Cache-Control") != "no-store" {
			t.Fatal("sensitive request was echoed or cached")
		}
	}
}

func TestConsentRequiresSingleSignatureHeadersAndForbidsQuery(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := New(subscriptionauth.New(subscriptionauth.Config{}, nil, nil, nil, nil))
	router := gin.New()
	RegisterConsentRoute(router, h)
	for _, path := range []string{"/subscription-authorization-consents/auth_one", "/subscription-authorization-consents/auth_one?token=unsafe"} {
		request := httptest.NewRequest(http.MethodGet, path, nil)
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)
		var envelope struct {
			StatusCode int `json:"status_code"`
		}
		if json.Unmarshal(response.Body.Bytes(), &envelope) != nil || envelope.StatusCode == 0 || response.Header().Get("Cache-Control") != "no-store" {
			t.Fatal("unsigned consent accepted or cached")
		}
	}
}
