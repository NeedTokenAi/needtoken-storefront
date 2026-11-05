package application

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"github.com/dujiao-next/internal/constants"
	mappingcontract "github.com/dujiao-next/internal/modules/catalog/mapping/contract"
	mappingdomain "github.com/dujiao-next/internal/modules/catalog/mapping/domain"
	"github.com/dujiao-next/internal/upstream"
)

// 文件组织约定:
//   service.go       — 端口/错误/装配 + 基础查询与 CRUD
//   import.go        — 单品导入 + 上游元数据列表 + 图片下载
//   batch_import.go  — 批量导入与上游分类自动创建
//   sync.go          — 同步流程（单品 / 全量库存 / 下单前兜底）
//   markup.go        — 加价重算
//   pricing.go       — 汇率与加价换算

type Options struct {
	Mappings     mappingcontract.MappingRepository
	SKUMappings  mappingcontract.SKUMappingRepository
	Products     mappingcontract.ProductRepository
	SKUs         mappingcontract.SKURepository
	Categories   mappingcontract.CategoryRepository
	Connections  mappingcontract.ConnectionProvider
	Media        mappingcontract.MediaRecorder
	Transactions mappingcontract.UnitOfWork
}

// Service 承载本地商品与上游站点之间的映射用例。
type Service struct {
	mappings        mappingcontract.MappingRepository
	skuMappings     mappingcontract.SKUMappingRepository
	products        mappingcontract.ProductRepository
	skus            mappingcontract.SKURepository
	categories      mappingcontract.CategoryRepository
	connections     mappingcontract.ConnectionProvider
	media           mappingcontract.MediaRecorder
	transactions    mappingcontract.UnitOfWork
	categoryCreator mappingcontract.CategoryCreator
	settings        mappingcontract.SettingsProvider
}

func NewService(options Options) (*Service, error) {
	if options.Media == nil {
		return nil, mappingcontract.ErrMediaRecorderRequired
	}
	return &Service{
		mappings:     options.Mappings,
		skuMappings:  options.SKUMappings,
		products:     options.Products,
		skus:         options.SKUs,
		categories:   options.Categories,
		connections:  options.Connections,
		media:        options.Media,
		transactions: options.Transactions,
	}, nil
}

// SetCategoryCreator 注入分类创建端口（装配时调用）。
func (s *Service) SetCategoryCreator(creator mappingcontract.CategoryCreator) {
	s.categoryCreator = creator
}

// SetSettings 注入设置端口（用于读取上游同步动态配置）。
func (s *Service) SetSettings(settings mappingcontract.SettingsProvider) {
	s.settings = settings
}

// GetByID 获取映射详情
func (s *Service) GetByID(id uint) (*mappingdomain.Mapping, error) {
	return s.mappings.GetByID(id)
}

// List 列表查询映射
func (s *Service) List(filter mappingcontract.ListFilter) ([]mappingdomain.Mapping, int64, error) {
	return s.mappings.List(filter)
}

// SetActive 启用/禁用映射
func (s *Service) SetActive(id uint, active bool) error {
	mapping, err := s.mappings.GetByID(id)
	if err != nil {
		return err
	}
	if mapping == nil {
		return mappingcontract.ErrMappingNotFound
	}
	mapping.IsActive = active
	return s.mappings.Update(mapping)
}

// Delete 删除映射（不删除本地商品）
func (s *Service) Delete(id uint) error {
	mapping, err := s.mappings.GetByID(id)
	if err != nil {
		return err
	}
	if mapping == nil {
		return mappingcontract.ErrMappingNotFound
	}

	// 删除 SKU 映射
	if err := s.skuMappings.DeleteByProductMapping(id); err != nil {
		return err
	}

	// 还原本地商品状态：取消映射标记、交付类型改回 manual、自动下架
	if mapping.LocalProductID > 0 {
		localProduct, err := s.products.GetByID(strconv.FormatUint(uint64(mapping.LocalProductID), 10))
		if err == nil && localProduct != nil {
			localProduct.IsMapped = false
			if localProduct.FulfillmentType == constants.FulfillmentTypeUpstream {
				localProduct.FulfillmentType = constants.FulfillmentTypeManual
				localProduct.IsActive = false // 下架，防止用户下单后无法交付
			}
			_ = s.products.Update(localProduct)
		}
	}

	return s.mappings.Delete(id)
}

// GetSKUMappings 获取映射的 SKU 映射列表
func (s *Service) GetSKUMappings(mappingID uint) ([]mappingdomain.SKUMapping, error) {
	return s.skuMappings.ListByProductMapping(mappingID)
}

// GetMappedUpstreamIDs 获取指定连接下所有已映射的上游商品 ID
func (s *Service) GetMappedUpstreamIDs(connectionID uint) ([]uint, error) {
	return s.mappings.ListUpstreamIDsByConnection(connectionID)
}

// PreparePlanCheckout resolves real SKU mappings before freezing the route. This
// use case is deliberately separate from the stock check's optional fail-open
// behavior: an unavailable account or entitlement always blocks a plan checkout.
func (s *Service) PreparePlanCheckout(localSKUID, userID uint, quantity int, contract json.RawMessage, hash string) (mappingcontract.PlanCheckoutRoute, error) {
	var route mappingcontract.PlanCheckoutRoute
	if s == nil || s.skuMappings == nil || s.mappings == nil || s.skus == nil || localSKUID == 0 {
		return route, mappingcontract.ErrPlanEligibilityUnavailable
	}
	skuMapping, err := s.skuMappings.GetByLocalSKUID(localSKUID)
	if err != nil || skuMapping == nil {
		return route, mappingcontract.ErrPlanEligibilityUnavailable
	}
	mapping, err := s.mappings.GetByID(skuMapping.ProductMappingID)
	if err != nil || mapping == nil {
		return route, mappingcontract.ErrPlanEligibilityUnavailable
	}
	if !mapping.IsActive || mapping.DeletedAt != nil || mapping.UpstreamStatus != mappingdomain.UpstreamStatusActive || !skuMapping.UpstreamIsActive || skuMapping.DeletedAt != nil {
		return route, mappingcontract.ErrPlanUnavailable
	}
	sku, err := s.skus.GetByID(localSKUID)
	if err != nil || sku == nil {
		return route, mappingcontract.ErrPlanEligibilityUnavailable
	}
	if !sku.IsActive || sku.ProductID != mapping.LocalProductID || skuMapping.UpstreamSKUID == 0 || mapping.ConnectionID == 0 {
		return route, mappingcontract.ErrPlanUnavailable
	}
	route = mappingcontract.PlanCheckoutRoute{ConnectionID: mapping.ConnectionID, UpstreamSKUID: skuMapping.UpstreamSKUID}
	if err := s.CheckPlanCheckout(mappingcontract.PlanCheckoutRequest{Route: route, UserID: userID, Quantity: quantity, EntitlementContract: contract, EntitlementContractHash: hash}); err != nil {
		return mappingcontract.PlanCheckoutRoute{}, err
	}
	return route, nil
}

// CheckPlanCheckout also serves payment retries. It only reads the frozen route
// and immutable terms supplied by the server-side order snapshot; it never
// substitutes current SKU terms or resolves a submitted email into an identity.
func (s *Service) CheckPlanCheckout(request mappingcontract.PlanCheckoutRequest) error {
	if request.UserID == 0 {
		return mappingcontract.ErrPlanAccountLinkRequired
	}
	if request.Route.ConnectionID == 0 || request.Route.UpstreamSKUID == 0 || request.Quantity != 1 {
		return mappingcontract.ErrPlanChanged
	}
	var shape struct {
		Kind        string `json:"kind"`
		IssueAPIKey bool   `json:"issue_api_key"`
	}
	if len(request.EntitlementContract) == 0 || len(request.EntitlementContract) > 128*1024 || json.Unmarshal(request.EntitlementContract, &shape) != nil || shape.Kind != "plan" || !shape.IssueAPIKey {
		return mappingcontract.ErrPlanChanged
	}
	if hash, err := hex.DecodeString(request.EntitlementContractHash); err != nil || len(hash) != 32 || strings.ToLower(request.EntitlementContractHash) != request.EntitlementContractHash {
		return mappingcontract.ErrPlanChanged
	}
	if s == nil || s.connections == nil {
		return mappingcontract.ErrPlanEligibilityUnavailable
	}
	connection, err := s.connections.GetByID(request.Route.ConnectionID)
	if err != nil || connection == nil || connection.Status != constants.ConnectionStatusActive || connection.DeletedAt != nil {
		return mappingcontract.ErrPlanEligibilityUnavailable
	}
	adapter, err := s.connections.GetAdapter(connection)
	if err != nil {
		return mappingcontract.ErrPlanEligibilityUnavailable
	}
	checker, ok := adapter.(upstream.PlanCheckoutAdapter)
	if !ok {
		return mappingcontract.ErrPlanEligibilityUnavailable
	}
	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
	defer cancel()
	result, err := checker.CheckPlanCheckoutEligibility(ctx, upstream.PlanCheckoutEligibilityRequest{
		SKUID: request.Route.UpstreamSKUID, Quantity: request.Quantity,
		EntitlementContract: request.EntitlementContract, EntitlementContractHash: request.EntitlementContractHash,
		CustomerReference: upstream.PlanCheckoutCustomerReference{
			MerchantCustomerID: "needtoken-storefront:user:" + strconv.FormatUint(uint64(request.UserID), 10),
			Source:             "needtoken-storefront", Verification: "merchant_session",
		},
	})
	now := time.Now()
	if err != nil || result == nil || !result.OK || ctx.Err() != nil || result.SKUID != request.Route.UpstreamSKUID || result.EntitlementContractHash != request.EntitlementContractHash || result.CollectedAt.IsZero() || result.CollectedAt.Before(now.Add(-15*time.Second)) || result.CollectedAt.After(now.Add(5*time.Second)) {
		return mappingcontract.ErrPlanEligibilityUnavailable
	}
	if result.Ready && result.ReasonCode == "ready" {
		return nil
	}
	switch result.ReasonCode {
	case "account_link_required":
		return mappingcontract.ErrPlanAccountLinkRequired
	case "account_unavailable":
		return mappingcontract.ErrPlanAccountUnavailable
	case "plan_unavailable":
		return mappingcontract.ErrPlanUnavailable
	case "plan_changed":
		return mappingcontract.ErrPlanChanged
	case "purchase_limit_reached":
		return mappingcontract.ErrPlanPurchaseLimitReached
	case "token_limit_reached":
		return mappingcontract.ErrPlanTokenLimitReached
	default:
		return mappingcontract.ErrPlanEligibilityUnavailable
	}
}

var _ mappingcontract.PlanCheckoutPreparer = (*Service)(nil)
var _ mappingcontract.PlanCheckoutChecker = (*Service)(nil)
