# 006-Eino查询重写优化知识检索-实现总结

## 1. 实现了什么

基于提交 `9750d16c7e5a07b50a1d94b58fedb9284424e8e4`，chat 工作流新增了一段面向知识检索的查询重写链路：

- 在 [prompt.go](/home/ristial/projects/agent/OpsPilot/internal/ai/workflows/chat/prompt.go) 中新增 `queryRewritePrompt`
- 在 [workflow.go](/home/ristial/projects/agent/OpsPilot/internal/ai/workflows/chat/workflow.go) 中新增 `RewriteModel` 节点和 `rewriteLambda`
- 多轮对话时，先用历史消息和当前问题生成一条独立检索语句，再交给 retriever
- 单轮对话时，直接使用原始 query，避免不必要的模型调用
- 最终回答阶段仍然沿用原始用户问题和历史消息，不把改写后的 query 暴露给最终回答模板

## 2. 与需求的对应关系

对应 [006-Eino查询重写优化知识检索-需求.md](/home/ristial/projects/agent/OpsPilot/docs/requirements/006-Eino查询重写优化知识检索-需求.md)：

- “存在历史消息时先做查询重写”：已完成
- “查询重写只影响检索输入”：已完成
- “无历史消息时直接使用原始 query”：已完成
- “不新增 HTTP 接口字段、tool 注册和配置项”：已完成
- “保持启动时编译、运行时复用”：已完成

## 3. 关键实现点

- 在 [prompt.go](/home/ristial/projects/agent/OpsPilot/internal/ai/workflows/chat/prompt.go) 中新增独立提示词，明确要求模型只返回一条适合知识库召回的纯文本检索语句
- 在 [workflow.go](/home/ristial/projects/agent/OpsPilot/internal/ai/workflows/chat/workflow.go) 中新增 `rewriteLambda`，先判断 `History` 是否为空，再决定是否调用模型重写
- `rewriteLambda` 复用现有 `w.chatModel.Generate(...)`，不引入新的模型装配项
- 执行图改成两条并行分支：
  - `START -> InputToChat`：继续为最终回答阶段保留原始问题和历史消息
  - `START -> RewriteModel -> InputToRag -> Retriever`：仅为检索链路提供改写后的 query
- `Chat` 和 `Stream` 对外签名没有变化，上层 application、HTTP、bootstrap 不需要联动修改

## 4. 已知限制或待改进点

- 多轮对话会新增一次模型调用，检索前耗时会增加
- 查询重写和最终回答共用同一个 chat model，后续如果模型职责继续增加，可能需要拆分配置
- 当前实现直接使用模型返回的 `msg.Content` 作为检索 query，没有增加格式清洗或回退逻辑
- 目前没有补充离线评估样本或线上指标，收益主要体现在工程路径打通

## 5. 验证结果

本次为补充需求与实现总结文档，未修改 Go 代码。

结合提交 `9750d16c7e5a07b50a1d94b58fedb9284424e8e4` 的 diff 可以确认：

- 改动范围只落在 [prompt.go](/home/ristial/projects/agent/OpsPilot/internal/ai/workflows/chat/prompt.go) 和 [workflow.go](/home/ristial/projects/agent/OpsPilot/internal/ai/workflows/chat/workflow.go)
- 查询重写提示词、节点装配、无历史透传逻辑都已落地
- chat 工作流对外接口和上层调用路径未发生变化

## 6. 实现总结

这次改动是在 `005-Milvus混合检索改造` 之后，补齐 chat 工作流对多轮追问场景的检索适配能力。底层混合检索负责“怎么召回”，而这次查询重写负责“拿什么去召回”，两者组合后，知识库链路才具备对上下文追问更稳定的召回基础。
