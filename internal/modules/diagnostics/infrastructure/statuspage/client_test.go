package statuspage

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	diagnosticscontract "github.com/dujiao-next/internal/modules/diagnostics/contract"
)

type transportFunc func(*http.Request) (*http.Response, error)

func (f transportFunc) RoundTrip(request *http.Request) (*http.Response, error) { return f(request) }

const summaryFixture = `{"page":{"name":"Official provider","url":"https://other.invalid/","updated_at":"2026-01-01T00:00:00Z"},"status":{"indicator":"none","description":"All Systems Operational"},"components":[{"id":"chatgpt-group","name":"ChatGPT","status":"operational","group":true,"components":["gpt-login"]},{"id":"gpt-login","name":"Login","status":"operational","group_id":"chatgpt-group"}],"private":"must-not-be-returned"}`
const componentsFixture = `{"components":[{"id":"gpt-login","name":"Login","status":"degraded_performance"},{"id":"api-login","name":"Login","status":"operational"},{"id":"gpt-voice","name":"Voice","status":"operational","group_id":"chatgpt-group"}]}`

func upstreamResponse(request *http.Request, status int, body string) *http.Response {
	return &http.Response{StatusCode: status, Header: http.Header{"Content-Type": {"application/json; charset=utf-8"}}, Body: io.NopCloser(strings.NewReader(body)), ContentLength: -1, Request: request}
}

func TestOfficialRequestsAreFixedReadOnlyAndOpenAIComponentsAreComplete(t *testing.T) {
	var requested []string
	client := NewClient(&http.Client{Timeout: time.Hour, CheckRedirect: func(*http.Request, []*http.Request) error { t.Fatal("caller redirect policy used"); return nil }, Transport: transportFunc(func(request *http.Request) (*http.Response, error) {
		requested = append(requested, request.URL.String())
		if request.Method != http.MethodGet || request.Header.Get("Authorization") != "" || request.Header.Get("Cookie") != "" || request.Header.Get("Accept") != "application/json" {
			t.Fatal("unexpected credential or mutation in public status request")
		}
		if deadline, ok := request.Context().Deadline(); !ok || time.Until(deadline) > requestTimeout {
			t.Fatal("official request has no bounded context")
		}
		if request.URL.Path == "/api/v2/components.json" {
			return upstreamResponse(request, 200, componentsFixture), nil
		}
		return upstreamResponse(request, 200, summaryFixture), nil
	})})
	if client.client.Timeout != requestTimeout || client.client.Jar != nil {
		t.Fatal("unsafe status client settings")
	}
	summary, err := client.Fetch(context.Background(), diagnosticscontract.ServiceStatusOpenAI)
	if err != nil || len(requested) != 2 || requested[0] != "https://status.openai.com/api/v2/summary.json" || requested[1] != "https://status.openai.com/api/v2/components.json" {
		t.Fatalf("official endpoints changed: %v %v", requested, err)
	}
	if len(summary.Components) != 4 || summary.Components[1].ID != "gpt-login" || summary.Components[1].Status != "degraded_performance" || summary.Components[1].GroupID != "chatgpt-group" || !summary.Components[0].Group || summary.Components[0].Components[0] != "gpt-login" {
		t.Fatalf("complete component IDs/status/group metadata were lost: %#v", summary.Components)
	}
	if summary.Page.URL != "https://status.openai.com/" || summary.Page.UpdatedAt == nil || summary.Page.UpdatedAt.Year() != 2026 || summary.Page.UpdatedAt.Month() != time.January {
		t.Fatalf("source URL/date relabeled: %#v", summary.Page)
	}
	encoded, _ := json.Marshal(summary)
	if strings.Contains(string(encoded), "must-not-be-returned") || strings.Contains(string(encoded), "other.invalid") {
		t.Fatal("unexpected upstream fields were reflected")
	}
	requested = nil
	if _, err := client.Fetch(context.Background(), diagnosticscontract.ServiceStatusClaude); err != nil || len(requested) != 1 || requested[0] != "https://status.claude.com/api/v2/summary.json" {
		t.Fatalf("Claude source request changed: %v %v", requested, err)
	}
	requested = nil
	if _, err := client.Fetch(context.Background(), "https://private.invalid"); !errors.Is(err, diagnosticscontract.ErrServiceStatusProvider) || len(requested) != 0 {
		t.Fatal("unrecognized provider reached HTTP transport")
	}
}

func TestOpenAICompleteListFailureCannotBecomeFreshPartialStatus(t *testing.T) {
	client := NewClient(&http.Client{Transport: transportFunc(func(request *http.Request) (*http.Response, error) {
		if request.URL.Path == "/api/v2/components.json" {
			return upstreamResponse(request, 503, "private upstream error"), nil
		}
		return upstreamResponse(request, 200, summaryFixture), nil
	})})
	if _, err := client.Fetch(context.Background(), diagnosticscontract.ServiceStatusOpenAI); !errors.Is(err, diagnosticscontract.ErrServiceStatusUnavailable) {
		t.Fatal("partial OpenAI summary was accepted after complete list failure")
	}
}

func TestOfficialSourceRejectsRedirectsMalformedAndOversizedResponses(t *testing.T) {
	for _, variant := range []string{"redirect", "forbidden", "html", "malformed", "declared-size", "stream-size"} {
		t.Run(variant, func(t *testing.T) {
			calls := 0
			client := NewClient(&http.Client{Transport: transportFunc(func(request *http.Request) (*http.Response, error) {
				calls++
				response := upstreamResponse(request, 200, summaryFixture)
				switch variant {
				case "redirect":
					response.StatusCode = 302
					response.Header.Set("Location", "https://untrusted.invalid/")
				case "forbidden":
					response.StatusCode = 403
				case "html":
					response.Header.Set("Content-Type", "text/html")
				case "malformed":
					response.Body = io.NopCloser(strings.NewReader(`{"page":{},"status":{},"components":null}`))
				case "declared-size":
					response.ContentLength = maxResponseSize + 1
				case "stream-size":
					response.Body = io.NopCloser(strings.NewReader(strings.Repeat("x", maxResponseSize+1)))
				}
				return response, nil
			})})
			if _, err := client.Fetch(context.Background(), diagnosticscontract.ServiceStatusClaude); !errors.Is(err, diagnosticscontract.ErrServiceStatusUnavailable) || calls != 1 || strings.Contains(err.Error(), "private") {
				t.Fatalf("unsafe official response: calls=%d err=%v", calls, err)
			}
		})
	}
}

func TestSourceParserPreservesUnknownStatusesAndDoesNotInventUpdateTime(t *testing.T) {
	body := `{"page":{"name":"Claude"},"status":{"indicator":"unrecognized","description":"New status"},"components":[{"id":"claude-web","name":"claude.ai","status":"unrecognized"}]}`
	summary, err := parseSummary([]byte(body), "https://status.claude.com/")
	if err != nil || summary.Page.UpdatedAt != nil || summary.Status.Indicator != "unknown" || summary.Components[0].Status != "unknown" {
		t.Fatalf("unknown source data became healthy or current: %#v %v", summary, err)
	}
	for _, body := range []string{
		`{}`, `<html>challenge</html>`, summaryFixture + `{}`,
		strings.Replace(summaryFixture, "2026-01-01T00:00:00Z", "2026-02-30T00:00:00Z", 1),
		strings.Replace(summaryFixture, `"gpt-login","name":"Login"`, `"chatgpt-group","name":"Login"`, 1),
		strings.Replace(summaryFixture, `"group_id":"chatgpt-group"`, `"group_id":"invalid/id"`, 1),
	} {
		if _, err := parseSummary([]byte(body), "https://status.openai.com/"); !errors.Is(err, diagnosticscontract.ErrServiceStatusUnavailable) {
			t.Fatal("malformed official source was accepted")
		}
	}
}

func TestStatusSourceHonorsCancellationAndTimeout(t *testing.T) {
	client := NewClient(&http.Client{Timeout: 5 * time.Millisecond, Transport: transportFunc(func(request *http.Request) (*http.Response, error) {
		<-request.Context().Done()
		return nil, request.Context().Err()
	})})
	if _, err := client.Fetch(context.Background(), diagnosticscontract.ServiceStatusClaude); !errors.Is(err, diagnosticscontract.ErrServiceStatusUnavailable) {
		t.Fatal("source request ignored client timeout")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := client.Fetch(ctx, diagnosticscontract.ServiceStatusOpenAI); !errors.Is(err, diagnosticscontract.ErrServiceStatusUnavailable) {
		t.Fatal("source request ignored caller cancellation")
	}
}
