package application

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/dujiao-next/internal/constants"
	mappingcontract "github.com/dujiao-next/internal/modules/catalog/mapping/contract"
	mappingdomain "github.com/dujiao-next/internal/modules/catalog/mapping/domain"
	productdomain "github.com/dujiao-next/internal/modules/catalog/product/domain"
	siteconnectiondomain "github.com/dujiao-next/internal/modules/siteconnection/domain"
	"github.com/dujiao-next/internal/upstream"
)

type recordingMediaRecorder struct {
	ctx       context.Context
	localPath string
	scene     string
}

type planCheckoutAdapter struct {
	upstream.Adapter
	request upstream.PlanCheckoutEligibilityRequest
	reason  string
	calls   int
}

func (a *planCheckoutAdapter) CheckPlanCheckoutEligibility(_ context.Context, request upstream.PlanCheckoutEligibilityRequest) (*upstream.PlanCheckoutEligibilityResult, error) {
	a.calls++
	a.request = request
	return &upstream.PlanCheckoutEligibilityResult{OK: true, Ready: a.reason == "ready", ReasonCode: a.reason, SKUID: request.SKUID, EntitlementContractHash: request.EntitlementContractHash, CollectedAt: time.Now().UTC()}, nil
}

type planCheckoutConnections struct{ adapter upstream.Adapter }

func (c planCheckoutConnections) GetByID(id uint) (*siteconnectiondomain.Connection, error) {
	return &siteconnectiondomain.Connection{ID: id, Status: constants.ConnectionStatusActive}, nil
}
func (c planCheckoutConnections) GetAdapter(*siteconnectiondomain.Connection) (upstream.Adapter, error) {
	return c.adapter, nil
}

type planCheckoutSKUMappings struct {
	mappingcontract.SKUMappingRepository
	row *mappingdomain.SKUMapping
}

func (r planCheckoutSKUMappings) GetByLocalSKUID(uint) (*mappingdomain.SKUMapping, error) {
	return r.row, nil
}

func TestPlanCheckoutUsesActualMappingAndAuthenticatedMemberReference(t *testing.T) {
	adapter := &planCheckoutAdapter{reason: "ready"}
	mapping := &mappingdomain.Mapping{ID: 7, ConnectionID: 77, LocalProductID: 11, UpstreamProductID: 22, IsActive: true, UpstreamStatus: mappingdomain.UpstreamStatusActive}
	skuMap := &mappingdomain.SKUMapping{ProductMappingID: 7, LocalSKUID: 111, UpstreamSKUID: 901, UpstreamIsActive: true}
	svc := &Service{connections: planCheckoutConnections{adapter}, mappings: &reservationMappingRepo{row: mapping}, skuMappings: planCheckoutSKUMappings{row: skuMap}, skus: &reservationSKURepo{rows: map[uint]*productdomain.ProductSKU{111: {ID: 111, ProductID: 11, IsActive: true}}}}
	raw := json.RawMessage(`{"kind":"plan","issue_api_key":true,"plan":{"quota":9007199254740993},"future_field":{"preserve":true}}`)
	hash := strings.Repeat("a", 64)
	route, err := svc.PreparePlanCheckout(111, 19, 1, raw, hash)
	if err != nil || route.ConnectionID != 77 || route.UpstreamSKUID != 901 {
		t.Fatalf("actual route not frozen: %#v %v", route, err)
	}
	got := adapter.request
	if got.SKUID != 901 || string(got.EntitlementContract) != string(raw) || got.CustomerReference.MerchantCustomerID != "needtoken-storefront:user:19" || got.CustomerReference.Source != "needtoken-storefront" || got.CustomerReference.Verification != "merchant_session" {
		t.Fatalf("preflight changed immutable terms or identity: %#v", got)
	}
	for reason, want := range map[string]error{"account_link_required": mappingcontract.ErrPlanAccountLinkRequired, "account_unavailable": mappingcontract.ErrPlanAccountUnavailable, "plan_unavailable": mappingcontract.ErrPlanUnavailable, "plan_changed": mappingcontract.ErrPlanChanged, "purchase_limit_reached": mappingcontract.ErrPlanPurchaseLimitReached, "token_limit_reached": mappingcontract.ErrPlanTokenLimitReached, "unknown": mappingcontract.ErrPlanEligibilityUnavailable} {
		adapter.reason = reason
		if _, err := svc.PreparePlanCheckout(111, 19, 1, raw, hash); !errors.Is(err, want) {
			t.Fatalf("%s mapped to %v", reason, err)
		}
	}
	before := adapter.calls
	for _, request := range []mappingcontract.PlanCheckoutRequest{
		{Route: route, UserID: 0, Quantity: 1, EntitlementContract: raw, EntitlementContractHash: hash},
		{Route: route, UserID: 19, Quantity: 2, EntitlementContract: raw, EntitlementContractHash: hash},
		{Route: route, UserID: 19, Quantity: 1, EntitlementContract: json.RawMessage(`{"kind":"plan"}`), EntitlementContractHash: hash},
	} {
		if err := svc.CheckPlanCheckout(request); err == nil {
			t.Fatal("invalid plan checkout accepted")
		}
	}
	if adapter.calls != before {
		t.Fatal("invalid member/quantity/legacy terms reached upstream")
	}
	mapping.IsActive = false
	if _, err := svc.PreparePlanCheckout(111, 19, 1, raw, hash); !errors.Is(err, mappingcontract.ErrPlanUnavailable) {
		t.Fatal(err)
	}
	if adapter.calls != before {
		t.Fatal("inactive mapping reached upstream")
	}
}

func (r *recordingMediaRecorder) RecordLocalFile(ctx context.Context, localPath, scene string) {
	r.ctx = ctx
	r.localPath = localPath
	r.scene = scene
}

func TestNewServiceRejectsNilMediaRecorder(t *testing.T) {
	if _, err := NewService(Options{}); !errors.Is(err, mappingcontract.ErrMediaRecorderRequired) {
		t.Fatalf("error = %v, want mappingcontract.ErrMediaRecorderRequired", err)
	}
}

func TestRecordUpstreamMediaPropagatesCallerContext(t *testing.T) {
	type contextKey struct{}
	ctx := context.WithValue(context.Background(), contextKey{}, "mapping-request")
	recorder := &recordingMediaRecorder{}
	service := &Service{media: recorder}

	service.recordUpstreamMedia(ctx, "/uploads/upstream/image.png")

	if recorder.ctx.Value(contextKey{}) != "mapping-request" {
		t.Fatal("caller context was not propagated")
	}
	if recorder.localPath != "/uploads/upstream/image.png" || recorder.scene != "upstream" {
		t.Fatalf("recorded media = (%q, %q)", recorder.localPath, recorder.scene)
	}
}
