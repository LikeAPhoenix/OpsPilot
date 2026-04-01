package ark

import (
	"context"
	"fmt"

	appconfig "OpsPilot/internal/infra/config"

	arkemb "github.com/cloudwego/eino-ext/components/embedding/ark"
	arkmodel "github.com/cloudwego/eino-ext/components/model/ark"
	"github.com/cloudwego/eino/components/embedding"
	"github.com/cloudwego/eino/components/model"
)

// Provider 提供火山方舟模型与嵌入模型的创建能力。
type Provider struct{}

// NewProvider 创建方舟模型提供者。
func NewProvider() *Provider {
	return &Provider{}
}

// NewChatModel 根据配置创建方舟聊天模型。
func (p *Provider) NewChatModel(ctx context.Context, cfg appconfig.ModelConfig) (model.ToolCallingChatModel, error) {
	return arkmodel.NewChatModel(ctx, &arkmodel.ChatModelConfig{
		Model:   cfg.Model,
		APIKey:  cfg.APIKey,
		BaseURL: cfg.BaseURL,
	})
}

// NewEmbedder 根据配置创建方舟嵌入模型。
func (p *Provider) NewEmbedder(ctx context.Context, cfg appconfig.EmbeddingConfig) (embedding.Embedder, error) {
	apiType, err := parseAPIType(cfg.APIType)
	if err != nil {
		return nil, err
	}

	return arkemb.NewEmbedder(ctx, &arkemb.EmbeddingConfig{
		Model:   cfg.Model,
		APIKey:  cfg.APIKey,
		BaseURL: cfg.BaseURL,
		APIType: apiType,
	})
}

// parseAPIType 将配置文件中的字符串转换为方舟 SDK 需要的 API 类型。
func parseAPIType(raw string) (*arkemb.APIType, error) {
	switch raw {
	case "text", "text_api":
		apiType := arkemb.APITypeText
		return &apiType, nil
	case "multimodal", "multi_modal", "multi_modal_api":
		apiType := arkemb.APITypeMultiModal
		return &apiType, nil
	default:
		return nil, fmt.Errorf("unsupported ark embedding api_type: %s", raw)
	}
}
