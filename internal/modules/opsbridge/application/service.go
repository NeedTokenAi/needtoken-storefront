package application

import (
	"fmt"
	"time"

	"github.com/dujiao-next/internal/constants"
	opscontract "github.com/dujiao-next/internal/modules/opsbridge/contract"
	ordercontract "github.com/dujiao-next/internal/modules/order/contract"
	orderdomain "github.com/dujiao-next/internal/modules/order/domain"
	paymentcontract "github.com/dujiao-next/internal/modules/payment/contract"
	procurementcontract "github.com/dujiao-next/internal/modules/procurement/contract"
)

type Service struct {
	orders       ordercontract.Store
	payments     paymentcontract.Store
	procurements procurementcontract.Repository
}

func New(orders ordercontract.Store, payments paymentcontract.Store, procurements procurementcontract.Repository) *Service {
	return &Service{orders: orders, payments: payments, procurements: procurements}
}
func (s *Service) Orders(page, pageSize int) (*opscontract.OrdersResult, error) {
	collected := time.Now().UTC()
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 50
	}
	if pageSize > 100 {
		pageSize = 100
	}
	rows, total, err := s.orders.ListAdmin(ordercontract.ListFilter{Page: page, PageSize: pageSize, SortBy: "id", SortOrder: "desc"})
	if err != nil {
		return nil, err
	}
	result := &opscontract.OrdersResult{Source: "storefront", CollectedAt: collected, Freshness: "fresh", Orders: make([]opscontract.Order, 0, len(rows)), Total: total, Page: page, PageSize: pageSize}
	var mapOrder func(orderdomain.Order) (opscontract.Order, error)
	mapOrder = func(row orderdomain.Order) (opscontract.Order, error) {
		out := opscontract.Order{ID: row.ID, OrderNo: row.OrderNo, ParentID: row.ParentID, Status: row.Status, Currency: row.Currency, TotalAmount: row.TotalAmount.String(), RefundedAmount: row.RefundedAmount.String(), PaymentStatus: "unpaid", FulfillmentStatus: row.Status, EntitlementStatus: "unknown", PaidAt: row.PaidAt, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt, Payments: []opscontract.Payment{}, Refunds: []opscontract.Refund{}}
		if row.PaidAt != nil {
			out.PaymentStatus = "paid"
		}
		if row.UserID > 0 {
			out.MerchantCustomerID = fmt.Sprintf("needtoken-storefront:user:%d", row.UserID)
		}
		if row.Fulfillment != nil {
			out.FulfillmentStatus = row.Fulfillment.Status
		}
		payments, err := s.payments.ListByOrderID(row.ID)
		if err != nil {
			return opscontract.Order{}, err
		}
		for _, payment := range payments {
			out.Payments = append(out.Payments, opscontract.Payment{ID: payment.ID, Status: payment.Status, Amount: payment.Amount.String(), Currency: payment.Currency, ProviderRef: payment.ProviderRef, PaidAt: payment.PaidAt, UpdatedAt: payment.UpdatedAt})
		}
		refunds, err := s.orders.ListRefundRecordsByOrderIDs([]uint{row.ID})
		if err != nil {
			return opscontract.Order{}, err
		}
		for _, refund := range refunds {
			settlement := refund.SettlementStatus
			if settlement == "" {
				settlement = "unknown"
				if refund.Type == constants.OrderRefundTypeManual {
					settlement = "external_unverified"
				}
			}
			out.Refunds = append(out.Refunds, opscontract.Refund{ID: refund.ID, OperationID: refund.OperationID, Type: refund.Type, Amount: refund.Amount.String(), Currency: refund.Currency, SettlementStatus: settlement, UpdatedAt: refund.UpdatedAt})
		}
		procurement, err := s.procurements.GetByLocalOrderID(row.ID)
		if err != nil {
			return opscontract.Order{}, err
		}
		if procurement != nil {
			out.Procurement = &opscontract.Procurement{ID: procurement.ID, Status: procurement.Status, UpstreamOrderNo: procurement.UpstreamOrderNo, UpdatedAt: procurement.UpdatedAt, LocalOrderID: procurement.LocalOrderID}
		}
		out.Children = []opscontract.Order{}
		out.Procurements = []opscontract.Procurement{}
		out.Reservations = []opscontract.Reservation{}
		reservationRootID := row.ID
		if row.ParentID != nil {
			reservationRootID = *row.ParentID
		}
		reservations, err := s.payments.ListCheckoutReservations(reservationRootID)
		if err != nil {
			return opscontract.Order{}, err
		}
		for _, r := range reservations {
			if r.OrderID != row.ID {
				continue
			}
			out.Reservations = append(out.Reservations, opscontract.Reservation{OrderItemID: r.OrderItemID, LocalOrderID: r.OrderID, HoldID: r.HoldID, OrderRef: r.OrderRef, Version: r.HoldVersion, Status: r.State, BookStatus: r.BookStatus, ReadyForPayment: r.ReadyForPayment, ExpiresAt: r.ExpiresAt, AttentionCode: r.AttentionCode, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt})
		}
		if out.Procurement != nil {
			out.Procurements = append(out.Procurements, *out.Procurement)
		}
		for _, child := range row.Children {
			mapped, err := mapOrder(child)
			if err != nil {
				return opscontract.Order{}, err
			}
			out.Children = append(out.Children, mapped)
			out.Procurements = append(out.Procurements, mapped.Procurements...)
			out.Reservations = append(out.Reservations, mapped.Reservations...)
		}
		return out, nil
	}
	for _, row := range rows {
		mapped, err := mapOrder(row)
		if err != nil {
			return nil, err
		}
		result.Orders = append(result.Orders, mapped)
	}

	return result, nil
}
func (s *Service) Overview() (*opscontract.Overview, error) {
	result := &opscontract.Overview{Source: "storefront", CollectedAt: time.Now().UTC(), Freshness: "fresh", OrdersByStatus: map[string]int64{}, Capabilities: []string{"orders.read", "payments.read", "refunds.read", "procurement.read", "checkout_reservations.read"}}
	_, total, err := s.orders.ListAdmin(ordercontract.ListFilter{Page: 1, PageSize: 1})
	if err != nil {
		return nil, err
	}
	result.TotalOrders = total
	for _, status := range []string{constants.OrderStatusPendingPayment, constants.OrderStatusPaid, constants.OrderStatusFulfilling, constants.OrderStatusPartiallyDelivered, constants.OrderStatusDelivered, constants.OrderStatusCompleted, constants.OrderStatusCanceled, constants.OrderStatusRefunded, constants.OrderStatusPartiallyRefunded} {
		_, count, err := s.orders.ListAdmin(ordercontract.ListFilter{Status: status, Page: 1, PageSize: 1})
		if err != nil {
			return nil, err
		}
		result.OrdersByStatus[status] = count
	}
	return result, nil
}
