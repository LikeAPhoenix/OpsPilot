package dto

// ChatRequest 表示普通聊天接口的请求体。
type ChatRequest struct {
	ID       string `json:"Id"`
	Question string `json:"Question" binding:"required"`
}

// ChatResponse 表示普通聊天接口的响应体。
type ChatResponse struct {
	Answer    string `json:"answer"`
	SessionID string `json:"sessionId"`
}

// ChatStreamRequest 表示流式聊天接口的请求体。
type ChatStreamRequest struct {
	ID       string `json:"Id"`
	Question string `json:"Question" binding:"required"`
}

// RegisterRequest 表示注册接口请求体。
type RegisterRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// RegisterResponse 表示注册接口响应体。
type RegisterResponse struct {
	UserID   string `json:"userId"`
	Username string `json:"username"`
}

// LoginRequest 表示登录接口请求体。
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// LoginResponse 表示登录接口响应体。
type LoginResponse struct {
	UserID      string `json:"userId"`
	AccessToken string `json:"accessToken"`
	ExpiresAt   string `json:"expiresAt"`
}

// FileUploadResponse 表示上传文档后的落盘结果。
type FileUploadResponse struct {
	FileName string `json:"fileName"`
	FilePath string `json:"filePath"`
	FileSize int64  `json:"fileSize"`
}

// AIOpsResponse 表示 AIOps 分析接口的响应体。
type AIOpsResponse struct {
	Result string   `json:"result"`
	Detail []string `json:"detail"`
}
