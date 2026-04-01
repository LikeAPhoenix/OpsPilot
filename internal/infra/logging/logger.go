package logging

import (
	appconfig "OpsPilot/internal/infra/config"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// NewLogger 根据配置创建 zap 日志器。
func NewLogger(config appconfig.LoggerConfig) (*zap.Logger, error) {
	atomicLevel := zap.NewAtomicLevel()
	if err := atomicLevel.UnmarshalText([]byte(config.Level)); err != nil {
		return nil, err
	}

	encoderConfig := zap.NewProductionEncoderConfig()
	if config.Development {
		encoderConfig = zap.NewDevelopmentEncoderConfig()
	}

	// 默认写入 stderr，便于容器日志采集；显式要求时再切到 stdout。
	outputPaths := []string{"stdout"}
	// 如果配置了日志文件，则将日志写入文件
	if config.LogFile != "" {
		outputPaths = append(outputPaths, config.LogFile)
	}

	zapConfig := zap.Config{
		Level:            atomicLevel,
		Development:      config.Development,
		Encoding:         config.Encoding,
		EncoderConfig:    encoderConfig,
		OutputPaths:      outputPaths,
	}

	return zapConfig.Build(zap.AddStacktrace(zapcore.ErrorLevel))
}

// Sync 将日志缓冲区刷新到底层输出。
func Sync(logger *zap.Logger) {
	if logger == nil {
		return
	}
	_ = logger.Sync()
}
