package diagnosticshttp

import (
	"net/http"

	"github.com/dujiao-next/internal/i18n"
	diagnosticscontract "github.com/dujiao-next/internal/modules/diagnostics/contract"
	"github.com/dujiao-next/internal/platform/http/response"

	"github.com/gin-gonic/gin"
)

type ResetHandler struct {
	feed        diagnosticscontract.ResetFeedReader
	channelFeed diagnosticscontract.ChannelResetFeedReader
}

func NewResetHandler(feed diagnosticscontract.ResetFeedReader) *ResetHandler {
	ch, _ := feed.(diagnosticscontract.ChannelResetFeedReader)
	return &ResetHandler{feed: feed, channelFeed: ch}
}

func RegisterResetRoutes(group gin.IRoutes, handler *ResetHandler) {
	group.GET("/codex-resets", handler.GetCodexResets)
}

// RegisterChannelResetRoutes mounts public multi-channel radar endpoints that
// emit CDN-friendly public Cache-Control headers so Cloudflare absorbs bursts
// without repeatedly waking the origin server.
func RegisterChannelResetRoutes(group gin.IRoutes, handler *ResetHandler) {
	group.GET("/resets/:channel", handler.GetChannelResets)
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

func (h *ResetHandler) GetChannelResets(c *gin.Context) {
	channel := c.Param("channel")
	cacheControl := ""
	switch channel {
	case "codex", "claude", "grok":
		cacheControl = "public, max-age=180, s-maxage=300, stale-while-revalidate=600"
	case "gemini":
		cacheControl = "public, max-age=300, s-maxage=600, stale-while-revalidate=1200"
	default:
		c.Header("Cache-Control", "no-store")
		c.JSON(http.StatusBadRequest, response.Response{StatusCode: http.StatusBadRequest, Msg: "invalid radar channel", Data: gin.H{"error_code": "invalid_channel"}})
		return
	}
	var (
		snapshot diagnosticscontract.ResetFeedSnapshot
		err      error
	)
	if h.channelFeed != nil {
		snapshot, err = h.channelFeed.LatestForChannel(c.Request.Context(), channel)
	} else if channel == "codex" && h.feed != nil {
		snapshot, err = h.feed.Latest(c.Request.Context())
	} else {
		err = diagnosticscontract.ErrResetFeedUnavailable
	}
	if err == nil {
		c.Header("Cache-Control", cacheControl)
		response.Success(c, snapshot)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, response.Response{StatusCode: http.StatusServiceUnavailable, Msg: i18n.T(i18n.ResolveLocale(c), "error.codex_resets_unavailable"), Data: gin.H{"error_code": "unavailable"}})
}
