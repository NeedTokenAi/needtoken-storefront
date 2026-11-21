package opsbridge

import (
	"errors"
	"strings"
	"time"

	"github.com/dujiao-next/internal/app/container"
	"github.com/dujiao-next/internal/constants"
	opsapp "github.com/dujiao-next/internal/modules/opsbridge/application"
	opscontract "github.com/dujiao-next/internal/modules/opsbridge/contract"
	opstransport "github.com/dujiao-next/internal/modules/opsbridge/transport/http"
	settingsapp "github.com/dujiao-next/internal/modules/settings/application"
	siteconnectionapp "github.com/dujiao-next/internal/modules/siteconnection/application"
	siteconnectioncontract "github.com/dujiao-next/internal/modules/siteconnection/contract"
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
