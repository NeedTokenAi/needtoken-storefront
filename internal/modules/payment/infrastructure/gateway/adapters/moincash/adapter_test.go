package moincashadapter

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/dujiao-next/internal/constants"
	paymentcontract "github.com/dujiao-next/internal/modules/payment/contract"
	"github.com/dujiao-next/internal/shared/jsonmap"
	"github.com/dujiao-next/internal/shared/money"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMoinCashCreateQueryAndCallbackVerification(t *testing.T) {
	fixedNow := time.Date(2026, 11, 22, 12, 0, 0, 0, time.UTC)
	secret := "whsec_moin_test_secret_123456"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "Bearer mk_live_123", r.Header.Get("Authorization"))
		assert.Equal(t, "mch_01", r.Header.Get("X-Moin-Merchant"))
		assert.NotEmpty(t, r.Header.Get("X-Moin-Signature"))
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodPost && r.URL.Path == "/api/v1/checkout/sessions" {
			_ = json.NewEncoder(w).Encode(map[string]string{
				"trade_id":             "mc_tr_1001",
				"pay_url":              "https://pay.moin.cash/s/mc_tr_1001",
				"amount":               "20.00",
				"currency":             "USD",
				"display_channel_type": "usdt.trc20",
			})
			return
		}
		if r.Method == http.MethodGet && r.URL.Path == "/api/v1/checkout/sessions/mc_tr_1001" {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"trade_id": "mc_tr_1001",
				"status":   "succeeded",
				"amount":   "20.00",
				"currency": "USD",
				"paid_at":  fixedNow.Format(time.RFC3339),
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	adapter := &moinCashAdapter{
		client: server.Client(),
		now:    func() time.Time { return fixedNow },
	}
	cfg := jsonmap.JSON{
		"api_url":             server.URL,
		"merchant_id":         "mch_01",
		"api_key":             "mk_live_123",
		"webhook_secret":      secret,
		"payment_environment": "live",
	}

	require.NoError(t, adapter.ValidateConfig(cfg, "usdt"))
	require.ErrorIs(t, adapter.ValidateConfig(jsonmap.JSON{"api_url": server.URL}, "usdt"), paymentcontract.ErrGatewayConfigInvalid)

	created, err := adapter.CreatePayment(context.Background(), cfg, paymentcontract.GatewayCreateInput{
		OrderNo:     "DJP_ORDER_1",
		Subject:     "NeedToken Plan",
		Amount:      money.FromDecimal(decimal.NewFromInt(20)),
		Currency:    "USD",
		ChannelType: "usdt",
	})
	require.NoError(t, err)
	assert.Equal(t, "mc_tr_1001", created.ProviderRef)
	assert.Equal(t, "https://pay.moin.cash/s/mc_tr_1001", created.RedirectURL)
	assert.Equal(t, "usdt.trc20", created.DisplayChannelType)

	queried, err := adapter.QueryPayment(context.Background(), cfg, "mc_tr_1001")
	require.NoError(t, err)
	assert.Equal(t, constants.PaymentStatusSuccess, queried.Status)
	assert.Equal(t, "USD", queried.Currency)

	ts := fixedNow.Unix()
	sig := SignCallback(secret, ts, "mc_tr_1001", "DJP_ORDER_1", "succeeded", "20.00", "USD")
	rawCallback, _ := json.Marshal(map[string]any{
		"provider":  "moincash",
		"trade_id":  "mc_tr_1001",
		"order_no":  "DJP_ORDER_1",
		"status":    "succeeded",
		"amount":    "20.00",
		"currency":  "USD",
		"timestamp": ts,
		"signature": sig,
	})
	verified, err := adapter.VerifyCallback(cfg, nil, rawCallback)
	require.NoError(t, err)
	assert.Equal(t, constants.PaymentStatusSuccess, verified.Status)
	assert.Equal(t, "DJP_ORDER_1", verified.OrderNo)

	// Replay outside window must fail closed.
	staleTS := fixedNow.Add(-10 * time.Minute).Unix()
	staleSig := SignCallback(secret, staleTS, "mc_tr_1001", "DJP_ORDER_1", "succeeded", "20.00", "USD")
	staleBody, _ := json.Marshal(map[string]any{
		"provider":  "moincash",
		"trade_id":  "mc_tr_1001",
		"order_no":  "DJP_ORDER_1",
		"status":    "succeeded",
		"amount":    "20.00",
		"currency":  "USD",
		"timestamp": staleTS,
		"signature": staleSig,
	})
	_, err = adapter.VerifyCallback(cfg, nil, staleBody)
	assert.True(t, errors.Is(err, paymentcontract.ErrGatewaySignatureInvalid))
}
