# 005-Milvus混合检索改造-实现总结

## 1. 实现了什么

本次改动把 `internal/infra/vectorstore/milvus` 从单路 dense 检索改成了一个最小可用的 hybrid 结构：

- schema 新增 `doc_id`、`title`、`chunk_id`、`dense_vector`、`sparse_vector`
- indexer 改为同时写入 dense 向量和 sparse 向量
- retriever 改为调用 Milvus `HybridSearch + WeightedReranker`
- `DeleteBySource` 改为基于 `doc_id` 删除
- 新增 `sparse.go`，使用 `jieba` 做中文分词并构建简化 sparse 向量

## 2. 与需求的对应关系

对应 [005-Milvus混合检索改造-需求.md](/home/ristial/projects/agent/OpsPilot/docs/requirements/005-Milvus混合检索改造-需求.md)：

- “schema 支持新增字段”：已完成
- “入库同时写 dense 和 sparse”：已完成
- “检索切到 HybridSearch”：已完成
- “外部签名保持不变”：已完成
- “不改 chat 与上层调用方式”：已完成

## 3. 关键实现点

- 在 [schema.go](/home/ristial/projects/agent/OpsPilot/internal/infra/vectorstore/milvus/schema.go) 中重写 collection 字段定义
- 在 [client.go](/home/ristial/projects/agent/OpsPilot/internal/infra/vectorstore/milvus/client.go) 中更新 collection 初始化和按 `doc_id` 删除逻辑
- 在 [indexer.go](/home/ristial/projects/agent/OpsPilot/internal/infra/vectorstore/milvus/indexer.go) 中去掉默认 `eino-ext` indexer，直接写自定义列式插入
- 在 [retriever.go](/home/ristial/projects/agent/OpsPilot/internal/infra/vectorstore/milvus/retriever.go) 中切到 `HybridSearch`
- 在 [sparse.go](/home/ristial/projects/agent/OpsPilot/internal/infra/vectorstore/milvus/sparse.go) 中补充分词和 sparse 向量构造

## 4. 已知限制或待改进点

- `dense_vector` 维度目前是包内固定常量，后续应与真实 embedding 维度对齐
- sparse 权重是简化版 TF/BM25 风格实现，不是完整标准 BM25
- 本期没有实现 query rewrite、去重节点和上下文压缩
- 本期不处理旧 schema 的兼容迁移

## 5. 验证结果

本次改动后已执行：

```bash
gofmt -w internal/infra/vectorstore/milvus/*.go

go test ./...

go test ./internal/infra/vectorstore/milvus
```

验证结果：

- `gofmt -w internal/infra/vectorstore/milvus/*.go` 已执行
- `go test ./internal/infra/vectorstore/milvus` 通过
- `go test ./...` 未通过，失败原因是仓库内已有的 [workflow.go](/home/ristial/projects/agent/OpsPilot/internal/ai/workflows/chat/workflow.go) 第 146 行附近存在语法错误，与本次 Milvus 改造无关

## 6. 实现总结

这次改动先把 Milvus 这一层的能力补齐，为后续 chat 工作流中的 query rewrite 和节点去重留出了稳定接口。上层调用方仍然只需要依赖 `NewIndexer` 和 `NewRetriever`，不需要感知 dense/sparse 两路检索的内部细节。
