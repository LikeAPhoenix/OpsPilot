package bootstrap

import (
	"context"

	"github.com/cloudwego/eino-ext/components/document/loader/file"
	"github.com/cloudwego/eino-ext/components/document/transformer/splitter/markdown"
	"github.com/cloudwego/eino/components/document"
	"github.com/google/uuid"
)

// newFileLoader 创建本地文件加载器，用于读取待入库的知识文档。
func newFileLoader(ctx context.Context) (document.Loader, error) {
	return file.NewFileLoader(ctx, &file.FileLoaderConfig{})
}

// newMarkdownSplitter 按 Markdown 标题切分文档，为后续索引生成稳定的片段结构。
func newMarkdownSplitter(ctx context.Context) (document.Transformer, error) {
	return markdown.NewHeaderSplitter(ctx, &markdown.HeaderConfig{
		Headers: map[string]string{
			"#": "title",
			"##": "section",
		},
		TrimHeaders: false,
		IDGenerator: func(ctx context.Context, originalID string, splitIndex int) string {
			// 片段切分后不复用原始 ID，避免多次重建索引时发生冲突。
			return uuid.New().String()
		},
	})
}
