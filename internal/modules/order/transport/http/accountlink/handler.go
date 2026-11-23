package accountlinkhttp

import (
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"net/url"
	"strconv"
	"strings"

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
	routes.GET("/me/gateway/tokens", h.gatewayListTokens)
	routes.GET("/me/gateway/logs", h.gatewayListLogs)
	routes.GET("/me/gateway/usage", h.gatewayUsage)
	routes.GET("/me/gateway/models", h.gatewayModels)
	routes.GET("/me/gateway/tasks", h.gatewayListTasks)
	routes.GET("/me/gateway/drawings", h.gatewayListDrawings)
	routes.GET("/me/gateway/subscriptions", h.gatewaySubscriptions)
}

func RegisterMemberWriteRoutes(routes gin.IRoutes, h *Handler) {
	routes.POST("/me/api-account-link/start", h.start)
	routes.POST("/me/api-account-link/complete", h.complete)
	routes.POST("/me/api-account-link/ensure", h.ensure)
	routes.POST("/me/api-console-sso", h.consoleSSO)
	routes.POST("/me/logout", h.logout)
	routes.POST("/me/gateway/tokens", h.gatewayCreateToken)
	routes.PUT("/me/gateway/tokens/:token_id", h.gatewayUpdateToken)
	routes.DELETE("/me/gateway/tokens/:token_id", h.gatewayDeleteToken)
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

func (h *Handler) gatewayConsoleUser(c *gin.Context) (uint, bool) {
	h.privateHeaders(c)
	tenant := reseller.TenantOrMain(c.Request.Context())
	if tenant.Unavailable || tenant.ResellerID != nil {
		h.fail(c, accountlink.ErrForbidden)
		return 0, false
	}
	userID, ok := ginutil.GetUserID(c)
	if !ok || userID == 0 {
		h.fail(c, accountlink.ErrForbidden)
		return 0, false
	}
	return userID, true
}

func (h *Handler) readGatewayJSONBody(c *gin.Context) ([]byte, bool) {
	if c.GetHeader("Content-Encoding") != "" {
		h.fail(c, accountlink.ErrInvalid)
		return nil, false
	}
	media, _, err := mime.ParseMediaType(c.GetHeader("Content-Type"))
	if err != nil || media != "application/json" {
		h.fail(c, accountlink.ErrInvalid)
		return nil, false
	}
	raw, err := io.ReadAll(io.LimitReader(c.Request.Body, 16<<10+1))
	if err != nil || len(raw) == 0 || len(raw) > 16<<10 || !json.Valid(raw) {
		h.fail(c, accountlink.ErrInvalid)
		return nil, false
	}
	return raw, true
}

func (h *Handler) respondGatewayData(c *gin.Context, data json.RawMessage, err error) {
	if err != nil {
		h.fail(c, err)
		return
	}
	var payload any
	if err := json.Unmarshal(data, &payload); err != nil {
		h.fail(c, accountlink.ErrUnavailable)
		return
	}
	response.Success(c, payload)
}

func (h *Handler) gatewayListTokens(c *gin.Context) {
	userID, ok := h.gatewayConsoleUser(c)
	if !ok {
		return
	}
	data, err := h.service.GatewayConsole(c.Request.Context(), userID, http.MethodGet, "/tokens", nil, nil)
	h.respondGatewayData(c, data, err)
}

func (h *Handler) gatewayCreateToken(c *gin.Context) {
	userID, ok := h.gatewayConsoleUser(c)
	if !ok {
		return
	}
	raw, ok := h.readGatewayJSONBody(c)
	if !ok {
		return
	}
	data, err := h.service.GatewayConsole(c.Request.Context(), userID, http.MethodPost, "/tokens", nil, raw)
	h.respondGatewayData(c, data, err)
}

func (h *Handler) gatewayUpdateToken(c *gin.Context) {
	userID, ok := h.gatewayConsoleUser(c)
	if !ok {
		return
	}
	tokenID, err := strconv.Atoi(strings.TrimSpace(c.Param("token_id")))
	if err != nil || tokenID <= 0 {
		h.fail(c, accountlink.ErrInvalid)
		return
	}
	raw, ok := h.readGatewayJSONBody(c)
	if !ok {
		return
	}
	data, callErr := h.service.GatewayConsole(c.Request.Context(), userID, http.MethodPut, "/tokens/"+strconv.Itoa(tokenID), nil, raw)
	h.respondGatewayData(c, data, callErr)
}

func (h *Handler) gatewayDeleteToken(c *gin.Context) {
	userID, ok := h.gatewayConsoleUser(c)
	if !ok {
		return
	}
	tokenID, err := strconv.Atoi(strings.TrimSpace(c.Param("token_id")))
	if err != nil || tokenID <= 0 {
		h.fail(c, accountlink.ErrInvalid)
		return
	}
	data, callErr := h.service.GatewayConsole(c.Request.Context(), userID, http.MethodDelete, "/tokens/"+strconv.Itoa(tokenID), nil, nil)
	h.respondGatewayData(c, data, callErr)
}

func (h *Handler) gatewayListLogs(c *gin.Context) {
	userID, ok := h.gatewayConsoleUser(c)
	if !ok {
		return
	}
	q := url.Values{}
	for _, key := range []string{"p", "page_size", "model_name", "token_name", "group", "type", "start_timestamp", "end_timestamp"} {
		if val := strings.TrimSpace(c.Query(key)); val != "" {
			q.Set(key, val)
		}
	}
	data, err := h.service.GatewayConsole(c.Request.Context(), userID, http.MethodGet, "/logs", q, nil)
	h.respondGatewayData(c, data, err)
}

func (h *Handler) gatewayUsage(c *gin.Context) {
	userID, ok := h.gatewayConsoleUser(c)
	if !ok {
		return
	}
	q := url.Values{}
	for _, key := range []string{"days", "start_timestamp", "end_timestamp"} {
		if val := strings.TrimSpace(c.Query(key)); val != "" {
			q.Set(key, val)
		}
	}
	data, err := h.service.GatewayConsole(c.Request.Context(), userID, http.MethodGet, "/usage", q, nil)
	h.respondGatewayData(c, data, err)
}

func (h *Handler) gatewayModels(c *gin.Context) {
	userID, ok := h.gatewayConsoleUser(c)
	if !ok {
		return
	}
	data, err := h.service.GatewayConsole(c.Request.Context(), userID, http.MethodGet, "/models", nil, nil)
	h.respondGatewayData(c, data, err)
}

func (h *Handler) gatewayListTasks(c *gin.Context) {
	userID, ok := h.gatewayConsoleUser(c)
	if !ok {
		return
	}
	q := url.Values{}
	for _, key := range []string{"p", "page_size", "platform", "task_id", "status", "action", "start_timestamp", "end_timestamp"} {
		if val := strings.TrimSpace(c.Query(key)); val != "" {
			q.Set(key, val)
		}
	}
	data, err := h.service.GatewayConsole(c.Request.Context(), userID, http.MethodGet, "/tasks", q, nil)
	h.respondGatewayData(c, data, err)
}

func (h *Handler) gatewayListDrawings(c *gin.Context) {
	userID, ok := h.gatewayConsoleUser(c)
	if !ok {
		return
	}
	q := url.Values{}
	for _, key := range []string{"p", "page_size", "mj_id", "start_timestamp", "end_timestamp"} {
		if val := strings.TrimSpace(c.Query(key)); val != "" {
			q.Set(key, val)
		}
	}
	data, err := h.service.GatewayConsole(c.Request.Context(), userID, http.MethodGet, "/drawings", q, nil)
	h.respondGatewayData(c, data, err)
}

func (h *Handler) gatewaySubscriptions(c *gin.Context) {
	userID, ok := h.gatewayConsoleUser(c)
	if !ok {
		return
	}
	data, err := h.service.GatewayConsole(c.Request.Context(), userID, http.MethodGet, "/subscriptions", nil, nil)
	h.respondGatewayData(c, data, err)
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
	case errors.Is(err, accountlink.ErrNotLinked):
		code, key = response.CodeNotFound, "error.api_account_link_expired"
	}
	ginutil.RespondError(c, code, key, nil)
}
