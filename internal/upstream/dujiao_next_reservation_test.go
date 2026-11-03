package upstream

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	paymentcontract "github.com/dujiao-next/internal/modules/payment/contract"
	siteconnectiondomain "github.com/dujiao-next/internal/modules/siteconnection/domain"
)

func TestReservationOnlyExplicitOperationNotFoundPermitsPosting(t *testing.T) {
	for _, code := range []string{"operation_not_found", "reservation_not_found", "route_not_found", ""} {
		t.Run(code, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(404)
				_ = json.NewEncoder(w).Encode(map[string]any{"ok": false, "error_code": code})
			}))
			defer server.Close()
			a := NewDujiaoNextAdapter(&siteconnectiondomain.Connection{BaseURL: server.URL, ApiKey: "key", ApiSecret: "secret"}, "")
			_, err := a.GetReservation(context.Background(), "order_ref_01", "operation_01")
			if errors.Is(err, paymentcontract.ErrReservationNotFound) != (code == "operation_not_found") {
				t.Fatalf("ambiguous 404 treated as safe retry: %s %v", code, err)
			}
		})
	}
}

func TestReservationRedirectNeverForwardsSigningCredentials(t *testing.T) {
	calls := 0
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls++; w.WriteHeader(200) }))
	defer target.Close()
	origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target.URL, http.StatusTemporaryRedirect)
	}))
	defer origin.Close()
	a := NewDujiaoNextAdapter(&siteconnectiondomain.Connection{BaseURL: origin.URL, ApiKey: "scoped-key", ApiSecret: "secret"}, "")
	if _, err := a.GetReservation(context.Background(), "order_ref_01", "operation_01"); !errors.Is(err, paymentcontract.ErrReservationUnavailable) {
		t.Fatal(err)
	}
	if calls != 0 {
		t.Fatal("reservation credentials followed a redirect")
	}
}

func TestReservationMismatchedDigestIsRejectedBeforeHTTP(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls++ }))
	defer server.Close()
	a := NewDujiaoNextAdapter(&siteconnectiondomain.Connection{BaseURL: server.URL, ApiKey: "key", ApiSecret: "secret"}, "")
	if _, err := a.ApplyReservation(context.Background(), paymentcontract.ReservationRequest{OperationID: "operation_01", PayloadHash: "incorrect"}); !errors.Is(err, paymentcontract.ErrReservationInvalid) {
		t.Fatal(err)
	}
	if calls != 0 {
		t.Fatal("invalid immutable request reached upstream")
	}
}
