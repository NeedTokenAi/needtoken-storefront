package subscriptionauthorizationhttp

import (
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"strconv"
	"strings"

	"github.com/dujiao-next/internal/i18n"
	subscriptionauth "github.com/dujiao-next/internal/modules/order/application/subscriptionauthorization"
	ordercontract "github.com/dujiao-next/internal/modules/order/contract"
	reseller "github.com/dujiao-next/internal/modules/reseller/contract"
	"github.com/dujiao-next/internal/platform/http/ginutil"
	"github.com/dujiao-next/internal/platform/http/response"
	"github.com/gin-gonic/gin"
)

type Handler struct{ service *subscriptionauth.Service }

func New(service *subscriptionauth.Service) *Handler { return &Handler{service: service} }

func RegisterMemberRoutes(routes gin.IRoutes, h *Handler) {
	routes.GET("/orders/:order_no/subscription-authorization", h.memberEligibility)
	routes.POST("/orders/:order_no/subscription-authorization/session", h.memberOpen)
}
func RegisterGuestReadRoutes(routes gin.IRoutes, h *Handler) {
	routes.GET("/orders/:order_no/subscription-authorization", h.guestEligibility)
}
func RegisterGuestWriteRoutes(routes gin.IRoutes, h *Handler) {
	routes.POST("/orders/:order_no/subscription-authorization/challenge", h.guestChallenge)
	routes.POST("/orders/:order_no/subscription-authorization/session", h.guestOpen)
}
func RegisterConsentRoute(routes gin.IRoutes, h *Handler) {
	routes.GET("/subscription-authorization-consents/:authorization_id", h.consent)
}

func (h *Handler) subject(c *gin.Context, member bool) (subscriptionauth.Subject, bool) {
	tenant := reseller.TenantOrMain(c.Request.Context())
	if tenant.Unavailable {
		h.fail(c, ordercontract.ErrAuthorizationNotFound)
		return subscriptionauth.Subject{}, false
	}
	s := subscriptionauth.Subject{OrderNo: strings.TrimSpace(c.Param("order_no")), ResellerID: tenant.ResellerID}
	if s.OrderNo == "" {
		h.fail(c, ordercontract.ErrAuthorizationInvalid)
		return s, false
	}
	if member {
		var ok bool
		s.UserID, ok = ginutil.GetUserID(c)
		if !ok {
			return s, false
		}
	} else {
		var ok bool
		s.GuestEmail, s.GuestPassword, ok = ginutil.GetGuestCredentials(c)
		if !ok {
			h.fail(c, ordercontract.ErrAuthorizationNotFound)
			return s, false
		}
	}
	return s, true
}
func (h *Handler) memberEligibility(c *gin.Context) { h.eligibility(c, true) }
func (h *Handler) guestEligibility(c *gin.Context)  { h.eligibility(c, false) }
func (h *Handler) eligibility(c *gin.Context, member bool) {
	h.private(c)
	subject, ok := h.subject(c, member)
	if !ok {
		return
	}
	if c.Request.URL.RawQuery != "" {
		h.fail(c, ordercontract.ErrAuthorizationInvalid)
		return
	}
	result, err := h.service.Eligibility(c.Request.Context(), subject)
	if err != nil {
		h.fail(c, err)
		return
	}
	response.Success(c, result)
}
func (h *Handler) guestChallenge(c *gin.Context) {
	h.private(c)
	subject, ok := h.subject(c, false)
	if !ok {
		return
	}
	var input struct {
		OperationID string `json:"operation_id"`
	}
	if !h.decode(c, &input) {
		return
	}
	result, err := h.service.Challenge(c.Request.Context(), subject, input.OperationID, i18n.ResolveLocale(c))
	if err != nil {
		h.fail(c, err)
		return
	}
	response.Success(c, result)
}
func (h *Handler) memberOpen(c *gin.Context) { h.open(c, true) }
func (h *Handler) guestOpen(c *gin.Context)  { h.open(c, false) }
func (h *Handler) open(c *gin.Context, member bool) {
	h.private(c)
	subject, ok := h.subject(c, member)
	if !ok {
		return
	}
	var input subscriptionauth.SessionRequest
	if !h.decode(c, &input) {
		return
	}
	result, err := h.service.Open(c.Request.Context(), subject, input)
	if err != nil {
		h.fail(c, err)
		return
	}
	response.Success(c, result)
}
func (h *Handler) consent(c *gin.Context) {
	h.private(c)
	if c.Request.URL.RawQuery != "" || c.Request.ContentLength > 0 || len(c.Request.TransferEncoding) > 0 {
		h.fail(c, ordercontract.ErrAuthorizationInvalid)
		return
	}
	for _, key := range []string{"Dujiao-Next-Api-Key", "Dujiao-Next-Timestamp", "Dujiao-Next-Signature"} {
		if len(c.Request.Header.Values(key)) != 1 {
			h.fail(c, ordercontract.ErrAuthorizationNotFound)
			return
		}
	}
	stamp, err := strconv.ParseInt(c.GetHeader("Dujiao-Next-Timestamp"), 10, 64)
	if err != nil {
		h.fail(c, ordercontract.ErrAuthorizationNotFound)
		return
	}
	result, err := h.service.Consent(c.Request.Context(), c.Param("authorization_id"), subscriptionauth.Signature{APIKey: c.GetHeader("Dujiao-Next-Api-Key"), Value: c.GetHeader("Dujiao-Next-Signature"), Timestamp: stamp, Path: c.Request.URL.Path})
	if err != nil {
		h.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true, "consent": result})
}
func (h *Handler) private(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	c.Header("Referrer-Policy", "no-referrer")
	c.Header("X-Content-Type-Options", "nosniff")
}
func (h *Handler) decode(c *gin.Context, out any) bool {
	media, _, err := mime.ParseMediaType(c.GetHeader("Content-Type"))
	if err != nil || media != "application/json" || c.Request.URL.RawQuery != "" || c.GetHeader("Content-Encoding") != "" {
		h.fail(c, ordercontract.ErrAuthorizationInvalid)
		return false
	}
	decoder := json.NewDecoder(http.MaxBytesReader(c.Writer, c.Request.Body, 4096))
	decoder.DisallowUnknownFields()
	if decoder.Decode(out) != nil || decoder.Decode(new(any)) != io.EOF {
		h.fail(c, ordercontract.ErrAuthorizationInvalid)
		return false
	}
	return true
}
func (h *Handler) fail(c *gin.Context, err error) {
	code, key := response.CodeInternal, "error.subscription_authorization_unavailable"
	switch {
	case errors.Is(err, ordercontract.ErrAuthorizationNotFound):
		code, key = response.CodeNotFound, "error.order_not_found"
	case errors.Is(err, ordercontract.ErrAuthorizationInvalid):
		code, key = response.CodeBadRequest, "error.subscription_authorization_invalid"
	case errors.Is(err, ordercontract.ErrAuthorizationConflict):
		code, key = response.CodeBadRequest, "error.subscription_authorization_conflict"
	case errors.Is(err, ordercontract.ErrAuthorizationRateLimited):
		code, key = response.CodeTooManyRequests, "error.verify_code_too_frequent"
	}
	ginutil.RespondError(c, code, key, nil)
}
