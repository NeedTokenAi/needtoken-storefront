package contract

import (
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/shopspring/decimal"
)

const (
	WalletBonusTypeFixed   = "fixed"
	WalletBonusTypePercent = "percent"
)

// WalletRechargeBonusTier 定义单笔满充阶梯赠送规则。
type WalletRechargeBonusTier struct {
	MinAmount  float64 `json:"min_amount"`
	BonusType  string  `json:"bonus_type"`
	BonusValue float64 `json:"bonus_value"`
}

// WalletRechargeBonusConfig 定义钱包充值福利配置（首充加赠 + 阶梯满充赠送）。
type WalletRechargeBonusConfig struct {
	Enabled                 bool                      `json:"enabled"`
	FirstRechargeBonusType  string                    `json:"first_recharge_bonus_type"`
	FirstRechargeBonusValue float64                   `json:"first_recharge_bonus_value"`
	FirstRechargeMinAmount  float64                   `json:"first_recharge_min_amount"`
	Tiers                   []WalletRechargeBonusTier `json:"tiers"`
	MaxBonusPerOrder        float64                   `json:"max_bonus_per_order"`
	CostRatePercent         float64                   `json:"cost_rate_percent"`
}

// DefaultWalletRechargeBonusConfig 返回默认关闭且成本比例 100% 的充值福利配置。
func DefaultWalletRechargeBonusConfig() WalletRechargeBonusConfig {
	return WalletRechargeBonusConfig{
		Enabled:                 false,
		FirstRechargeBonusType:  WalletBonusTypeFixed,
		FirstRechargeBonusValue: 0,
		FirstRechargeMinAmount:  0,
		Tiers:                   []WalletRechargeBonusTier{},
		MaxBonusPerOrder:        0,
		CostRatePercent:         100,
	}
}

func normalizeWalletBonusType(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case WalletBonusTypePercent:
		return WalletBonusTypePercent
	default:
		return WalletBonusTypeFixed
	}
}

func roundTwoDecimals(v float64) float64 {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return 0
	}
	return math.Round(v*100) / 100
}

// NormalizeWalletRechargeBonusConfig 规范化数值范围、排序阶梯并兜底默认成本比例。
func NormalizeWalletRechargeBonusConfig(cfg WalletRechargeBonusConfig) WalletRechargeBonusConfig {
	cfg.FirstRechargeBonusType = normalizeWalletBonusType(cfg.FirstRechargeBonusType)
	cfg.FirstRechargeBonusValue = max(0, roundTwoDecimals(cfg.FirstRechargeBonusValue))
	if cfg.FirstRechargeBonusType == WalletBonusTypePercent && cfg.FirstRechargeBonusValue > 1000 {
		cfg.FirstRechargeBonusValue = 1000
	}
	cfg.FirstRechargeMinAmount = max(0, roundTwoDecimals(cfg.FirstRechargeMinAmount))
	cfg.MaxBonusPerOrder = max(0, roundTwoDecimals(cfg.MaxBonusPerOrder))
	if cfg.CostRatePercent < 0 {
		cfg.CostRatePercent = 0
	} else if cfg.CostRatePercent > 100 {
		cfg.CostRatePercent = 100
	}
	// 仅在显式未设置或负值外场景保留 0..100，若解码时未传字段则在 Decode 中初始化为 100。
	cfg.CostRatePercent = roundTwoDecimals(cfg.CostRatePercent)

	tiers := make([]WalletRechargeBonusTier, 0, len(cfg.Tiers))
	for _, tier := range cfg.Tiers {
		minAmt := roundTwoDecimals(tier.MinAmount)
		val := roundTwoDecimals(tier.BonusValue)
		if minAmt <= 0 || val <= 0 {
			continue
		}
		bType := normalizeWalletBonusType(tier.BonusType)
		if bType == WalletBonusTypePercent && val > 1000 {
			val = 1000
		}
		tiers = append(tiers, WalletRechargeBonusTier{
			MinAmount:  minAmt,
			BonusType:  bType,
			BonusValue: val,
		})
	}
	// 同门槛保留后者覆盖并按门槛升序排列。
	dedup := make(map[float64]WalletRechargeBonusTier, len(tiers))
	for _, tier := range tiers {
		dedup[tier.MinAmount] = tier
	}
	normalizedTiers := make([]WalletRechargeBonusTier, 0, len(dedup))
	for _, tier := range dedup {
		normalizedTiers = append(normalizedTiers, tier)
	}
	sort.Slice(normalizedTiers, func(i, j int) bool {
		return normalizedTiers[i].MinAmount < normalizedTiers[j].MinAmount
	})
	cfg.Tiers = normalizedTiers
	return cfg
}

func parseConfigFloat(raw any) (float64, bool) {
	switch v := raw.(type) {
	case float64:
		return v, true
	case float32:
		return float64(v), true
	case int:
		return float64(v), true
	case int64:
		return float64(v), true
	case string:
		d, err := decimal.NewFromString(strings.TrimSpace(v))
		if err != nil {
			return 0, false
		}
		f, _ := d.Float64()
		return f, true
	default:
		return 0, false
	}
}

func parseConfigBool(raw any) bool {
	switch v := raw.(type) {
	case bool:
		return v
	case string:
		s := strings.ToLower(strings.TrimSpace(v))
		return s == "true" || s == "1" || s == "yes"
	case float64:
		return v != 0
	case int:
		return v != 0
	default:
		return false
	}
}

// DecodeWalletRechargeBonusConfig 从 wallet_config.recharge_bonus map 安全解码配置。
func DecodeWalletRechargeBonusConfig(raw map[string]any) WalletRechargeBonusConfig {
	cfg := DefaultWalletRechargeBonusConfig()
	if raw == nil {
		return cfg
	}
	if v, ok := raw["enabled"]; ok {
		cfg.Enabled = parseConfigBool(v)
	}
	if v, ok := raw["first_recharge_bonus_type"].(string); ok {
		cfg.FirstRechargeBonusType = v
	}
	if v, ok := parseConfigFloat(raw["first_recharge_bonus_value"]); ok {
		cfg.FirstRechargeBonusValue = v
	}
	if v, ok := parseConfigFloat(raw["first_recharge_min_amount"]); ok {
		cfg.FirstRechargeMinAmount = v
	}
	if v, ok := parseConfigFloat(raw["max_bonus_per_order"]); ok {
		cfg.MaxBonusPerOrder = v
	}
	if v, ok := parseConfigFloat(raw["cost_rate_percent"]); ok {
		cfg.CostRatePercent = v
	}
	if rawTiers, ok := raw["tiers"].([]any); ok {
		for _, item := range rawTiers {
			m, ok := item.(map[string]any)
			if !ok {
				continue
			}
			minAmt, _ := parseConfigFloat(m["min_amount"])
			val, _ := parseConfigFloat(m["bonus_value"])
			bType, _ := m["bonus_type"].(string)
			cfg.Tiers = append(cfg.Tiers, WalletRechargeBonusTier{
				MinAmount:  minAmt,
				BonusType:  bType,
				BonusValue: val,
			})
		}
	} else if typedTiers, ok := raw["tiers"].([]WalletRechargeBonusTier); ok {
		cfg.Tiers = append(cfg.Tiers, typedTiers...)
	} else if mapTiers, ok := raw["tiers"].([]map[string]any); ok {
		for _, m := range mapTiers {
			minAmt, _ := parseConfigFloat(m["min_amount"])
			val, _ := parseConfigFloat(m["bonus_value"])
			bType, _ := m["bonus_type"].(string)
			cfg.Tiers = append(cfg.Tiers, WalletRechargeBonusTier{
				MinAmount:  minAmt,
				BonusType:  bType,
				BonusValue: val,
			})
		}
	}
	return NormalizeWalletRechargeBonusConfig(cfg)
}

// EncodeWalletRechargeBonusConfig 编码为稳定的持久化 map（存入 wallet_config.recharge_bonus）。
func EncodeWalletRechargeBonusConfig(cfg WalletRechargeBonusConfig) map[string]any {
	normalized := NormalizeWalletRechargeBonusConfig(cfg)
	tiers := make([]map[string]any, 0, len(normalized.Tiers))
	for _, tier := range normalized.Tiers {
		tiers = append(tiers, map[string]any{
			"min_amount":  tier.MinAmount,
			"bonus_type":  tier.BonusType,
			"bonus_value": tier.BonusValue,
		})
	}
	return map[string]any{
		"enabled":                    normalized.Enabled,
		"first_recharge_bonus_type":  normalized.FirstRechargeBonusType,
		"first_recharge_bonus_value": normalized.FirstRechargeBonusValue,
		"first_recharge_min_amount":  normalized.FirstRechargeMinAmount,
		"tiers":                      tiers,
		"max_bonus_per_order":        normalized.MaxBonusPerOrder,
		"cost_rate_percent":          normalized.CostRatePercent,
	}
}

// PublicMap 返回面向前台公开配置（隐藏内部成本折算比例）。
func (c WalletRechargeBonusConfig) PublicMap() map[string]any {
	normalized := NormalizeWalletRechargeBonusConfig(c)
	tiers := make([]map[string]any, 0, len(normalized.Tiers))
	for _, tier := range normalized.Tiers {
		tiers = append(tiers, map[string]any{
			"min_amount":  tier.MinAmount,
			"bonus_type":  tier.BonusType,
			"bonus_value": tier.BonusValue,
		})
	}
	return map[string]any{
		"enabled":                    normalized.Enabled,
		"first_recharge_bonus_type":  normalized.FirstRechargeBonusType,
		"first_recharge_bonus_value": normalized.FirstRechargeBonusValue,
		"first_recharge_min_amount":  normalized.FirstRechargeMinAmount,
		"tiers":                      tiers,
		"max_bonus_per_order":        normalized.MaxBonusPerOrder,
	}
}

func computeBonusPart(amount decimal.Decimal, bonusType string, bonusValue float64) decimal.Decimal {
	if bonusValue <= 0 || amount.LessThanOrEqual(decimal.Zero) {
		return decimal.Zero
	}
	val := decimal.NewFromFloat(bonusValue)
	if normalizeWalletBonusType(bonusType) == WalletBonusTypePercent {
		return amount.Mul(val).Div(decimal.NewFromInt(100)).Round(2)
	}
	return val.Round(2)
}

// CalculateBonus 计算单笔充值的赠金与展示说明（阶梯最高命中档 + 首充加赠，按上限截断）。
func (c WalletRechargeBonusConfig) CalculateBonus(amount decimal.Decimal, isFirstRecharge bool) (decimal.Decimal, string) {
	normalized := NormalizeWalletRechargeBonusConfig(c)
	if !normalized.Enabled || amount.LessThanOrEqual(decimal.Zero) {
		return decimal.Zero, ""
	}
	total := decimal.Zero
	parts := make([]string, 0, 2)

	var matchedTier *WalletRechargeBonusTier
	for i := range normalized.Tiers {
		tier := normalized.Tiers[i]
		if amount.GreaterThanOrEqual(decimal.NewFromFloat(tier.MinAmount)) {
			matchedTier = &tier
		}
	}
	if matchedTier != nil {
		tierBonus := computeBonusPart(amount, matchedTier.BonusType, matchedTier.BonusValue)
		if tierBonus.GreaterThan(decimal.Zero) {
			total = total.Add(tierBonus)
			parts = append(parts, fmt.Sprintf("满%s赠", decimal.NewFromFloat(matchedTier.MinAmount).StringFixed(2)))
		}
	}

	if isFirstRecharge && normalized.FirstRechargeBonusValue > 0 &&
		amount.GreaterThanOrEqual(decimal.NewFromFloat(normalized.FirstRechargeMinAmount)) {
		firstBonus := computeBonusPart(amount, normalized.FirstRechargeBonusType, normalized.FirstRechargeBonusValue)
		if firstBonus.GreaterThan(decimal.Zero) {
			total = total.Add(firstBonus)
			parts = append(parts, "首充加赠")
		}
	}

	total = total.Round(2)
	if total.LessThanOrEqual(decimal.Zero) {
		return decimal.Zero, ""
	}
	if normalized.MaxBonusPerOrder > 0 {
		capVal := decimal.NewFromFloat(normalized.MaxBonusPerOrder)
		if total.GreaterThan(capVal) {
			total = capVal
		}
	}
	if total.LessThanOrEqual(decimal.Zero) {
		return decimal.Zero, ""
	}
	return total, "充值福利（" + strings.Join(parts, " + ") + "）"
}
