package paymentbootstrap

import (
	"errors"
	"fmt"
	"time"

	paymentapp "github.com/dujiao-next/internal/modules/payment/application"
	"github.com/dujiao-next/internal/modules/payment/application/plancheckout"
	paymentcontract "github.com/dujiao-next/internal/modules/payment/contract"

	paymentdomain "github.com/dujiao-next/internal/modules/payment/domain"

	orderapp "github.com/dujiao-next/internal/modules/order/application"
	orderdomain "github.com/dujiao-next/internal/modules/order/domain"

	paymenttransport "github.com/dujiao-next/internal/modules/payment/transport/http"
	reseller "github.com/dujiao-next/internal/modules/reseller/contract"
	walletcontract "github.com/dujiao-next/internal/modules/wallet/contract"
)

type guestOrderLookupAdapter struct {
	orders *orderapp.OrderService
}

func (a guestOrderLookupAdapter) GetOrderByGuestOrderNoForTenant(tenant reseller.TenantContext, orderNo, email, password string) (*orderdomain.Order, error) {
	order, err := a.orders.GetOrderByGuestOrderNoForTenant(tenant, orderNo, email, password)
	return order, mapTransportError(err)
}

func (a guestOrderLookupAdapter) GetOrderByGuestForTenant(tenant reseller.TenantContext, orderID uint, email, password string) (*orderdomain.Order, error) {
	order, err := a.orders.GetOrderByGuestForTenant(tenant, orderID, email, password)
	return order, mapTransportError(err)
}

type userOrderLookupAdapter struct {
	orders *orderapp.OrderService
}

func (a userOrderLookupAdapter) GetOrderByUserOrderNoForTenant(tenant reseller.TenantContext, orderNo string, userID uint) (*orderdomain.Order, error) {
	order, err := a.orders.GetOrderByUserOrderNoForTenant(tenant, orderNo, userID)
	return order, mapTransportError(err)
}

func (a userOrderLookupAdapter) GetOrderByUserForTenant(tenant reseller.TenantContext, orderID, userID uint) (*orderdomain.Order, error) {
	order, err := a.orders.GetOrderByUserForTenant(tenant, orderID, userID)
	return order, mapTransportError(err)
}

type pendingLookupAdapter struct {
	payments paymentcontract.Store
	plans    *plancheckout.Guard
}

func (a pendingLookupAdapter) GetLatestPendingByOrder(orderID uint, now time.Time) (*paymentdomain.Payment, error) {
	// An older pending payment may contain a usable gateway URL. Recheck the
	// plan before exposing it, just as when creating or reusing a payment.
	if _, err := a.plans.Ensure(orderID); err != nil {
		return nil, mapTransportError(err)
	}
	return a.payments.GetLatestPendingByOrder(orderID, now)
}

type writerAdapter struct {
	payments *paymentapp.PaymentService
}

func (a writerAdapter) CreatePayment(input paymenttransport.CreatePaymentInput) (*paymenttransport.CreatePaymentResult, error) {
	result, err := a.payments.CreatePayment(paymentapp.CreatePaymentInput{
		OrderID:       input.OrderID,
		ChannelID:     input.ChannelID,
		UseBalance:    input.UseBalance,
		ClientIP:      input.ClientIP,
		Context:       input.Context,
		RequestScheme: input.RequestScheme,
	})
	if err != nil {
		return nil, mapTransportError(err)
	}
	if result == nil {
		return nil, nil
	}
	return &paymenttransport.CreatePaymentResult{
		Payment:          result.Payment,
		Channel:          result.Channel,
		OrderPaid:        result.OrderPaid,
		WalletPaidAmount: result.WalletPaidAmount,
		OnlinePayAmount:  result.OnlinePayAmount,
	}, nil
}

func (a writerAdapter) GetPayment(id uint) (*paymentdomain.Payment, error) {
	payment, err := a.payments.GetPayment(id)
	return payment, mapTransportError(err)
}

func (a writerAdapter) CapturePayment(input paymenttransport.CapturePaymentInput) (*paymentdomain.Payment, error) {
	payment, err := a.payments.CapturePayment(paymentapp.CapturePaymentInput{
		PaymentID: input.PaymentID,
		Context:   input.Context,
	})
	return payment, mapTransportError(err)
}

func mapTransportError(err error) error {
	if err == nil {
		return nil
	}
	for _, mapping := range []struct {
		source error
		target error
	}{
		{paymentapp.ErrCheckoutPolicyUnknown, paymenttransport.ErrCheckoutPolicyUnknown},
		{paymentapp.ErrCheckoutReservationUnavailable, paymenttransport.ErrCheckoutReservationUnavailable},
		{orderapp.ErrOrderNotFound, paymenttransport.ErrOrderNotFound},
		{orderapp.ErrGuestOrderNotFound, paymenttransport.ErrGuestOrderNotFound},
		{orderapp.ErrOrderStatusInvalid, paymenttransport.ErrOrderStatusInvalid},
		{paymentapp.ErrPaymentInvalid, paymenttransport.ErrPaymentInvalid},
		{paymentapp.ErrPaymentNotFound, paymenttransport.ErrPaymentNotFound},
		{paymentapp.ErrPaymentChannelNotFound, paymenttransport.ErrPaymentChannelNotFound},
		{paymentapp.ErrPaymentChannelInactive, paymenttransport.ErrPaymentChannelInactive},
		{paymentapp.ErrPaymentProviderNotSupported, paymenttransport.ErrPaymentProviderNotSupported},
		{paymentapp.ErrPaymentChannelConfigInvalid, paymenttransport.ErrPaymentChannelConfigInvalid},
		{paymentapp.ErrPaymentGatewayRequestFailed, paymenttransport.ErrPaymentGatewayRequestFailed},
		{paymentapp.ErrPaymentGatewayResponseInvalid, paymenttransport.ErrPaymentGatewayResponseInvalid},
		{paymentapp.ErrPaymentCurrencyMismatch, paymenttransport.ErrPaymentCurrencyMismatch},
		{walletcontract.ErrNotSupportedForGuest, paymenttransport.ErrWalletNotSupportedForGuest},
		{paymentapp.ErrPaymentChannelNotAllowedForProduct, paymenttransport.ErrPaymentChannelNotAllowedForProduct},
		{paymentapp.ErrPaymentChannelNotAllowedForRecharge, paymenttransport.ErrPaymentChannelNotAllowedForRecharge},
		{walletcontract.ErrOnlyPaymentRequired, paymenttransport.ErrWalletOnlyPaymentRequired},
		{paymentapp.ErrPaymentStatusInvalid, paymenttransport.ErrPaymentStatusInvalid},
		{paymentapp.ErrPaymentAmountMismatch, paymenttransport.ErrPaymentAmountMismatch},
		{orderapp.ErrCardSecretInsufficient, paymenttransport.ErrCardSecretInsufficient},
		{orderapp.ErrManualStockInsufficient, paymenttransport.ErrManualStockInsufficient},
		{paymentapp.ErrPaymentUnderpaidCredited, paymenttransport.ErrPaymentUnderpaidCredited},
		{paymentapp.ErrPaymentCanceledOrderUnderpaid, paymenttransport.ErrPaymentCanceledOrderUnderpaid},
	} {
		if errors.Is(err, mapping.source) {
			return fmt.Errorf("%w: %v", mapping.target, err)
		}
	}
	return err
}
