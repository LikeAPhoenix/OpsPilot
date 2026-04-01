package openai

import (
	"context"

	appconfig "OpsPilot/internal/infra/config"

	openaiext "github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/model"
)

// Provider 提供 OpenAI 聊天模型的创建能力。
type Provider struct{}

// NewProvider 创建 OpenAI 模型提供者。
func NewProvider() *Provider {
	return &Provider{}
}

// NewChatModel 根据配置创建 OpenAI 聊天模型。
func (p *Provider) NewChatModel(ctx context.Context, cfg appconfig.ModelConfig) (model.ToolCallingChatModel, error) {
	return openaiext.NewChatModel(ctx, &openaiext.ChatModelConfig{
		Model:   cfg.Model,
		APIKey:  cfg.APIKey,
		BaseURL: cfg.BaseURL,
	})
}
