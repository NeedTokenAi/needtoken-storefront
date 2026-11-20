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
	ErrAssetClaimUnavailable = errors.New("asset claim unavailable")
	ErrAssetClaimInvalid     = errors.New("asset claim invalid")
	ErrAssetClaimForbidden   = errors.New("asset claim forbidden")
	ErrAssetClaimExhausted   = errors.New("asset claim reveal exhausted")
	ErrAssetClaimNotFound    = errors.New("asset claim not found")
	ErrAssetClaimConflict    = errors.New("asset claim conflict")
)

type AssetClaimVerification struct {
	Source          string     `json:"source"`
	ObservedAt      time.Time  `json:"observed_at"`
	RemoteAccountID string     `json:"remote_account_id,omitempty"`
	PlanType        string     `json:"plan_type,omitempty"`
	SubscriptionEnd *time.Time `json:"subscription_end,omitempty"`
	EvidenceRef     string     `json:"evidence_ref,omitempty"`
	EvidenceSHA256  string     `json:"evidence_sha256"`
}

type AssetClaimStatus struct {
	Eligible         bool                    `json:"eligible"`
	AssetID          string                  `json:"asset_id,omitempty"`
	SKUCode          string                  `json:"sku_code,omitempty"`
	Platform         string                  `json:"platform,omitempty"`
	SaleMode         string                  `json:"sale_mode,omitempty"`
	MaskedIdentifier string                  `json:"masked_identifier,omitempty"`
	DeliveredAt      *time.Time              `json:"delivered_at,omitempty"`
	ExpiresAt        *time.Time              `json:"expires_at,omitempty"`
	RentalExpiresAt  *time.Time              `json:"rental_expires_at,omitempty"`
	Verification     *AssetClaimVerification `json:"verification,omitempty"`
	MaxReveals       int                     `json:"max_reveals"`
	RevealCount      int                     `json:"reveal_count"`
	RemainingReveals int                     `json:"remaining_reveals"`
	FirstRevealedAt  *time.Time              `json:"first_revealed_at,omitempty"`
	LastRevealedAt   *time.Time              `json:"last_revealed_at,omitempty"`
}

type AssetClaimReveal struct {
	AssetClaimStatus
	AccountIdentifier string            `json:"account_identifier"`
	SecretPayload     string            `json:"secret_payload"`
	DeliveryNotes     string            `json:"delivery_notes,omitempty"`
	Extra             map[string]string `json:"extra,omitempty"`
}

func (a *DujiaoNextAdapter) AssetClaimStatus(ctx context.Context, upstreamOrderID uint, buyerRef string) (AssetClaimStatus, error) {
	var result struct {
		OK bool `json:"ok"`
		AssetClaimStatus
	}
	path := "/api/v1/upstream/orders/" + strconv.FormatUint(uint64(upstreamOrderID), 10) + "/asset-claim"
	if trimmed := strings.TrimSpace(buyerRef); trimmed != "" {
		path += "?buyer_ref=" + url.QueryEscape(trimmed)
	}
	if err := a.assetClaimRequest(ctx, http.MethodGet, path, nil, &result); err != nil {
		return AssetClaimStatus{}, err
	}
	if !result.OK {
		return AssetClaimStatus{}, ErrAssetClaimUnavailable
	}
	return result.AssetClaimStatus, nil
}

func (a *DujiaoNextAdapter) RevealAssetClaim(ctx context.Context, upstreamOrderID uint, buyerRef, actor string) (AssetClaimReveal, error) {
	var result struct {
		OK bool `json:"ok"`
		AssetClaimReveal
	}
	path := "/api/v1/upstream/orders/" + strconv.FormatUint(uint64(upstreamOrderID), 10) + "/asset-claim/reveal"
	payload := struct {
		BuyerRef string `json:"buyer_ref"`
		Actor    string `json:"actor"`
	}{
		BuyerRef: strings.TrimSpace(buyerRef),
		Actor:    strings.TrimSpace(actor),
	}
	if err := a.assetClaimRequest(ctx, http.MethodPost, path, payload, &result); err != nil {
		return AssetClaimReveal{}, err
	}
	if !result.OK || strings.TrimSpace(result.SecretPayload) == "" {
		return AssetClaimReveal{}, ErrAssetClaimUnavailable
	}
	return result.AssetClaimReveal, nil
}

// assetClaimRequest performs bounded, redirect-free, cookie-free signed I/O
// without ever logging response bodies that carry decrypted buyer credentials.
func (a *DujiaoNextAdapter) assetClaimRequest(ctx context.Context, method, path string, input, output any) error {
	base, err := url.Parse(a.baseURL)
	if err != nil || (base.Scheme != "https" && base.Scheme != "http") || base.Host == "" || base.User != nil || base.Path != "" || base.RawQuery != "" || base.Fragment != "" {
		return ErrAssetClaimUnavailable
	}
	ctx, cancel := context.WithTimeout(ctx, 6*time.Second)
	defer cancel()

	var raw []byte
	if input != nil {
		raw, err = json.Marshal(input)
		if err != nil {
			return ErrAssetClaimInvalid
		}
	}
	signPath := path
	if idx := strings.IndexByte(path, '?'); idx >= 0 {
		signPath = path[:idx]
	}

	req, err := http.NewRequestWithContext(ctx, method, a.baseURL+path, bytes.NewReader(raw))
	if err != nil {
		return ErrAssetClaimUnavailable
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
		return ErrAssetClaimUnavailable
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 32769))
	if err != nil || len(body) > 32768 {
		return ErrAssetClaimUnavailable
	}
	if resp.StatusCode != http.StatusOK {
		return mapAssetClaimError(resp.StatusCode, body)
	}
	decoder := json.NewDecoder(bytes.NewReader(body))
	if decoder.Decode(output) != nil || decoder.Decode(new(any)) != io.EOF {
		return ErrAssetClaimUnavailable
	}
	return nil
}

func mapAssetClaimError(statusCode int, body []byte) error {
	var payload struct {
		ErrorCode string `json:"error_code"`
	}
	_ = json.Unmarshal(body, &payload)
	switch payload.ErrorCode {
	case "invalid_request":
		return ErrAssetClaimInvalid
	case "asset_claim_forbidden":
		return ErrAssetClaimForbidden
	case "asset_claim_exhausted":
		return ErrAssetClaimExhausted
	case "asset_claim_not_found", "order_not_found":
		return ErrAssetClaimNotFound
	case "asset_claim_not_ready", "asset_claim_conflict":
		return ErrAssetClaimConflict
	}
	switch statusCode {
	case http.StatusBadRequest:
		return ErrAssetClaimInvalid
	case http.StatusForbidden:
		return ErrAssetClaimForbidden
	case http.StatusNotFound:
		return ErrAssetClaimNotFound
	case http.StatusConflict:
		return ErrAssetClaimConflict
	default:
		return ErrAssetClaimUnavailable
	}
}
