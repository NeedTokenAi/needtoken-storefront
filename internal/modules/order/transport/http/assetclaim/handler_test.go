package assetclaimhttp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dujiao-next/internal/modules/order/application/assetclaim"
	"github.com/gin-gonic/gin"
)

type fakeDir struct{ snap assetclaim.OrderSnapshot }

func (f fakeDir) Resolve(context.Context, assetclaim.Subject) (assetclaim.OrderSnapshot, error) {
	return f.snap, nil
}

type fakeBridge struct {
	err error
}

func (f *fakeBridge) Status(context.Context, uint, uint, string) (assetclaim.Status, error) {
	if f.err != nil {
		return assetclaim.Status{}, f.err
	}
	return assetclaim.Status{Eligible: true, AssetID: "av_1", RemainingReveals: 1, MaxReveals: 1}, nil
}
func (f *fakeBridge) Reveal(context.Context, uint, uint, string, string) (assetclaim.RevealResult, error) {
	if f.err != nil {
		return assetclaim.RevealResult{}, f.err
	}
	return assetclaim.RevealResult{
		Status:            assetclaim.Status{Eligible: true, AssetID: "av_1", RemainingReveals: 0, MaxReveals: 1, RevealCount: 1},
		AccountIdentifier: "user@example.com",
		SecretPayload:     "Pass#2026",
	}, nil
}

func TestAssetClaimHTTPHeadersAndExhaustedMapping(t *testing.T) {
	gin.SetMode(gin.TestMode)
	bridge := &fakeBridge{}
	h := New(assetclaim.New(fakeDir{snap: assetclaim.OrderSnapshot{OrderID: 1, OrderNo: "ORD-1", UserID: 5, Leaf: true, Delivered: true, ConnectionID: 2, UpstreamOrderID: 3}}, bridge))
	r := gin.New()
	auth := r.Group("", func(c *gin.Context) {
		c.Set("user_id", uint(5))
		c.Next()
	})
	RegisterMemberReadRoutes(auth, h)
	RegisterMemberWriteRoutes(auth, h)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/orders/ORD-1/asset-claim", nil))
	if w.Code != http.StatusOK || w.Header().Get("Cache-Control") != "no-store" || w.Header().Get("Referrer-Policy") != "no-referrer" {
		t.Fatalf("status=%d headers=%v", w.Code, w.Header())
	}

	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/orders/ORD-1/asset-claim/reveal", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("reveal status=%d body=%s", w.Code, w.Body.String())
	}

	bridge.err = assetclaim.ErrExhausted
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/orders/ORD-1/asset-claim/reveal", nil))
	var env struct {
		StatusCode int `json:"status_code"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &env)
	if env.StatusCode == 0 {
		t.Fatalf("exhausted reveal must return non-zero business status_code: %s", w.Body.String())
	}
}
