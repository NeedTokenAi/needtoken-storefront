package accountlinkhttp

import (
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"

	"github.com/dujiao-next/internal/modules/order/application/accountlink"
	reseller "github.com/dujiao-next/internal/modules/reseller/contract"
	"github.com/dujiao-next/internal/platform/http/ginutil"
	"github.com/dujiao-next/internal/platform/http/response"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *accountlink.Service
}

func New(service *accountlink.Service) *Handler {
	return &Handler{service: service}
}

func RegisterMemberReadRoutes(routes gin.IRoutes, h *Handler) {
	routes.GET("/me/api-account-link", h.status)
	routes.GET("/me/api-account-summary", h.apiSummary)
}

func RegisterMemberWriteRoutes(routes gin.IRoutes, h *Handler) {
	routes.POST("/me/api-account-link/start", h.start)
	routes.POST("/me/api-account-link/complete", h.complete)
	routes.POST("/me/api-account-link/ensure", h.ensure)
	routes.POST("/me/api-console-sso", h.consoleSSO)
	routes.POST("/me/logout", h.logout)
}

func RegisterMemberRoutes(routes gin.IRoutes, h *Handler) {
	RegisterMemberReadRoutes(routes, h)
	RegisterMemberWriteRoutes(routes, h)
}

func (h *Handler) status(c *gin.Context) {
	h.privateHeaders(c)
	if c.Request.URL.RawQuery != "" {
		h.fail(c, accountlink.ErrInvalid)
		return
	}
	userID, ok := h.memberOnMainTenant(c)
	if !ok {
		return
	}
	result, err := h.service.Status(c.Request.Context(), userID)
	if err != nil {
		h.fail(c, err)
		return
	}
	response.Success(c, result)
}

func (h *Handler) start(c *gin.Context) {
	h.privateHeaders(c)
	userID, ok := h.memberOnMainTenant(c)
	if !ok {
		return
	}
	var input struct {
		NextPath string `json:"next_path"`
	}
	if !h.decodeJSON(c, &input, true) {
		return
	}
	result, err := h.service.Start(c.Request.Context(), userID, input.NextPath)
	if err != nil {
		h.fail(c, err)
		return
	}
	response.Success(c, result)
}

func (h *Handler) complete(c *gin.Context) {
	h.privateHeaders(c)
	userID, ok := h.memberOnMainTenant(c)
	if !ok {
		return
	}
	var input struct {
		ChallengeID string `json:"challenge_id"`
		Approval    string `json:"approval"`
	}
	if !h.decodeJSON(c, &input, false) {
		return
	}
	result, err := h.service.Complete(c.Request.Context(), userID, input.ChallengeID, input.Approval)
	if err != nil {
		h.fail(c, err)
		return
	}
	response.Success(c, result)
}

func (h *Handler) consoleSSO(c *gin.Context) {
	h.privateHeaders(c)
	tenant := reseller.TenantOrMain(c.Request.Context())
	if tenant.Unavailable || tenant.ResellerID != nil {
		h.fail(c, accountlink.ErrForbidden)
		return
	}
	userID, ok := ginutil.GetUserID(c)
	if !ok || userID == 0 {
		h.fail(c, accountlink.ErrForbidden)
		return
	}
	var input struct {
		TargetPath string `json:"target_path"`
		NextPath   string `json:"next_path"`
	}
	if !h.decodeJSON(c, &input, true) {
		return
	}
	result, err := h.service.ConsoleSSO(c.Request.Context(), userID, input.TargetPath, input.NextPath)
	if err != nil {
		h.fail(c, err)
		return
	}
	response.Success(c, result)
}

func (h *Handler) ensure(c *gin.Context) {
	h.privateHeaders(c)
	tenant := reseller.TenantOrMain(c.Request.Context())
	if tenant.Unavailable || tenant.ResellerID != nil {
		response.Success(c, accountlink.AutoLinkResult{Linked: false, Status: "disabled"})
		return
	}
	userID, ok := ginutil.GetUserID(c)
	if !ok || userID == 0 {
		h.fail(c, accountlink.ErrForbidden)
		return
	}
	var empty struct{}
	if !h.decodeJSON(c, &empty, true) {
		return
	}
	result, err := h.service.Ensure(c.Request.Context(), userID)
	if err != nil {
		h.fail(c, err)
		return
	}
	response.Success(c, result)
}

func (h *Handler) apiSummary(c *gin.Context) {
	h.privateHeaders(c)
	if c.Request.URL.RawQuery != "" {
		h.fail(c, accountlink.ErrInvalid)
		return
	}
	tenant := reseller.TenantOrMain(c.Request.Context())
	if tenant.Unavailable || tenant.ResellerID != nil {
		response.Success(c, accountlink.APISummaryResult{Linked: false})
		return
	}
	userID, ok := ginutil.GetUserID(c)
	if !ok || userID == 0 {
		h.fail(c, accountlink.ErrForbidden)
		return
	}
	result, err := h.service.APISummary(c.Request.Context(), userID)
	if err != nil {
		h.fail(c, err)
		return
	}
	response.Success(c, result)
}

func (h *Handler) logout(c *gin.Context) {
	h.privateHeaders(c)
	userID, ok := ginutil.GetUserID(c)
	if !ok || userID == 0 {
		h.fail(c, accountlink.ErrForbidden)
		return
	}
	var empty struct{}
	if !h.decodeJSON(c, &empty, true) {
		return
	}
	result, err := h.service.Logout(c.Request.Context(), userID)
	if err != nil {
		h.fail(c, err)
		return
	}
	response.Success(c, result)
}

func (h *Handler) memberOnMainTenant(c *gin.Context) (uint, bool) {
	tenant := reseller.TenantOrMain(c.Request.Context())
	if tenant.Unavailable || tenant.ResellerID != nil {
		response.Success(c, accountlink.Status{Enabled: false, Linked: false, Status: "disabled"})
		return 0, false
	}
	userID, ok := ginutil.GetUserID(c)
	if !ok || userID == 0 {
		return 0, false
	}
	return userID, true
}

func (h *Handler) privateHeaders(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	c.Header("Referrer-Policy", "no-referrer")
	c.Header("X-Content-Type-Options", "nosniff")
}

func (h *Handler) decodeJSON(c *gin.Context, out any, allowEmpty bool) bool {
	if c.Request.URL.RawQuery != "" || c.GetHeader("Content-Encoding") != "" {
		h.fail(c, accountlink.ErrInvalid)
		return false
	}
	if allowEmpty && c.Request.ContentLength == 0 {
		return true
	}
	media, _, err := mime.ParseMediaType(c.GetHeader("Content-Type"))
	if err != nil || media != "application/json" {
		h.fail(c, accountlink.ErrInvalid)
		return false
	}
	decoder := json.NewDecoder(http.MaxBytesReader(c.Writer, c.Request.Body, 4096))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(out); err != nil {
		if allowEmpty && errors.Is(err, io.EOF) {
			return true
		}
		h.fail(c, accountlink.ErrInvalid)
		return false
	}
	if !errors.Is(decoder.Decode(new(any)), io.EOF) {
		h.fail(c, accountlink.ErrInvalid)
		return false
	}
	return true
}

func (h *Handler) fail(c *gin.Context, err error) {
	code, key := response.CodeInternal, "error.api_account_link_unavailable"
	switch {
	case errors.Is(err, accountlink.ErrInvalid):
		code, key = response.CodeBadRequest, "error.api_account_link_invalid"
	case errors.Is(err, accountlink.ErrExpired):
		code, key = response.CodeBadRequest, "error.api_account_link_expired"
	case errors.Is(err, accountlink.ErrForbidden):
		code, key = response.CodeForbidden, "error.api_account_link_forbidden"
	case errors.Is(err, accountlink.ErrConflict):
		code, key = response.CodeBadRequest, "error.api_account_link_conflict"
	}
	ginutil.RespondError(c, code, key, nil)
}
