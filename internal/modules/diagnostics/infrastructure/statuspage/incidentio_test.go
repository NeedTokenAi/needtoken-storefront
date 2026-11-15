package statuspage

import (
	"encoding/json"
	"strings"
	"testing"
)

// Matches the native schema used by status.openai.com on 2026-11-15. The
// compatible /api/v2 feed drops these product and component associations.
const nativeIncidentFixture = `{"id":"cost-delay","name":"Delayed Costs Data in the Compliance API","status":"monitoring","published_at":"2026-11-13T16:10:07.366Z","updates":[{"id":"first-update","published_at":"2026-11-13T16:10:07.366Z","to_status":"identified","message_string":"API and ChatGPT usage are not affected.","component_statuses":[{"component_id":"compliance","status":"degraded_performance"}]},{"id":"latest-update","published_at":"2026-11-14T05:31:20.182Z","to_status":"monitoring","message":{"markdown":"Monitoring recovery."},"component_statuses":[{"component_id":"compliance","status":"degraded_performance"}]}],"component_impacts":[{"component_id":"compliance","status":"degraded_performance","start_at":"2026-11-13T16:10:07.366Z"}],"affected_components":[{"component_id":"compliance","status":"degraded_performance"}]}`

func TestNativeOpenAIPreservesRealGroupsAndIncidentComponentIDs(t *testing.T) {
	body := strings.Replace(nativeSummaryFixture, `"ongoing_incidents":[]`, `"ongoing_incidents":[`+nativeIncidentFixture+`]`, 1)
	body = strings.ReplaceAll(body, "gpt-voice", "compliance")
	body = strings.ReplaceAll(body, `"name":"Voice"`, `"name":"Compliance API"`)
	// The same display name Login belongs to two different products.
	summary, err := parseIncidentIOSummary([]byte(body))
	if err != nil {
		t.Fatal(err)
	}
	if summary.Components[0].GroupID != "chatgpt-group" || summary.Components[1].GroupID != "api-group" || summary.Components[1].Status != "operational" {
		t.Fatal("native product boundaries or current statuses were lost")
	}
	incident := summary.Incidents[0]
	if incident.Components[0].ID != "compliance" || incident.Components[0].Name != "Compliance API" || incident.IncidentUpdates[0].AffectedComponents[0].Code != "compliance" || incident.IncidentUpdates[0].Body != "API and ChatGPT usage are not affected." || incident.IncidentUpdates[1].Body != "Monitoring recovery." {
		t.Fatalf("native component linkage or update context lost: %#v", incident)
	}
	if incident.StartedAt == nil || incident.StartedAt.Nanosecond() != 366000000 || incident.UpdatedAt == nil || incident.UpdatedAt.Nanosecond() != 182000000 || incident.ResolvedAt != nil || incident.URL != "https://status.openai.com/incidents/cost-delay" {
		t.Fatal("native incident dates or official link were invented or lost")
	}
}

func TestNativeHistoryUsesExplicitAffectedIDsAndResolutionUpdate(t *testing.T) {
	body := strings.Replace(nativeIncidentFixture, `"status":"monitoring"`, `"status":"resolved"`, 1)
	body = strings.Replace(body, `"to_status":"monitoring"`, `"to_status":"resolved"`, 1)
	rows, err := parseIncidentIOIncidents([]byte("["+body+"]"), nil)
	if err != nil || len(rows) != 1 || rows[0].ResolvedAt == nil || rows[0].ResolvedAt.Hour() != 5 || rows[0].Components[0].ID != "compliance" || rows[0].Impact != "unknown" {
		t.Fatalf("history lost native IDs/timestamps or inferred impact: %#v %v", rows, err)
	}
	encoded, _ := json.Marshal(rows)
	if strings.Contains(string(encoded), "uptime") || strings.Contains(string(encoded), "percentage") {
		t.Fatal("native event history was turned into uptime")
	}
}

func TestNativeSummaryCannotTreatMissingStateOrGroupDataAsHealthy(t *testing.T) {
	for _, body := range []string{
		`{}`, `{"summary":null}`,
		strings.Replace(nativeSummaryFixture, `"affected_components":[{"component_id":"gpt-login","status":"degraded_performance"}],`, "", 1),
		strings.Replace(nativeSummaryFixture, `"affected_components":[{"component_id":"gpt-login","status":"degraded_performance"}]`, `"affected_components":null`, 1),
		strings.Replace(nativeSummaryFixture, `"ongoing_incidents":[]`, `"ongoing_incidents":null`, 1),
		strings.Replace(nativeSummaryFixture, `"gpt-login","status":"degraded_performance"`, `"unknown-component","status":"degraded_performance"`, 1),
		strings.Replace(nativeSummaryFixture, `"id":"chatgpt-group"`, `"id":"invalid/group"`, 1),
		strings.Replace(nativeSummaryFixture, `"component_id":"gpt-voice"`, `"component_id":"missing-child"`, 1),
		strings.Replace(nativeSummaryFixture, `"id":"api-login"`, `"id":"gpt-login"`, 1),
	} {
		if _, err := parseIncidentIOSummary([]byte(body)); err == nil {
			t.Fatalf("incomplete native state was accepted: %s", body)
		}
	}
	for _, body := range []string{
		"null", `{}`,
		strings.Replace("["+nativeIncidentFixture+"]", `"component_id":"compliance"`, `"component_id":"invalid/id"`, 1),
		strings.Replace("["+nativeIncidentFixture+"]", "2026-11-13T16:10:07.366Z", "invalid-date", 1),
	} {
		if _, err := parseIncidentIOIncidents([]byte(body), nil); err == nil {
			t.Fatal("invalid native incident was accepted")
		}
	}
}
