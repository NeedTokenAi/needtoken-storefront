package chatgpt

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	diagnosticscontract "github.com/dujiao-next/internal/modules/diagnostics/contract"
)

const fixtureToken = "fixture-access-token-DO-NOT-LOG"
const accountPayload = `{"accounts":{"personal":{"account":{"account_id":"personal-1","plan_type":"plus","is_default":true,"email":"private@example.test"},"entitlement":{"expires_at":"2026-12-01T00:00:00Z"}},"workspace":{"account":{"account_id":"workspace-2","plan_type":"team"}}}}`

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) { return f(request) }

// The production client still creates fixed https://chatgpt.com requests. Only
// this test transport redirects those requests to an isolated fixture server.
func fixtureClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	upstream, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	client := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		if request.URL.Scheme != "https" || request.URL.Host != "chatgpt.com" || request.Method != http.MethodGet {
			t.Errorf("unexpected diagnostic target: %s %s", request.Method, request.URL)
		}
		if request.Header.Get("Authorization") != "Bearer "+fixtureToken {
			t.Error("missing fixture credential")
		}
		clone := request.Clone(request.Context())
		copyURL := *request.URL
		clone.URL = &copyURL
		clone.URL.Scheme, clone.URL.Host = upstream.Scheme, upstream.Host
		clone.Host = upstream.Host
		return http.DefaultTransport.RoundTrip(clone)
	})}
	return NewClient(client)
}

func TestInspectListsAccountsWithoutChoosingWorkspaceOrFetchingQuota(t *testing.T) {
	var calls atomic.Int32
	client := fixtureClient(t, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.URL.Path != "/backend-api/accounts/check/v4-2023-04-27" || r.Header.Get("chatgpt-account-id") != "" {
			t.Errorf("unexpected account-list request: %s", r.URL.Path)
		}
		fmt.Fprint(w, accountPayload)
	})
	result, err := client.Inspect(context.Background(), diagnosticscontract.Input{AccessToken: fixtureToken})
	if err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 1 || len(result.Accounts) != 2 || result.Accounts[0].ID != "personal-1" || result.Accounts[0].Plan != "plus" || result.Accounts[1].Plan != "team" || result.UsageStatus != "not_requested" || result.SelectedAccountID != "" || result.Usage != nil {
		t.Fatalf("unexpected sanitized result: %+v", result)
	}
	if result.Accounts[0].ExpiresAt == nil || result.Accounts[0].ExpiresAt.Format(time.RFC3339) != "2026-12-01T00:00:00Z" {
		t.Fatal("expiry was not preserved for the correct account")
	}
	encoded, _ := json.Marshal(result)
	for _, private := range []string{fixtureToken, "private@example.test", "email", "access_token"} {
		if strings.Contains(string(encoded), private) {
			t.Errorf("private field leaked: %s", private)
		}
	}
}

func TestInspectFetchesUsageOnlyForExplicitVerifiedAccount(t *testing.T) {
	var calls atomic.Int32
	client := fixtureClient(t, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.URL.Path == "/backend-api/accounts/check/v4-2023-04-27" {
			fmt.Fprint(w, accountPayload)
			return
		}
		if r.URL.Path != "/backend-api/wham/usage" || r.Header.Get("chatgpt-account-id") != "workspace-2" || r.Header.Get("originator") != "codex_cli_rs" {
			t.Errorf("usage was not scoped to explicit selected workspace: %s", r.URL.Path)
		}
		fmt.Fprint(w, `{"email":"private@example.test","plan_type":"not-used","rate_limit":{"primary_window":{"used_percent":12.5,"limit_window_seconds":18000,"reset_at":1794693600},"secondary_window":{"used_percent":0,"limit_window_seconds":604800,"reset_after_seconds":100}}}`)
	})
	client.now = func() time.Time { return time.Date(2026, 11, 14, 0, 0, 0, 0, time.UTC) }
	result, err := client.Inspect(context.Background(), diagnosticscontract.Input{AccessToken: fixtureToken, AccountID: "workspace-2"})
	if err != nil || calls.Load() != 2 || result.UsageStatus != "ready" || result.SelectedAccountID != "workspace-2" || result.Usage == nil {
		t.Fatalf("usage result missing: %+v, %v", result, err)
	}
	if result.Usage.Primary.UsedPercent != 12.5 || result.Usage.Secondary.UsedPercent != 0 || result.Usage.Secondary.ResetAt == nil || !result.Usage.Secondary.ResetAt.Equal(result.CheckedAt.Add(100*time.Second)) {
		t.Fatalf("usage parsed incorrectly: %+v", result.Usage)
	}
}

func TestInspectRejectsForeignAccountBeforeQuotaRequest(t *testing.T) {
	var calls atomic.Int32
	client := fixtureClient(t, func(w http.ResponseWriter, _ *http.Request) { calls.Add(1); fmt.Fprint(w, accountPayload) })
	_, err := client.Inspect(context.Background(), diagnosticscontract.Input{AccessToken: fixtureToken, AccountID: "someone-else"})
	if !errors.Is(err, diagnosticscontract.ErrAccountNotFound) || calls.Load() != 1 {
		t.Fatalf("foreign account was queried: calls=%d, error=%v", calls.Load(), err)
	}
}

func TestInspectRejectsMalformedCredentialWithoutRequests(t *testing.T) {
	client := NewClient(&http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		t.Fatal("invalid credential reached upstream")
		return nil, nil
	})})
	for _, token := range []string{"", "tiny", "sk-unsupported-api-token", "at-unsupported-personal-token", "Bearer " + fixtureToken, fixtureToken + "\r\nheader", strings.Repeat("a", 16*1024+1)} {
		if _, err := client.Inspect(context.Background(), diagnosticscontract.Input{AccessToken: token}); !errors.Is(err, diagnosticscontract.ErrInvalidInput) {
			t.Errorf("expected invalid input, got %v", err)
		}
	}
	if _, err := client.Inspect(context.Background(), diagnosticscontract.Input{AccessToken: fixtureToken, AccountID: "../another?account"}); !errors.Is(err, diagnosticscontract.ErrInvalidInput) {
		t.Fatal("invalid account ID accepted")
	}
}

func TestInspectMapsUpstreamFailuresWithoutReflectingBodies(t *testing.T) {
	for _, row := range []struct {
		name      string
		status    int
		challenge bool
		want      error
	}{
		{"unauthorized", 401, false, diagnosticscontract.ErrUnauthorized},
		{"forbidden", 403, false, diagnosticscontract.ErrChallenge},
		{"challenge-header", 200, true, diagnosticscontract.ErrChallenge},
		{"rate-limit", 429, false, diagnosticscontract.ErrUnavailable},
		{"redirect", 302, false, diagnosticscontract.ErrChallenge},
	} {
		t.Run(row.name, func(t *testing.T) {
			var calls atomic.Int32
			client := fixtureClient(t, func(w http.ResponseWriter, _ *http.Request) {
				calls.Add(1)
				w.Header().Set("Location", "https://untrusted.invalid/collect")
				if row.challenge {
					w.Header().Set("cf-mitigated", "challenge")
				}
				w.WriteHeader(row.status)
				fmt.Fprint(w, fixtureToken+" private@example.test")
			})
			_, err := client.Inspect(context.Background(), diagnosticscontract.Input{AccessToken: fixtureToken})
			if !errors.Is(err, row.want) || calls.Load() != 1 || strings.Contains(fmt.Sprint(err), fixtureToken) {
				t.Fatalf("unsafe failure handling: %v, requests=%d", err, calls.Load())
			}
		})
	}
}

func TestInspectCapsKnownAndStreamingResponses(t *testing.T) {
	for _, streaming := range []bool{false, true} {
		t.Run(fmt.Sprint(streaming), func(t *testing.T) {
			client := fixtureClient(t, func(w http.ResponseWriter, _ *http.Request) {
				if streaming {
					w.(http.Flusher).Flush()
				} else {
					w.Header().Set("Content-Length", fmt.Sprint(maxResponseSize+1))
				}
				fmt.Fprint(w, strings.Repeat("a", maxResponseSize+1))
			})
			if _, err := client.Inspect(context.Background(), diagnosticscontract.Input{AccessToken: fixtureToken}); !errors.Is(err, diagnosticscontract.ErrInvalidResponse) {
				t.Fatalf("oversized upstream response accepted: %v", err)
			}
		})
	}
}

func TestInspectRejectsMalformedAccountsAndRedactsUnknownPlans(t *testing.T) {
	for _, payload := range []string{`{`, `{}`, `{"accounts":{}}`, `{"accounts":{"x":{"account":null}}}`, `{"accounts":{"x":{"account":{"account_id":"a","plan_type":"plus"},"entitlement":{"expires_at":"bad"}}}}`, `{"accounts":{"x":{"account":{"account_id":"duplicate","plan_type":"plus"}},"y":{"account":{"account_id":"duplicate","plan_type":"team"}}}}`} {
		if _, err := parseAccounts([]byte(payload), fixtureToken); !errors.Is(err, diagnosticscontract.ErrInvalidResponse) {
			t.Errorf("malformed account payload accepted: %s", payload)
		}
	}
	accounts, err := parseAccounts([]byte(`{"accounts":{"a":{"account":{"account_id":"account-1","plan_type":"`+fixtureToken+`"}}}}`), fixtureToken)
	if err != nil || len(accounts) != 1 || accounts[0].Plan != "unknown" {
		t.Fatal("unexpected upstream plan text must not be reflected")
	}
}

func TestInspectQuotaFailurePreservesVerifiedAccountWithoutInventingUsage(t *testing.T) {
	client := fixtureClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/backend-api/wham/usage" {
			w.WriteHeader(401)
			return
		}
		fmt.Fprint(w, accountPayload)
	})
	result, err := client.Inspect(context.Background(), diagnosticscontract.Input{AccessToken: fixtureToken, AccountID: "personal-1"})
	if err != nil || len(result.Accounts) != 2 || result.Usage != nil || result.UsageStatus != "unavailable" {
		t.Fatalf("quota failure invented data or erased verified account: %+v %v", result, err)
	}
	for _, body := range []string{`{}`, `{"rate_limit":null}`, `{"rate_limit":{"primary_window":{"used_percent":101,"limit_window_seconds":18000}}}`, `{"rate_limit":{"primary_window":{"used_percent":0}}}`} {
		if parseUsage([]byte(body), time.Now()) != nil {
			t.Errorf("unusable quota accepted: %s", body)
		}
	}
}

func TestInspectHonorsCancellationAndClientDeadline(t *testing.T) {
	client := fixtureClient(t, func(w http.ResponseWriter, r *http.Request) { <-r.Context().Done() })
	client.client.Timeout = 25 * time.Millisecond
	started := time.Now()
	_, err := client.Inspect(context.Background(), diagnosticscontract.Input{AccessToken: fixtureToken})
	if !errors.Is(err, diagnosticscontract.ErrUnavailable) || time.Since(started) > time.Second {
		t.Fatalf("timeout not bounded: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := client.Inspect(ctx, diagnosticscontract.Input{AccessToken: fixtureToken}); !errors.Is(err, diagnosticscontract.ErrUnavailable) {
		t.Fatalf("cancelled request: %v", err)
	}
}
