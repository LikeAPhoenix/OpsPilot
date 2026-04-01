# OpsPilot

一个面向智能运维场景的 Go 后端服务，提供聊天问答、知识库检索与索引、告警分析（AIOps）能力。

项目入口在 `cmd/server/main.go`，装配链路在 `internal/bootstrap/app.go`。

## 项目简介

OpsPilot 通过大模型工作流把三类能力整合在一个服务中：

- 面向运维问答的会话式聊天（支持普通问答和 SSE 流式返回）
- 面向内部文档的知识索引与向量检索（Milvus）
- 面向 Prometheus 活跃告警的 AIOps 分析（可接入 MCP 日志工具）

适合用于：

- 运维助手/智能排障平台后端
- AI Agent + RAG + 工具调用的工程化示例
- 面试或简历中展示“LLM 在真实后端中的落地能力”

## 项目背景 / 业务场景

在运维场景中，常见痛点是：

- 告警信息分散在监控、日志、文档多个系统中
- 新人排障依赖经验，知识传承成本高
- 只做聊天机器人无法直接消费告警和工具结果

OpsPilot 的设计目标是将“文档知识 + 监控告警 + 日志工具 + 会话记忆”统一到一条后端服务链路，支持从“提问”到“分析建议”的完整闭环。

## 核心功能

- 用户认证与统一鉴权（JWT）
- 聊天问答：`POST /api/chat`
- 流式聊天：`POST /api/chat_stream`（SSE）
- 知识文件上传并即时索引：`POST /api/upload`
- AIOps 告警分析：`POST /api/ai_ops`
- 知识检索与重排（Milvus + Embedding）
- MCP 工具接入（日志工具加载失败时自动降级）
- 会话短期记忆策略：最近轮次 + 摘要压缩 + Redis TTL

## 项目亮点

1. **启动期预编译工作流**
   - `chat`、`knowledgeindex`、`aiopsplan` 在服务启动时完成工作流构建，避免请求路径上的高成本初始化。
2. **服务降级策略清晰**
   - MCP 日志工具不可用时，系统降级为“无日志工具模式”，不阻塞主服务启动。
3. **多模型分工**
   - `think_chat_model` 与 `quick_chat_model` 分别用于规划/执行等不同阶段，兼顾质量和延迟。
4. **会话记忆工程化**
   - Redis 存储短期上下文，超过阈值触发摘要压缩，控制上下文长度与成本。
5. **分层架构明确**
   - `interfaces` / `application` / `ai workflows` / `infra` 边界清晰，便于扩展与测试。

## 技术栈

- 语言与框架
  - Go 1.25+
  - Gin（HTTP API）
  - GORM（MySQL）
  - go-redis/v9（Redis）
- AI 与工作流
  - CloudWeGo Eino / Eino-Ext
  - Ark / OpenAI Provider（按配置切换）
  - MCP 工具协议（日志工具）
- 检索与索引
  - Milvus（向量库）
  - Markdown Loader + Splitter
- 工程能力
  - Viper（配置加载 + 环境变量覆盖）
  - Zap（结构化日志）
  - Docker Compose（Milvus 依赖栈）

## 项目架构 / 设计说明

### 分层职责

- `internal/interfaces/http`：HTTP Handler、DTO、SSE、路由
- `internal/application`：应用服务编排，不关心底层实现细节
- `internal/ai/workflows`：聊天、知识索引、AIOps、会话摘要工作流
- `internal/ai/registry`：不同场景下的工具注册
- `internal/infra`：配置、模型 Provider、Milvus、MCP、存储、日志

### 启动装配链路

`bootstrap.New(...)` 会完成：

1. 加载配置（配置文件 + `OPSPILOT_` 环境变量覆盖）
2. 初始化日志、JWT、Snowflake
3. 连接 MySQL / Redis / Milvus
4. 注册模型 Provider，创建 Chat Model 与 Embedding
5. 构建检索器、索引器、文档加载与切分器
6. 注册 Chat/AIOps 工具，构建各工作流
7. 装配服务层和 HTTP 路由

## 目录结构

```text
OpsPilot/
├── cmd/server/                    # 服务入口
├── configs/                       # 配置文件
├── deploy/docker/                 # Docker 部署（Milvus 依赖栈）
├── docs/knowledge/                # 默认知识库文档目录（Markdown）
├── examples/                      # 示例程序（chat/knowledge/recall/aiops/...）
├── internal/
│   ├── ai/
│   │   ├── registry/              # 工具注册
│   │   ├── tools/                 # 工具定义
│   │   └── workflows/             # 工作流实现
│   ├── application/               # 应用服务层
│   ├── bootstrap/                 # 统一装配入口
│   ├── infra/                     # 基础设施层
│   └── interfaces/http/           # HTTP 接口层
├── pkg/                           # 通用库（JWT、Snowflake）
└── scripts/init.sql               # MySQL 初始化脚本
```

## 快速开始

### 环境要求

- Go 1.25+
- MySQL 8+
- Redis 6+
- Milvus 2.5+
- 可用的大模型与 Embedding API（如 Ark/OpenAI）
- Prometheus HTTP API（AIOps 需要）
- 可选：MCP 日志服务（不可用时可降级）

### 安装依赖

```bash
go mod download
```

### 配置说明

1. 复制示例配置：

```bash
cp configs/config_example.yaml configs/config.yaml
```

2. 按实际环境修改 `configs/config.yaml`，重点关注：

- `think_chat_model` / `quick_chat_model`：聊天模型配置
- `embedding_model`：向量化模型配置
- `milvus.*`：向量库连接与库表名
- `mysql.dsn`：用户与认证数据存储
- `redis.*`：会话短期记忆存储
- `prometheus.base_url`：告警来源
- `mcp.url`：日志工具服务地址（SSE）
- `auth.jwt_secret`：JWT 签名密钥
- `session.*`：记忆保留与压缩策略

3. 通过环境变量覆盖（前缀 `OPSPILOT_`）：

```bash
export OPSPILOT_SERVER_ADDRESS=":6872"
export OPSPILOT_MYSQL_DSN="root:root1234@tcp(127.0.0.1:3306)/opspilot?charset=utf8mb4&parseTime=True&loc=Local"
export OPSPILOT_REDIS_ADDRESS="127.0.0.1:6379"
export OPSPILOT_AUTH_JWT_SECRET="change-me-in-prod"
```

> 规则：配置路径中的 `.` 会映射为 `_`，例如 `auth.jwt_secret -> OPSPILOT_AUTH_JWT_SECRET`。

### 数据初始化（MySQL）

```bash
mysql -uroot -p < scripts/init.sql
```

### 启动方式

#### 方式一：本地直接启动（推荐开发）

1. 启动 Milvus（见下方 Docker 方式）
2. 确保 MySQL、Redis、Prometheus、模型配置可用
3. 启动服务：

```bash
go run ./cmd/server
```

默认监听地址来自配置：`server.address`（示例是 `:6872`）。

#### 方式二：Docker 启动 Milvus 依赖栈

项目内 `deploy/docker/docker-compose.yml` 主要用于启动 Milvus 及其依赖（etcd / minio / attu）：

```bash
cd deploy/docker
./docker.sh up -d
```

停止：

```bash
cd deploy/docker
./docker.sh down
```

> 注意：当前 compose **不包含** MySQL、Redis、Prometheus 和主服务容器，请自行准备或扩展编排文件。

#### 方式三：运行示例程序（快速验证链路）

```bash
# 聊天链路
go run ./examples/chat

# 索引 docs/knowledge 下 markdown
go run ./examples/knowledge ./docs/knowledge

# 检索验证
go run ./examples/recall "服务下线是什么原因"

# AIOps 示例（会自动探测并按需启动 mock）
go run ./examples/aiops

# 工具演示
go run ./examples/tools
```

## 接口说明

### 统一响应结构

```json
{
  "message": "OK",
  "data": {}
}
```

错误时：`message` 为错误信息，`data` 为 `null`。

### 鉴权规则

- 无需鉴权：
  - `POST /api/auth/register`
  - `POST /api/auth/login`
- 需要鉴权（Bearer Token）：
  - `POST /api/chat`
  - `POST /api/chat_stream`
  - `POST /api/upload`
  - `POST /api/ai_ops`

请求头示例：

```http
Authorization: Bearer <access_token>
```

### 认证接口

#### 1) 注册

```bash
curl -X POST http://127.0.0.1:6872/api/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "username": "demo",
    "password": "demo123456"
  }'
```

#### 2) 登录

```bash
curl -X POST http://127.0.0.1:6872/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "username": "demo",
    "password": "demo123456"
  }'
```

### 业务接口

#### 1) 普通聊天

> 当前请求字段名为 `Id` 与 `Question`（注意大小写，和常见驼峰不同）。

```bash
curl -X POST http://127.0.0.1:6872/api/chat \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <access_token>" \
  -d '{
    "Id": "",
    "Question": "现在有哪些告警需要优先处理？"
  }'
```

#### 2) 流式聊天（SSE）

```bash
curl -N -X POST http://127.0.0.1:6872/api/chat_stream \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <access_token>" \
  -d '{
    "Id": "",
    "Question": "请分步骤分析这条告警"
  }'
```

#### 3) 上传知识文档并索引

```bash
curl -X POST http://127.0.0.1:6872/api/upload \
  -H "Authorization: Bearer <access_token>" \
  -F "file=@./docs/knowledge/告警处理手册.md"
```

#### 4) 触发 AIOps 分析

```bash
curl -X POST http://127.0.0.1:6872/api/ai_ops \
  -H "Authorization: Bearer <access_token>"
```

## 数据库 / 缓存 / 消息队列说明

### MySQL

用途：

- 本地账号体系（用户注册/登录）

当前核心表：

- `users`（见 `scripts/init.sql`）

### Redis

用途：

- 会话短期记忆存储
- 摘要压缩后的会话状态持久化（带 TTL）

关键配置：

- `session.retain_turns`：保留最近明细轮次
- `session.compact_threshold_turns`：超过阈值触发摘要
- `session.ttl`：会话过期时间

### Milvus

用途：

- Markdown 知识文档向量索引与召回

关键配置：

- `milvus.default_db`
- `milvus.database`
- `milvus.collection`

### 消息队列

- 当前版本未引入 MQ（如 Kafka/RabbitMQ）。
- 告警数据由 Prometheus HTTP API 拉取，日志通过 MCP 工具调用获取。

## 日志与监控

### 日志

- 使用 Zap 输出日志
- 可配置级别、编码、落盘文件：`logger.*`
- 示例日志文件路径：`./logs/opspilot.log`

### 监控与外部依赖

- AIOps 依赖 Prometheus `/api/v1/alerts`
- 可选接入 MCP 日志服务（SSE）
- 项目内提供了本地 mock：

```bash
# Prometheus mock
go run ./examples/prometheusmock

# MCP 日志 mock
go run ./examples/mcplogmock
```

## 部署说明

### 开发环境建议

- 服务进程：直接 `go run ./cmd/server`
- 依赖服务：
  - Milvus 通过 `deploy/docker` 启动
  - MySQL / Redis / Prometheus 可用本地服务、容器或测试环境

### 生产环境建议

- 使用环境变量覆盖敏感配置（API Key、JWT Secret、DSN）
- 使用独立配置中心或 Secret 管理系统（如 K8s Secret）
- 对外暴露前加网关限流与 TLS
- 将 `mcp.url`、`prometheus.base_url` 指向可用内网地址

## 常见问题

### 1. 服务启动时报错：`read config configs/config.yaml`

原因：未提供配置文件或路径不对。  
处理：复制 `configs/config_example.yaml` 为 `configs/config.yaml` 并补齐配置。

### 2. 启动失败：Milvus / MySQL / Redis 连接错误

原因：依赖未启动或配置不匹配。  
处理：

- 先确认端口与地址
- 用最小命令测试连接
- 再启动服务

### 3. 调用业务接口返回 401

原因：未携带 `Authorization: Bearer <token>` 或 token 过期。  
处理：重新登录获取 `accessToken`，并确认请求头格式。

### 4. 上传文件成功但检索不到内容

原因：仅支持 Markdown 索引，或索引流程未成功。  
处理：确认文件后缀是 `.md`，检查服务日志中的索引错误。

### 5. AIOps 无法返回有效分析

原因：Prometheus 无活跃告警、MCP 工具不可用、模型配置异常。  
处理：先运行 `examples/prometheusmock` 和 `examples/mcplogmock` 验证链路。

## 后续优化方向

- 补充 OpenAPI/Swagger 文档与接口自动校验
- 增加单元测试与集成测试（当前 `go test ./...` 以编译回归为主）
- 提供完整的一键化 docker-compose（含 MySQL/Redis/Prometheus/服务本体）
- 引入可观测性（Tracing、Metrics、慢查询分析）
- 增加多租户隔离与更细粒度权限模型
- 对 `chat` / `chat_stream` 请求字段命名进行版本化优化（兼容升级）

## 贡献指南

欢迎提交 Issue / PR。建议流程：

1. Fork 并新建功能分支
2. 完成开发后执行格式化与回归检查：

```bash
gofmt -w <changed-files>
go test ./...
```

3. 补充必要文档（需求实现总结或缺陷文档，遵循 `docs/文档编写规范`）
4. 提交 PR，描述变更动机、影响范围和验证步骤

## License

仓库未显式提供 License 文件。若计划开源分发，建议补充标准开源许可证（如 MIT / Apache-2.0）。
