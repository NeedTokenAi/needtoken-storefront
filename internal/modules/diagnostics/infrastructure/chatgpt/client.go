package chatgpt

import (
	"context"
	"encoding/json"
	"io"
	"math"
	"net/http"
	"sort"
	"strings"
	"time"

	diagnosticscontract "github.com/dujiao-next/internal/modules/diagnostics/contract"
)

const (
	accountsURL     = "https://chatgpt.com/backend-api/accounts/check/v4-2023-04-27"
	usageURL        = "https://chatgpt.com/backend-api/wham/usage"
	requestTimeout  = 12 * time.Second
	maxResponseSize = 1024 * 1024
)

// Client issues only read requests to fixed ChatGPT endpoints. It never consumes
// a reset, starts a subscription or retains a supplied access token.
type Client struct {
	client *http.Client
	now    func() time.Time
}

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
	return &Client{client: &configured, now: time.Now}
}

func (c *Client) Inspect(ctx context.Context, input diagnosticscontract.Input) (diagnosticscontract.Result, error) {
	token := strings.TrimSpace(input.AccessToken)
	accountID := strings.TrimSpace(input.AccountID)
	if !validToken(token) || (accountID != "" && !validAccountID(accountID)) {
		return diagnosticscontract.Result{}, diagnosticscontract.ErrInvalidInput
	}
	ctx, cancel := context.WithTimeout(ctx, requestTimeout)
	defer cancel()
	body, err := c.read(ctx, accountsURL, token, "")
	if err != nil {
		return diagnosticscontract.Result{}, err
	}
	defer clear(body)
	accounts, err := parseAccounts(body, token)
	if err != nil {
		return diagnosticscontract.Result{}, err
	}
	result := diagnosticscontract.Result{CheckedAt: c.now().UTC(), Accounts: accounts, UsageStatus: "not_requested"}
	if accountID == "" {
		return result, nil
	}
	found := false
	for _, account := range accounts {
		if account.ID == accountID {
			found = true
			break
		}
	}
	if !found {
		return diagnosticscontract.Result{}, diagnosticscontract.ErrAccountNotFound
	}
	result.SelectedAccountID = accountID
	result.UsageStatus = "unavailable"
	usageBody, err := c.read(ctx, usageURL, token, accountID)
	if err != nil {
		return result, nil
	}
	defer clear(usageBody)
	if usage := parseUsage(usageBody, result.CheckedAt); usage != nil {
		result.Usage = usage
		result.UsageStatus = "ready"
	}
	return result, nil
}

func validToken(value string) bool {
	if len(value) < 16 || len(value) > 16*1024 || strings.HasPrefix(value, "sk-") || strings.HasPrefix(value, "at-") {
		return false
	}
	for _, ch := range value {
		if !(ch >= 'a' && ch <= 'z' || ch >= 'A' && ch <= 'Z' || ch >= '0' && ch <= '9' || strings.ContainsRune("-._~+/=", ch)) {
			return false
		}
	}
	return true
}

func validAccountID(value string) bool {
	if len(value) == 0 || len(value) > 128 {
		return false
	}
	for _, ch := range value {
		if !(ch >= 'a' && ch <= 'z' || ch >= 'A' && ch <= 'Z' || ch >= '0' && ch <= '9' || ch == '-' || ch == '_') {
			return false
		}
	}
	return true
}

func (c *Client) read(ctx context.Context, endpoint, token, accountID string) ([]byte, error) {
	if ctx.Err() != nil {
		return nil, diagnosticscontract.ErrUnavailable
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, diagnosticscontract.ErrUnavailable
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")
	if accountID != "" {
		req.Header.Set("chatgpt-account-id", accountID)
		req.Header.Set("originator", "codex_cli_rs")
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, diagnosticscontract.ErrUnavailable
	}
	defer resp.Body.Close()
	if strings.EqualFold(resp.Header.Get("cf-mitigated"), "challenge") || (resp.StatusCode >= 300 && resp.StatusCode < 400) {
		return nil, diagnosticscontract.ErrChallenge
	}
	if resp.StatusCode == http.StatusUnauthorized {
		return nil, diagnosticscontract.ErrUnauthorized
	}
	if resp.StatusCode == http.StatusForbidden {
		return nil, diagnosticscontract.ErrChallenge
	}
	if resp.StatusCode != http.StatusOK {
		return nil, diagnosticscontract.ErrUnavailable
	}
	if resp.ContentLength > maxResponseSize {
		return nil, diagnosticscontract.ErrInvalidResponse
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseSize+1))
	if err != nil {
		clear(body)
		return nil, diagnosticscontract.ErrUnavailable
	}
	if len(body) > maxResponseSize {
		clear(body)
		return nil, diagnosticscontract.ErrInvalidResponse
	}
	return body, nil
}

func parseAccounts(body []byte, token string) ([]diagnosticscontract.Account, error) {
	var payload struct {
		Accounts map[string]struct {
			Account *struct {
				ID          string `json:"account_id"`
				Plan        string `json:"plan_type"`
				IsDefault   bool   `json:"is_default"`
				Disabled    bool   `json:"disabled"`
				Deactivated bool   `json:"deactivated"`
			} `json:"account"`
			Entitlement *struct {
				ExpiresAt *string `json:"expires_at"`
			} `json:"entitlement"`
		} `json:"accounts"`
	}
	if json.Unmarshal(body, &payload) != nil || len(payload.Accounts) == 0 || len(payload.Accounts) > 100 {
		return nil, diagnosticscontract.ErrInvalidResponse
	}
	accounts := make([]diagnosticscontract.Account, 0, len(payload.Accounts))
	seen := make(map[string]bool)
	for _, row := range payload.Accounts {
		account := row.Account
		if account == nil || !validAccountID(account.ID) || account.ID == token || seen[account.ID] {
			return nil, diagnosticscontract.ErrInvalidResponse
		}
		seen[account.ID] = true
		if account.Disabled || account.Deactivated {
			continue
		}
		result := diagnosticscontract.Account{ID: account.ID, Plan: publicPlan(account.Plan), IsDefault: account.IsDefault}
		if row.Entitlement != nil && row.Entitlement.ExpiresAt != nil && *row.Entitlement.ExpiresAt != "" {
			expiry, err := time.Parse(time.RFC3339, *row.Entitlement.ExpiresAt)
			if err != nil {
				return nil, diagnosticscontract.ErrInvalidResponse
			}
			expiry = expiry.UTC()
			result.ExpiresAt = &expiry
		}
		accounts = append(accounts, result)
	}
	if len(accounts) == 0 {
		return nil, diagnosticscontract.ErrInvalidResponse
	}
	sort.Slice(accounts, func(i, j int) bool {
		if accounts[i].IsDefault != accounts[j].IsDefault {
			return accounts[i].IsDefault
		}
		return accounts[i].ID < accounts[j].ID
	})
	return accounts, nil
}

func publicPlan(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "free", "chatgptfreeplan":
		return "free"
	case "go", "chatgptgoplan":
		return "go"
	case "plus", "chatgptplusplan":
		return "plus"
	case "pro", "chatgptproplan", "pro_100", "pro_200", "pro_500":
		return "pro"
	case "team", "chatgptteamplan":
		return "team"
	case "business", "chatgptbusinessplan":
		return "business"
	case "enterprise", "chatgptenterpriseplan":
		return "enterprise"
	case "edu", "chatgpteduplan":
		return "edu"
	default:
		return "unknown"
	}
}

type rawWindow struct {
	UsedPercent   *float64 `json:"used_percent"`
	WindowSeconds *int64   `json:"limit_window_seconds"`
	ResetAt       *int64   `json:"reset_at"`
	ResetAfter    *int64   `json:"reset_after_seconds"`
}

func parseUsage(body []byte, checkedAt time.Time) *diagnosticscontract.Usage {
	var payload struct {
		RateLimit *struct {
			Primary   *rawWindow `json:"primary_window"`
			Secondary *rawWindow `json:"secondary_window"`
		} `json:"rate_limit"`
	}
	if json.Unmarshal(body, &payload) != nil || payload.RateLimit == nil {
		return nil
	}
	usage := &diagnosticscontract.Usage{
		Primary: parseWindow(payload.RateLimit.Primary, checkedAt), Secondary: parseWindow(payload.RateLimit.Secondary, checkedAt),
	}
	if usage.Primary == nil && usage.Secondary == nil {
		return nil
	}
	return usage
}

func parseWindow(raw *rawWindow, checkedAt time.Time) *diagnosticscontract.UsageWindow {
	const longestWindow = 366 * 24 * 60 * 60
	if raw == nil || raw.UsedPercent == nil || raw.WindowSeconds == nil || math.IsNaN(*raw.UsedPercent) || math.IsInf(*raw.UsedPercent, 0) || *raw.UsedPercent < 0 || *raw.UsedPercent > 100 || *raw.WindowSeconds <= 0 || *raw.WindowSeconds > longestWindow {
		return nil
	}
	window := &diagnosticscontract.UsageWindow{UsedPercent: *raw.UsedPercent, WindowSeconds: *raw.WindowSeconds}
	if raw.ResetAt != nil && *raw.ResetAt > 0 {
		reset := time.Unix(*raw.ResetAt, 0).UTC()
		if reset.Year() >= 2000 && reset.Year() <= 2100 {
			window.ResetAt = &reset
		}
	} else if raw.ResetAfter != nil && *raw.ResetAfter >= 0 && *raw.ResetAfter <= longestWindow {
		reset := checkedAt.Add(time.Duration(*raw.ResetAfter) * time.Second)
		window.ResetAt = &reset
	}
	return window
}

var _ diagnosticscontract.Inspector = (*Client)(nil)
