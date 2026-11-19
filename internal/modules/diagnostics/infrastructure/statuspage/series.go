package statuspage

import (
	"context"
	"encoding/json"
	"math"
	"sort"
	"strings"
	"time"

	diagnosticscontract "github.com/dujiao-next/internal/modules/diagnostics/contract"
)

const (
	seriesWindowDays        = 90
	maxSeriesEventsPerDay   = 5
	claudeUptimeShowcaseURL = "https://status.claude.com/uptime_showcase?components=rwppv331jlwc,yyzkbfz2thpt,k8w3r06qmzrp"
)

var claudeSeriesTargets = []struct {
	key          string
	label        string
	componentID  string
	fallbackName string
}{
	{key: "claude", label: "Claude", componentID: "rwppv331jlwc", fallbackName: "claude.ai"},
	{key: "claude_code", label: "Claude Code", componentID: "yyzkbfz2thpt", fallbackName: "Claude Code"},
	{key: "claude_api", label: "Claude API", componentID: "k8w3r06qmzrp", fallbackName: "Claude API"},
}

var openAISeriesTargets = []struct {
	key   string
	label string
}{
	{key: "chatgpt", label: "ChatGPT"},
	{key: "codex", label: "Codex"},
}

func (c *Client) FetchSeries(ctx context.Context, provider diagnosticscontract.ServiceStatusProvider, summary diagnosticscontract.ServiceStatusSummary) ([]diagnosticscontract.ServiceStatusSeries, error) {
	if !diagnosticscontract.ValidServiceStatusProvider(provider) {
		return nil, diagnosticscontract.ErrServiceStatusProvider
	}
	ctx, cancel := context.WithTimeout(ctx, requestTimeout)
	defer cancel()
	switch provider {
	case diagnosticscontract.ServiceStatusOpenAI:
		body, err := c.read(ctx, openAINativeURL+"/incidents")
		if err != nil {
			return nil, err
		}
		return buildOpenAISeries(summary, body, time.Now().UTC())
	case diagnosticscontract.ServiceStatusClaude:
		body, err := c.read(ctx, claudeUptimeShowcaseURL)
		if err != nil {
			return nil, err
		}
		return parseClaudeSeries(summary, body)
	default:
		return nil, diagnosticscontract.ErrServiceStatusProvider
	}
}

type timeInterval struct {
	start time.Time
	end   time.Time
}

func buildOpenAISeries(summary diagnosticscontract.ServiceStatusSummary, body []byte, now time.Time) ([]diagnosticscontract.ServiceStatusSeries, error) {
	var payload struct {
		Incidents json.RawMessage `json:"incidents"`
	}
	if json.Unmarshal(body, &payload) != nil || !strings.HasPrefix(strings.TrimSpace(string(payload.Incidents)), "[") {
		return nil, diagnosticscontract.ErrServiceStatusUnavailable
	}
	var rows []nativeIncident
	if json.Unmarshal(payload.Incidents, &rows) != nil || len(rows) > maxComponents {
		return nil, diagnosticscontract.ErrServiceStatusUnavailable
	}

	nowUTC := now.UTC()
	today := time.Date(nowUTC.Year(), nowUTC.Month(), nowUTC.Day(), 0, 0, 0, 0, time.UTC)
	windowStart := today.AddDate(0, 0, -(seriesWindowDays - 1))
	windowEnd := nowUTC
	if windowEnd.Before(today) {
		windowEnd = today.Add(24 * time.Hour)
	}
	totalWindowSec := float64(seriesWindowDays * 24 * 60 * 60)

	result := make([]diagnosticscontract.ServiceStatusSeries, 0, len(openAISeriesTargets))
	for _, target := range openAISeriesTargets {
		memberIDs, memberStatuses := openAIGroupMembers(summary.Components, target.label)
		if len(memberIDs) == 0 {
			continue
		}
		days := make([]diagnosticscontract.ServiceStatusSeriesDay, seriesWindowDays)
		dayIndex := make(map[string]int, seriesWindowDays)
		dayEventSeen := make([]map[string]bool, seriesWindowDays)
		for i := range seriesWindowDays {
			dateStr := windowStart.AddDate(0, 0, i).Format("2006-01-02")
			days[i] = diagnosticscontract.ServiceStatusSeriesDay{
				Date:   dateStr,
				Status: "operational",
			}
			dayIndex[dateStr] = i
		}

		var fullIntervals []timeInterval
		var partialIntervals []timeInterval
		for _, inc := range rows {
			if len(inc.ComponentImpacts) > maxComponents {
				return nil, diagnosticscontract.ErrServiceStatusUnavailable
			}
			incStatus := nativeIncidentStatus(inc.Status)
			incActive := incStatus == "investigating" || incStatus == "identified" || incStatus == "monitoring"
			incName := strings.TrimSpace(inc.Name)
			incValidID := componentID.MatchString(inc.ID)
			incURL := ""
			if incValidID {
				incURL = "https://status.openai.com/incidents/" + inc.ID
			}

			for _, impact := range inc.ComponentImpacts {
				if !memberIDs[impact.ComponentID] {
					continue
				}
				mappedStatus := nativeComponentStatus(impact.Status)
				if mappedStatus == "" || mappedStatus == "operational" {
					continue
				}
				var startPtr, endPtr *time.Time
				if parseIncidentTime(impact.StartAt, &startPtr) != nil || parseIncidentTime(impact.EndAt, &endPtr) != nil {
					return nil, diagnosticscontract.ErrServiceStatusUnavailable
				}
				if startPtr == nil {
					continue
				}
				start := *startPtr
				var end time.Time
				hasDuration := false
				if endPtr != nil && !endPtr.Before(start) {
					end = *endPtr
					hasDuration = true
				} else if incActive && !nowUTC.Before(start) {
					end = nowUTC
					hasDuration = true
				} else {
					end = start
				}
				if end.Before(windowStart) || start.After(windowEnd) {
					continue
				}
				clampedStart := start
				if clampedStart.Before(windowStart) {
					clampedStart = windowStart
				}
				clampedEnd := end
				if clampedEnd.After(windowEnd) {
					clampedEnd = windowEnd
				}
				if hasDuration && clampedEnd.After(clampedStart) {
					switch mappedStatus {
					case "major_outage":
						fullIntervals = append(fullIntervals, timeInterval{start: clampedStart, end: clampedEnd})
					case "partial_outage":
						partialIntervals = append(partialIntervals, timeInterval{start: clampedStart, end: clampedEnd})
					}
				}
				curDay := time.Date(clampedStart.Year(), clampedStart.Month(), clampedStart.Day(), 0, 0, 0, 0, time.UTC)
				endDay := time.Date(clampedEnd.Year(), clampedEnd.Month(), clampedEnd.Day(), 0, 0, 0, 0, time.UTC)
				for !curDay.After(endDay) {
					if idx, ok := dayIndex[curDay.Format("2006-01-02")]; ok {
						days[idx].Status = worseComponentStatus(days[idx].Status, mappedStatus)
						if validIncidentText(incName, 500) && len(days[idx].Events) < maxSeriesEventsPerDay {
							if dayEventSeen[idx] == nil {
								dayEventSeen[idx] = make(map[string]bool)
							}
							eventKey := inc.ID
							if eventKey == "" {
								eventKey = incName
							}
							if !dayEventSeen[idx][eventKey] {
								dayEventSeen[idx][eventKey] = true
								event := diagnosticscontract.ServiceStatusSeriesEvent{Name: incName, URL: incURL}
								if incValidID {
									event.ID = inc.ID
								}
								days[idx].Events = append(days[idx].Events, event)
							}
						}
					}
					curDay = curDay.AddDate(0, 0, 1)
				}
			}
		}

		groupStatus := aggregateComponentStatuses(memberStatuses)
		days[seriesWindowDays-1].Status = worseComponentStatus(days[seriesWindowDays-1].Status, groupStatus)

		fullSec := mergedIntervalSeconds(fullIntervals)
		bothIntervals := make([]timeInterval, 0, len(fullIntervals)+len(partialIntervals))
		bothIntervals = append(bothIntervals, fullIntervals...)
		bothIntervals = append(bothIntervals, partialIntervals...)
		outageSec := mergedIntervalSeconds(bothIntervals)
		partialOnlySec := outageSec - fullSec
		if partialOnlySec < 0 {
			partialOnlySec = 0
		}
		downtimeSec := fullSec + 0.5*partialOnlySec
		pct := clampPercent((1.0 - downtimeSec/totalWindowSec) * 100)

		result = append(result, diagnosticscontract.ServiceStatusSeries{
			Key:            target.key,
			Label:          target.label,
			ComponentCount: len(memberIDs),
			Status:         groupStatus,
			UptimePercent:  &pct,
			Days:           days,
		})
	}

	if len(result) == 0 {
		return nil, diagnosticscontract.ErrServiceStatusUnavailable
	}
	return result, nil
}

func openAIGroupMembers(components []diagnosticscontract.ServiceStatusComponent, groupName string) (map[string]bool, []string) {
	groupIDs := make(map[string]bool)
	childIDs := make(map[string]bool)
	for _, c := range components {
		if c.Group && strings.EqualFold(strings.TrimSpace(c.Name), groupName) {
			if c.ID != "" {
				groupIDs[c.ID] = true
			}
			for _, cid := range c.Components {
				if cid != "" {
					childIDs[cid] = true
				}
			}
		}
	}
	if len(groupIDs) == 0 && len(childIDs) == 0 {
		return nil, nil
	}
	members := make(map[string]bool)
	statuses := make([]string, 0)
	for _, c := range components {
		if c.Group || c.ID == "" {
			continue
		}
		if groupIDs[c.GroupID] || childIDs[c.ID] {
			members[c.ID] = true
			statuses = append(statuses, c.Status)
		}
	}
	return members, statuses
}

func parseClaudeSeries(summary diagnosticscontract.ServiceStatusSummary, body []byte) ([]diagnosticscontract.ServiceStatusSeries, error) {
	var payload struct {
		Values []struct {
			Component string   `json:"component"`
			Ninety    *float64 `json:"ninety"`
		} `json:"values"`
		Timelines map[string]struct {
			Days []struct {
				Date    string `json:"date"`
				Outages struct {
					P int `json:"p"`
					M int `json:"m"`
				} `json:"outages"`
				RelatedEvents []struct {
					Name string `json:"name"`
					Code string `json:"code"`
				} `json:"related_events"`
			} `json:"days"`
		} `json:"timelines"`
	}
	if json.Unmarshal(body, &payload) != nil || len(payload.Timelines) == 0 {
		return nil, diagnosticscontract.ErrServiceStatusUnavailable
	}

	uptimeByID := make(map[string]float64, len(payload.Values))
	for _, v := range payload.Values {
		if v.Ninety == nil || math.IsNaN(*v.Ninety) || math.IsInf(*v.Ninety, 0) || *v.Ninety < 0 || *v.Ninety > 100 {
			continue
		}
		uptimeByID[v.Component] = clampPercent(*v.Ninety)
	}

	componentStatusByID := make(map[string]string, len(summary.Components))
	componentStatusByName := make(map[string]string, len(summary.Components))
	for _, c := range summary.Components {
		if c.Group {
			continue
		}
		if c.ID != "" {
			componentStatusByID[c.ID] = c.Status
		}
		componentStatusByName[strings.ToLower(strings.TrimSpace(c.Name))] = c.Status
	}

	result := make([]diagnosticscontract.ServiceStatusSeries, 0, len(claudeSeriesTargets))
	for _, target := range claudeSeriesTargets {
		timeline, ok := payload.Timelines[target.componentID]
		if !ok || len(timeline.Days) == 0 || len(timeline.Days) > maxComponents {
			continue
		}
		rawDays := timeline.Days
		if len(rawDays) > seriesWindowDays {
			rawDays = rawDays[len(rawDays)-seriesWindowDays:]
		}
		days := make([]diagnosticscontract.ServiceStatusSeriesDay, 0, len(rawDays))
		for _, d := range rawDays {
			if _, err := time.Parse("2006-01-02", d.Date); err != nil {
				return nil, diagnosticscontract.ErrServiceStatusUnavailable
			}
			dayStatus := "operational"
			if d.Outages.M > 0 {
				dayStatus = "major_outage"
			} else if d.Outages.P > 0 {
				dayStatus = "degraded_performance"
			}
			var events []diagnosticscontract.ServiceStatusSeriesEvent
			seen := make(map[string]bool)
			for _, ev := range d.RelatedEvents {
				name := strings.TrimSpace(ev.Name)
				if !validIncidentText(name, 500) {
					continue
				}
				key := ev.Code
				if key == "" {
					key = name
				}
				if seen[key] {
					continue
				}
				seen[key] = true
				entry := diagnosticscontract.ServiceStatusSeriesEvent{Name: name}
				if componentID.MatchString(ev.Code) {
					entry.ID = ev.Code
					entry.URL = "https://status.claude.com/incidents/" + ev.Code
				}
				events = append(events, entry)
				if len(events) >= maxSeriesEventsPerDay {
					break
				}
			}
			days = append(days, diagnosticscontract.ServiceStatusSeriesDay{
				Date:   d.Date,
				Status: dayStatus,
				Events: events,
			})
		}
		if len(days) == 0 {
			continue
		}
		status := componentStatusByID[target.componentID]
		if status == "" {
			lowerFallback := strings.ToLower(target.fallbackName)
			for name, s := range componentStatusByName {
				if name == lowerFallback || strings.HasPrefix(name, lowerFallback+" ") {
					status = s
					break
				}
			}
		}
		if status == "" {
			status = days[len(days)-1].Status
		}
		var uptime *float64
		if pct, exists := uptimeByID[target.componentID]; exists {
			value := pct
			uptime = &value
		}
		result = append(result, diagnosticscontract.ServiceStatusSeries{
			Key:            target.key,
			Label:          target.label,
			ComponentCount: 1,
			Status:         status,
			UptimePercent:  uptime,
			Days:           days,
		})
	}
	if len(result) == 0 {
		return nil, diagnosticscontract.ErrServiceStatusUnavailable
	}
	return result, nil
}

func statusSeverityRank(status string) int {
	switch status {
	case "operational":
		return 1
	case "under_maintenance":
		return 2
	case "degraded_performance":
		return 3
	case "partial_outage":
		return 4
	case "major_outage":
		return 5
	default:
		return 0
	}
}

func worseComponentStatus(a, b string) string {
	if statusSeverityRank(b) > statusSeverityRank(a) {
		return b
	}
	if a == "" {
		return b
	}
	return a
}

func aggregateComponentStatuses(statuses []string) string {
	if len(statuses) == 0 {
		return "unknown"
	}
	best := "operational"
	for _, s := range statuses {
		if s == "" || s == "unknown" {
			if statusSeverityRank(best) <= 1 {
				best = "unknown"
			}
			continue
		}
		best = worseComponentStatus(best, s)
	}
	return best
}

func mergedIntervalSeconds(intervals []timeInterval) float64 {
	if len(intervals) == 0 {
		return 0
	}
	sorted := append(make([]timeInterval, 0, len(intervals)), intervals...)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].start.Before(sorted[j].start)
	})
	mergedStart := sorted[0].start
	mergedEnd := sorted[0].end
	var total float64
	for _, iv := range sorted[1:] {
		if !iv.start.After(mergedEnd) {
			if iv.end.After(mergedEnd) {
				mergedEnd = iv.end
			}
			continue
		}
		total += mergedEnd.Sub(mergedStart).Seconds()
		mergedStart = iv.start
		mergedEnd = iv.end
	}
	total += mergedEnd.Sub(mergedStart).Seconds()
	return total
}

func clampPercent(value float64) float64 {
	if value < 0 {
		return 0
	}
	if value > 100 {
		return 100
	}
	return math.Round(value*100) / 100
}

var _ diagnosticscontract.ServiceStatusSeriesSource = (*Client)(nil)
