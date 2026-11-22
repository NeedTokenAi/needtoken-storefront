package application

import (
	crand "crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/dujiao-next/internal/constants"
	categorycontract "github.com/dujiao-next/internal/modules/catalog/category/contract"
	categorydomain "github.com/dujiao-next/internal/modules/catalog/category/domain"
	productcontract "github.com/dujiao-next/internal/modules/catalog/product/contract"
	productdomain "github.com/dujiao-next/internal/modules/catalog/product/domain"
	couponcontract "github.com/dujiao-next/internal/modules/coupon/contract"
	coupondomain "github.com/dujiao-next/internal/modules/coupon/domain"
	giftcardcontract "github.com/dujiao-next/internal/modules/giftcard/contract"
	giftcarddomain "github.com/dujiao-next/internal/modules/giftcard/domain"
	opscontract "github.com/dujiao-next/internal/modules/opsbridge/contract"
	ordercontract "github.com/dujiao-next/internal/modules/order/contract"
	orderdomain "github.com/dujiao-next/internal/modules/order/domain"
	paymentcontract "github.com/dujiao-next/internal/modules/payment/contract"
	procurementcontract "github.com/dujiao-next/internal/modules/procurement/contract"
	"github.com/dujiao-next/internal/shared/jsonmap"
	"github.com/dujiao-next/internal/shared/money"
	"github.com/shopspring/decimal"
)

type Service struct {
	orders          ordercontract.Store
	payments        paymentcontract.Store
	procurements    procurementcontract.Repository
	categories      categorycontract.Repository
	categoryStore   opscontract.CategoryStore
	products        productcontract.Repository
	productDelete   opscontract.ProductDeleteStore
	skus            productcontract.SKURepository
	cardSecretStock opscontract.CardSecretStockCounter
	skuDeleteUoW    opscontract.SKUDeleteUnitOfWork
	coupons         couponcontract.Repository
	giftCards       giftcardcontract.Repository
	pricing         opscontract.PricingStore
	acquiring       opscontract.AcquiringStore
	upstream        opscontract.UpstreamStore
	refunds         opscontract.OrderRefundStore
	orderEnricher   opscontract.OrderEnricher
	orderActions    opscontract.OrderActionStore
	users           opscontract.UserStore
}

func New(orders ordercontract.Store, payments paymentcontract.Store, procurements procurementcontract.Repository) *Service {
	return &Service{orders: orders, payments: payments, procurements: procurements}
}

func NewWithCatalog(
	orders ordercontract.Store,
	payments paymentcontract.Store,
	procurements procurementcontract.Repository,
	categories categorycontract.Repository,
	products productcontract.Repository,
	skus productcontract.SKURepository,
	coupons couponcontract.Repository,
) *Service {
	return &Service{
		orders:       orders,
		payments:     payments,
		procurements: procurements,
		categories:   categories,
		products:     products,
		skus:         skus,
		coupons:      coupons,
	}
}

func (s *Service) WithPricing(pricing opscontract.PricingStore) *Service {
	s.pricing = pricing
	return s
}

func (s *Service) WithGiftCards(giftCards giftcardcontract.Repository) *Service {
	s.giftCards = giftCards
	return s
}

func (s *Service) WithAcquiring(acquiring opscontract.AcquiringStore) *Service {
	s.acquiring = acquiring
	return s
}

func (s *Service) WithUsers(users opscontract.UserStore) *Service {
	s.users = users
	return s
}

func (s *Service) WithUpstream(upstream opscontract.UpstreamStore) *Service {
	s.upstream = upstream
	return s
}

func (s *Service) WithOrderRefunds(refunds opscontract.OrderRefundStore) *Service {
	s.refunds = refunds
	return s
}

func (s *Service) WithOrderEnricher(enricher opscontract.OrderEnricher) *Service {
	s.orderEnricher = enricher
	return s
}

func (s *Service) WithOrderActions(actions opscontract.OrderActionStore) *Service {
	s.orderActions = actions
	return s
}

func (s *Service) WithCategoryStore(store opscontract.CategoryStore) *Service {
	s.categoryStore = store
	return s
}

func (s *Service) WithProductDeleteStore(store opscontract.ProductDeleteStore) *Service {
	s.productDelete = store
	return s
}

func (s *Service) WithCardSecretStock(counter opscontract.CardSecretStockCounter) *Service {
	s.cardSecretStock = counter
	return s
}

func (s *Service) WithSKUDeleteUoW(uow opscontract.SKUDeleteUnitOfWork) *Service {
	s.skuDeleteUoW = uow
	return s
}

func (s *Service) Orders(page, pageSize int) (*opscontract.OrdersResult, error) {
	return s.OrdersFiltered(opscontract.OrdersFilter{Page: page, PageSize: pageSize})
}

func (s *Service) OrdersFiltered(filter opscontract.OrdersFilter) (*opscontract.OrdersResult, error) {
	collected := time.Now().UTC()
	page := filter.Page
	pageSize := filter.PageSize
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 50
	}
	if pageSize > 100 {
		pageSize = 100
	}
	rows, total, err := s.orders.ListAdmin(ordercontract.ListFilter{
		Page:            page,
		PageSize:        pageSize,
		Status:          strings.TrimSpace(filter.Status),
		OrderNo:         strings.TrimSpace(filter.OrderNo),
		Keyword:         strings.TrimSpace(filter.Keyword),
		CategoryID:      filter.CategoryID,
		FulfillmentType: strings.TrimSpace(filter.FulfillmentType),
		BuyerType:       strings.TrimSpace(filter.BuyerType),
		PaymentChannel:  strings.TrimSpace(filter.PaymentChannel),
		ClientIP:        strings.TrimSpace(filter.ClientIP),
		ExceptionOnly:   filter.ExceptionOnly,
		CreatedFrom:     filter.CreatedFrom,
		CreatedTo:       filter.CreatedTo,
		SortBy:          "id",
		SortOrder:       "desc",
	})
	if err != nil {
		return nil, err
	}

	buyerByID := map[uint]opscontract.OrderBuyerInfo{}
	productCategoryByID := map[uint]uint{}
	userIDSet := map[uint]struct{}{}
	productIDSet := map[uint]struct{}{}
	var collectIDs func(orderdomain.Order)
	collectIDs = func(row orderdomain.Order) {
		if row.UserID > 0 {
			userIDSet[row.UserID] = struct{}{}
		}
		for _, item := range row.Items {
			if item.ProductID > 0 {
				productIDSet[item.ProductID] = struct{}{}
			}
		}
		for _, child := range row.Children {
			collectIDs(child)
		}
	}
	for _, row := range rows {
		collectIDs(row)
	}
	if s.orderEnricher != nil && len(userIDSet) > 0 {
		ids := make([]uint, 0, len(userIDSet))
		for id := range userIDSet {
			ids = append(ids, id)
		}
		if looked, err := s.orderEnricher.LookupBuyers(ids); err == nil && looked != nil {
			buyerByID = looked
		}
	}
	if s.products != nil && len(productIDSet) > 0 {
		ids := make([]uint, 0, len(productIDSet))
		for id := range productIDSet {
			ids = append(ids, id)
		}
		if loaded, err := s.products.ListByIDs(ids); err == nil {
			for _, p := range loaded {
				productCategoryByID[p.ID] = p.CategoryID
			}
		}
	}

	result := &opscontract.OrdersResult{
		Source:      "storefront",
		CollectedAt: collected,
		Freshness:   "fresh",
		Orders:      make([]opscontract.Order, 0, len(rows)),
		Total:       total,
		Page:        page,
		PageSize:    pageSize,
	}
	var mapOrder func(orderdomain.Order) (opscontract.Order, error)
	mapOrder = func(row orderdomain.Order) (opscontract.Order, error) {
		buyerType := "guest"
		buyerEmail := ""
		buyerName := ""
		if row.UserID > 0 {
			buyerType = "member"
			if info, ok := buyerByID[row.UserID]; ok {
				buyerEmail = strings.TrimSpace(info.Email)
				buyerName = strings.TrimSpace(info.DisplayName)
			}
		} else {
			buyerEmail = strings.TrimSpace(row.GuestEmail)
		}
		out := opscontract.Order{
			ID:                row.ID,
			OrderNo:           row.OrderNo,
			ParentID:          row.ParentID,
			UserID:            row.UserID,
			BuyerType:         buyerType,
			BuyerEmail:        buyerEmail,
			BuyerName:         buyerName,
			SourceDomain:      strings.TrimSpace(row.ResellerDomain),
			ClientIP:          strings.TrimSpace(row.ClientIP),
			UserAgent:         strings.TrimSpace(row.UserAgent),
			ClientFingerprint: strings.TrimSpace(row.ClientFingerprint),
			Status:            row.Status,
			Currency:          row.Currency,
			OriginalAmount:    row.OriginalAmount.String(),
			DiscountAmount:    row.DiscountAmount.String(),
			CouponID:          row.CouponID,
			TotalAmount:       row.TotalAmount.String(),
			WalletPaidAmount:  row.WalletPaidAmount.String(),
			OnlinePaidAmount:  row.OnlinePaidAmount.String(),
			RefundedAmount:    row.RefundedAmount.String(),
			PaymentStatus:     "unpaid",
			FulfillmentStatus: row.Status,
			EntitlementStatus: "unknown",
			PaidAt:            row.PaidAt,
			CreatedAt:         row.CreatedAt,
			UpdatedAt:         row.UpdatedAt,
			Items:             []opscontract.OrderItemSummary{},
			Payments:          []opscontract.Payment{},
			Refunds:           []opscontract.Refund{},
		}
		if row.PaidAt != nil {
			out.PaymentStatus = "paid"
		}
		if row.UserID > 0 {
			out.MerchantCustomerID = fmt.Sprintf("needtoken-storefront:user:%d", row.UserID)
		}
		if row.Fulfillment != nil {
			out.FulfillmentStatus = row.Fulfillment.Status
		}
		for _, item := range row.Items {
			skuCode := ""
			if item.SKUSnapshotJSON != nil {
				if raw, ok := item.SKUSnapshotJSON["sku_code"]; ok && raw != nil {
					skuCode = strings.TrimSpace(fmt.Sprint(raw))
				}
			}
			out.Items = append(out.Items, opscontract.OrderItemSummary{
				ID:              item.ID,
				OrderID:         item.OrderID,
				ProductID:       item.ProductID,
				CategoryID:      productCategoryByID[item.ProductID],
				SKUID:           item.SKUID,
				SKUCode:         skuCode,
				Title:           stringMap(item.TitleJSON),
				FulfillmentType: item.FulfillmentType,
				Quantity:        item.Quantity,
				UnitPrice:       item.UnitPrice.String(),
				TotalPrice:      item.TotalPrice.String(),
				Tags:            append([]string(nil), item.Tags...),
			})
		}

		exceptionSet := map[string]struct{}{}
		addException := func(reason string) {
			reason = strings.TrimSpace(reason)
			if reason == "" {
				return
			}
			if _, exists := exceptionSet[reason]; !exists {
				exceptionSet[reason] = struct{}{}
				out.ExceptionReasons = append(out.ExceptionReasons, reason)
			}
		}

		payments, err := s.payments.ListByOrderID(row.ID)
		if err != nil {
			return opscontract.Order{}, err
		}
		for _, payment := range payments {
			if code := strings.TrimSpace(payment.ExceptionCode); code != "" {
				addException("payment_exception:" + code)
			}
			if payment.Status == constants.PaymentStatusSuccess && row.OnlinePaidAmount.Decimal.GreaterThan(decimal.Zero) {
				netPaid := payment.Amount.Decimal.Sub(payment.FeeAmount.Decimal)
				if netPaid.Sub(row.OnlinePaidAmount.Decimal).Abs().GreaterThan(decimal.NewFromFloat(0.01)) {
					addException("amount_mismatch")
				}
			}
			out.Payments = append(out.Payments, opscontract.Payment{
				ID:                 payment.ID,
				ProviderType:       payment.ProviderType,
				ChannelType:        payment.ChannelType,
				PaymentEnvironment: payment.FrozenPaymentEnvironment(),
				Status:             payment.Status,
				Amount:             payment.Amount.String(),
				FeeAmount:          payment.FeeAmount.String(),
				Currency:           payment.Currency,
				ExceptionCode:      strings.TrimSpace(payment.ExceptionCode),
				GatewayOrderNo:     strings.TrimSpace(payment.GatewayOrderNo),
				ProviderRef:        payment.ProviderRef,
				CallbackAt:         payment.CallbackAt,
				CallbackSummary:    sanitizePaymentCallbackPayload(payment.ProviderPayload),
				PaidAt:             payment.PaidAt,
				UpdatedAt:          payment.UpdatedAt,
			})
		}
		refunds, err := s.orders.ListRefundRecordsByOrderIDs([]uint{row.ID})
		if err != nil {
			return opscontract.Order{}, err
		}
		for _, refund := range refunds {
			settlement := refund.SettlementStatus
			if settlement == "" {
				settlement = "unknown"
				if refund.Type == constants.OrderRefundTypeManual {
					settlement = "external_unverified"
				}
			}
			out.Refunds = append(out.Refunds, opscontract.Refund{ID: refund.ID, OperationID: refund.OperationID, Type: refund.Type, Amount: refund.Amount.String(), Currency: refund.Currency, SettlementStatus: settlement, UpdatedAt: refund.UpdatedAt})
		}
		procurement, err := s.procurements.GetByLocalOrderID(row.ID)
		if err != nil {
			return opscontract.Order{}, err
		}
		if procurement != nil {
			out.Procurement = &opscontract.Procurement{
				ID:              procurement.ID,
				Status:          procurement.Status,
				UpstreamOrderNo: procurement.UpstreamOrderNo,
				ErrorMessage:    strings.TrimSpace(procurement.ErrorMessage),
				RetryCount:      procurement.RetryCount,
				UpdatedAt:       procurement.UpdatedAt,
				LocalOrderID:    procurement.LocalOrderID,
			}
			if procurement.Status == constants.ProcurementStatusFailed || procurement.Status == constants.ProcurementStatusRejected {
				addException("procurement_failed")
			}
		}
		out.Children = []opscontract.Order{}
		out.Procurements = []opscontract.Procurement{}
		out.Reservations = []opscontract.Reservation{}
		reservationRootID := row.ID
		if row.ParentID != nil {
			reservationRootID = *row.ParentID
		}
		reservations, err := s.payments.ListCheckoutReservations(reservationRootID)
		if err != nil {
			return opscontract.Order{}, err
		}
		for _, r := range reservations {
			if r.OrderID != row.ID {
				continue
			}
			if code := strings.TrimSpace(r.AttentionCode); code != "" {
				addException("reservation_attention:" + code)
			}
			out.Reservations = append(out.Reservations, opscontract.Reservation{OrderItemID: r.OrderItemID, LocalOrderID: r.OrderID, HoldID: r.HoldID, OrderRef: r.OrderRef, Version: r.HoldVersion, Status: r.State, BookStatus: r.BookStatus, ReadyForPayment: r.ReadyForPayment, ExpiresAt: r.ExpiresAt, AttentionCode: r.AttentionCode, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt})
		}
		if out.Procurement != nil {
			out.Procurements = append(out.Procurements, *out.Procurement)
		}
		for _, child := range row.Children {
			mapped, err := mapOrder(child)
			if err != nil {
				return opscontract.Order{}, err
			}
			out.Children = append(out.Children, mapped)
			if len(out.Items) == 0 || row.ParentID == nil {
				out.Items = append(out.Items, mapped.Items...)
			}
			out.Procurements = append(out.Procurements, mapped.Procurements...)
			out.Reservations = append(out.Reservations, mapped.Reservations...)
			for _, reason := range mapped.ExceptionReasons {
				addException(reason)
			}
		}
		out.HasException = len(out.ExceptionReasons) > 0
		return out, nil
	}
	for _, row := range rows {
		mapped, err := mapOrder(row)
		if err != nil {
			return nil, err
		}
		result.Orders = append(result.Orders, mapped)
	}

	return result, nil
}

func sanitizePaymentCallbackPayload(raw jsonmap.JSON) map[string]any {
	if len(raw) == 0 {
		return nil
	}
	out := make(map[string]any, len(raw))
	for k, v := range raw {
		lower := strings.ToLower(strings.TrimSpace(k))
		if lower == "" || isSensitiveCallbackKey(lower) {
			continue
		}
		if nested, ok := v.(map[string]any); ok {
			cleaned := sanitizeCallbackMap(nested, 0)
			if len(cleaned) > 0 {
				out[k] = cleaned
			}
			continue
		}
		if nested, ok := v.(jsonmap.JSON); ok {
			cleaned := sanitizeCallbackMap(map[string]any(nested), 0)
			if len(cleaned) > 0 {
				out[k] = cleaned
			}
			continue
		}
		out[k] = v
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func sanitizeCallbackMap(m map[string]any, depth int) map[string]any {
	if depth > 2 || len(m) == 0 {
		return nil
	}
	out := make(map[string]any, len(m))
	for k, v := range m {
		lower := strings.ToLower(strings.TrimSpace(k))
		if lower == "" || isSensitiveCallbackKey(lower) {
			continue
		}
		if nested, ok := v.(map[string]any); ok {
			if cleaned := sanitizeCallbackMap(nested, depth+1); len(cleaned) > 0 {
				out[k] = cleaned
			}
			continue
		}
		out[k] = v
	}
	return out
}

func isSensitiveCallbackKey(lowerKey string) bool {
	for _, needle := range []string{"secret", "token", "password", "private", "signature", "sign", "authorization", "cookie", "api_key", "apikey", "credential", "cert"} {
		if strings.Contains(lowerKey, needle) {
			return true
		}
	}
	return false
}

func (s *Service) ManualConfirmPayment(orderNo string, input opscontract.ManualConfirmPaymentInput) (*opscontract.ManualConfirmPaymentResult, error) {
	if s.orderActions == nil {
		return nil, opscontract.ErrNotConfigured
	}
	orderNo = strings.TrimSpace(orderNo)
	if orderNo == "" {
		return nil, opscontract.ErrInvalidInput
	}
	return s.orderActions.ManualConfirmPayment(orderNo, input)
}

func (s *Service) ManualFulfillOrder(orderNo string, input opscontract.ManualFulfillOrderInput) (*opscontract.ManualFulfillOrderResult, error) {
	if s.orderActions == nil {
		return nil, opscontract.ErrNotConfigured
	}
	orderNo = strings.TrimSpace(orderNo)
	if orderNo == "" || strings.TrimSpace(input.Payload) == "" {
		return nil, opscontract.ErrInvalidInput
	}
	return s.orderActions.ManualFulfillOrder(orderNo, input)
}

func (s *Service) UpdateOrderStatus(orderNo string, input opscontract.UpdateOrderStatusInput) (*opscontract.UpdateOrderStatusResult, error) {
	if s.orderActions == nil {
		return nil, opscontract.ErrNotConfigured
	}
	orderNo = strings.TrimSpace(orderNo)
	status := strings.TrimSpace(input.Status)
	if orderNo == "" || status == "" {
		return nil, opscontract.ErrInvalidInput
	}
	input.Status = status
	return s.orderActions.UpdateOrderStatus(orderNo, input)
}

func (s *Service) RetryProcurement(id uint) (*opscontract.RetryProcurementResult, error) {
	if s.orderActions == nil {
		return nil, opscontract.ErrNotConfigured
	}
	if id == 0 {
		return nil, opscontract.ErrInvalidInput
	}
	return s.orderActions.RetryProcurement(id)
}

func (s *Service) Overview() (*opscontract.Overview, error) {
	caps := []string{"orders.read", "payments.read", "refunds.read", "procurement.read", "checkout_reservations.read"}
	if s.categories != nil {
		caps = append(caps, "categories.read", "categories.write")
	}
	if s.categoryStore != nil {
		caps = append(caps, "categories.delete")
	}
	if s.products != nil && s.skus != nil {
		caps = append(caps, "products.read", "products.write", "skus.write", "skus.delete")
	}
	if s.productDelete != nil {
		caps = append(caps, "products.delete")
	}
	if s.coupons != nil {
		caps = append(caps, "coupons.read", "coupons.write")
	}
	if s.giftCards != nil {
		caps = append(caps, "gift_cards.read", "gift_cards.write")
	}
	if s.acquiring != nil {
		caps = append(caps, "acquiring.read", "acquiring.write", "mail_health.read")
	}
	if s.refunds != nil {
		caps = append(caps, "refunds.write")
	}
	if s.orderActions != nil {
		caps = append(caps, "orders.filter", "orders.manual_confirm", "orders.manual_fulfill", "orders.status_write", "procurement.retry")
	}
	result := &opscontract.Overview{Source: "storefront", CollectedAt: time.Now().UTC(), Freshness: "fresh", OrdersByStatus: map[string]int64{}, Capabilities: caps}
	_, total, err := s.orders.ListAdmin(ordercontract.ListFilter{Page: 1, PageSize: 1})
	if err != nil {
		return nil, err
	}
	result.TotalOrders = total
	for _, status := range []string{constants.OrderStatusPendingPayment, constants.OrderStatusPaid, constants.OrderStatusFulfilling, constants.OrderStatusPartiallyDelivered, constants.OrderStatusDelivered, constants.OrderStatusCompleted, constants.OrderStatusCanceled, constants.OrderStatusRefunded, constants.OrderStatusPartiallyRefunded} {
		_, count, err := s.orders.ListAdmin(ordercontract.ListFilter{Status: status, Page: 1, PageSize: 1})
		if err != nil {
			return nil, err
		}
		result.OrdersByStatus[status] = count
	}
	return result, nil
}

func stringMap(raw jsonmap.JSON) map[string]string {
	out := map[string]string{}
	for k, v := range raw {
		if str, ok := v.(string); ok {
			out[k] = str
		}
	}
	return out
}

func mapCategory(c categorydomain.Category) opscontract.Category {
	return opscontract.Category{
		ID:        c.ID,
		ParentID:  c.ParentID,
		Slug:      c.Slug,
		Name:      stringMap(c.NameJSON),
		Icon:      c.Icon,
		SortOrder: c.SortOrder,
		IsActive:  c.IsActive,
		CreatedAt: c.CreatedAt,
	}
}

func mapSKU(sku productdomain.ProductSKU) opscontract.ProductSKU {
	return opscontract.ProductSKU{
		ID:                sku.ID,
		ProductID:         sku.ProductID,
		SKUCode:           sku.SKUCode,
		PriceAmount:       sku.PriceAmount.String(),
		CostPriceAmount:   sku.CostPriceAmount.String(),
		ManualStockTotal:  sku.ManualStockTotal,
		ManualStockLocked: sku.ManualStockLocked,
		ManualStockSold:   sku.ManualStockSold,
		IsActive:          sku.IsActive,
		SortOrder:         sku.SortOrder,
		UpdatedAt:         sku.UpdatedAt,
	}
}

func mapProduct(p productdomain.Product, skus []productdomain.ProductSKU) opscontract.Product {
	mappedSKUs := make([]opscontract.ProductSKU, 0, len(skus))
	for _, sku := range skus {
		mappedSKUs = append(mappedSKUs, mapSKU(sku))
	}
	return opscontract.Product{
		ID:                p.ID,
		CategoryID:        p.CategoryID,
		Slug:              p.Slug,
		Title:             stringMap(p.TitleJSON),
		Description:       stringMap(p.DescriptionJSON),
		PriceAmount:       p.PriceAmount.String(),
		CostPriceAmount:   p.CostPriceAmount.String(),
		PurchaseType:      p.PurchaseType,
		FulfillmentType:   p.FulfillmentType,
		ManualStockTotal:  p.ManualStockTotal,
		ManualStockLocked: p.ManualStockLocked,
		ManualStockSold:   p.ManualStockSold,
		IsMapped:          p.IsMapped,
		IsActive:          p.IsActive,
		SortOrder:         p.SortOrder,
		SKUs:              mappedSKUs,
		UpdatedAt:         p.UpdatedAt,
	}
}

func mapCoupon(c coupondomain.Coupon) opscontract.Coupon {
	return opscontract.Coupon{
		ID:           c.ID,
		Code:         c.Code,
		Type:         c.Type,
		Value:        c.Value.String(),
		MinAmount:    c.MinAmount.String(),
		MaxDiscount:  c.MaxDiscount.String(),
		UsageLimit:   c.UsageLimit,
		UsedCount:    c.UsedCount,
		PerUserLimit: c.PerUserLimit,
		ScopeType:    c.ScopeType,
		ScopeRefIDs:  c.ScopeRefIDs,
		StartsAt:     c.StartsAt,
		EndsAt:       c.EndsAt,
		IsActive:     c.IsActive,
		CreatedAt:    c.CreatedAt,
		UpdatedAt:    c.UpdatedAt,
	}
}

func (s *Service) Categories() (*opscontract.CategoriesResult, error) {
	if s.categories == nil {
		return nil, opscontract.ErrNotConfigured
	}
	rows, err := s.categories.List()
	if err != nil {
		return nil, err
	}
	items := make([]opscontract.Category, 0, len(rows))
	for _, row := range rows {
		items = append(items, mapCategory(row))
	}
	return &opscontract.CategoriesResult{
		Source:      "storefront",
		CollectedAt: time.Now().UTC(),
		Freshness:   "fresh",
		Categories:  items,
	}, nil
}

func normalizeOpsSlug(raw string) (string, error) {
	slug := strings.ToLower(strings.TrimSpace(raw))
	if slug == "" || len(slug) > 120 {
		return "", opscontract.ErrInvalidInput
	}
	for _, r := range slug {
		if !((r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_') {
			return "", opscontract.ErrInvalidInput
		}
	}
	return slug, nil
}

func normalizeLocalizedStringMap(raw map[string]string) map[string]any {
	out := make(map[string]any, len(raw))
	for k, v := range raw {
		key := strings.TrimSpace(k)
		val := strings.TrimSpace(v)
		if key != "" && val != "" {
			out[key] = val
		}
	}
	return out
}

func (s *Service) CreateCategory(input opscontract.CreateCategoryInput) (*opscontract.Category, error) {
	if s.categoryStore == nil {
		return nil, opscontract.ErrNotConfigured
	}
	slug, err := normalizeOpsSlug(input.Slug)
	if err != nil {
		return nil, err
	}
	nameMap := normalizeLocalizedStringMap(input.Name)
	if len(nameMap) == 0 {
		return nil, opscontract.ErrInvalidInput
	}
	icon := strings.TrimSpace(input.Icon)
	if utf8.RuneCountInString(icon) > 255 {
		return nil, opscontract.ErrInvalidInput
	}
	isActive := true
	if input.IsActive != nil {
		isActive = *input.IsActive
	}
	return s.categoryStore.Create(input.ParentID, slug, nameMap, icon, input.SortOrder, isActive)
}

func (s *Service) UpdateCategory(id string, input opscontract.UpdateCategoryInput) (*opscontract.Category, error) {
	if s.categories == nil {
		return nil, opscontract.ErrNotConfigured
	}
	id = strings.TrimSpace(id)
	hasCoreFields := input.ParentID != nil || input.Slug != nil || input.Name != nil || input.Icon != nil
	if id == "" || (!hasCoreFields && input.SortOrder == nil && input.IsActive == nil) {
		return nil, opscontract.ErrInvalidInput
	}
	cat, err := s.categories.GetByID(id)
	if err != nil {
		return nil, err
	}
	if cat == nil {
		return nil, opscontract.ErrNotFound
	}
	if s.categoryStore == nil {
		if hasCoreFields {
			return nil, opscontract.ErrNotConfigured
		}
		if input.SortOrder != nil {
			cat.SortOrder = *input.SortOrder
		}
		if input.IsActive != nil {
			cat.IsActive = *input.IsActive
		}
		if err := s.categories.Update(cat); err != nil {
			return nil, err
		}
		out := mapCategory(*cat)
		return &out, nil
	}

	parentID := cat.ParentID
	if input.ParentID != nil {
		parentID = *input.ParentID
	}
	slug := cat.Slug
	if input.Slug != nil {
		normalizedSlug, slugErr := normalizeOpsSlug(*input.Slug)
		if slugErr != nil {
			return nil, slugErr
		}
		slug = normalizedSlug
	}
	nameMap := map[string]any{}
	for k, v := range cat.NameJSON {
		nameMap[k] = v
	}
	if input.Name != nil {
		normalizedName := normalizeLocalizedStringMap(input.Name)
		if len(normalizedName) == 0 {
			return nil, opscontract.ErrInvalidInput
		}
		nameMap = normalizedName
	}
	icon := cat.Icon
	if input.Icon != nil {
		trimmedIcon := strings.TrimSpace(*input.Icon)
		if utf8.RuneCountInString(trimmedIcon) > 255 {
			return nil, opscontract.ErrInvalidInput
		}
		icon = trimmedIcon
	}
	sortOrder := cat.SortOrder
	if input.SortOrder != nil {
		sortOrder = *input.SortOrder
	}
	return s.categoryStore.Update(id, parentID, slug, nameMap, icon, sortOrder, input.IsActive)
}

func (s *Service) DeleteCategory(id string) error {
	if s.categoryStore == nil {
		return opscontract.ErrNotConfigured
	}
	id = strings.TrimSpace(id)
	parsed, err := strconv.ParseUint(id, 10, 64)
	if err != nil || parsed == 0 {
		return opscontract.ErrInvalidInput
	}
	return s.categoryStore.Delete(id)
}

func (s *Service) Products(page, pageSize int, categoryID, search string) (*opscontract.ProductsResult, error) {
	if s.products == nil {
		return nil, opscontract.ErrNotConfigured
	}
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 50
	}
	if pageSize > 100 {
		pageSize = 100
	}
	rows, total, err := s.products.List(productcontract.ListFilter{
		Page:       page,
		PageSize:   pageSize,
		CategoryID: strings.TrimSpace(categoryID),
		Search:     strings.TrimSpace(search),
	})
	if err != nil {
		return nil, err
	}
	items := make([]opscontract.Product, 0, len(rows))
	for _, row := range rows {
		skus := row.SKUs
		if len(skus) == 0 && s.skus != nil {
			if listed, err := s.skus.ListByProduct(row.ID, false); err == nil {
				skus = listed
			}
		}
		items = append(items, mapProduct(row, skus))
	}
	return &opscontract.ProductsResult{
		Source:      "storefront",
		CollectedAt: time.Now().UTC(),
		Freshness:   "fresh",
		Products:    items,
		Total:       total,
		Page:        page,
		PageSize:    pageSize,
	}, nil
}

func (s *Service) CreateProduct(input opscontract.CreateProductInput) (*opscontract.Product, error) {
	if s.products == nil || s.categories == nil {
		return nil, opscontract.ErrNotConfigured
	}
	if input.CategoryID == 0 {
		return nil, opscontract.ErrInvalidInput
	}
	if err := productdomain.ValidateCategoryAssignment(s.categories, input.CategoryID, 0, opscontract.ErrInvalidInput); err != nil {
		return nil, err
	}
	slug := strings.ToLower(strings.TrimSpace(input.Slug))
	if slug == "" || len(slug) > 120 {
		return nil, opscontract.ErrInvalidInput
	}
	for _, r := range slug {
		if !((r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_') {
			return nil, opscontract.ErrInvalidInput
		}
	}
	count, err := s.products.CountBySlug(slug, nil)
	if err != nil {
		return nil, err
	}
	if count > 0 {
		return nil, opscontract.ErrConflict
	}
	title := jsonmap.JSON{}
	for k, v := range input.Title {
		if trimmed := strings.TrimSpace(v); trimmed != "" {
			title[strings.TrimSpace(k)] = trimmed
		}
	}
	if len(title) == 0 {
		return nil, opscontract.ErrInvalidInput
	}
	desc := jsonmap.JSON{}
	for k, v := range input.Description {
		if trimmed := strings.TrimSpace(v); trimmed != "" {
			desc[strings.TrimSpace(k)] = trimmed
		}
	}
	priceDec, err := decimal.NewFromString(strings.TrimSpace(input.PriceAmount))
	if err != nil || !priceDec.IsPositive() {
		return nil, opscontract.ErrInvalidInput
	}
	costDec := decimal.Zero
	if rawCost := strings.TrimSpace(input.CostPriceAmount); rawCost != "" {
		costDec, err = decimal.NewFromString(rawCost)
		if err != nil || costDec.IsNegative() {
			return nil, opscontract.ErrInvalidInput
		}
	}
	purchaseType := strings.TrimSpace(input.PurchaseType)
	if purchaseType == "" {
		purchaseType = constants.ProductPurchaseMember
	}
	if purchaseType != constants.ProductPurchaseMember && purchaseType != constants.ProductPurchaseGuest {
		return nil, opscontract.ErrInvalidInput
	}
	fulfillmentType := strings.TrimSpace(input.FulfillmentType)
	if fulfillmentType == "" {
		fulfillmentType = constants.FulfillmentTypeManual
	}
	if fulfillmentType != constants.FulfillmentTypeManual && fulfillmentType != constants.FulfillmentTypeAuto {
		return nil, opscontract.ErrInvalidInput
	}
	manualStock := 0
	if input.ManualStockTotal != nil {
		manualStock = *input.ManualStockTotal
	}
	if manualStock < constants.ManualStockUnlimited {
		return nil, opscontract.ErrInvalidInput
	}
	isActive := true
	if input.IsActive != nil {
		isActive = *input.IsActive
	}
	now := time.Now().UTC()
	prod := &productdomain.Product{
		CategoryID:       input.CategoryID,
		Slug:             slug,
		TitleJSON:        title,
		DescriptionJSON:  desc,
		PriceAmount:      money.FromDecimal(priceDec.Round(2)),
		CostPriceAmount:  money.FromDecimal(costDec.Round(2)),
		PurchaseType:     purchaseType,
		StockDisplayMode: constants.ProductStockDisplayExact,
		FulfillmentType:  fulfillmentType,
		ManualStockTotal: manualStock,
		IsActive:         isActive,
		SortOrder:        input.SortOrder,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	if err := s.products.Create(prod); err != nil {
		return nil, err
	}
	var skus []productdomain.ProductSKU
	if s.skus != nil {
		skuCode := strings.ToUpper(strings.TrimSpace(input.SKUCode))
		if skuCode == "" {
			skuCode = strings.ToUpper(slug)
		}
		if len(skuCode) > 64 {
			skuCode = skuCode[:64]
		}
		sku := &productdomain.ProductSKU{
			ProductID:        prod.ID,
			SKUCode:          skuCode,
			PriceAmount:      prod.PriceAmount,
			CostPriceAmount:  prod.CostPriceAmount,
			ManualStockTotal: manualStock,
			IsActive:         true,
			SortOrder:        0,
			CreatedAt:        now,
			UpdatedAt:        now,
		}
		if err := s.skus.Create(sku); err != nil {
			return nil, err
		}
		skus = append(skus, *sku)
	}
	out := mapProduct(*prod, skus)
	return &out, nil
}

func (s *Service) UpdateProduct(id string, input opscontract.UpdateProductInput) (*opscontract.Product, error) {
	if s.products == nil {
		return nil, opscontract.ErrNotConfigured
	}
	id = strings.TrimSpace(id)
	hasFullFields := input.CategoryID != nil || input.Slug != nil || len(input.Title) > 0 || input.Description != nil ||
		input.PriceAmount != nil || input.CostPriceAmount != nil || input.PurchaseType != nil ||
		input.FulfillmentType != nil || input.ManualStockTotal != nil
	if id == "" || (!hasFullFields && input.IsActive == nil && input.SortOrder == nil) {
		return nil, opscontract.ErrInvalidInput
	}
	prod, err := s.products.GetAdminByID(id)
	if err != nil {
		return nil, err
	}
	if prod == nil {
		return nil, opscontract.ErrNotFound
	}
	if !hasFullFields {
		fields := map[string]interface{}{}
		if input.IsActive != nil {
			fields["is_active"] = *input.IsActive
			prod.IsActive = *input.IsActive
		}
		if input.SortOrder != nil {
			fields["sort_order"] = *input.SortOrder
			prod.SortOrder = *input.SortOrder
		}
		if err := s.products.QuickUpdate(id, fields); err != nil {
			return nil, err
		}
	} else {
		if input.CategoryID != nil {
			if *input.CategoryID == 0 || s.categories == nil {
				return nil, opscontract.ErrInvalidInput
			}
			if catErr := productdomain.ValidateCategoryAssignment(s.categories, *input.CategoryID, prod.CategoryID, opscontract.ErrInvalidInput); catErr != nil {
				return nil, catErr
			}
			prod.CategoryID = *input.CategoryID
		}
		if input.Slug != nil {
			slug := strings.ToLower(strings.TrimSpace(*input.Slug))
			if slug == "" || len(slug) > 120 {
				return nil, opscontract.ErrInvalidInput
			}
			for _, r := range slug {
				if !((r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_') {
					return nil, opscontract.ErrInvalidInput
				}
			}
			if slug != prod.Slug {
				excludeID := strconv.FormatUint(uint64(prod.ID), 10)
				count, cntErr := s.products.CountBySlug(slug, &excludeID)
				if cntErr != nil {
					return nil, cntErr
				}
				if count > 0 {
					return nil, opscontract.ErrConflict
				}
				prod.Slug = slug
			}
		}
		if len(input.Title) > 0 {
			title := jsonmap.JSON{}
			for k, v := range input.Title {
				if trimmed := strings.TrimSpace(v); trimmed != "" {
					title[strings.TrimSpace(k)] = trimmed
				}
			}
			if len(title) == 0 {
				return nil, opscontract.ErrInvalidInput
			}
			prod.TitleJSON = title
		}
		if input.Description != nil {
			desc := jsonmap.JSON{}
			for k, v := range input.Description {
				if trimmed := strings.TrimSpace(v); trimmed != "" {
					desc[strings.TrimSpace(k)] = trimmed
				}
			}
			prod.DescriptionJSON = desc
		}
		if input.PriceAmount != nil {
			dec, pErr := decimal.NewFromString(strings.TrimSpace(*input.PriceAmount))
			if pErr != nil || !dec.IsPositive() {
				return nil, opscontract.ErrInvalidInput
			}
			prod.PriceAmount = money.FromDecimal(dec.Round(2))
		}
		if input.CostPriceAmount != nil {
			rawCost := strings.TrimSpace(*input.CostPriceAmount)
			if rawCost == "" {
				prod.CostPriceAmount = money.FromDecimal(decimal.Zero)
			} else {
				dec, cErr := decimal.NewFromString(rawCost)
				if cErr != nil || dec.IsNegative() {
					return nil, opscontract.ErrInvalidInput
				}
				prod.CostPriceAmount = money.FromDecimal(dec.Round(2))
			}
		}
		if input.PurchaseType != nil {
			pt := strings.TrimSpace(*input.PurchaseType)
			if pt != constants.ProductPurchaseMember && pt != constants.ProductPurchaseGuest {
				return nil, opscontract.ErrInvalidInput
			}
			prod.PurchaseType = pt
		}
		if input.FulfillmentType != nil {
			ft := strings.TrimSpace(*input.FulfillmentType)
			if ft != constants.FulfillmentTypeManual && ft != constants.FulfillmentTypeAuto {
				return nil, opscontract.ErrInvalidInput
			}
			prod.FulfillmentType = ft
		}
		if input.ManualStockTotal != nil {
			if *input.ManualStockTotal < constants.ManualStockUnlimited {
				return nil, opscontract.ErrInvalidInput
			}
			prod.ManualStockTotal = *input.ManualStockTotal
		}
		if input.IsActive != nil {
			prod.IsActive = *input.IsActive
		}
		if input.SortOrder != nil {
			prod.SortOrder = *input.SortOrder
		}
		prod.UpdatedAt = time.Now().UTC()
		if err := s.products.Update(prod); err != nil {
			return nil, err
		}
	}
	skus := prod.SKUs
	if len(skus) == 0 && s.skus != nil {
		if listed, err := s.skus.ListByProduct(prod.ID, false); err == nil {
			skus = listed
		}
	}
	out := mapProduct(*prod, skus)
	return &out, nil
}

func (s *Service) DeleteProduct(id string) error {
	if s.productDelete == nil {
		return opscontract.ErrNotConfigured
	}
	id = strings.TrimSpace(id)
	parsed, err := strconv.ParseUint(id, 10, 64)
	if err != nil || parsed == 0 {
		return opscontract.ErrInvalidInput
	}
	return s.productDelete.Delete(id)
}

func batchDeleteErrorCode(err error) string {
	var conflictErr *opscontract.ConflictError
	switch {
	case errors.As(err, &conflictErr):
		return conflictErr.Code
	case errors.Is(err, opscontract.ErrNotFound):
		return "not_found"
	case errors.Is(err, opscontract.ErrInvalidInput):
		return "invalid_input"
	case errors.Is(err, opscontract.ErrConflict):
		return "conflict"
	case errors.Is(err, opscontract.ErrNotConfigured):
		return "not_configured"
	default:
		return "delete_failed"
	}
}

func (s *Service) BatchDeleteProducts(input opscontract.BatchDeleteProductsInput) (*opscontract.BatchDeleteResult, error) {
	if s.productDelete == nil {
		return nil, opscontract.ErrNotConfigured
	}
	if len(input.IDs) == 0 || len(input.IDs) > 100 {
		return nil, opscontract.ErrInvalidInput
	}
	res := &opscontract.BatchDeleteResult{
		Total:  len(input.IDs),
		Errors: make([]opscontract.BatchDeleteError, 0),
	}
	for _, id := range input.IDs {
		if id == 0 {
			res.Failed++
			res.Errors = append(res.Errors, opscontract.BatchDeleteError{ID: id, Error: "invalid_id"})
			continue
		}
		if err := s.DeleteProduct(strconv.FormatUint(uint64(id), 10)); err != nil {
			res.Failed++
			res.Errors = append(res.Errors, opscontract.BatchDeleteError{ID: id, Error: batchDeleteErrorCode(err)})
			continue
		}
		res.Succeeded++
	}
	return res, nil
}

func minActiveSKUPrices(skus []productdomain.ProductSKU) (minPrice money.Amount, minCost money.Amount, hasActive bool) {
	for _, item := range skus {
		if !item.IsActive {
			continue
		}
		if !hasActive {
			minPrice = item.PriceAmount
			minCost = item.CostPriceAmount
			hasActive = true
			continue
		}
		if item.PriceAmount.Decimal.LessThan(minPrice.Decimal) {
			minPrice = item.PriceAmount
		}
		if item.CostPriceAmount.Decimal.LessThan(minCost.Decimal) {
			minCost = item.CostPriceAmount
		}
	}
	return minPrice, minCost, hasActive
}

func (s *Service) syncProductPricesFromActiveSKUs(productID uint) {
	if s.products == nil || s.skus == nil || productID == 0 {
		return
	}
	allSKUs, err := s.skus.ListByProduct(productID, false)
	if err != nil {
		return
	}
	minPrice, minCost, hasActive := minActiveSKUPrices(allSKUs)
	if !hasActive {
		return
	}
	_ = s.products.QuickUpdate(strconv.FormatUint(uint64(productID), 10), map[string]interface{}{
		"price_amount":      minPrice,
		"cost_price_amount": minCost,
	})
}

func (s *Service) CreateSKU(productID uint, input opscontract.CreateSKUInput) (*opscontract.ProductSKU, error) {
	if s.products == nil || s.skus == nil {
		return nil, opscontract.ErrNotConfigured
	}
	if productID == 0 {
		return nil, opscontract.ErrInvalidInput
	}
	prod, err := s.products.GetAdminByID(strconv.FormatUint(uint64(productID), 10))
	if err != nil {
		return nil, err
	}
	if prod == nil {
		return nil, opscontract.ErrNotFound
	}
	code := strings.ToUpper(strings.TrimSpace(input.SKUCode))
	if code == "" || len(code) > 64 {
		return nil, opscontract.ErrInvalidInput
	}
	if existing, err := s.skus.GetByProductAndCode(productID, code); err != nil {
		return nil, err
	} else if existing != nil {
		return nil, opscontract.ErrConflict
	}
	priceDec, err := decimal.NewFromString(strings.TrimSpace(input.PriceAmount))
	if err != nil || !priceDec.IsPositive() {
		return nil, opscontract.ErrInvalidInput
	}
	costDec := decimal.Zero
	if rawCost := strings.TrimSpace(input.CostPriceAmount); rawCost != "" {
		costDec, err = decimal.NewFromString(rawCost)
		if err != nil || costDec.IsNegative() {
			return nil, opscontract.ErrInvalidInput
		}
	}
	manualStock := 0
	if input.ManualStockTotal != nil {
		manualStock = *input.ManualStockTotal
	}
	if manualStock < constants.ManualStockUnlimited {
		return nil, opscontract.ErrInvalidInput
	}
	active := true
	if input.IsActive != nil {
		active = *input.IsActive
	}
	now := time.Now().UTC()
	sku := &productdomain.ProductSKU{
		ProductID:        productID,
		SKUCode:          code,
		PriceAmount:      money.FromDecimal(priceDec.Round(2)),
		CostPriceAmount:  money.FromDecimal(costDec.Round(2)),
		ManualStockTotal: manualStock,
		IsActive:         active,
		SortOrder:        input.SortOrder,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	if err := s.skus.Create(sku); err != nil {
		return nil, err
	}
	if sku.IsActive {
		s.syncProductPricesFromActiveSKUs(productID)
	}
	out := mapSKU(*sku)
	return &out, nil
}

func (s *Service) UpdateSKU(id uint, input opscontract.UpdateSKUInput) (*opscontract.ProductSKU, error) {
	if s.skus == nil {
		return nil, opscontract.ErrNotConfigured
	}
	if id == 0 || (input.SKUCode == nil && input.PriceAmount == nil && input.CostPriceAmount == nil && input.ManualStockTotal == nil && input.SortOrder == nil && input.IsActive == nil) {
		return nil, opscontract.ErrInvalidInput
	}
	sku, err := s.skus.GetByID(id)
	if err != nil {
		return nil, err
	}
	if sku == nil {
		return nil, opscontract.ErrNotFound
	}
	if input.SKUCode != nil {
		code := strings.ToUpper(strings.TrimSpace(*input.SKUCode))
		if code == "" || len(code) > 64 {
			return nil, opscontract.ErrInvalidInput
		}
		if code != sku.SKUCode {
			if existing, err := s.skus.GetByProductAndCode(sku.ProductID, code); err != nil {
				return nil, err
			} else if existing != nil && existing.ID != sku.ID {
				return nil, opscontract.ErrConflict
			}
			sku.SKUCode = code
		}
	}
	if input.PriceAmount != nil {
		dec, err := decimal.NewFromString(strings.TrimSpace(*input.PriceAmount))
		if err != nil || dec.IsNegative() {
			return nil, opscontract.ErrInvalidInput
		}
		sku.PriceAmount = money.FromDecimal(dec.Round(2))
	}
	if input.CostPriceAmount != nil {
		rawCost := strings.TrimSpace(*input.CostPriceAmount)
		if rawCost == "" {
			sku.CostPriceAmount = money.FromDecimal(decimal.Zero)
		} else {
			dec, err := decimal.NewFromString(rawCost)
			if err != nil || dec.IsNegative() {
				return nil, opscontract.ErrInvalidInput
			}
			sku.CostPriceAmount = money.FromDecimal(dec.Round(2))
		}
	}
	if input.ManualStockTotal != nil {
		if *input.ManualStockTotal < -1 {
			return nil, opscontract.ErrInvalidInput
		}
		sku.ManualStockTotal = *input.ManualStockTotal
	}
	if input.SortOrder != nil {
		sku.SortOrder = *input.SortOrder
	}
	if input.IsActive != nil {
		sku.IsActive = *input.IsActive
	}
	sku.UpdatedAt = time.Now().UTC()
	if err := s.skus.Update(sku); err != nil {
		return nil, err
	}
	// Recalculate product display price from the lowest active SKU price.
	if (input.PriceAmount != nil || input.CostPriceAmount != nil || input.IsActive != nil) && s.products != nil {
		s.syncProductPricesFromActiveSKUs(sku.ProductID)
	}
	out := mapSKU(*sku)
	return &out, nil
}

func (s *Service) DeleteSKU(id uint) error {
	if s.skus == nil || s.products == nil {
		return opscontract.ErrNotConfigured
	}
	if id == 0 {
		return opscontract.ErrInvalidInput
	}

	// Pre-transaction read-only validations (SQLite MaxOpenConns=1 safe).
	sku, err := s.skus.GetByID(id)
	if err != nil {
		return err
	}
	if sku == nil {
		return opscontract.ErrNotFound
	}
	prod, err := s.products.GetAdminByID(strconv.FormatUint(uint64(sku.ProductID), 10))
	if err != nil {
		return err
	}
	if prod == nil {
		return opscontract.ErrNotFound
	}
	allSKUs, err := s.skus.ListByProduct(sku.ProductID, false)
	if err != nil {
		return err
	}
	remaining := make([]productdomain.ProductSKU, 0, len(allSKUs))
	for _, item := range allSKUs {
		if item.ID == sku.ID {
			continue
		}
		remaining = append(remaining, item)
	}
	minPrice, minCost, hasActive := minActiveSKUPrices(remaining)
	if !hasActive {
		return opscontract.ErrProductRequiresActiveSKU
	}
	if strings.TrimSpace(prod.FulfillmentType) == constants.FulfillmentTypeAuto {
		if s.cardSecretStock == nil {
			return opscontract.ErrNotConfigured
		}
		total, available, used, countErr := s.cardSecretStock.CountByProduct(prod.ID, sku.ID)
		if countErr != nil {
			return countErr
		}
		if available > 0 || total-used > 0 {
			return opscontract.ErrSKUHasCardSecretStock
		}
	}

	prod.WholesalePrices = productdomain.PruneWholesalePricesForSKUs(prod.WholesalePrices, remaining)
	prod.PriceAmount = minPrice
	prod.CostPriceAmount = minCost
	prod.UpdatedAt = time.Now().UTC()

	apply := func(txProducts productcontract.Repository, txSKUs productcontract.SKURepository) error {
		if err := txSKUs.Delete(sku.ID); err != nil {
			return err
		}
		return txProducts.Update(prod)
	}
	if s.skuDeleteUoW != nil {
		return s.skuDeleteUoW.WithinTransaction(apply)
	}
	return apply(s.products, s.skus)
}

func (s *Service) Pricing() (*opscontract.PricingSettings, error) {
	if s.pricing == nil {
		return nil, opscontract.ErrNotConfigured
	}
	siteCurrency, displayCurrency, usdCnyRate, rateUpdatedAt, err := s.pricing.GetSitePricing()
	if err != nil {
		return nil, err
	}
	conns, err := s.pricing.ListConnectionPricing()
	if err != nil {
		return nil, err
	}
	if conns == nil {
		conns = []opscontract.ConnectionPricing{}
	}
	return &opscontract.PricingSettings{
		Source:                 "storefront",
		CollectedAt:            time.Now().UTC(),
		Freshness:              "fresh",
		SiteCurrency:           siteCurrency,
		DefaultDisplayCurrency: displayCurrency,
		USDCNYRate:             usdCnyRate,
		RateUpdatedAt:          rateUpdatedAt,
		Connections:            conns,
	}, nil
}

func (s *Service) UpdatePricing(input opscontract.UpdatePricingInput) (*opscontract.PricingSettings, error) {
	if s.pricing == nil {
		return nil, opscontract.ErrNotConfigured
	}
	hasSite := input.SiteCurrency != nil || input.DefaultDisplayCurrency != nil || input.USDCNYRate != nil
	hasConn := input.ConnectionID != nil && *input.ConnectionID > 0 && (input.ExchangeRate != nil || input.PriceMarkupPercent != nil || input.PriceRoundingMode != nil || input.AutoSyncPrice != nil)
	if !hasSite && !hasConn {
		return nil, opscontract.ErrInvalidInput
	}
	if hasSite {
		if err := s.pricing.UpdateSitePricing(input.SiteCurrency, input.DefaultDisplayCurrency, input.USDCNYRate); err != nil {
			return nil, err
		}
	}
	if hasConn {
		if input.ExchangeRate != nil && (*input.ExchangeRate <= 0 || *input.ExchangeRate > 1000) {
			return nil, opscontract.ErrInvalidInput
		}
		if err := s.pricing.UpdateConnectionPricing(*input.ConnectionID, input.ExchangeRate, input.PriceMarkupPercent, input.PriceRoundingMode, input.AutoSyncPrice); err != nil {
			return nil, err
		}
	}
	return s.Pricing()
}

func (s *Service) Coupons(page, pageSize int, code string, isActive *bool) (*opscontract.CouponsResult, error) {
	if s.coupons == nil {
		return nil, opscontract.ErrNotConfigured
	}
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 50
	}
	if pageSize > 100 {
		pageSize = 100
	}
	rows, total, err := s.coupons.List(couponcontract.ListFilter{
		Code:     strings.TrimSpace(code),
		IsActive: isActive,
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		return nil, err
	}
	items := make([]opscontract.Coupon, 0, len(rows))
	for _, row := range rows {
		items = append(items, mapCoupon(row))
	}
	return &opscontract.CouponsResult{
		Source:      "storefront",
		CollectedAt: time.Now().UTC(),
		Freshness:   "fresh",
		Coupons:     items,
		Total:       total,
		Page:        page,
		PageSize:    pageSize,
	}, nil
}

func (s *Service) CreateCoupon(input opscontract.CreateCouponInput) (*opscontract.Coupon, error) {
	if s.coupons == nil {
		return nil, opscontract.ErrNotConfigured
	}
	code := strings.ToUpper(strings.TrimSpace(input.Code))
	if code == "" || len(code) > 64 {
		return nil, opscontract.ErrInvalidInput
	}
	if input.Type != constants.CouponTypeFixed && input.Type != constants.CouponTypePercent {
		return nil, opscontract.ErrInvalidInput
	}
	val, err := decimal.NewFromString(strings.TrimSpace(input.Value))
	if err != nil || !val.IsPositive() {
		return nil, opscontract.ErrInvalidInput
	}
	if input.Type == constants.CouponTypePercent && val.GreaterThan(decimal.NewFromInt(100)) {
		return nil, opscontract.ErrInvalidInput
	}
	minAmt := decimal.Zero
	if strings.TrimSpace(input.MinAmount) != "" {
		minAmt, err = decimal.NewFromString(strings.TrimSpace(input.MinAmount))
		if err != nil || minAmt.IsNegative() {
			return nil, opscontract.ErrInvalidInput
		}
	}
	maxDisc := decimal.Zero
	if strings.TrimSpace(input.MaxDiscount) != "" {
		maxDisc, err = decimal.NewFromString(strings.TrimSpace(input.MaxDiscount))
		if err != nil || maxDisc.IsNegative() {
			return nil, opscontract.ErrInvalidInput
		}
	}
	if input.UsageLimit < 0 || input.PerUserLimit < 0 {
		return nil, opscontract.ErrInvalidInput
	}
	if existing, err := s.coupons.GetByCode(code); err != nil {
		return nil, err
	} else if existing != nil {
		return nil, opscontract.ErrConflict
	}
	scopeRefJSON := "[]"
	if len(input.ScopeRefIDs) > 0 {
		if raw, err := json.Marshal(input.ScopeRefIDs); err == nil {
			scopeRefJSON = string(raw)
		}
	}
	active := true
	if input.IsActive != nil {
		active = *input.IsActive
	}
	now := time.Now().UTC()
	rec := &coupondomain.Coupon{
		Code:         code,
		Type:         input.Type,
		Value:        money.FromDecimal(val.Round(2)),
		MinAmount:    money.FromDecimal(minAmt.Round(2)),
		MaxDiscount:  money.FromDecimal(maxDisc.Round(2)),
		UsageLimit:   input.UsageLimit,
		PerUserLimit: input.PerUserLimit,
		ScopeType:    constants.ScopeTypeProduct,
		ScopeRefIDs:  scopeRefJSON,
		StartsAt:     input.StartsAt,
		EndsAt:       input.EndsAt,
		IsActive:     active,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := s.coupons.Create(rec); err != nil {
		return nil, err
	}
	out := mapCoupon(*rec)
	return &out, nil
}

func (s *Service) GiftCards(page, pageSize int, code, status, batchNo string, redeemedUserID uint) (*opscontract.GiftCardsResult, error) {
	if s.giftCards == nil {
		return nil, opscontract.ErrNotConfigured
	}
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 50
	}
	if pageSize > 100 {
		pageSize = 100
	}
	normalizedStatus := strings.TrimSpace(strings.ToLower(status))
	if normalizedStatus != "" {
		switch normalizedStatus {
		case giftcarddomain.GiftCardStatusActive, giftcarddomain.GiftCardStatusRedeemed, giftcarddomain.GiftCardStatusDisabled, "expired":
		default:
			return nil, opscontract.ErrInvalidInput
		}
	}
	rows, total, err := s.giftCards.List(giftcardcontract.ListFilter{
		Code:           strings.TrimSpace(strings.ToUpper(code)),
		Status:         normalizedStatus,
		BatchNo:        strings.TrimSpace(strings.ToUpper(batchNo)),
		RedeemedUserID: redeemedUserID,
		Page:           page,
		PageSize:       pageSize,
	})
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	items := make([]opscontract.GiftCard, 0, len(rows))
	for _, row := range rows {
		items = append(items, mapGiftCard(row, "", now))
	}
	return &opscontract.GiftCardsResult{
		Source:      "storefront",
		CollectedAt: now,
		Freshness:   "fresh",
		GiftCards:   items,
		Cards:       items,
		Total:       total,
		Page:        page,
		PageSize:    pageSize,
	}, nil
}

func (s *Service) GenerateGiftCards(input opscontract.GenerateGiftCardsInput) (*opscontract.GenerateGiftCardsResult, error) {
	if s.giftCards == nil {
		return nil, opscontract.ErrNotConfigured
	}
	name := strings.TrimSpace(input.Name)
	if name == "" || utf8.RuneCountInString(name) > 120 {
		return nil, opscontract.ErrInvalidInput
	}
	if input.Quantity <= 0 || input.Quantity > 1000 {
		return nil, opscontract.ErrInvalidInput
	}
	rawAmt, err := decimal.NewFromString(strings.TrimSpace(input.Amount))
	if err != nil {
		return nil, opscontract.ErrInvalidInput
	}
	amount := rawAmt.Round(2)
	if !amount.Equal(rawAmt) || amount.LessThanOrEqual(decimal.Zero) {
		return nil, opscontract.ErrInvalidInput
	}

	now := time.Now().UTC()
	var expiresAt *time.Time
	if rawExp := strings.TrimSpace(input.ExpiresAt); rawExp != "" {
		parsed, parseErr := parseOptionalTime(rawExp)
		if parseErr != nil || !parsed.After(now) {
			return nil, opscontract.ErrInvalidInput
		}
		utc := parsed.UTC()
		expiresAt = &utc
	}

	// Read site currency BEFORE opening transaction to prevent SQLite MaxOpenConns=1 self-deadlock.
	currency := constants.SiteCurrencyDefault
	if s.pricing != nil {
		if siteCur, _, _, _, curErr := s.pricing.GetSitePricing(); curErr == nil && strings.TrimSpace(siteCur) != "" {
			currency = strings.ToUpper(strings.TrimSpace(siteCur))
		}
	}

	batchNo := strings.ToUpper(fmt.Sprintf("GCB%s%s", now.Format("20060102150405"), opsRandomHex(4)))
	batch := &giftcarddomain.GiftCardBatch{
		BatchNo:   batchNo,
		Name:      name,
		Amount:    money.FromDecimal(amount),
		Currency:  currency,
		Quantity:  input.Quantity,
		ExpiresAt: expiresAt,
		CreatedAt: now,
		UpdatedAt: now,
	}

	cards := make([]giftcarddomain.GiftCard, 0, input.Quantity)
	for i := range input.Quantity {
		cards = append(cards, giftcarddomain.GiftCard{
			Name:      name,
			Code:      strings.ToUpper(fmt.Sprintf("GC%s%04d%s", now.Format("060102150405"), i%10000, opsRandomHex(5))),
			Amount:    money.FromDecimal(amount),
			Currency:  currency,
			Status:    giftcarddomain.GiftCardStatusActive,
			ExpiresAt: expiresAt,
			CreatedAt: now,
			UpdatedAt: now,
		})
	}

	if err := s.giftCards.WithinTransaction(func(repo giftcardcontract.Repository) error {
		return repo.CreateBatch(batch, cards)
	}); err != nil {
		return nil, err
	}

	outCards := make([]opscontract.GiftCard, 0, len(cards))
	for _, card := range cards {
		outCards = append(outCards, mapGiftCard(card, batch.BatchNo, now))
	}

	return &opscontract.GenerateGiftCardsResult{
		Source:      "storefront",
		CollectedAt: now,
		Freshness:   "fresh",
		Batch: opscontract.GiftCardBatch{
			ID:        batch.ID,
			BatchNo:   batch.BatchNo,
			Name:      batch.Name,
			Amount:    batch.Amount.String(),
			Currency:  batch.Currency,
			Quantity:  batch.Quantity,
			ExpiresAt: batch.ExpiresAt,
			CreatedAt: batch.CreatedAt,
		},
		BatchNo: batch.BatchNo,
		Created: len(outCards),
		Cards:   outCards,
	}, nil
}

func mapGiftCard(card giftcarddomain.GiftCard, fallbackBatchNo string, now time.Time) opscontract.GiftCard {
	batchNo := fallbackBatchNo
	if card.Batch != nil && card.Batch.BatchNo != "" {
		batchNo = card.Batch.BatchNo
	}
	isExpired := card.Status == giftcarddomain.GiftCardStatusActive && card.ExpiresAt != nil && card.ExpiresAt.Before(now)
	return opscontract.GiftCard{
		ID:             card.ID,
		BatchID:        card.BatchID,
		BatchNo:        batchNo,
		Name:           card.Name,
		Code:           card.Code,
		Amount:         card.Amount.String(),
		Currency:       card.Currency,
		Status:         card.Status,
		IsExpired:      isExpired,
		ExpiresAt:      card.ExpiresAt,
		RedeemedAt:     card.RedeemedAt,
		RedeemedUserID: card.RedeemedUserID,
		WalletTxnID:    card.WalletTxnID,
		CreatedAt:      card.CreatedAt,
		UpdatedAt:      card.UpdatedAt,
	}
}

func parseOptionalTime(raw string) (time.Time, error) {
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339, "2006-01-02 15:04:05", "2006-01-02"} {
		if parsed, err := time.Parse(layout, raw); err == nil {
			if layout == "2006-01-02" {
				parsed = time.Date(parsed.Year(), parsed.Month(), parsed.Day(), 23, 59, 59, 0, time.UTC)
			}
			return parsed, nil
		}
	}
	return time.Time{}, opscontract.ErrInvalidInput
}

func opsRandomHex(n int) string {
	buf := make([]byte, n)
	if _, err := crand.Read(buf); err != nil {
		for i := range buf {
			buf[i] = byte('A' + (i % 26))
		}
	}
	return hex.EncodeToString(buf)
}

func (s *Service) UpdateCouponStatus(id uint, active bool) (*opscontract.Coupon, error) {
	if s.coupons == nil {
		return nil, opscontract.ErrNotConfigured
	}
	if id == 0 {
		return nil, opscontract.ErrInvalidInput
	}
	rec, err := s.coupons.GetByID(id)
	if err != nil {
		return nil, err
	}
	if rec == nil {
		return nil, opscontract.ErrNotFound
	}
	rec.IsActive = active
	rec.UpdatedAt = time.Now().UTC()
	if err := s.coupons.Update(rec); err != nil {
		return nil, err
	}
	out := mapCoupon(*rec)
	return &out, nil
}

func (s *Service) AcquiringChannels() (*opscontract.AcquiringChannelsResult, error) {
	if s.acquiring == nil {
		return nil, opscontract.ErrNotConfigured
	}
	channels, total, err := s.acquiring.ListChannels()
	if err != nil {
		return nil, err
	}
	return &opscontract.AcquiringChannelsResult{
		Source:      "storefront",
		CollectedAt: time.Now().UTC(),
		Freshness:   "fresh",
		Channels:    channels,
		Total:       total,
	}, nil
}

func (s *Service) UpsertAcquiringChannel(input opscontract.UpsertAcquiringChannelInput) (*opscontract.AcquiringChannel, error) {
	if s.acquiring == nil {
		return nil, opscontract.ErrNotConfigured
	}
	if strings.TrimSpace(input.Name) == "" || strings.TrimSpace(input.ProviderType) == "" || strings.TrimSpace(input.InteractionMode) == "" {
		return nil, opscontract.ErrInvalidInput
	}
	return s.acquiring.UpsertChannel(input)
}

func (s *Service) UpdateAcquiringChannelStatus(id uint, active bool) (*opscontract.AcquiringChannel, error) {
	if s.acquiring == nil {
		return nil, opscontract.ErrNotConfigured
	}
	if id == 0 {
		return nil, opscontract.ErrInvalidInput
	}
	return s.acquiring.UpdateChannelStatus(id, active)
}

func (s *Service) MailHealth() (*opscontract.MailHealthResult, error) {
	if s.acquiring == nil {
		return nil, opscontract.ErrNotConfigured
	}
	return s.acquiring.MailHealth()
}

func (s *Service) UpstreamConnections() (*opscontract.UpstreamConnectionsResult, error) {
	if s.upstream == nil {
		return nil, opscontract.ErrNotConfigured
	}
	conns, total, err := s.upstream.ListConnections()
	if err != nil {
		return nil, err
	}
	return &opscontract.UpstreamConnectionsResult{
		Source:      "storefront",
		CollectedAt: time.Now().UTC(),
		Freshness:   "fresh",
		Connections: conns,
		Total:       total,
	}, nil
}

func (s *Service) SaveUpstreamConnection(input opscontract.SaveUpstreamConnectionInput) (*opscontract.UpstreamConnection, error) {
	if s.upstream == nil {
		return nil, opscontract.ErrNotConfigured
	}
	if strings.TrimSpace(input.Name) == "" || strings.TrimSpace(input.BaseURL) == "" || strings.TrimSpace(input.ApiKey) == "" {
		return nil, opscontract.ErrInvalidInput
	}
	return s.upstream.SaveConnection(input)
}

func (s *Service) UpdateUpstreamConnectionStatus(id uint, status string) (*opscontract.UpstreamConnection, error) {
	if s.upstream == nil {
		return nil, opscontract.ErrNotConfigured
	}
	if id == 0 {
		return nil, opscontract.ErrInvalidInput
	}
	st := strings.ToLower(strings.TrimSpace(status))
	if st != constants.ConnectionStatusActive && st != constants.ConnectionStatusDisabled {
		return nil, opscontract.ErrInvalidInput
	}
	return s.upstream.UpdateConnectionStatus(id, st)
}

func (s *Service) DeleteUpstreamConnection(id uint) error {
	if s.upstream == nil {
		return opscontract.ErrNotConfigured
	}
	if id == 0 {
		return opscontract.ErrInvalidInput
	}
	return s.upstream.DeleteConnection(id)
}

func (s *Service) UpstreamCatalog(connectionID uint) (*opscontract.UpstreamCatalogResult, error) {
	if s.upstream == nil {
		return nil, opscontract.ErrNotConfigured
	}
	if connectionID == 0 {
		return nil, opscontract.ErrInvalidInput
	}
	return s.upstream.FetchCatalog(connectionID)
}

func (s *Service) ImportUpstreamProducts(connectionID uint, input opscontract.ImportUpstreamProductsInput) (*opscontract.ImportUpstreamProductsResult, error) {
	if s.upstream == nil {
		return nil, opscontract.ErrNotConfigured
	}
	if connectionID == 0 {
		return nil, opscontract.ErrInvalidInput
	}
	return s.upstream.ImportOrSync(connectionID, input)
}

func (s *Service) RefundOrder(orderNo string, input opscontract.RefundOrderInput) (*opscontract.RefundOrderResult, error) {
	if s.refunds == nil {
		return nil, opscontract.ErrNotConfigured
	}
	orderNo = strings.TrimSpace(orderNo)
	opID := strings.TrimSpace(input.OperationID)
	if orderNo == "" || opID == "" || len(opID) > 128 {
		return nil, opscontract.ErrInvalidInput
	}
	mode := strings.ToLower(strings.TrimSpace(input.Mode))
	if mode == "" {
		mode = "external_original"
	}
	if mode != "external_original" && mode != "wallet" {
		return nil, opscontract.ErrInvalidInput
	}
	amtDec, err := decimal.NewFromString(strings.TrimSpace(input.Amount))
	if err != nil || amtDec.Round(2).LessThanOrEqual(decimal.Zero) {
		return nil, opscontract.ErrInvalidInput
	}
	input.OperationID = opID
	input.Mode = mode
	input.Amount = amtDec.Round(2).StringFixed(2)
	return s.refunds.RefundOrder(orderNo, input)
}

func (s *Service) Users(page, pageSize int, keyword, status string) (*opscontract.UsersResult, error) {
	if s.users == nil {
		return nil, opscontract.ErrNotConfigured
	}
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 50
	}
	if pageSize > 200 {
		pageSize = 200
	}
	status = strings.ToLower(strings.TrimSpace(status))
	if status != "" && status != constants.UserStatusActive && status != constants.UserStatusDisabled {
		return nil, opscontract.ErrInvalidInput
	}
	return s.users.ListUsers(page, pageSize, strings.TrimSpace(keyword), status)
}

func (s *Service) CreateUser(input opscontract.CreateUserInput) (*opscontract.StorefrontUser, error) {
	if s.users == nil {
		return nil, opscontract.ErrNotConfigured
	}
	input.Email = strings.TrimSpace(input.Email)
	if input.Email == "" || len(input.Password) < 8 || len(input.Password) > 128 {
		return nil, opscontract.ErrInvalidInput
	}
	if st := strings.ToLower(strings.TrimSpace(input.Status)); st != "" {
		if st != constants.UserStatusActive && st != constants.UserStatusDisabled {
			return nil, opscontract.ErrInvalidInput
		}
		input.Status = st
	} else {
		input.Status = constants.UserStatusActive
	}
	return s.users.CreateUser(input)
}

func (s *Service) UpdateUser(id uint, input opscontract.UpdateUserInput) (*opscontract.StorefrontUser, error) {
	if s.users == nil {
		return nil, opscontract.ErrNotConfigured
	}
	if id == 0 {
		return nil, opscontract.ErrInvalidInput
	}
	if input.Password != nil && *input.Password != "" {
		if len(*input.Password) < 8 || len(*input.Password) > 128 {
			return nil, opscontract.ErrInvalidInput
		}
	}
	if input.Status != nil {
		st := strings.ToLower(strings.TrimSpace(*input.Status))
		if st != constants.UserStatusActive && st != constants.UserStatusDisabled {
			return nil, opscontract.ErrInvalidInput
		}
		input.Status = &st
	}
	return s.users.UpdateUser(id, input)
}

func (s *Service) DeleteUser(id uint) error {
	if s.users == nil {
		return opscontract.ErrNotConfigured
	}
	if id == 0 {
		return opscontract.ErrInvalidInput
	}
	return s.users.DeleteUser(id)
}

func (s *Service) AdjustUserWallet(id uint, input opscontract.AdjustUserWalletInput) (*opscontract.WalletAdjustmentResult, error) {
	if s.users == nil {
		return nil, opscontract.ErrNotConfigured
	}
	if id == 0 {
		return nil, opscontract.ErrInvalidInput
	}
	op := strings.ToLower(strings.TrimSpace(input.Operation))
	if op != "credit" && op != "debit" {
		return nil, opscontract.ErrInvalidInput
	}
	amtDec, err := decimal.NewFromString(strings.TrimSpace(input.Amount))
	if err != nil || amtDec.Round(2).LessThanOrEqual(decimal.Zero) {
		return nil, opscontract.ErrInvalidInput
	}
	input.Operation = op
	input.Amount = amtDec.Round(2).StringFixed(2)
	return s.users.AdjustUserWallet(id, input)
}

func (s *Service) ClearUserTOTP(id uint) error {
	if s.users == nil {
		return opscontract.ErrNotConfigured
	}
	if id == 0 {
		return opscontract.ErrInvalidInput
	}
	return s.users.ClearUserTOTP(id)
}

func (s *Service) UnbindUserOAuth(id uint, provider string) error {
	if s.users == nil {
		return opscontract.ErrNotConfigured
	}
	if id == 0 {
		return opscontract.ErrInvalidInput
	}
	provider = strings.ToLower(strings.TrimSpace(provider))
	if provider != "telegram" && provider != "google" {
		return opscontract.ErrInvalidInput
	}
	return s.users.UnbindUserOAuth(id, provider)
}
