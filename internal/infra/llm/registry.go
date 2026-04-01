package llm

import (
	"context"
	"fmt"

	appconfig "OpsPilot/internal/infra/config"

	"github.com/cloudwego/eino/components/embedding"
	"github.com/cloudwego/eino/components/model"
)

const (
	ProviderArk    = "ark"
	ProviderOpenai = "openai"
)

// ChatProvider 定义聊天模型提供者需要实现的工厂方法。
type ChatProvider interface {
	NewChatModel(ctx context.Context, cfg appconfig.ModelConfig) (model.ToolCallingChatModel, error)
}

// EmbeddingProvider 定义嵌入模型提供者需要实现的工厂方法。
type EmbeddingProvider interface {
	NewEmbedder(ctx context.Context, cfg appconfig.EmbeddingConfig) (embedding.Embedder, error)
}

// Registry 维护不同模型提供者的注册表。
type Registry struct {
	chatProviders      map[string]ChatProvider
	embeddingProviders map[string]EmbeddingProvider
}

// NewRegistry 创建空的模型提供者注册表。
func NewRegistry() *Registry {
	return &Registry{
		chatProviders:      map[string]ChatProvider{},
		embeddingProviders: map[string]EmbeddingProvider{},
	}
}

// RegisterChatProvider 注册聊天模型提供者。
func (r *Registry) RegisterChatProvider(name string, provider ChatProvider) {
	r.chatProviders[name] = provider
}

// RegisterEmbeddingProvider 注册嵌入模型提供者。
func (r *Registry) RegisterEmbeddingProvider(name string, provider EmbeddingProvider) {
	r.embeddingProviders[name] = provider
}

// NewChatModel 按配置中的 provider 名称创建聊天模型实例。
func (r *Registry) NewChatModel(ctx context.Context, cfg appconfig.ModelConfig) (model.ToolCallingChatModel, error) {
	name := cfg.Provider
	provider, ok := r.chatProviders[name]
	if !ok {
		return nil, fmt.Errorf("unsupported chat provider: %s", cfg.Provider)
	}
	return provider.NewChatModel(ctx, cfg)
}

// NewEmbedder 按配置中的 provider 名称创建嵌入模型实例。
func (r *Registry) NewEmbedder(ctx context.Context, cfg appconfig.EmbeddingConfig) (embedding.Embedder, error) {
	name := cfg.Provider
	provider, ok := r.embeddingProviders[name]
	if !ok {
		return nil, fmt.Errorf("unsupported embedding provider: %s", cfg.Provider)
	}
	return provider.NewEmbedder(ctx, cfg)
}
