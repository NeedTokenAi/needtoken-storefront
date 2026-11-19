package statuspage

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	diagnosticscontract "github.com/dujiao-next/internal/modules/diagnostics/contract"
)

func TestOpenAISeriesBuildsChatGPTAndCodexGroupsBeyond50Incidents(t *testing.T) {
	now := time.Date(2026, 11, 19, 12, 0, 0, 0, time.UTC)
	summary := diagnosticscontract.ServiceStatusSummary{
		Components: []diagnosticscontract.ServiceStatusComponent{
			{ID: "chatgpt-group", Name: "ChatGPT", Group: true, Status: "unknown", Components: []string{"cg-1", "cg-2"}},
			{ID: "cg-1", Name: "Conversations", GroupID: "chatgpt-group", Status: "operational"},
			{ID: "cg-2", Name: "Login", GroupID: "chatgpt-group", Status: "operational"},
			{ID: "codex-group", Name: "Codex", Group: true, Status: "unknown", Components: []string{"cx-1"}},
			{ID: "cx-1", Name: "Codex Web", GroupID: "codex-group", Status: "degraded_performance"},
			{ID: "api-group", Name: "APIs", Group: true, Status: "unknown", Components: []string{"api-1"}},
			{ID: "api-1", Name: "Chat Completions", GroupID: "api-group", Status: "major_outage"},
		},
	}

	// Place 55 unrelated API incidents ahead of the ChatGPT/Codex incidents to prove
	// the 90-day series does not silently truncate at 50 records.
	var records []string
	for i := range 55 {
		records = append(records, fmt.Sprintf(`{"id":"api-inc-%d","name":"API event %d","status":"resolved","published_at":"2026-11-18T00:00:00Z","component_impacts":[{"component_id":"api-1","status":"full_outage","start_at":"2026-11-18T00:00:00Z","end_at":"2026-11-18T04:00:00Z"}]}`, i, i))
	}
	// ChatGPT: overlapping full_outage (12h) + partial_outage (24h, 12h non-overlapping -> 6h weighted) = 18h / 90d = 99.17%
	records = append(records,
		`{"id":"cg-full","name":"ChatGPT major outage","status":"resolved","published_at":"2026-11-10T00:00:00Z","component_impacts":[{"component_id":"cg-1","status":"full_outage","start_at":"2026-11-10T00:00:00Z","end_at":"2026-11-10T12:00:00Z"},{"component_id":"cg-2","status":"partial_outage","start_at":"2026-11-10T00:00:00Z","end_at":"2026-11-11T00:00:00Z"}]}`,
		`{"id":"cx-deg","name":"Codex latency spike","status":"resolved","published_at":"2026-11-12T05:00:00Z","component_impacts":[{"component_id":"cx-1","status":"degraded_performance","start_at":"2026-11-12T05:00:00Z","end_at":"2026-11-12T06:00:00Z"}]}`,
	)
	body := `{"incidents":[` + strings.Join(records, ",") + `]}`

	series, err := buildOpenAISeries(summary, []byte(body), now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(series) != 2 {
		t.Fatalf("expected 2 OpenAI series (ChatGPT + Codex), got %d", len(series))
	}

	chatgpt := series[0]
	if chatgpt.Key != "chatgpt" || chatgpt.Label != "ChatGPT" || chatgpt.ComponentCount != 2 || chatgpt.Status != "operational" || len(chatgpt.Days) != 90 {
		t.Fatalf("unexpected ChatGPT series metadata: %#v", chatgpt)
	}
	if chatgpt.UptimePercent == nil || *chatgpt.UptimePercent != 99.17 {
		t.Fatalf("expected ChatGPT uptime 99.17%%, got %v", chatgpt.UptimePercent)
	}
	if chatgpt.Days[0].Date != "2026-08-22" || chatgpt.Days[89].Date != "2026-11-19" {
		t.Fatalf("unexpected 90-day bounds: %s .. %s", chatgpt.Days[0].Date, chatgpt.Days[89].Date)
	}
	dayNov10 := findSeriesDay(t, chatgpt.Days, "2026-11-10")
	if dayNov10.Status != "major_outage" || len(dayNov10.Events) != 1 || dayNov10.Events[0].URL != "https://status.openai.com/incidents/cg-full" {
		t.Fatalf("unexpected Nov 10 ChatGPT day: %#v", dayNov10)
	}
	dayNov18 := findSeriesDay(t, chatgpt.Days, "2026-11-18")
	if dayNov18.Status != "operational" || len(dayNov18.Events) != 0 {
		t.Fatalf("unrelated API outage leaked into ChatGPT series: %#v", dayNov18)
	}

	codex := series[1]
	if codex.Key != "codex" || codex.Label != "Codex" || codex.ComponentCount != 1 || codex.Status != "degraded_performance" {
		t.Fatalf("unexpected Codex series metadata: %#v", codex)
	}
	if codex.UptimePercent == nil || *codex.UptimePercent != 100 {
		t.Fatalf("degraded-only Codex should remain 100%% uptime, got %v", codex.UptimePercent)
	}
	dayNov12 := findSeriesDay(t, codex.Days, "2026-11-12")
	if dayNov12.Status != "degraded_performance" || len(dayNov12.Events) != 1 {
		t.Fatalf("unexpected Nov 12 Codex day: %#v", dayNov12)
	}
	if codex.Days[89].Status != "degraded_performance" {
		t.Fatalf("current Codex degradation should reflect on today bar: %#v", codex.Days[89])
	}
}

func TestClaudeSeriesParsesOfficialThreeGroupsFromUptimeShowcase(t *testing.T) {
	summary := diagnosticscontract.ServiceStatusSummary{
		Components: []diagnosticscontract.ServiceStatusComponent{
			{ID: "rwppv331jlwc", Name: "claude.ai", Status: "operational"},
			{ID: "yyzkbfz2thpt", Name: "Claude Code", Status: "degraded_performance"},
			{ID: "k8w3r06qmzrp", Name: "Claude API (api.anthropic.com)", Status: "operational"},
		},
	}
	body := `{
		"values": [
			{"component": "rwppv331jlwc", "ninety": 99.56},
			{"component": "yyzkbfz2thpt", "ninety": 99.7},
			{"component": "k8w3r06qmzrp", "ninety": 99.76}
		],
		"timelines": {
			"rwppv331jlwc": {"days": [
				{"date": "2026-11-17", "outages": {}, "related_events": []},
				{"date": "2026-11-18", "outages": {"p": 1200}, "related_events": [{"name": "Elevated errors on claude.ai", "code": "abc123"}]},
				{"date": "2026-11-19", "outages": {"m": 600}, "related_events": [{"name": "Major outage", "code": "../unsafe"}]}
			]},
			"yyzkbfz2thpt": {"days": [
				{"date": "2026-11-19", "outages": {}, "related_events": []}
			]},
			"k8w3r06qmzrp": {"days": [
				{"date": "2026-11-19", "outages": {}, "related_events": []}
			]}
		}
	}`

	var requested string
	client := NewClient(&http.Client{Transport: transportFunc(func(request *http.Request) (*http.Response, error) {
		requested = request.URL.String()
		return upstreamResponse(request, 200, body), nil
	})})

	series, err := client.FetchSeries(context.Background(), diagnosticscontract.ServiceStatusClaude, summary)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if requested != claudeUptimeShowcaseURL {
		t.Fatalf("unexpected Claude uptime endpoint: %s", requested)
	}
	if len(series) != 3 {
		t.Fatalf("expected 3 Claude series, got %d", len(series))
	}
	if series[0].Key != "claude" || series[0].Label != "Claude" || *series[0].UptimePercent != 99.56 {
		t.Fatalf("unexpected claude.ai series: %#v", series[0])
	}
	if series[0].Days[0].Status != "operational" || series[0].Days[1].Status != "degraded_performance" || series[0].Days[2].Status != "major_outage" {
		t.Fatalf("unexpected Claude daily statuses: %#v", series[0].Days)
	}
	if series[0].Days[1].Events[0].URL != "https://status.claude.com/incidents/abc123" {
		t.Fatalf("expected official Claude incident URL, got %#v", series[0].Days[1].Events)
	}
	if series[0].Days[2].Events[0].URL != "" || series[0].Days[2].Events[0].ID != "" {
		t.Fatalf("unsafe Claude incident code should not produce a link: %#v", series[0].Days[2].Events)
	}
	if series[1].Key != "claude_code" || series[1].Status != "degraded_performance" || *series[1].UptimePercent != 99.7 {
		t.Fatalf("unexpected Claude Code series: %#v", series[1])
	}
	if series[2].Key != "claude_api" || series[2].Status != "operational" || *series[2].UptimePercent != 99.76 {
		t.Fatalf("unexpected Claude API series: %#v", series[2])
	}
}

func TestFetchSeriesRejectsInvalidProviderAndMalformedPayloads(t *testing.T) {
	client := NewClient(&http.Client{Transport: transportFunc(func(request *http.Request) (*http.Response, error) {
		return upstreamResponse(request, 200, `{"timelines":{"rwppv331jlwc":{"days":[{"date":"not-a-date"}]}}}`), nil
	})})
	if _, err := client.FetchSeries(context.Background(), "other", diagnosticscontract.ServiceStatusSummary{}); !errors.Is(err, diagnosticscontract.ErrServiceStatusProvider) {
		t.Fatalf("expected invalid provider error, got %v", err)
	}
	if _, err := client.FetchSeries(context.Background(), diagnosticscontract.ServiceStatusClaude, diagnosticscontract.ServiceStatusSummary{}); !errors.Is(err, diagnosticscontract.ErrServiceStatusUnavailable) {
		t.Fatalf("expected unavailable error for invalid date, got %v", err)
	}
}

func findSeriesDay(t *testing.T, days []diagnosticscontract.ServiceStatusSeriesDay, date string) diagnosticscontract.ServiceStatusSeriesDay {
	t.Helper()
	for _, d := range days {
		if d.Date == date {
			return d
		}
	}
	t.Fatalf("date %s not found in series", date)
	return diagnosticscontract.ServiceStatusSeriesDay{}
}
