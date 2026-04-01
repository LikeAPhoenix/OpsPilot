package tools

import (
	"context"
	"encoding/json"
	"time"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
)

// GetCurrentTimeInput 表示获取当前时间工具的输入。
type GetCurrentTimeInput struct{}

// GetCurrentTimeOutput 表示当前时间工具的输出结构。
type GetCurrentTimeOutput struct {
	Success      bool   `json:"success" jsonschema:"description=Indicates whether the time retrieval was successful"`
	Seconds      int64  `json:"seconds" jsonschema:"description=Current Unix timestamp in seconds since epoch (1970-01-01 00:00:00 UTC)"`
	Milliseconds int64  `json:"milliseconds" jsonschema:"description=Current Unix timestamp in milliseconds since epoch (1970-01-01 00:00:00 UTC)"`
	Microseconds int64  `json:"microseconds" jsonschema:"description=Current Unix timestamp in microseconds since epoch (1970-01-01 00:00:00 UTC)"`
	Timestamp    string `json:"timestamp" jsonschema:"description=Human-readable timestamp in format time.RFC3339 (e.g., '2006-01-02T15:04:05Z07:00')"`
	Message      string `json:"message" jsonschema:"description=Status message describing the operation result"`
}

// NewGetCurrentTimeTool 创建当前时间查询工具。
func NewGetCurrentTimeTool() (tool.BaseTool, error) {
	return utils.InferOptionableTool(
		"get_current_time",
		"Get current system time in multiple formats. Returns the current time in seconds (Unix timestamp), milliseconds, and microseconds. Use this tool when you need to retrieve current system time for logging, timing operations, or timestamping events.",
		func(ctx context.Context, input *GetCurrentTimeInput, opts ...tool.Option) (string, error) {
			now := time.Now()
			result := GetCurrentTimeOutput{
				Success:      true,
				Seconds:      now.Unix(),
				Milliseconds: now.UnixMilli(),
				Microseconds: now.UnixMicro(),
				Timestamp:    now.Format(time.RFC3339),
				Message:      "Current time retrieved successfully",
			}

			data, err := json.MarshalIndent(result, "", "  ")
			if err != nil {
				return "", err
			}
			return string(data), nil
		},
	)
}
