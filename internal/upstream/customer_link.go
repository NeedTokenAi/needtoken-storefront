package upstream

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

var (
	ErrCustomerLinkUnavailable = errors.New("customer link unavailable")
	ErrCustomerLinkInvalid     = errors.New("customer link invalid")
	ErrCustomerLinkExpired     = errors.New("customer link expired")
	ErrCustomerLinkConflict    = errors.New("customer link conflict")
	ErrCustomerLinkForbidden   = errors.New("customer link forbidden")
)

type CustomerLinkStatus struct {
	Enabled       bool       `json:"enabled"`
	Linked        bool       `json:"linked"`
	Status        string     `json:"status"`
	NativeUserID  int        `json:"native_user_id,omitempty"`
	Version       uint64     `json:"version,omitempty"`
	PendingOrders int        `json:"pending_orders"`
	UpdatedAt     *time.Time `json:"updated_at,omitempty"`
}

type CustomerLinkChallenge struct {
	ChallengeID  string    `json:"challenge_id"`
	AuthorizeURL string    `json:"authorize_url"`
	ExpiresAt    time.Time `json:"expires_at"`
}

type CustomerLinkComplete struct {
	Linked         bool   `json:"linked"`
	Status         string `json:"status"`
	NativeUserID   int    `json:"native_user_id,omitempty"`
	NextPath       string `json:"next_path,omitempty"`
	AwakenedOrders int    `json:"awakened_orders"`
}

type CustomerConsoleSSOInput struct {
	MerchantCustomerID string `json:"merchant_customer_id"`
	Email              string `json:"email"`
	EmailVerified      bool   `json:"email_verified"`
	DisplayName        string `json:"display_name,omitempty"`
	TargetPath         string `json:"target_path,omitempty"`
	NextPath           string `json:"next_path,omitempty"`
}

type CustomerConsoleSSOResult struct {
	Mode           string `json:"mode"`
	RedirectURL    string `json:"redirect_url"`
	Linked         bool   `json:"linked"`
	NativeUserID   int    `json:"native_user_id,omitempty"`
	AutoCreated    bool   `json:"auto_created"`
	AwakenedOrders int    `json:"awakened_orders"`
}

type CustomerAutoLinkInput struct {
	StorefrontUserID   string `json:"storefront_user_id,omitempty"`
	MerchantCustomerID string `json:"merchant_customer_id,omitempty"`
	Email              string `json:"email"`
	EmailVerified      bool   `json:"email_verified"`
	DisplayName        string `json:"display_name,omitempty"`
	Username           string `json:"username,omitempty"`
}

type CustomerAutoLinkResult struct {
	Linked         bool   `json:"linked"`
	Status         string `json:"status"`
	NativeUserID   int    `json:"native_user_id,omitempty"`
	AwakenedOrders int    `json:"awakened_orders,omitempty"`
}

type CustomerAPISubscriptionSummary struct {
	ID                  int64  `json:"id"`
	PlanID              int    `json:"plan_id"`
	PlanTitle           string `json:"plan_title,omitempty"`
	Status              string `json:"status"`
	AmountTotal         int64  `json:"amount_total"`
	AmountUsed          int64  `json:"amount_used"`
	StartTime           int64  `json:"start_time"`
	EndTime             int64  `json:"end_time"`
	AllowWalletOverflow bool   `json:"allow_wallet_overflow"`
}

type CustomerAPISummaryResult struct {
	Linked              bool                             `json:"linked"`
	NativeUserID        int                              `json:"native_user_id,omitempty"`
	Quota               int64                            `json:"quota,omitempty"`
	UsedQuota           int64                            `json:"used_quota,omitempty"`
	RequestCount        int64                            `json:"request_count,omitempty"`
	Group               string                           `json:"group,omitempty"`
	ActiveSubscriptions []CustomerAPISubscriptionSummary `json:"active_subscriptions,omitempty"`
}

func (a *DujiaoNextAdapter) CustomerLinkStatus(ctx context.Context, merchantCustomerID string) (CustomerLinkStatus, error) {
	var result struct {
		OK bool `json:"ok"`
		CustomerLinkStatus
	}
	path := "/api/v1/upstream/customer-links/status?merchant_customer_id=" + url.QueryEscape(strings.TrimSpace(merchantCustomerID))
	if err := a.customerLinkRequest(ctx, http.MethodGet, path, nil, &result); err != nil {
		return CustomerLinkStatus{}, err
	}
	if !result.OK {
		return CustomerLinkStatus{}, ErrCustomerLinkUnavailable
	}
	return result.CustomerLinkStatus, nil
}

func (a *DujiaoNextAdapter) StartCustomerLink(ctx context.Context, merchantCustomerID, nextPath string) (CustomerLinkChallenge, error) {
	var result struct {
		OK bool `json:"ok"`
		CustomerLinkChallenge
	}
	payload := struct {
		MerchantCustomerID string `json:"merchant_customer_id"`
		NextPath           string `json:"next_path,omitempty"`
	}{
		MerchantCustomerID: strings.TrimSpace(merchantCustomerID),
		NextPath:           strings.TrimSpace(nextPath),
	}
	if err := a.customerLinkRequest(ctx, http.MethodPost, "/api/v1/upstream/customer-links/challenges", payload, &result); err != nil {
		return CustomerLinkChallenge{}, err
	}
	if !result.OK || strings.TrimSpace(result.ChallengeID) == "" || strings.TrimSpace(result.AuthorizeURL) == "" {
		return CustomerLinkChallenge{}, ErrCustomerLinkUnavailable
	}
	return result.CustomerLinkChallenge, nil
}

func (a *DujiaoNextAdapter) CompleteCustomerLink(ctx context.Context, merchantCustomerID, challengeID, approval string) (CustomerLinkComplete, error) {
	var result struct {
		OK bool `json:"ok"`
		CustomerLinkComplete
	}
	payload := struct {
		MerchantCustomerID string `json:"merchant_customer_id"`
		ChallengeID        string `json:"challenge_id"`
		Approval           string `json:"approval"`
	}{
		MerchantCustomerID: strings.TrimSpace(merchantCustomerID),
		ChallengeID:        strings.TrimSpace(challengeID),
		Approval:           strings.TrimSpace(approval),
	}
	if err := a.customerLinkRequest(ctx, http.MethodPost, "/api/v1/upstream/customer-links/complete", payload, &result); err != nil {
		return CustomerLinkComplete{}, err
	}
	if !result.OK || !result.Linked {
		return CustomerLinkComplete{}, ErrCustomerLinkUnavailable
	}
	return result.CustomerLinkComplete, nil
}

func (a *DujiaoNextAdapter) ConsoleSSOCustomerLink(ctx context.Context, input CustomerConsoleSSOInput) (CustomerConsoleSSOResult, error) {
	var result struct {
		OK *bool `json:"ok,omitempty"`
		CustomerConsoleSSOResult
	}
	payload := CustomerConsoleSSOInput{
		MerchantCustomerID: strings.TrimSpace(input.MerchantCustomerID),
		Email:              strings.TrimSpace(input.Email),
		EmailVerified:      input.EmailVerified,
		DisplayName:        strings.TrimSpace(input.DisplayName),
		TargetPath:         strings.TrimSpace(input.TargetPath),
		NextPath:           strings.TrimSpace(input.NextPath),
	}
	if err := a.customerLinkRequest(ctx, http.MethodPost, "/api/v1/upstream/customer-links/console-sso", payload, &result); err != nil {
		return CustomerConsoleSSOResult{}, err
	}
	if result.OK != nil && !*result.OK {
		return CustomerConsoleSSOResult{}, ErrCustomerLinkUnavailable
	}
	result.RedirectURL = strings.TrimSpace(result.RedirectURL)
	result.Mode = strings.TrimSpace(result.Mode)
	if result.RedirectURL == "" {
		return CustomerConsoleSSOResult{}, ErrCustomerLinkUnavailable
	}
	if result.Mode == "" {
		if result.Linked {
			result.Mode = "sso"
		} else {
			result.Mode = "bind_challenge"
		}
	}
	return result.CustomerConsoleSSOResult, nil
}

func (a *DujiaoNextAdapter) AutoLink(ctx context.Context, req CustomerAutoLinkInput) (CustomerAutoLinkResult, error) {
	var result struct {
		OK *bool `json:"ok,omitempty"`
		CustomerAutoLinkResult
	}
	payload := CustomerAutoLinkInput{
		StorefrontUserID:   strings.TrimSpace(req.StorefrontUserID),
		MerchantCustomerID: strings.TrimSpace(req.MerchantCustomerID),
		Email:              strings.TrimSpace(req.Email),
		EmailVerified:      req.EmailVerified,
		DisplayName:        strings.TrimSpace(req.DisplayName),
		Username:           strings.TrimSpace(req.Username),
	}
	if err := a.customerLinkRequest(ctx, http.MethodPost, "/api/v1/upstream/customer-links/auto-link", payload, &result); err != nil {
		return CustomerAutoLinkResult{}, err
	}
	if result.OK != nil && !*result.OK {
		return CustomerAutoLinkResult{}, ErrCustomerLinkUnavailable
	}
	out := result.CustomerAutoLinkResult
	out.Status = strings.TrimSpace(out.Status)
	if !out.Linked || out.Status == "existing_requires_link" {
		// Never expose native account identity when manual two-sided confirmation is required.
		out.Linked = false
		out.NativeUserID = 0
		out.AwakenedOrders = 0
	}
	return out, nil
}

func (a *DujiaoNextAdapter) APISummary(ctx context.Context, storefrontUserID string) (CustomerAPISummaryResult, error) {
	var result struct {
		OK *bool `json:"ok,omitempty"`
		CustomerAPISummaryResult
	}
	path := "/api/v1/upstream/customer-links/api-summary?storefront_user_id=" + url.QueryEscape(strings.TrimSpace(storefrontUserID))
	if err := a.customerLinkRequest(ctx, http.MethodGet, path, nil, &result); err != nil {
		return CustomerAPISummaryResult{}, err
	}
	if result.OK != nil && !*result.OK {
		return CustomerAPISummaryResult{}, ErrCustomerLinkUnavailable
	}
	out := result.CustomerAPISummaryResult
	if !out.Linked {
		return CustomerAPISummaryResult{Linked: false}, nil
	}
	return out, nil
}

// customerLinkRequest signs the exact request path (without query) and body with
// merchant HMAC credentials, disables redirects/cookies, and bounds I/O so
// challenge/approval tokens never leak through redirects or logs.
func (a *DujiaoNextAdapter) customerLinkRequest(ctx context.Context, method, path string, input, output any) error {
	base, err := url.Parse(a.baseURL)
	if err != nil || (base.Scheme != "https" && base.Scheme != "http") || base.Host == "" || base.User != nil || base.Path != "" || base.RawQuery != "" || base.Fragment != "" {
		return ErrCustomerLinkUnavailable
	}
	ctx, cancel := context.WithTimeout(ctx, 6*time.Second)
	defer cancel()

	var raw []byte
	if input != nil {
		raw, err = json.Marshal(input)
		if err != nil {
			return ErrCustomerLinkInvalid
		}
	}

	signPath := path
	if idx := strings.IndexByte(path, '?'); idx >= 0 {
		signPath = path[:idx]
	}

	req, err := http.NewRequestWithContext(ctx, method, a.baseURL+path, bytes.NewReader(raw))
	if err != nil {
		return ErrCustomerLinkUnavailable
	}
	now := time.Now().Unix()
	req.Header.Set(HeaderApiKey, a.apiKey)
	req.Header.Set(HeaderTimestamp, strconv.FormatInt(now, 10))
	req.Header.Set(HeaderSignature, Sign(a.apiSecret, method, signPath, now, raw))
	req.Header.Set("Accept", "application/json")
	if input != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	client := *a.client
	client.Jar = nil
	client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }

	resp, err := client.Do(req)
	if err != nil {
		return ErrCustomerLinkUnavailable
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 65537))
	if err != nil || len(body) > 65536 {
		return ErrCustomerLinkUnavailable
	}
	if resp.StatusCode != http.StatusOK {
		return mapCustomerLinkError(resp.StatusCode, body)
	}
	decoder := json.NewDecoder(bytes.NewReader(body))
	if decoder.Decode(output) != nil || decoder.Decode(new(any)) != io.EOF {
		return ErrCustomerLinkUnavailable
	}
	return nil
}

func mapCustomerLinkError(statusCode int, body []byte) error {
	var payload struct {
		ErrorCode string `json:"error_code"`
	}
	_ = json.Unmarshal(body, &payload)
	switch payload.ErrorCode {
	case "invalid_request":
		return ErrCustomerLinkInvalid
	case "link_challenge_expired", "link_challenge_not_found":
		return ErrCustomerLinkExpired
	case "link_challenge_forbidden":
		return ErrCustomerLinkForbidden
	case "link_challenge_consumed", "account_already_linked", "native_account_already_linked", "native_account_unavailable":
		return ErrCustomerLinkConflict
	}
	switch statusCode {
	case http.StatusBadRequest:
		return ErrCustomerLinkInvalid
	case http.StatusGone, http.StatusNotFound:
		return ErrCustomerLinkExpired
	case http.StatusForbidden:
		return ErrCustomerLinkForbidden
	case http.StatusConflict:
		return ErrCustomerLinkConflict
	default:
		return ErrCustomerLinkUnavailable
	}
}
