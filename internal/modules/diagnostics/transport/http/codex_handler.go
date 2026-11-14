package diagnosticshttp

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/dujiao-next/internal/i18n"
	diagnosticscontract "github.com/dujiao-next/internal/modules/diagnostics/contract"
	"github.com/dujiao-next/internal/platform/http/response"

	"github.com/gin-gonic/gin"
)

const maxRequestSize = 24 * 1024

type Handler struct {
	inspector diagnosticscontract.Inspector
}

func NewHandler(inspector diagnosticscontract.Inspector) *Handler {
	return &Handler{inspector: inspector}
}

// NoStore also applies to validation and rate-limit responses. Credentials are
// accepted only in the request body, never in query strings or log fields.
func NoStore() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Cache-Control", "private, no-store, max-age=0")
		c.Header("Pragma", "no-cache")
		c.Next()
	}
}

func RegisterPublicRoutes(group gin.IRoutes, handler *Handler) {
	group.POST("/codex-account", handler.CheckCodexAccount)
}

func (h *Handler) CheckCodexAccount(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxRequestSize)
	decoder := json.NewDecoder(c.Request.Body)
	decoder.DisallowUnknownFields()
	var input diagnosticscontract.Input
	// Do not pass decoder/upstream errors to RespondBindError, c.Error or logs:
	// these errors can include caller-controlled data and credentials.
	if decoder.Decode(&input) != nil {
		h.respondError(c, diagnosticscontract.ErrInvalidInput)
		return
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		h.respondError(c, diagnosticscontract.ErrInvalidInput)
		return
	}
	if h.inspector == nil {
		h.respondError(c, diagnosticscontract.ErrUnavailable)
		return
	}
	result, err := h.inspector.Inspect(c.Request.Context(), input)
	input.AccessToken = ""
	if err != nil {
		h.respondError(c, err)
		return
	}
	response.Success(c, result)
}

func (h *Handler) respondError(c *gin.Context, err error) {
	code, key := http.StatusBadGateway, "unavailable"
	switch {
	case errors.Is(err, diagnosticscontract.ErrInvalidInput):
		code, key = response.CodeBadRequest, "invalid_input"
	case errors.Is(err, diagnosticscontract.ErrUnauthorized):
		code, key = response.CodeUnauthorized, "unauthorized"
	case errors.Is(err, diagnosticscontract.ErrChallenge):
		code, key = response.CodeForbidden, "challenge"
	case errors.Is(err, diagnosticscontract.ErrInvalidResponse):
		code, key = http.StatusBadGateway, "invalid_response"
	case errors.Is(err, diagnosticscontract.ErrAccountNotFound):
		code, key = response.CodeBadRequest, "account_not_found"
	case errors.Is(err, diagnosticscontract.ErrUnavailable):
		code, key = http.StatusBadGateway, "unavailable"
	}
	c.JSON(http.StatusOK, response.Response{
		StatusCode: code,
		Msg:        i18n.T(i18n.ResolveLocale(c), "error.codex_check_"+key),
		Data:       gin.H{"error_code": key},
	})
}
