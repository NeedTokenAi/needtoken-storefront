package captchahttp

import "github.com/gin-gonic/gin"

// RegisterPublicRoutes 注册公开验证码路由（支持附加限流中间件）。
func RegisterPublicRoutes(public gin.IRoutes, handler *PublicHandler, middlewares ...gin.HandlerFunc) {
	handlers := make([]gin.HandlerFunc, 0, len(middlewares)+1)
	for _, mw := range middlewares {
		if mw != nil {
			handlers = append(handlers, mw)
		}
	}
	handlers = append(handlers, handler.GetImageCaptcha)
	public.GET("/captcha/image", handlers...)
}
