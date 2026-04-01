package router

import (
	"OpsPilot/internal/interfaces/http/handler"
	"OpsPilot/internal/interfaces/http/middleware"

	"github.com/gin-gonic/gin"
)

// New 创建 HTTP 路由引擎并注册全部接口。
func New(authHandler *handler.AuthHandler, chatHandler *handler.ChatHandler, authMiddleware gin.HandlerFunc) *gin.Engine {
	engine := gin.New()
	engine.Use(gin.Logger(), gin.Recovery(), middleware.CORS())

	// 当前 HTTP 接口统一挂在 /api 前缀下，便于后续版本化扩展。
	api := engine.Group("/api")
	authGroup := api.Group("/auth")
	authGroup.POST("/register", authHandler.Register)
	authGroup.POST("/login", authHandler.Login)

	protected := api.Group("")
	protected.Use(authMiddleware)
	protected.POST("/chat", chatHandler.Chat)
	protected.POST("/chat_stream", chatHandler.ChatStream)
	protected.POST("/upload", chatHandler.Upload)
	protected.POST("/ai_ops", chatHandler.AIOps)

	return engine
}
