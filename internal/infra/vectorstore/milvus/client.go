package milvus

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/components/embedding"
	cli "github.com/milvus-io/milvus-sdk-go/v2/client"
	"github.com/milvus-io/milvus-sdk-go/v2/entity"
)

// Config 定义 Milvus 连接与集合信息。
type Config struct {
	Address    string
	DefaultDB  string
	Database   string
	Collection string
}

// Store 封装 Milvus 客户端以及嵌入模型依赖。
type Store struct {
	client   cli.Client
	config   Config
	embedder embedding.Embedder
}

// NewStore 创建 Milvus 存储，并确保数据库和集合存在。
func NewStore(ctx context.Context, config Config, embedder embedding.Embedder) (*Store, error) {
	if err := validateConfig(config); err != nil {
		return nil, err
	}

	// 先连接默认数据库，用于检查并创建业务数据库。
	defaultClient, err := cli.NewClient(ctx, cli.Config{
		Address: config.Address,
		DBName:  config.DefaultDB,
	})
	if err != nil {
		return nil, fmt.Errorf("connect default milvus database: %w", err)
	}
	defer defaultClient.Close()

	if err := ensureDatabase(ctx, defaultClient, config.Database); err != nil {
		return nil, err
	}

	// 再连接业务数据库，后续检索和索引都基于这个连接。
	agentClient, err := cli.NewClient(ctx, cli.Config{
		Address: config.Address,
		DBName:  config.Database,
	})
	if err != nil {
		return nil, fmt.Errorf("connect milvus database %s: %w", config.Database, err)
	}

	if err := ensureCollection(ctx, agentClient, config.Collection); err != nil {
		agentClient.Close()
		return nil, err
	}

	return &Store{
		client:   agentClient,
		config:   config,
		embedder: embedder,
	}, nil
}

// Close 关闭 Milvus 客户端连接。
func (s *Store) Close() {
	if s == nil || s.client == nil {
		return
	}
	s.client.Close()
}

// DeleteBySource 删除指定来源文档对应的向量记录。
func (s *Store) DeleteBySource(ctx context.Context, source string) error {
	source = strings.TrimSpace(source)
	if source == "" {
		return nil
	}

	return s.client.Delete(ctx, s.config.Collection, "", fmt.Sprintf(`%s == "%s"`, DocIDField, sourceDocID(source)))
}

// validateConfig 校验 Milvus 必填配置。
func validateConfig(config Config) error {
	if strings.TrimSpace(config.Address) == "" {
		return fmt.Errorf("milvus address is required")
	}
	if strings.TrimSpace(config.DefaultDB) == "" {
		return fmt.Errorf("milvus default database is required")
	}
	if strings.TrimSpace(config.Database) == "" {
		return fmt.Errorf("milvus database is required")
	}
	if strings.TrimSpace(config.Collection) == "" {
		return fmt.Errorf("milvus collection is required")
	}
	return nil
}

// ensureDatabase 确保目标数据库存在，不存在时自动创建。
func ensureDatabase(ctx context.Context, client cli.Client, database string) error {
	databases, err := client.ListDatabases(ctx)
	if err != nil {
		return fmt.Errorf("list milvus databases: %w", err)
	}

	for _, db := range databases {
		if db.Name == database {
			return nil
		}
	}

	if err := client.CreateDatabase(ctx, database); err != nil {
		if isAlreadyExistsError(err) {
			return nil
		}
		return fmt.Errorf("create milvus database %s: %w", database, err)
	}
	return nil
}

// ensureCollection 确保目标集合和索引存在。
func ensureCollection(ctx context.Context, client cli.Client, collection string) error {
	collections, err := client.ListCollections(ctx)
	if err != nil {
		return fmt.Errorf("list milvus collections: %w", err)
	}

	for _, item := range collections {
		if item.Name == collection {
			return nil
		}
	}

	// 首次启动时创建集合结构，后续索引和检索统一复用该 schema。
	schema := &entity.Schema{
		CollectionName: collection,
		Description:    "Business knowledge collection",
		Fields:         Fields(),
	}

	if err := client.CreateCollection(ctx, schema, entity.DefaultShardNumber); err != nil {
		if isAlreadyExistsError(err) {
			return nil
		}
		return fmt.Errorf("create milvus collection %s: %w", collection, err)
	}

	idIndex, err := entity.NewIndexAUTOINDEX(entity.L2)
	if err != nil {
		return fmt.Errorf("create milvus id index: %w", err)
	}
	if err := client.CreateIndex(ctx, collection, IDField, idIndex, false); err != nil {
		return fmt.Errorf("create milvus id index: %w", err)
	}

	denseIndex, err := entity.NewIndexAUTOINDEX(entity.COSINE)
	if err != nil {
		return fmt.Errorf("create milvus dense index: %w", err)
	}
	if err := client.CreateIndex(ctx, collection, DenseVectorField, denseIndex, false); err != nil {
		return fmt.Errorf("create milvus dense index: %w", err)
	}

	sparseIndex := entity.NewGenericIndex(SparseVectorField, entity.SparseInverted, map[string]string{
		"metric_type": "IP",
	})
	if err := client.CreateIndex(ctx, collection, SparseVectorField, sparseIndex, false); err != nil {
		return fmt.Errorf("create milvus sparse index: %w", err)
	}

	return nil
}

// isAlreadyExistsError 判断创建数据库或集合时返回的错误是否表示资源已存在。
func isAlreadyExistsError(err error) bool {
	if err == nil {
		return false
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "already exist")
}

func sourceDocID(source string) string {
	sum := sha1.Sum([]byte(strings.TrimSpace(source)))
	return hex.EncodeToString(sum[:])
}
