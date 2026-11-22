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

const (
	UniversalGiftCardClaimPath   = "/api/v1/upstream/universal-gift-cards/claim"
	UniversalGiftCardConfirmPath = "/api/v1/upstream/universal-gift-cards/confirm"
	UniversalGiftCardReleasePath = "/api/v1/upstream/universal-gift-cards/release"
)

var (
	ErrUniversalGiftCardUnavailable = errors.New("universal gift card unavailable")
	ErrUniversalGiftCardInvalid     = errors.New("universal gift card invalid")
	ErrUniversalGiftCardNotFound    = errors.New("universal gift card not found")
	ErrUniversalGiftCardRedeemed    = errors.New("universal gift card redeemed")
	ErrUniversalGiftCardDisabled    = errors.New("universal gift card disabled")
	ErrUniversalGiftCardExpired     = errors.New("universal gift card expired")
)

type UniversalGiftCardClaimInput struct {
	Code    string `json:"code"`
	Target  string `json:"target"`
	UserRef string `json:"user_ref"`
	ClaimID string `json:"claim_id"`
}

type UniversalGiftCardClaimResult struct {
	OK           bool       `json:"ok"`
	CardID       string     `json:"card_id"`
	BatchNo      string     `json:"batch_no"`
	Name         string     `json:"name"`
	Code         string     `json:"code"`
	ClaimID      string     `json:"claim_id"`
	FaceUSDCents int64      `json:"face_usd_cents"`
	ExpiresAt    *time.Time `json:"expires_at,omitempty"`
}

type UniversalGiftCardConfirmInput struct {
	Code          string `json:"code"`
	ClaimID       string `json:"claim_id"`
	TxnRef        string `json:"txn_ref"`
	CreditDisplay string `json:"credit_display,omitempty"`
}

type UniversalGiftCardReleaseInput struct {
	Code    string `json:"code"`
	ClaimID string `json:"claim_id"`
	Reason  string `json:"reason,omitempty"`
}

func (a *DujiaoNextAdapter) ClaimUniversalGiftCard(ctx context.Context, input UniversalGiftCardClaimInput) (UniversalGiftCardClaimResult, error) {
	var out UniversalGiftCardClaimResult
	if err := a.universalGiftCardRequest(ctx, http.MethodPost, UniversalGiftCardClaimPath, input, &out); err != nil {
		return UniversalGiftCardClaimResult{}, err
	}
	if !out.OK || out.FaceUSDCents <= 0 || strings.TrimSpace(out.ClaimID) == "" {
		return UniversalGiftCardClaimResult{}, ErrUniversalGiftCardUnavailable
	}
	return out, nil
}

func (a *DujiaoNextAdapter) ConfirmUniversalGiftCard(ctx context.Context, input UniversalGiftCardConfirmInput) error {
	var out struct {
		OK bool `json:"ok"`
	}
	return a.universalGiftCardRequest(ctx, http.MethodPost, UniversalGiftCardConfirmPath, input, &out)
}

func (a *DujiaoNextAdapter) ReleaseUniversalGiftCard(ctx context.Context, input UniversalGiftCardReleaseInput) error {
	var out struct {
		OK bool `json:"ok"`
	}
	return a.universalGiftCardRequest(ctx, http.MethodPost, UniversalGiftCardReleasePath, input, &out)
}

func (a *DujiaoNextAdapter) universalGiftCardRequest(ctx context.Context, method, path string, input, output any) error {
	base, err := url.Parse(a.baseURL)
	if err != nil || (base.Scheme != "https" && base.Scheme != "http") || base.Host == "" || base.User != nil || base.Path != "" || base.RawQuery != "" || base.Fragment != "" {
		return ErrUniversalGiftCardUnavailable
	}
	ctx, cancel := context.WithTimeout(ctx, 6*time.Second)
	defer cancel()

	raw, err := json.Marshal(input)
	if err != nil {
		return ErrUniversalGiftCardInvalid
	}

	req, err := http.NewRequestWithContext(ctx, method, a.baseURL+path, bytes.NewReader(raw))
	if err != nil {
		return ErrUniversalGiftCardUnavailable
	}
	now := time.Now().Unix()
	req.Header.Set(HeaderApiKey, a.apiKey)
	req.Header.Set(HeaderTimestamp, strconv.FormatInt(now, 10))
	req.Header.Set(HeaderSignature, Sign(a.apiSecret, method, path, now, raw))
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")

	client := *a.client
	client.Jar = nil
	client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }

	resp, err := client.Do(req)
	if err != nil {
		return ErrUniversalGiftCardUnavailable
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 65537))
	if err != nil || len(body) > 65536 {
		return ErrUniversalGiftCardUnavailable
	}
	if resp.StatusCode != http.StatusOK {
		return mapUniversalGiftCardError(resp.StatusCode, body)
	}
	decoder := json.NewDecoder(bytes.NewReader(body))
	if decoder.Decode(output) != nil || decoder.Decode(new(any)) != io.EOF {
		return ErrUniversalGiftCardUnavailable
	}
	return nil
}

func mapUniversalGiftCardError(statusCode int, body []byte) error {
	var payload struct {
		ErrorCode string `json:"error_code"`
	}
	_ = json.Unmarshal(body, &payload)
	switch payload.ErrorCode {
	case "not_found":
		return ErrUniversalGiftCardNotFound
	case "already_redeemed", "card_in_progress":
		return ErrUniversalGiftCardRedeemed
	case "card_disabled":
		return ErrUniversalGiftCardDisabled
	case "card_expired":
		return ErrUniversalGiftCardExpired
	case "invalid_request":
		return ErrUniversalGiftCardInvalid
	}
	switch statusCode {
	case http.StatusNotFound:
		return ErrUniversalGiftCardNotFound
	case http.StatusConflict:
		return ErrUniversalGiftCardRedeemed
	case http.StatusGone:
		return ErrUniversalGiftCardExpired
	case http.StatusBadRequest:
		return ErrUniversalGiftCardInvalid
	default:
		return ErrUniversalGiftCardUnavailable
	}
}
