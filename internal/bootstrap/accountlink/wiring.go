package accountlinkwiring

import (
	"context"
	"errors"
	"os"
	"strconv"
	"strings"

	"github.com/dujiao-next/internal/app/container"
	"github.com/dujiao-next/internal/constants"
	"github.com/dujiao-next/internal/modules/order/application/accountlink"
	accountlinkhttp "github.com/dujiao-next/internal/modules/order/transport/http/accountlink"
	siteconnectiondomain "github.com/dujiao-next/internal/modules/siteconnection/domain"
	"github.com/dujiao-next/internal/upstream"
)

func NewHandler(c *container.Container) *accountlinkhttp.Handler {
	var connID uint
	if raw := strings.TrimSpace(os.Getenv("STOREFRONT_ACCOUNT_LINK_CONNECTION_ID")); raw != "" {
		if parsed, err := strconv.ParseUint(raw, 10, strconv.IntSize); err == nil && parsed > 0 {
			connID = uint(parsed)
		}
	}
	return accountlinkhttp.New(accountlink.New(upstreamBridge{c: c, configuredConnectionID: connID}))
}

type upstreamBridge struct {
	c                      *container.Container
	configuredConnectionID uint
}

func (b upstreamBridge) resolveAdapter() (*upstream.DujiaoNextAdapter, bool, error) {
	if b.c == nil || b.c.SiteConnectionService == nil {
		return nil, false, nil
	}
	var conn *siteconnectiondomain.Connection
	if b.configuredConnectionID > 0 {
		found, err := b.c.SiteConnectionService.GetByID(b.configuredConnectionID)
		if err != nil {
			return nil, false, accountlink.ErrUnavailable
		}
		if found == nil || found.Status != constants.ConnectionStatusActive || found.Protocol != constants.ConnectionProtocolDujiaoNext {
			return nil, false, nil
		}
		conn = found
	} else {
		if b.c.SiteConnectionRepo == nil {
			return nil, false, nil
		}
		active, err := b.c.SiteConnectionRepo.ListActive()
		if err != nil {
			return nil, false, accountlink.ErrUnavailable
		}
		var matches []siteconnectiondomain.Connection
		for _, item := range active {
			if item.Protocol == constants.ConnectionProtocolDujiaoNext {
				matches = append(matches, item)
			}
		}
		if len(matches) == 0 {
			return nil, false, nil
		}
		if len(matches) > 1 {
			// Ambiguous upstream connections must fail closed unless explicitly pinned.
			return nil, false, accountlink.ErrUnavailable
		}
		conn = &matches[0]
	}

	adapter, err := b.c.SiteConnectionService.GetAdapter(conn)
	if err != nil {
		return nil, false, accountlink.ErrUnavailable
	}
	dujiaoAdapter, ok := adapter.(*upstream.DujiaoNextAdapter)
	if !ok {
		return nil, false, accountlink.ErrUnavailable
	}
	return dujiaoAdapter, true, nil
}

func (b upstreamBridge) Status(ctx context.Context, merchantCustomerID string) (accountlink.Status, error) {
	adapter, enabled, err := b.resolveAdapter()
	if err != nil {
		return accountlink.Status{}, err
	}
	if !enabled {
		return accountlink.Status{Enabled: false, Linked: false, Status: "disabled"}, nil
	}
	res, err := adapter.CustomerLinkStatus(ctx, merchantCustomerID)
	if err != nil {
		return accountlink.Status{}, translateUpstreamError(err)
	}
	return accountlink.Status{
		Enabled:       res.Enabled,
		Linked:        res.Linked,
		Status:        res.Status,
		NativeUserID:  res.NativeUserID,
		PendingOrders: res.PendingOrders,
		UpdatedAt:     res.UpdatedAt,
	}, nil
}

func (b upstreamBridge) Start(ctx context.Context, merchantCustomerID, nextPath string) (accountlink.Challenge, error) {
	adapter, enabled, err := b.resolveAdapter()
	if err != nil {
		return accountlink.Challenge{}, err
	}
	if !enabled {
		return accountlink.Challenge{}, accountlink.ErrUnavailable
	}
	res, err := adapter.StartCustomerLink(ctx, merchantCustomerID, nextPath)
	if err != nil {
		return accountlink.Challenge{}, translateUpstreamError(err)
	}
	return accountlink.Challenge{
		ChallengeID:  res.ChallengeID,
		AuthorizeURL: res.AuthorizeURL,
		ExpiresAt:    res.ExpiresAt,
	}, nil
}

func (b upstreamBridge) Complete(ctx context.Context, merchantCustomerID, challengeID, approval string) (accountlink.CompleteResult, error) {
	adapter, enabled, err := b.resolveAdapter()
	if err != nil {
		return accountlink.CompleteResult{}, err
	}
	if !enabled {
		return accountlink.CompleteResult{}, accountlink.ErrUnavailable
	}
	res, err := adapter.CompleteCustomerLink(ctx, merchantCustomerID, challengeID, approval)
	if err != nil {
		return accountlink.CompleteResult{}, translateUpstreamError(err)
	}
	return accountlink.CompleteResult{
		Linked:         res.Linked,
		Status:         res.Status,
		NativeUserID:   res.NativeUserID,
		NextPath:       res.NextPath,
		AwakenedOrders: res.AwakenedOrders,
	}, nil
}

func translateUpstreamError(err error) error {
	switch {
	case errors.Is(err, upstream.ErrCustomerLinkInvalid):
		return accountlink.ErrInvalid
	case errors.Is(err, upstream.ErrCustomerLinkExpired):
		return accountlink.ErrExpired
	case errors.Is(err, upstream.ErrCustomerLinkForbidden):
		return accountlink.ErrForbidden
	case errors.Is(err, upstream.ErrCustomerLinkConflict):
		return accountlink.ErrConflict
	default:
		return accountlink.ErrUnavailable
	}
}
