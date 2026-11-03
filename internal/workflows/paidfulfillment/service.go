package paidfulfillment

import (
	"context"
	"errors"
	"time"

	paymentcontract "github.com/dujiao-next/internal/modules/payment/contract"
	procurementcontract "github.com/dujiao-next/internal/modules/procurement/contract"
	"github.com/google/uuid"
)

type Creator interface{ CreateForOrder(orderID uint) error }
type ReservationConfirmer interface {
	ConfirmPaidOrder(context.Context, uint) error
}
type Service struct {
	payments     paymentcontract.Store
	procurements Creator
	reservations ReservationConfirmer
}

func New(payments paymentcontract.Store, procurements Creator, confirmations ...ReservationConfirmer) *Service {
	s := &Service{payments: payments, procurements: procurements}
	if len(confirmations) > 0 {
		s.reservations = confirmations[0]
	}
	return s
}

// RecoverPaidFulfillment is independent of Redis. Acknowledge the durable paid
// event only after all required procurement rows exist. Leases and unique keys
// allow overlapping API/worker processes and recover interrupted dispatches.
func (s *Service) RecoverPaidFulfillment() error {
	if s == nil || s.payments == nil || s.procurements == nil {
		return nil
	}
	now := time.Now()
	if err := s.payments.BackfillPaidFulfillment(now, 200); err != nil {
		return err
	}
	events, err := s.payments.ListPendingPaidFulfillment(now, 200)
	if err != nil {
		return err
	}
	var errs []error
	for _, event := range events {
		leaseID := uuid.NewString()
		claimed, err := s.payments.ClaimPaidFulfillment(event.OrderID, leaseID, time.Now())
		if err != nil {
			errs = append(errs, err)
			continue
		}
		if !claimed {
			continue
		}
		if s.reservations != nil {
			err = s.reservations.ConfirmPaidOrder(context.Background(), event.OrderID)
		} else {
			// A missed integration must fail closed for an explicitly scarce
			// order rather than turn an absent confirmer into permission.
			err = s.payments.WithinTransaction(func(tx paymentcontract.Transaction) error {
				order, e := tx.Orders().GetByIDForUpdateWithChildren(event.OrderID)
				if e != nil {
					return e
				}
				reqs, e := paymentcontract.CheckoutRequirements(order)
				if e != nil {
					return e
				}
				if len(reqs) > 0 {
					return paymentcontract.ErrReservationUnavailable
				}
				return nil
			})
		}
		if err == nil {
			err = s.procurements.CreateForOrder(event.OrderID)
		}
		completed := err == nil || errors.Is(err, procurementcontract.ErrExists)
		if !completed {
			errs = append(errs, err)
		}
		if err := s.payments.FinishPaidFulfillment(event.OrderID, leaseID, completed, time.Now()); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}
