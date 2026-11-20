package assetclaimhttp

import (
	"errors"
	"strings"

	"github.com/dujiao-next/internal/modules/order/application/assetclaim"
	reseller "github.com/dujiao-next/internal/modules/reseller/contract"
	"github.com/dujiao-next/internal/platform/http/ginutil"
	"github.com/dujiao-next/internal/platform/http/response"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *assetclaim.Service
}

func New(service *assetclaim.Service) *Handler {
	return &Handler{service: service}
}

func RegisterMemberReadRoutes(routes gin.IRoutes, h *Handler) {
	routes.GET("/orders/:order_no/asset-claim", h.memberStatus)
}

func RegisterMemberWriteRoutes(routes gin.IRoutes, h *Handler) {
	routes.POST("/orders/:order_no/asset-claim/reveal", h.memberReveal)
}

func RegisterGuestReadRoutes(routes gin.IRoutes, h *Handler) {
	routes.GET("/orders/:order_no/asset-claim", h.guestStatus)
}

func RegisterGuestWriteRoutes(routes gin.IRoutes, h *Handler) {
	routes.POST("/orders/:order_no/asset-claim/reveal", h.guestReveal)
}

func (h *Handler) memberStatus(c *gin.Context) { h.status(c, true) }
func (h *Handler) guestStatus(c *gin.Context)  { h.status(c, false) }
func (h *Handler) memberReveal(c *gin.Context) { h.reveal(c, true) }
func (h *Handler) guestReveal(c *gin.Context)  { h.reveal(c, false) }

func (h *Handler) subject(c *gin.Context, member bool) (assetclaim.Subject, bool) {
	tenant := reseller.TenantOrMain(c.Request.Context())
	if tenant.Unavailable {
		h.fail(c, assetclaim.ErrNotFound)
		return assetclaim.Subject{}, false
	}
	sub := assetclaim.Subject{
		OrderNo:    strings.TrimSpace(c.Param("order_no")),
		ResellerID: tenant.ResellerID,
	}
	if sub.OrderNo == "" {
		h.fail(c, assetclaim.ErrInvalid)
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
			h.fail(c, assetclaim.ErrNotFound)
			return sub, false
		}
		sub.GuestEmail, sub.GuestPassword = email, pwd
	}
	return sub, true
}

func (h *Handler) status(c *gin.Context, member bool) {
	h.privateHeaders(c)
	if c.Request.URL.RawQuery != "" {
		h.fail(c, assetclaim.ErrInvalid)
		return
	}
	sub, ok := h.subject(c, member)
	if !ok {
		return
	}
	res, err := h.service.Status(c.Request.Context(), sub)
	if err != nil {
		h.fail(c, err)
		return
	}
	response.Success(c, res)
}

func (h *Handler) reveal(c *gin.Context, member bool) {
	h.privateHeaders(c)
	if c.Request.URL.RawQuery != "" {
		h.fail(c, assetclaim.ErrInvalid)
		return
	}
	sub, ok := h.subject(c, member)
	if !ok {
		return
	}
	res, err := h.service.Reveal(c.Request.Context(), sub)
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
	code, key := response.CodeInternal, "error.asset_claim_unavailable"
	switch {
	case errors.Is(err, assetclaim.ErrNotFound):
		code, key = response.CodeNotFound, "error.order_not_found"
	case errors.Is(err, assetclaim.ErrInvalid):
		code, key = response.CodeBadRequest, "error.asset_claim_invalid"
	case errors.Is(err, assetclaim.ErrForbidden):
		code, key = response.CodeForbidden, "error.asset_claim_forbidden"
	case errors.Is(err, assetclaim.ErrExhausted):
		code, key = response.CodeBadRequest, "error.asset_claim_exhausted"
	case errors.Is(err, assetclaim.ErrConflict):
		code, key = response.CodeBadRequest, "error.asset_claim_conflict"
	}
	ginutil.RespondError(c, code, key, nil)
}
