package application

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/dujiao-next/internal/constants"
	categorycontract "github.com/dujiao-next/internal/modules/catalog/category/contract"
	categorydomain "github.com/dujiao-next/internal/modules/catalog/category/domain"
	productcontract "github.com/dujiao-next/internal/modules/catalog/product/contract"
	productdomain "github.com/dujiao-next/internal/modules/catalog/product/domain"
	couponcontract "github.com/dujiao-next/internal/modules/coupon/contract"
	coupondomain "github.com/dujiao-next/internal/modules/coupon/domain"
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
	orders       ordercontract.Store
	payments     paymentcontract.Store
	procurements procurementcontract.Repository
	categories   categorycontract.Repository
	products     productcontract.Repository
	skus         productcontract.SKURepository
	coupons      couponcontract.Repository
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

func (s *Service) Orders(page, pageSize int) (*opscontract.OrdersResult, error) {
	collected := time.Now().UTC()
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 50
	}
	if pageSize > 100 {
		pageSize = 100
	}
	rows, total, err := s.orders.ListAdmin(ordercontract.ListFilter{Page: page, PageSize: pageSize, SortBy: "id", SortOrder: "desc"})
	if err != nil {
		return nil, err
	}
	result := &opscontract.OrdersResult{Source: "storefront", CollectedAt: collected, Freshness: "fresh", Orders: make([]opscontract.Order, 0, len(rows)), Total: total, Page: page, PageSize: pageSize}
	var mapOrder func(orderdomain.Order) (opscontract.Order, error)
	mapOrder = func(row orderdomain.Order) (opscontract.Order, error) {
		out := opscontract.Order{
			ID:                row.ID,
			OrderNo:           row.OrderNo,
			ParentID:          row.ParentID,
			Status:            row.Status,
			Currency:          row.Currency,
			OriginalAmount:    row.OriginalAmount.String(),
			DiscountAmount:    row.DiscountAmount.String(),
			CouponID:          row.CouponID,
			TotalAmount:       row.TotalAmount.String(),
			RefundedAmount:    row.RefundedAmount.String(),
			PaymentStatus:     "unpaid",
			FulfillmentStatus: row.Status,
			EntitlementStatus: "unknown",
			PaidAt:            row.PaidAt,
			CreatedAt:         row.CreatedAt,
			UpdatedAt:         row.UpdatedAt,
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
		payments, err := s.payments.ListByOrderID(row.ID)
		if err != nil {
			return opscontract.Order{}, err
		}
		for _, payment := range payments {
			out.Payments = append(out.Payments, opscontract.Payment{
				ID:           payment.ID,
				ProviderType: payment.ProviderType,
				ChannelType:  payment.ChannelType,
				Status:       payment.Status,
				Amount:       payment.Amount.String(),
				Currency:     payment.Currency,
				ProviderRef:  payment.ProviderRef,
				PaidAt:       payment.PaidAt,
				UpdatedAt:    payment.UpdatedAt,
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
			out.Procurement = &opscontract.Procurement{ID: procurement.ID, Status: procurement.Status, UpstreamOrderNo: procurement.UpstreamOrderNo, UpdatedAt: procurement.UpdatedAt, LocalOrderID: procurement.LocalOrderID}
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
			out.Procurements = append(out.Procurements, mapped.Procurements...)
			out.Reservations = append(out.Reservations, mapped.Reservations...)
		}
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

func (s *Service) Overview() (*opscontract.Overview, error) {
	caps := []string{"orders.read", "payments.read", "refunds.read", "procurement.read", "checkout_reservations.read"}
	if s.categories != nil {
		caps = append(caps, "categories.read", "categories.write")
	}
	if s.products != nil && s.skus != nil {
		caps = append(caps, "products.read", "products.write", "skus.write")
	}
	if s.coupons != nil {
		caps = append(caps, "coupons.read", "coupons.write")
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

func (s *Service) UpdateCategory(id string, input opscontract.UpdateCategoryInput) (*opscontract.Category, error) {
	if s.categories == nil {
		return nil, opscontract.ErrNotConfigured
	}
	if strings.TrimSpace(id) == "" || (input.SortOrder == nil && input.IsActive == nil) {
		return nil, opscontract.ErrInvalidInput
	}
	cat, err := s.categories.GetByID(id)
	if err != nil {
		return nil, err
	}
	if cat == nil {
		return nil, opscontract.ErrNotFound
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

func (s *Service) UpdateProduct(id string, input opscontract.UpdateProductInput) (*opscontract.Product, error) {
	if s.products == nil {
		return nil, opscontract.ErrNotConfigured
	}
	if strings.TrimSpace(id) == "" || (input.IsActive == nil && input.SortOrder == nil) {
		return nil, opscontract.ErrInvalidInput
	}
	prod, err := s.products.GetAdminByID(id)
	if err != nil {
		return nil, err
	}
	if prod == nil {
		return nil, opscontract.ErrNotFound
	}
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
	skus := prod.SKUs
	if len(skus) == 0 && s.skus != nil {
		if listed, err := s.skus.ListByProduct(prod.ID, false); err == nil {
			skus = listed
		}
	}
	out := mapProduct(*prod, skus)
	return &out, nil
}

func (s *Service) UpdateSKU(id uint, input opscontract.UpdateSKUInput) (*opscontract.ProductSKU, error) {
	if s.skus == nil {
		return nil, opscontract.ErrNotConfigured
	}
	if id == 0 || (input.PriceAmount == nil && input.ManualStockTotal == nil && input.IsActive == nil) {
		return nil, opscontract.ErrInvalidInput
	}
	sku, err := s.skus.GetByID(id)
	if err != nil {
		return nil, err
	}
	if sku == nil {
		return nil, opscontract.ErrNotFound
	}
	if input.PriceAmount != nil {
		dec, err := decimal.NewFromString(strings.TrimSpace(*input.PriceAmount))
		if err != nil || dec.IsNegative() {
			return nil, opscontract.ErrInvalidInput
		}
		sku.PriceAmount = money.FromDecimal(dec.Round(2))
	}
	if input.ManualStockTotal != nil {
		if *input.ManualStockTotal < -1 {
			return nil, opscontract.ErrInvalidInput
		}
		sku.ManualStockTotal = *input.ManualStockTotal
	}
	if input.IsActive != nil {
		sku.IsActive = *input.IsActive
	}
	sku.UpdatedAt = time.Now().UTC()
	if err := s.skus.Update(sku); err != nil {
		return nil, err
	}
	// Sync product display price when updating SKU price on single-SKU or lowest active SKU.
	if input.PriceAmount != nil && s.products != nil {
		_ = s.products.QuickUpdate(strconv.FormatUint(uint64(sku.ProductID), 10), map[string]interface{}{
			"price_amount": sku.PriceAmount,
		})
	}
	out := mapSKU(*sku)
	return &out, nil
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
