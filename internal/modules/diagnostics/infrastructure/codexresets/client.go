package codexresets

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	diagnosticscontract "github.com/dujiao-next/internal/modules/diagnostics/contract"
)

const (
	feedURL         = "https://codex-resets.com/api/v1/resets?limit=100"
	requestTimeout  = 8 * time.Second
	maxResponseSize = 2 * 1024 * 1024
)

var (
	recordIDPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,128}$`)
	postPathPattern = regexp.MustCompile(`^/[A-Za-z0-9_]{1,15}/status/[0-9]{1,32}/?$`)
)

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

func (c *Client) Fetch(ctx context.Context) (diagnosticscontract.ResetFeedPage, error) {
	ctx, cancel := context.WithTimeout(ctx, requestTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, feedURL, nil)
	if err != nil {
		return diagnosticscontract.ResetFeedPage{}, diagnosticscontract.ErrResetFeedUnavailable
	}
	req.Header.Set("Accept", "application/json")
	resp, err := c.client.Do(req)
	if err != nil {
		return diagnosticscontract.ResetFeedPage{}, diagnosticscontract.ErrResetFeedUnavailable
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK || resp.ContentLength > maxResponseSize {
		return diagnosticscontract.ResetFeedPage{}, diagnosticscontract.ErrResetFeedUnavailable
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseSize+1))
	if err != nil || len(body) > maxResponseSize {
		return diagnosticscontract.ResetFeedPage{}, diagnosticscontract.ErrResetFeedUnavailable
	}
	return parsePage(body)
}

func parsePage(body []byte) (diagnosticscontract.ResetFeedPage, error) {
	var payload struct {
		Data       json.RawMessage `json:"data"`
		Pagination struct {
			HasMore *bool `json:"has_more"`
		} `json:"pagination"`
	}
	if json.Unmarshal(body, &payload) != nil || payload.Pagination.HasMore == nil || !strings.HasPrefix(strings.TrimSpace(string(payload.Data)), "[") {
		return diagnosticscontract.ResetFeedPage{}, diagnosticscontract.ErrResetFeedUnavailable
	}
	var rows []json.RawMessage
	if json.Unmarshal(payload.Data, &rows) != nil || len(rows) > 100 {
		return diagnosticscontract.ResetFeedPage{}, diagnosticscontract.ErrResetFeedUnavailable
	}
	page := diagnosticscontract.ResetFeedPage{HasMore: *payload.Pagination.HasMore, Records: make([]diagnosticscontract.ResetRecord, 0, len(rows))}
	seen := make(map[string]bool, len(rows))
	for _, row := range rows {
		var raw struct {
			ID          string                          `json:"id"`
			Type        string                          `json:"reset_type"`
			AnnouncedAt string                          `json:"announced_at"`
			Text        string                          `json:"text"`
			Source      diagnosticscontract.ResetSource `json:"source"`
		}
		if json.Unmarshal(row, &raw) != nil || !recordIDPattern.MatchString(raw.ID) || seen[raw.ID] || (raw.Type != "regular" && raw.Type != "banked") || (raw.Source.Type != "x_post" && raw.Source.Type != "observed") {
			continue
		}
		announced, err := time.Parse(time.RFC3339Nano, raw.AnnouncedAt)
		text := strings.TrimSpace(raw.Text)
		if err != nil || announced.IsZero() || text == "" || len(text) > 20000 {
			continue
		}
		raw.Source.URL = safePostURL(raw.Source.URL)
		if raw.Source.Type == "x_post" && raw.Source.URL == "" {
			continue
		}
		raw.Source.Author = strings.TrimSpace(raw.Source.Author)
		if len(raw.Source.Author) > 128 {
			raw.Source.Author = ""
		}
		seen[raw.ID] = true
		page.Records = append(page.Records, diagnosticscontract.ResetRecord{ID: raw.ID, Type: raw.Type, AnnouncedAt: announced.UTC(), Text: text, Source: raw.Source})
	}
	// A valid empty page is meaningful; a nonempty malformed page is not a refresh.
	if len(rows) > 0 && len(page.Records) == 0 {
		return diagnosticscontract.ResetFeedPage{}, diagnosticscontract.ErrResetFeedUnavailable
	}
	return page, nil
}

func safePostURL(raw string) string {
	if len(raw) > 2048 {
		return ""
	}
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Scheme != "https" || u.User != nil || u.Port() != "" || !postPathPattern.MatchString(u.EscapedPath()) {
		return ""
	}
	switch strings.ToLower(u.Hostname()) {
	case "x.com", "twitter.com":
	default:
		return ""
	}
	u.Host = strings.ToLower(u.Hostname())
	u.RawQuery, u.Fragment = "", ""
	u.ForceQuery = false
	u.Path = strings.TrimSuffix(u.Path, "/")
	return u.String()
}

var _ diagnosticscontract.ResetFeedSource = (*Client)(nil)
