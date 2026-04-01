package mcp

import (
	"context"
	"strings"

	einomcp "github.com/cloudwego/eino-ext/components/tool/mcp"
	"github.com/cloudwego/eino/components/tool"
	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
)

// ToolProvider 负责从远端 MCP 服务拉取工具定义。
type ToolProvider struct {
	url string
}

// NewToolProvider 创建 MCP 工具提供者。
func NewToolProvider(url string) *ToolProvider {
	return &ToolProvider{url: strings.TrimSpace(url)}
}

// Tools 连接 MCP SSE 服务并返回当前可用的工具列表。
func (p *ToolProvider) Tools(ctx context.Context) ([]tool.BaseTool, error) {
	if p.url == "" {
		return []tool.BaseTool{}, nil
	}

	cli, err := client.NewSSEMCPClient(p.url)
	if err != nil {
		return nil, err
	}

	if err := cli.Start(ctx); err != nil {
		return nil, err
	}

	// 初始化握手必须显式声明协议版本和客户端信息，否则服务端可能拒绝请求。
	request := mcp.InitializeRequest{}
	request.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	request.Params.ClientInfo = mcp.Implementation{
		Name:    "opspilot",
		Version: "1.0.0",
	}
	if _, err := cli.Initialize(ctx, request); err != nil {
		return nil, err
	}

	return einomcp.GetTools(ctx, &einomcp.Config{Cli: cli})
}
