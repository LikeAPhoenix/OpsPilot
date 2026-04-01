package registry

import (
	"net/http"

	"github.com/cloudwego/eino/components/retriever"
	"github.com/cloudwego/eino/components/tool"

	"OpsPilot/internal/ai/tools"
)

// AIOpsToolsConfig 定义 AIOps 工作流构造工具集所需的依赖。
type AIOpsToolsConfig struct {
	LogTools             []tool.BaseTool
	Retriever            retriever.Retriever
	PrometheusBaseURL    string
	PrometheusHTTPClient *http.Client
}

// NewAIOpsTools 创建 AIOps 工作流可调用的全部工具。
func NewAIOpsTools(config AIOpsToolsConfig) ([]tool.BaseTool, error) {
	alertTool, err := tools.NewPrometheusAlertsQueryTool(config.PrometheusBaseURL, config.PrometheusHTTPClient)
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

	// 先拷贝外部注入的日志工具，再追加 AIOps 需要的内置工具。
	result := append([]tool.BaseTool{}, config.LogTools...)
	result = append(result, alertTool, internalDocsTool, currentTimeTool)
	return result, nil
}
