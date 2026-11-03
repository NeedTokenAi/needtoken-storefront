package upstream

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"net/url"

	paymentcontract "github.com/dujiao-next/internal/modules/payment/contract"
)

func (a *DujiaoNextAdapter) ReservationIdentity() string {
	h := sha256.Sum256([]byte(a.baseURL + "\x00" + a.apiKey))
	return hex.EncodeToString(h[:])
}

// Reservation methods are an optional capability, deliberately separate from
// Adapter so unrelated upstreams cannot accidentally claim scarce support.
// They use the existing HMAC protocol and refuse redirects carrying its keys.
func (a *DujiaoNextAdapter) reservationRequest(ctx context.Context, method, path string, body any) (*paymentcontract.ReservationResult, error) {
	adapter := *a
	client := *a.client
	client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	adapter.client = &client
	var result paymentcontract.ReservationResult
	if err := adapter.doRequest(ctx, method, path, body, &result); err != nil {
		var remote *upstreamHTTPError
		if errors.As(err, &remote) {
			if remote.Status == http.StatusNotFound && remote.Code == "operation_not_found" {
				return nil, paymentcontract.ErrReservationNotFound
			}
			if remote.Status == http.StatusConflict {
				return nil, paymentcontract.ErrReservationConflict
			}
			if remote.Status == http.StatusBadRequest {
				return nil, paymentcontract.ErrReservationInvalid
			}
		}
		return nil, paymentcontract.ErrReservationUnavailable
	}
	if !result.OK {
		return nil, paymentcontract.ErrReservationUnavailable
	}
	return &result, nil
}

func (a *DujiaoNextAdapter) GetReservation(ctx context.Context, orderRef, operationID string) (*paymentcontract.ReservationResult, error) {
	q := url.Values{"order_ref": []string{orderRef}, "operation_id": []string{operationID}}
	return a.reservationRequest(ctx, http.MethodGet, "/api/v1/upstream/reservations?"+q.Encode(), nil)
}

func (a *DujiaoNextAdapter) ApplyReservation(ctx context.Context, r paymentcontract.ReservationRequest) (*paymentcontract.ReservationResult, error) {
	if r.PayloadHash != paymentcontract.ReservationDigest(r) {
		return nil, paymentcontract.ErrReservationInvalid
	}
	return a.reservationRequest(ctx, http.MethodPost, "/api/v1/upstream/reservations", r)
}

var _ paymentcontract.ReservationConnection = (*DujiaoNextAdapter)(nil)
