package contract

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"time"

	orderdomain "github.com/dujiao-next/internal/modules/order/domain"
	paymentdomain "github.com/dujiao-next/internal/modules/payment/domain"
	"github.com/dujiao-next/internal/shared/reservationterms"
)

var (
	ErrReservationUnavailable   = errors.New("checkout reservation unavailable")
	ErrReservationAttention     = errors.New("checkout reservation requires review")
	ErrReservationInvalid       = errors.New("checkout reservation snapshot invalid")
	ErrReservationNotFound      = errors.New("checkout reservation operation not found")
	ErrReservationConflict      = errors.New("checkout reservation content conflict")
	ErrReservationPolicyUnknown = errors.New("upstream checkout reservation policy is undeclared")
)

// ReservationSnapshot and ReservationRequest use the exact signed wire order.
// All monetary amounts and versions stay canonical decimal strings.
type ReservationSnapshot struct {
	SKU           string `json:"sku"`
	Region        string `json:"region"`
	Currency      string `json:"currency"`
	RequiredMinor string `json:"required_minor"`
	ExpiresAt     string `json:"expires_at"`
}

type ReservationRequest struct {
	OperationID     string              `json:"operation_id"`
	Action          string              `json:"action"`
	OrderRef        string              `json:"order_ref"`
	Snapshot        ReservationSnapshot `json:"snapshot"`
	ExpectedVersion string              `json:"expected_version"`
	EvidenceRef     string              `json:"evidence_ref,omitempty"`
	ReleaseReason   string              `json:"release_reason,omitempty"`
	PayloadHash     string              `json:"payload_hash"`
}

func ReservationDigest(r ReservationRequest) string {
	r.PayloadHash = ""
	b, _ := json.Marshal(r)
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}

type ReservationResult struct {
	OK                bool                `json:"ok"`
	OperationID       string              `json:"operation_id,omitempty"`
	PayloadHash       string              `json:"payload_hash,omitempty"`
	OrderRef          string              `json:"order_ref"`
	HoldID            string              `json:"hold_id"`
	Version           string              `json:"version"`
	Status            string              `json:"status"`
	BookStatus        string              `json:"book_status"`
	ReadyForPayment   bool                `json:"ready_for_payment"`
	EntitlementStatus string              `json:"entitlement_status"`
	BookOnly          bool                `json:"book_only"`
	Snapshot          ReservationSnapshot `json:"snapshot"`
	CreatedAt         time.Time           `json:"created_at"`
	UpdatedAt         time.Time           `json:"updated_at"`
	PaidAt            *time.Time          `json:"paid_at,omitempty"`
}

type ReservationConnection interface {
	ReservationIdentity() string
	GetReservation(context.Context, string, string) (*ReservationResult, error)
	ApplyReservation(context.Context, ReservationRequest) (*ReservationResult, error)
}
type ReservationProvider interface {
	OpenReservation(uint) (ReservationConnection, error)
}
type CheckoutReservationGate interface {
	EnsureBeforePayment(context.Context, uint) error
	ConfirmPaidOrder(context.Context, uint) error
}

type CheckoutRequirement struct {
	RootOrderID    uint
	RootOrderNo    string
	OrderID        uint
	ItemID         uint
	ConnectionID   uint
	UpstreamSKUID  uint
	Terms          reservationterms.Terms
	OrderExpiresAt *time.Time
}

func (r CheckoutRequirement) Hash() string {
	b, _ := json.Marshal(r)
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}

// CheckoutRequirements reads only the immutable server SKU snapshot. Buyer
// form fields never enter this path. Incomplete scarce metadata fails closed.
func CheckoutRequirements(root *orderdomain.Order) ([]CheckoutRequirement, error) {
	if root == nil {
		return nil, ErrReservationInvalid
	}
	out := []CheckoutRequirement{}
	var visit func(*orderdomain.Order) error
	visit = func(o *orderdomain.Order) error {
		for _, i := range o.Items {
			snapshot := i.SKUSnapshotJSON
			required := false
			if raw, exists := snapshot["requires_reservation"]; exists {
				v, ok := raw.(bool)
				if !ok {
					return ErrReservationInvalid
				}
				required = v
			}
			if raw, exists := snapshot["reservation_contract"]; exists && raw != nil {
				required = true
			}
			if !required {
				continue
			}
			terms, err := reservationterms.Parse(snapshot["reservation_contract"])
			if err != nil || i.Quantity != 1 || i.ID == 0 || i.OrderID == 0 {
				return ErrReservationInvalid
			}
			connectionID, err := snapshotUint(snapshot["reservation_connection_id"])
			if err != nil {
				return err
			}
			skuID, err := snapshotUint(snapshot["reservation_upstream_sku_id"])
			if err != nil {
				return err
			}
			out = append(out, CheckoutRequirement{RootOrderID: root.ID, RootOrderNo: root.OrderNo, OrderID: i.OrderID, ItemID: i.ID, ConnectionID: connectionID, UpstreamSKUID: skuID, Terms: terms, OrderExpiresAt: root.ExpiresAt})
		}
		for n := range o.Children {
			if err := visit(&o.Children[n]); err != nil {
				return err
			}
		}
		return nil
	}
	if err := visit(root); err != nil {
		return nil, err
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ItemID < out[j].ItemID })
	return out, nil
}

func snapshotUint(raw any) (uint, error) {
	s, ok := raw.(string)
	if !ok || s == "" || s[0] == '0' {
		return 0, ErrReservationInvalid
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, ErrReservationInvalid
		}
	}
	v, e := strconv.ParseUint(s, 10, strconv.IntSize)
	if e != nil || v == 0 {
		return 0, ErrReservationInvalid
	}
	return uint(v), nil
}

// ValidateCheckoutReservations repeats the persisted promise check while the
// payment/order lock is held, before any wallet debit or payment is created.
func ValidateCheckoutReservations(root *orderdomain.Order, rows []paymentdomain.CheckoutReservation, now time.Time) error {
	reqs, err := CheckoutRequirements(root)
	if err != nil {
		return err
	}
	byItem := map[uint]paymentdomain.CheckoutReservation{}
	for _, r := range rows {
		byItem[r.OrderItemID] = r
	}
	for _, req := range reqs {
		r, ok := byItem[req.ItemID]
		if !ok || r.RequirementHash != req.Hash() || r.RootOrderID != root.ID || r.ConnectionID != req.ConnectionID || r.UpstreamSKUID != req.UpstreamSKUID {
			return ErrReservationAttention
		}
		if r.State != "reserved" || r.BookStatus != "reserved" || !r.ReadyForPayment || r.HoldID == "" || r.HoldVersion == "" || !r.ExpiresAt.After(now) {
			return ErrReservationUnavailable
		}
		v, err := strconv.ParseUint(r.HoldVersion, 10, 64)
		if err != nil || v == 0 {
			return fmt.Errorf("%w: invalid hold version", ErrReservationInvalid)
		}
	}
	return nil
}
