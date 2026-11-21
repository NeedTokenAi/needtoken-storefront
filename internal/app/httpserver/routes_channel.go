package httpserver

import (
	"github.com/dujiao-next/internal/app/container"
	"github.com/dujiao-next/internal/app/httpserver/middleware"
	"github.com/dujiao-next/internal/bootstrap/opsbridge"
	affiliatetransport "github.com/dujiao-next/internal/modules/affiliate/transport/http"
	channeltransport "github.com/dujiao-next/internal/modules/channelapi/transport/http"
	giftcardtransport "github.com/dujiao-next/internal/modules/giftcard/transport/http"
	memberleveltransport "github.com/dujiao-next/internal/modules/memberlevel/transport/http"
	telegramtransport "github.com/dujiao-next/internal/modules/telegram/channelbot/transport/http"
	wallettransport "github.com/dujiao-next/internal/modules/wallet/transport/http"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

func registerChannelRoutes(
	apiV1 *gin.RouterGroup,
	c *container.Container,
	channelHandler *channeltransport.Handler,
	channelMemberLevelHandler *memberleveltransport.ChannelHandler,
	channelGiftCardHandler *giftcardtransport.ChannelHandler,
	channelAffiliateHandler *affiliatetransport.ChannelHandler,
	channelTelegramBotHandler *telegramtransport.ChannelBotHandler,
	channelWalletHandler *wallettransport.ChannelHandler,
	redisClient *redis.Client,
	channelAPIRule middleware.RateLimitRule,
) {
	// 渠道 API（Telegram Bot 等外部服务调用）
	channelAPI := apiV1.Group("/channel")
	channelAPI.Use(middleware.RateLimitMiddleware(redisClient, channelAPIRule, middleware.KeyByIPAndHeader("Dujiao-Next-Channel-Key")))
	channelAPI.Use(middleware.ChannelAPIAuthMiddleware(c))
	{
		opsHandler := opsbridge.NewHandler(c)
		channelAPI.GET("/needtoken-ops/overview", opsHandler.Overview)
		channelAPI.GET("/needtoken-ops/orders", opsHandler.Orders)
		channelAPI.GET("/needtoken-ops/catalog/categories", opsHandler.Categories)
		channelAPI.PATCH("/needtoken-ops/catalog/categories/:id", opsHandler.UpdateCategory)
		channelAPI.GET("/needtoken-ops/catalog/products", opsHandler.Products)
		channelAPI.POST("/needtoken-ops/catalog/products", opsHandler.CreateProduct)
		channelAPI.PATCH("/needtoken-ops/catalog/products/:id", opsHandler.UpdateProduct)
		channelAPI.POST("/needtoken-ops/catalog/products/:id/skus", opsHandler.CreateSKU)
		channelAPI.PATCH("/needtoken-ops/catalog/skus/:id", opsHandler.UpdateSKU)
		channelAPI.GET("/needtoken-ops/coupons", opsHandler.Coupons)
		channelAPI.POST("/needtoken-ops/coupons", opsHandler.CreateCoupon)
		channelAPI.PATCH("/needtoken-ops/coupons/:id/status", opsHandler.UpdateCouponStatus)
		channelAPI.GET("/needtoken-ops/pricing", opsHandler.Pricing)
		channelAPI.PATCH("/needtoken-ops/pricing", opsHandler.UpdatePricing)
		telegramtransport.RegisterChannelBotRoutes(channelAPI, channelTelegramBotHandler)
		channeltransport.RegisterRoutes(channelAPI, channelHandler)
		affiliatetransport.RegisterChannelRoutes(channelAPI, channelAffiliateHandler)

		// Catalog 端点（商品浏览）
		memberleveltransport.RegisterChannelRoutes(channelAPI, channelMemberLevelHandler)

		// Order / Payment 端点（购买流程）

		// Wallet 端点（钱包）
		wallettransport.RegisterChannelRoutes(channelAPI, channelWalletHandler)
		giftcardtransport.RegisterChannelRoutes(channelAPI, channelGiftCardHandler)
	}
}
