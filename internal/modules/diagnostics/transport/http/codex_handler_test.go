package diagnosticshttp

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	diagnosticscontract "github.com/dujiao-next/internal/modules/diagnostics/contract"
	"github.com/dujiao-next/internal/platform/http/response"

	"github.com/gin-gonic/gin"
)

type inspectorFunc func(context.Context, diagnosticscontract.Input) (diagnosticscontract.Result, error)

func (f inspectorFunc) Inspect(ctx context.Context, input diagnosticscontract.Input) (diagnosticscontract.Result, error) {
	return f(ctx, input)
}

func testRouter(inspector diagnosticscontract.Inspector) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	group := router.Group("/api/v1/public/tools", NoStore())
	RegisterPublicRoutes(group, NewHandler(inspector))
	return router
}

func callDiagnostic(router http.Handler, body string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(http.MethodPost, "/api/v1/public/tools/codex-account", strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-Lang", "en-US")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	return recorder
}

func TestHandlerBindsOnlyBoundedDiagnosticInput(t *testing.T) {
	var calls int
	router := testRouter(inspectorFunc(func(_ context.Context, input diagnosticscontract.Input) (diagnosticscontract.Result, error) {
		calls++
		if input.AccessToken != "fixture-access-token" || input.AccountID != "account-1" {
			t.Fatalf("wrong input: account=%s", input.AccountID)
		}
		return diagnosticscontract.Result{CheckedAt: time.Now().UTC(), Accounts: []diagnosticscontract.Account{{ID: "account-1", Plan: "plus"}}, UsageStatus: "not_requested"}, nil
	}))
	recorder := callDiagnostic(router, `{"access_token":"fixture-access-token","account_id":"account-1"}`)
	if calls != 1 || recorder.Code != 200 || !strings.Contains(recorder.Body.String(), `"status_code":0`) || strings.Contains(recorder.Body.String(), "fixture-access-token") {
		t.Fatalf("unexpected response: %s", recorder.Body.String())
	}
	if recorder.Header().Get("Cache-Control") != "private, no-store, max-age=0" || recorder.Header().Get("Pragma") != "no-cache" {
		t.Fatal("diagnostic response can be cached")
	}
	for _, body := range []string{`{`, `{"access_token":"fixture-access-token"} {}`, `{"access_token":"fixture-access-token","url":"https://other.invalid"}`, `{"access_token":"` + strings.Repeat("x", maxRequestSize) + `"}`} {
		calls = 0
		recorder = callDiagnostic(router, body)
		if calls != 0 || !strings.Contains(recorder.Body.String(), `"error_code":"invalid_input"`) || strings.Contains(recorder.Body.String(), "fixture-access-token") {
			t.Fatalf("unsafe malformed input handling: %s", recorder.Body.String())
		}
	}
}

func TestHandlerLocalizesAndRedactsUpstreamErrors(t *testing.T) {
	for _, row := range []struct {
		err  error
		code int
		key  string
	}{
		{diagnosticscontract.ErrInvalidInput, 400, "invalid_input"},
		{diagnosticscontract.ErrUnauthorized, 401, "unauthorized"},
		{diagnosticscontract.ErrChallenge, 403, "challenge"},
		{diagnosticscontract.ErrUnavailable, 502, "unavailable"},
		{diagnosticscontract.ErrInvalidResponse, 502, "invalid_response"},
		{diagnosticscontract.ErrAccountNotFound, 400, "account_not_found"},
		{errors.New("PRIVATE_UPSTREAM_BODY_WITH_SECRET"), 502, "unavailable"},
	} {
		router := testRouter(inspectorFunc(func(context.Context, diagnosticscontract.Input) (diagnosticscontract.Result, error) {
			return diagnosticscontract.Result{}, row.err
		}))
		recorder := callDiagnostic(router, `{"access_token":"fixture-access-token"}`)
		var envelope response.Response
		if json.Unmarshal(recorder.Body.Bytes(), &envelope) != nil || envelope.StatusCode != row.code {
			t.Fatalf("unexpected response: %s", recorder.Body.String())
		}
		data, _ := envelope.Data.(map[string]interface{})
		if data["error_code"] != row.key || strings.HasPrefix(envelope.Msg, "error.") || strings.Contains(recorder.Body.String(), "PRIVATE_") || strings.Contains(recorder.Body.String(), "fixture-access-token") {
			t.Fatalf("unsafe error response: %s", recorder.Body.String())
		}
	}
}
