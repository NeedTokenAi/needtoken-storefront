package resetsources

import (
	"context"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

func TestTSalonClientParsesChannelsAndSharesSingleUpstreamRequest(t *testing.T) {
	var calls atomic.Int32
	payload := `{
		"status": "ready",
		"claude": {
			"events": [
				{
					"id": "claude-2120275296637321476",
					"type": "card",
					"category": "banked-card",
					"typeLabelZh": "重置卡",
					"typeLabelEn": "Reset card",
					"scopeZh": "付费用户",
					"scopeEn": "Paid users",
					"reasonZh": "Fable 5.2 上线补偿",
					"reasonEn": "Fable 5.2 launch credit",
					"landedAtBeijing": "2026-11-11 05:59",
					"postUrl": "https://x.com/ClaudeDevs/status/2120275296637321476?s=20"
				},
				{
					"id": "claude-bad-link",
					"type": "reset",
					"reasonEn": "Unsafe link",
					"landedAtBeijing": "2026-11-10 04:00",
					"postUrl": "https://evil.example/ClaudeDevs/status/1"
				}
			]
		},
		"grok": {
			"events": [
				{
					"id": "grok-2119871982946971659",
					"type": "reset",
					"category": "scheduled-global",
					"typeLabelZh": "全局重置",
					"typeLabelEn": "Global reset",
					"scopeZh": "全部用户",
					"scopeEn": "All users",
					"reasonZh": "该记录未说明",
					"reasonEn": "Not specified in this record",
					"landedAtBeijing": "2026-11-10 03:17",
					"postUrl": "https://x.com/xai/status/2119871982946971659"
				}
			]
		}
	}`
	client := NewTSalonClient(&http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		calls.Add(1)
		return &http.Response{StatusCode: 200, Header: make(http.Header), Request: req, Body: io.NopCloser(strings.NewReader(payload))}, nil
	})})

	claudePage, err := client.ChannelSource("claude").Fetch(context.Background())
	if err != nil || len(claudePage.Records) != 2 {
		t.Fatalf("unexpected claude page: %+v err=%v", claudePage, err)
	}
	first := claudePage.Records[0]
	if first.Type != "banked" || first.Source.Author != "ClaudeDevs" || first.Source.URL != "https://x.com/ClaudeDevs/status/2120275296637321476" || first.TextZH != "Fable 5.2 上线补偿" || first.ScopeZH != "付费用户" {
		t.Fatalf("unexpected first claude record: %+v", first)
	}
	if first.AnnouncedAt.Format("2006-01-02T15:04:05Z") != "2026-11-10T21:59:00Z" {
		t.Fatalf("beijing time not converted to UTC: %s", first.AnnouncedAt)
	}
	// Unsafe link stripped and downgraded to observed.
	if claudePage.Records[1].Source.URL != "" || claudePage.Records[1].Source.Type != "observed" {
		t.Fatalf("unsafe post URL not sanitized: %+v", claudePage.Records[1].Source)
	}

	grokPage, err := client.ChannelSource("grok").Fetch(context.Background())
	if err != nil || len(grokPage.Records) != 1 || grokPage.Records[0].Type != "regular" || grokPage.Records[0].Source.Author != "xai" {
		t.Fatalf("unexpected grok page: %+v err=%v", grokPage, err)
	}
	if calls.Load() != 1 {
		t.Fatalf("expected 1 shared upstream request across claude+grok, got %d", calls.Load())
	}
}

func TestAntigravityClientExtractsEscapedRSCRecordsAndRejectsUnsafeLinks(t *testing.T) {
	html := `<!doctype html><html><body><script>self.__next_f.push([1,"{\"events\":[{\"id\":\"cmwq77dh000ic1ypltycs72pu\",\"kind\":\"reset\",\"announcedAt\":\"2026-11-21T09:18:53.000Z\",\"textZh\":\"已重置所有用户在 \\\"Antigravity\\\" 上的 Gemini 额度\",\"text\":\"Reset everyone's \\\"Gemini\\\" quota on Antigravity.\",\"quoteText\":null,\"scope\":\"全体用户 · Gemini 模型\",\"scopeEn\":\"All users · Gemini models\",\"url\":\"https://x.com/_mohansolo/status/2124070033534161407\"},{\"id\":\"cmwpdk3y200ln1ypcwxbt3vnf\",\"kind\":\"preview\",\"announcedAt\":\"2026-11-20T19:26:20.000Z\",\"textZh\":\"预告缓存优化后将发放重置补偿\",\"text\":\"Should be fixed soon!\",\"quoteText\":null,\"scope\":\"全体用户\",\"scopeEn\":\"All users\",\"url\":\"https://x.com/_mohansolo/status/2123860512949825830\"},{\"id\":\"bad-link\",\"kind\":\"reset\",\"announcedAt\":\"2026-11-19T10:00:00.000Z\",\"textZh\":\"bad\",\"text\":\"bad\",\"url\":\"javascript:alert(1)\"}]}"])</script></body></html>`

	client := NewAntigravityClient(&http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Header: make(http.Header), Request: req, Body: io.NopCloser(strings.NewReader(html))}, nil
	})})

	page, err := client.Fetch(context.Background())
	if err != nil || len(page.Records) != 2 {
		t.Fatalf("unexpected antigravity page: %+v err=%v", page, err)
	}
	if page.Records[0].ID != "cmwq77dh000ic1ypltycs72pu" || page.Records[0].Type != "regular" || page.Records[0].Source.Author != "_mohansolo" || page.Records[0].ScopeZH != "全体用户 · Gemini 模型" || !strings.Contains(page.Records[0].TextZH, `"Antigravity"`) {
		t.Fatalf("unexpected first antigravity record: %+v", page.Records[0])
	}
	if page.Records[1].Type != "banked" || page.Records[1].TextZH != "预告缓存优化后将发放重置补偿" {
		t.Fatalf("unexpected preview antigravity record: %+v", page.Records[1])
	}
}
