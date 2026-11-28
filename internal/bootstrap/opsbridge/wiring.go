package opsbridge

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/dujiao-next/internal/app/container"
	"github.com/dujiao-next/internal/cache"
	"github.com/dujiao-next/internal/config"
	"github.com/dujiao-next/internal/constants"
	affiliateapp "github.com/dujiao-next/internal/modules/affiliate/application"
	categoryapp "github.com/dujiao-next/internal/modules/catalog/category/application"
	categorydomain "github.com/dujiao-next/internal/modules/catalog/category/domain"
	mappingapp "github.com/dujiao-next/internal/modules/catalog/mapping/application"
	mappingcontract "github.com/dujiao-next/internal/modules/catalog/mapping/contract"
	mappingdomain "github.com/dujiao-next/internal/modules/catalog/mapping/domain"
	productadmin "github.com/dujiao-next/internal/modules/catalog/product/application/admin"
	productcontract "github.com/dujiao-next/internal/modules/catalog/product/contract"
	productgormstore "github.com/dujiao-next/internal/modules/catalog/product/store/gormstore"
	usercontract "github.com/dujiao-next/internal/modules/identity/user/contract"
	opsapp "github.com/dujiao-next/internal/modules/opsbridge/application"
	opscontract "github.com/dujiao-next/internal/modules/opsbridge/contract"
	opstransport "github.com/dujiao-next/internal/modules/opsbridge/transport/http"
	orderrefund "github.com/dujiao-next/internal/modules/order/application/refund"
	ordercontract "github.com/dujiao-next/internal/modules/order/contract"
	orderdomain "github.com/dujiao-next/internal/modules/order/domain"
	paymentapp "github.com/dujiao-next/internal/modules/payment/application"
	paymentcontract "github.com/dujiao-next/internal/modules/payment/contract"
	paymentdomain "github.com/dujiao-next/internal/modules/payment/domain"
	resellerapp "github.com/dujiao-next/internal/modules/reseller/application"
	resellercontract "github.com/dujiao-next/internal/modules/reseller/contract"
	resellerdomain "github.com/dujiao-next/internal/modules/reseller/domain"
	resellergormstore "github.com/dujiao-next/internal/modules/reseller/infrastructure/gormstore"
	settingsapp "github.com/dujiao-next/internal/modules/settings/application"
	settingscontract "github.com/dujiao-next/internal/modules/settings/contract"
	settingsintegration "github.com/dujiao-next/internal/modules/settings/schema/integration"
	siteconnectionapp "github.com/dujiao-next/internal/modules/siteconnection/application"
	siteconnectioncontract "github.com/dujiao-next/internal/modules/siteconnection/contract"
	siteconnectiondomain "github.com/dujiao-next/internal/modules/siteconnection/domain"
	walletapp "github.com/dujiao-next/internal/modules/wallet/application"
	walletcontract "github.com/dujiao-next/internal/modules/wallet/contract"
	walletdomain "github.com/dujiao-next/internal/modules/wallet/domain"
	"github.com/dujiao-next/internal/shared/jsonmap"
	"github.com/dujiao-next/internal/shared/jsonslice"
	"github.com/dujiao-next/internal/shared/money"
	"github.com/dujiao-next/internal/upstream"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

func NewHandler(c *container.Container) *opstransport.Handler {
	svc := opsapp.NewWithCatalog(
		c.OrderStore,
		c.PaymentStore,
		c.ProcurementOrderRepo,
		c.CategoryRepo,
		c.ProductRepo,
		c.ProductSKURepo,
		c.CouponRepo,
	)
	if c.CategoryService != nil {
		svc.WithCategoryStore(&categoryStoreAdapter{svc: c.CategoryService})
	}
	if c.ProductAdminService != nil {
		svc.WithProductDeleteStore(&productDeleteStoreAdapter{svc: c.ProductAdminService})
	}
	if c.CardSecretRepo != nil {
		svc.WithCardSecretStock(c.CardSecretRepo)
	}
	if c.ProductRepo != nil && c.ProductSKURepo != nil {
		svc.WithSKUDeleteUoW(&skuDeleteUoWAdapter{
			products: c.ProductRepo,
			skus:     c.ProductSKURepo,
		})
	}
	if c.SettingService != nil || c.SiteConnectionService != nil {
		svc.WithPricing(&pricingStoreAdapter{
			settings:    c.SettingService,
			connections: c.SiteConnectionService,
		})
	}
	if c.GiftCardRepo != nil {
		svc.WithGiftCards(c.GiftCardRepo)
	}
	if c.WalletRepo != nil {
		svc.WithUniversalGiftCardClaims(&universalGiftCardClaimAdapter{wallets: c.WalletRepo})
	}
	if c.PaymentChannelStore != nil || c.SettingService != nil {
		var defaultEmail config.EmailConfig
		if c.Config != nil {
			defaultEmail = c.Config.Email
		}
		svc.WithAcquiring(&acquiringStoreAdapter{
			channels:     c.PaymentChannelStore,
			payments:     c.PaymentService,
			settings:     c.SettingService,
			defaultEmail: defaultEmail,
		})
	}
	if c.SiteConnectionService != nil && c.ProductMappingService != nil {
		svc.WithUpstream(&upstreamStoreAdapter{
			connections: c.SiteConnectionService,
			mappings:    c.ProductMappingService,
			mappingRepo: c.ProductMappingRepo,
		})
	}
	if c.OrderStore != nil && c.OrderRefundService != nil {
		svc.WithOrderRefunds(&orderRefundStoreAdapter{
			orders:  c.OrderStore,
			refunds: c.OrderRefundService,
			wallets: c.WalletService,
		})
	}
	if c.UserStore != nil {
		svc.WithUserSessionRevoker(&userSessionRevokerAdapter{users: c.UserStore})
		svc.WithUsers(&userStoreAdapter{
			users:    c.UserStore,
			wallets:  c.WalletService,
			oauth:    c.ExternalIdentityStore,
			userAuth: c.UserAuthService,
			settings: c.SettingService,
		})
		svc.WithOrderEnricher(&orderEnricherAdapter{users: c.UserStore})
	}
	if c.OrderStore != nil || c.ProcurementOrderService != nil {
		svc.WithOrderActions(&orderActionStoreAdapter{
			orders:       c.OrderStore,
			orderService: c.OrderService,
			payments:     c.PaymentStore,
			paymentSvc:   c.PaymentService,
			fulfillments: c.FulfillmentService,
			procurements: c.ProcurementOrderService,
		})
	}
	if c.RadarSubscriptionRepo != nil && c.UserStore != nil {
		svc.WithRadarSubscriptions(&radarSubscriptionAdapter{
			radar: c.RadarSubscriptionRepo,
			users: c.UserStore,
		})
	}
	if c.WalletService != nil {
		svc.WithUnifiedOps(&unifiedOpReaderAdapter{wallets: c.WalletService})
	}
	svc.WithGrowth(&growthStoreAdapter{
		cfg:              c.Config,
		settings:         c.SettingService,
		affiliate:        c.AffiliateService,
		wallets:          c.WalletRepo,
		resellerMgmt:     c.ResellerManagementService,
		resellerWithdraw: c.ResellerAccountingWithdraw,
		resellerStore:    c.ResellerStore,
	})
	return opstransport.New(svc)
}

type categoryStoreAdapter struct {
	svc *categoryapp.Service
}

func mapCategoryModel(c categorydomain.Category) *opscontract.Category {
	name := map[string]string{}
	for k, v := range c.NameJSON {
		if str, ok := v.(string); ok {
			name[k] = str
		}
	}
	return &opscontract.Category{
		ID:        c.ID,
		ParentID:  c.ParentID,
		Slug:      c.Slug,
		Name:      name,
		Icon:      c.Icon,
		SortOrder: c.SortOrder,
		IsActive:  c.IsActive,
		CreatedAt: c.CreatedAt,
	}
}

func translateCategoryError(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, categoryapp.ErrNotFound):
		return opscontract.ErrNotFound
	case errors.Is(err, categoryapp.ErrParentInvalid):
		return opscontract.ErrInvalidInput
	case errors.Is(err, categoryapp.ErrSlugExists):
		return opscontract.ErrConflict
	case errors.Is(err, categoryapp.ErrInUse):
		return opscontract.ErrCategoryInUse
	default:
		return err
	}
}

func (a *categoryStoreAdapter) Create(parentID uint, slug string, name map[string]any, icon string, sortOrder int, isActive bool) (*opscontract.Category, error) {
	if a.svc == nil {
		return nil, opscontract.ErrNotConfigured
	}
	created, err := a.svc.Create(categoryapp.UpsertInput{
		ParentID:  parentID,
		Slug:      slug,
		NameJSON:  name,
		Icon:      icon,
		SortOrder: sortOrder,
	})
	if err != nil {
		return nil, translateCategoryError(err)
	}
	if !isActive {
		updated, activeErr := a.svc.SetActive(strconv.FormatUint(uint64(created.ID), 10), false)
		if activeErr != nil {
			return nil, translateCategoryError(activeErr)
		}
		created = updated
	}
	return mapCategoryModel(*created), nil
}

func (a *categoryStoreAdapter) Update(id string, parentID uint, slug string, name map[string]any, icon string, sortOrder int, isActive *bool) (*opscontract.Category, error) {
	if a.svc == nil {
		return nil, opscontract.ErrNotConfigured
	}
	updated, err := a.svc.Update(id, categoryapp.UpsertInput{
		ParentID:  parentID,
		Slug:      slug,
		NameJSON:  name,
		Icon:      icon,
		SortOrder: sortOrder,
	})
	if err != nil {
		return nil, translateCategoryError(err)
	}
	if isActive != nil && updated.IsActive != *isActive {
		updated, err = a.svc.SetActive(id, *isActive)
		if err != nil {
			return nil, translateCategoryError(err)
		}
	}
	return mapCategoryModel(*updated), nil
}

func (a *categoryStoreAdapter) Delete(id string) error {
	if a.svc == nil {
		return opscontract.ErrNotConfigured
	}
	return translateCategoryError(a.svc.Delete(id))
}

type productDeleteStoreAdapter struct {
	svc *productadmin.AdminService
}

func (a *productDeleteStoreAdapter) Delete(id string) error {
	if a.svc == nil {
		return opscontract.ErrNotConfigured
	}
	err := a.svc.Delete(id)
	switch {
	case err == nil:
		return nil
	case errors.Is(err, productcontract.ErrNotFound):
		return opscontract.ErrNotFound
	case errors.Is(err, productcontract.ErrProductHasStock):
		return opscontract.ErrProductHasStock
	case errors.Is(err, productcontract.ErrProductHasOrderRecord):
		return opscontract.ErrProductHasOrderRecord
	default:
		return err
	}
}

type skuDeleteUoWAdapter struct {
	products *productgormstore.ProductStore
	skus     *productgormstore.SKUStore
}

func (u *skuDeleteUoWAdapter) WithinTransaction(fn func(products productcontract.Repository, skus productcontract.SKURepository) error) error {
	if u.products == nil || u.skus == nil {
		return opscontract.ErrNotConfigured
	}
	return u.products.Transaction(func(tx *gorm.DB) error {
		return fn(u.products.BindTx(tx), u.skus.BindTx(tx))
	})
}

type pricingStoreAdapter struct {
	settings    *settingsapp.Service
	connections *siteconnectionapp.Service
}

func (p *pricingStoreAdapter) GetSitePricing() (string, string, string, string, error) {
	siteCurrency := constants.SiteCurrencyDefault
	displayCurrency := constants.SiteCurrencyDefault
	usdCnyRate := ""
	rateUpdatedAt := ""
	if p.settings == nil {
		return siteCurrency, displayCurrency, usdCnyRate, rateUpdatedAt, nil
	}
	cfg, err := p.settings.GetConfig(map[string]interface{}{
		constants.SettingFieldSiteCurrency: constants.SiteCurrencyDefault,
	})
	if err != nil {
		return "", "", "", "", err
	}
	if cur, ok := cfg[constants.SettingFieldSiteCurrency].(string); ok && strings.TrimSpace(cur) != "" {
		siteCurrency = strings.ToUpper(strings.TrimSpace(cur))
		displayCurrency = siteCurrency
	}
	if rawPricing, ok := cfg["needtoken_pricing"].(map[string]interface{}); ok {
		if cur, ok := rawPricing["default_display_currency"].(string); ok && strings.TrimSpace(cur) != "" {
			displayCurrency = strings.ToUpper(strings.TrimSpace(cur))
		}
		if rate, ok := rawPricing["usd_cny_rate"].(string); ok {
			usdCnyRate = strings.TrimSpace(rate)
		}
		if updated, ok := rawPricing["rate_updated_at"].(string); ok {
			rateUpdatedAt = strings.TrimSpace(updated)
		}
	}
	return siteCurrency, displayCurrency, usdCnyRate, rateUpdatedAt, nil
}

func (p *pricingStoreAdapter) UpdateSitePricing(siteCurrency, defaultDisplayCurrency, usdCnyRate *string) error {
	if p.settings == nil {
		return opscontract.ErrNotConfigured
	}
	existing, err := p.settings.GetByKey(constants.SettingKeySiteConfig)
	if err != nil {
		return err
	}
	merged := map[string]interface{}{}
	for k, v := range existing {
		merged[k] = v
	}
	if siteCurrency != nil {
		cur := strings.ToUpper(strings.TrimSpace(*siteCurrency))
		if cur == "" || len(cur) > 16 {
			return opscontract.ErrInvalidInput
		}
		merged[constants.SettingFieldSiteCurrency] = cur
	}
	pricing := map[string]interface{}{}
	if raw, ok := merged["needtoken_pricing"].(map[string]interface{}); ok {
		for k, v := range raw {
			pricing[k] = v
		}
	}
	if defaultDisplayCurrency != nil {
		pricing["default_display_currency"] = strings.ToUpper(strings.TrimSpace(*defaultDisplayCurrency))
	}
	if usdCnyRate != nil {
		pricing["usd_cny_rate"] = strings.TrimSpace(*usdCnyRate)
		pricing["rate_updated_at"] = time.Now().UTC().Format(time.RFC3339)
	}
	if len(pricing) > 0 {
		merged["needtoken_pricing"] = pricing
	}
	if _, err := p.settings.Update(constants.SettingKeySiteConfig, merged); err != nil {
		var fieldErr *settingsapp.SettingFieldError
		if errors.As(err, &fieldErr) {
			return opscontract.ErrInvalidInput
		}
		return err
	}
	return nil
}

func (p *pricingStoreAdapter) ListConnectionPricing() ([]opscontract.ConnectionPricing, error) {
	if p.connections == nil {
		return []opscontract.ConnectionPricing{}, nil
	}
	rows, _, err := p.connections.List(siteconnectioncontract.ListFilter{Page: 1, PageSize: 50})
	if err != nil {
		return nil, err
	}
	out := make([]opscontract.ConnectionPricing, 0, len(rows))
	for _, c := range rows {
		out = append(out, opscontract.ConnectionPricing{
			ID:                 c.ID,
			Name:               c.Name,
			Protocol:           c.Protocol,
			Status:             c.Status,
			ExchangeRate:       c.ExchangeRate.String(),
			PriceMarkupPercent: c.PriceMarkupPercent.String(),
			PriceRoundingMode:  c.PriceRoundingMode,
			AutoSyncPrice:      c.AutoSyncPrice,
			UpdatedAt:          c.UpdatedAt,
		})
	}
	return out, nil
}

func (p *pricingStoreAdapter) UpdateConnectionPricing(id uint, exchangeRate, markupPercent *float64, roundingMode *string, autoSyncPrice *bool) error {
	if p.connections == nil {
		return opscontract.ErrNotConfigured
	}
	_, err := p.connections.Update(id, siteconnectionapp.UpdateInput{
		ExchangeRate:       exchangeRate,
		PriceMarkupPercent: markupPercent,
		PriceRoundingMode:  roundingMode,
		AutoSyncPrice:      autoSyncPrice,
	})
	if errors.Is(err, siteconnectioncontract.ErrNotFound) {
		return opscontract.ErrNotFound
	}
	return err
}

type acquiringStoreAdapter struct {
	channels     paymentcontract.ChannelStore
	payments     *paymentapp.PaymentService
	settings     *settingsapp.Service
	defaultEmail config.EmailConfig
}

var safeChannelSummaryKeys = []string{
	"app_id", "mch_id", "merchant_id", "pid", "client_id",
	"api_url", "gateway_url", "notify_url", "return_url",
	"trade_type", "order_mode", "payment_environment",
}

func mapAcquiringChannel(ch paymentdomain.PaymentChannel) opscontract.AcquiringChannel {
	summary := map[string]string{}
	opsRef := ""
	if ch.ConfigJSON != nil {
		if rawRef, ok := ch.ConfigJSON["_ops_channel_ref"]; ok && rawRef != nil {
			opsRef = strings.TrimSpace(fmt.Sprint(rawRef))
		}
		for _, k := range safeChannelSummaryKeys {
			if v, ok := ch.ConfigJSON[k]; ok && v != nil {
				str := strings.TrimSpace(fmt.Sprint(v))
				if str != "" {
					summary[k] = str
				}
			}
		}
	}
	return opscontract.AcquiringChannel{
		ID:                 ch.ID,
		OpsChannelRef:      opsRef,
		Name:               ch.Name,
		Icon:               ch.Icon,
		ProviderType:       ch.ProviderType,
		ChannelType:        ch.ChannelType,
		InteractionMode:    ch.InteractionMode,
		PaymentEnvironment: ch.ResolvePaymentEnvironment(),
		FeeRate:            ch.FeeRate.String(),
		FixedFee:           ch.FixedFee.String(),
		MinAmount:          ch.MinAmount.String(),
		MaxAmount:          ch.MaxAmount.String(),
		PaymentRoles:       append([]string(nil), ch.PaymentRoles...),
		PaymentTypes:       append([]string(nil), ch.PaymentTypes...),
		ConfigSummary:      summary,
		IsActive:           ch.IsActive,
		SortOrder:          ch.SortOrder,
		UpdatedAt:          ch.UpdatedAt,
	}
}

func parseOptionalMoney(raw string) (money.Amount, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return money.FromDecimal(decimal.Zero), nil
	}
	dec, err := decimal.NewFromString(trimmed)
	if err != nil || dec.IsNegative() {
		return money.Amount{}, opscontract.ErrInvalidInput
	}
	return money.FromDecimal(dec), nil
}

func (a *acquiringStoreAdapter) ListChannels() ([]opscontract.AcquiringChannel, int64, error) {
	if a.channels == nil {
		return nil, 0, opscontract.ErrNotConfigured
	}
	rows, total, err := a.channels.List(paymentcontract.ChannelListFilter{Page: 1, PageSize: 200})
	if err != nil {
		return nil, 0, err
	}
	out := make([]opscontract.AcquiringChannel, 0, len(rows))
	for _, row := range rows {
		out = append(out, mapAcquiringChannel(row))
	}
	return out, total, nil
}

func (a *acquiringStoreAdapter) UpsertChannel(input opscontract.UpsertAcquiringChannelInput) (*opscontract.AcquiringChannel, error) {
	if a.channels == nil {
		return nil, opscontract.ErrNotConfigured
	}
	opsRef := strings.TrimSpace(input.OpsChannelRef)
	env := strings.ToLower(strings.TrimSpace(input.PaymentEnvironment))
	if env == "" {
		env = "live"
	}
	if env != "live" && env != "sandbox" {
		return nil, opscontract.ErrInvalidInput
	}
	feeRate, err := parseOptionalMoney(input.FeeRate)
	if err != nil {
		return nil, err
	}
	fixedFee, err := parseOptionalMoney(input.FixedFee)
	if err != nil {
		return nil, err
	}
	minAmount, err := parseOptionalMoney(input.MinAmount)
	if err != nil {
		return nil, err
	}
	maxAmount, err := parseOptionalMoney(input.MaxAmount)
	if err != nil {
		return nil, err
	}

	var target *paymentdomain.PaymentChannel
	if input.ID > 0 {
		existing, getErr := a.channels.GetByID(input.ID)
		if getErr != nil {
			return nil, getErr
		}
		if existing == nil {
			return nil, opscontract.ErrNotFound
		}
		target = existing
	} else if opsRef != "" {
		rows, _, listErr := a.channels.List(paymentcontract.ChannelListFilter{Page: 1, PageSize: 200})
		if listErr != nil {
			return nil, listErr
		}
		for i := range rows {
			if rows[i].ConfigJSON != nil && strings.TrimSpace(fmt.Sprint(rows[i].ConfigJSON["_ops_channel_ref"])) == opsRef {
				found := rows[i]
				target = &found
				break
			}
		}
	}

	mergedCfg := jsonmap.JSON{}
	if target != nil && target.ConfigJSON != nil {
		for k, v := range target.ConfigJSON {
			mergedCfg[k] = v
		}
	}
	for k, v := range input.ConfigJSON {
		if v == nil {
			continue
		}
		if s, ok := v.(string); ok && strings.TrimSpace(s) == "" {
			continue
		}
		mergedCfg[k] = v
	}
	if opsRef != "" {
		mergedCfg["_ops_channel_ref"] = opsRef
	}
	mergedCfg["payment_environment"] = env

	active := true
	if target != nil {
		active = target.IsActive
	}
	if input.IsActive != nil {
		active = *input.IsActive
	}

	roles := input.PaymentRoles
	if len(roles) == 0 && target != nil {
		roles = target.PaymentRoles
	}
	if len(roles) == 0 {
		roles = []string{constants.PaymentRoleMember, constants.PaymentRoleGuest}
	}
	types := input.PaymentTypes
	if len(types) == 0 && target != nil {
		types = target.PaymentTypes
	}
	if len(types) == 0 {
		types = []string{constants.PaymentTypeOrder}
	}

	now := time.Now().UTC()
	var ch paymentdomain.PaymentChannel
	if target != nil {
		ch = *target
	} else {
		ch.CreatedAt = now
	}
	ch.Name = strings.TrimSpace(input.Name)
	if strings.TrimSpace(input.Icon) != "" || target == nil {
		ch.Icon = strings.TrimSpace(input.Icon)
	}
	ch.ProviderType = strings.ToLower(strings.TrimSpace(input.ProviderType))
	ch.ChannelType = strings.ToLower(strings.TrimSpace(input.ChannelType))
	ch.InteractionMode = strings.ToLower(strings.TrimSpace(input.InteractionMode))
	ch.FeeRate = feeRate
	ch.FixedFee = fixedFee
	ch.MinAmount = minAmount
	ch.MaxAmount = maxAmount
	ch.HideAmountOutRange = input.HideAmountOutRange
	ch.PaymentRoles = jsonslice.Strings(roles)
	ch.PaymentTypes = jsonslice.Strings(types)
	ch.ConfigJSON = mergedCfg
	ch.IsActive = active
	ch.SortOrder = input.SortOrder
	ch.UpdatedAt = now

	if a.payments != nil {
		if valErr := a.payments.ValidateChannel(&ch); valErr != nil {
			return nil, opscontract.ErrInvalidInput
		}
	}
	if target != nil {
		if err := a.channels.Update(&ch); err != nil {
			return nil, err
		}
	} else {
		if err := a.channels.Create(&ch); err != nil {
			return nil, err
		}
	}
	out := mapAcquiringChannel(ch)
	return &out, nil
}

func (a *acquiringStoreAdapter) UpdateChannelStatus(id uint, active bool) (*opscontract.AcquiringChannel, error) {
	if a.channels == nil {
		return nil, opscontract.ErrNotConfigured
	}
	ch, err := a.channels.GetByID(id)
	if err != nil {
		return nil, err
	}
	if ch == nil {
		return nil, opscontract.ErrNotFound
	}
	ch.IsActive = active
	ch.UpdatedAt = time.Now().UTC()
	if err := a.channels.Update(ch); err != nil {
		return nil, err
	}
	out := mapAcquiringChannel(*ch)
	return &out, nil
}

func (a *acquiringStoreAdapter) MailHealth() (*opscontract.MailHealthResult, error) {
	now := time.Now().UTC()
	res := &opscontract.MailHealthResult{
		Source:      "storefront",
		CollectedAt: now,
		Freshness:   "fresh",
		Enabled:     a.defaultEmail.Enabled,
		Host:        strings.TrimSpace(a.defaultEmail.Host),
		Port:        a.defaultEmail.Port,
		From:        strings.TrimSpace(a.defaultEmail.From),
	}
	if a.settings != nil {
		if smtp, err := a.settings.GetSMTPSetting(a.defaultEmail); err == nil {
			res.Enabled = smtp.Enabled
			res.Host = strings.TrimSpace(smtp.Host)
			res.Port = smtp.Port
			res.From = strings.TrimSpace(smtp.From)
		}
	}
	res.Configured = res.Host != "" && res.Port > 0 && res.From != ""
	switch {
	case res.Enabled && res.Configured:
		res.Status = "ready"
	case res.Configured && !res.Enabled:
		res.Status = "disabled"
	default:
		res.Status = "unconfigured"
	}
	return res, nil
}

type upstreamMappingService interface {
	ListUpstreamCategories(connectionID uint) ([]upstream.UpstreamCategory, bool, error)
	ListUpstreamProducts(connectionID uint, page, pageSize int) (*upstream.ProductListResult, error)
	SyncProduct(mappingID uint) error
	BatchImportUpstreamProducts(connectionID uint, upstreamProductIDs []uint, categoryID uint, autoCreateCategory bool) ([]mappingapp.BatchUpstreamProductImportOutcome, error)
}

type upstreamStoreAdapter struct {
	connections *siteconnectionapp.Service
	mappings    upstreamMappingService
	mappingRepo mappingcontract.MappingRepository
}

func mapUpstreamConn(c siteconnectiondomain.Connection) opscontract.UpstreamConnection {
	return opscontract.UpstreamConnection{
		ID:                 c.ID,
		Name:               c.Name,
		BaseURL:            c.BaseURL,
		ApiKey:             c.ApiKey,
		Protocol:           c.Protocol,
		CallbackURL:        c.CallbackURL,
		Status:             c.Status,
		LastPingAt:         c.LastPingAt,
		LastPingOK:         c.LastPingOK,
		ExchangeRate:       c.ExchangeRate.StringFixed(4),
		PriceMarkupPercent: c.PriceMarkupPercent.StringFixed(2),
		PriceRoundingMode:  c.PriceRoundingMode,
		AutoSyncPrice:      c.AutoSyncPrice,
		UpdatedAt:          c.UpdatedAt.UTC(),
	}
}

func (u *upstreamStoreAdapter) ListConnections() ([]opscontract.UpstreamConnection, int64, error) {
	if u.connections == nil {
		return nil, 0, opscontract.ErrNotConfigured
	}
	rows, total, err := u.connections.List(siteconnectioncontract.ListFilter{Page: 1, PageSize: 100})
	if err != nil {
		return nil, 0, err
	}
	out := make([]opscontract.UpstreamConnection, 0, len(rows))
	for _, row := range rows {
		out = append(out, mapUpstreamConn(row))
	}
	return out, total, nil
}

func (u *upstreamStoreAdapter) SaveConnection(input opscontract.SaveUpstreamConnectionInput) (*opscontract.UpstreamConnection, error) {
	if u.connections == nil {
		return nil, opscontract.ErrNotConfigured
	}
	baseURL := strings.TrimRight(strings.TrimSpace(input.BaseURL), "/")
	targetID := input.ID
	if targetID == 0 {
		existing, _, err := u.connections.List(siteconnectioncontract.ListFilter{Page: 1, PageSize: 100})
		if err != nil {
			return nil, err
		}
		for _, item := range existing {
			if strings.EqualFold(strings.TrimRight(strings.TrimSpace(item.BaseURL), "/"), baseURL) {
				targetID = item.ID
				break
			}
		}
	}

	var saved *siteconnectiondomain.Connection
	var err error
	if targetID > 0 {
		saved, err = u.connections.Update(targetID, siteconnectionapp.UpdateInput{
			Name:               input.Name,
			BaseURL:            baseURL,
			ApiKey:             input.ApiKey,
			ApiSecret:          input.ApiSecret,
			Protocol:           input.Protocol,
			CallbackURL:        input.CallbackURL,
			ExchangeRate:       input.ExchangeRate,
			PriceMarkupPercent: input.PriceMarkupPercent,
			PriceRoundingMode:  input.PriceRoundingMode,
			AutoSyncPrice:      input.AutoSyncPrice,
		})
	} else {
		if strings.TrimSpace(input.ApiSecret) == "" {
			return nil, opscontract.ErrInvalidInput
		}
		exRate := 1.0
		if input.ExchangeRate != nil && *input.ExchangeRate > 0 {
			exRate = *input.ExchangeRate
		}
		markup := 0.0
		if input.PriceMarkupPercent != nil && *input.PriceMarkupPercent >= 0 {
			markup = *input.PriceMarkupPercent
		}
		rounding := "none"
		if input.PriceRoundingMode != nil && strings.TrimSpace(*input.PriceRoundingMode) != "" {
			rounding = strings.TrimSpace(*input.PriceRoundingMode)
		}
		autoSync := false
		if input.AutoSyncPrice != nil {
			autoSync = *input.AutoSyncPrice
		}
		saved, err = u.connections.Create(siteconnectionapp.CreateInput{
			Name:               input.Name,
			BaseURL:            baseURL,
			ApiKey:             input.ApiKey,
			ApiSecret:          input.ApiSecret,
			Protocol:           input.Protocol,
			CallbackURL:        input.CallbackURL,
			ExchangeRate:       exRate,
			PriceMarkupPercent: markup,
			PriceRoundingMode:  rounding,
			AutoSyncPrice:      autoSync,
		})
	}
	if err != nil {
		if errors.Is(err, siteconnectioncontract.ErrInvalid) {
			return nil, opscontract.ErrInvalidInput
		}
		if errors.Is(err, siteconnectioncontract.ErrNotFound) {
			return nil, opscontract.ErrNotFound
		}
		return nil, err
	}

	shouldPing := true
	if input.Ping != nil {
		shouldPing = *input.Ping
	}
	var pingSite, pingErrStr string
	if shouldPing {
		pingRes, pingErr := u.connections.Ping(saved.ID)
		if pingErr != nil {
			pingErrStr = pingErr.Error()
		} else if pingRes != nil {
			pingSite = pingRes.SiteName
		}
		if reloaded, getErr := u.connections.GetByID(saved.ID); getErr == nil && reloaded != nil {
			saved = reloaded
		}
	}
	out := mapUpstreamConn(*saved)
	out.PingSiteName = pingSite
	out.PingError = pingErrStr
	return &out, nil
}

func (u *upstreamStoreAdapter) UpdateConnectionStatus(id uint, status string) (*opscontract.UpstreamConnection, error) {
	if u.connections == nil {
		return nil, opscontract.ErrNotConfigured
	}
	if err := u.connections.SetStatus(id, status); err != nil {
		if errors.Is(err, siteconnectioncontract.ErrNotFound) {
			return nil, opscontract.ErrNotFound
		}
		return nil, err
	}
	reloaded, err := u.connections.GetByID(id)
	if err != nil {
		return nil, err
	}
	if reloaded == nil {
		return nil, opscontract.ErrNotFound
	}
	out := mapUpstreamConn(*reloaded)
	return &out, nil
}

func (u *upstreamStoreAdapter) DeleteConnection(id uint) error {
	if u.connections == nil {
		return opscontract.ErrNotConfigured
	}
	conn, err := u.connections.GetByID(id)
	if err != nil {
		if errors.Is(err, siteconnectioncontract.ErrNotFound) {
			return opscontract.ErrNotFound
		}
		return err
	}
	if conn == nil {
		return opscontract.ErrNotFound
	}
	if strings.EqualFold(strings.TrimSpace(conn.Status), constants.ConnectionStatusActive) {
		return &opscontract.ConflictError{Code: "upstream_connection_active"}
	}
	if err := u.connections.Delete(id); err != nil {
		if errors.Is(err, siteconnectioncontract.ErrNotFound) {
			return opscontract.ErrNotFound
		}
		return err
	}
	return nil
}

func (u *upstreamStoreAdapter) FetchCatalog(connectionID uint) (*opscontract.UpstreamCatalogResult, error) {
	if u.mappings == nil {
		return nil, opscontract.ErrNotConfigured
	}
	cats, catSupported, catErr := u.mappings.ListUpstreamCategories(connectionID)
	if catErr != nil {
		if errors.Is(catErr, siteconnectioncontract.ErrNotFound) {
			return nil, opscontract.ErrNotFound
		}
		return nil, catErr
	}
	var allProducts []opscontract.UpstreamCatalogProduct
	total := 0
	mappedByUpID := map[uint]mappingdomain.Mapping{}
	if u.mappingRepo != nil {
		if existing, _, listErr := u.mappingRepo.List(mappingcontract.ListFilter{ConnectionID: connectionID, Page: 1, PageSize: 500}); listErr == nil {
			for _, m := range existing {
				mappedByUpID[m.UpstreamProductID] = m
			}
		}
	}

	for page := 1; page <= 10; page++ {
		prodRes, prodErr := u.mappings.ListUpstreamProducts(connectionID, page, 50)
		if prodErr != nil {
			return nil, prodErr
		}
		total = prodRes.Total
		for _, item := range prodRes.Items {
			m, isMapped := mappedByUpID[item.ID]
			skuCodes := make([]string, 0, len(item.SKUs))
			for _, sku := range item.SKUs {
				if code := strings.TrimSpace(sku.SKUCode); code != "" {
					skuCodes = append(skuCodes, code)
				}
			}
			allProducts = append(allProducts, opscontract.UpstreamCatalogProduct{
				ID:                      item.ID,
				CategoryID:              item.CategoryID,
				Slug:                    item.Slug,
				Title:                   map[string]any(item.Title),
				PriceAmount:             item.PriceAmount,
				Currency:                item.Currency,
				IsActive:                item.IsActive,
				SKUCount:                len(item.SKUs),
				SKUCodes:                skuCodes,
				RequiresReservation:     item.RequiresReservation,
				EntitlementContractHash: item.EntitlementContractHash,
				Mapped:                  isMapped,
				MappingID:               m.ID,
				LocalProductID:          m.LocalProductID,
			})
		}
		if len(prodRes.Items) < 50 || page*50 >= prodRes.Total {
			break
		}
	}

	outCats := make([]opscontract.UpstreamCatalogCategory, 0, len(cats))
	for _, c := range cats {
		outCats = append(outCats, opscontract.UpstreamCatalogCategory{
			ID:        c.ID,
			ParentID:  c.ParentID,
			Slug:      c.Slug,
			Name:      map[string]any(c.Name),
			SortOrder: c.SortOrder,
		})
	}
	return &opscontract.UpstreamCatalogResult{
		Source:              "storefront",
		CollectedAt:         time.Now().UTC(),
		Freshness:           "fresh",
		ConnectionID:        connectionID,
		CategoriesSupported: catSupported,
		Categories:          outCats,
		Products:            allProducts,
		Total:               total,
	}, nil
}

func (u *upstreamStoreAdapter) ImportOrSync(connectionID uint, input opscontract.ImportUpstreamProductsInput) (*opscontract.ImportUpstreamProductsResult, error) {
	if u.mappings == nil || u.mappingRepo == nil {
		return nil, opscontract.ErrNotConfigured
	}
	autoCreate := true
	if input.AutoCreateCategory != nil {
		autoCreate = *input.AutoCreateCategory
	}
	skuCode := strings.TrimSpace(input.SKUCode)
	targetIDs := append([]uint(nil), input.UpstreamProductIDs...)
	res := &opscontract.ImportUpstreamProductsResult{
		Source:       "storefront",
		CollectedAt:  time.Now().UTC(),
		ConnectionID: connectionID,
	}

	if len(targetIDs) == 0 && skuCode != "" {
		cat, err := u.FetchCatalog(connectionID)
		if err != nil {
			return nil, err
		}
		for _, p := range cat.Products {
			matched := strings.EqualFold(strings.TrimSpace(p.Slug), skuCode)
			if !matched {
				for _, sc := range p.SKUCodes {
					if strings.EqualFold(strings.TrimSpace(sc), skuCode) {
						matched = true
						break
					}
				}
			}
			if matched {
				targetIDs = append(targetIDs, p.ID)
			}
		}
		if len(targetIDs) == 0 {
			res.SkippedReason = "sku_not_found_in_upstream_catalog"
			return res, nil
		}
	} else if len(targetIDs) == 0 && input.SyncStockOnly {
		existingList, _, err := u.mappingRepo.List(mappingcontract.ListFilter{ConnectionID: connectionID, Page: 1, PageSize: 500})
		if err != nil {
			return nil, err
		}
		for _, m := range existingList {
			if m.UpstreamProductID > 0 {
				targetIDs = append(targetIDs, m.UpstreamProductID)
			}
		}
		if len(targetIDs) == 0 {
			res.SkippedReason = "no_mapped_products"
			return res, nil
		}
	} else if len(targetIDs) == 0 {
		cat, err := u.FetchCatalog(connectionID)
		if err != nil {
			return nil, err
		}
		for _, p := range cat.Products {
			if p.IsActive {
				targetIDs = append(targetIDs, p.ID)
			}
		}
	}

	res.Requested = len(targetIDs)
	var toImport []uint
	for _, upID := range targetIDs {
		if upID == 0 {
			continue
		}
		existing, err := u.mappingRepo.GetByConnectionAndUpstreamID(connectionID, upID)
		if err != nil {
			res.Failures = append(res.Failures, opscontract.ImportUpstreamProductsFailure{UpstreamProductID: upID, Error: err.Error()})
			continue
		}
		if existing != nil {
			if syncErr := u.mappings.SyncProduct(existing.ID); syncErr != nil {
				res.Failures = append(res.Failures, opscontract.ImportUpstreamProductsFailure{UpstreamProductID: upID, Error: syncErr.Error()})
			} else {
				res.SyncedCount++
			}
			continue
		}
		if input.SyncStockOnly {
			res.SkippedCount++
			continue
		}
		toImport = append(toImport, upID)
	}
	if input.SyncStockOnly && res.SyncedCount == 0 && res.SkippedCount > 0 && len(res.Failures) == 0 {
		res.SkippedReason = "target_sku_not_mapped"
	}
	if len(toImport) > 0 {
		outcomes, err := u.mappings.BatchImportUpstreamProducts(connectionID, toImport, input.LocalCategoryID, autoCreate)
		if err != nil {
			return nil, err
		}
		for _, out := range outcomes {
			if out.Err != nil {
				res.Failures = append(res.Failures, opscontract.ImportUpstreamProductsFailure{UpstreamProductID: out.UpstreamProductID, Error: out.Err.Error()})
			} else {
				res.ImportedCount++
			}
		}
	}
	return res, nil
}

type orderRefundStoreAdapter struct {
	orders  ordercontract.Store
	refunds *orderrefund.Service
	wallets *walletapp.Service
}

func findExactOrderByNo(rows []orderdomain.Order, orderNo string) *orderdomain.Order {
	for i := range rows {
		if strings.EqualFold(strings.TrimSpace(rows[i].OrderNo), orderNo) {
			found := rows[i]
			return &found
		}
		for j := range rows[i].Children {
			if strings.EqualFold(strings.TrimSpace(rows[i].Children[j].OrderNo), orderNo) {
				found := rows[i].Children[j]
				return &found
			}
		}
	}
	return nil
}

func mapRefundErr(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, orderrefund.ErrOrderNotFound):
		return opscontract.ErrNotFound
	case errors.Is(err, orderrefund.ErrRefundOperationInvalid),
		errors.Is(err, walletcontract.ErrInvalidAmount):
		return opscontract.ErrInvalidInput
	case errors.Is(err, orderrefund.ErrRefundOperationConflict),
		errors.Is(err, orderrefund.ErrOrderStatusInvalid),
		errors.Is(err, orderrefund.ErrOrderRefundExpired),
		errors.Is(err, walletcontract.ErrRefundExceeded),
		errors.Is(err, walletcontract.ErrNotSupportedForGuest),
		errors.Is(err, walletcontract.ErrAccountNotFound):
		return opscontract.ErrConflict
	default:
		return err
	}
}

func (a *orderRefundStoreAdapter) RefundOrder(orderNo string, input opscontract.RefundOrderInput) (*opscontract.RefundOrderResult, error) {
	if a.orders == nil || a.refunds == nil {
		return nil, opscontract.ErrNotConfigured
	}
	orderNo = strings.TrimSpace(orderNo)
	opID := strings.TrimSpace(input.OperationID)
	amount, err := a.refunds.ParseRefundAmount(input.Amount)
	if err != nil {
		return nil, opscontract.ErrInvalidInput
	}

	rows, _, err := a.orders.ListAdmin(ordercontract.ListFilter{OrderNo: orderNo, Page: 1, PageSize: 20})
	if err != nil {
		return nil, err
	}
	target := findExactOrderByNo(rows, orderNo)
	if target == nil {
		return nil, opscontract.ErrNotFound
	}

	remark := strings.TrimSpace(input.Remark)
	if remark == "" {
		remark = fmt.Sprintf("ops_refund channel=%s tx=%s", strings.TrimSpace(input.Channel), strings.TrimSpace(input.ExternalTxRef))
	}

	if input.Mode == "wallet" {
		if opID == "" {
			return nil, opscontract.ErrInvalidInput
		}
		updatedOrder, _, record, wErr := a.refunds.AdminRefundToWallet(orderrefund.AdminRefundToWalletInput{
			OperationID: opID,
			OrderID:     target.ID,
			Amount:      amount,
			Remark:      remark,
		})
		if wErr != nil {
			return nil, mapRefundErr(wErr)
		}
		res := &opscontract.RefundOrderResult{
			Source:           "storefront",
			CollectedAt:      time.Now().UTC(),
			OrderID:          updatedOrder.ID,
			OrderNo:          updatedOrder.OrderNo,
			Status:           updatedOrder.Status,
			TotalAmount:      updatedOrder.TotalAmount.String(),
			RefundedAmount:   updatedOrder.RefundedAmount.String(),
			Currency:         updatedOrder.Currency,
			RefundRecordID:   record.ID,
			RefundType:       record.Type,
			SettlementStatus: "settled",
			OperationID:      opID,
		}
		if a.wallets != nil && a.wallets.HasUnifiedBalance() {
			unifiedOpID := fmt.Sprintf("order:%d:refund:%d", target.ID, record.ID)
			op, opErr := a.wallets.GetUnifiedOpByOpID(unifiedOpID)
			switch {
			case opErr != nil:
				res.SettlementStatus = "balance_sync_failed"
				res.BalanceSyncStatus = "sync_error"
				res.BalanceSyncOpID = unifiedOpID
				res.BalanceSyncError = opErr.Error()
			case op != nil && op.Status == walletdomain.UnifiedOpStatusCredited:
				updatedAt := op.UpdatedAt
				res.SettlementStatus = "settled"
				res.BalanceSyncStatus = "credited"
				res.BalanceSyncOpID = op.OpID
				res.BalanceSyncUpdatedAt = &updatedAt
			case op != nil && op.Status == walletdomain.UnifiedOpStatusCreditPending:
				updatedAt := op.UpdatedAt
				res.BalanceSyncOpID = op.OpID
				res.BalanceSyncError = op.LastError
				res.BalanceSyncUpdatedAt = &updatedAt
				if strings.TrimSpace(op.LastError) != "" {
					res.SettlementStatus = "balance_sync_failed"
					res.BalanceSyncStatus = "sync_error"
				} else {
					res.SettlementStatus = "balance_syncing"
					res.BalanceSyncStatus = "syncing"
				}
			case op != nil:
				updatedAt := op.UpdatedAt
				res.SettlementStatus = "balance_syncing"
				res.BalanceSyncStatus = op.Status
				res.BalanceSyncOpID = op.OpID
				res.BalanceSyncError = op.LastError
				res.BalanceSyncUpdatedAt = &updatedAt
			default:
				res.SettlementStatus = "settled"
				res.BalanceSyncStatus = "credited"
				res.BalanceSyncOpID = unifiedOpID
			}
		}
		return res, nil
	}

	updatedOrder, record, mErr := a.refunds.AdminManualRefund(orderrefund.AdminManualRefundInput{
		OperationID:        opID,
		OrderID:            target.ID,
		Amount:             amount,
		Remark:             remark,
		PaymentFeeRefunded: input.PaymentFeeRefunded,
	})
	if mErr != nil {
		return nil, mapRefundErr(mErr)
	}
	return &opscontract.RefundOrderResult{
		Source:           "storefront",
		CollectedAt:      time.Now().UTC(),
		OrderID:          updatedOrder.ID,
		OrderNo:          updatedOrder.OrderNo,
		Status:           updatedOrder.Status,
		TotalAmount:      updatedOrder.TotalAmount.String(),
		RefundedAmount:   updatedOrder.RefundedAmount.String(),
		Currency:         updatedOrder.Currency,
		RefundRecordID:   record.ID,
		RefundType:       record.Type,
		SettlementStatus: "settled",
		OperationID:      opID,
	}, nil
}

type userSessionRevokerAdapter struct {
	users usercontract.Store
}

func (a *userSessionRevokerAdapter) RevokeUserSessions(userID uint) (bool, error) {
	if a == nil || a.users == nil || userID == 0 {
		return false, nil
	}
	user, err := a.users.GetByID(userID)
	if err != nil {
		return false, err
	}
	if user == nil {
		return false, nil
	}
	now := time.Now()
	user.TokenVersion++
	user.TokenInvalidBefore = &now
	user.UpdatedAt = now
	if err := a.users.Update(user); err != nil {
		return false, err
	}
	_ = cache.SetUserAuthState(context.Background(), cache.BuildUserAuthState(user))
	return true, nil
}

type universalGiftCardClaimAdapter struct {
	wallets walletcontract.Repository
}

func (a *universalGiftCardClaimAdapter) LookupUniversalGiftCardClaim(claimID string) (*opscontract.UniversalGiftCardClaimStatus, error) {
	if a == nil || a.wallets == nil {
		return nil, opscontract.ErrNotConfigured
	}
	claimID = strings.TrimSpace(claimID)
	if claimID == "" {
		return nil, opscontract.ErrInvalidInput
	}
	now := time.Now().UTC()
	txn, err := a.wallets.GetTransactionByReference("universal_gift_card:" + claimID)
	if err != nil {
		return nil, err
	}
	if txn == nil {
		return &opscontract.UniversalGiftCardClaimStatus{
			Source:      "storefront",
			CollectedAt: now,
			ClaimID:     claimID,
			Found:       false,
		}, nil
	}
	createdAt := txn.CreatedAt.UTC()
	return &opscontract.UniversalGiftCardClaimStatus{
		Source:        "storefront",
		CollectedAt:   now,
		ClaimID:       claimID,
		Found:         true,
		UserID:        txn.UserID,
		TransactionID: txn.ID,
		Amount:        txn.Amount.Decimal.StringFixed(2),
		Currency:      strings.TrimSpace(txn.Currency),
		TxnRef:        fmt.Sprintf("storefront:wallet_txn:%d", txn.ID),
		CreditDisplay: fmt.Sprintf("%s %s", txn.Amount.Decimal.StringFixed(2), strings.TrimSpace(txn.Currency)),
		CreatedAt:     &createdAt,
	}, nil
}

type unifiedOpReaderAdapter struct {
	wallets *walletapp.Service
}

func (a *unifiedOpReaderAdapter) ListPendingUnifiedOpsForOps(limit int) ([]opscontract.PendingUnifiedOp, error) {
	if a == nil || a.wallets == nil || !a.wallets.HasUnifiedBalance() {
		return nil, nil
	}
	rows, err := a.wallets.ListUnresolvedUnifiedOps(limit)
	if err != nil {
		return nil, err
	}
	out := make([]opscontract.PendingUnifiedOp, 0, len(rows))
	for _, row := range rows {
		out = append(out, opscontract.PendingUnifiedOp{
			OpID:      row.OpID,
			UserID:    row.UserID,
			Kind:      row.Direction,
			Status:    row.Status,
			Amount:    row.Amount.String(),
			Currency:  row.Currency,
			Attempts:  row.Attempts,
			LastError: strings.TrimSpace(row.LastError),
			UpdatedAt: row.UpdatedAt.UTC(),
		})
	}
	return out, nil
}

const opsSystemAdminID uint = 1

type growthStoreAdapter struct {
	cfg              *config.Config
	settings         *settingsapp.Service
	affiliate        *affiliateapp.Service
	wallets          walletcontract.Repository
	resellerMgmt     *resellerapp.ManagementService
	resellerWithdraw *resellerapp.AccountingWithdrawService
	resellerStore    *resellergormstore.Store
}

func decimalToMinor(d decimal.Decimal) int64 {
	return d.Mul(decimal.NewFromInt(100)).Round(0).IntPart()
}

func (a *growthStoreAdapter) siteCurrency() string {
	curr := constants.SiteCurrencyDefault
	if a.settings != nil {
		if c, err := a.settings.GetSiteCurrency(constants.SiteCurrencyDefault); err == nil && strings.TrimSpace(c) != "" {
			curr = strings.ToUpper(strings.TrimSpace(c))
		}
	}
	return curr
}

func (a *growthStoreAdapter) GrowthSummary() (*opscontract.GrowthSummaryResult, error) {
	siteCurrency := a.siteCurrency()
	res := &opscontract.GrowthSummaryResult{
		Source:       "storefront",
		CollectedAt:  time.Now().UTC(),
		SiteCurrency: siteCurrency,
	}
	res.Affiliate.Stats = opscontract.AffiliateStatsSummary{
		PendingCommissionByCurrency:   map[string]int64{},
		AvailableCommissionByCurrency: map[string]int64{},
		WithdrawnCommissionByCurrency: map[string]int64{},
		EffectiveCostByCurrency:       map[string]int64{},
	}
	res.RechargeBonus.Settings = settingscontract.DefaultWalletRechargeBonusConfig()
	res.RechargeBonus.Stats = opscontract.RechargeBonusStatsSummary{
		TotalBonusByCurrency: map[string]int64{},
		CostByCurrency:       map[string]int64{},
	}
	res.Reseller.Stats = opscontract.ResellerStatsSummary{
		ProfitCostByCurrency: map[string]int64{},
	}

	if a.settings != nil {
		if aff, err := a.settings.GetAffiliateSetting(); err == nil {
			res.Affiliate.Settings = opscontract.AffiliateSettingDTO{
				Enabled:           aff.Enabled,
				CommissionRate:    aff.CommissionRate,
				ConfirmDays:       aff.ConfirmDays,
				MinWithdrawAmount: aff.MinWithdrawAmount,
				WithdrawChannels:  aff.WithdrawChannels,
			}
		}
		res.RechargeBonus.Settings = a.settings.GetWalletRechargeBonusConfig()
	}

	if a.affiliate != nil {
		var pendingSum, availSum, withdrawnSum decimal.Decimal
		for page := 1; ; page++ {
			items, total, err := a.affiliate.ListAdminUsers(affiliateapp.AdminProfileListFilter{Page: page, PageSize: 200})
			if err != nil {
				break
			}
			res.Affiliate.Stats.ProfileCount = total
			for _, item := range items {
				pendingSum = pendingSum.Add(item.Stats.PendingCommission.Decimal)
				availSum = availSum.Add(item.Stats.AvailableCommission.Decimal)
				withdrawnSum = withdrawnSum.Add(item.Stats.WithdrawnCommission.Decimal)
			}
			if len(items) < 200 || int64(page*200) >= total {
				break
			}
		}
		res.Affiliate.Stats.PendingCommissionByCurrency[siteCurrency] = decimalToMinor(pendingSum)
		res.Affiliate.Stats.AvailableCommissionByCurrency[siteCurrency] = decimalToMinor(availSum)
		res.Affiliate.Stats.WithdrawnCommissionByCurrency[siteCurrency] = decimalToMinor(withdrawnSum)
		res.Affiliate.Stats.EffectiveCostByCurrency[siteCurrency] = decimalToMinor(availSum.Add(withdrawnSum))

		if _, pendingW, err := a.affiliate.ListAdminWithdraws(affiliateapp.AdminWithdrawListFilter{
			Page:     1,
			PageSize: 1,
			Status:   constants.AffiliateWithdrawStatusPendingReview,
		}); err == nil {
			res.Affiliate.Stats.PendingWithdrawCount = pendingW
		}
	}

	if a.wallets != nil {
		costRate := decimal.NewFromFloat(res.RechargeBonus.Settings.CostRatePercent).Div(decimal.NewFromInt(100))
		bonusSums := map[string]decimal.Decimal{}
		for page := 1; ; page++ {
			txns, total, err := a.wallets.ListTransactions(walletcontract.TransactionListFilter{
				Page:     page,
				PageSize: 200,
				Type:     constants.WalletTxnTypeRechargeBonus,
			})
			if err != nil {
				break
			}
			res.RechargeBonus.Stats.BonusCount = total
			for _, txn := range txns {
				curr := strings.ToUpper(strings.TrimSpace(txn.Currency))
				if curr == "" {
					curr = siteCurrency
				}
				bonusSums[curr] = bonusSums[curr].Add(txn.Amount.Decimal)
			}
			if len(txns) < 200 || int64(page*200) >= total {
				break
			}
		}
		for curr, sum := range bonusSums {
			res.RechargeBonus.Stats.TotalBonusByCurrency[curr] = decimalToMinor(sum)
			res.RechargeBonus.Stats.CostByCurrency[curr] = decimalToMinor(sum.Mul(costRate))
		}
	}

	if a.cfg != nil {
		res.Reseller.Config = opscontract.ResellerConfigSummary{
			Enabled:               a.cfg.Reseller.Enabled,
			SelfApplyEnabled:      a.cfg.Reseller.SelfApplyEnabled,
			SubdomainBase:         strings.TrimSpace(a.cfg.Reseller.SubdomainBase),
			SettlementConfirmDays: a.cfg.Reseller.SettlementConfirmDays,
		}
	}

	if a.resellerStore != nil {
		if _, total, err := a.resellerStore.ListProfiles(resellercontract.ProfileListFilter{Page: 1, PageSize: 1}); err == nil {
			res.Reseller.Stats.TotalProfiles = total
		}
		if _, pending, err := a.resellerStore.ListProfiles(resellercontract.ProfileListFilter{Page: 1, PageSize: 1, Status: resellerdomain.ProfileStatusPendingReview}); err == nil {
			res.Reseller.Stats.PendingReviewCount = pending
		}
		if _, active, err := a.resellerStore.ListProfiles(resellercontract.ProfileListFilter{Page: 1, PageSize: 1, Status: resellerdomain.ProfileStatusActive}); err == nil {
			res.Reseller.Stats.ActiveCount = active
		}
		if _, pendingW, err := a.resellerStore.ListAdminResellerWithdrawRequests(resellercontract.AdminWithdrawListFilter{
			Page:     1,
			PageSize: 1,
			Status:   resellerdomain.WithdrawStatusPending,
		}); err == nil {
			res.Reseller.Stats.PendingWithdrawCount = pendingW
		}
		ledgerSums := map[string]decimal.Decimal{}
		for page := 1; ; page++ {
			entries, total, err := a.resellerStore.ListAdminResellerLedgerEntries(resellercontract.AdminLedgerListFilter{
				Page:     page,
				PageSize: 200,
			})
			if err != nil {
				break
			}
			for _, entry := range entries {
				if entry.Status == resellerdomain.LedgerStatusCanceled {
					continue
				}
				if entry.Type != resellerdomain.LedgerTypeOrderProfit &&
					entry.Type != resellerdomain.LedgerTypeRefundDeduct &&
					entry.Type != resellerdomain.LedgerTypeManualAdjust {
					continue
				}
				curr := strings.ToUpper(strings.TrimSpace(entry.Currency))
				if curr == "" {
					curr = siteCurrency
				}
				ledgerSums[curr] = ledgerSums[curr].Add(entry.Amount.Decimal)
			}
			if len(entries) < 200 || int64(page*200) >= total {
				break
			}
		}
		for curr, sum := range ledgerSums {
			if sum.IsPositive() {
				res.Reseller.Stats.ProfitCostByCurrency[curr] = decimalToMinor(sum)
			} else {
				res.Reseller.Stats.ProfitCostByCurrency[curr] = 0
			}
		}
	}

	return res, nil
}

func (a *growthStoreAdapter) UpdateAffiliateSettings(input opscontract.AffiliateSettingDTO) (*opscontract.AffiliateSettingDTO, error) {
	if a.settings == nil {
		return nil, opscontract.ErrNotConfigured
	}
	updated, err := a.settings.UpdateAffiliateSetting(settingsintegration.AffiliateSetting{
		Enabled:           input.Enabled,
		CommissionRate:    input.CommissionRate,
		ConfirmDays:       input.ConfirmDays,
		MinWithdrawAmount: input.MinWithdrawAmount,
		WithdrawChannels:  input.WithdrawChannels,
	})
	if err != nil {
		return nil, mapGrowthError(err)
	}
	_ = cache.DelAllPublicConfig(context.Background())
	return &opscontract.AffiliateSettingDTO{
		Enabled:           updated.Enabled,
		CommissionRate:    updated.CommissionRate,
		ConfirmDays:       updated.ConfirmDays,
		MinWithdrawAmount: updated.MinWithdrawAmount,
		WithdrawChannels:  updated.WithdrawChannels,
	}, nil
}

func (a *growthStoreAdapter) ListAffiliateWithdraws(page, pageSize int, status string) (*opscontract.AffiliateWithdrawsResult, error) {
	if a.affiliate == nil {
		return nil, opscontract.ErrNotConfigured
	}
	siteCurrency := a.siteCurrency()
	rows, total, err := a.affiliate.ListAdminWithdraws(affiliateapp.AdminWithdrawListFilter{
		Page:     page,
		PageSize: pageSize,
		Status:   status,
	})
	if err != nil {
		return nil, mapGrowthError(err)
	}
	items := make([]opscontract.AffiliateWithdrawItem, 0, len(rows))
	for _, row := range rows {
		item := opscontract.AffiliateWithdrawItem{
			ID:                 row.ID,
			AffiliateProfileID: row.AffiliateProfileID,
			AffiliateCode:      row.AffiliateProfile.AffiliateCode,
			Amount:             row.Amount.String(),
			Currency:           siteCurrency,
			Channel:            row.Channel,
			Account:            row.Account,
			Status:             row.Status,
			RejectReason:       row.RejectReason,
			ProcessedAt:        row.ProcessedAt,
			CreatedAt:          row.CreatedAt,
		}
		if row.AffiliateProfile.UserID > 0 {
			item.UserID = row.AffiliateProfile.UserID
		}
		if row.AffiliateProfile.User.ID > 0 {
			item.UserEmail = row.AffiliateProfile.User.Email
		}
		items = append(items, item)
	}
	return &opscontract.AffiliateWithdrawsResult{
		Source:      "storefront",
		CollectedAt: time.Now().UTC(),
		Items:       items,
		Total:       total,
		Page:        page,
		PageSize:    pageSize,
	}, nil
}

func (a *growthStoreAdapter) ReviewAffiliateWithdraw(id uint, action, rejectReason string) (*opscontract.AffiliateWithdrawItem, error) {
	if a.affiliate == nil {
		return nil, opscontract.ErrNotConfigured
	}
	row, err := a.affiliate.ReviewWithdraw(opsSystemAdminID, id, action, rejectReason)
	if err != nil {
		return nil, mapGrowthError(err)
	}
	item := &opscontract.AffiliateWithdrawItem{
		ID:                 row.ID,
		AffiliateProfileID: row.AffiliateProfileID,
		AffiliateCode:      row.AffiliateProfile.AffiliateCode,
		Amount:             row.Amount.String(),
		Currency:           a.siteCurrency(),
		Channel:            row.Channel,
		Account:            row.Account,
		Status:             row.Status,
		RejectReason:       row.RejectReason,
		ProcessedAt:        row.ProcessedAt,
		CreatedAt:          row.CreatedAt,
	}
	return item, nil
}

func (a *growthStoreAdapter) UpdateRechargeBonus(input settingscontract.WalletRechargeBonusConfig) (*settingscontract.WalletRechargeBonusConfig, error) {
	if a.settings == nil {
		return nil, opscontract.ErrNotConfigured
	}
	updated, err := a.settings.UpdateWalletRechargeBonusConfig(input)
	if err != nil {
		return nil, mapGrowthError(err)
	}
	_ = cache.DelAllPublicConfig(context.Background())
	return &updated, nil
}

func (a *growthStoreAdapter) ListResellerProfiles(page, pageSize int, status, keyword string) (*opscontract.ResellerProfilesResult, error) {
	if a.resellerStore == nil {
		return nil, opscontract.ErrNotConfigured
	}
	rows, total, err := a.resellerStore.ListProfiles(resellercontract.ProfileListFilter{
		Page:     page,
		PageSize: pageSize,
		Status:   status,
		Keyword:  keyword,
	})
	if err != nil {
		return nil, mapGrowthError(err)
	}
	items := make([]opscontract.ResellerProfileItem, 0, len(rows))
	for i := range rows {
		domains, _ := a.resellerStore.ListDomainsByResellerID(rows[i].ID)
		items = append(items, mapResellerProfileItem(&rows[i], domains))
	}
	return &opscontract.ResellerProfilesResult{
		Source:      "storefront",
		CollectedAt: time.Now().UTC(),
		Items:       items,
		Total:       total,
		Page:        page,
		PageSize:    pageSize,
	}, nil
}

func (a *growthStoreAdapter) ApproveResellerProfile(id uint, input opscontract.ApproveResellerInput) (*opscontract.ResellerProfileItem, error) {
	if a.resellerMgmt == nil {
		return nil, opscontract.ErrNotConfigured
	}
	res, err := a.resellerMgmt.ApproveProfile(context.Background(), opsSystemAdminID, id, resellerapp.ResellerApproveInput{
		DefaultMarkupPercent: decimal.NewFromFloat(input.DefaultMarkupPercent).Round(2),
		MaxMarkupPercent:     decimal.NewFromFloat(input.MaxMarkupPercent).Round(2),
	})
	if err != nil {
		return nil, mapGrowthError(err)
	}
	item := mapResellerProfileItem(res.Profile, nil)
	return &item, nil
}

func (a *growthStoreAdapter) RejectResellerProfile(id uint, input opscontract.RejectResellerInput) (*opscontract.ResellerProfileItem, error) {
	if a.resellerMgmt == nil {
		return nil, opscontract.ErrNotConfigured
	}
	profile, err := a.resellerMgmt.RejectProfile(opsSystemAdminID, id, input.Reason)
	if err != nil {
		return nil, mapGrowthError(err)
	}
	item := mapResellerProfileItem(profile, nil)
	return &item, nil
}

func (a *growthStoreAdapter) UpdateResellerProfileStatus(id uint, input opscontract.UpdateResellerStatusInput) (*opscontract.ResellerProfileItem, error) {
	if a.resellerMgmt == nil {
		return nil, opscontract.ErrNotConfigured
	}
	var (
		profile *resellerdomain.Profile
		err     error
	)
	if input.Status == resellerdomain.ProfileStatusDisabled {
		profile, err = a.resellerMgmt.DisableProfile(opsSystemAdminID, id, input.Reason)
	} else {
		profile, err = a.resellerMgmt.RestoreProfile(opsSystemAdminID, id)
	}
	if err != nil {
		return nil, mapGrowthError(err)
	}
	item := mapResellerProfileItem(profile, nil)
	return &item, nil
}

func (a *growthStoreAdapter) ListResellerWithdraws(page, pageSize int, status string) (*opscontract.ResellerWithdrawsResult, error) {
	if a.resellerStore == nil {
		return nil, opscontract.ErrNotConfigured
	}
	rows, total, err := a.resellerStore.ListAdminResellerWithdrawRequests(resellercontract.AdminWithdrawListFilter{
		Page:     page,
		PageSize: pageSize,
		Status:   status,
	})
	if err != nil {
		return nil, mapGrowthError(err)
	}
	items := make([]opscontract.ResellerWithdrawItem, 0, len(rows))
	for _, row := range rows {
		item := opscontract.ResellerWithdrawItem{
			ID:           row.ID,
			ResellerID:   row.ResellerID,
			Amount:       row.Amount.String(),
			Currency:     row.Currency,
			Channel:      row.Channel,
			Account:      row.Account,
			Status:       row.Status,
			RejectReason: row.RejectReason,
			ProcessedAt:  row.ProcessedAt,
			CreatedAt:    row.CreatedAt,
		}
		if row.Profile != nil {
			item.UserID = row.Profile.UserID
			if row.Profile.User != nil {
				item.UserEmail = row.Profile.User.Email
			}
		}
		items = append(items, item)
	}
	return &opscontract.ResellerWithdrawsResult{
		Source:      "storefront",
		CollectedAt: time.Now().UTC(),
		Items:       items,
		Total:       total,
		Page:        page,
		PageSize:    pageSize,
	}, nil
}

func (a *growthStoreAdapter) ReviewResellerWithdraw(id uint, action, rejectReason string) (*opscontract.ResellerWithdrawItem, error) {
	if a.resellerWithdraw == nil {
		return nil, opscontract.ErrNotConfigured
	}
	row, err := a.resellerWithdraw.ReviewWithdraw(opsSystemAdminID, id, action, rejectReason)
	if err != nil {
		return nil, mapGrowthError(err)
	}
	item := &opscontract.ResellerWithdrawItem{
		ID:           row.ID,
		ResellerID:   row.ResellerID,
		Amount:       row.Amount.String(),
		Currency:     row.Currency,
		Channel:      row.Channel,
		Account:      row.Account,
		Status:       row.Status,
		RejectReason: row.RejectReason,
		ProcessedAt:  row.ProcessedAt,
		CreatedAt:    row.CreatedAt,
	}
	return item, nil
}

func mapResellerProfileItem(profile *resellerdomain.Profile, domains []resellerdomain.Domain) opscontract.ResellerProfileItem {
	if profile == nil {
		return opscontract.ResellerProfileItem{}
	}
	item := opscontract.ResellerProfileItem{
		ID:                   profile.ID,
		UserID:               profile.UserID,
		Status:               profile.Status,
		SettlementStatus:     profile.SettlementStatus,
		ApplyReason:          profile.ApplyReason,
		RejectReason:         profile.RejectReason,
		DefaultMarkupPercent: profile.DefaultMarkupPercent.String(),
		MaxMarkupPercent:     profile.MaxMarkupPercent.String(),
		ReviewedAt:           profile.ReviewedAt,
		CreatedAt:            profile.CreatedAt,
	}
	if profile.User != nil {
		item.UserEmail = profile.User.Email
		item.UserDisplayName = profile.User.DisplayName
	}
	for _, d := range domains {
		if d.IsPrimary || item.PrimaryDomain == "" {
			item.PrimaryDomain = d.Domain
		}
	}
	return item
}

func mapGrowthError(err error) error {
	if err == nil {
		return nil
	}
	switch {
	case errors.Is(err, affiliateapp.ErrNotFound):
		return opscontract.ErrNotFound
	case errors.Is(err, affiliateapp.ErrWithdrawStatusInvalid),
		errors.Is(err, resellercontract.ErrProfileStatusInvalid),
		errors.Is(err, resellercontract.ErrWithdrawStatusInvalid),
		errors.Is(err, resellercontract.ErrMarkupExceeded):
		return opscontract.ErrInvalidInput
	default:
		msg := strings.ToLower(err.Error())
		if strings.Contains(msg, "not found") {
			return opscontract.ErrNotFound
		}
		if strings.Contains(msg, "invalid") {
			return opscontract.ErrInvalidInput
		}
		return err
	}
}
