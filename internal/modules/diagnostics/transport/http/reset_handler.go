package diagnosticshttp

import (
	"net/http"

	"github.com/dujiao-next/internal/i18n"
	diagnosticscontract "github.com/dujiao-next/internal/modules/diagnostics/contract"
	"github.com/dujiao-next/internal/platform/http/response"

	"github.com/gin-gonic/gin"
)

type ResetHandler struct {
	feed diagnosticscontract.ResetFeedReader
}

func NewResetHandler(feed diagnosticscontract.ResetFeedReader) *ResetHandler {
	return &ResetHandler{feed: feed}
}

func RegisterResetRoutes(group gin.IRoutes, handler *ResetHandler) {
	group.GET("/codex-resets", handler.GetCodexResets)
}

func (h *ResetHandler) GetCodexResets(c *gin.Context) {
	if h.feed != nil {
		if snapshot, err := h.feed.Latest(c.Request.Context()); err == nil {
			response.Success(c, snapshot)
			return
		}
	}
	c.JSON(http.StatusOK, response.Response{StatusCode: http.StatusServiceUnavailable, Msg: i18n.T(i18n.ResolveLocale(c), "error.codex_resets_unavailable"), Data: gin.H{"error_code": "unavailable"}})
}
