package logging

import (
	"context"
	"encoding/json"

	"github.com/cloudwego/eino/callbacks"
	"go.uber.org/zap"
)

// CallbackConfig 控制工作流回调日志的详细程度。
type CallbackConfig struct {
	Detail bool
	Debug  bool
}

// NewCallback 创建 Eino 工作流回调处理器，用于输出链路执行日志。
func NewCallback(logger *zap.Logger, config *CallbackConfig) callbacks.Handler {
	if config == nil {
		config = &CallbackConfig{Detail: true}
	}
	if logger == nil {
		logger = zap.NewNop()
	}
	logger = logger.Named("eino")

	builder := callbacks.NewHandlerBuilder()
	builder.OnStartFn(func(ctx context.Context, info *callbacks.RunInfo, input callbacks.CallbackInput) context.Context {
		fields := []zap.Field{
			zap.String("component", string(info.Component)),
			zap.String("type", info.Type),
			zap.String("name", info.Name),
		}
		if config.Detail {
			// 输入内容按需序列化，避免在默认场景输出过于冗长的结构化数据。
			var data []byte
			if config.Debug {
				data, _ = json.MarshalIndent(input, "", "  ")
			} else {
				data, _ = json.Marshal(input)
			}
			fields = append(fields, zap.ByteString("input", data))
		}
		logger.Debug("workflow start", fields...)
		return ctx
	})
	builder.OnEndFn(func(ctx context.Context, info *callbacks.RunInfo, output callbacks.CallbackOutput) context.Context {
		// 结束日志只保留基础元数据，避免把大体积输出重复打印到调试日志里。
		logger.Debug("workflow end",
			zap.String("component", string(info.Component)),
			zap.String("type", info.Type),
			zap.String("name", info.Name),
		)
		return ctx
	})
	return builder.Build()
}
