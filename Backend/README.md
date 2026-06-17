# Go Fiber Refactor (Control + Executor)

这是 Araneae 的 Go Fiber 重构版本，包含：

- Control 服务：用户认证、项目仓库上传、任务定义、定时触发、手动/API 触发
- Executor 服务：RabbitMQ 消费任务、gRPC 拉取制品、执行任务并回调结果
- Service to Service 通信：
  - RabbitMQ：Control -> Executor 任务下发
  - gRPC：Executor -> Control 拉取上传的项目制品

## 目录

- cmd/control: 控制端入口
- cmd/executor: 执行端入口
- internal/control: 控制端核心逻辑
- internal/executor: 执行端核心逻辑
- proto: gRPC 协议
- gen/pb: 生成代码
- examples/simple-job: 可上传的示例项目
- scripts/e2e.sh: 上传 -> 定时任务 -> 自动执行闭环脚本

## 默认端口

- Control HTTP: 8180
- Control gRPC: 9190
- Executor HTTP: 4280
- RabbitMQ: 5672

## 启动依赖

在 GoRefactor 目录执行：

- docker compose up -d rabbitmq

## 启动服务

终端 1：

- go run ./cmd/control

终端 2：

- go run ./cmd/executor

## 跨服务器部署（1 控制节点 + N 工作节点）

如果你要将控制节点与工作节点部署在不同服务器，请使用以下模板与指南：

- 部署指南：deploy/MULTI_NODE_DEPLOYMENT.md
- 控制节点环境变量模板：deploy/control.env.example
- 工作节点环境变量模板：deploy/executor.env.example
- systemd 模板：
  - deploy/systemd/araneae-control.service
  - deploy/systemd/araneae-executor.service

核心要点：

- 控制节点对工作节点开放 gRPC 端口（默认 9190）。
- 控制节点 HTTP（默认 8180）需可被工作节点回调访问。
- 所有节点共享同一个 RabbitMQ。
- 通过任务里的 node_queue 字段，把任务路由到对应 EXECUTOR_QUEUE。
- 工作节点启动会生成节点密钥并保存到 EXECUTOR_NODE_KEY_FILE（或从 EXECUTOR_NODE_KEY/EXECUTOR_NODE_KEY_FILE 读取）；日志会输出密钥文件路径。
- 控制节点注册工作节点时必须填写该节点密钥；注册成功后，Control gRPC 仅接受已注册节点密钥请求。

## 前端接入（Front）

在 Front 的环境变量中配置：

- VITE_API_FLAVOR=go
- VITE_BACKEND_BASE_URL=http://localhost:8180

这样 Front 会使用 Go 控制端 API（/api/v1）。

## 核心 API

1. 登录
- POST /api/v1/auth/login
- body: {"username":"admin","password":"<INIT_ADMIN_PASSWORD>"}

2. 创建项目
- POST /api/v1/projects
- header: Authorization: Bearer <token>

3. 上传项目代码包
- POST /api/v1/projects/:id/upload
- multipart: file=<zip>

4. 创建任务（支持定时）
- POST /api/v1/tasks
- body 示例：
  {
    "name": "demo",
    "project_id": "...",
    "version_id": "...",
    "entry_command": "bash run.sh",
    "cron_expr": "*/30 * * * * *",
    "node_queue": "default"
  }

5. 手动/API 触发任务
- POST /api/v1/tasks/:id/trigger

6. 查询运行记录
- GET /api/v1/tasks/:id/runs

## 权限与安全

- 内置 JWT 鉴权
- 角色控制：admin/operator/viewer
- 初始管理员密码来自 INIT_ADMIN_PASSWORD（生产环境必须显式设置强密码）
- 执行端回调 Control 使用 X-Execution-Key
- 工作节点 HTTP 端口请求需要携带 X-Node-Key
- Control gRPC 拉制品请求需要携带节点密钥元数据（x-node-key）

## 一键闭环验证

- chmod +x scripts/e2e.sh
- ./scripts/e2e.sh

脚本会完成：

1. 启动 RabbitMQ
2. 启动 Control 与 Executor
3. 登录并创建项目
4. 上传示例 zip
5. 创建定时任务
6. 等待自动触发并打印运行结果

## 可选：RabbitMQ 实链路集成测试

该测试覆盖 Control 端 `trigger -> RabbitMQ publish -> callback -> run 状态更新`。

本地运行：

- docker compose up -d rabbitmq
- RABBITMQ_URL=amqp://guest:guest@localhost:5672/ go test -tags=integration ./internal/control -run TestControlIntegration_TriggerQueueCallbackFlow -v

GitHub Actions：

- 使用独立工作流 `GoRefactor Integration`（手动触发，不影响主 CI）。

## Araneae Sink -> HashSlip（新增）

Executor 现在支持在任务运行目录中收集 sink 接收区文件并自动转存到 HashSlip：

- 接收区默认目录：`.araneae/sink`
- 文件格式：`*.jsonl`，每行一个 envelope
- 支持类型：`timeseries`、`text`、`structured`

### Envelope 示例

```json
{"type":"timeseries","data":{"source":"araneae.fed","metric":"fed_overnight_rate","timestamp":"2026-06-17T15:00:00Z","value":5.25,"hash_key":"fed_overnight_rate_daily","bucket_date":"2026-06-17"}}
```

### Executor 环境变量

- `EXECUTOR_SINK_ENABLED`：是否启用 sink 转存（默认 `true`）
- `EXECUTOR_SINK_STRICT`：转存失败是否让任务失败（默认 `false`）
- `HASHSLIP_BASE_URL`：HashSlip 地址（例如 `http://hashslip:8106`）
- `HASHSLIP_TIMESERIES_PATH`：默认 `/api/v1/timeseries/records`
- `HASHSLIP_TEXT_PATH`：默认 `/api/v1/text/records`
- `HASHSLIP_STRUCTURED_PATH`：默认 `/api/v1/data`
- 可选 BasaltPass S2S：`BASALTPASS_TOKEN_URL` + `BASALTPASS_OAUTH_CLIENT_ID` + `BASALTPASS_OAUTH_CLIENT_SECRET`

### Python 爬虫使用方式

Executor 会自动在任务运行目录注入 `araneae_sink.py`，脚本可直接：

- `import araneae_sink`
- 调用 `araneae_sink.emit_timeseries(...)` / `emit_text(...)` / `emit_structured(...)`

示例脚本：`examples/simple-job/fed_overnight_rate.py`

联调脚本（本地 RabbitMQ + HashSlip）：

- `python scripts/e2e_sink_hashslip.py`
