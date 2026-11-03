package http

import (
	"net/http"
	"strconv"

	"github.com/dujiao-next/internal/i18n"
	opscontract "github.com/dujiao-next/internal/modules/opsbridge/contract"
	"github.com/dujiao-next/internal/platform/http/response"
	"github.com/gin-gonic/gin"
)

type Handler struct{ reader opscontract.Reader }

func New(reader opscontract.Reader) *Handler { return &Handler{reader: reader} }
func (h *Handler) Overview(c *gin.Context) {
	result, err := h.reader.Overview()
	if err != nil {
		fail(c)
		return
	}
	response.Success(c, result)
}
func (h *Handler) Orders(c *gin.Context) {
	page, _ := strconv.Atoi(c.Query("page"))
	size, _ := strconv.Atoi(c.Query("page_size"))
	result, err := h.reader.Orders(page, size)
	if err != nil {
		fail(c)
		return
	}
	response.Success(c, result)
}
func fail(c *gin.Context) {
	response.ErrorWithHTTPStatus(c, http.StatusServiceUnavailable, response.CodeInternal, i18n.T(i18n.ResolveLocale(c), "error.internal_error"))
}
