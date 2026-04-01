package bootstrap

import (
	"OpsPilot/internal/ai/registry"
	"OpsPilot/internal/infra/llm"
	"OpsPilot/internal/infra/llm/ark"
	"OpsPilot/internal/infra/llm/openai"
	"OpsPilot/internal/infra/logging"
	"OpsPilot/internal/infra/mcp"
	sessionredis "OpsPilot/internal/infra/session/redis"
	"OpsPilot/internal/infra/storage/localfs"
	"OpsPilot/internal/interfaces/http/handler"
	"OpsPilot/internal/interfaces/http/middleware"
	"OpsPilot/internal/interfaces/http/router"
	pkgauth "OpsPilot/pkg/auth"
	"OpsPilot/pkg/snowflake"
	"context"
	"database/sql"
	"net/http"
	"time"

	aiopsplan "OpsPilot/internal/ai/workflows/aiopsplan"
	chatflow "OpsPilot/internal/ai/workflows/chat"
	knowledgeflow "OpsPilot/internal/ai/workflows/knowledgeindex"
	summaryflow "OpsPilot/internal/ai/workflows/sessionsummary"
	aiopsapp "OpsPilot/internal/application/aiops"
	authapp "OpsPilot/internal/application/auth"
	chatapp "OpsPilot/internal/application/chat"
	knowledgeapp "OpsPilot/internal/application/knowledge"
	appconfig "OpsPilot/internal/infra/config"
	usermysql "OpsPilot/internal/infra/user/mysql"

	milvusstore "OpsPilot/internal/infra/vectorstore/milvus"

	"github.com/cloudwego/eino/components/retriever"
	"github.com/cloudwego/eino/components/tool"
	"github.com/gin-gonic/gin"
	goredis "github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// App 聚合启动完成后的核心依赖，便于服务端和示例程序复用同一套装配结果。
type App struct {
	Config           *appconfig.Config
	Logger           *zap.Logger
	Engine           *gin.Engine
	AuthService      *authapp.Service
	ChatService      *chatapp.Service
	KnowledgeService *knowledgeapp.Service
	AIOpsService     *aiopsapp.Service
	Retriever        retriever.Retriever
	ChatTools        []tool.BaseTool
	AIOpsTools       []tool.BaseTool

	vectorStore *milvusstore.Store
	redisClient *goredis.Client
	sqlDB       *sql.DB
}

// New 根据配置装配日志、模型、向量库、工作流和 HTTP 路由。
func New(ctx context.Context, configPath string) (*App, error) {
	cfg, err := appconfig.Load(configPath)
	if err != nil {
		return nil, err
	}

	logger, err := logging.NewLogger(cfg.Logger)
	if err != nil {
		return nil, err
	}

	accessTokenTTL, err := time.ParseDuration(cfg.Auth.AccessTokenTTL)
	if err != nil {
		logging.Sync(logger)
		return nil, err
	}
	sessionTTL, err := time.ParseDuration(cfg.Session.TTL)
	if err != nil {
		logging.Sync(logger)
		return nil, err
	}
	idGenerator, err := snowflake.NewGenerator(cfg.Snowflake.NodeID)
	if err != nil {
		logging.Sync(logger)
		return nil, err
	}
	tokenManager, err := pkgauth.NewManager(cfg.Auth.JWTSecret, accessTokenTTL)
	if err != nil {
		logging.Sync(logger)
		return nil, err
	}

	db, err := gorm.Open(mysql.Open(cfg.MySQL.DSN), &gorm.Config{})
	if err != nil {
		logging.Sync(logger)
		return nil, err
	}
	sqlDB, err := db.DB()
	if err != nil {
		logging.Sync(logger)
		return nil, err
	}
	if err := sqlDB.PingContext(ctx); err != nil {
		sqlDB.Close()
		logging.Sync(logger)
		return nil, err
	}
	userStore, err := usermysql.NewStore(db)
	if err != nil {
		sqlDB.Close()
		logging.Sync(logger)
		return nil, err
	}

	redisClient := goredis.NewClient(&goredis.Options{
		Addr:     cfg.Redis.Address,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
	if err := redisClient.Ping(ctx).Err(); err != nil {
		sqlDB.Close()
		redisClient.Close()
		logging.Sync(logger)
		return nil, err
	}

	// 统一注册聊天模型与嵌入模型提供者，后续按配置动态选择实现。
	modelRegistry := llm.NewRegistry()
	openaiProvider := openai.NewProvider()
	arkProvider := ark.NewProvider()
	modelRegistry.RegisterChatProvider(llm.ProviderOpenai, openaiProvider)
	modelRegistry.RegisterChatProvider(llm.ProviderArk, arkProvider)
	modelRegistry.RegisterEmbeddingProvider(llm.ProviderArk, arkProvider)

	logCallback := logging.NewCallback(logger, nil)

	embedder, err := modelRegistry.NewEmbedder(ctx, cfg.EmbeddingModel)
	if err != nil {
		sqlDB.Close()
		redisClient.Close()
		logging.Sync(logger)
		return nil, err
	}

	// 启动时确保目标数据库和集合存在，避免知识检索和索引流程在首次运行时失败。
	vectorStore, err := milvusstore.NewStore(ctx, milvusstore.Config{
		Address:    cfg.Milvus.Address,
		DefaultDB:  cfg.Milvus.DefaultDB,
		Database:   cfg.Milvus.Database,
		Collection: cfg.Milvus.Collection,
	}, embedder)
	if err != nil {
		sqlDB.Close()
		redisClient.Close()
		logging.Sync(logger)
		return nil, err
	}

	quickModel, err := modelRegistry.NewChatModel(ctx, cfg.QuickChatModel)
	if err != nil {
		vectorStore.Close()
		sqlDB.Close()
		redisClient.Close()
		logging.Sync(logger)
		return nil, err
	}
	thinkModel, err := modelRegistry.NewChatModel(ctx, cfg.ThinkChatModel)
	if err != nil {
		vectorStore.Close()
		sqlDB.Close()
		redisClient.Close()
		logging.Sync(logger)
		return nil, err
	}

	// 工作流会共享同一个检索器和工具集，因此这里先完成底层依赖装配。
	retriever, err := vectorStore.NewRetriever(ctx, 1)
	if err != nil {
		vectorStore.Close()
		sqlDB.Close()
		redisClient.Close()
		logging.Sync(logger)
		return nil, err
	}
	indexer, err := vectorStore.NewIndexer(ctx)
	if err != nil {
		vectorStore.Close()
		sqlDB.Close()
		redisClient.Close()
		logging.Sync(logger)
		return nil, err
	}
	loader, err := newFileLoader(ctx)
	if err != nil {
		vectorStore.Close()
		sqlDB.Close()
		redisClient.Close()
		logging.Sync(logger)
		return nil, err
	}
	splitter, err := newMarkdownSplitter(ctx)
	if err != nil {
		vectorStore.Close()
		sqlDB.Close()
		redisClient.Close()
		logging.Sync(logger)
		return nil, err
	}

	// MCP 工具加载失败时仅降级为无日志工具，避免阻塞主流程启动。
	logToolProvider := mcp.NewToolProvider(cfg.MCP.URL)
	httpClient := &http.Client{Timeout: 10 * time.Second}
	logTools, err := logToolProvider.Tools(ctx)
	if err != nil {
		logger.Warn("load log tools failed", zap.Error(err))
		logTools = nil
	}

	chatTools, err := registry.NewChatTools(registry.ChatToolsConfig{
		LogTools:             logTools,
		Retriever:            retriever,
		PrometheusBaseURL:    cfg.Prometheus.BaseURL,
		PrometheusHTTPClient: httpClient,
	})
	if err != nil {
		vectorStore.Close()
		sqlDB.Close()
		redisClient.Close()
		logging.Sync(logger)
		return nil, err
	}
	aiopsTools, err := registry.NewAIOpsTools(registry.AIOpsToolsConfig{
		LogTools:             logTools,
		Retriever:            retriever,
		PrometheusBaseURL:    cfg.Prometheus.BaseURL,
		PrometheusHTTPClient: httpClient,
	})
	if err != nil {
		vectorStore.Close()
		sqlDB.Close()
		redisClient.Close()
		logging.Sync(logger)
		return nil, err
	}

	chatWorkflow, err := chatflow.NewWorkflow(ctx, chatflow.Config{
		ChatModel: quickModel,
		Retriever: retriever,
		Tools:     chatTools,
		Callback:  logCallback,
	})
	if err != nil {
		vectorStore.Close()
		sqlDB.Close()
		redisClient.Close()
		logging.Sync(logger)
		return nil, err
	}
	knowledgeWorkflow, err := knowledgeflow.NewWorkflow(ctx, knowledgeflow.Config{
		Loader:      loader,
		Transformer: splitter,
		Indexer:     indexer,
		Cleaner:     vectorStore,
		Callback:    logCallback,
		Logger:      logger,
	})
	if err != nil {
		vectorStore.Close()
		sqlDB.Close()
		redisClient.Close()
		logging.Sync(logger)
		return nil, err
	}
	aiopsWorkflow, err := aiopsplan.NewWorkflow(ctx, aiopsplan.Config{
		PlannerModel:  thinkModel,
		ExecutorModel: quickModel,
		Tools:         aiopsTools,
		Logger:        logger,
	})
	if err != nil {
		vectorStore.Close()
		sqlDB.Close()
		redisClient.Close()
		logging.Sync(logger)
		return nil, err
	}
	summaryWorkflow, err := summaryflow.NewWorkflow(ctx, summaryflow.Config{
		ChatModel: quickModel,
		Callback:  logCallback,
	})
	if err != nil {
		vectorStore.Close()
		sqlDB.Close()
		redisClient.Close()
		logging.Sync(logger)
		return nil, err
	}

	sessionStore := sessionredis.NewStore(redisClient, sessionTTL)
	fileStore := localfs.NewStore(cfg.FileDir)
	authService := authapp.NewService(userStore, idGenerator, tokenManager)

	// 将应用服务与 HTTP 处理器解耦，便于 CLI 示例和 HTTP 服务共享同一服务层实现。
	chatService := chatapp.NewService(sessionStore, chatWorkflow, summaryWorkflow, idGenerator, chatapp.Config{
		RetainTurns:           cfg.Session.RetainTurns,
		CompactThresholdTurns: cfg.Session.CompactThresholdTurns,
	})
	knowledgeService := knowledgeapp.NewService(fileStore, knowledgeWorkflow)
	aiopsService := aiopsapp.NewService(aiopsWorkflow)

	authHandler := handler.NewAuthHandler(authService)
	chatHandler := handler.NewChatHandler(chatService, knowledgeService, aiopsService)

	authMiddleware := middleware.JWT(tokenManager)

	return &App{
		Config:           cfg,
		Logger:           logger,
		Engine:           router.New(authHandler, chatHandler, authMiddleware),
		AuthService:      authService,
		ChatService:      chatService,
		KnowledgeService: knowledgeService,
		AIOpsService:     aiopsService,
		Retriever:        retriever,
		ChatTools:        append([]tool.BaseTool{}, chatTools...),
		AIOpsTools:       append([]tool.BaseTool{}, aiopsTools...),
		vectorStore:      vectorStore,
		redisClient:      redisClient,
		sqlDB:            sqlDB,
	}, nil
}

// Close 释放向量库连接并同步落盘日志。
func (a *App) Close() {
	if a == nil {
		return
	}
	if a.vectorStore != nil {
		a.vectorStore.Close()
	}
	if a.redisClient != nil {
		_ = a.redisClient.Close()
	}
	if a.sqlDB != nil {
		_ = a.sqlDB.Close()
	}
	logging.Sync(a.Logger)
}
