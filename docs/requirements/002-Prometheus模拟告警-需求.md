# 0. 文件修改记录表

| 修改人 | 修改时间 | 修改内容 |
| ------ | -------- | -------- |
| Codex | 2026-03-31 | 初始版本，新增 Prometheus 模拟告警示例程序需求 |

# 1. 背景（Why）

当前 `AIOps` 分析流程依赖 Prometheus `/api/v1/alerts` 返回活跃告警。当联调环境没有真实 firing 告警时，`query_prometheus_alerts` 工具只能拿到空结果，导致 AIOps 分析难以稳定验证。因此需要提供一个仓库内可直接运行的 Prometheus 模拟告警程序，用于本地联调与演示。

# 2. 目标（What，必须可验证）

- [ ] 提供一个独立可运行的 Prometheus 模拟告警程序，能够返回 Prometheus 兼容的 `/api/v1/alerts` 响应
- [ ] 默认返回至少一条 firing 告警，便于 `examples/aiops` 和 `/api/ai_ops` 直接联调
- [ ] 支持通过启动参数切换不同告警场景，至少包含单告警、多告警和空告警场景
- [ ] 保持主服务装配逻辑、工具名和 HTTP 接口路径不变

# 3. 非目标（Explicitly Out of Scope）

- 不实现完整 Prometheus 查询语言或其他 Prometheus API
- 不改造 `query_prometheus_alerts` 工具的接口定义
- 不将模拟服务嵌入 `cmd/server` 主服务进程
- 不引入数据库、第三方缓存或额外基础设施依赖

# 4. 使用场景 / 用户路径

1. 开发者在仓库根目录运行 Prometheus 模拟告警程序。
2. 程序监听本地 HTTP 端口，并暴露 `GET /api/v1/alerts`。
3. 开发者将 `configs/config.yaml` 中的 `prometheus.base_url` 指向该模拟服务地址，或通过环境变量覆盖。
4. 开发者调用 `go run ./examples/aiops` 或启动主服务后请求 `/api/ai_ops`。
5. `query_prometheus_alerts` 工具从模拟服务获取 firing 告警，AIOps 工作流继续执行后续分析。

# 5. 功能需求清单（Checklist）

- [ ] 在 `examples/` 目录下新增独立示例程序，不污染 `internal/` 或 `cmd/server/`
- [ ] 模拟服务提供 `GET /api/v1/alerts` 路由，响应结构兼容当前工具解析逻辑
- [ ] 程序支持通过命令行参数指定监听地址
- [ ] 程序支持通过命令行参数指定默认告警场景
- [ ] 内置静态告警样例数据，并在运行时生成合理的 `activeAt` 时间
- [ ] 支持 `empty` 场景，用于验证无活跃告警时的行为
- [ ] 输出清晰的启动日志，提示可用地址和场景名称

# 6. 约束条件（非常关键）

- 技术约束：继续使用 Go 标准库实现 HTTP 服务，不新增额外 Web 框架依赖
- 架构约束：该能力作为联调示例存在于 `examples/`，不下沉到 `internal/application`、`internal/interfaces/http` 或 `internal/bootstrap`
- 安全约束：不得写入任何真实密钥、租户信息或生产地址
- 性能约束：示例程序以本地联调为目标，优先保证可读性和稳定性，不追求高并发优化

# 7. 可修改 / 不可修改项

- ❌ 不可修改：`query_prometheus_alerts` 工具名、主服务 HTTP 路由、现有 AIOps 分析入口语义
- ✅ 可调整：示例程序目录结构、样例数据组织方式、示例程序启动参数和日志内容

# 8. 接口与数据约定（如适用）

- 模拟服务必须提供 `GET /api/v1/alerts`
- 返回体顶层字段至少包含：
  - `status`
  - `data.alerts`
- `data.alerts` 中的每条告警至少包含：
  - `labels`
  - `annotations`
  - `state`
  - `activeAt`
  - `value`
- 示例程序允许提供额外健康检查接口，如 `GET /-/healthy`

# 9. 验收标准（Acceptance Criteria）

- 如果执行 `go run ./examples/prometheusmock`，则程序可以成功启动并监听配置的地址
- 如果请求 `GET /api/v1/alerts`，则返回 JSON 且结构可被 `internal/ai/tools/query_metrics_alerts.go` 正常解析
- 如果默认场景启动后调用 AIOps 相关联调路径，则可以拿到至少一条 firing 告警
- 如果以 `empty` 场景启动后请求 `GET /api/v1/alerts`，则返回空告警列表而不是报错
- 如果传入不存在的场景名称，则程序启动失败并给出明确错误信息

# 10. 风险与已知不确定点

- 当前 AIOps 工作流除了告警接口外，还依赖内部文档和可选日志工具，模拟告警程序只能解决“无告警数据”这一类联调问题
- 模拟服务返回的是样例数据，不代表真实 Prometheus 告警维度的完整字段集
- 如果后续 `query_prometheus_alerts` 工具改为依赖更多 Prometheus 字段，需要同步扩展模拟数据结构

# 11. 非目标

- 不提供 Web UI 或可视化配置页面
- 不实现告警的动态增删改接口
- 不自动修改 `configs/config.yaml`
