package resetsources

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

	diagnosticscontract "github.com/dujiao-next/internal/modules/diagnostics/contract"
)

const (
	AntigravitySourceURL = "https://yidinghuiziyou.com/ai/antigravity-reset/"
)

// AntigravityClient extracts verified public Antigravity / Gemini quota reset
// announcements from the public Next.js SSR page with strict size/URL bounds.
type AntigravityClient struct {
	client *http.Client
	url    string
}

func NewAntigravityClient(client *http.Client) *AntigravityClient {
	return &AntigravityClient{
		client: configuredHTTPClient(client, true),
		url:    AntigravitySourceURL,
	}
}

func (c *AntigravityClient) Fetch(ctx context.Context) (diagnosticscontract.ResetFeedPage, error) {
	ctx, cancel := context.WithTimeout(ctx, requestTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.url, nil)
	if err != nil {
		return diagnosticscontract.ResetFeedPage{}, diagnosticscontract.ErrResetFeedUnavailable
	}
	req.Header.Set("Accept", "text/html,application/xhtml+xml")
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
	return parseAntigravityHTML(body)
}

type antigravityRecord struct {
	ID          string  `json:"id"`
	Kind        string  `json:"kind"`
	AnnouncedAt string  `json:"announcedAt"`
	TextZh      *string `json:"textZh"`
	Text        *string `json:"text"`
	Scope       *string `json:"scope"`
	ScopeEn     *string `json:"scopeEn"`
	URL         string  `json:"url"`
}

func parseAntigravityHTML(body []byte) (diagnosticscontract.ResetFeedPage, error) {
	raw := string(body)
	// Next.js RSC embeds serialized JSON inside self.__next_f.push([1,"..."]).
	// Decode the JS/JSON string literals properly so nested quotes (\\\") survive.
	candidates := make([]string, 0, 2)
	if decoded := extractNextFRSCChunks(raw); decoded != "" {
		candidates = append(candidates, decoded)
	}
	candidates = append(candidates, raw)

	records := make([]diagnosticscontract.ResetRecord, 0, 32)
	seen := make(map[string]bool, 32)

	for _, text := range candidates {
		offset := 0
		for {
			marker := strings.Index(text[offset:], `{"id":"`)
			if marker < 0 {
				break
			}
			start := offset + marker
			end := findJSONObjectEnd(text, start)
			if end <= start {
				offset = start + 7
				continue
			}
			offset = end
			chunk := text[start:end]
			if !strings.Contains(chunk, `"announcedAt"`) || !strings.Contains(chunk, `"kind"`) {
				continue
			}
			var item antigravityRecord
			if json.Unmarshal([]byte(chunk), &item) != nil {
				continue
			}
			if !recordIDPattern.MatchString(item.ID) || seen[item.ID] {
				continue
			}
			announced, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(item.AnnouncedAt))
			if err != nil || announced.IsZero() {
				continue
			}
			kind := "regular"
			switch strings.ToLower(strings.TrimSpace(item.Kind)) {
			case "reset":
				kind = "regular"
			case "preview":
				kind = "banked"
			default:
				continue
			}
			postURL, author := safePostURLAndAuthor(item.URL)
			if postURL == "" {
				continue
			}
			enText := ""
			if item.Text != nil {
				enText = strings.TrimSpace(*item.Text)
			}
			zhText := ""
			if item.TextZh != nil {
				zhText = strings.TrimSpace(*item.TextZh)
			}
			if enText == "" && zhText == "" {
				continue
			}
			if enText == "" {
				enText = zhText
			}
			scopeZh := ""
			if item.Scope != nil {
				scopeZh = strings.TrimSpace(*item.Scope)
			}
			scopeEn := ""
			if item.ScopeEn != nil {
				scopeEn = strings.TrimSpace(*item.ScopeEn)
			}
			if len(enText) > 20000 || len(zhText) > 20000 {
				continue
			}
			seen[item.ID] = true
			records = append(records, diagnosticscontract.ResetRecord{
				ID:          item.ID,
				Type:        kind,
				AnnouncedAt: announced.UTC(),
				Text:        enText,
				TextZH:      zhText,
				Scope:       scopeEn,
				ScopeZH:     scopeZh,
				Source: diagnosticscontract.ResetSource{
					Type:   "x_post",
					Author: author,
					URL:    postURL,
				},
			})
		}
		if len(records) > 0 {
			break
		}
	}

	if len(records) == 0 {
		return diagnosticscontract.ResetFeedPage{}, diagnosticscontract.ErrResetFeedUnavailable
	}
	sort.Slice(records, func(i, j int) bool {
		return records[i].AnnouncedAt.After(records[j].AnnouncedAt)
	})
	hasMore := len(records) > 100
	if hasMore {
		records = records[:100]
	}
	return diagnosticscontract.ResetFeedPage{HasMore: hasMore, Records: records}, nil
}

// extractNextFRSCChunks decodes string payloads inside self.__next_f.push([1,"..."]).
func extractNextFRSCChunks(raw string) string {
	const prefix = `self.__next_f.push([1,"`
	var b strings.Builder
	offset := 0
	for {
		idx := strings.Index(raw[offset:], prefix)
		if idx < 0 {
			break
		}
		strStart := offset + idx + len(prefix) - 1 // points at opening '"'
		strEnd := findJSONStringEnd(raw, strStart)
		if strEnd <= strStart {
			offset = strStart + 1
			continue
		}
		offset = strEnd
		var decoded string
		if json.Unmarshal([]byte(raw[strStart:strEnd]), &decoded) == nil {
			b.WriteString(decoded)
		}
	}
	return b.String()
}

func findJSONStringEnd(text string, start int) int {
	escaped := false
	for i := start + 1; i < len(text); i++ {
		ch := text[i]
		if escaped {
			escaped = false
			continue
		}
		if ch == '\\' {
			escaped = true
		} else if ch == '"' {
			return i + 1
		}
	}
	return -1
}

// findJSONObjectEnd scans a balanced JSON object starting at text[start]=='{'.
func findJSONObjectEnd(text string, start int) int {
	depth := 0
	inString := false
	escaped := false
	limit := min(len(text), start+32768)
	for i := start; i < limit; i++ {
		ch := text[i]
		if inString {
			if escaped {
				escaped = false
				continue
			}
			if ch == '\\' {
				escaped = true
			} else if ch == '"' {
				inString = false
			}
			continue
		}
		switch ch {
		case '"':
			inString = true
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return i + 1
			}
		}
	}
	return -1
}

var _ diagnosticscontract.ResetFeedSource = (*AntigravityClient)(nil)
