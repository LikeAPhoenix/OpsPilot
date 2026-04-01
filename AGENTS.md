# OpsPilot Agent Guide

本文件面向在本仓库内工作的 AI Agent 和协作者，目标是让改动尽量贴合当前代码结构，而不是依赖泛化假设。

## 1. 项目概览

OpsPilot 是一个基于 Go 的智能运维服务，核心能力包括：

- 聊天问答：带会话历史、RAG 检索和工具调用
- 知识库索引：将 Markdown 文档写入 Milvus，供检索使用
- AIOps 分析：读取 Prometheus 活跃告警，结合内部文档和工具生成分析报告

主服务入口是 [cmd/server/main.go](/home/ristial/projects/agent/OpsPilot/cmd/server/main.go)，装配逻辑集中在 [internal/bootstrap/app.go](/home/ristial/projects/agent/OpsPilot/internal/bootstrap/app.go)。

## 2. 目录边界

- [cmd/server/main.go](/home/ristial/projects/agent/OpsPilot/cmd/server/main.go)：HTTP 服务启动入口
- [internal/bootstrap](/home/ristial/projects/agent/OpsPilot/internal/bootstrap)：统一装配配置、日志、模型、Milvus、工作流、HTTP 路由
- [internal/application](/home/ristial/projects/agent/OpsPilot/internal/application)：应用服务层，负责编排业务流程，不直接关心底层实现
- [internal/ai/workflows](/home/ristial/projects/agent/OpsPilot/internal/ai/workflows)：聊天、知识索引、AIOps 工作流
- [internal/ai/tools](/home/ristial/projects/agent/OpsPilot/internal/ai/tools)：Eino 工具定义
- [internal/ai/registry](/home/ristial/projects/agent/OpsPilot/internal/ai/registry)：不同场景下的工具注册入口
- [internal/infra](/home/ristial/projects/agent/OpsPilot/internal/infra)：配置、模型提供者、MCP、日志、Milvus、会话存储、文件存储
- [internal/interfaces/http](/home/ristial/projects/agent/OpsPilot/internal/interfaces/http)：Gin handler、DTO、SSE、路由
- [examples](/home/ristial/projects/agent/OpsPilot/examples)：独立示例程序，适合验证单条链路
- [docs](/home/ristial/projects/agent/OpsPilot/docs)：内部知识库源文档。当前知识索引只处理 Markdown 文件

改动时优先保持这层边界，不要把 HTTP 细节下沉到 application，也不要把业务编排塞回 infra。

## 3. 配置与依赖

默认配置文件是 [configs/config.yaml](/home/ristial/projects/agent/OpsPilot/configs/config.yaml)，加载逻辑在 [internal/infra/config/loader.go](/home/ristial/projects/agent/OpsPilot/internal/infra/config/loader.go)。

- 默认启动会读取 `configs/config.yaml`
- 环境变量前缀为 `OPSPILOT_`
- 环境变量可以覆盖配置文件字段，例如 `OPSPILOT_SERVER_ADDRESS`

运行前通常需要准备：

- 可用的聊天模型与嵌入模型配置
- Milvus 实例
- Prometheus HTTP API
- MCP 日志工具地址
- MySQL 实例
- Redis 实例

注意：

- MCP 工具加载失败会降级为“无日志工具”，不会阻止服务启动
- Milvus、模型配置不可用时，应用装配会失败
- `chat`、`knowledgeindex`、`aiopsplan` 三个工作流会在 `NewWorkflow(ctx, ...)` 阶段预编译执行图或预构建 Agent，相关构建失败会直接阻止启动，而不是推迟到首个请求
- `file_dir` 默认是 `./docs/knowledge`，上传知识文件会先落盘再立即索引
- `mysql.dsn`、`redis.address`、`auth.jwt_secret`、`session.ttl`、`snowflake.node_id` 不可用时，应用装配会失败
- 会话短期记忆默认采用“最近 6 轮明细 + 摘要 + Redis TTL”的策略，超过 9 轮时才触发摘要压缩

## 4. 主要接口与能力

HTTP 路由定义在 [internal/interfaces/http/router/router.go](/home/ristial/projects/agent/OpsPilot/internal/interfaces/http/router/router.go)：

- `POST /api/auth/register`：注册本地账号
- `POST /api/auth/login`：登录并获取 JWT access token
- `POST /api/chat`：普通问答
- `POST /api/chat_stream`：SSE 流式问答
- `POST /api/upload`：上传并索引知识文件
- `POST /api/ai_ops`：触发 AIOps 分析

说明：

- 除 `/api/auth/*` 外，其他业务接口统一要求携带 `Authorization: Bearer <token>`
- `chat`、`chat_stream` 的 `sessionId` 允许首轮为空，由服务端自动生成并回传

工具注册约定：

- 聊天工具入口：[internal/ai/registry/chat_tools.go](/home/ristial/projects/agent/OpsPilot/internal/ai/registry/chat_tools.go)
- AIOps 工具入口：[internal/ai/registry/aiops_tools.go](/home/ristial/projects/agent/OpsPilot/internal/ai/registry/aiops_tools.go)

如果新增工具，至少同步检查：

- 工具描述是否足够让模型正确调用
- 是否应该暴露给 chat、aiops 或两者
- 上层 prompt 或工作流里是否引用了正确工具名

## 5. 常用命令

建议优先使用仓库内已有入口验证改动：

```bash
go run ./cmd/server
go run ./examples/chat
go run ./examples/knowledge ./docs
go run ./examples/recall "服务下线是什么原因"
go run ./examples/aiops
go run ./examples/tools
go test ./...
gofmt -w <changed-files>
```

说明：

- `examples/knowledge` 只会索引 `.md` 文件
- `examples/aiops` 和主服务都依赖外部模型、Milvus、Prometheus 等真实配置
- 当前仓库没有明显的测试目录，`go test ./...` 更多用于编译与回归检查

## 6. 修改约束

1. 修改前先读受影响链路的入口文件，不要只看单个函数就下手。
2. 保持工具名、配置名、接口路径稳定，除非任务明确要求变更。
3. 涉及知识库功能时，优先兼容现有 Markdown 文档与 `docs/` 目录约定。
4. 涉及 AIOps 分析时，注意当前实现强依赖内部文档内容，不应随意引入文档外推断。
5. 不要把密钥、DSN、真实租户信息直接写入代码或提交到配置文件。
6. 修改 Go 代码后至少运行 `gofmt`；能跑通的情况下再补 `go test ./...` 或相关示例。
7. 如果变更影响工具注册、配置结构或接口入参，记得同步更新本文档和相关示例。

## 7. 编码规范

代码简洁明了，不做过多兜底逻辑

- 字符串不做过多的兜底，信任上层传入的数据


## 7. 功能总结与需求对应

### 7.1 功能总结文档（强制）

每一个功能开发完成后，AI **必须撰写功能总结文档**，内容包括：

- 实现了什么
- 与需求的对应关系
- 关键实现点
- 已知限制或待改进点
- 实现总结必须能明确对应到原始需求文档

### 7.2 存放规则

- 请遵循 [docs/文档编写规范/README.md](/home/ristial/projects/agent/OpsPilot/docs/文档编写规范/README.md) 中的通用约定
- 涉及需求文档时，请遵循 [docs/文档编写规范/需求编写规范.md](/home/ristial/projects/agent/OpsPilot/docs/文档编写规范/需求编写规范.md)

## 8. 缺陷处理

### 8.1 缺陷修复说明文档（强制）

每个缺陷都必须记录在缺陷目录中，推荐目录结构如下：

```text
bugs/
  001-仓库分支无法创建/
    001-仓库分支无法创建-缺陷说明.md
    001-仓库分支无法创建-缺陷分析.md
    001-仓库分支无法创建-修复总结.md
```

每个缺陷目录至少应包含：

- 缺陷说明文档
- 缺陷分析文档
- 修复总结文档

### 8.2 存放规则

- 请遵循 [docs/文档编写规范/README.md](/home/ristial/projects/agent/OpsPilot/docs/文档编写规范/README.md) 中的通用约定
- 缺陷文档结构与命名请遵循 [docs/文档编写规范/Bug缺陷处理文档编写规范.md](/home/ristial/projects/agent/OpsPilot/docs/文档编写规范/Bug缺陷处理文档编写规范.md)

## 9. 推荐工作方式

对这个仓库，较稳妥的顺序通常是：

1. 先定位改动落在 `interfaces`、`application`、`ai/workflows`、`infra` 中哪一层。
2. 再确认是否牵涉工具注册、配置项、示例程序或知识文档。
3. 实现后优先跑最短路径验证，而不是一上来跑整套流程。
4. 最后检查是否破坏了启动装配链路 `bootstrap.New(...)`。

补充：

- 如果修改 `internal/ai/workflows` 下的装配逻辑，优先保持“启动时构建、运行时复用”的约定，不要把高成本的 `Compile`、`NewPlanner`、`NewExecutor`、`NewReplanner` 等重新放回请求路径。
