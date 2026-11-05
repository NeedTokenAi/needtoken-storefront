// Package plancheckout revalidates member API subscriptions before any payment
// transaction. A plan's frozen entitlement and route are independent of scarce
// inventory reservations.
package plancheckout

import (
	"crypto/sha256"
	"encoding/json"
	"sort"
	"strconv"
	"strings"

	mappingcontract "github.com/dujiao-next/internal/modules/catalog/mapping/contract"
	productcontract "github.com/dujiao-next/internal/modules/catalog/product/contract"
	ordercontract "github.com/dujiao-next/internal/modules/order/contract"
	orderdomain "github.com/dujiao-next/internal/modules/order/domain"
	"github.com/dujiao-next/internal/shared/jsonmap"
)

type Guard struct {
	orders   ordercontract.Store
	products productcontract.Repository
	skus     productcontract.SKURepository
	checker  mappingcontract.PlanCheckoutChecker
}

func New(orders ordercontract.Store, products productcontract.Repository, skus productcontract.SKURepository, checker mappingcontract.PlanCheckoutChecker) *Guard {
	return &Guard{orders: orders, products: products, skus: skus, checker: checker}
}

// Evidence binds the remote observation to the order tree read before payment.
// CheckLocked only compares persisted data; it never performs I/O under a lock.
type Evidence struct {
	digest [sha256.Size]byte
	active bool
}

func (e *Evidence) CheckLocked(root *orderdomain.Order) error {
	if e == nil || !e.active {
		return nil
	}
	digest, err := orderDigest(root)
	if err != nil || digest != e.digest {
		return mappingcontract.ErrPlanChanged
	}
	return nil
}

// Ensure must run before reservation acquisition, payment-link reuse, wallet
// debits and online payment creation. Missing historical plan metadata is never
// reconstructed from today's mutable SKU or mapping.
func (g *Guard) Ensure(orderID uint) (*Evidence, error) {
	root, err := g.orders.GetByID(orderID)
	if err != nil {
		return nil, err
	}
	if root == nil {
		// The existing payment order lookup supplies its established not-found
		// error; there is no plan to preflight in this case.
		return nil, nil
	}
	productPlans := make(map[uint]bool)
	skuPlans := make(map[uint]bool)
	var requests []mappingcontract.PlanCheckoutRequest
	var visit func(*orderdomain.Order) error
	visit = func(order *orderdomain.Order) error {
		for _, item := range order.Items {
			isPlan := frozenPlan(item.SKUSnapshotJSON)
			if !isPlan && g.products != nil && item.ProductID != 0 {
				value, known := productPlans[item.ProductID]
				if !known {
					product, fetchErr := g.products.GetByID(strconv.FormatUint(uint64(item.ProductID), 10))
					if fetchErr != nil {
						return mappingcontract.ErrPlanEligibilityUnavailable
					}
					value = product != nil && (product.MemberOnlyByCategory || strings.EqualFold(strings.TrimSpace(product.Category.Slug), "plans"))
					productPlans[item.ProductID] = value
				}
				isPlan = value
			}
			if !isPlan && g.skus != nil && item.SKUID != 0 {
				value, known := skuPlans[item.SKUID]
				if !known {
					sku, fetchErr := g.skus.GetByID(item.SKUID)
					if fetchErr != nil {
						return mappingcontract.ErrPlanEligibilityUnavailable
					}
					value = sku != nil && isPlanContract(sku.EntitlementContractJSON)
					skuPlans[item.SKUID] = value
				}
				isPlan = value
			}
			if !isPlan {
				continue
			}
			if root.UserID == 0 || order.UserID == 0 || order.UserID != root.UserID {
				return mappingcontract.ErrPlanAccountLinkRequired
			}
			request, requestErr := frozenRequest(item, root.UserID)
			if requestErr != nil {
				return requestErr
			}
			requests = append(requests, request)
		}
		for i := range order.Children {
			if err := visit(&order.Children[i]); err != nil {
				return err
			}
		}
		return nil
	}
	if err := visit(root); err != nil {
		return nil, err
	}
	if len(requests) == 0 {
		return nil, nil
	}
	if g.checker == nil {
		return nil, mappingcontract.ErrPlanEligibilityUnavailable
	}
	digest, err := orderDigest(root)
	if err != nil {
		return nil, mappingcontract.ErrPlanChanged
	}
	for _, request := range requests {
		if err := g.checker.CheckPlanCheckout(request); err != nil {
			return nil, err
		}
	}
	return &Evidence{digest: digest, active: true}, nil
}

func frozenPlan(snapshot jsonmap.JSON) bool {
	if value, ok := snapshot["api_plan"].(bool); ok && value {
		return true
	}
	if _, ok := snapshot["plan_checkout_connection_id"]; ok {
		return true
	}
	if _, ok := snapshot["plan_checkout_upstream_sku_id"]; ok {
		return true
	}
	if _, ok := snapshot["plan_checkout_entitlement_raw"]; ok {
		return true
	}
	return isPlanContract(snapshot["entitlement_contract"])
}

func isPlanContract(value any) bool {
	raw, err := json.Marshal(value)
	if err != nil {
		return false
	}
	var shape struct {
		Kind         string `json:"kind"`
		OfferingKind string `json:"offering_kind"`
	}
	return json.Unmarshal(raw, &shape) == nil && (strings.EqualFold(strings.TrimSpace(shape.Kind), "plan") || strings.EqualFold(strings.TrimSpace(shape.OfferingKind), "api_plan"))
}

func frozenRequest(item orderdomain.OrderItem, userID uint) (mappingcontract.PlanCheckoutRequest, error) {
	invalid := mappingcontract.ErrPlanChanged
	if item.ID == 0 || item.OrderID == 0 || item.ProductID == 0 || item.SKUID == 0 || item.Quantity < 1 {
		return mappingcontract.PlanCheckoutRequest{}, invalid
	}
	snapshot := item.SKUSnapshotJSON
	raw, err := frozenContract(snapshot)
	if err != nil {
		return mappingcontract.PlanCheckoutRequest{}, invalid
	}
	hash, ok := snapshot["entitlement_contract_hash"].(string)
	if !ok || hash == "" || hash != strings.TrimSpace(hash) {
		return mappingcontract.PlanCheckoutRequest{}, invalid
	}
	connectionID, validConnection := positiveSnapshotID(snapshot["plan_checkout_connection_id"])
	skuID, validSKU := positiveSnapshotID(snapshot["plan_checkout_upstream_sku_id"])
	if !validConnection || !validSKU {
		return mappingcontract.PlanCheckoutRequest{}, invalid
	}
	return mappingcontract.PlanCheckoutRequest{Route: mappingcontract.PlanCheckoutRoute{ConnectionID: connectionID, UpstreamSKUID: skuID}, UserID: userID, Quantity: item.Quantity, EntitlementContract: raw, EntitlementContractHash: hash}, nil
}

func frozenContract(snapshot jsonmap.JSON) (json.RawMessage, error) {
	var raw json.RawMessage
	if value, exists := snapshot["plan_checkout_entitlement_raw"]; exists {
		// New orders keep the complete original JSON as a string so a generic
		// order-snapshot decoder cannot round large integer quota values.
		exact, ok := value.(string)
		if !ok {
			return nil, mappingcontract.ErrPlanChanged
		}
		raw = json.RawMessage(exact)
	} else {
		// Historical snapshots are still sent in full. The upstream verifies
		// their frozen hash, so a lossy or changed value cannot become payable.
		encoded, err := json.Marshal(snapshot["entitlement_contract"])
		if err != nil {
			return nil, mappingcontract.ErrPlanChanged
		}
		raw = encoded
	}
	if len(raw) == 0 || raw[0] != '{' || !json.Valid(raw) || !isPlanContract(raw) {
		return nil, mappingcontract.ErrPlanChanged
	}
	return raw, nil
}

func positiveSnapshotID(value any) (uint, bool) {
	raw, ok := value.(string)
	if !ok || raw == "" || raw[0] == '0' {
		return 0, false
	}
	for _, digit := range raw {
		if digit < '0' || digit > '9' {
			return 0, false
		}
	}
	id, err := strconv.ParseUint(raw, 10, strconv.IntSize)
	return uint(id), err == nil && id != 0
}

func orderDigest(root *orderdomain.Order) ([sha256.Size]byte, error) {
	type itemBinding struct {
		ID, ProductID, SKUID uint
		Quantity             int
		Snapshot             jsonmap.JSON
	}
	type orderBinding struct {
		ID, UserID uint
		ParentID   *uint
		Items      []itemBinding
	}
	var bindings []orderBinding
	var visit func(*orderdomain.Order)
	visit = func(order *orderdomain.Order) {
		binding := orderBinding{ID: order.ID, UserID: order.UserID, ParentID: order.ParentID}
		for _, item := range order.Items {
			// A mixed cart may acquire an unrelated scarce-item hold between
			// preflight and payment. Bind plan terms, not reservation receipts.
			snapshot := make(jsonmap.JSON)
			for _, key := range []string{"api_plan", "entitlement_contract", "entitlement_contract_hash", "plan_checkout_entitlement_raw", "plan_checkout_connection_id", "plan_checkout_upstream_sku_id"} {
				if value, exists := item.SKUSnapshotJSON[key]; exists {
					snapshot[key] = value
				}
			}
			binding.Items = append(binding.Items, itemBinding{ID: item.ID, ProductID: item.ProductID, SKUID: item.SKUID, Quantity: item.Quantity, Snapshot: snapshot})
		}
		sort.Slice(binding.Items, func(i, j int) bool { return binding.Items[i].ID < binding.Items[j].ID })
		bindings = append(bindings, binding)
		for i := range order.Children {
			visit(&order.Children[i])
		}
	}
	if root == nil {
		return [sha256.Size]byte{}, mappingcontract.ErrPlanChanged
	}
	visit(root)
	sort.Slice(bindings, func(i, j int) bool { return bindings[i].ID < bindings[j].ID })
	raw, err := json.Marshal(bindings)
	if err != nil {
		return [sha256.Size]byte{}, err
	}
	return sha256.Sum256(raw), nil
}
