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

	body, err := io.ReadAll(io.LimitReader(resp.Body, 16385))
	if err != nil || len(body) > 16384 {
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
