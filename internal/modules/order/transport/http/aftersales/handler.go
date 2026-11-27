package aftersaleshttp

import (
	"errors"
	"strings"

	"github.com/dujiao-next/internal/modules/order/application/aftersales"
	reseller "github.com/dujiao-next/internal/modules/reseller/contract"
	"github.com/dujiao-next/internal/platform/http/ginutil"
	"github.com/dujiao-next/internal/platform/http/response"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *aftersales.Service
}

func New(service *aftersales.Service) *Handler {
	return &Handler{service: service}
}

func RegisterMemberReadRoutes(routes gin.IRoutes, h *Handler) {
	routes.GET("/orders/:order_no/after-sales", h.memberList)
}

func RegisterMemberWriteRoutes(routes gin.IRoutes, h *Handler) {
	routes.POST("/orders/:order_no/after-sales", h.memberCreate)
	routes.POST("/orders/:order_no/after-sales/:ticket_id/supplement", h.memberSupplement)
}

func RegisterGuestReadRoutes(routes gin.IRoutes, h *Handler) {
	routes.GET("/orders/:order_no/after-sales", h.guestList)
}

func RegisterGuestWriteRoutes(routes gin.IRoutes, h *Handler) {
	routes.POST("/orders/:order_no/after-sales", h.guestCreate)
	routes.POST("/orders/:order_no/after-sales/:ticket_id/supplement", h.guestSupplement)
}

func (h *Handler) memberList(c *gin.Context)       { h.list(c, true) }
func (h *Handler) guestList(c *gin.Context)        { h.list(c, false) }
func (h *Handler) memberCreate(c *gin.Context)     { h.create(c, true) }
func (h *Handler) guestCreate(c *gin.Context)      { h.create(c, false) }
func (h *Handler) memberSupplement(c *gin.Context) { h.supplement(c, true) }
func (h *Handler) guestSupplement(c *gin.Context)  { h.supplement(c, false) }

func (h *Handler) subject(c *gin.Context, member bool) (aftersales.Subject, bool) {
	tenant := reseller.TenantOrMain(c.Request.Context())
	if tenant.Unavailable {
		h.fail(c, aftersales.ErrNotFound)
		return aftersales.Subject{}, false
	}
	sub := aftersales.Subject{
		OrderNo:    strings.TrimSpace(c.Param("order_no")),
		ResellerID: tenant.ResellerID,
	}
	if sub.OrderNo == "" {
		h.fail(c, aftersales.ErrInvalid)
		return sub, false
	}
	if member {
		uid, ok := ginutil.GetUserID(c)
		if !ok || uid == 0 {
			return sub, false
		}
		sub.UserID = uid
	} else {
		email, pwd, ok := ginutil.GetGuestCredentials(c)
		if !ok {
			h.fail(c, aftersales.ErrNotFound)
			return sub, false
		}
		sub.GuestEmail, sub.GuestPassword = email, pwd
	}
	return sub, true
}

func (h *Handler) list(c *gin.Context, member bool) {
	h.privateHeaders(c)
	if c.Request.URL.RawQuery != "" {
		h.fail(c, aftersales.ErrInvalid)
		return
	}
	sub, ok := h.subject(c, member)
	if !ok {
		return
	}
	res, err := h.service.List(c.Request.Context(), sub)
	if err != nil {
		h.fail(c, err)
		return
	}
	response.Success(c, res)
}

func (h *Handler) create(c *gin.Context, member bool) {
	h.privateHeaders(c)
	sub, ok := h.subject(c, member)
	if !ok {
		return
	}
	var req struct {
		Category       string `json:"category"`
		Summary        string `json:"summary"`
		IdempotencyKey string `json:"idempotency_key"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		h.fail(c, aftersales.ErrInvalid)
		return
	}
	res, err := h.service.Create(c.Request.Context(), sub, aftersales.CreateInput{
		Category:       req.Category,
		Summary:        req.Summary,
		IdempotencyKey: req.IdempotencyKey,
	})
	if err != nil {
		h.fail(c, err)
		return
	}
	response.Success(c, res)
}

func (h *Handler) supplement(c *gin.Context, member bool) {
	h.privateHeaders(c)
	sub, ok := h.subject(c, member)
	if !ok {
		return
	}
	ticketID := strings.TrimSpace(c.Param("ticket_id"))
	var req struct {
		Message string `json:"message"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		h.fail(c, aftersales.ErrInvalid)
		return
	}
	res, err := h.service.AppendSupplement(c.Request.Context(), sub, ticketID, req.Message)
	if err != nil {
		h.fail(c, err)
		return
	}
	response.Success(c, res)
}

func (h *Handler) privateHeaders(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	c.Header("Referrer-Policy", "no-referrer")
	c.Header("X-Content-Type-Options", "nosniff")
}

func (h *Handler) fail(c *gin.Context, err error) {
	code, key := response.CodeInternal, "error.after_sales_unavailable"
	switch {
	case errors.Is(err, aftersales.ErrNotFound):
		code, key = response.CodeNotFound, "error.order_not_found"
	case errors.Is(err, aftersales.ErrInvalid):
		code, key = response.CodeBadRequest, "error.after_sales_invalid"
	case errors.Is(err, aftersales.ErrConflict):
		code, key = response.CodeBadRequest, "error.after_sales_conflict"
	}
	ginutil.RespondError(c, code, key, nil)
}
