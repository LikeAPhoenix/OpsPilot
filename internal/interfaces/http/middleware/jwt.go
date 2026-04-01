package middleware

import (
	"errors"
	"net/http"
	"strings"

	"OpsPilot/internal/interfaces/http/response"
	pkgauth "OpsPilot/pkg/auth"

	"github.com/gin-gonic/gin"
)

const userIDKey = "userID"

// JWT 创建统一的 Gin JWT 鉴权中间件。
func JWT(manager *pkgauth.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := strings.TrimSpace(c.GetHeader("Authorization"))
		if !strings.HasPrefix(header, "Bearer ") {
			response.Error(c, http.StatusUnauthorized, errors.New("缺少有效的 Authorization Bearer Token"))
			c.Abort()
			return
		}

		claims, err := manager.ParseAccessToken(strings.TrimSpace(strings.TrimPrefix(header, "Bearer ")))
		if err != nil {
			response.Error(c, http.StatusUnauthorized, errors.New("token 无效或已过期"))
			c.Abort()
			return
		}

		c.Set(userIDKey, claims.UserID)
		c.Next()
	}
}

// UserID 从 Gin 上下文中提取鉴权后的 userID。
func UserID(c *gin.Context) string {
	value, _ := c.Get(userIDKey)
	userID, _ := value.(string)
	return userID
}
