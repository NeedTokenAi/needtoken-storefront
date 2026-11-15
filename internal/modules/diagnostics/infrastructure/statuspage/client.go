package statuspage

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	diagnosticscontract "github.com/dujiao-next/internal/modules/diagnostics/contract"
)

const (
	requestTimeout  = 8 * time.Second
	maxResponseSize = 2 * 1024 * 1024
	maxComponents   = 2048
)

var componentID = regexp.MustCompile(`^[A-Za-z0-9_-]{1,128}$`)

type Client struct{ client *http.Client }

func NewClient(client *http.Client) *Client {
	var configured http.Client
	if client != nil {
		configured = *client
	}
	configured.Jar = nil
	configured.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	if configured.Timeout <= 0 || configured.Timeout > requestTimeout {
		configured.Timeout = requestTimeout
	}
	return &Client{client: &configured}
}

func (c *Client) Fetch(ctx context.Context, provider diagnosticscontract.ServiceStatusProvider) (diagnosticscontract.ServiceStatusSummary, error) {
	ctx, cancel := context.WithTimeout(ctx, requestTimeout)
	defer cancel()
	switch provider {
	case diagnosticscontract.ServiceStatusOpenAI:
		// OpenAI's Atlassian-compatible API drops product groups and incident
		// component associations. Use the same native source as its public page.
		body, err := c.read(ctx, openAINativeURL)
		if err != nil {
			return diagnosticscontract.ServiceStatusSummary{}, err
		}
		return parseIncidentIOSummary(body)
	case diagnosticscontract.ServiceStatusClaude:
		body, err := c.read(ctx, "https://status.claude.com/api/v2/summary.json")
		if err != nil {
			return diagnosticscontract.ServiceStatusSummary{}, err
		}
		return parseSummary(body, "https://status.claude.com/")
	default:
		return diagnosticscontract.ServiceStatusSummary{}, diagnosticscontract.ErrServiceStatusProvider
	}
}

func (c *Client) read(ctx context.Context, endpoint string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, diagnosticscontract.ErrServiceStatusUnavailable
	}
	req.Header.Set("Accept", "application/json")
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, diagnosticscontract.ErrServiceStatusUnavailable
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK || resp.ContentLength > maxResponseSize || !strings.HasPrefix(strings.ToLower(resp.Header.Get("Content-Type")), "application/json") {
		return nil, diagnosticscontract.ErrServiceStatusUnavailable
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseSize+1))
	if err != nil || len(body) > maxResponseSize {
		return nil, diagnosticscontract.ErrServiceStatusUnavailable
	}
	return body, nil
}

func parseSummary(body []byte, sourceURL string) (diagnosticscontract.ServiceStatusSummary, error) {
	var payload struct {
		Page *struct {
			Name      string `json:"name"`
			UpdatedAt string `json:"updated_at"`
		} `json:"page"`
		Status *struct {
			Indicator   string `json:"indicator"`
			Description string `json:"description"`
		} `json:"status"`
		Components json.RawMessage `json:"components"`
		Incidents  json.RawMessage `json:"incidents"`
	}
	if json.Unmarshal(body, &payload) != nil || payload.Page == nil || payload.Status == nil {
		return diagnosticscontract.ServiceStatusSummary{}, diagnosticscontract.ErrServiceStatusUnavailable
	}
	components, err := parseComponents(payload.Components)
	if err != nil {
		return diagnosticscontract.ServiceStatusSummary{}, err
	}
	incidents, err := parseIncidents(payload.Incidents, sourceURL)
	if err != nil {
		return diagnosticscontract.ServiceStatusSummary{}, err
	}
	name, description := strings.TrimSpace(payload.Page.Name), strings.TrimSpace(payload.Status.Description)
	if name == "" || len(name) > 200 || len(description) > 1000 || (description == "" && len(components) == 0) {
		return diagnosticscontract.ServiceStatusSummary{}, diagnosticscontract.ErrServiceStatusUnavailable
	}
	page := diagnosticscontract.ServiceStatusPage{Name: name, URL: sourceURL}
	if payload.Page.UpdatedAt != "" {
		updated, err := time.Parse(time.RFC3339Nano, payload.Page.UpdatedAt)
		if err != nil {
			return diagnosticscontract.ServiceStatusSummary{}, diagnosticscontract.ErrServiceStatusUnavailable
		}
		updated = updated.UTC()
		page.UpdatedAt = &updated
	}
	indicator := payload.Status.Indicator
	switch indicator {
	case "none", "minor", "major", "critical":
	default:
		indicator = "unknown"
	}
	return diagnosticscontract.ServiceStatusSummary{Page: page, Status: diagnosticscontract.ServiceStatusState{Indicator: indicator, Description: description}, Components: components, Incidents: incidents}, nil
}

func parseComponents(body json.RawMessage) ([]diagnosticscontract.ServiceStatusComponent, error) {
	if !strings.HasPrefix(strings.TrimSpace(string(body)), "[") {
		return nil, diagnosticscontract.ErrServiceStatusUnavailable
	}
	var rows []diagnosticscontract.ServiceStatusComponent
	if json.Unmarshal(body, &rows) != nil || len(rows) > maxComponents {
		return nil, diagnosticscontract.ErrServiceStatusUnavailable
	}
	seen := make(map[string]bool, len(rows))
	for index := range rows {
		row := &rows[index]
		row.Name = strings.TrimSpace(row.Name)
		if !componentID.MatchString(row.ID) || seen[row.ID] || row.Name == "" || len(row.Name) > 200 || (row.GroupID != "" && !componentID.MatchString(row.GroupID)) || len(row.Components) > maxComponents {
			return nil, diagnosticscontract.ErrServiceStatusUnavailable
		}
		seen[row.ID] = true
		for _, id := range row.Components {
			if !componentID.MatchString(id) {
				return nil, diagnosticscontract.ErrServiceStatusUnavailable
			}
		}
		switch row.Status {
		case "operational", "degraded_performance", "partial_outage", "major_outage", "under_maintenance":
		default:
			row.Status = "unknown"
		}
	}
	return rows, nil
}

var _ diagnosticscontract.ServiceStatusSource = (*Client)(nil)
