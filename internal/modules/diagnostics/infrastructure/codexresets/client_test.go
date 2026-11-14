package codexresets

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	diagnosticscontract "github.com/dujiao-next/internal/modules/diagnostics/contract"
)

type transportFunc func(*http.Request) (*http.Response, error)

func (f transportFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

const fixturePage = `{"data":[{"id":"2120999993079451919","reset_type":"banked","announced_at":"2026-11-12T21:59:39.000Z","text":"Banked reset announcement","source":{"type":"x_post","author":"thsottiaux","url":"https://x.com/thsottiaux/status/2120999993079451919"}},{"id":"observed-2114588174249071102","reset_type":"regular","announced_at":"2026-11-10T12:00:00Z","text":"Third-party observation","source":{"type":"observed","url":"https://x.com/thsottiaux/status/2114588174249071102"}}],"pagination":{"has_more":true},"meta":{"generated_at":"2099-01-01T00:00:00Z"}}`

func upstreamResponse(req *http.Request, status int, body string) *http.Response {
	return &http.Response{StatusCode: status, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(body)), ContentLength: -1, Request: req}
}

func TestClientUsesFixedPublicPageWithoutCredentials(t *testing.T) {
	var calls int
	client := NewClient(&http.Client{Timeout: time.Hour, CheckRedirect: func(*http.Request, []*http.Request) error { t.Fatal("caller redirect policy used"); return nil }, Transport: transportFunc(func(req *http.Request) (*http.Response, error) {
		calls++
		if req.Method != http.MethodGet || req.URL.String() != feedURL || req.Header.Get("Authorization") != "" || req.Header.Get("Cookie") != "" || req.Header.Get("Accept") != "application/json" {
			t.Fatalf("unexpected public request: %s %s", req.Method, req.URL)
		}
		if deadline, ok := req.Context().Deadline(); !ok || time.Until(deadline) > requestTimeout {
			t.Fatal("request has no bounded deadline")
		}
		return upstreamResponse(req, 200, fixturePage), nil
	})})
	if client.client.Timeout != requestTimeout || client.client.Jar != nil {
		t.Fatal("unsafe client settings")
	}
	page, err := client.Fetch(context.Background())
	if err != nil || calls != 1 || !page.HasMore || len(page.Records) != 2 {
		t.Fatalf("unexpected page: %#v %v", page, err)
	}
	if page.Records[0].Type != "banked" || page.Records[1].Source.Type != "observed" || page.Records[1].Source.Author != "" || page.Records[0].AnnouncedAt.Year() != 2026 {
		t.Fatalf("upstream semantics changed: %#v", page.Records)
	}
}

func TestPageFiltersUnsafeRecordsAndPreservesObservedAttribution(t *testing.T) {
	body := `{"data":[
{"id":"valid","reset_type":"regular","announced_at":"2026-11-12T21:59:39Z","text":" valid ","source":{"type":"x_post","url":"https://x.com/author/status/123?s=20#fragment"}},
{"id":"bad-date","reset_type":"banked","announced_at":"2026-02-30T00:00:00Z","text":"bad","source":{"type":"observed"}},
{"id":"bad-type","reset_type":"scheduled","announced_at":"2026-11-12T00:00:00Z","text":"bad","source":{"type":"observed"}},
{"id":"bad-source","reset_type":"regular","announced_at":"2026-11-12T00:00:00Z","text":"bad","source":{"type":"official"}},
{"id":"bad-url","reset_type":"regular","announced_at":"2026-11-12T00:00:00Z","text":"bad","source":{"type":"x_post","url":"https://x.com.evil.invalid/author/status/123"}},
{"id":"missing-post","reset_type":"regular","announced_at":"2026-11-12T00:00:00Z","text":"bad","source":{"type":"x_post"}},
{"id":"observation","reset_type":"banked","announced_at":"2026-11-12T00:00:00Z","text":"observed","source":{"type":"observed","url":"javascript:alert(1)"}},
{"id":"valid","reset_type":"regular","announced_at":"2026-11-12T00:00:00Z","text":"duplicate","source":{"type":"observed"}},
{"id":99,"reset_type":"regular"}
],"pagination":{"has_more":false}}`
	page, err := parsePage([]byte(body))
	if err != nil || len(page.Records) != 2 || page.HasMore {
		t.Fatalf("unsafe filtering: %#v %v", page, err)
	}
	if page.Records[0].Source.URL != "https://x.com/author/status/123" || page.Records[0].Text != "valid" || page.Records[1].Source.URL != "" || page.Records[1].Source.Type != "observed" {
		t.Fatalf("unsafe attribution: %#v", page.Records)
	}
	for _, raw := range []string{"http://x.com/a/status/123", "https://user@x.com/a/status/123", "https://x.com:443/a/status/123", "https://x.com/a/status/not-numeric", "https://x.com/intent/tweet", "https://twitter.com/a%2fb/status/123"} {
		if safePostURL(raw) != "" {
			t.Fatalf("unsafe post accepted: %s", raw)
		}
	}
}

func TestInvalidSchemaCannotRefreshButEmptyPageIsValid(t *testing.T) {
	for _, body := range []string{`{}`, `{"data":null,"pagination":{"has_more":false}}`, `{"data":{},"pagination":{"has_more":false}}`, `{"data":[],"pagination":{}}`, `{"data":[],"pagination":{"has_more":"false"}}`, `{"data":[],"pagination":{"has_more":false}} {}`, `{"data":[{}],"pagination":{"has_more":false}}`, `<html>challenge</html>`} {
		if _, err := parsePage([]byte(body)); !errors.Is(err, diagnosticscontract.ErrResetFeedUnavailable) {
			t.Fatalf("invalid feed accepted: %s", body)
		}
	}
	page, err := parsePage([]byte(`{"data":[],"pagination":{"has_more":false}}`))
	if err != nil || page.Records == nil || len(page.Records) != 0 {
		t.Fatalf("empty page lost: %#v %v", page, err)
	}
}

func TestClientRejectsRedirectsErrorsAndOversizedBodies(t *testing.T) {
	for _, variant := range []string{"redirect", "limited", "declared-size", "stream-size"} {
		t.Run(variant, func(t *testing.T) {
			var calls atomic.Int32
			client := NewClient(&http.Client{Transport: transportFunc(func(req *http.Request) (*http.Response, error) {
				calls.Add(1)
				resp := upstreamResponse(req, 200, fixturePage)
				switch variant {
				case "redirect":
					resp.StatusCode = 302
					resp.Header.Set("Location", "https://private.invalid/reset-feed")
				case "limited":
					resp.StatusCode = 429
				case "declared-size":
					resp.ContentLength = maxResponseSize + 1
				case "stream-size":
					resp.Body = io.NopCloser(strings.NewReader(strings.Repeat("x", maxResponseSize+1)))
				}
				return resp, nil
			})})
			if _, err := client.Fetch(context.Background()); !errors.Is(err, diagnosticscontract.ErrResetFeedUnavailable) || calls.Load() != 1 {
				t.Fatalf("unsafe upstream handling: calls=%d err=%v", calls.Load(), err)
			}
		})
	}
}

func TestClientBoundsRequestsAndPreservesCallerCancellation(t *testing.T) {
	client := NewClient(&http.Client{Timeout: 5 * time.Millisecond, Transport: transportFunc(func(req *http.Request) (*http.Response, error) {
		<-req.Context().Done()
		return nil, req.Context().Err()
	})})
	if _, err := client.Fetch(context.Background()); !errors.Is(err, diagnosticscontract.ErrResetFeedUnavailable) {
		t.Fatal("timeout was not bounded")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := client.Fetch(ctx); !errors.Is(err, diagnosticscontract.ErrResetFeedUnavailable) {
		t.Fatal("cancelled request did not fail")
	}
}
