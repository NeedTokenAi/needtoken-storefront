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
	giftcardcontract "github.com/dujiao-next/internal/modules/giftcard/contract"
	"github.com/dujiao-next/internal/modules/order/application/accountlink"
	accountlinkhttp "github.com/dujiao-next/internal/modules/order/transport/http/accountlink"
	siteconnectiondomain "github.com/dujiao-next/internal/modules/siteconnection/domain"
	walletcontract "github.com/dujiao-next/internal/modules/wallet/contract"
	"github.com/dujiao-next/internal/shared/money"
	"github.com/dujiao-next/internal/telegramidentity"
	"github.com/dujiao-next/internal/upstream"
	"github.com/shopspring/decimal"
)

func NewService(c *container.Container) *accountlink.Service {
	var connID uint
	if raw := strings.TrimSpace(os.Getenv("STOREFRONT_ACCOUNT_LINK_CONNECTION_ID")); raw != "" {
		if parsed, err := strconv.ParseUint(raw, 10, strconv.IntSize); err == nil && parsed > 0 {
			connID = uint(parsed)
		}
	}
	reader := memberIdentityReader{c: c}
	return accountlink.New(upstreamBridge{c: c, configuredConnectionID: connID}).
		WithIdentityReader(reader).
		WithSessionRevoker(reader)
}

func NewHandler(c *container.Container) *accountlinkhttp.Handler {
	return accountlinkhttp.New(NewService(c))
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
	case errors.Is(err, upstream.ErrUniversalGiftCardNotFound),
		errors.Is(err, upstream.ErrUniversalGiftCardRedeemed),
		errors.Is(err, upstream.ErrUniversalGiftCardInProgress),
		errors.Is(err, upstream.ErrUniversalGiftCardDisabled),
		errors.Is(err, upstream.ErrUniversalGiftCardExpired):
		return err
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

const needTokenQuotaPerUSD int64 = 500000

// UnifiedBalanceAdapter bridges storefront wallet and gift card operations to
// the authoritative NeedToken API User.Quota ledger via Glue.
type UnifiedBalanceAdapter struct {
	c      *container.Container
	svc    *accountlink.Service
	bridge upstreamBridge
}

func NewUnifiedBalanceBridge(c *container.Container) *UnifiedBalanceAdapter {
	var connID uint
	if raw := strings.TrimSpace(os.Getenv("STOREFRONT_ACCOUNT_LINK_CONNECTION_ID")); raw != "" {
		if parsed, err := strconv.ParseUint(raw, 10, strconv.IntSize); err == nil && parsed > 0 {
			connID = uint(parsed)
		}
	}
	bridge := upstreamBridge{c: c, configuredConnectionID: connID}
	reader := memberIdentityReader{c: c}
	svc := accountlink.New(bridge).
		WithIdentityReader(reader).
		WithSessionRevoker(reader)
	return &UnifiedBalanceAdapter{c: c, svc: svc, bridge: bridge}
}

func (a *UnifiedBalanceAdapter) Enabled() bool {
	if a == nil || a.c == nil || a.svc == nil {
		return false
	}
	_, enabled, err := a.bridge.resolveAdapter()
	return err == nil && enabled
}

func (a *UnifiedBalanceAdapter) SiteCurrency() string {
	if a == nil || a.c == nil || a.c.SettingService == nil {
		return constants.SiteCurrencyDefault
	}
	currency, err := a.c.SettingService.GetSiteCurrency(constants.SiteCurrencyDefault)
	if err != nil || strings.TrimSpace(currency) == "" {
		return constants.SiteCurrencyDefault
	}
	return strings.ToUpper(strings.TrimSpace(currency))
}

// ResolveUSDExchangeRate reads the official USD->CNY rate configured in
// site_config.needtoken_pricing.usd_cny_rate instead of abusing SiteConnection.ExchangeRate.
func ResolveUSDExchangeRate(c *container.Container, targetCurrency string) (decimal.Decimal, bool) {
	curr := strings.ToUpper(strings.TrimSpace(targetCurrency))
	if curr == "" || curr == "USD" {
		return decimal.NewFromInt(1), true
	}
	if c != nil && c.SettingService != nil {
		if rawMap, err := c.SettingService.GetByKey(constants.SettingKeySiteConfig); err == nil && rawMap != nil {
			if pricing, ok := rawMap["needtoken_pricing"].(map[string]interface{}); ok {
				if rawRate, ok := pricing["usd_cny_rate"]; ok {
					rateStr := strings.TrimSpace(asString(rawRate))
					if parsed, parseErr := decimal.NewFromString(rateStr); parseErr == nil && parsed.GreaterThan(decimal.Zero) {
						return parsed, true
					}
				}
			}
		}
	}
	return decimal.Zero, false
}

func asString(v interface{}) string {
	switch val := v.(type) {
	case string:
		return val
	case float64:
		return strconv.FormatFloat(val, 'f', -1, 64)
	case int:
		return strconv.Itoa(val)
	case int64:
		return strconv.FormatInt(val, 10)
	default:
		return ""
	}
}

func (a *UnifiedBalanceAdapter) quotaToCurrencyAmount(quota int64, currency string) money.Amount {
	if quota <= 0 {
		return money.FromDecimal(decimal.Zero)
	}
	usd := decimal.NewFromInt(quota).Div(decimal.NewFromInt(needTokenQuotaPerUSD))
	rate, ok := ResolveUSDExchangeRate(a.c, currency)
	if !ok {
		rate = decimal.NewFromInt(1)
	}
	// Truncate down to cents so tiny API charges (e.g. $0.00072 -> $99.99928 -> 666.995 CNY)
	// never round back up to 667.00 CNY and mask deductions or overstate cash value.
	return money.FromDecimal(usd.Mul(rate).Truncate(2))
}

func (a *UnifiedBalanceAdapter) currencyAmountToQuotaDecimal(amount money.Amount, currency string) (decimal.Decimal, error) {
	dec := amount.Decimal.Round(2)
	if dec.LessThanOrEqual(decimal.Zero) {
		return decimal.Zero, nil
	}
	rate, ok := ResolveUSDExchangeRate(a.c, currency)
	if !ok || rate.LessThanOrEqual(decimal.Zero) {
		if strings.EqualFold(strings.TrimSpace(currency), "USD") {
			rate = decimal.NewFromInt(1)
		} else {
			return decimal.Zero, errors.New("needtoken_pricing.usd_cny_rate is required for non-USD unified balance conversion")
		}
	}
	usd := dec.Div(rate)
	return usd.Mul(decimal.NewFromInt(needTokenQuotaPerUSD)), nil
}

// currencyAmountToCreditQuota floors fractional quota when crediting so conversions never over-issue API quota.
func (a *UnifiedBalanceAdapter) currencyAmountToCreditQuota(amount money.Amount, currency string) (int64, error) {
	quotaDec, err := a.currencyAmountToQuotaDecimal(amount, currency)
	if err != nil || quotaDec.LessThanOrEqual(decimal.Zero) {
		return 0, err
	}
	return quotaDec.Floor().IntPart(), nil
}

// currencyAmountToDebitQuota ceils fractional quota when debiting so deductions never under-charge.
func (a *UnifiedBalanceAdapter) currencyAmountToDebitQuota(amount money.Amount, currency string) (int64, error) {
	quotaDec, err := a.currencyAmountToQuotaDecimal(amount, currency)
	if err != nil || quotaDec.LessThanOrEqual(decimal.Zero) {
		return 0, err
	}
	return quotaDec.Ceil().IntPart(), nil
}

func (a *UnifiedBalanceAdapter) FetchBalance(ctx context.Context, userID uint, siteCurrency string) (money.Amount, bool, error) {
	if !a.Enabled() || userID == 0 {
		return money.Amount{}, false, nil
	}
	summary, err := a.svc.APISummary(ctx, userID)
	if err != nil {
		return money.Amount{}, false, err
	}
	if !summary.Linked {
		ensured, ensureErr := a.svc.Ensure(ctx, userID)
		if ensureErr != nil || !ensured.Linked {
			return money.Amount{}, false, nil
		}
		summary, err = a.svc.APISummary(ctx, userID)
		if err != nil || !summary.Linked {
			return money.Amount{}, false, err
		}
	}
	curr := strings.TrimSpace(siteCurrency)
	if curr == "" {
		curr = a.SiteCurrency()
	}
	return a.quotaToCurrencyAmount(summary.Quota, curr), true, nil
}

func (a *UnifiedBalanceAdapter) Credit(ctx context.Context, userID uint, amount money.Amount, currency, operationID, remark string) (money.Amount, error) {
	if !a.Enabled() || userID == 0 {
		return money.Amount{}, nil
	}
	curr := strings.TrimSpace(currency)
	if curr == "" {
		curr = a.SiteCurrency()
	}
	quota, err := a.currencyAmountToCreditQuota(amount, curr)
	if err != nil || quota <= 0 {
		return money.Amount{}, err
	}
	body, err := json.Marshal(map[string]any{
		"operation_id": strings.TrimSpace(operationID),
		"quota":        quota,
		"remark":       strings.TrimSpace(remark),
	})
	if err != nil {
		return money.Amount{}, err
	}
	raw, err := a.svc.GatewayConsole(ctx, userID, "POST", "/wallet/credit", nil, body)
	if err != nil {
		return money.Amount{}, err
	}
	var resp struct {
		BalanceQuota int64 `json:"balance_quota"`
	}
	_ = json.Unmarshal(raw, &resp)
	return a.quotaToCurrencyAmount(resp.BalanceQuota, curr), nil
}

func (a *UnifiedBalanceAdapter) Debit(ctx context.Context, userID uint, maxAmount money.Amount, currency string, allowPartial bool, operationID, remark string) (money.Amount, money.Amount, bool, error) {
	if !a.Enabled() || userID == 0 {
		return money.FromDecimal(decimal.Zero), money.FromDecimal(decimal.Zero), false, nil
	}
	curr := strings.TrimSpace(currency)
	if curr == "" {
		curr = a.SiteCurrency()
	}
	maxQuota, err := a.currencyAmountToDebitQuota(maxAmount, curr)
	if err != nil || maxQuota <= 0 {
		return money.FromDecimal(decimal.Zero), money.FromDecimal(decimal.Zero), false, err
	}
	body, err := json.Marshal(map[string]any{
		"operation_id":  strings.TrimSpace(operationID),
		"quota":         maxQuota,
		"allow_partial": allowPartial,
		"remark":        strings.TrimSpace(remark),
	})
	if err != nil {
		return money.FromDecimal(decimal.Zero), money.FromDecimal(decimal.Zero), false, err
	}
	raw, err := a.svc.GatewayConsole(ctx, userID, "POST", "/wallet/debit", nil, body)
	if err != nil {
		return money.FromDecimal(decimal.Zero), money.FromDecimal(decimal.Zero), false, err
	}
	var resp struct {
		AppliedQuota int64 `json:"applied_quota"`
		BalanceQuota int64 `json:"balance_quota"`
		Replayed     bool  `json:"replayed"`
	}
	if err := json.Unmarshal(raw, &resp); err != nil {
		return money.FromDecimal(decimal.Zero), money.FromDecimal(decimal.Zero), false, err
	}
	debited := a.quotaToCurrencyAmount(resp.AppliedQuota, curr)
	if debited.Decimal.GreaterThan(maxAmount.Decimal.Round(2)) {
		debited = money.FromDecimal(maxAmount.Decimal.Round(2))
	}
	balanceAfter := a.quotaToCurrencyAmount(resp.BalanceQuota, curr)
	return debited, balanceAfter, resp.Replayed, nil
}

func (a *UnifiedBalanceAdapter) Redeem(ctx context.Context, userID uint, code, siteCurrency string) (*walletcontract.UnifiedBalanceRedeemResult, error) {
	if !a.Enabled() || userID == 0 {
		return nil, accountlink.ErrUnavailable
	}
	curr := strings.TrimSpace(siteCurrency)
	if curr == "" {
		curr = a.SiteCurrency()
	}
	body, err := json.Marshal(map[string]string{
		"code": strings.TrimSpace(code),
	})
	if err != nil {
		return nil, err
	}
	raw, err := a.svc.GatewayConsole(ctx, userID, "POST", "/wallet/redeem", nil, body)
	if err != nil {
		return nil, err
	}
	var resp struct {
		Source        string `json:"source"`
		CreditedQuota int64  `json:"credited_quota"`
		BalanceQuota  int64  `json:"balance_quota"`
	}
	if err := json.Unmarshal(raw, &resp); err != nil || resp.CreditedQuota <= 0 {
		return nil, accountlink.ErrInvalid
	}
	return &walletcontract.UnifiedBalanceRedeemResult{
		Source:       strings.TrimSpace(resp.Source),
		Credited:     a.quotaToCurrencyAmount(resp.CreditedQuota, curr),
		BalanceAfter: a.quotaToCurrencyAmount(resp.BalanceQuota, curr),
		Currency:     curr,
	}, nil
}

func (a *UnifiedBalanceAdapter) RedeemCode(userID uint, code, siteCurrency string) (money.Amount, money.Amount, string, error) {
	res, err := a.Redeem(context.Background(), userID, code, siteCurrency)
	if err != nil {
		switch {
		case errors.Is(err, upstream.ErrUniversalGiftCardRedeemed):
			return money.Amount{}, money.Amount{}, "", giftcardcontract.ErrRedeemed
		case errors.Is(err, upstream.ErrUniversalGiftCardInProgress):
			return money.Amount{}, money.Amount{}, "", giftcardcontract.ErrInProgress
		case errors.Is(err, upstream.ErrUniversalGiftCardDisabled):
			return money.Amount{}, money.Amount{}, "", giftcardcontract.ErrDisabled
		case errors.Is(err, upstream.ErrUniversalGiftCardExpired):
			return money.Amount{}, money.Amount{}, "", giftcardcontract.ErrExpired
		case errors.Is(err, upstream.ErrUniversalGiftCardNotFound), errors.Is(err, accountlink.ErrInvalid):
			return money.Amount{}, money.Amount{}, "", giftcardcontract.ErrNotFound
		default:
			return money.Amount{}, money.Amount{}, "", giftcardcontract.ErrFetchFailed
		}
	}
	return res.Credited, res.BalanceAfter, res.Source, nil
}

func (a *UnifiedBalanceAdapter) SyncLocalCredit(userID uint, amount money.Amount, currency, operationID, remark string) error {
	if a.c != nil && a.c.WalletService != nil {
		a.c.WalletService.SyncCreditToUnified(userID, amount, currency, operationID, remark)
		return nil
	}
	_, err := a.Credit(context.Background(), userID, amount, currency, operationID, remark)
	return err
}

var (
	_ walletcontract.UnifiedBalanceBridge      = (*UnifiedBalanceAdapter)(nil)
	_ giftcardcontract.UnifiedGiftCardRedeemer = (*UnifiedBalanceAdapter)(nil)
)
