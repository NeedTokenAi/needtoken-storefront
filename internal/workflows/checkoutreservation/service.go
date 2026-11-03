// Package checkoutreservation coordinates durable local checkout promises with
// the upstream asset book. Network calls always run outside DB transactions.
package checkoutreservation

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"time"

	"github.com/dujiao-next/internal/constants"
	paymentcontract "github.com/dujiao-next/internal/modules/payment/contract"
	paymentdomain "github.com/dujiao-next/internal/modules/payment/domain"
	"github.com/dujiao-next/internal/shared/jsonmap"
)

type Service struct {
	payments paymentcontract.Store
	upstream paymentcontract.ReservationProvider
	now      func() time.Time
}

func New(payments paymentcontract.Store, upstream paymentcontract.ReservationProvider) *Service {
	return &Service{payments: payments, upstream: upstream, now: time.Now}
}

func requestJSON(r paymentcontract.ReservationRequest) jsonmap.JSON {
	b, _ := json.Marshal(r)
	var p jsonmap.JSON
	_ = json.Unmarshal(b, &p)
	return p
}
func decodeRequest(p jsonmap.JSON) (paymentcontract.ReservationRequest, error) {
	b, err := json.Marshal(p)
	if err != nil {
		return paymentcontract.ReservationRequest{}, err
	}
	var r paymentcontract.ReservationRequest
	err = json.Unmarshal(b, &r)
	if err != nil || r.PayloadHash != paymentcontract.ReservationDigest(r) {
		return r, paymentcontract.ErrReservationInvalid
	}
	return r, nil
}

func stableSuffix(orderNo string, itemID uint) string {
	h := sha256.Sum256([]byte(fmt.Sprintf("%s\x00%d", orderNo, itemID)))
	return hex.EncodeToString(h[:20])
}

func (s *Service) prepare(rootID uint) ([]paymentdomain.CheckoutReservation, error) {
	rows := []paymentdomain.CheckoutReservation{}
	err := s.payments.WithinTransaction(func(tx paymentcontract.Transaction) error {
		root, err := tx.Orders().GetByIDForUpdateWithChildren(rootID)
		if err != nil {
			return err
		}
		if root == nil || root.ParentID != nil {
			return paymentcontract.ErrReservationInvalid
		}
		// PostgreSQL timestamps store microsecond precision. Freeze the wire
		// expiry at that precision before committing its immutable hash so a
		// DB readback cannot appear to change the request's identity.
		now := s.now().UTC().Truncate(time.Microsecond)
		if root.Status != constants.OrderStatusPendingPayment || root.PaidAt != nil || (root.ExpiresAt != nil && !root.ExpiresAt.After(now)) {
			return paymentcontract.ErrReservationUnavailable
		}
		reqs, err := paymentcontract.CheckoutRequirements(root)
		if err != nil {
			return err
		}
		if len(reqs) == 0 {
			return nil
		}
		stored, err := tx.Payments().ListCheckoutReservations(rootID)
		if err != nil {
			return err
		}
		byItem := map[uint]paymentdomain.CheckoutReservation{}
		for _, r := range stored {
			byItem[r.OrderItemID] = r
		}
		for _, req := range reqs {
			if old, ok := byItem[req.ItemID]; ok {
				if old.RequirementHash != req.Hash() {
					return paymentcontract.ErrReservationConflict
				}
				rows = append(rows, old)
				continue
			}
			expires := now.Add(time.Duration(req.Terms.TTLSeconds) * time.Second)
			if req.OrderExpiresAt != nil && req.OrderExpiresAt.Before(expires) {
				expires = req.OrderExpiresAt.UTC()
			}
			if !expires.After(now) {
				return paymentcontract.ErrReservationUnavailable
			}
			suffix := stableSuffix(req.RootOrderNo, req.ItemID)
			wire := paymentcontract.ReservationRequest{OperationID: "sf_reserve_" + suffix, Action: "reserve", OrderRef: "sf_order_" + suffix, Snapshot: paymentcontract.ReservationSnapshot{SKU: req.Terms.SKU, Region: req.Terms.Region, Currency: req.Terms.Currency, RequiredMinor: strconv.FormatInt(req.Terms.RequiredMinor, 10), ExpiresAt: expires.Format(time.RFC3339Nano)}, ExpectedVersion: "0"}
			wire.PayloadHash = paymentcontract.ReservationDigest(wire)
			r := paymentdomain.CheckoutReservation{OrderItemID: req.ItemID, RootOrderID: rootID, OrderID: req.OrderID, ConnectionID: req.ConnectionID, UpstreamSKUID: req.UpstreamSKUID, RequirementHash: req.Hash(), OrderRef: wire.OrderRef, ReserveOperationID: wire.OperationID, ReservePayloadHash: wire.PayloadHash, ReserveRequestJSON: requestJSON(wire), State: "pending", ExpiresAt: expires, CreatedAt: now, UpdatedAt: now}
			created, err := tx.Payments().CreateCheckoutReservation(r)
			if err != nil {
				return err
			}
			rows = append(rows, *created)
		}
		return nil
	})
	return rows, err
}

func (s *Service) EnsureBeforePayment(ctx context.Context, rootID uint) error {
	if s == nil || s.payments == nil || s.upstream == nil {
		return paymentcontract.ErrReservationUnavailable
	}
	rows, err := s.prepare(rootID)
	if err != nil {
		return err
	}
	for _, row := range rows {
		if row.State == "needs_attention" || row.State == "expired" || row.State == "released" || !row.ExpiresAt.After(s.now()) {
			return paymentcontract.ErrReservationAttention
		}
		request, err := decodeRequest(row.ReserveRequestJSON)
		if err != nil {
			return err
		}
		conn, err := s.upstream.OpenReservation(row.ConnectionID)
		if err != nil {
			return s.noteFailure(row, err)
		}
		row, err = s.bindIdentity(row, conn)
		if err != nil {
			return err
		}
		result, err := lookupOrApply(ctx, conn, request)
		if err != nil {
			return s.noteFailure(row, err)
		}
		row, err = s.saveResult(row, request, result)
		if err != nil {
			return err
		}
		if row.State != "reserved" || !row.ReadyForPayment || !row.ExpiresAt.After(s.now()) {
			return paymentcontract.ErrReservationUnavailable
		}
	}
	return nil
}

func lookupOrApply(ctx context.Context, c paymentcontract.ReservationConnection, r paymentcontract.ReservationRequest) (*paymentcontract.ReservationResult, error) {
	result, err := c.GetReservation(ctx, r.OrderRef, r.OperationID)
	if err == nil {
		return result, nil
	}
	if !errors.Is(err, paymentcontract.ErrReservationNotFound) {
		return nil, err
	}
	// An explicit, authenticated 404 is the only result that permits posting
	// this same immutable operation. Timeouts never allocate another account.
	return c.ApplyReservation(ctx, r)
}

func (s *Service) noteFailure(row paymentdomain.CheckoutReservation, cause error) error {
	row.UpdatedAt = s.now().UTC()
	row.ReadyForPayment = false
	if errors.Is(cause, paymentcontract.ErrReservationConflict) || errors.Is(cause, paymentcontract.ErrReservationInvalid) {
		row.State = "needs_attention"
		row.AttentionCode = "upstream_rejected_immutable_reservation"
	} else {
		row.State = "unknown"
		row.AttentionCode = "upstream_result_unknown_query_original_operation"
	}
	if _, err := s.payments.SaveCheckoutReservation(row); err != nil {
		return err
	}
	if row.State == "needs_attention" {
		return paymentcontract.ErrReservationAttention
	}
	return paymentcontract.ErrReservationUnavailable
}

var opaqueHold = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_-]{2,127}$`)
var identityHash = regexp.MustCompile(`^[0-9a-f]{64}$`)

func (s *Service) bindIdentity(row paymentdomain.CheckoutReservation, conn paymentcontract.ReservationConnection) (paymentdomain.CheckoutReservation, error) {
	identity := conn.ReservationIdentity()
	if !identityHash.MatchString(identity) || (row.UpstreamIdentity != "" && row.UpstreamIdentity != identity) || (row.UpstreamIdentity == "" && row.HoldID != "") {
		return row, s.noteFailure(row, paymentcontract.ErrReservationConflict)
	}
	if row.UpstreamIdentity == "" {
		row.UpstreamIdentity = identity
		row.UpdatedAt = s.now().UTC()
		stored, err := s.payments.SaveCheckoutReservation(row)
		if err != nil {
			return row, err
		}
		row = *stored
	}
	return row, nil
}

func validateResult(row paymentdomain.CheckoutReservation, request paymentcontract.ReservationRequest, r *paymentcontract.ReservationResult) error {
	if r == nil || !r.OK || r.OperationID != request.OperationID || r.PayloadHash != request.PayloadHash || r.OrderRef != row.OrderRef || r.Snapshot != request.Snapshot || !opaqueHold.MatchString(r.HoldID) || !r.BookOnly || r.EntitlementStatus != "unverified" {
		return paymentcontract.ErrReservationConflict
	}
	if row.HoldID != "" && r.HoldID != row.HoldID {
		return paymentcontract.ErrReservationConflict
	}
	v, err := strconv.ParseUint(r.Version, 10, 64)
	if err != nil || v == 0 || strconv.FormatUint(v, 10) != r.Version {
		return paymentcontract.ErrReservationInvalid
	}
	switch r.Status {
	case "reserved", "paid", "bound", "consumed", "needs_attention", "expired", "released":
	default:
		return paymentcontract.ErrReservationInvalid
	}
	if r.ReadyForPayment && (r.Status != "reserved" || r.BookStatus != "reserved") {
		return paymentcontract.ErrReservationInvalid
	}
	if (r.Status == "paid" || r.Status == "bound" || r.Status == "consumed") && r.PaidAt == nil {
		return paymentcontract.ErrReservationInvalid
	}
	return nil
}

func (s *Service) saveResult(row paymentdomain.CheckoutReservation, request paymentcontract.ReservationRequest, r *paymentcontract.ReservationResult) (paymentdomain.CheckoutReservation, error) {
	if err := validateResult(row, request, r); err != nil {
		_ = s.noteFailure(row, err)
		return row, err
	}
	row.HoldID, row.HoldVersion, row.State, row.BookStatus, row.ReadyForPayment = r.HoldID, r.Version, r.Status, r.BookStatus, r.ReadyForPayment
	row.UpdatedAt = s.now().UTC()
	row.AttentionCode = ""
	if row.State == "needs_attention" {
		row.AttentionCode = "paid_hold_or_asset_requires_manual_review"
	}
	stored, err := s.payments.SaveCheckoutReservation(row)
	if err != nil {
		return row, err
	}
	return *stored, nil
}

func paidStatus(status string) bool {
	return status == constants.OrderStatusPaid || status == constants.OrderStatusFulfilling || status == constants.OrderStatusPartiallyDelivered || status == constants.OrderStatusDelivered || status == constants.OrderStatusCompleted
}

func (s *Service) paidRows(rootID uint) ([]paymentdomain.CheckoutReservation, error) {
	rows := []paymentdomain.CheckoutReservation{}
	err := s.payments.WithinTransaction(func(tx paymentcontract.Transaction) error {
		root, err := tx.Orders().GetByIDForUpdateWithChildren(rootID)
		if err != nil {
			return err
		}
		if root == nil || root.PaidAt == nil || !paidStatus(root.Status) {
			return paymentcontract.ErrReservationAttention
		}
		reqs, err := paymentcontract.CheckoutRequirements(root)
		if err != nil {
			return err
		}
		if len(reqs) == 0 {
			return nil
		}
		stored, err := tx.Payments().ListCheckoutReservations(rootID)
		if err != nil {
			return err
		}
		byItem := map[uint]paymentdomain.CheckoutReservation{}
		for _, r := range stored {
			byItem[r.OrderItemID] = r
		}
		for _, req := range reqs {
			row, ok := byItem[req.ItemID]
			if !ok || row.RequirementHash != req.Hash() {
				return paymentcontract.ErrReservationAttention
			}
			rows = append(rows, row)
		}
		return nil
	})
	return rows, err
}

// ConfirmPaidOrder is called by the resident paid outbox before procurement.
// It never reserves after a payment. Late/expired/unknown holds stop automatic
// fulfillment and preserve the original capacity reference for review.
func (s *Service) ConfirmPaidOrder(ctx context.Context, rootID uint) error {
	if s == nil || s.payments == nil || s.upstream == nil {
		return paymentcontract.ErrReservationUnavailable
	}
	rows, err := s.paidRows(rootID)
	if err != nil {
		return err
	}
	for _, row := range rows {
		if row.AttentionCode == "upstream_rejected_immutable_reservation" {
			return paymentcontract.ErrReservationAttention
		}
		reserve, err := decodeRequest(row.ReserveRequestJSON)
		if err != nil {
			return err
		}
		conn, err := s.upstream.OpenReservation(row.ConnectionID)
		if err != nil {
			return s.noteFailure(row, err)
		}
		row, err = s.bindIdentity(row, conn)
		if err != nil {
			return err
		}
		// A paid order cannot create a replacement hold, even if the original
		// operation is missing after a remote restore.
		result, err := conn.GetReservation(ctx, row.OrderRef, reserve.OperationID)
		if err != nil {
			return s.noteFailure(row, err)
		}
		row, err = s.saveResult(row, reserve, result)
		if err != nil {
			return err
		}
		if row.State == "paid" || row.State == "bound" || row.State == "consumed" {
			continue
		}
		if row.State == "needs_attention" && result.PaidAt != nil {
			return paymentcontract.ErrReservationAttention
		}
		var confirm paymentcontract.ReservationRequest
		err = s.payments.WithinTransaction(func(tx paymentcontract.Transaction) error {
			root, e := tx.Orders().GetByIDForUpdateWithChildren(rootID)
			if e != nil {
				return e
			}
			if root == nil || root.PaidAt == nil || !paidStatus(root.Status) {
				return paymentcontract.ErrReservationAttention
			}
			stored, e := tx.Payments().ListCheckoutReservations(rootID)
			if e != nil {
				return e
			}
			for _, current := range stored {
				if current.OrderItemID == row.OrderItemID {
					row = current
					break
				}
			}
			if len(row.ConfirmRequestJSON) > 0 {
				confirm, e = decodeRequest(row.ConfirmRequestJSON)
				return e
			}
			confirm = reserve
			confirm.OperationID = "sf_paid_" + stableSuffix(root.OrderNo, row.OrderItemID)
			confirm.Action = "confirm_payment"
			confirm.ExpectedVersion = row.HoldVersion
			confirm.EvidenceRef = "ev_storefront_paid_" + stableSuffix(root.OrderNo, row.OrderItemID)
			confirm.PayloadHash = paymentcontract.ReservationDigest(confirm)
			row.ConfirmRequestJSON = requestJSON(confirm)
			row.UpdatedAt = s.now().UTC()
			_, e = tx.Payments().SaveCheckoutReservation(row)
			return e
		})
		if err != nil {
			return err
		}
		result, err = lookupOrApply(ctx, conn, confirm)
		if err != nil {
			return s.noteFailure(row, err)
		}
		row, err = s.saveResult(row, confirm, result)
		if err != nil {
			return err
		}
		if row.State != "paid" && row.State != "bound" && row.State != "consumed" {
			return paymentcontract.ErrReservationAttention
		}
	}
	return nil
}

var _ paymentcontract.CheckoutReservationGate = (*Service)(nil)
