package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Body 定义 HTTP 接口统一响应结构。
type Body struct {
	Message string `json:"message"`
	Data    any    `json:"data"`
}

// OK 返回统一的成功响应。
func OK(c *gin.Context, data any) {
	c.JSON(http.StatusOK, Body{
		Message: "OK",
		Data:    data,
	})
}

// Error 返回统一的错误响应。
func Error(c *gin.Context, status int, err error) {
	c.JSON(status, Body{
		Message: err.Error(),
		Data:    nil,
	})
}
