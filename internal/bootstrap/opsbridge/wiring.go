package opsbridge

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/dujiao-next/internal/app/container"
	"github.com/dujiao-next/internal/config"
	"github.com/dujiao-next/internal/constants"
	mappingapp "github.com/dujiao-next/internal/modules/catalog/mapping/application"
	mappingcontract "github.com/dujiao-next/internal/modules/catalog/mapping/contract"
	mappingdomain "github.com/dujiao-next/internal/modules/catalog/mapping/domain"
	opsapp "github.com/dujiao-next/internal/modules/opsbridge/application"
	opscontract "github.com/dujiao-next/internal/modules/opsbridge/contract"
	opstransport "github.com/dujiao-next/internal/modules/opsbridge/transport/http"
	orderrefund "github.com/dujiao-next/internal/modules/order/application/refund"
	ordercontract "github.com/dujiao-next/internal/modules/order/contract"
	orderdomain "github.com/dujiao-next/internal/modules/order/domain"
	paymentapp "github.com/dujiao-next/internal/modules/payment/application"
	paymentcontract "github.com/dujiao-next/internal/modules/payment/contract"
	paymentdomain "github.com/dujiao-next/internal/modules/payment/domain"
	settingsapp "github.com/dujiao-next/internal/modules/settings/application"
	siteconnectionapp "github.com/dujiao-next/internal/modules/siteconnection/application"
	siteconnectioncontract "github.com/dujiao-next/internal/modules/siteconnection/contract"
	siteconnectiondomain "github.com/dujiao-next/internal/modules/siteconnection/domain"
	walletcontract "github.com/dujiao-next/internal/modules/wallet/contract"
	"github.com/dujiao-next/internal/shared/jsonmap"
	"github.com/dujiao-next/internal/shared/jsonslice"
	"github.com/dujiao-next/internal/shared/money"
	"github.com/shopspring/decimal"
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
	if c.SettingService != nil || c.SiteConnectionService != nil {
		svc.WithPricing(&pricingStoreAdapter{
			settings:    c.SettingService,
			connections: c.SiteConnectionService,
		})
	}
	if c.GiftCardRepo != nil {
		svc.WithGiftCards(c.GiftCardRepo)
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
		})
	}
	return opstransport.New(svc)
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

type upstreamStoreAdapter struct {
	connections *siteconnectionapp.Service
	mappings    *mappingapp.Service
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
			allProducts = append(allProducts, opscontract.UpstreamCatalogProduct{
				ID:                      item.ID,
				CategoryID:              item.CategoryID,
				Slug:                    item.Slug,
				Title:                   map[string]any(item.Title),
				PriceAmount:             item.PriceAmount,
				Currency:                item.Currency,
				IsActive:                item.IsActive,
				SKUCount:                len(item.SKUs),
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
	targetIDs := input.UpstreamProductIDs
	if len(targetIDs) == 0 {
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

	res := &opscontract.ImportUpstreamProductsResult{
		Source:       "storefront",
		CollectedAt:  time.Now().UTC(),
		ConnectionID: connectionID,
		Requested:    len(targetIDs),
	}
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
		toImport = append(toImport, upID)
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
		opTag := "[ops_op:" + opID + "]"
		existingList, listErr := a.orders.ListRefundRecordsByOrderIDs([]uint{target.ID})
		if listErr != nil {
			return nil, listErr
		}
		for i := range existingList {
			rec := existingList[i]
			matchedOp := (rec.OperationID != nil && *rec.OperationID == opID) || strings.Contains(rec.Remark, opTag)
			if !matchedOp {
				continue
			}
			if rec.Type != constants.OrderRefundTypeWallet || !rec.Amount.Decimal.Round(2).Equal(amount.Decimal.Round(2)) {
				return nil, opscontract.ErrConflict
			}
			reloaded, getErr := a.orders.GetByID(target.ID)
			if getErr != nil {
				return nil, getErr
			}
			if reloaded == nil {
				reloaded = target
			}
			return &opscontract.RefundOrderResult{
				Source:           "storefront",
				CollectedAt:      time.Now().UTC(),
				OrderID:          reloaded.ID,
				OrderNo:          reloaded.OrderNo,
				Status:           reloaded.Status,
				TotalAmount:      reloaded.TotalAmount.String(),
				RefundedAmount:   reloaded.RefundedAmount.String(),
				Currency:         reloaded.Currency,
				RefundRecordID:   rec.ID,
				RefundType:       rec.Type,
				SettlementStatus: "settled",
				OperationID:      opID,
			}, nil
		}

		walletRemark := remark + " " + opTag
		updatedOrder, _, record, wErr := a.refunds.AdminRefundToWallet(orderrefund.AdminRefundToWalletInput{
			OrderID: target.ID,
			Amount:  amount,
			Remark:  walletRemark,
		})
		if wErr != nil {
			return nil, mapRefundErr(wErr)
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
