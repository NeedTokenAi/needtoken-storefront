package aftersaleswiring

import (
	"context"
	"errors"
	"strings"

	"github.com/dujiao-next/internal/app/container"
	"github.com/dujiao-next/internal/constants"
	"github.com/dujiao-next/internal/modules/order/application/aftersales"
	ordercontract "github.com/dujiao-next/internal/modules/order/contract"
	orderdomain "github.com/dujiao-next/internal/modules/order/domain"
	aftersaleshttp "github.com/dujiao-next/internal/modules/order/transport/http/aftersales"
	siteconnectiondomain "github.com/dujiao-next/internal/modules/siteconnection/domain"
	"github.com/dujiao-next/internal/upstream"
)

func NewHandler(c *container.Container) *aftersaleshttp.Handler {
	return aftersaleshttp.New(aftersales.New(orderDirectory{c: c}, upstreamBridge{c: c}))
}

type orderDirectory struct{ c *container.Container }

func (d orderDirectory) Resolve(_ context.Context, sub aftersales.Subject) (aftersales.OrderSnapshot, error) {
	if d.c == nil || d.c.OrderStore == nil {
		return aftersales.OrderSnapshot{}, aftersales.ErrUnavailable
	}
	var order *orderdomain.Order
	var err error
	scope := ordercontract.TenantScope{ResellerID: sub.ResellerID}
	if sub.UserID > 0 {
		order, err = d.c.OrderStore.GetAnyByOrderNoAndUserScoped(sub.OrderNo, sub.UserID, scope)
	} else {
		order, err = d.c.OrderStore.GetAnyByOrderNoAndGuestScoped(sub.OrderNo, sub.GuestEmail, sub.GuestPassword, scope)
	}
	if err != nil {
		return aftersales.OrderSnapshot{}, aftersales.ErrUnavailable
	}
	if order == nil {
		return aftersales.OrderSnapshot{}, aftersales.ErrNotFound
	}
	customerEmail := strings.TrimSpace(order.GuestEmail)
	if customerEmail == "" && order.UserID > 0 && d.c.UserStore != nil {
		if user, uErr := d.c.UserStore.GetByID(order.UserID); uErr == nil && user != nil {
			customerEmail = strings.TrimSpace(user.Email)
		}
	}
	snap := aftersales.OrderSnapshot{
		OrderID:       order.ID,
		OrderNo:       order.OrderNo,
		UserID:        order.UserID,
		CustomerEmail: customerEmail,
	}
	if d.c.ProcurementOrderRepo != nil {
		proc, err := d.c.ProcurementOrderRepo.GetByLocalOrderID(order.ID)
		if err != nil {
			return aftersales.OrderSnapshot{}, aftersales.ErrUnavailable
		}
		if proc != nil && proc.ConnectionID > 0 {
			snap.ConnectionID = proc.ConnectionID
			return snap, nil
		}
	}
	// Fallback for non-procurement orders (e.g. balance/card disputes): use the unique active DujiaoNext upstream connection.
	if d.c.SiteConnectionRepo != nil {
		active, err := d.c.SiteConnectionRepo.ListActive()
		if err != nil {
			return aftersales.OrderSnapshot{}, aftersales.ErrUnavailable
		}
		var matches []siteconnectiondomain.Connection
		for _, item := range active {
			if item.Protocol == constants.ConnectionProtocolDujiaoNext {
				matches = append(matches, item)
			}
		}
		if len(matches) == 1 {
			snap.ConnectionID = matches[0].ID
			return snap, nil
		}
	}
	return aftersales.OrderSnapshot{}, aftersales.ErrUnavailable
}

type upstreamBridge struct{ c *container.Container }

func (b upstreamBridge) adapter(connectionID uint) (*upstream.DujiaoNextAdapter, error) {
	if b.c == nil || b.c.SiteConnectionService == nil {
		return nil, aftersales.ErrUnavailable
	}
	conn, err := b.c.SiteConnectionService.GetByID(connectionID)
	if err != nil || conn == nil || conn.Status != constants.ConnectionStatusActive {
		return nil, aftersales.ErrUnavailable
	}
	adapter, err := b.c.SiteConnectionService.GetAdapter(conn)
	if err != nil {
		return nil, aftersales.ErrUnavailable
	}
	dujiao, ok := adapter.(*upstream.DujiaoNextAdapter)
	if !ok {
		return nil, aftersales.ErrUnavailable
	}
	return dujiao, nil
}

func (b upstreamBridge) List(ctx context.Context, connectionID uint, orderNo string) ([]aftersales.Ticket, error) {
	adapter, err := b.adapter(connectionID)
	if err != nil {
		return nil, err
	}
	items, err := adapter.ListOrderAfterSalesTickets(ctx, orderNo)
	if err != nil {
		return nil, mapBridgeErr(err)
	}
	out := make([]aftersales.Ticket, 0, len(items))
	for _, it := range items {
		out = append(out, convertTicket(it))
	}
	return out, nil
}

func (b upstreamBridge) Create(ctx context.Context, connectionID uint, orderNo, customerEmail, category, summary, idempotencyKey string) (aftersales.Ticket, error) {
	adapter, err := b.adapter(connectionID)
	if err != nil {
		return aftersales.Ticket{}, err
	}
	res, err := adapter.CreateOrderAfterSalesTicket(ctx, orderNo, customerEmail, category, summary, idempotencyKey)
	if err != nil {
		return aftersales.Ticket{}, mapBridgeErr(err)
	}
	return convertTicket(res), nil
}

func (b upstreamBridge) AppendSupplement(ctx context.Context, connectionID uint, orderNo, ticketID, message string) (aftersales.Ticket, error) {
	adapter, err := b.adapter(connectionID)
	if err != nil {
		return aftersales.Ticket{}, err
	}
	res, err := adapter.AppendOrderAfterSalesSupplement(ctx, orderNo, ticketID, message)
	if err != nil {
		return aftersales.Ticket{}, mapBridgeErr(err)
	}
	return convertTicket(res), nil
}

func convertTicket(in upstream.OrderAfterSalesTicket) aftersales.Ticket {
	var supps []aftersales.Supplement
	if len(in.Supplements) > 0 {
		supps = make([]aftersales.Supplement, 0, len(in.Supplements))
		for _, s := range in.Supplements {
			supps = append(supps, aftersales.Supplement{
				Author:    s.Author,
				Message:   s.Message,
				CreatedAt: s.CreatedAt,
			})
		}
	}
	return aftersales.Ticket{
		ID:             in.ID,
		OrderNo:        in.OrderNo,
		Category:       in.Category,
		Priority:       in.Priority,
		Status:         in.Status,
		Summary:        in.Summary,
		ResolutionNote: in.ResolutionNote,
		Supplements:    supps,
		CreatedAt:      in.CreatedAt,
		UpdatedAt:      in.UpdatedAt,
		ResolvedAt:     in.ResolvedAt,
	}
}

func mapBridgeErr(err error) error {
	switch {
	case errors.Is(err, upstream.ErrAfterSalesInvalid):
		return aftersales.ErrInvalid
	case errors.Is(err, upstream.ErrAfterSalesNotFound):
		return aftersales.ErrNotFound
	case errors.Is(err, upstream.ErrAfterSalesConflict):
		return aftersales.ErrConflict
	default:
		return aftersales.ErrUnavailable
	}
}
