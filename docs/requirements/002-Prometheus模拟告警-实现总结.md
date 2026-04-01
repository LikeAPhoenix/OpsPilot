# 002-Prometheus模拟告警-实现总结

## 1. 实现了什么

本次实现新增了一个独立的 Prometheus 模拟告警示例程序，用于在本地没有真实 firing 告警时，为 AIOps 联调提供稳定的 `/api/v1/alerts` 数据源：

- 新增 `examples/prometheusmock/main.go`，启动一个基于 Go 标准库的 HTTP 服务
- 新增内置样例数据 `examples/prometheusmock/testdata/scenarios.json`
- 支持 `single`、`multiple`、`empty` 三种告警场景
- 响应结构兼容当前 `query_prometheus_alerts` 工具的解析逻辑

## 2. 与需求的对应关系

对应 [002-Prometheus模拟告警-需求.md](/home/ristial/projects/agent/OpsPilot/docs/requirements/002-Prometheus模拟告警-需求.md)：

- “提供独立可运行的模拟告警程序”：已完成，程序位于 `examples/prometheusmock`
- “默认返回至少一条 firing 告警”：已完成，默认场景为 `single`
- “支持切换单告警、多告警和空告警场景”：已完成，通过 `-scenario` 参数和查询字符串切换
- “保持主服务装配逻辑不变”：已完成，本次未修改 `cmd/server`、`bootstrap` 和现有 AIOps 路径

## 3. 关键实现点

### 3.1 示例程序位置

- 按仓库边界约定，将模拟服务放在 [main.go](/home/ristial/projects/agent/OpsPilot/examples/prometheusmock/main.go)
- 该程序是独立联调工具，不进入 `internal/`、`application/` 或正式服务入口

### 3.2 Prometheus 兼容响应

- 模拟服务提供 `GET /api/v1/alerts`
- 返回体包含 `status` 和 `data.alerts`
- 每条告警包含 `labels`、`annotations`、`state`、`activeAt`、`value`
- `activeAt` 不是写死时间，而是根据样例中的 `activeAgo` 在运行时动态生成，避免出现过旧时间戳

### 3.3 场景切换

- 默认场景通过 `-scenario` 指定
- 运行时也可通过 `GET /api/v1/alerts?scenario=multiple` 临时覆盖
- `empty` 场景直接返回空告警数组，便于验证“无活跃告警”路径

### 3.4 内置样例数据

- 样例数据放在 [scenarios.json](/home/ristial/projects/agent/OpsPilot/examples/prometheusmock/testdata/scenarios.json)
- 通过 `go:embed` 打包进程序，运行时无需额外准备文件

## 4. 已知限制或待改进点

- 当前只模拟了 `alerts` 接口，没有覆盖 Prometheus 其他 API
- 样例数据字段只覆盖当前工具实际依赖的最小集合，不是完整 Prometheus 告警对象
- 如果后续需要动态增删告警或从外部文件热加载场景，可以再扩展管理接口或文件参数

## 5. 验证结果

本次改动后已执行：

```bash
gofmt -w examples/prometheusmock/main.go
go test ./examples/prometheusmock
go run ./examples/prometheusmock -addr 127.0.0.1:19091
curl http://127.0.0.1:19091/api/v1/alerts
curl "http://127.0.0.1:19091/api/v1/alerts?scenario=empty"
```

验证结果：

- 示例程序代码格式化通过
- `go test ./examples/prometheusmock` 编译通过
- 默认场景返回至少一条 firing 告警
- `empty` 场景返回空告警数组

## 6. 实现总结

这次改动补齐了 AIOps 本地联调缺少活跃告警输入的问题，但没有把测试逻辑混入主服务。模拟服务被明确收敛在 `examples/` 目录内，既符合仓库边界，也能直接服务于 `query_prometheus_alerts` 的联调验证。
