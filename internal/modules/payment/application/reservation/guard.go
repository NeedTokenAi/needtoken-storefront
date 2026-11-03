package reservation

import (
	"context"
	"errors"
	"fmt"
	"github.com/dujiao-next/internal/constants"
	"time"

	productcontract "github.com/dujiao-next/internal/modules/catalog/product/contract"
	orderapp "github.com/dujiao-next/internal/modules/order/application"
	ordercontract "github.com/dujiao-next/internal/modules/order/contract"
	orderdomain "github.com/dujiao-next/internal/modules/order/domain"
	paymentcontract "github.com/dujiao-next/internal/modules/payment/contract"
)

type Guard struct {
	orders ordercontract.Store
	skus   productcontract.SKURepository
	gate   paymentcontract.CheckoutReservationGate
}

func New(orders ordercontract.Store, skus productcontract.SKURepository, gate paymentcontract.CheckoutReservationGate) *Guard {
	return &Guard{orders: orders, skus: skus, gate: gate}
}

var ErrUnavailable = errors.New("checkout reservation not confirmed")
var ErrPolicyUnknown = paymentcontract.ErrReservationPolicyUnknown

func checkoutReservationError(err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%w: %w", ErrUnavailable, err)
}

// Missing historical scarce terms cannot be reconstructed from today's SKU.
// Ordinary products retain their existing behavior. Production always supplies
// a SKU repository; narrow non-catalog service callers may leave it unset.
func checkCurrentReservationMetadata(root *orderdomain.Order, skus productcontract.SKURepository) error {
	if skus == nil {
		return nil
	}
	var visit func(*orderdomain.Order) error
	visit = func(o *orderdomain.Order) error {
		for _, item := range o.Items {
			if item.FulfillmentType == constants.FulfillmentTypeUpstream {
				if known, ok := item.SKUSnapshotJSON["reservation_policy_known"].(bool); !ok || !known {
					return ErrPolicyUnknown
				}
			}
			if required, ok := item.SKUSnapshotJSON["requires_reservation"].(bool); ok && required {
				continue
			}
			if item.SKUID == 0 {
				continue
			}
			sku, err := skus.GetByID(item.SKUID)
			if err != nil {
				return err
			}
			if sku != nil && sku.RequiresReservation {
				return paymentcontract.ErrReservationInvalid
			}
		}
		for i := range o.Children {
			if err := visit(&o.Children[i]); err != nil {
				return err
			}
		}
		return nil
	}
	return visit(root)
}

func (s *Guard) Ensure(ctx context.Context, orderID uint) error {
	root, err := s.orders.GetByID(orderID)
	if err != nil {
		return orderapp.ErrOrderFetchFailed
	}
	if root == nil {
		return orderapp.ErrOrderNotFound
	}
	if err = checkCurrentReservationMetadata(root, s.skus); err != nil {
		return checkoutReservationError(err)
	}
	reqs, err := paymentcontract.CheckoutRequirements(root)
	if err != nil {
		return checkoutReservationError(err)
	}
	if len(reqs) == 0 {
		return nil
	}
	if s.gate == nil {
		return checkoutReservationError(paymentcontract.ErrReservationUnavailable)
	}
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	return checkoutReservationError(s.gate.EnsureBeforePayment(ctx, orderID))
}

func (s *Guard) Check(tx paymentcontract.Transaction, root *orderdomain.Order, now time.Time) error {
	if s.skus != nil {
		if err := checkCurrentReservationMetadata(root, tx.ProductSKUs()); err != nil {
			return checkoutReservationError(err)
		}
	}
	reqs, err := paymentcontract.CheckoutRequirements(root)
	if err != nil {
		return checkoutReservationError(err)
	}
	if len(reqs) == 0 {
		return nil
	}
	rows, err := tx.Payments().ListCheckoutReservations(root.ID)
	if err != nil {
		return checkoutReservationError(err)
	}
	return checkoutReservationError(paymentcontract.ValidateCheckoutReservations(root, rows, now))
}

// ConfirmPaidOrder satisfies the resident outbox's minimal confirmation port.
// Its caller only creates procurement after this returns a known success.
func (s *Guard) ConfirmPaidOrder(ctx context.Context, orderID uint) error {
	root, err := s.orders.GetByID(orderID)
	if err != nil {
		return err
	}
	if root == nil {
		return paymentcontract.ErrReservationAttention
	}
	if err = checkCurrentReservationMetadata(root, s.skus); err != nil {
		return err
	}
	reqs, err := paymentcontract.CheckoutRequirements(root)
	if err != nil {
		return err
	}
	if len(reqs) == 0 {
		return nil
	}
	if s.gate == nil {
		return paymentcontract.ErrReservationUnavailable
	}
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	return s.gate.ConfirmPaidOrder(ctx, orderID)
}

func (s *Guard) DeferProcurement(root *orderdomain.Order) bool {
	reqs, err := paymentcontract.CheckoutRequirements(root)
	if err != nil || len(reqs) > 0 {
		return true
	}
	return checkCurrentReservationMetadata(root, s.skus) != nil
}
