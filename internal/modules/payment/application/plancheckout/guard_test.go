package plancheckout

import (
	"encoding/json"
	"errors"
	"testing"

	mappingcontract "github.com/dujiao-next/internal/modules/catalog/mapping/contract"
	productcontract "github.com/dujiao-next/internal/modules/catalog/product/contract"
	productdomain "github.com/dujiao-next/internal/modules/catalog/product/domain"
	ordercontract "github.com/dujiao-next/internal/modules/order/contract"
	orderdomain "github.com/dujiao-next/internal/modules/order/domain"
	"github.com/dujiao-next/internal/shared/jsonmap"
)

type orderReader struct {
	ordercontract.Store
	root *orderdomain.Order
}

func (r orderReader) GetByID(uint) (*orderdomain.Order, error) { return r.root, nil }

type productReader struct {
	productcontract.Repository
	plan bool
}

func (r productReader) GetByID(string) (*productdomain.Product, error) {
	return &productdomain.Product{MemberOnlyByCategory: r.plan}, nil
}

type skuReader struct {
	productcontract.SKURepository
}

func (skuReader) GetByID(uint) (*productdomain.ProductSKU, error) {
	return &productdomain.ProductSKU{EntitlementContractJSON: jsonmap.ExactJSON{"kind": "plan", "updated": true}}, nil
}

type checker struct {
	requests []mappingcontract.PlanCheckoutRequest
	err      error
}

func (c *checker) CheckPlanCheckout(request mappingcontract.PlanCheckoutRequest) error {
	c.requests = append(c.requests, request)
	return c.err
}

func planItem(id, orderID uint) orderdomain.OrderItem {
	return orderdomain.OrderItem{ID: id, OrderID: orderID, ProductID: 7, SKUID: 8, Quantity: 2, SKUSnapshotJSON: jsonmap.JSON{
		"api_plan":                      true,
		"entitlement_contract":          jsonmap.JSON{"kind": "plan", "unknown": jsonmap.JSON{"quota": "9007199254740993"}},
		"plan_checkout_entitlement_raw": `{"kind":"plan","unknown":{"quota":9007199254740993}}`,
		"entitlement_contract_hash":     "frozen-hash",
		"plan_checkout_connection_id":   "11", "plan_checkout_upstream_sku_id": "22",
		// These unrelated reservation values must never select a plan route.
		"reservation_connection_id": "31", "reservation_upstream_sku_id": "32",
	}}
}

func TestPlanPaymentChecksEveryRootAndChildFrozenItem(t *testing.T) {
	parent := uint(1)
	root := &orderdomain.Order{ID: 1, UserID: 99, Items: []orderdomain.OrderItem{planItem(1, 1)}, Children: []orderdomain.Order{{ID: 2, ParentID: &parent, UserID: 99, Items: []orderdomain.OrderItem{planItem(2, 2)}}}}
	remote := &checker{}
	evidence, err := New(orderReader{root: root}, nil, skuReader{}, remote).Ensure(1)
	if err != nil || len(remote.requests) != 2 {
		t.Fatalf("root/child plan skipped: %v (%d requests)", err, len(remote.requests))
	}
	for _, request := range remote.requests {
		if request.Route.ConnectionID != 11 || request.Route.UpstreamSKUID != 22 || request.UserID != 99 || request.Quantity != 2 || request.EntitlementContractHash != "frozen-hash" || string(request.EntitlementContract) != root.Items[0].SKUSnapshotJSON["plan_checkout_entitlement_raw"] {
			t.Fatalf("mutable/reservation data replaced the frozen request: %+v", request)
		}
		var contract struct {
			Unknown struct {
				Quota json.Number `json:"quota"`
			} `json:"unknown"`
		}
		if err := json.Unmarshal(request.EntitlementContract, &contract); err != nil || contract.Unknown.Quota.String() != "9007199254740993" {
			t.Fatal("raw entitlement precision lost", err)
		}
	}
	if err := evidence.CheckLocked(root); err != nil {
		t.Fatal("unchanged order did not match preflight", err)
	}
	root.Items[0].SKUSnapshotJSON["reservation_hold"] = jsonmap.JSON{"id": "unrelated"}
	if err := evidence.CheckLocked(root); err != nil {
		t.Fatal("a mixed-cart reservation invalidated API eligibility", err)
	}
	root.Children[0].Items[0].SKUSnapshotJSON["plan_checkout_upstream_sku_id"] = "23"
	if err := evidence.CheckLocked(root); !errors.Is(err, mappingcontract.ErrPlanChanged) {
		t.Fatal("changed locked route reused preflight", err)
	}
}

func TestPlanPaymentFailsClosedForLegacyUnpaidOrders(t *testing.T) {
	for _, source := range []string{"category", "sku", "frozen-contract", "frozen-flag", "malformed-route"} {
		t.Run(source, func(t *testing.T) {
			item := planItem(1, 1)
			item.SKUSnapshotJSON = nil
			var products productcontract.Repository
			var skus productcontract.SKURepository
			switch source {
			case "category":
				products = productReader{plan: true}
			case "sku":
				skus = skuReader{}
			case "frozen-contract":
				item.SKUSnapshotJSON = jsonmap.JSON{"entitlement_contract": jsonmap.JSON{"kind": "plan"}}
			case "frozen-flag":
				item.SKUSnapshotJSON = jsonmap.JSON{"api_plan": true}
			case "malformed-route":
				item.SKUSnapshotJSON = jsonmap.JSON{"plan_checkout_connection_id": 11}
			}
			remote := &checker{}
			root := &orderdomain.Order{ID: 1, UserID: 99, Items: []orderdomain.OrderItem{item}}
			if _, err := New(orderReader{root: root}, products, skus, remote).Ensure(1); !errors.Is(err, mappingcontract.ErrPlanChanged) || len(remote.requests) != 0 {
				t.Fatalf("legacy plan reconstructed today's terms: %v", err)
			}
		})
	}
}

func TestPlanPaymentRequiresNativeMemberAndAuthoritativeEligibility(t *testing.T) {
	for _, test := range []struct {
		name    string
		userID  uint
		remote  error
		missing bool
		want    error
	}{
		{"guest", 0, nil, false, mappingcontract.ErrPlanAccountLinkRequired},
		{"missing-checker", 99, nil, true, mappingcontract.ErrPlanEligibilityUnavailable},
		{"account-unavailable", 99, mappingcontract.ErrPlanAccountUnavailable, false, mappingcontract.ErrPlanAccountUnavailable},
		{"plan-changed", 99, mappingcontract.ErrPlanChanged, false, mappingcontract.ErrPlanChanged},
		{"token-limit", 99, mappingcontract.ErrPlanTokenLimitReached, false, mappingcontract.ErrPlanTokenLimitReached},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := &orderdomain.Order{ID: 1, UserID: test.userID, GuestEmail: "not-an-identity@example.test", Items: []orderdomain.OrderItem{planItem(1, 1)}}
			var remote mappingcontract.PlanCheckoutChecker = &checker{err: test.remote}
			if test.missing {
				remote = nil
			}
			if _, err := New(orderReader{root: root}, nil, nil, remote).Ensure(1); !errors.Is(err, test.want) {
				t.Fatalf("got %v, want %v", err, test.want)
			}
		})
	}
}

func TestOrdinaryProductPaymentDoesNotRequirePlanChecker(t *testing.T) {
	root := &orderdomain.Order{ID: 1, Items: []orderdomain.OrderItem{{ID: 1, ProductID: 7, SKUID: 8, Quantity: 1}}}
	evidence, err := New(orderReader{root: root}, productReader{}, nil, nil).Ensure(1)
	if err != nil || evidence != nil || evidence.CheckLocked(root) != nil {
		t.Fatalf("ordinary guest order was changed by API-plan guard: %v", err)
	}
}
