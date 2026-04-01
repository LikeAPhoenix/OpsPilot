package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

// Config 汇总服务启动所需的全部配置项。
type Config struct {
	Server         ServerConfig     `yaml:"server" mapstructure:"server"`
	Logger         LoggerConfig     `yaml:"logger" mapstructure:"logger"`
	ThinkChatModel ModelConfig      `yaml:"think_chat_model" mapstructure:"think_chat_model"`
	QuickChatModel ModelConfig      `yaml:"quick_chat_model" mapstructure:"quick_chat_model"`
	EmbeddingModel EmbeddingConfig  `yaml:"embedding_model" mapstructure:"embedding_model"`
	FileDir        string           `yaml:"file_dir" mapstructure:"file_dir"`
	MCP            MCPConfig        `yaml:"mcp" mapstructure:"mcp"`
	MCPURL         string           `yaml:"mcp_url" mapstructure:"mcp_url"`
	Milvus         MilvusConfig     `yaml:"milvus" mapstructure:"milvus"`
	Prometheus     PrometheusConfig `yaml:"prometheus" mapstructure:"prometheus"`
	MySQL          MySQLConfig      `yaml:"mysql" mapstructure:"mysql"`
	Redis          RedisConfig      `yaml:"redis" mapstructure:"redis"`
	Auth           AuthConfig       `yaml:"auth" mapstructure:"auth"`
	Session        SessionConfig    `yaml:"session" mapstructure:"session"`
	Snowflake      SnowflakeConfig  `yaml:"snowflake" mapstructure:"snowflake"`
}

// ServerConfig 定义 HTTP 服务监听配置。
type ServerConfig struct {
	Address string `yaml:"address" mapstructure:"address"`
}

// LoggerConfig 定义日志输出格式和级别。
type LoggerConfig struct {
	Level       string `yaml:"level" mapstructure:"level"`
	Encoding    string `yaml:"encoding" mapstructure:"encoding"`
	Development bool   `yaml:"development" mapstructure:"development"`
	LogFile     string `yaml:"log_file" mapstructure:"log_file"`
}

// ModelConfig 描述聊天模型接入配置。
type ModelConfig struct {
	Provider string `yaml:"provider" mapstructure:"provider"`
	APIKey   string `yaml:"api_key" mapstructure:"api_key"`
	BaseURL  string `yaml:"base_url" mapstructure:"base_url"`
	Model    string `yaml:"model" mapstructure:"model"`
}

// EmbeddingConfig 描述嵌入模型接入配置。
type EmbeddingConfig struct {
	Provider   string `yaml:"provider" mapstructure:"provider"`
	APIKey     string `yaml:"api_key" mapstructure:"api_key"`
	BaseURL    string `yaml:"base_url" mapstructure:"base_url"`
	Model      string `yaml:"model" mapstructure:"model"`
	APIType    string `yaml:"api_type" mapstructure:"api_type"`
	Dimensions int    `yaml:"dimensions" mapstructure:"dimensions"`
}

// MCPConfig 定义 MCP 服务地址。
type MCPConfig struct {
	URL string `yaml:"url" mapstructure:"url"`
}

// MilvusConfig 定义向量库连接配置。
type MilvusConfig struct {
	Address    string `yaml:"address" mapstructure:"address"`
	DefaultDB  string `yaml:"default_db" mapstructure:"default_db"`
	Database   string `yaml:"database" mapstructure:"database"`
	Collection string `yaml:"collection" mapstructure:"collection"`
}

// PrometheusConfig 定义 Prometheus HTTP API 地址。
type PrometheusConfig struct {
	BaseURL string `yaml:"base_url" mapstructure:"base_url"`
}

// MySQLConfig 定义用户账号库连接配置。
type MySQLConfig struct {
	DSN string `yaml:"dsn" mapstructure:"dsn"`
}

// RedisConfig 定义短期记忆存储配置。
type RedisConfig struct {
	Address  string `yaml:"address" mapstructure:"address"`
	Password string `yaml:"password" mapstructure:"password"`
	DB       int    `yaml:"db" mapstructure:"db"`
}

// AuthConfig 定义 JWT access token 配置。
type AuthConfig struct {
	JWTSecret      string `yaml:"jwt_secret" mapstructure:"jwt_secret"`
	AccessTokenTTL string `yaml:"access_token_ttl" mapstructure:"access_token_ttl"`
}

// SessionConfig 定义短期记忆策略。
type SessionConfig struct {
	RetainTurns           int    `yaml:"retain_turns" mapstructure:"retain_turns"`
	CompactThresholdTurns int    `yaml:"compact_threshold_turns" mapstructure:"compact_threshold_turns"`
	TTL                   string `yaml:"ttl" mapstructure:"ttl"`
}

// SnowflakeConfig 定义 ID 生成器节点编号。
type SnowflakeConfig struct {
	NodeID int64 `yaml:"node_id" mapstructure:"node_id"`
}

// Load 从配置文件和环境变量加载服务配置。
func Load(path string) (*Config, error) {
	if path == "" {
		path = "configs/config.yaml"
	}
	v := viper.New()
	v.SetConfigFile(path)
	v.SetConfigType("yaml")
	v.SetEnvPrefix("OPSPILOT")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	// 允许使用环境变量覆盖配置文件中的字段，便于容器化部署。
	v.AutomaticEnv()

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("read config %s: %w", path, err)
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("parse config %s: %w", path, err)
	}

	return &cfg, nil
}
