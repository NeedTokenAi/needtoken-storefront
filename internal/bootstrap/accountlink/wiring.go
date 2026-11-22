package accountlinkwiring

import (
	"context"
	"encoding/json"
	"errors"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/dujiao-next/internal/app/container"
	"github.com/dujiao-next/internal/cache"
	"github.com/dujiao-next/internal/constants"
	"github.com/dujiao-next/internal/modules/order/application/accountlink"
	accountlinkhttp "github.com/dujiao-next/internal/modules/order/transport/http/accountlink"
	siteconnectiondomain "github.com/dujiao-next/internal/modules/siteconnection/domain"
	"github.com/dujiao-next/internal/telegramidentity"
	"github.com/dujiao-next/internal/upstream"
)

func NewHandler(c *container.Container) *accountlinkhttp.Handler {
	var connID uint
	if raw := strings.TrimSpace(os.Getenv("STOREFRONT_ACCOUNT_LINK_CONNECTION_ID")); raw != "" {
		if parsed, err := strconv.ParseUint(raw, 10, strconv.IntSize); err == nil && parsed > 0 {
			connID = uint(parsed)
		}
	}
	reader := memberIdentityReader{c: c}
	svc := accountlink.New(upstreamBridge{c: c, configuredConnectionID: connID}).
		WithIdentityReader(reader).
		WithSessionRevoker(reader)
	return accountlinkhttp.New(svc)
}

type memberIdentityReader struct {
	c *container.Container
}

func (r memberIdentityReader) LookupMember(_ context.Context, userID uint) (accountlink.MemberIdentity, error) {
	if r.c == nil || r.c.UserStore == nil {
		return accountlink.MemberIdentity{}, accountlink.ErrUnavailable
	}
	user, err := r.c.UserStore.GetByID(userID)
	if err != nil {
		return accountlink.MemberIdentity{}, err
	}
	if user == nil {
		return accountlink.MemberIdentity{UserID: userID, Active: false}, nil
	}
	email := strings.TrimSpace(user.Email)
	return accountlink.MemberIdentity{
		UserID:        user.ID,
		Email:         email,
		EmailVerified: user.EmailVerifiedAt != nil && email != "" && !telegramidentity.IsPlaceholderEmail(email),
		DisplayName:   strings.TrimSpace(user.DisplayName),
		Active:        user.Status == constants.UserStatusActive,
	}, nil
}

func (r memberIdentityReader) RevokeUserSessions(ctx context.Context, userID uint) error {
	if r.c == nil || r.c.UserStore == nil {
		return nil
	}
	user, err := r.c.UserStore.GetByID(userID)
	if err != nil {
		return err
	}
	if user == nil {
		return nil
	}
	now := time.Now()
	user.TokenVersion++
	user.TokenInvalidBefore = &now
	user.UpdatedAt = now
	if err := r.c.UserStore.Update(user); err != nil {
		return err
	}
	_ = cache.SetUserAuthState(ctx, cache.BuildUserAuthState(user))
	return nil
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

func (b upstreamBridge) ConsoleSSO(ctx context.Context, input accountlink.ConsoleSSOInput) (accountlink.ConsoleSSOResult, error) {
	adapter, enabled, err := b.resolveAdapter()
	if err != nil {
		return accountlink.ConsoleSSOResult{}, err
	}
	if !enabled {
		return accountlink.ConsoleSSOResult{}, accountlink.ErrUnavailable
	}
	res, err := adapter.ConsoleSSOCustomerLink(ctx, upstream.CustomerConsoleSSOInput{
		MerchantCustomerID: input.MerchantCustomerID,
		Email:              input.Email,
		EmailVerified:      input.EmailVerified,
		DisplayName:        input.DisplayName,
		TargetPath:         input.TargetPath,
		NextPath:           input.NextPath,
	})
	if err != nil {
		return accountlink.ConsoleSSOResult{}, translateUpstreamError(err)
	}
	return accountlink.ConsoleSSOResult{
		Mode:           res.Mode,
		RedirectURL:    res.RedirectURL,
		Linked:         res.Linked,
		NativeUserID:   res.NativeUserID,
		AutoCreated:    res.AutoCreated,
		AwakenedOrders: res.AwakenedOrders,
	}, nil
}

func (b upstreamBridge) AutoLink(ctx context.Context, input accountlink.AutoLinkInput) (accountlink.AutoLinkResult, error) {
	adapter, enabled, err := b.resolveAdapter()
	if err != nil {
		return accountlink.AutoLinkResult{}, err
	}
	if !enabled {
		return accountlink.AutoLinkResult{Linked: false, Status: "disabled"}, nil
	}
	res, err := adapter.AutoLink(ctx, upstream.CustomerAutoLinkInput{
		StorefrontUserID:   input.MerchantCustomerID,
		MerchantCustomerID: input.MerchantCustomerID,
		Email:              input.Email,
		EmailVerified:      input.EmailVerified,
		DisplayName:        input.DisplayName,
	})
	if err != nil {
		return accountlink.AutoLinkResult{}, translateUpstreamError(err)
	}
	return accountlink.AutoLinkResult{
		Linked:         res.Linked,
		Status:         res.Status,
		NativeUserID:   res.NativeUserID,
		AwakenedOrders: res.AwakenedOrders,
	}, nil
}

func (b upstreamBridge) APISummary(ctx context.Context, merchantCustomerID string) (accountlink.APISummaryResult, error) {
	adapter, enabled, err := b.resolveAdapter()
	if err != nil {
		return accountlink.APISummaryResult{}, err
	}
	if !enabled {
		return accountlink.APISummaryResult{Linked: false}, nil
	}
	res, err := adapter.APISummary(ctx, merchantCustomerID)
	if err != nil {
		return accountlink.APISummaryResult{}, translateUpstreamError(err)
	}
	subs := make([]accountlink.APISubscriptionSummary, 0, len(res.ActiveSubscriptions))
	for _, sub := range res.ActiveSubscriptions {
		subs = append(subs, accountlink.APISubscriptionSummary{
			ID:                  sub.ID,
			PlanID:              sub.PlanID,
			PlanTitle:           sub.PlanTitle,
			AllowedGroups:       sub.AllowedGroups,
			Status:              sub.Status,
			AmountTotal:         sub.AmountTotal,
			AmountUsed:          sub.AmountUsed,
			StartTime:           sub.StartTime,
			EndTime:             sub.EndTime,
			AllowWalletOverflow: sub.AllowWalletOverflow,
		})
	}
	return accountlink.APISummaryResult{
		Linked:              res.Linked,
		NativeUserID:        res.NativeUserID,
		Quota:               res.Quota,
		UsedQuota:           res.UsedQuota,
		RequestCount:        res.RequestCount,
		Group:               res.Group,
		ActiveSubscriptions: subs,
	}, nil
}

func (b upstreamBridge) StorefrontLogout(ctx context.Context, merchantCustomerID, reason string) (bool, error) {
	adapter, enabled, err := b.resolveAdapter()
	if err != nil {
		return false, err
	}
	if !enabled {
		return false, nil
	}
	revoked, err := adapter.StorefrontLogout(ctx, merchantCustomerID, reason)
	if err != nil {
		return false, translateUpstreamError(err)
	}
	return revoked, nil
}

func (b upstreamBridge) GatewayConsoleRequest(ctx context.Context, method, merchantCustomerID, subPath string, query url.Values, rawBody []byte) (json.RawMessage, error) {
	adapter, enabled, err := b.resolveAdapter()
	if err != nil {
		return nil, err
	}
	if !enabled {
		return nil, accountlink.ErrUnavailable
	}
	data, err := adapter.CustomerConsoleRequest(ctx, method, merchantCustomerID, subPath, query, rawBody)
	if err != nil {
		return nil, translateUpstreamError(err)
	}
	return data, nil
}

func translateUpstreamError(err error) error {
	switch {
	case errors.Is(err, upstream.ErrCustomerLinkNotLinked):
		return accountlink.ErrNotLinked
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
