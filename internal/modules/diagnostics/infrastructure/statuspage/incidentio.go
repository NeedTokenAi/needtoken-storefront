package statuspage

import (
	"encoding/json"
	"strings"
	"time"

	diagnosticscontract "github.com/dujiao-next/internal/modules/diagnostics/contract"
)

const openAINativeURL = "https://status.openai.com/proxy/status.openai.com"

type nativeComponentState struct {
	ComponentID string `json:"component_id"`
	Status      string `json:"status"`
}

type nativeIncident struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Status      string `json:"status"`
	PublishedAt string `json:"published_at"`
	Updates     []struct {
		ID                string                 `json:"id"`
		PublishedAt       string                 `json:"published_at"`
		Status            string                 `json:"to_status"`
		MessageString     string                 `json:"message_string"`
		ComponentStatuses []nativeComponentState `json:"component_statuses"`
		Message           struct {
			Markdown string `json:"markdown"`
		} `json:"message"`
	} `json:"updates"`
	AffectedComponents []nativeComponentState `json:"affected_components"`
	ComponentImpacts   []struct {
		ComponentID string `json:"component_id"`
		Status      string `json:"status"`
		StartAt     string `json:"start_at"`
		EndAt       string `json:"end_at"`
	} `json:"component_impacts"`
}

// This is OpenAI's public page data, not the authenticated incident.io API.
// Its explicit affected-components list defines current state, while structure
// retains the product groups needed to distinguish two different Login services.
func parseIncidentIOSummary(body []byte) (diagnosticscontract.ServiceStatusSummary, error) {
	var payload struct {
		Summary *struct {
			Name               string                                       `json:"name"`
			Components         []diagnosticscontract.ServiceStatusComponent `json:"components"`
			AffectedComponents *[]nativeComponentState                      `json:"affected_components"`
			Incidents          json.RawMessage                              `json:"ongoing_incidents"`
			Structure          struct {
				Items []struct {
					Group *struct {
						ID         string `json:"id"`
						Name       string `json:"name"`
						Components []struct {
							ID string `json:"component_id"`
						} `json:"components"`
					} `json:"group"`
				} `json:"items"`
			} `json:"structure"`
		} `json:"summary"`
	}
	if json.Unmarshal(body, &payload) != nil || payload.Summary == nil {
		return diagnosticscontract.ServiceStatusSummary{}, diagnosticscontract.ErrServiceStatusUnavailable
	}
	data := payload.Summary
	if !validIncidentText(data.Name, 200) || len(data.Components) == 0 || data.AffectedComponents == nil || len(*data.AffectedComponents) > maxComponents || len(data.Structure.Items) == 0 || len(data.Structure.Items) > maxComponents {
		return diagnosticscontract.ServiceStatusSummary{}, diagnosticscontract.ErrServiceStatusUnavailable
	}
	components := data.Components
	positions, names := make(map[string]int, len(components)), make(map[string]string, len(components))
	for index := range components {
		components[index].Status = "operational"
		positions[components[index].ID], names[components[index].ID] = index, components[index].Name
	}
	for _, affected := range *data.AffectedComponents {
		index, found := positions[affected.ComponentID]
		if !found {
			return diagnosticscontract.ServiceStatusSummary{}, diagnosticscontract.ErrServiceStatusUnavailable
		}
		components[index].Status = nativeComponentStatus(affected.Status)
	}
	for _, item := range data.Structure.Items {
		if item.Group == nil || len(item.Group.Components) > maxComponents {
			return diagnosticscontract.ServiceStatusSummary{}, diagnosticscontract.ErrServiceStatusUnavailable
		}
		group := diagnosticscontract.ServiceStatusComponent{ID: item.Group.ID, Name: item.Group.Name, Group: true, Status: "unknown", Components: []string{}}
		for _, child := range item.Group.Components {
			index, found := positions[child.ID]
			if !found || components[index].GroupID != "" {
				return diagnosticscontract.ServiceStatusSummary{}, diagnosticscontract.ErrServiceStatusUnavailable
			}
			components[index].GroupID = group.ID
			group.Components = append(group.Components, child.ID)
		}
		components = append(components, group)
	}
	encoded, _ := json.Marshal(components)
	components, err := parseComponents(encoded)
	if err != nil {
		return diagnosticscontract.ServiceStatusSummary{}, err
	}
	incidents, err := parseIncidentIOIncidents(data.Incidents, names)
	if err != nil {
		return diagnosticscontract.ServiceStatusSummary{}, err
	}
	return diagnosticscontract.ServiceStatusSummary{
		Page:       diagnosticscontract.ServiceStatusPage{Name: strings.TrimSpace(data.Name), URL: "https://status.openai.com/"},
		Status:     diagnosticscontract.ServiceStatusState{Indicator: "unknown", Description: "Official component status"},
		Components: components, Incidents: incidents,
	}, nil
}

func parseIncidentIOIncidents(body json.RawMessage, names map[string]string) ([]diagnosticscontract.ServiceStatusIncident, error) {
	if !strings.HasPrefix(strings.TrimSpace(string(body)), "[") {
		return nil, diagnosticscontract.ErrServiceStatusUnavailable
	}
	var rows []nativeIncident
	if json.Unmarshal(body, &rows) != nil || len(rows) > maxComponents {
		return nil, diagnosticscontract.ErrServiceStatusUnavailable
	}
	if len(rows) > maxIncidents {
		rows = rows[:maxIncidents]
	}
	incidents := make([]incidentPayload, 0, len(rows))
	for _, row := range rows {
		if len(row.Updates) > maxComponents || len(row.ComponentImpacts) > maxComponents || len(row.AffectedComponents) > maxComponents {
			return nil, diagnosticscontract.ErrServiceStatusUnavailable
		}
		incident := incidentPayload{ID: row.ID, Name: row.Name, Status: nativeIncidentStatus(row.Status), Impact: "unknown", CreatedAt: row.PublishedAt, Components: []incidentPart{}, IncidentUpdates: []incidentUpdate{}}
		linked := make(map[string]bool)
		appendComponent := func(id, status string) {
			if !linked[id] {
				incident.Components = append(incident.Components, incidentPart{ID: id, Name: names[id], Status: nativeComponentStatus(status)})
				linked[id] = true
			}
		}
		var started, updated, resolved *time.Time
		for _, part := range row.ComponentImpacts {
			appendComponent(part.ComponentID, part.Status)
			var start, end *time.Time
			if parseIncidentTime(part.StartAt, &start) != nil || parseIncidentTime(part.EndAt, &end) != nil {
				return nil, diagnosticscontract.ErrServiceStatusUnavailable
			}
			if start != nil && (started == nil || start.Before(*started)) {
				started = start
			}
		}
		for _, part := range row.AffectedComponents {
			appendComponent(part.ComponentID, part.Status)
		}
		for _, update := range row.Updates {
			body := update.MessageString
			if strings.TrimSpace(body) == "" {
				body = update.Message.Markdown
			}
			parsed := incidentUpdate{ID: update.ID, Status: nativeIncidentStatus(update.Status), Body: body, DisplayAt: update.PublishedAt, AffectedComponents: []incidentPart{}}
			var stamp *time.Time
			if parseIncidentTime(update.PublishedAt, &stamp) != nil || len(update.ComponentStatuses) > maxComponents {
				return nil, diagnosticscontract.ErrServiceStatusUnavailable
			}
			if stamp != nil && (updated == nil || stamp.After(*updated)) {
				updated = stamp
			}
			if stamp != nil && parsed.Status == "resolved" && (resolved == nil || stamp.After(*resolved)) {
				resolved = stamp
			}
			for _, part := range update.ComponentStatuses {
				appendComponent(part.ComponentID, part.Status)
				parsed.AffectedComponents = append(parsed.AffectedComponents, incidentPart{Code: part.ComponentID, Name: names[part.ComponentID], NewStatus: nativeComponentStatus(part.Status)})
			}
			incident.IncidentUpdates = append(incident.IncidentUpdates, parsed)
		}
		if started != nil {
			incident.StartedAt = started.Format(time.RFC3339Nano)
		}
		if updated != nil {
			incident.UpdatedAt = updated.Format(time.RFC3339Nano)
		}
		if resolved != nil && incident.Status == "resolved" {
			incident.ResolvedAt = resolved.Format(time.RFC3339Nano)
		}
		incidents = append(incidents, incident)
	}
	encoded, _ := json.Marshal(incidents)
	return parseIncidents(encoded, "https://status.openai.com/")
}

func nativeComponentStatus(status string) string {
	if status == "full_outage" {
		return "major_outage"
	}
	return optionalComponentStatus(status)
}

func nativeIncidentStatus(status string) string {
	if status == "fixing" {
		return "identified"
	}
	return incidentStage(status)
}
