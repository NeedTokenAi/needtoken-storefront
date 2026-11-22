package moincashadapter

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/dujiao-next/internal/constants"
	paymentcontract "github.com/dujiao-next/internal/modules/payment/contract"
	gatewaycommon "github.com/dujiao-next/internal/modules/payment/infrastructure/gateway/common"
	"github.com/dujiao-next/internal/shared/jsonmap"
	"github.com/dujiao-next/internal/shared/money"

	"github.com/shopspring/decimal"
)

const (
	checkoutPath     = "/api/v1/checkout/sessions"
	maxResponseBytes = 256 << 10
	replayWindow     = 5 * time.Minute
)

type Config struct {
	APIURL             string
	MerchantID         string
	APIKey             string
	WebhookSecret      string
	NotifyURL          string
	ReturnURL          string
	PaymentEnvironment string
}

type moinCashAdapter struct {
	client *http.Client
	now    func() time.Time
}

// NewMoinCashAdapter 创建 MoinCash（moin.cash）收单网关适配器。
func NewMoinCashAdapter() paymentcontract.GatewayProvider {
	return &moinCashAdapter{
		client: &http.Client{Timeout: 15 * time.Second},
		now:    time.Now,
	}
}

var (
	_ paymentcontract.GatewayProvider         = (*moinCashAdapter)(nil)
	_ paymentcontract.GatewayCapturer         = (*moinCashAdapter)(nil)
	_ paymentcontract.GatewayCallbackVerifier = (*moinCashAdapter)(nil)
)

func (a *moinCashAdapter) Type() string {
	return constants.PaymentProviderMoinCash + ":"
}

func parseConfig(raw jsonmap.JSON) (*Config, error) {
	if raw == nil {
		return nil, paymentcontract.ErrGatewayConfigInvalid
	}
	getStr := func(key string) string {
		val, ok := raw[key]
		if !ok || val == nil {
			return ""
		}
		if s, ok := val.(string); ok {
			return strings.TrimSpace(s)
		}
		return strings.TrimSpace(fmt.Sprint(val))
	}
	cfg := &Config{
		APIURL:             strings.TrimRight(getStr("api_url"), "/"),
		MerchantID:         getStr("merchant_id"),
		APIKey:             getStr("api_key"),
		WebhookSecret:      getStr("webhook_secret"),
		NotifyURL:          getStr("notify_url"),
		ReturnURL:          getStr("return_url"),
		PaymentEnvironment: strings.ToLower(getStr("payment_environment")),
	}
	u, err := url.Parse(cfg.APIURL)
	if err != nil || u.Host == "" || (u.Scheme != "https" && u.Scheme != "http") {
		return nil, fmt.Errorf("%w: moincash api_url invalid", paymentcontract.ErrGatewayConfigInvalid)
	}
	if cfg.MerchantID == "" || cfg.APIKey == "" || len(cfg.WebhookSecret) < 16 {
		return nil, fmt.Errorf("%w: moincash merchant_id, api_key and webhook_secret(>=16) are required", paymentcontract.ErrGatewayConfigInvalid)
	}
	if cfg.PaymentEnvironment != "" && cfg.PaymentEnvironment != "live" && cfg.PaymentEnvironment != "sandbox" {
		return nil, fmt.Errorf("%w: moincash payment_environment must be live or sandbox", paymentcontract.ErrGatewayConfigInvalid)
	}
	return cfg, nil
}

func SignRequest(secret string, ts int64, method, path string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = fmt.Fprintf(mac, "%d.%s.%s.", ts, strings.ToUpper(strings.TrimSpace(method)), path)
	_, _ = mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}

func SignCallback(secret string, ts int64, tradeID, orderNo, status, amount, currency string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = fmt.Fprintf(mac, "moincash:%d:%s:%s:%s:%s:%s",
		ts,
		strings.TrimSpace(tradeID),
		strings.TrimSpace(orderNo),
		strings.ToLower(strings.TrimSpace(status)),
		strings.TrimSpace(amount),
		strings.ToUpper(strings.TrimSpace(currency)),
	)
	return hex.EncodeToString(mac.Sum(nil))
}

func (a *moinCashAdapter) ValidateConfig(raw jsonmap.JSON, channelType string) error {
	if _, err := parseConfig(raw); err != nil {
		return err
	}
	ct := strings.ToLower(strings.TrimSpace(channelType))
	switch ct {
	case "", "moincash", "usdt", "usdc", "usdt-trc20", "usdc-trc20", "crypto", "card", "alipay", "wechat":
		return nil
	default:
		return fmt.Errorf("%w: moincash channel_type %s", paymentcontract.ErrGatewayUnsupportedChannel, channelType)
	}
}

func (a *moinCashAdapter) CreatePayment(ctx context.Context, raw jsonmap.JSON, input paymentcontract.GatewayCreateInput) (*paymentcontract.GatewayCreateResult, error) {
	cfg, err := parseConfig(raw)
	if err != nil {
		return nil, err
	}
	if err := a.ValidateConfig(raw, input.ChannelType); err != nil {
		return nil, err
	}
	notifyURL := strings.TrimSpace(input.NotifyURL)
	if notifyURL == "" {
		notifyURL = cfg.NotifyURL
	}
	returnURL := strings.TrimSpace(input.ReturnURL)
	if returnURL == "" {
		returnURL = cfg.ReturnURL
	}
	returnURL = gatewaycommon.AppendQueryParams(returnURL, input.ReturnURLQuery)

	reqPayload := map[string]string{
		"order_no":     strings.TrimSpace(input.OrderNo),
		"subject":      strings.TrimSpace(input.Subject),
		"amount":       input.Amount.Decimal.StringFixed(2),
		"currency":     strings.ToUpper(strings.TrimSpace(input.Currency)),
		"channel_type": strings.ToLower(strings.TrimSpace(input.ChannelType)),
		"email":        strings.TrimSpace(input.Email),
		"notify_url":   notifyURL,
		"return_url":   returnURL,
	}
	rawBody, err := json.Marshal(reqPayload)
	if err != nil {
		return nil, paymentcontract.ErrGatewayRequestFailed
	}

	ts := a.now().UTC().Unix()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, cfg.APIURL+checkoutPath, bytes.NewReader(rawBody))
	if err != nil {
		return nil, paymentcontract.ErrGatewayRequestFailed
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+cfg.APIKey)
	req.Header.Set("X-Moin-Merchant", cfg.MerchantID)
	req.Header.Set("X-Moin-Timestamp", strconv.FormatInt(ts, 10))
	req.Header.Set("X-Moin-Signature", SignRequest(cfg.WebhookSecret, ts, http.MethodPost, checkoutPath, rawBody))

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", paymentcontract.ErrGatewayRequestFailed, err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		return nil, paymentcontract.ErrGatewayResponseInvalid
	}
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return nil, paymentcontract.ErrGatewayAuthFailed
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, paymentcontract.ErrGatewayRequestFailed
	}

	var parsed struct {
		TradeID            string `json:"trade_id"`
		PayURL             string `json:"pay_url"`
		QRCode             string `json:"qr_code"`
		Amount             string `json:"amount"`
		Currency           string `json:"currency"`
		DisplayChannelType string `json:"display_channel_type"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, paymentcontract.ErrGatewayResponseInvalid
	}
	tradeID := strings.TrimSpace(parsed.TradeID)
	payURL := strings.TrimSpace(parsed.PayURL)
	qrCode := strings.TrimSpace(parsed.QRCode)
	if tradeID == "" || (payURL == "" && qrCode == "") {
		return nil, paymentcontract.ErrGatewayResponseInvalid
	}
	mode, _ := input.Extra["interaction_mode"].(string)
	if strings.EqualFold(strings.TrimSpace(mode), constants.PaymentInteractionQR) && qrCode == "" {
		qrCode = payURL
		payURL = ""
	}
	return &paymentcontract.GatewayCreateResult{
		ProviderRef:        tradeID,
		RedirectURL:        payURL,
		QRCodeURL:          qrCode,
		AmountSent:         strings.TrimSpace(parsed.Amount),
		CurrencySent:       strings.ToUpper(strings.TrimSpace(parsed.Currency)),
		DisplayChannelType: strings.TrimSpace(parsed.DisplayChannelType),
		Payload: jsonmap.JSON{
			"provider": constants.PaymentProviderMoinCash,
			"trade_id": tradeID,
		},
	}, nil
}

func (a *moinCashAdapter) QueryPayment(ctx context.Context, raw jsonmap.JSON, providerRef string) (*paymentcontract.GatewayQueryResult, error) {
	cfg, err := parseConfig(raw)
	if err != nil {
		return nil, err
	}
	ref := strings.TrimSpace(providerRef)
	if ref == "" {
		return nil, paymentcontract.ErrGatewayRequestFailed
	}
	path := checkoutPath + "/" + url.PathEscape(ref)
	ts := a.now().UTC().Unix()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, cfg.APIURL+path, nil)
	if err != nil {
		return nil, paymentcontract.ErrGatewayRequestFailed
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+cfg.APIKey)
	req.Header.Set("X-Moin-Merchant", cfg.MerchantID)
	req.Header.Set("X-Moin-Timestamp", strconv.FormatInt(ts, 10))
	req.Header.Set("X-Moin-Signature", SignRequest(cfg.WebhookSecret, ts, http.MethodGet, path, nil))

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", paymentcontract.ErrGatewayRequestFailed, err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		return nil, paymentcontract.ErrGatewayResponseInvalid
	}
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return nil, paymentcontract.ErrGatewayAuthFailed
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, paymentcontract.ErrGatewayRequestFailed
	}
	var parsed struct {
		TradeID  string     `json:"trade_id"`
		Status   string     `json:"status"`
		Amount   string     `json:"amount"`
		Currency string     `json:"currency"`
		PaidAt   *time.Time `json:"paid_at"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, paymentcontract.ErrGatewayResponseInvalid
	}
	amt, err := decimal.NewFromString(strings.TrimSpace(parsed.Amount))
	if err != nil || !amt.IsPositive() {
		return nil, paymentcontract.ErrGatewayResponseInvalid
	}
	return &paymentcontract.GatewayQueryResult{
		ProviderRef: strings.TrimSpace(parsed.TradeID),
		Status:      mapStatus(parsed.Status),
		Amount:      money.FromDecimal(amt),
		Currency:    strings.ToUpper(strings.TrimSpace(parsed.Currency)),
		PaidAt:      parsed.PaidAt,
		Payload: jsonmap.JSON{
			"provider": constants.PaymentProviderMoinCash,
			"status":   parsed.Status,
		},
	}, nil
}

func (a *moinCashAdapter) VerifyCallback(raw jsonmap.JSON, _ map[string][]string, body []byte) (*paymentcontract.GatewayCallbackResult, error) {
	cfg, err := parseConfig(raw)
	if err != nil {
		return nil, err
	}
	var payload struct {
		Provider  string     `json:"provider"`
		TradeID   string     `json:"trade_id"`
		OrderNo   string     `json:"order_no"`
		Status    string     `json:"status"`
		Amount    string     `json:"amount"`
		Currency  string     `json:"currency"`
		Timestamp int64      `json:"timestamp"`
		Signature string     `json:"signature"`
		PaidAt    *time.Time `json:"paid_at,omitempty"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, paymentcontract.ErrGatewayResponseInvalid
	}
	if !strings.EqualFold(strings.TrimSpace(payload.Provider), constants.PaymentProviderMoinCash) ||
		strings.TrimSpace(payload.TradeID) == "" ||
		strings.TrimSpace(payload.OrderNo) == "" ||
		strings.TrimSpace(payload.Signature) == "" ||
		payload.Timestamp <= 0 {
		return nil, paymentcontract.ErrGatewayResponseInvalid
	}
	now := a.now().UTC()
	callbackTime := time.Unix(payload.Timestamp, 0).UTC()
	if callbackTime.Before(now.Add(-replayWindow)) || callbackTime.After(now.Add(replayWindow)) {
		return nil, paymentcontract.ErrGatewaySignatureInvalid
	}
	expectedSig := SignCallback(cfg.WebhookSecret, payload.Timestamp, payload.TradeID, payload.OrderNo, payload.Status, payload.Amount, payload.Currency)
	gotSig, err := hex.DecodeString(strings.TrimSpace(payload.Signature))
	if err != nil {
		return nil, paymentcontract.ErrGatewaySignatureInvalid
	}
	wantBytes, _ := hex.DecodeString(expectedSig)
	if !hmac.Equal(gotSig, wantBytes) {
		return nil, paymentcontract.ErrGatewaySignatureInvalid
	}
	amt, err := decimal.NewFromString(strings.TrimSpace(payload.Amount))
	if err != nil || !amt.IsPositive() {
		return nil, paymentcontract.ErrGatewayResponseInvalid
	}
	return &paymentcontract.GatewayCallbackResult{
		OrderNo:     strings.TrimSpace(payload.OrderNo),
		ProviderRef: strings.TrimSpace(payload.TradeID),
		Status:      mapStatus(payload.Status),
		Amount:      money.FromDecimal(amt),
		Currency:    strings.ToUpper(strings.TrimSpace(payload.Currency)),
		PaidAt:      payload.PaidAt,
		Payload: jsonmap.JSON{
			"provider": constants.PaymentProviderMoinCash,
			"trade_id": strings.TrimSpace(payload.TradeID),
			"status":   strings.TrimSpace(payload.Status),
		},
	}, nil
}

func mapStatus(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "paid", "succeeded", "success", "completed":
		return constants.PaymentStatusSuccess
	case "expired", "canceled", "cancelled":
		return constants.PaymentStatusExpired
	case "failed", "declined":
		return constants.PaymentStatusFailed
	default:
		return constants.PaymentStatusPending
	}
}
