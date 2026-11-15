package statuspage

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	diagnosticscontract "github.com/dujiao-next/internal/modules/diagnostics/contract"
)

const openAIIncidentFixture = `{"id":"01M7ATRS6BMZ61E4JDHJGM79Y0","name":"Delayed Costs Data in the Compliance API","status":"monitoring","created_at":"2026-11-13T16:10:07Z","updated_at":"2026-11-14T05:31:20Z","impact":"minor","incident_updates":[{"id":"01M7C8KV75AD3DJ6B0CA5EACSH","body":"We are monitoring the affected services to confirm recovery.\nDelays in cost data are still expected.","created_at":"2026-11-14T05:31:20Z","display_at":"2026-11-14T05:31:20Z","status":"monitoring","updated_at":"2026-11-14T05:31:20Z"}]}`
const claudeIncidentFixture = `{"id":"ct4c2pwf2xpj","name":"Some users unable to sign up with Apple on the Claude iOS app","status":"investigating","impact":"major","created_at":"2026-11-14T07:11:36.117Z","updated_at":"2026-11-14T07:11:36.220Z","started_at":"2026-11-14T07:11:36.111Z","resolved_at":null,"shortlink":"https://untrusted.invalid/private","components":[{"id":"rwppv331jlwc","name":"claude.ai","status":"degraded_performance"}],"incident_updates":[{"id":"ywtdk17vlyf7","status":"investigating","body":"We are investigating an issue with Sign in with Apple on the Claude iOS app.","display_at":"2026-11-14T07:11:36.218Z","created_at":"2026-11-14T07:11:36.218Z","updated_at":"2026-11-14T07:11:36.218Z","affected_components":[{"code":"rwppv331jlwc","name":"claude.ai","old_status":"operational","new_status":"partial_outage"}]}]}`

func TestSummaryRetainsOfficialIncidentWithoutInventingOpenAIComponentScope(t *testing.T) {
	body := strings.TrimSuffix(summaryFixture, "}") + `,"incidents":[` + openAIIncidentFixture + `]}`
	summary, err := parseSummary([]byte(body), "https://status.openai.com/")
	if err != nil || len(summary.Incidents) != 1 {
		t.Fatalf("summary incident was lost: %v", err)
	}
	incident := summary.Incidents[0]
	if incident.Status != "monitoring" || incident.Impact != "minor" || incident.URL != "https://status.openai.com/incidents/01M7ATRS6BMZ61E4JDHJGM79Y0" || incident.CreatedAt == nil || incident.UpdatedAt == nil || incident.StartedAt != nil || incident.ResolvedAt != nil {
		t.Fatalf("official incident stage or timestamps changed: %#v", incident)
	}
	if len(incident.Components) != 0 || len(incident.IncidentUpdates[0].AffectedComponents) != 0 || incident.IncidentUpdates[0].DisplayAt == nil || incident.IncidentUpdates[0].DisplayAt.Hour() != 5 {
		t.Fatal("OpenAI incident acquired an inferred product scope or fabricated time")
	}
	encoded, _ := json.Marshal(summary)
	if !strings.Contains(string(encoded), `"components":[]`) || strings.Contains(string(encoded), "uptime") {
		t.Fatal("missing associations were not explicit or incident data became uptime")
	}
}

func TestClaudeIncidentPreservesExactComponentAssociationsAndUpdateText(t *testing.T) {
	rows, err := parseIncidents([]byte("["+claudeIncidentFixture+"]"), "https://status.claude.com/")
	if err != nil || len(rows) != 1 {
		t.Fatalf("Claude incident rejected: %v", err)
	}
	incident := rows[0]
	update := incident.IncidentUpdates[0]
	if incident.Components[0].ID != "rwppv331jlwc" || incident.Components[0].Name != "claude.ai" || incident.Components[0].Status != "degraded_performance" || update.AffectedComponents[0].Code != "rwppv331jlwc" || update.AffectedComponents[0].NewStatus != "partial_outage" || update.Status != "investigating" || !strings.Contains(update.Body, "Sign in with Apple") {
		t.Fatal("scoped incident associations or latest update were lost")
	}
	if incident.StartedAt == nil || incident.StartedAt.Nanosecond() != 111000000 || incident.ResolvedAt != nil || update.DisplayAt == nil || update.DisplayAt.Nanosecond() != 218000000 || incident.URL != "https://status.claude.com/incidents/ct4c2pwf2xpj" {
		t.Fatal("incident source precision or official URL changed")
	}
	encoded, _ := json.Marshal(incident)
	if strings.Contains(string(encoded), "untrusted.invalid") || strings.Contains(string(encoded), "shortlink") {
		t.Fatal("untrusted upstream link was reflected")
	}
}

func TestHistoryReadsOnlyFixedPublicEndpointsAndCapsRecentRecords(t *testing.T) {
	var records, nativeRecords []string
	for index := 0; index < 55; index++ {
		records = append(records, fmt.Sprintf(`{"id":"incident-%d","name":"Official incident","status":"resolved","impact":"minor","created_at":"2026-11-01T00:00:00Z","resolved_at":"2026-11-01T01:00:00Z"}`, index))
		nativeRecords = append(nativeRecords, fmt.Sprintf(`{"id":"incident-%d","name":"Official incident","status":"resolved","published_at":"2026-11-01T00:00:00Z","updates":[]}`, index))
	}
	var requested []string
	client := NewClient(&http.Client{Transport: transportFunc(func(request *http.Request) (*http.Response, error) {
		requested = append(requested, request.URL.String())
		if request.Method != http.MethodGet || request.Header.Get("Cookie") != "" || request.Header.Get("Authorization") != "" || request.URL.RawQuery != "" || request.Header.Get("Accept") != "application/json" {
			t.Fatal("history read carried credentials, parameters or mutation")
		}
		if deadline, ok := request.Context().Deadline(); !ok || time.Until(deadline) > requestTimeout {
			t.Fatal("history source is not bounded by timeout")
		}
		if request.URL.Host == "status.openai.com" {
			return upstreamResponse(request, 200, `{"incidents":[`+strings.Join(nativeRecords, ",")+`]}`), nil
		}
		return upstreamResponse(request, 200, `{"incidents":[`+strings.Join(records, ",")+`]}`), nil
	})})
	for _, provider := range []diagnosticscontract.ServiceStatusProvider{diagnosticscontract.ServiceStatusOpenAI, diagnosticscontract.ServiceStatusClaude} {
		history, err := client.FetchHistory(context.Background(), provider)
		if err != nil || history.State != "ready" || history.Coverage != "partial" || len(history.Incidents) != 50 || history.Incidents[49].ID != "incident-49" || !strings.HasSuffix(history.SourceURL, "/history") {
			t.Fatalf("recent feed was lost, unbounded or claimed full coverage: %#v %v", history, err)
		}
		encoded, _ := json.Marshal(history)
		if strings.Contains(string(encoded), "percent") || strings.Contains(string(encoded), "90") {
			t.Fatal("recent event history claimed a measured uptime window")
		}
	}
	if len(requested) != 2 || requested[0] != openAINativeURL+"/incidents" || requested[1] != "https://status.claude.com/api/v2/incidents.json" {
		t.Fatalf("history endpoints changed: %v", requested)
	}
	if _, err := client.FetchHistory(context.Background(), "other"); !errors.Is(err, diagnosticscontract.ErrServiceStatusProvider) || len(requested) != 2 {
		t.Fatal("unrecognized provider reached history transport")
	}
}

func TestUnavailableHistoryCannotBecomeAnEmptyHealthyFeed(t *testing.T) {
	for _, variant := range []string{"redirect", "forbidden", "malformed", "missing", "null", "oversized", "html"} {
		t.Run(variant, func(t *testing.T) {
			calls := 0
			client := NewClient(&http.Client{Transport: transportFunc(func(request *http.Request) (*http.Response, error) {
				calls++
				response := upstreamResponse(request, 200, `{"incidents":[]}`)
				switch variant {
				case "redirect":
					response.StatusCode = 302
					response.Header.Set("Location", "https://untrusted.invalid/")
				case "forbidden":
					response.StatusCode = 403
				case "malformed":
					response = upstreamResponse(request, 200, `{"incidents":[`)
				case "missing":
					response = upstreamResponse(request, 200, `{}`)
				case "null":
					response = upstreamResponse(request, 200, `{"incidents":null}`)
				case "oversized":
					response = upstreamResponse(request, 200, strings.Repeat("x", maxResponseSize+1))
				case "html":
					response.Header.Set("Content-Type", "text/html")
				}
				return response, nil
			})})
			history, err := client.FetchHistory(context.Background(), diagnosticscontract.ServiceStatusClaude)
			if !errors.Is(err, diagnosticscontract.ErrServiceStatusUnavailable) || history.State != "unavailable" || history.Coverage != "partial" || len(history.Incidents) != 0 || calls != 1 {
				t.Fatalf("failed history looked complete or healthy: %#v %v", history, err)
			}
		})
	}
	client := NewClient(&http.Client{Transport: transportFunc(func(request *http.Request) (*http.Response, error) {
		return upstreamResponse(request, 200, `{"incidents":[]}`), nil
	})})
	history, err := client.FetchHistory(context.Background(), diagnosticscontract.ServiceStatusClaude)
	if err != nil || history.State != "ready" || history.Coverage != "partial" || history.Incidents == nil {
		t.Fatal("a real empty recent feed was not distinguishable from an unavailable feed")
	}
}

func TestIncidentParserValidatesDatesAndKeepsUnknownStagesUnknown(t *testing.T) {
	for _, body := range []string{
		"null", `{}`, `[{"id":"x"}]`,
		strings.Replace("["+claudeIncidentFixture+"]", "2026-11-14T07:11:36.111Z", "2026-02-30T07:11:36Z", 1),
		strings.Replace("["+claudeIncidentFixture+"]", "2026-11-14T07:11:36.218Z", "not-a-time", 1),
		strings.Replace("["+claudeIncidentFixture+"]", `"ct4c2pwf2xpj"`, `"../unsafe"`, 1),
		strings.Replace("["+claudeIncidentFixture+"]", `"rwppv331jlwc"`, `"invalid/component"`, 1),
		strings.Replace("["+claudeIncidentFixture+"]", `"code":"rwppv331jlwc"`, `"code":"invalid/component"`, 1),
		strings.Replace("["+claudeIncidentFixture+"]", `"body":"`, `"body":"\u0000`, 1),
		"[" + claudeIncidentFixture + "," + claudeIncidentFixture + "]",
	} {
		if _, err := parseIncidents([]byte(body), "https://status.claude.com/"); !errors.Is(err, diagnosticscontract.ErrServiceStatusUnavailable) {
			t.Fatal("malformed source incident was accepted")
		}
	}
	body := strings.ReplaceAll(claudeIncidentFixture, `"investigating"`, `"future-stage"`)
	body = strings.ReplaceAll(body, `"degraded_performance"`, `"future-component-state"`)
	body = strings.Replace(body, `"major"`, `"future-impact"`, 1)
	rows, err := parseIncidents([]byte("["+body+"]"), "https://status.claude.com/")
	if err != nil || rows[0].Status != "unknown" || rows[0].Impact != "unknown" || rows[0].Components[0].Status != "unknown" || rows[0].IncidentUpdates[0].Status != "unknown" {
		t.Fatal("unknown incident data was presented as a known healthy stage")
	}
	var parsed *time.Time
	if err := parseIncidentTime("2026-11-14T15:11:36.218+08:00", &parsed); err != nil || parsed == nil || parsed.Hour() != 7 || parsed.Location() != time.UTC {
		t.Fatal("official timestamp timezone was relabeled rather than converted")
	}
}
