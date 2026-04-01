package registry

import (
	"net/http"

	"github.com/cloudwego/eino/components/retriever"
	"github.com/cloudwego/eino/components/tool"

	"OpsPilot/internal/ai/tools"
)

// ChatToolsConfig 定义聊天工作流构造工具集所需的依赖。
type ChatToolsConfig struct {
	LogTools             []tool.BaseTool
	Retriever            retriever.Retriever
	PrometheusBaseURL    string
	PrometheusHTTPClient *http.Client
}

// NewChatTools 创建聊天工作流可调用的全部工具。
func NewChatTools(config ChatToolsConfig) ([]tool.BaseTool, error) {
	alertTool, err := tools.NewPrometheusAlertsQueryTool(config.PrometheusBaseURL, config.PrometheusHTTPClient)
	if err != nil {
		return nil, err
	}
	mysqlTool, err := tools.NewMysqlCrudTool()
	if err != nil {
		return nil, err
	}
	currentTimeTool, err := tools.NewGetCurrentTimeTool()
	if err != nil {
		return nil, err
	}
	internalDocsTool, err := tools.NewQueryInternalDocsTool(config.Retriever)
	if err != nil {
		return nil, err
	}

	// 先拷贝外部注入的日志工具，再追加内置工具，避免调用方切片被原地修改。
	result := append([]tool.BaseTool{}, config.LogTools...)
	result = append(result, alertTool, mysqlTool, currentTimeTool, internalDocsTool)
	return result, nil
}
