package statuspage

import (
	"context"
	"encoding/json"
	"strings"
	"time"
	"unicode"

	diagnosticscontract "github.com/dujiao-next/internal/modules/diagnostics/contract"
)

const (
	maxIncidents       = 50
	maxIncidentUpdates = 100
)

type incidentPayload struct {
	ID              string           `json:"id"`
	Name            string           `json:"name"`
	Status          string           `json:"status"`
	Impact          string           `json:"impact"`
	CreatedAt       string           `json:"created_at"`
	StartedAt       string           `json:"started_at"`
	UpdatedAt       string           `json:"updated_at"`
	ResolvedAt      string           `json:"resolved_at"`
	Components      []incidentPart   `json:"components"`
	IncidentUpdates []incidentUpdate `json:"incident_updates"`
}

type incidentPart struct {
	ID        string `json:"id"`
	Code      string `json:"code"`
	Name      string `json:"name"`
	Status    string `json:"status"`
	OldStatus string `json:"old_status"`
	NewStatus string `json:"new_status"`
}

type incidentUpdate struct {
	ID                 string         `json:"id"`
	Status             string         `json:"status"`
	Body               string         `json:"body"`
	DisplayAt          string         `json:"display_at"`
	CreatedAt          string         `json:"created_at"`
	UpdatedAt          string         `json:"updated_at"`
	AffectedComponents []incidentPart `json:"affected_components"`
}

func (c *Client) FetchHistory(ctx context.Context, provider diagnosticscontract.ServiceStatusProvider) (diagnosticscontract.ServiceStatusHistory, error) {
	if !diagnosticscontract.ValidServiceStatusProvider(provider) {
		return diagnosticscontract.ServiceStatusHistory{}, diagnosticscontract.ErrServiceStatusProvider
	}
	base := "https://status.openai.com/"
	if provider == diagnosticscontract.ServiceStatusClaude {
		base = "https://status.claude.com/"
	}
	history := diagnosticscontract.ServiceStatusHistory{State: "unavailable", Coverage: "partial", SourceURL: base + "history", Incidents: []diagnosticscontract.ServiceStatusIncident{}}
	ctx, cancel := context.WithTimeout(ctx, requestTimeout)
	defer cancel()
	endpoint := base + "api/v2/incidents.json"
	if provider == diagnosticscontract.ServiceStatusOpenAI {
		endpoint = openAINativeURL + "/incidents"
	}
	body, err := c.read(ctx, endpoint)
	if err != nil {
		return history, err
	}
	var payload struct {
		Incidents json.RawMessage `json:"incidents"`
	}
	if json.Unmarshal(body, &payload) != nil || len(payload.Incidents) == 0 {
		return history, diagnosticscontract.ErrServiceStatusUnavailable
	}
	var incidents []diagnosticscontract.ServiceStatusIncident
	if provider == diagnosticscontract.ServiceStatusOpenAI {
		incidents, err = parseIncidentIOIncidents(payload.Incidents, nil)
	} else {
		incidents, err = parseIncidents(payload.Incidents, base)
	}
	if err != nil {
		return history, err
	}
	history.Incidents = incidents
	history.State = "ready"
	return history, nil
}

func parseIncidents(body json.RawMessage, sourceURL string) ([]diagnosticscontract.ServiceStatusIncident, error) {
	incidents := make([]diagnosticscontract.ServiceStatusIncident, 0)
	if len(body) == 0 {
		return incidents, nil
	}
	if !strings.HasPrefix(strings.TrimSpace(string(body)), "[") {
		return nil, diagnosticscontract.ErrServiceStatusUnavailable
	}
	var rows []json.RawMessage
	if json.Unmarshal(body, &rows) != nil || len(rows) > maxComponents {
		return nil, diagnosticscontract.ErrServiceStatusUnavailable
	}
	if len(rows) > maxIncidents {
		rows = rows[:maxIncidents]
	}
	seen := make(map[string]bool, len(rows))
	for _, raw := range rows {
		var row incidentPayload
		if json.Unmarshal(raw, &row) != nil || !componentID.MatchString(row.ID) || seen[row.ID] || !validIncidentText(row.Name, 500) || len(row.Components) > maxComponents || len(row.IncidentUpdates) > maxComponents {
			return nil, diagnosticscontract.ErrServiceStatusUnavailable
		}
		seen[row.ID] = true
		incident := diagnosticscontract.ServiceStatusIncident{
			ID: row.ID, Name: strings.TrimSpace(row.Name), Status: incidentStage(row.Status), Impact: incidentImpact(row.Impact), URL: sourceURL + "incidents/" + row.ID,
			Components: []diagnosticscontract.ServiceStatusIncidentComponent{}, IncidentUpdates: []diagnosticscontract.ServiceStatusIncidentUpdate{},
		}
		for _, field := range []struct {
			value  string
			target **time.Time
		}{
			{row.CreatedAt, &incident.CreatedAt}, {row.StartedAt, &incident.StartedAt}, {row.UpdatedAt, &incident.UpdatedAt}, {row.ResolvedAt, &incident.ResolvedAt},
		} {
			if err := parseIncidentTime(field.value, field.target); err != nil {
				return nil, err
			}
		}
		for _, part := range row.Components {
			if !componentID.MatchString(part.ID) || (part.Name != "" && !validIncidentText(part.Name, 200)) {
				return nil, diagnosticscontract.ErrServiceStatusUnavailable
			}
			incident.Components = append(incident.Components, diagnosticscontract.ServiceStatusIncidentComponent{ID: part.ID, Name: strings.TrimSpace(part.Name), Status: optionalComponentStatus(part.Status)})
		}
		if len(row.IncidentUpdates) > maxIncidentUpdates {
			row.IncidentUpdates = row.IncidentUpdates[:maxIncidentUpdates]
		}
		for _, update := range row.IncidentUpdates {
			if !componentID.MatchString(update.ID) || !validIncidentText(update.Body, 16*1024) || len(update.AffectedComponents) > maxComponents {
				return nil, diagnosticscontract.ErrServiceStatusUnavailable
			}
			parsed := diagnosticscontract.ServiceStatusIncidentUpdate{ID: update.ID, Status: incidentStage(update.Status), Body: strings.TrimSpace(update.Body), AffectedComponents: []diagnosticscontract.ServiceStatusAffectedComponent{}}
			for _, field := range []struct {
				value  string
				target **time.Time
			}{
				{update.DisplayAt, &parsed.DisplayAt}, {update.CreatedAt, &parsed.CreatedAt}, {update.UpdatedAt, &parsed.UpdatedAt},
			} {
				if err := parseIncidentTime(field.value, field.target); err != nil {
					return nil, err
				}
			}
			for _, part := range update.AffectedComponents {
				if !componentID.MatchString(part.Code) || (part.Name != "" && !validIncidentText(part.Name, 200)) {
					return nil, diagnosticscontract.ErrServiceStatusUnavailable
				}
				parsed.AffectedComponents = append(parsed.AffectedComponents, diagnosticscontract.ServiceStatusAffectedComponent{Code: part.Code, Name: strings.TrimSpace(part.Name), OldStatus: optionalComponentStatus(part.OldStatus), NewStatus: optionalComponentStatus(part.NewStatus)})
			}
			incident.IncidentUpdates = append(incident.IncidentUpdates, parsed)
		}
		incidents = append(incidents, incident)
	}
	return incidents, nil
}

func parseIncidentTime(value string, target **time.Time) error {
	if value == "" {
		return nil
	}
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return diagnosticscontract.ErrServiceStatusUnavailable
	}
	parsed = parsed.UTC()
	*target = &parsed
	return nil
}

func validIncidentText(value string, limit int) bool {
	return strings.TrimSpace(value) != "" && len(value) <= limit && !strings.ContainsFunc(value, func(char rune) bool { return unicode.IsControl(char) && char != '\n' && char != '\r' && char != '\t' })
}

func incidentStage(value string) string {
	switch value {
	case "investigating", "identified", "monitoring", "resolved", "postmortem":
		return value
	default:
		return "unknown"
	}
}

func incidentImpact(value string) string {
	switch value {
	case "none", "minor", "major", "critical":
		return value
	default:
		return "unknown"
	}
}

func optionalComponentStatus(value string) string {
	switch value {
	case "", "operational", "degraded_performance", "partial_outage", "major_outage", "under_maintenance":
		return value
	default:
		return "unknown"
	}
}

var _ diagnosticscontract.ServiceStatusHistorySource = (*Client)(nil)
