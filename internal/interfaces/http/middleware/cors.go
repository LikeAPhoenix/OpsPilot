package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// CORS 为前端跨域访问提供统一响应头。
func CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		headers := c.Writer.Header()
		headers.Set("Access-Control-Allow-Origin", "*")
		headers.Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		headers.Set("Access-Control-Allow-Headers", "Content-Type, Authorization, Accept, Origin, Cache-Control")
		headers.Set("Access-Control-Expose-Headers", "Content-Type, Cache-Control")

		// 预检请求只需要返回允许的跨域头部，不进入后续业务处理链。
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
