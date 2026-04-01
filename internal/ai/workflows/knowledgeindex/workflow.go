package knowledgeindex

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/callbacks"
	"github.com/cloudwego/eino/components/document"
	"github.com/cloudwego/eino/components/indexer"
	"github.com/cloudwego/eino/compose"
	"go.uber.org/zap"
)

// SourceCleaner 定义知识重建前对旧向量数据的清理能力。
type SourceCleaner interface {
	DeleteBySource(ctx context.Context, source string) error
}

// Workflow 封装知识文档加载、切分和索引流程。
type Workflow struct {
	loader      document.Loader
	transformer document.Transformer
	indexer     indexer.Indexer
	cleaner     SourceCleaner
	runner      compose.Runnable[document.Source, []string]
	callback    callbacks.Handler
	logger      *zap.Logger
}

// Config 定义知识索引工作流的装配参数。
type Config struct {
	Loader      document.Loader
	Transformer document.Transformer
	Indexer     indexer.Indexer
	Cleaner     SourceCleaner
	Callback    callbacks.Handler
	Logger      *zap.Logger
}

// NewWorkflow 创建知识索引工作流，并在启动阶段预编译执行图。
func NewWorkflow(ctx context.Context, config Config) (*Workflow, error) {
	if config.Loader == nil {
		return nil, fmt.Errorf("loader is required")
	}
	if config.Transformer == nil {
		return nil, fmt.Errorf("transformer is required")
	}
	if config.Indexer == nil {
		return nil, fmt.Errorf("indexer is required")
	}
	if config.Cleaner == nil {
		return nil, fmt.Errorf("cleaner is required")
	}
	if config.Logger == nil {
		config.Logger = zap.NewNop()
	}

	workflow := &Workflow{
		loader:      config.Loader,
		transformer: config.Transformer,
		indexer:     config.Indexer,
		cleaner:     config.Cleaner,
		callback:    config.Callback,
		logger:      config.Logger.Named("knowledge_index"),
	}

	runner, err := workflow.buildRunner(ctx)
	if err != nil {
		return nil, fmt.Errorf("compile knowledge index workflow: %w", err)
	}
	workflow.runner = runner

	return workflow, nil
}

// Refresh 重新索引指定路径的文档。
func (w *Workflow) Refresh(ctx context.Context, path string) error {
	docs, err := w.loader.Load(ctx, document.Source{URI: path})
	if err != nil {
		return err
	}
	if len(docs) == 0 {
		return fmt.Errorf("no documents loaded from %s", path)
	}

	source := fmt.Sprint(docs[0].MetaData["_source"])
	if err := w.cleaner.DeleteBySource(ctx, source); err != nil {
		w.logger.Warn("delete existing data failed", zap.Error(err), zap.String("source", source))
	}

	ids, err := w.runner.Invoke(ctx, document.Source{URI: path}, w.invokeOptions()...)
	if err != nil {
		return fmt.Errorf("invoke index graph failed: %w", err)
	}

	w.logger.Info("knowledge indexing completed", zap.String("path", path), zap.Int("parts", len(ids)))
	return nil
}

// buildRunner 组装文档加载、切分和向量入库的执行图。
func (w *Workflow) buildRunner(ctx context.Context) (compose.Runnable[document.Source, []string], error) {
	const (
		fileLoader       = "FileLoader"
		markdownSplitter = "MarkdownSplitter"
		milvusIndexer    = "MilvusIndexer"
	)

	graph := compose.NewGraph[document.Source, []string]()
	_ = graph.AddLoaderNode(fileLoader, w.loader)
	_ = graph.AddDocumentTransformerNode(markdownSplitter, w.transformer)
	_ = graph.AddIndexerNode(milvusIndexer, w.indexer)
	_ = graph.AddEdge(compose.START, fileLoader)
	_ = graph.AddEdge(fileLoader, markdownSplitter)
	_ = graph.AddEdge(markdownSplitter, milvusIndexer)
	_ = graph.AddEdge(milvusIndexer, compose.END)

	return graph.Compile(ctx, compose.WithGraphName("KnowledgeIndexing"), compose.WithNodeTriggerMode(compose.AnyPredecessor))
}

// invokeOptions 根据是否配置回调决定是否注入链路日志能力。
func (w *Workflow) invokeOptions() []compose.Option {
	if w.callback == nil {
		return nil
	}
	return []compose.Option{compose.WithCallbacks(w.callback)}
}
