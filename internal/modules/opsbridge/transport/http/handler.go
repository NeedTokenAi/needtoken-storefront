package http

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/dujiao-next/internal/i18n"
	opscontract "github.com/dujiao-next/internal/modules/opsbridge/contract"
	"github.com/dujiao-next/internal/platform/http/response"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	reader    opscontract.Reader
	bridge    opscontract.Bridge
	slo       opscontract.SLORevokeHandler
	ugcClaims opscontract.UniversalGiftCardClaimLookup
}

func New(reader opscontract.Reader) *Handler {
	h := &Handler{reader: reader}
	if b, ok := reader.(opscontract.Bridge); ok {
		h.bridge = b
	}
	if s, ok := reader.(opscontract.SLORevokeHandler); ok {
		h.slo = s
	}
	if u, ok := reader.(opscontract.UniversalGiftCardClaimLookup); ok {
		h.ugcClaims = u
	}
	return h
}

func (h *Handler) Overview(c *gin.Context) {
	result, err := h.reader.Overview()
	if err != nil {
		fail(c, err)
		return
	}
	response.Success(c, result)
}

func (h *Handler) Orders(c *gin.Context) {
	page, _ := strconv.Atoi(c.Query("page"))
	size, _ := strconv.Atoi(c.Query("page_size"))
	if h.bridge == nil {
		result, err := h.reader.Orders(page, size)
		if err != nil {
			fail(c, err)
			return
		}
		response.Success(c, result)
		return
	}
	filter := opscontract.OrdersFilter{
		Page:            page,
		PageSize:        size,
		Status:          strings.TrimSpace(c.Query("status")),
		OrderNo:         strings.TrimSpace(c.Query("order_no")),
		Keyword:         strings.TrimSpace(c.Query("keyword")),
		FulfillmentType: strings.TrimSpace(c.Query("fulfillment_type")),
		BuyerType:       strings.TrimSpace(c.Query("buyer_type")),
		PaymentChannel:  strings.TrimSpace(c.Query("payment_channel")),
		ClientIP:        strings.TrimSpace(c.Query("client_ip")),
	}
	if rawCat := strings.TrimSpace(c.Query("category_id")); rawCat != "" {
		if id, err := strconv.ParseUint(rawCat, 10, 64); err == nil {
			filter.CategoryID = uint(id)
		}
	}
	if rawExc := strings.ToLower(strings.TrimSpace(c.Query("exception_only"))); rawExc == "true" || rawExc == "1" {
		filter.ExceptionOnly = true
	}
	if from := parseQueryTime(c.Query("created_from"), false); from != nil {
		filter.CreatedFrom = from
	}
	if to := parseQueryTime(c.Query("created_to"), true); to != nil {
		filter.CreatedTo = to
	}
	result, err := h.bridge.OrdersFiltered(filter)
	if err != nil {
		fail(c, err)
		return
	}
	response.Success(c, result)
}

func parseQueryTime(raw string, endOfDay bool) *time.Time {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil
	}
	if t, err := time.Parse(time.RFC3339, trimmed); err == nil {
		return &t
	}
	if t, err := time.Parse(time.RFC3339Nano, trimmed); err == nil {
		return &t
	}
	if t, err := time.Parse("2006-01-02", trimmed); err == nil {
		if endOfDay {
			t = t.Add(24*time.Hour - time.Nanosecond)
		}
		return &t
	}
	return nil
}

func (h *Handler) ManualConfirmPayment(c *gin.Context) {
	if h.bridge == nil {
		fail(c, opscontract.ErrNotConfigured)
		return
	}
	var req opscontract.ManualConfirmPaymentInput
	if c.Request.ContentLength > 0 {
		if err := c.ShouldBindJSON(&req); err != nil {
			fail(c, opscontract.ErrInvalidInput)
			return
		}
	}
	result, err := h.bridge.ManualConfirmPayment(c.Param("order_no"), req)
	if err != nil {
		fail(c, err)
		return
	}
	response.Success(c, result)
}

func (h *Handler) ManualFulfillOrder(c *gin.Context) {
	if h.bridge == nil {
		fail(c, opscontract.ErrNotConfigured)
		return
	}
	var req opscontract.ManualFulfillOrderInput
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, opscontract.ErrInvalidInput)
		return
	}
	result, err := h.bridge.ManualFulfillOrder(c.Param("order_no"), req)
	if err != nil {
		fail(c, err)
		return
	}
	response.Success(c, result)
}

func (h *Handler) UpdateOrderStatus(c *gin.Context) {
	if h.bridge == nil {
		fail(c, opscontract.ErrNotConfigured)
		return
	}
	var req opscontract.UpdateOrderStatusInput
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, opscontract.ErrInvalidInput)
		return
	}
	result, err := h.bridge.UpdateOrderStatus(c.Param("order_no"), req)
	if err != nil {
		fail(c, err)
		return
	}
	response.Success(c, result)
}

func (h *Handler) RetryProcurement(c *gin.Context) {
	if h.bridge == nil {
		fail(c, opscontract.ErrNotConfigured)
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		fail(c, opscontract.ErrInvalidInput)
		return
	}
	result, err := h.bridge.RetryProcurement(uint(id))
	if err != nil {
		fail(c, err)
		return
	}
	response.Success(c, result)
}

func (h *Handler) Categories(c *gin.Context) {
	if h.bridge == nil {
		fail(c, opscontract.ErrNotConfigured)
		return
	}
	result, err := h.bridge.Categories()
	if err != nil {
		fail(c, err)
		return
	}
	response.Success(c, result)
}

func (h *Handler) CreateCategory(c *gin.Context) {
	if h.bridge == nil {
		fail(c, opscontract.ErrNotConfigured)
		return
	}
	var req opscontract.CreateCategoryInput
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, opscontract.ErrInvalidInput)
		return
	}
	result, err := h.bridge.CreateCategory(req)
	if err != nil {
		fail(c, err)
		return
	}
	response.Success(c, result)
}

func (h *Handler) UpdateCategory(c *gin.Context) {
	if h.bridge == nil {
		fail(c, opscontract.ErrNotConfigured)
		return
	}
	var req opscontract.UpdateCategoryInput
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, opscontract.ErrInvalidInput)
		return
	}
	result, err := h.bridge.UpdateCategory(c.Param("id"), req)
	if err != nil {
		fail(c, err)
		return
	}
	response.Success(c, result)
}

func (h *Handler) DeleteCategory(c *gin.Context) {
	if h.bridge == nil {
		fail(c, opscontract.ErrNotConfigured)
		return
	}
	rawID := c.Param("id")
	id, err := strconv.ParseUint(rawID, 10, 64)
	if err != nil || id == 0 {
		fail(c, opscontract.ErrInvalidInput)
		return
	}
	if err := h.bridge.DeleteCategory(rawID); err != nil {
		fail(c, err)
		return
	}
	response.Success(c, opscontract.DeleteResult{Deleted: true, ID: uint(id)})
}

func (h *Handler) Products(c *gin.Context) {
	if h.bridge == nil {
		fail(c, opscontract.ErrNotConfigured)
		return
	}
	page, _ := strconv.Atoi(c.Query("page"))
	size, _ := strconv.Atoi(c.Query("page_size"))
	result, err := h.bridge.Products(page, size, c.Query("category_id"), c.Query("search"))
	if err != nil {
		fail(c, err)
		return
	}
	response.Success(c, result)
}

func (h *Handler) CreateProduct(c *gin.Context) {
	if h.bridge == nil {
		fail(c, opscontract.ErrNotConfigured)
		return
	}
	var req opscontract.CreateProductInput
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, opscontract.ErrInvalidInput)
		return
	}
	result, err := h.bridge.CreateProduct(req)
	if err != nil {
		fail(c, err)
		return
	}
	response.Success(c, result)
}

func (h *Handler) UpdateProduct(c *gin.Context) {
	if h.bridge == nil {
		fail(c, opscontract.ErrNotConfigured)
		return
	}
	var req opscontract.UpdateProductInput
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, opscontract.ErrInvalidInput)
		return
	}
	result, err := h.bridge.UpdateProduct(c.Param("id"), req)
	if err != nil {
		fail(c, err)
		return
	}
	response.Success(c, result)
}

func (h *Handler) DeleteProduct(c *gin.Context) {
	if h.bridge == nil {
		fail(c, opscontract.ErrNotConfigured)
		return
	}
	rawID := c.Param("id")
	id, err := strconv.ParseUint(rawID, 10, 64)
	if err != nil || id == 0 {
		fail(c, opscontract.ErrInvalidInput)
		return
	}
	if err := h.bridge.DeleteProduct(rawID); err != nil {
		fail(c, err)
		return
	}
	response.Success(c, opscontract.DeleteResult{Deleted: true, ID: uint(id)})
}

func (h *Handler) BatchDeleteProducts(c *gin.Context) {
	if h.bridge == nil {
		fail(c, opscontract.ErrNotConfigured)
		return
	}
	var req opscontract.BatchDeleteProductsInput
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, opscontract.ErrInvalidInput)
		return
	}
	result, err := h.bridge.BatchDeleteProducts(req)
	if err != nil {
		fail(c, err)
		return
	}
	response.Success(c, result)
}

func (h *Handler) CreateSKU(c *gin.Context) {
	if h.bridge == nil {
		fail(c, opscontract.ErrNotConfigured)
		return
	}
	productID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || productID == 0 {
		fail(c, opscontract.ErrInvalidInput)
		return
	}
	var req opscontract.CreateSKUInput
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, opscontract.ErrInvalidInput)
		return
	}
	result, err := h.bridge.CreateSKU(uint(productID), req)
	if err != nil {
		fail(c, err)
		return
	}
	response.Success(c, result)
}

func (h *Handler) UpdateSKU(c *gin.Context) {
	if h.bridge == nil {
		fail(c, opscontract.ErrNotConfigured)
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		fail(c, opscontract.ErrInvalidInput)
		return
	}
	var req opscontract.UpdateSKUInput
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, opscontract.ErrInvalidInput)
		return
	}
	result, err := h.bridge.UpdateSKU(uint(id), req)
	if err != nil {
		fail(c, err)
		return
	}
	response.Success(c, result)
}

func (h *Handler) DeleteSKU(c *gin.Context) {
	if h.bridge == nil {
		fail(c, opscontract.ErrNotConfigured)
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		fail(c, opscontract.ErrInvalidInput)
		return
	}
	if err := h.bridge.DeleteSKU(uint(id)); err != nil {
		fail(c, err)
		return
	}
	response.Success(c, opscontract.DeleteResult{Deleted: true, ID: uint(id)})
}

func (h *Handler) Coupons(c *gin.Context) {
	if h.bridge == nil {
		fail(c, opscontract.ErrNotConfigured)
		return
	}
	page, _ := strconv.Atoi(c.Query("page"))
	size, _ := strconv.Atoi(c.Query("page_size"))
	var isActive *bool
	if raw := c.Query("is_active"); raw == "true" || raw == "false" {
		v := raw == "true"
		isActive = &v
	}
	result, err := h.bridge.Coupons(page, size, c.Query("code"), isActive)
	if err != nil {
		fail(c, err)
		return
	}
	response.Success(c, result)
}

func (h *Handler) CreateCoupon(c *gin.Context) {
	if h.bridge == nil {
		fail(c, opscontract.ErrNotConfigured)
		return
	}
	var req opscontract.CreateCouponInput
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, opscontract.ErrInvalidInput)
		return
	}
	result, err := h.bridge.CreateCoupon(req)
	if err != nil {
		fail(c, err)
		return
	}
	response.Success(c, result)
}

func (h *Handler) UpdateCouponStatus(c *gin.Context) {
	if h.bridge == nil {
		fail(c, opscontract.ErrNotConfigured)
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		fail(c, opscontract.ErrInvalidInput)
		return
	}
	var req opscontract.UpdateCouponStatusInput
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, opscontract.ErrInvalidInput)
		return
	}
	result, err := h.bridge.UpdateCouponStatus(uint(id), req.IsActive)
	if err != nil {
		fail(c, err)
		return
	}
	response.Success(c, result)
}

func (h *Handler) GiftCards(c *gin.Context) {
	if h.bridge == nil {
		fail(c, opscontract.ErrNotConfigured)
		return
	}
	page, _ := strconv.Atoi(c.Query("page"))
	size, _ := strconv.Atoi(c.Query("page_size"))
	var redeemedUserID uint
	if rawUID := c.Query("redeemed_user_id"); rawUID != "" {
		parsed, err := strconv.ParseUint(rawUID, 10, 64)
		if err != nil || parsed == 0 {
			fail(c, opscontract.ErrInvalidInput)
			return
		}
		redeemedUserID = uint(parsed)
	}
	result, err := h.bridge.GiftCards(
		page,
		size,
		c.Query("code"),
		c.Query("status"),
		c.Query("batch_no"),
		redeemedUserID,
	)
	if err != nil {
		fail(c, err)
		return
	}
	response.Success(c, result)
}

func (h *Handler) GenerateGiftCards(c *gin.Context) {
	response.ChannelError(
		c,
		http.StatusGone,
		response.CodeBadRequest,
		"single_side_gift_card_generation_disabled",
		"Single-side storefront gift card generation is disabled; please issue universal gift cards in NeedToken Ops.",
	)
}

func (h *Handler) Pricing(c *gin.Context) {
	if h.bridge == nil {
		fail(c, opscontract.ErrNotConfigured)
		return
	}
	result, err := h.bridge.Pricing()
	if err != nil {
		fail(c, err)
		return
	}
	response.Success(c, result)
}

func (h *Handler) UpdatePricing(c *gin.Context) {
	if h.bridge == nil {
		fail(c, opscontract.ErrNotConfigured)
		return
	}
	var req opscontract.UpdatePricingInput
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, opscontract.ErrInvalidInput)
		return
	}
	result, err := h.bridge.UpdatePricing(req)
	if err != nil {
		fail(c, err)
		return
	}
	response.Success(c, result)
}

func (h *Handler) AcquiringChannels(c *gin.Context) {
	if h.bridge == nil {
		fail(c, opscontract.ErrNotConfigured)
		return
	}
	result, err := h.bridge.AcquiringChannels()
	if err != nil {
		fail(c, err)
		return
	}
	response.Success(c, result)
}

func (h *Handler) UpsertAcquiringChannel(c *gin.Context) {
	if h.bridge == nil {
		fail(c, opscontract.ErrNotConfigured)
		return
	}
	var req opscontract.UpsertAcquiringChannelInput
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, opscontract.ErrInvalidInput)
		return
	}
	result, err := h.bridge.UpsertAcquiringChannel(req)
	if err != nil {
		fail(c, err)
		return
	}
	response.Success(c, result)
}

func (h *Handler) UpdateAcquiringChannelStatus(c *gin.Context) {
	if h.bridge == nil {
		fail(c, opscontract.ErrNotConfigured)
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		fail(c, opscontract.ErrInvalidInput)
		return
	}
	var req opscontract.UpdateAcquiringChannelStatusInput
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, opscontract.ErrInvalidInput)
		return
	}
	result, err := h.bridge.UpdateAcquiringChannelStatus(uint(id), req.IsActive)
	if err != nil {
		fail(c, err)
		return
	}
	response.Success(c, result)
}

func (h *Handler) MailHealth(c *gin.Context) {
	if h.bridge == nil {
		fail(c, opscontract.ErrNotConfigured)
		return
	}
	result, err := h.bridge.MailHealth()
	if err != nil {
		fail(c, err)
		return
	}
	response.Success(c, result)
}

func (h *Handler) UpstreamConnections(c *gin.Context) {
	if h.bridge == nil {
		fail(c, opscontract.ErrNotConfigured)
		return
	}
	result, err := h.bridge.UpstreamConnections()
	if err != nil {
		fail(c, err)
		return
	}
	response.Success(c, result)
}

func (h *Handler) SaveUpstreamConnection(c *gin.Context) {
	if h.bridge == nil {
		fail(c, opscontract.ErrNotConfigured)
		return
	}
	var req opscontract.SaveUpstreamConnectionInput
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, opscontract.ErrInvalidInput)
		return
	}
	result, err := h.bridge.SaveUpstreamConnection(req)
	if err != nil {
		fail(c, err)
		return
	}
	response.Success(c, result)
}

func (h *Handler) UpdateUpstreamConnectionStatus(c *gin.Context) {
	if h.bridge == nil {
		fail(c, opscontract.ErrNotConfigured)
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		fail(c, opscontract.ErrInvalidInput)
		return
	}
	var req opscontract.UpdateUpstreamConnectionStatusInput
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, opscontract.ErrInvalidInput)
		return
	}
	result, err := h.bridge.UpdateUpstreamConnectionStatus(uint(id), req.Status)
	if err != nil {
		fail(c, err)
		return
	}
	response.Success(c, result)
}

func (h *Handler) DeleteUpstreamConnection(c *gin.Context) {
	if h.bridge == nil {
		fail(c, opscontract.ErrNotConfigured)
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		fail(c, opscontract.ErrInvalidInput)
		return
	}
	if err := h.bridge.DeleteUpstreamConnection(uint(id)); err != nil {
		fail(c, err)
		return
	}
	response.Success(c, opscontract.DeleteResult{Deleted: true, ID: uint(id)})
}

func (h *Handler) UpstreamCatalog(c *gin.Context) {
	if h.bridge == nil {
		fail(c, opscontract.ErrNotConfigured)
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		fail(c, opscontract.ErrInvalidInput)
		return
	}
	result, err := h.bridge.UpstreamCatalog(uint(id))
	if err != nil {
		fail(c, err)
		return
	}
	response.Success(c, result)
}

func (h *Handler) ImportUpstreamProducts(c *gin.Context) {
	if h.bridge == nil {
		fail(c, opscontract.ErrNotConfigured)
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		fail(c, opscontract.ErrInvalidInput)
		return
	}
	var req opscontract.ImportUpstreamProductsInput
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, opscontract.ErrInvalidInput)
		return
	}
	result, err := h.bridge.ImportUpstreamProducts(uint(id), req)
	if err != nil {
		fail(c, err)
		return
	}
	response.Success(c, result)
}

func (h *Handler) RefundOrder(c *gin.Context) {
	if h.bridge == nil {
		fail(c, opscontract.ErrNotConfigured)
		return
	}
	var req opscontract.RefundOrderInput
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, opscontract.ErrInvalidInput)
		return
	}
	result, err := h.bridge.RefundOrder(c.Param("order_no"), req)
	if err != nil {
		fail(c, err)
		return
	}
	response.Success(c, result)
}

func (h *Handler) RevokeUserSLO(c *gin.Context) {
	if h.slo == nil {
		fail(c, opscontract.ErrNotConfigured)
		return
	}
	var req opscontract.SLORevokeUserInput
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, opscontract.ErrInvalidInput)
		return
	}
	result, err := h.slo.RevokeUserSLO(req)
	if err != nil {
		fail(c, err)
		return
	}
	response.Success(c, result)
}

func (h *Handler) LookupUniversalGiftCardClaim(c *gin.Context) {
	if h.ugcClaims == nil {
		fail(c, opscontract.ErrNotConfigured)
		return
	}
	claimID := strings.TrimSpace(c.Param("claim_id"))
	if claimID == "" {
		fail(c, opscontract.ErrInvalidInput)
		return
	}
	result, err := h.ugcClaims.LookupUniversalGiftCardClaim(claimID)
	if err != nil {
		fail(c, err)
		return
	}
	response.Success(c, result)
}

func (h *Handler) Users(c *gin.Context) {
	if h.bridge == nil {
		fail(c, opscontract.ErrNotConfigured)
		return
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "50"))
	result, err := h.bridge.Users(page, pageSize, c.Query("keyword"), c.Query("status"))
	if err != nil {
		fail(c, err)
		return
	}
	response.Success(c, result)
}

func (h *Handler) CreateUser(c *gin.Context) {
	if h.bridge == nil {
		fail(c, opscontract.ErrNotConfigured)
		return
	}
	var req opscontract.CreateUserInput
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, opscontract.ErrInvalidInput)
		return
	}
	result, err := h.bridge.CreateUser(req)
	if err != nil {
		fail(c, err)
		return
	}
	response.Success(c, result)
}

func (h *Handler) UpdateUser(c *gin.Context) {
	if h.bridge == nil {
		fail(c, opscontract.ErrNotConfigured)
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		fail(c, opscontract.ErrInvalidInput)
		return
	}
	var req opscontract.UpdateUserInput
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, opscontract.ErrInvalidInput)
		return
	}
	result, err := h.bridge.UpdateUser(uint(id), req)
	if err != nil {
		fail(c, err)
		return
	}
	response.Success(c, result)
}

func (h *Handler) DeleteUser(c *gin.Context) {
	if h.bridge == nil {
		fail(c, opscontract.ErrNotConfigured)
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		fail(c, opscontract.ErrInvalidInput)
		return
	}
	if err := h.bridge.DeleteUser(uint(id)); err != nil {
		fail(c, err)
		return
	}
	response.Success(c, opscontract.DeleteResult{Deleted: true, ID: uint(id)})
}

func (h *Handler) AdjustUserWallet(c *gin.Context) {
	if h.bridge == nil {
		fail(c, opscontract.ErrNotConfigured)
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		fail(c, opscontract.ErrInvalidInput)
		return
	}
	var req opscontract.AdjustUserWalletInput
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, opscontract.ErrInvalidInput)
		return
	}
	result, err := h.bridge.AdjustUserWallet(uint(id), req)
	if err != nil {
		fail(c, err)
		return
	}
	response.Success(c, result)
}

func (h *Handler) ClearUserTOTP(c *gin.Context) {
	if h.bridge == nil {
		fail(c, opscontract.ErrNotConfigured)
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		fail(c, opscontract.ErrInvalidInput)
		return
	}
	if err := h.bridge.ClearUserTOTP(uint(id)); err != nil {
		fail(c, err)
		return
	}
	response.Success(c, gin.H{"cleared": true, "user_id": uint(id)})
}

func (h *Handler) UnbindUserOAuth(c *gin.Context) {
	if h.bridge == nil {
		fail(c, opscontract.ErrNotConfigured)
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		fail(c, opscontract.ErrInvalidInput)
		return
	}
	if err := h.bridge.UnbindUserOAuth(uint(id), c.Param("provider")); err != nil {
		fail(c, err)
		return
	}
	response.Success(c, gin.H{"unbound": true, "user_id": uint(id), "provider": strings.ToLower(strings.TrimSpace(c.Param("provider")))})
}

func fail(c *gin.Context, err error) {
	locale := i18n.ResolveLocale(c)
	var conflictErr *opscontract.ConflictError
	switch {
	case errors.Is(err, opscontract.ErrInvalidInput):
		response.ErrorWithHTTPStatus(c, http.StatusBadRequest, response.CodeBadRequest, i18n.T(locale, "error.bad_request"))
	case errors.Is(err, opscontract.ErrNotFound):
		response.ErrorWithHTTPStatus(c, http.StatusNotFound, response.CodeNotFound, i18n.T(locale, "error.not_found"))
	case errors.As(err, &conflictErr):
		response.ChannelError(c, http.StatusConflict, response.CodeBadRequest, conflictErr.Code, conflictErr.Code)
	case errors.Is(err, opscontract.ErrConflict):
		response.ErrorWithHTTPStatus(c, http.StatusConflict, response.CodeBadRequest, i18n.T(locale, "error.bad_request"))
	default:
		response.ErrorWithHTTPStatus(c, http.StatusServiceUnavailable, response.CodeInternal, i18n.T(locale, "error.internal_error"))
	}
}
