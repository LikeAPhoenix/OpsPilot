package handler

import (
	"context"
	"errors"
	"net/http"

	aiopsapp "OpsPilot/internal/application/aiops"
	chatapp "OpsPilot/internal/application/chat"
	knowledgeapp "OpsPilot/internal/application/knowledge"
	"OpsPilot/internal/interfaces/http/dto"
	"OpsPilot/internal/interfaces/http/middleware"
	"OpsPilot/internal/interfaces/http/response"
	sseservice "OpsPilot/internal/interfaces/http/sse"

	"github.com/gin-gonic/gin"
)

// ChatHandler 聚合聊天、知识库和 AIOps 相关 HTTP 入口。
type ChatHandler struct {
	chatService      *chatapp.Service
	knowledgeService *knowledgeapp.Service
	aiopsService     *aiopsapp.Service
}

// NewChatHandler 创建 HTTP 处理器。
func NewChatHandler(chatService *chatapp.Service, knowledgeService *knowledgeapp.Service, aiopsService *aiopsapp.Service) *ChatHandler {
	return &ChatHandler{
		chatService:      chatService,
		knowledgeService: knowledgeService,
		aiopsService:     aiopsService,
	}
}

// Chat 处理普通问答请求，并返回一次性完整回答。
func (h *ChatHandler) Chat(c *gin.Context) {
	var req dto.ChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, err)
		return
	}

	// 服务层负责拼装历史上下文，这里只做请求解析和响应映射。
	res, err := h.chatService.Chat(c.Request.Context(), middleware.UserID(c), req.ID, req.Question)
	if err != nil {
		response.Error(c, statusFromError(err), err)
		return
	}

	response.OK(c, dto.ChatResponse{
		Answer:    res.Answer,
		SessionID: res.SessionID,
	})
}

// ChatStream 处理流式问答请求，通过 SSE 持续推送模型输出。
func (h *ChatHandler) ChatStream(c *gin.Context) {
	var req dto.ChatStreamRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, err)
		return
	}

	client, err := sseservice.NewClient(c.Writer, req.ID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err)
		return
	}

	// 客户端主动断开连接时会触发 context.Canceled，这类情况不再重复记为服务错误。
	if err := h.chatService.Stream(c.Request.Context(), middleware.UserID(c), req.ID, req.Question, client); err != nil && !errors.Is(err, context.Canceled) {
		_ = c.Error(err)
	}
}

// Upload 接收知识文档上传，并立即触发索引更新。
func (h *ChatHandler) Upload(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		response.Error(c, http.StatusBadRequest, errors.New("请上传文件"))
		return
	}

	res, err := h.knowledgeService.UploadFile(c.Request.Context(), file)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err)
		return
	}

	response.OK(c, dto.FileUploadResponse{
		FileName: res.FileName,
		FilePath: res.FilePath,
		FileSize: res.FileSize,
	})
}

// AIOps 触发预置的告警分析流程并返回结果。
func (h *ChatHandler) AIOps(c *gin.Context) {
	res, err := h.aiopsService.Analyze(c.Request.Context())
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err)
		return
	}

	response.OK(c, dto.AIOpsResponse{
		Result: res.Result,
		Detail: res.Detail,
	})
}

func statusFromError(err error) int {
	switch {
	case errors.Is(err, chatapp.ErrUnauthorized):
		return http.StatusUnauthorized
	case errors.Is(err, chatapp.ErrForbidden):
		return http.StatusForbidden
	case errors.Is(err, chatapp.ErrConflict):
		return http.StatusConflict
	case errors.Is(err, chatapp.ErrNotFound):
		return http.StatusNotFound
	default:
		return http.StatusInternalServerError
	}
}
