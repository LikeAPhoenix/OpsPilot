# 0. 文件修改记录表

| 修改人 | 修改时间 | 修改内容 |
| ------ | -------- | -------- |
| Codex | 2026-04-01 | 初始版本，新增 Eino 查询重写优化知识检索需求 |

# 1. 背景（Why）

在 `005-Milvus混合检索改造` 完成后，知识库检索已经具备更好的召回基础，但 chat 工作流在多轮对话场景下仍然直接使用用户当前输入做检索。对于“这个告警怎么处理”“那生产环境呢”这类依赖上下文的追问，原始 query 往往缺少明确对象、环境和时间范围，会直接拉低 Milvus 检索命中率。因此需要在 chat 工作流中补一个查询重写节点，先结合历史消息把当前问题改写成适合知识库召回的独立检索语句，再交给 retriever。

# 2. 目标（What，必须可验证）

- [ ] chat 工作流在存在历史消息时，先执行一次查询重写，再进入知识库检索
- [ ] 查询重写只影响检索输入，不改变最终回答阶段使用的用户原始问题
- [ ] 没有历史消息时，不额外调用模型做重写，直接使用原始 query 检索
- [ ] 不新增 HTTP 接口字段、不修改 tool 注册、不新增独立配置项
- [ ] 继续保持 chat 工作流“启动时编译、运行时复用”的装配方式

# 3. 非目标（Explicitly Out of Scope）

- 不改 Milvus schema、索引写入方式或混合检索权重
- 不实现查询改写结果缓存
- 不新增专用 rewrite model 配置，继续复用现有 chat model
- 不改最终回答提示词的输出格式和回答结构

# 4. 使用场景 / 用户路径

1. 用户在首轮提问，例如“服务下线是什么原因”。
2. chat 工作流收到请求后，发现没有历史消息，直接使用原始 query 进入 retriever。
3. 用户在下一轮追问，例如“那生产环境这个告警该怎么处理”。
4. chat 工作流发现存在历史消息，先把历史消息和当前问题交给查询重写节点。
5. 查询重写节点输出一条适合知识库召回的独立检索语句。
6. retriever 使用改写后的 query 检索知识库。
7. 最终回答阶段仍然使用原始用户问题、历史消息和检索文档生成答案。

# 5. 功能需求清单（Checklist）

- [ ] 在 `internal/ai/workflows/chat/prompt.go` 中新增查询重写提示词
- [ ] 在 `internal/ai/workflows/chat/workflow.go` 中新增查询重写节点
- [ ] 在 `internal/ai/workflows/chat/workflow.go` 中新增 `rewriteLambda`
- [ ] `rewriteLambda` 需要在无历史消息时直接透传原始输入
- [ ] 查询重写节点的输出只接入 RAG 检索链路，不覆盖 chat 模板中的原始问题
- [ ] 保持 `NewWorkflow`、`Chat`、`Stream` 外部签名不变

# 6. 约束条件（非常关键）

- 技术约束：继续使用当前 Eino graph 与 `model.ToolCallingChatModel`，不引入新的工作流框架
- 架构约束：改动范围只允许落在 `internal/ai/workflows/chat`
- 装配约束：必须继续遵循“启动时构建执行图，运行时直接复用”的现有约定
- 接口约束：不改 HTTP handler、application service、bootstrap 装配签名
- 提示词约束：查询重写输出必须是纯文本，且只返回一条检索语句
- 代码约束：实现保持简单直接，不增加复杂兜底或额外状态管理

# 7. 可修改 / 不可修改项

- ❌ 不可修改：`POST /api/chat`、`POST /api/chat_stream` 入参结构，tool 名称，retriever 外部接口，聊天工作流最终回答模板结构
- ✅ 可调整：chat 工作流内部节点命名、图边连接方式、查询重写提示词内容、`rewriteLambda` 的消息拼装方式

# 8. 接口与数据约定（如适用）

- `Input` 结构体保持不变，仍然使用 `ID`、`Query`、`History`
- 查询重写节点输入：完整 `Input`
- 查询重写节点输出：新的 `Input`
- 当 `len(input.History) == 0` 时，查询重写节点直接返回原始 `Input`
- 当 `len(input.History) > 0` 时，查询重写节点构造如下消息：
  - system：查询重写提示词
  - history：原始历史消息
  - user：要求将当前问题改写为可直接用于知识库检索的查询
- 查询重写后的结果只覆盖送入 retriever 的 `Query`
- 最终回答阶段继续使用原始 `input.Query` 和 `input.History`

# 9. 验收标准（Acceptance Criteria）

- 如果查看 [prompt.go](/home/ristial/projects/agent/OpsPilot/internal/ai/workflows/chat/prompt.go)，则可以明确看到独立的查询重写提示词
- 如果查看 [workflow.go](/home/ristial/projects/agent/OpsPilot/internal/ai/workflows/chat/workflow.go)，则可以明确看到查询重写节点被接入 `START -> RewriteModel -> InputToRag`
- 如果查看 [workflow.go](/home/ristial/projects/agent/OpsPilot/internal/ai/workflows/chat/workflow.go)，则可以明确看到无历史消息时直接透传原始 query
- 如果查看 [workflow.go](/home/ristial/projects/agent/OpsPilot/internal/ai/workflows/chat/workflow.go)，则可以明确看到最终 chat 模板仍然使用原始 `input.Query`
- 如果检查 chat 接口调用方，则不需要新增任何请求字段或配置项

# 10. 风险与已知不确定点

- 多轮对话会多一次模型调用，响应延迟会高于单轮检索
- 查询重写质量直接依赖当前 chat model，若模型改写偏离用户意图，检索结果也会偏移
- 当前实现没有对重写结果做额外裁剪、去噪或兜底校验
- 本期没有建立查询重写效果评估指标，收益主要依赖实际使用反馈

# 11. 非目标

- 不实现查询重写结果缓存
- 不实现独立 rewrite model 或额外模型路由
- 不实现查询重写效果打分与自动回退
- 不改知识库写入链路和混合检索底层实现
