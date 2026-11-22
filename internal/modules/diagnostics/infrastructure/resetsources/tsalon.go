package resetsources

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	diagnosticscontract "github.com/dujiao-next/internal/modules/diagnostics/contract"
)

const (
	TSalonAPIURL    = "https://www.tsalon.tech/api/reset/data"
	TSalonSourceURL = "https://www.tsalon.tech/whenreset/"
	requestTimeout  = 10 * time.Second
	maxResponseSize = 2 * 1024 * 1024
	sharedWindow    = 60 * time.Second
)

var (
	recordIDPattern = regexp.MustCompile(`^[A-Za-z0-9_.:-]{1,128}$`)
	postPathPattern = regexp.MustCompile(`^/([A-Za-z0-9_]{1,15})/status/[0-9]{1,32}/?$`)
	beijingLoc      = time.FixedZone("CST", 8*3600)
)

// TSalonClient fetches the aggregated public reset feed from tsalon.tech once
// and shares the parsed per-channel pages across claude/grok/codex requests.
type TSalonClient struct {
	client *http.Client
	url    string
	now    func() time.Time

	mu      sync.Mutex
	fetched time.Time
	pages   map[string]diagnosticscontract.ResetFeedPage
	flight  chan struct{}
}

func NewTSalonClient(client *http.Client) *TSalonClient {
	return &TSalonClient{
		client: configuredHTTPClient(client, true),
		url:    TSalonAPIURL,
		now:    time.Now,
	}
}

func configuredHTTPClient(client *http.Client, allowSameHostRedirect bool) *http.Client {
	var configured http.Client
	if client != nil {
		configured = *client
	}
	configured.Jar = nil
	configured.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if !allowSameHostRedirect || len(via) >= 2 {
			return http.ErrUseLastResponse
		}
		prev := via[len(via)-1]
		if !strings.EqualFold(req.URL.Hostname(), prev.URL.Hostname()) || req.URL.Scheme != "https" {
			return http.ErrUseLastResponse
		}
		return nil
	}
	if configured.Timeout <= 0 || configured.Timeout > requestTimeout {
		configured.Timeout = requestTimeout
	}
	return &configured
}

// ChannelSource adapts one channel slice from the shared TSalonClient.
func (c *TSalonClient) ChannelSource(channel string) diagnosticscontract.ResetFeedSource {
	return tsalonChannelSource{client: c, channel: channel}
}

type tsalonChannelSource struct {
	client  *TSalonClient
	channel string
}

func (s tsalonChannelSource) Fetch(ctx context.Context) (diagnosticscontract.ResetFeedPage, error) {
	return s.client.FetchChannel(ctx, s.channel)
}

func (c *TSalonClient) FetchChannel(ctx context.Context, channel string) (diagnosticscontract.ResetFeedPage, error) {
	pages, err := c.fetchAll(ctx)
	if err != nil {
		return diagnosticscontract.ResetFeedPage{}, err
	}
	page, ok := pages[channel]
	if !ok {
		return diagnosticscontract.ResetFeedPage{}, diagnosticscontract.ErrResetFeedUnavailable
	}
	copied := make([]diagnosticscontract.ResetRecord, len(page.Records))
	copy(copied, page.Records)
	return diagnosticscontract.ResetFeedPage{HasMore: page.HasMore, Records: copied}, nil
}

func (c *TSalonClient) fetchAll(ctx context.Context) (map[string]diagnosticscontract.ResetFeedPage, error) {
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		c.mu.Lock()
		if c.pages != nil && c.now().Sub(c.fetched) < sharedWindow {
			pages := c.pages
			c.mu.Unlock()
			return pages, nil
		}
		done := c.flight
		if done == nil {
			done = make(chan struct{})
			c.flight = done
			go c.refresh(done)
		}
		c.mu.Unlock()

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-done:
			c.mu.Lock()
			pages := c.pages
			fresh := c.pages != nil && c.now().Sub(c.fetched) < sharedWindow
			c.mu.Unlock()
			if fresh {
				return pages, nil
			}
			return nil, diagnosticscontract.ErrResetFeedUnavailable
		}
	}
}

func (c *TSalonClient) refresh(done chan struct{}) {
	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	defer cancel()
	pages, err := c.fetchRemote(ctx)
	c.mu.Lock()
	if err == nil {
		c.pages = pages
		c.fetched = c.now()
	}
	c.flight = nil
	c.mu.Unlock()
	close(done)
}

func (c *TSalonClient) fetchRemote(ctx context.Context) (map[string]diagnosticscontract.ResetFeedPage, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.url, nil)
	if err != nil {
		return nil, diagnosticscontract.ErrResetFeedUnavailable
	}
	req.Header.Set("Accept", "application/json")
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, diagnosticscontract.ErrResetFeedUnavailable
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK || resp.ContentLength > maxResponseSize {
		return nil, diagnosticscontract.ErrResetFeedUnavailable
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseSize+1))
	if err != nil || len(body) > maxResponseSize {
		return nil, diagnosticscontract.ErrResetFeedUnavailable
	}
	return parseTSalonPayload(body)
}

type tsalonProviderBlock struct {
	Events []struct {
		ID              string `json:"id"`
		Type            string `json:"type"`
		Category        string `json:"category"`
		TypeLabelZh     string `json:"typeLabelZh"`
		TypeLabelEn     string `json:"typeLabelEn"`
		ScopeZh         string `json:"scopeZh"`
		ScopeEn         string `json:"scopeEn"`
		ReasonZh        string `json:"reasonZh"`
		ReasonEn        string `json:"reasonEn"`
		LandedAtBeijing string `json:"landedAtBeijing"`
		PostURL         string `json:"postUrl"`
	} `json:"events"`
}

func parseTSalonPayload(body []byte) (map[string]diagnosticscontract.ResetFeedPage, error) {
	var payload struct {
		Status string              `json:"status"`
		Codex  tsalonProviderBlock `json:"codex"`
		Claude tsalonProviderBlock `json:"claude"`
		Grok   tsalonProviderBlock `json:"grok"`
	}
	if json.Unmarshal(body, &payload) != nil || (payload.Status != "" && payload.Status != "ready") {
		return nil, diagnosticscontract.ErrResetFeedUnavailable
	}
	out := map[string]diagnosticscontract.ResetFeedPage{
		"codex":  parseTSalonChannel(payload.Codex),
		"claude": parseTSalonChannel(payload.Claude),
		"grok":   parseTSalonChannel(payload.Grok),
	}
	return out, nil
}

func parseTSalonChannel(block tsalonProviderBlock) diagnosticscontract.ResetFeedPage {
	records := make([]diagnosticscontract.ResetRecord, 0, len(block.Events))
	seen := make(map[string]bool, len(block.Events))
	for _, ev := range block.Events {
		id := strings.TrimSpace(ev.ID)
		if !recordIDPattern.MatchString(id) || seen[id] {
			continue
		}
		announced, ok := parseBeijingMinute(ev.LandedAtBeijing)
		if !ok {
			continue
		}
		kind := "regular"
		if strings.EqualFold(ev.Type, "card") || strings.Contains(strings.ToLower(ev.Category), "banked") {
			kind = "banked"
		}
		postURL, author := safePostURLAndAuthor(ev.PostURL)
		sourceType := "observed"
		if postURL != "" {
			sourceType = "x_post"
		}
		scopeEn := cleanLabel(ev.ScopeEn, "Not specified")
		scopeZh := cleanLabel(ev.ScopeZh, "未说明")
		reasonEn := cleanLabel(ev.ReasonEn, "Not specified in this record")
		reasonZh := cleanLabel(ev.ReasonZh, "该记录未说明")

		textEn := reasonEn
		if textEn == "" {
			textEn = strings.TrimSpace(ev.TypeLabelEn)
			if textEn == "" {
				if kind == "banked" {
					textEn = "Reset card announcement"
				} else {
					textEn = "Global quota reset"
				}
			}
		}
		textZh := reasonZh
		if textZh == "" {
			textZh = strings.TrimSpace(ev.TypeLabelZh)
		}

		seen[id] = true
		records = append(records, diagnosticscontract.ResetRecord{
			ID:          id,
			Type:        kind,
			AnnouncedAt: announced.UTC(),
			Text:        textEn,
			TextZH:      textZh,
			Scope:       scopeEn,
			ScopeZH:     scopeZh,
			Source: diagnosticscontract.ResetSource{
				Type:   sourceType,
				Author: author,
				URL:    postURL,
			},
		})
	}
	sort.Slice(records, func(i, j int) bool {
		return records[i].AnnouncedAt.After(records[j].AnnouncedAt)
	})
	hasMore := len(records) > 100
	if hasMore {
		records = records[:100]
	}
	return diagnosticscontract.ResetFeedPage{HasMore: hasMore, Records: records}
}

func cleanLabel(raw string, placeholder string) string {
	v := strings.TrimSpace(raw)
	if v == "" || v == placeholder || v == "--" {
		return ""
	}
	if len(v) > 2000 {
		return v[:2000]
	}
	return v
}

func parseBeijingMinute(raw string) (time.Time, bool) {
	v := strings.TrimSpace(raw)
	for _, layout := range []string{"2006-01-02 15:04", "2006-01-02 15:04:05"} {
		if t, err := time.ParseInLocation(layout, v, beijingLoc); err == nil {
			return t.UTC(), true
		}
	}
	if t, err := time.Parse(time.RFC3339Nano, v); err == nil {
		return t.UTC(), true
	}
	return time.Time{}, false
}

func safePostURLAndAuthor(raw string) (string, string) {
	if len(raw) > 2048 {
		return "", ""
	}
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Scheme != "https" || u.User != nil || u.Port() != "" {
		return "", ""
	}
	host := strings.ToLower(u.Hostname())
	if host != "x.com" && host != "twitter.com" {
		return "", ""
	}
	matches := postPathPattern.FindStringSubmatch(u.EscapedPath())
	if len(matches) != 2 {
		return "", ""
	}
	u.Host = host
	u.RawQuery, u.Fragment = "", ""
	u.ForceQuery = false
	u.Path = strings.TrimSuffix(u.Path, "/")
	return u.String(), matches[1]
}
