package productdomain

import (
	"errors"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/dujiao-next/internal/shared/jsonmap"
	"github.com/dujiao-next/internal/shared/money"
	"github.com/shopspring/decimal"
)

const (
	// DefaultSKUCode 历史单规格商品迁移时使用的默认 SKU 编码
	DefaultSKUCode = "DEFAULT"
)

// ProductSKU 商品 SKU 表（v1：价格+库存维度）
type ProductSKU struct {
	// Server-sourced upstream terms. They are not editable through public SKU DTOs.
	EntitlementContractJSON  jsonmap.ExactJSON `gorm:"type:json" json:"-"`
	EntitlementContractHash  string            `gorm:"size:64" json:"-"`
	RequiresReservation      bool              `gorm:"not null;default:false" json:"-"`
	ReservationPolicyKnown   bool              `gorm:"column:reservation_policy_known;not null;default:false" json:"-"`
	ReservationContractJSON  jsonmap.JSON      `gorm:"type:json" json:"-"`
	ReservationConnectionID  uint              `gorm:"not null;default:0" json:"-"`
	ReservationUpstreamSKUID uint              `gorm:"column:reservation_upstream_sku_id;not null;default:0" json:"-"`
	ID                       uint              `gorm:"primarykey" json:"id"`                                                                       // 主键
	ProductID                uint              `gorm:"not null;index;uniqueIndex:idx_product_sku_code" json:"product_id"`                          // 商品ID
	SKUCode                  string            `gorm:"column:sku_code;type:varchar(64);not null;uniqueIndex:idx_product_sku_code" json:"sku_code"` // SKU编码（同商品内唯一）
	SpecValuesJSON           jsonmap.JSON      `gorm:"type:json" json:"spec_values"`                                                               // 规格值（如颜色/版本）
	DisplayMetaJSON          jsonmap.JSON      `gorm:"type:json" json:"display_meta"`                                                              // 商户维护的规格展示信息；不参与价格与履约计算
	PriceAmount              money.Amount      `gorm:"type:decimal(20,2);not null;default:0" json:"price_amount"`                                  // SKU价格
	CostPriceAmount          money.Amount      `gorm:"type:decimal(20,2);not null;default:0" json:"cost_price_amount"`                             // 成本价
	ManualStockTotal         int               `gorm:"not null;default:0" json:"manual_stock_total"`                                               // 手动剩余库存（-1 表示无限库存，>=0 表示当前可售数量）
	ManualStockLocked        int               `gorm:"not null;default:0" json:"manual_stock_locked"`                                              // 手动库存占用量（待支付）
	ManualStockSold          int               `gorm:"not null;default:0" json:"manual_stock_sold"`                                                // 手动库存已售量（支付成功后累加）
	AutoStockAvailable       int64             `gorm:"-" json:"auto_stock_available"`                                                              // 自动发货库存可用量（仅结构，不写入数据库）
	AutoStockTotal           int64             `gorm:"-" json:"auto_stock_total"`                                                                  // 自动发货库存总量（仅结构，不写入数据库）
	AutoStockLocked          int64             `gorm:"-" json:"auto_stock_locked"`                                                                 // 自动发货库存占用量（仅结构，不写入数据库）
	AutoStockSold            int64             `gorm:"-" json:"auto_stock_sold"`                                                                   // 自动发货库存已售量（仅结构，不写入数据库）
	UpstreamStock            int               `gorm:"-" json:"upstream_stock"`                                                                    // 上游库存（-1=无限, 0=售罄, >0=有货；仅结构，不写入数据库）
	IsActive                 bool              `gorm:"default:true;index" json:"is_active"`                                                        // 是否启用
	SortOrder                int               `gorm:"default:0;index" json:"sort_order"`                                                          // 排序权重
	CreatedAt                time.Time         `gorm:"index" json:"created_at"`                                                                    // 创建时间
	UpdatedAt                time.Time         `gorm:"index" json:"updated_at"`                                                                    // 更新时间
	DeletedAt                *time.Time        `gorm:"index" json:"-"`                                                                             // 软删除时间

	Product *Product `gorm:"foreignKey:ProductID" json:"product,omitempty"` // 关联商品
}

// TableName 指定表名
func (ProductSKU) TableName() string {
	return "product_skus"
}

var ErrSKUDisplayMetaInvalid = errors.New("sku display metadata invalid")

var officialUSDPattern = regexp.MustCompile(`^(?:0|[1-9][0-9]{0,7})(?:\.[0-9]{1,2})?$`)

// NormalizeSKUDisplayMeta accepts only public merchandising copy. In particular,
// official_price_usd is an optional reference, never a sale price or entitlement.
func NormalizeSKUDisplayMeta(raw map[string]interface{}) (jsonmap.JSON, error) {
	limits := map[string]int{"label": 80, "title": 160, "description": 1000, "feature_lines": 1500, "price_suffix": 32}
	result := make(jsonmap.JSON, len(limits)+1)
	for field, limit := range limits {
		localized := map[string]interface{}{"zh-CN": "", "zh-TW": "", "en-US": ""}
		if value, exists := raw[field]; exists && value != nil {
			values, ok := value.(map[string]interface{})
			if !ok {
				// Internal callers may already hold normalized jsonmap values.
				if mapped, valid := value.(jsonmap.JSON); valid {
					values, ok = map[string]interface{}(mapped), true
				}
			}
			if !ok {
				return nil, ErrSKUDisplayMetaInvalid
			}
			for locale := range localized {
				if value, exists := values[locale]; exists && value != nil {
					text, ok := value.(string)
					if !ok || !utf8.ValidString(text) {
						return nil, ErrSKUDisplayMetaInvalid
					}
					text = strings.TrimSpace(strings.ReplaceAll(text, "\r\n", "\n"))
					if utf8.RuneCountInString(text) > limit || strings.ContainsRune(text, '\x00') {
						return nil, ErrSKUDisplayMetaInvalid
					}
					if field == "feature_lines" {
						lines := make([]string, 0, 8)
						for _, line := range strings.Split(text, "\n") {
							line = strings.TrimSpace(line)
							if line == "" {
								continue
							}
							if len(lines) == 8 || utf8.RuneCountInString(line) > 160 {
								return nil, ErrSKUDisplayMetaInvalid
							}
							lines = append(lines, line)
						}
						text = strings.Join(lines, "\n")
					}
					localized[locale] = text
				}
			}
		}
		result[field] = localized
	}
	result["official_price_usd"] = ""
	if value, exists := raw["official_price_usd"]; exists && value != nil {
		text, ok := value.(string)
		if !ok {
			return nil, ErrSKUDisplayMetaInvalid
		}
		text = strings.TrimSpace(text)
		if text != "" {
			if !officialUSDPattern.MatchString(text) {
				return nil, ErrSKUDisplayMetaInvalid
			}
			amount, err := decimal.NewFromString(text)
			if err != nil || !amount.IsPositive() {
				return nil, ErrSKUDisplayMetaInvalid
			}
			result["official_price_usd"] = amount.StringFixed(2)
		}
	}
	return result, nil
}

// PublicSKUDisplayMeta keeps the same allowlist even for legacy/imported rows.
func PublicSKUDisplayMeta(raw jsonmap.JSON) jsonmap.JSON {
	if len(raw) == 0 {
		return nil
	}
	normalized, err := NormalizeSKUDisplayMeta(map[string]interface{}(raw))
	if err != nil {
		normalized, _ = NormalizeSKUDisplayMeta(nil)
	}
	return normalized
}
