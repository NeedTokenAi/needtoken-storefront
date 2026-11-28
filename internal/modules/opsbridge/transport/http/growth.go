package http

import (
	"strconv"

	opscontract "github.com/dujiao-next/internal/modules/opsbridge/contract"
	settingscontract "github.com/dujiao-next/internal/modules/settings/contract"
	"github.com/dujiao-next/internal/platform/http/response"
	"github.com/gin-gonic/gin"
)

func (h *Handler) GrowthSummary(c *gin.Context) {
	if h.growth == nil {
		fail(c, opscontract.ErrNotConfigured)
		return
	}
	result, err := h.growth.GrowthSummary()
	if err != nil {
		fail(c, err)
		return
	}
	response.Success(c, result)
}

func (h *Handler) UpdateAffiliateSettings(c *gin.Context) {
	if h.growth == nil {
		fail(c, opscontract.ErrNotConfigured)
		return
	}
	var req opscontract.AffiliateSettingDTO
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, opscontract.ErrInvalidInput)
		return
	}
	result, err := h.growth.UpdateAffiliateSettings(req)
	if err != nil {
		fail(c, err)
		return
	}
	response.Success(c, result)
}

func (h *Handler) ListAffiliateWithdraws(c *gin.Context) {
	if h.growth == nil {
		fail(c, opscontract.ErrNotConfigured)
		return
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "50"))
	result, err := h.growth.ListAffiliateWithdraws(page, pageSize, c.Query("status"))
	if err != nil {
		fail(c, err)
		return
	}
	response.Success(c, result)
}

func (h *Handler) PayAffiliateWithdraw(c *gin.Context) {
	h.reviewAffiliateWithdrawAction(c, "pay")
}

func (h *Handler) RejectAffiliateWithdraw(c *gin.Context) {
	h.reviewAffiliateWithdrawAction(c, "reject")
}

func (h *Handler) reviewAffiliateWithdrawAction(c *gin.Context, action string) {
	if h.growth == nil {
		fail(c, opscontract.ErrNotConfigured)
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		fail(c, opscontract.ErrInvalidInput)
		return
	}
	var req opscontract.ReviewWithdrawInput
	_ = c.ShouldBindJSON(&req)
	result, err := h.growth.ReviewAffiliateWithdraw(uint(id), action, req.RejectReason)
	if err != nil {
		fail(c, err)
		return
	}
	response.Success(c, result)
}

func (h *Handler) UpdateRechargeBonus(c *gin.Context) {
	if h.growth == nil {
		fail(c, opscontract.ErrNotConfigured)
		return
	}
	var req settingscontract.WalletRechargeBonusConfig
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, opscontract.ErrInvalidInput)
		return
	}
	result, err := h.growth.UpdateRechargeBonus(req)
	if err != nil {
		fail(c, err)
		return
	}
	response.Success(c, result)
}

func (h *Handler) ListResellerProfiles(c *gin.Context) {
	if h.growth == nil {
		fail(c, opscontract.ErrNotConfigured)
		return
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "50"))
	result, err := h.growth.ListResellerProfiles(page, pageSize, c.Query("status"), c.Query("keyword"))
	if err != nil {
		fail(c, err)
		return
	}
	response.Success(c, result)
}

func (h *Handler) ApproveResellerProfile(c *gin.Context) {
	if h.growth == nil {
		fail(c, opscontract.ErrNotConfigured)
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		fail(c, opscontract.ErrInvalidInput)
		return
	}
	var req opscontract.ApproveResellerInput
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, opscontract.ErrInvalidInput)
		return
	}
	result, err := h.growth.ApproveResellerProfile(uint(id), req)
	if err != nil {
		fail(c, err)
		return
	}
	response.Success(c, result)
}

func (h *Handler) RejectResellerProfile(c *gin.Context) {
	if h.growth == nil {
		fail(c, opscontract.ErrNotConfigured)
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		fail(c, opscontract.ErrInvalidInput)
		return
	}
	var req opscontract.RejectResellerInput
	_ = c.ShouldBindJSON(&req)
	result, err := h.growth.RejectResellerProfile(uint(id), req)
	if err != nil {
		fail(c, err)
		return
	}
	response.Success(c, result)
}

func (h *Handler) UpdateResellerProfileStatus(c *gin.Context) {
	if h.growth == nil {
		fail(c, opscontract.ErrNotConfigured)
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		fail(c, opscontract.ErrInvalidInput)
		return
	}
	var req opscontract.UpdateResellerStatusInput
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, opscontract.ErrInvalidInput)
		return
	}
	result, err := h.growth.UpdateResellerProfileStatus(uint(id), req)
	if err != nil {
		fail(c, err)
		return
	}
	response.Success(c, result)
}

func (h *Handler) ListResellerWithdraws(c *gin.Context) {
	if h.growth == nil {
		fail(c, opscontract.ErrNotConfigured)
		return
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "50"))
	result, err := h.growth.ListResellerWithdraws(page, pageSize, c.Query("status"))
	if err != nil {
		fail(c, err)
		return
	}
	response.Success(c, result)
}

func (h *Handler) PayResellerWithdraw(c *gin.Context) {
	h.reviewResellerWithdrawAction(c, "pay")
}

func (h *Handler) RejectResellerWithdraw(c *gin.Context) {
	h.reviewResellerWithdrawAction(c, "reject")
}

func (h *Handler) reviewResellerWithdrawAction(c *gin.Context, action string) {
	if h.growth == nil {
		fail(c, opscontract.ErrNotConfigured)
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		fail(c, opscontract.ErrInvalidInput)
		return
	}
	var req opscontract.ReviewWithdrawInput
	_ = c.ShouldBindJSON(&req)
	result, err := h.growth.ReviewResellerWithdraw(uint(id), action, req.RejectReason)
	if err != nil {
		fail(c, err)
		return
	}
	response.Success(c, result)
}
