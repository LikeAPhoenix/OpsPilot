package handler

import (
	"net/http"

	authapp "OpsPilot/internal/application/auth"
	"OpsPilot/internal/interfaces/http/dto"
	"OpsPilot/internal/interfaces/http/response"

	"github.com/gin-gonic/gin"
)

// AuthHandler 负责注册和登录接口。
type AuthHandler struct {
	authService *authapp.Service
}

// NewAuthHandler 创建认证 HTTP 处理器。
func NewAuthHandler(authService *authapp.Service) *AuthHandler {
	return &AuthHandler{authService: authService}
}

// Register 处理用户注册请求。
func (h *AuthHandler) Register(c *gin.Context) {
	var req dto.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, err)
		return
	}

	user, err := h.authService.Register(c.Request.Context(), req.Username, req.Password)
	if err != nil {
		status := http.StatusInternalServerError
		if err == authapp.ErrInvalidCredentials {
			status = http.StatusBadRequest
		}
		if err == authapp.ErrUsernameTaken {
			status = http.StatusConflict
		}
		response.Error(c, status, err)
		return
	}

	response.OK(c, dto.RegisterResponse{
		UserID:   user.ID,
		Username: user.Username,
	})
}

// Login 处理用户登录请求。
func (h *AuthHandler) Login(c *gin.Context) {
	var req dto.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, err)
		return
	}

	result, err := h.authService.Login(c.Request.Context(), req.Username, req.Password)
	if err != nil {
		status := http.StatusInternalServerError
		if err == authapp.ErrInvalidCredentials {
			status = http.StatusUnauthorized
		}
		response.Error(c, status, err)
		return
	}

	response.OK(c, dto.LoginResponse{
		UserID:      result.UserID,
		AccessToken: result.AccessToken,
		ExpiresAt:   result.ExpiresAt.Format("2006-01-02 15:04:05"),
	})
}
