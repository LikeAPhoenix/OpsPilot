# 0. 文件修改记录表

| 修改人 | 修改时间 | 修改内容 |
| ------ | -------- | -------- |
| Codex | 2026-04-01 | 初始版本，新增 Milvus 混合检索改造需求 |

# 1. 背景（Why）

当前知识检索只使用 Milvus 单路向量检索，索引写入和查询都只覆盖 dense 向量，无法支撑后续 chat 工作流中的查询改写和混合检索接入。因此需要先把 Milvus 存储层升级为同时支持 dense 向量与 sparse 向量，并在检索侧切到 Milvus HybridSearch。

# 2. 目标（What，必须可验证）

- [ ] Milvus collection schema 支持 `dense_vector`、`sparse_vector`、`doc_id`、`title`、`chunk_id`
- [ ] 知识入库时同时写入 dense 向量和 sparse 向量
- [ ] Milvus 检索从单路 `Search` 改为 `HybridSearch + WeightedReranker`
- [ ] `NewStore`、`NewIndexer`、`NewRetriever` 外部签名保持不变
- [ ] 不修改 chat 工作流、tool、AIOps 的上层调用方式

# 3. 非目标（Explicitly Out of Scope）

- 不改 chat 工作流中的 query rewrite 节点
- 不在本期实现去重节点或上下文压缩
- 不做旧 collection schema 的在线兼容或迁移
- 不追求严格学术 BM25 实现

# 4. 使用场景 / 用户路径

1. 服务启动时装配 Milvus store，并按新 schema 创建 collection。
2. 用户上传或重建知识库时，知识索引工作流调用 `NewIndexer` 返回的 indexer 写入文档。
3. indexer 为每个 chunk 写入 dense 向量、sparse 向量和新增标量字段。
4. 上层聊天或工具链路调用 `NewRetriever` 返回的 retriever 执行检索。
5. retriever 在 Milvus 内部执行 dense + sparse 混合检索，并返回文档列表。

# 5. 功能需求清单（Checklist）

- [ ] 修改 `internal/infra/vectorstore/milvus/schema.go`
- [ ] 修改 `internal/infra/vectorstore/milvus/client.go`
- [ ] 重写 `internal/infra/vectorstore/milvus/indexer.go`
- [ ] 重写 `internal/infra/vectorstore/milvus/retriever.go`
- [ ] 新增 `internal/infra/vectorstore/milvus/sparse.go`
- [ ] `DeleteBySource` 改为基于 `doc_id` 删除
- [ ] 文档 metadata 中带回 `id`、`doc_id`、`title`、`chunk_id`

# 6. 约束条件（非常关键）

- 技术约束：继续使用当前 Milvus Go SDK，不引入新的向量数据库
- 架构约束：改动范围只允许落在 `internal/infra/vectorstore/milvus`
- 代码约束：代码简洁明了，不做过多兜底和复杂校验
- 行为约束：本期不要求保证代码可以直接跑通，但实现路径必须清晰
- 数据约束：默认会删除原有 Milvus 数据，不做兼容迁移

# 7. 可修改 / 不可修改项

- ❌ 不可修改：chat 工作流结构、tool 名称、HTTP 接口路径、bootstrap 调用方式
- ✅ 可调整：Milvus schema、索引写入方式、检索实现、内部辅助函数、Go 依赖

# 8. 接口与数据约定（如适用）

- `Fields()` 返回字段固定包含：
  - `id`
  - `doc_id`
  - `title`
  - `chunk_id`
  - `dense_vector`
  - `sparse_vector`
  - `content`
  - `metadata`
- `id` 规则：`doc_id + ":" + chunk_id`
- `doc_id` 规则：由 `_source` 生成稳定字符串
- `chunk_id` 规则：按当前批次文档顺序从 `0` 开始递增
- 检索权重固定：
  - dense：`0.7`
  - sparse：`0.3`

# 9. 验收标准（Acceptance Criteria）

- 如果查看 `schema.go`，则可以明确看到新字段结构
- 如果查看 `indexer.go`，则可以明确看到 dense 和 sparse 同时写入
- 如果查看 `retriever.go`，则可以明确看到 `HybridSearch` 而不是原来的单路 `Search`
- 如果查看 `client.go`，则可以明确看到 `DeleteBySource` 基于 `doc_id`
- 如果查看返回文档 metadata，则包含 `id`、`doc_id`、`title`、`chunk_id`

# 10. 风险与已知不确定点

- 由于本期不追求可直接运行，`dense_vector` 的维度和实际 embedding 维度可能仍需要后续对齐
- `jieba` 依赖会增加本地构建要求
- 简化版 sparse 权重更偏工程实现，不保证与标准 BM25 完全一致

# 11. 非目标

- 不实现 query rewrite
- 不实现上层工作流节点去重
- 不实现 rerank 模型
- 不实现 BM25 统计持久化
