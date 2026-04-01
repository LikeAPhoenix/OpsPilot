package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/cloudwego/eino/components/retriever"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
)

// QueryInternalDocsInput 表示内部知识检索工具的输入参数。
type QueryInternalDocsInput struct {
	Query string `json:"query" jsonschema:"description=The query string to search in internal documentation for relevant information and processing steps"`
}

// NewQueryInternalDocsTool 创建内部文档检索工具。
func NewQueryInternalDocsTool(valueRetriever retriever.Retriever) (tool.BaseTool, error) {
	if valueRetriever == nil {
		return nil, fmt.Errorf("retriever is required")
	}

	return utils.InferOptionableTool(
		"query_internal_docs",
		"Use this tool to search internal documentation and knowledge base for relevant information. It performs RAG (Retrieval-Augmented Generation) to find similar documents and extract processing steps. This is useful when you need to understand internal procedures, best practices, or step-by-step guides stored in the company's documentation.",
		func(ctx context.Context, input *QueryInternalDocsInput, opts ...tool.Option) (string, error) {
			// 这里直接返回检索原始结果，交给上层工作流决定如何组织答案。
			resp, err := valueRetriever.Retrieve(ctx, input.Query)
			if err != nil {
				return "", err
			}
			data, err := json.Marshal(resp)
			if err != nil {
				return "", err
			}
			return string(data), nil
		},
	)
}
