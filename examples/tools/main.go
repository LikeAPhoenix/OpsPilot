package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"OpsPilot/internal/bootstrap"

	"github.com/cloudwego/eino/components/tool"
	"go.uber.org/zap"
)

// main 演示打印聊天和 AIOps 场景下注册的全部工具定义。
func main() {
	ctx := context.Background()

	app, err := bootstrap.New(ctx, "")
	if err != nil {
		log.Fatalf("bootstrap app: %v", err)
	}
	defer app.Close()
	logger := app.Logger.Named("examples.tools")

	seen := make(map[string]struct{})
	for _, currentTool := range mergedTools(app.ChatTools, app.AIOpsTools) {
		info, err := currentTool.Info(ctx)
		if err != nil {
			logger.Fatal("load tool info failed", zap.Error(err))
		}

		data, err := json.MarshalIndent(info, "", "  ")
		if err != nil {
			logger.Fatal("marshal tool info failed", zap.Error(err))
		}

		key := string(data)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}

		fmt.Println(string(data))
	}
}

// mergedTools 合并多个工具切片，保持原有顺序输出。
func mergedTools(groups ...[]tool.BaseTool) []tool.BaseTool {
	var result []tool.BaseTool
	for _, group := range groups {
		for _, currentTool := range group {
			result = append(result, currentTool)
		}
	}
	return result
}
