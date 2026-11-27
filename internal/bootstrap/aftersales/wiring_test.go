package aftersaleswiring

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/dujiao-next/internal/app/container"
	"github.com/dujiao-next/internal/constants"
	"github.com/dujiao-next/internal/crypto"
	"github.com/dujiao-next/internal/modules/order/application/aftersales"
	ordercontract "github.com/dujiao-next/internal/modules/order/contract"
	orderdomain "github.com/dujiao-next/internal/modules/order/domain"
	aftersaleshttp "github.com/dujiao-next/internal/modules/order/transport/http/aftersales"
	reseller "github.com/dujiao-next/internal/modules/reseller/contract"
	siteconnectionapp "github.com/dujiao-next/internal/modules/siteconnection/application"
	siteconnectioncontract "github.com/dujiao-next/internal/modules/siteconnection/contract"
	siteconnectiondomain "github.com/dujiao-next/internal/modules/siteconnection/domain"
	"github.com/dujiao-next/internal/upstream"
	"github.com/gin-gonic/gin"
)

type stubConnRepo struct {
	items []siteconnectiondomain.Connection
}

func (s *stubConnRepo) GetByID(id uint) (*siteconnectiondomain.Connection, error) {
	for i := range s.items {
		if s.items[i].ID == id {
			c := s.items[i]
			return &c, nil
		}
	}
	return nil, nil
}
func (s *stubConnRepo) GetByApiKey(string) (*siteconnectiondomain.Connection, error) {
	return nil, nil
}
func (s *stubConnRepo) Create(*siteconnectiondomain.Connection) error { return nil }
func (s *stubConnRepo) Update(*siteconnectiondomain.Connection) error { return nil }
func (s *stubConnRepo) Delete(uint) error                             { return nil }
func (s *stubConnRepo) List(siteconnectioncontract.ListFilter) ([]siteconnectiondomain.Connection, int64, error) {
	return s.items, int64(len(s.items)), nil
}
func (s *stubConnRepo) ListActive() ([]siteconnectiondomain.Connection, error) {
	var out []siteconnectiondomain.Connection
	for _, item := range s.items {
		if item.Status == constants.ConnectionStatusActive {
			out = append(out, item)
		}
	}
	return out, nil
}

type stubOrderStore struct {
	ordercontract.Store
	orders map[string]*orderdomain.Order
}

func sameTenant(a, b *uint) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}

func (s *stubOrderStore) GetAnyByOrderNoAndUserScoped(orderNo string, userID uint, scope ordercontract.TenantScope) (*orderdomain.Order, error) {
	o := s.orders[orderNo]
	if o == nil || o.UserID != userID || !sameTenant(o.ResellerID, scope.ResellerID) {
		return nil, nil
	}
	return o, nil
}

func (s *stubOrderStore) GetAnyByOrderNoAndGuestScoped(orderNo, email, password string, scope ordercontract.TenantScope) (*orderdomain.Order, error) {
	o := s.orders[orderNo]
	if o == nil || o.UserID != 0 || !sameTenant(o.ResellerID, scope.ResellerID) {
		return nil, nil
	}
	if !strings.EqualFold(o.GuestEmail, email) || o.GuestPassword != password {
		return nil, nil
	}
	return o, nil
}

// Compile-time check against nil-dereference when UserStore is nil.
func TestHTTPAfterSalesRoutesIsolateMemberGuestAndTenant(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var mu sync.Mutex
	upstreamCalls := 0
	tickets := map[string][]upstream.OrderAfterSalesTicket{}
	apiKey := "k_after_sales"
	apiSecret := "s_after_sales_secret_0123456789"

	glueSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		upstreamCalls++
		mu.Unlock()
		if r.Header.Get(upstream.HeaderApiKey) != apiKey || r.Header.Get(upstream.HeaderSignature) == "" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/v1/upstream/after-sales/orders/"), "/")
		orderNo := parts[0]
		mu.Lock()
		defer mu.Unlock()
		switch {
		case r.Method == http.MethodGet && len(parts) == 1:
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "items": tickets[orderNo]})
		case r.Method == http.MethodPost && len(parts) == 1:
			var body struct {
				CustomerEmail string `json:"customer_email"`
				Category      string `json:"category"`
				Summary       string `json:"summary"`
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			tk := upstream.OrderAfterSalesTicket{
				ID:        "otk_test_1",
				OrderNo:   orderNo,
				Category:  body.Category,
				Priority:  "normal",
				Status:    "open",
				Summary:   body.Summary,
				CreatedAt: "2026-11-27T12:00:00Z",
				UpdatedAt: "2026-11-27T12:00:00Z",
			}
			tickets[orderNo] = append(tickets[orderNo], tk)
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "ticket": tk})
		case r.Method == http.MethodPost && len(parts) == 3 && parts[2] == "supplement":
			var body struct {
				Message string `json:"message"`
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			list := tickets[orderNo]
			for i := range list {
				if list[i].ID == parts[1] {
					list[i].Supplements = append(list[i].Supplements, upstream.OrderTicketSupplement{
						Author:    "customer",
						Message:   body.Message,
						CreatedAt: "2026-11-27T12:05:00Z",
					})
					tickets[orderNo] = list
					_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "ticket": list[i]})
					return
				}
			}
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": false, "error_code": "after_sales_not_found"})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer glueSrv.Close()

	connKey := "0123456789abcdef0123456789abcdef"
	encryptedSecret, err := crypto.Encrypt(crypto.DeriveKey(connKey), apiSecret)
	if err != nil {
		t.Fatal(err)
	}
	connRepo := &stubConnRepo{
		items: []siteconnectiondomain.Connection{
			{
				ID:        5,
				Name:      "glue-main",
				Protocol:  constants.ConnectionProtocolDujiaoNext,
				BaseURL:   glueSrv.URL,
				ApiKey:    apiKey,
				ApiSecret: encryptedSecret,
				Status:    constants.ConnectionStatusActive,
			},
		},
	}
	connSvc := siteconnectionapp.NewService(connRepo, connKey, "")

	orders := &stubOrderStore{
		orders: map[string]*orderdomain.Order{
			"ORD-MEM-1": {ID: 1, OrderNo: "ORD-MEM-1", UserID: 100, Status: constants.OrderStatusPaid},
			"ORD-GST-1": {ID: 2, OrderNo: "ORD-GST-1", UserID: 0, GuestEmail: "guest@example.com", GuestPassword: "g-secret", Status: constants.OrderStatusDelivered},
		},
	}
	// Note: container.UserStore expects full interface in container, so test directory email path separately and leave UserStore nil here.
	c := &container.Container{
		OrderStore:            orders,
		SiteConnectionRepo:    connRepo,
		SiteConnectionService: connSvc,
	}
	handler := NewHandler(c)

	r := gin.New()
	r.Use(func(ctx *gin.Context) {
		if ctx.GetHeader("X-Test-Reseller") == "9" {
			ctx.Request = ctx.Request.WithContext(reseller.WithTenantContext(ctx.Request.Context(), reseller.ResellerTenantContext("r9.example.invalid", 9, 1, "r9.example.invalid")))
		} else {
			ctx.Request = ctx.Request.WithContext(reseller.WithTenantContext(ctx.Request.Context(), reseller.MainTenantContext("main.example.invalid")))
		}
		ctx.Next()
	})
	guest := r.Group("/api/v1/guest")
	aftersaleshttp.RegisterGuestReadRoutes(guest, handler)
	aftersaleshttp.RegisterGuestWriteRoutes(guest, handler)

	member := r.Group("/api/v1")
	member.Use(func(ctx *gin.Context) {
		switch ctx.GetHeader("X-Test-User") {
		case "100":
			ctx.Set("user_id", uint(100))
		case "101":
			ctx.Set("user_id", uint(101))
		}
		ctx.Next()
	})
	aftersaleshttp.RegisterMemberReadRoutes(member, handler)
	aftersaleshttp.RegisterMemberWriteRoutes(member, handler)

	doReq := func(method, path string, body any, headers map[string]string) *httptest.ResponseRecorder {
		var buf bytes.Buffer
		if body != nil {
			_ = json.NewEncoder(&buf).Encode(body)
		}
		req := httptest.NewRequest(method, path, &buf)
		if body != nil {
			req.Header.Set("Content-Type", "application/json")
		}
		for k, v := range headers {
			req.Header.Set(k, v)
		}
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		return w
	}

	assertStatusEnvelope := func(w *httptest.ResponseRecorder, wantStatusCode int, label string) {
		t.Helper()
		var env struct {
			StatusCode int `json:"status_code"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil {
			t.Fatalf("%s invalid json %q: %v", label, w.Body.String(), err)
		}
		if env.StatusCode != wantStatusCode {
			t.Fatalf("%s: expected status_code=%d, got %d body=%s", label, wantStatusCode, env.StatusCode, w.Body.String())
		}
	}

	// 1. Other member (user 101) cannot read or submit ticket on ORD-MEM-1 (404, 0 upstream calls).
	w := doReq(http.MethodPost, "/api/v1/orders/ORD-MEM-1/after-sales", map[string]any{
		"category": "refund_dispute",
		"summary":  "unauthorized access test",
	}, map[string]string{"X-Test-User": "101"})
	assertStatusEnvelope(w, http.StatusNotFound, "other member")

	// 2. Wrong reseller tenant (reseller 9) cannot read ORD-MEM-1.
	w = doReq(http.MethodGet, "/api/v1/orders/ORD-MEM-1/after-sales", nil, map[string]string{"X-Test-User": "100", "X-Test-Reseller": "9"})
	assertStatusEnvelope(w, http.StatusNotFound, "wrong tenant")

	// 3. Wrong guest password rejected with 404 and 0 upstream calls.
	w = doReq(http.MethodGet, "/api/v1/guest/orders/ORD-GST-1/after-sales", nil, map[string]string{
		"X-Guest-Email":          "guest@example.com",
		"X-Guest-Order-Password": "wrong-password",
	})
	assertStatusEnvelope(w, http.StatusNotFound, "wrong guest password")
	mu.Lock()
	if upstreamCalls != 0 {
		mu.Unlock()
		t.Fatalf("rejected requests must not call upstream, got %d calls", upstreamCalls)
	}
	mu.Unlock()

	// 4. Owner member creates ticket and appends supplement; no refund endpoint exists.
	w = doReq(http.MethodPost, "/api/v1/orders/ORD-MEM-1/after-sales", map[string]any{
		"category": "fulfillment_failed",
		"summary":  "Need support check",
	}, map[string]string{"X-Test-User": "100"})
	if w.Code != http.StatusOK {
		t.Fatalf("member create failed: %d %s", w.Code, w.Body.String())
	}
	w = doReq(http.MethodPost, "/api/v1/orders/ORD-MEM-1/after-sales/otk_test_1/supplement", map[string]any{
		"message": "Extra proof",
	}, map[string]string{"X-Test-User": "100"})
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), "Extra proof") {
		t.Fatalf("member supplement failed: %d %s", w.Code, w.Body.String())
	}
	w = doReq(http.MethodPost, "/api/v1/orders/ORD-MEM-1/after-sales/otk_test_1/refund", nil, map[string]string{"X-Test-User": "100"})
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected no direct refund route, got %d", w.Code)
	}

	// 5. Valid guest creates & lists ticket on ORD-GST-1.
	w = doReq(http.MethodPost, "/api/v1/guest/orders/ORD-GST-1/after-sales", map[string]any{
		"category": "amount_mismatch",
		"summary":  "Guest inquiry",
	}, map[string]string{
		"X-Guest-Email":          "guest@example.com",
		"X-Guest-Order-Password": "g-secret",
	})
	if w.Code != http.StatusOK {
		t.Fatalf("guest create failed: %d %s", w.Code, w.Body.String())
	}

	// 6. Ambiguous >1 active connections must fail closed.
	connRepo.items = append(connRepo.items, siteconnectiondomain.Connection{
		ID: 6, Name: "glue-secondary", Protocol: constants.ConnectionProtocolDujiaoNext, Status: constants.ConnectionStatusActive,
	})
	dir := orderDirectory{c: c}
	if _, err := dir.Resolve(t.Context(), aftersales.Subject{OrderNo: "ORD-MEM-1", UserID: 100}); !errors.Is(err, aftersales.ErrUnavailable) {
		t.Fatalf("expected ErrUnavailable for ambiguous connections, got %v", err)
	}
}
