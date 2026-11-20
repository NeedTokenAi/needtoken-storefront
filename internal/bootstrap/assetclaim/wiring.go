package assetclaimwiring

import (
	"context"
	"errors"

	"github.com/dujiao-next/internal/app/container"
	"github.com/dujiao-next/internal/constants"
	"github.com/dujiao-next/internal/modules/order/application/assetclaim"
	ordercontract "github.com/dujiao-next/internal/modules/order/contract"
	orderdomain "github.com/dujiao-next/internal/modules/order/domain"
	assetclaimhttp "github.com/dujiao-next/internal/modules/order/transport/http/assetclaim"
	"github.com/dujiao-next/internal/upstream"
)

func NewHandler(c *container.Container) *assetclaimhttp.Handler {
	return assetclaimhttp.New(assetclaim.New(orderDirectory{c: c}, upstreamBridge{c: c}))
}

type orderDirectory struct{ c *container.Container }

func (d orderDirectory) Resolve(_ context.Context, sub assetclaim.Subject) (assetclaim.OrderSnapshot, error) {
	if d.c == nil || d.c.OrderStore == nil {
		return assetclaim.OrderSnapshot{}, assetclaim.ErrUnavailable
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
		return assetclaim.OrderSnapshot{}, assetclaim.ErrUnavailable
	}
	if order == nil {
		return assetclaim.OrderSnapshot{}, assetclaim.ErrNotFound
	}
	snap := assetclaim.OrderSnapshot{
		OrderID: order.ID,
		OrderNo: order.OrderNo,
		UserID:  order.UserID,
		Leaf:    len(order.Children) == 0 && len(order.Items) == 1,
		Delivered: order.PaidAt != nil && order.RefundedAmount.IsZero() &&
			(order.Status == constants.OrderStatusDelivered || order.Status == constants.OrderStatusCompleted || order.Status == constants.OrderStatusFulfilling),
	}
	if !snap.Leaf || d.c.ProcurementOrderRepo == nil {
		return snap, nil
	}
	proc, err := d.c.ProcurementOrderRepo.GetByLocalOrderID(order.ID)
	if err != nil {
		return snap, assetclaim.ErrUnavailable
	}
	if proc != nil {
		snap.ConnectionID = proc.ConnectionID
		snap.UpstreamOrderID = proc.UpstreamOrderID
	}
	return snap, nil
}

type upstreamBridge struct{ c *container.Container }

func (b upstreamBridge) adapter(connectionID uint) (*upstream.DujiaoNextAdapter, error) {
	if b.c == nil || b.c.SiteConnectionService == nil {
		return nil, assetclaim.ErrUnavailable
	}
	conn, err := b.c.SiteConnectionService.GetByID(connectionID)
	if err != nil || conn == nil || conn.Status != constants.ConnectionStatusActive {
		return nil, assetclaim.ErrUnavailable
	}
	adapter, err := b.c.SiteConnectionService.GetAdapter(conn)
	if err != nil {
		return nil, assetclaim.ErrUnavailable
	}
	dujiao, ok := adapter.(*upstream.DujiaoNextAdapter)
	if !ok {
		return nil, assetclaim.ErrUnavailable
	}
	return dujiao, nil
}

func (b upstreamBridge) Status(ctx context.Context, connectionID, upstreamOrderID uint, buyerRef string) (assetclaim.Status, error) {
	adapter, err := b.adapter(connectionID)
	if err != nil {
		return assetclaim.Status{}, err
	}
	res, err := adapter.AssetClaimStatus(ctx, upstreamOrderID, buyerRef)
	if err != nil {
		return assetclaim.Status{}, mapBridgeErr(err)
	}
	return convertStatus(res), nil
}

func (b upstreamBridge) Reveal(ctx context.Context, connectionID, upstreamOrderID uint, buyerRef, actor string) (assetclaim.RevealResult, error) {
	adapter, err := b.adapter(connectionID)
	if err != nil {
		return assetclaim.RevealResult{}, err
	}
	res, err := adapter.RevealAssetClaim(ctx, upstreamOrderID, buyerRef, actor)
	if err != nil {
		return assetclaim.RevealResult{}, mapBridgeErr(err)
	}
	return assetclaim.RevealResult{
		Status:            convertStatus(res.AssetClaimStatus),
		AccountIdentifier: res.AccountIdentifier,
		SecretPayload:     res.SecretPayload,
		DeliveryNotes:     res.DeliveryNotes,
		Extra:             res.Extra,
	}, nil
}

func convertStatus(in upstream.AssetClaimStatus) assetclaim.Status {
	var ver *assetclaim.Verification
	if in.Verification != nil {
		ver = &assetclaim.Verification{
			Source:          in.Verification.Source,
			ObservedAt:      in.Verification.ObservedAt,
			RemoteAccountID: in.Verification.RemoteAccountID,
			PlanType:        in.Verification.PlanType,
			SubscriptionEnd: in.Verification.SubscriptionEnd,
			EvidenceRef:     in.Verification.EvidenceRef,
			EvidenceSHA256:  in.Verification.EvidenceSHA256,
		}
	}
	return assetclaim.Status{
		Eligible:         in.Eligible,
		AssetID:          in.AssetID,
		SKUCode:          in.SKUCode,
		Platform:         in.Platform,
		SaleMode:         in.SaleMode,
		MaskedIdentifier: in.MaskedIdentifier,
		DeliveredAt:      in.DeliveredAt,
		ExpiresAt:        in.ExpiresAt,
		RentalExpiresAt:  in.RentalExpiresAt,
		Verification:     ver,
		MaxReveals:       in.MaxReveals,
		RevealCount:      in.RevealCount,
		RemainingReveals: in.RemainingReveals,
		FirstRevealedAt:  in.FirstRevealedAt,
		LastRevealedAt:   in.LastRevealedAt,
	}
}

func mapBridgeErr(err error) error {
	switch {
	case errors.Is(err, upstream.ErrAssetClaimInvalid):
		return assetclaim.ErrInvalid
	case errors.Is(err, upstream.ErrAssetClaimForbidden):
		return assetclaim.ErrForbidden
	case errors.Is(err, upstream.ErrAssetClaimExhausted):
		return assetclaim.ErrExhausted
	case errors.Is(err, upstream.ErrAssetClaimNotFound):
		return assetclaim.ErrNotFound
	case errors.Is(err, upstream.ErrAssetClaimConflict):
		return assetclaim.ErrConflict
	default:
		return assetclaim.ErrUnavailable
	}
}
