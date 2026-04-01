# Araneae 开发手册（当前 Go 架构）

本文档描述当前仓库的实际开发架构与工作流。历史 Django/Flask 结构已不再是主线实现。

## 1. 架构总览

Araneae 采用前后端分离与主控-执行分布式架构：

- Frontend（Vue + Vite）：管理界面。
- Control（Go Fiber）：认证、项目与版本管理、任务创建、调度、下发、回调接收。
- Executor（Go Fiber）：消费任务、拉取制品、执行命令、回调结果。
- RabbitMQ：Control 到 Executor 的任务消息通道。
- gRPC：Executor 从 Control 拉取制品。

默认端口：

- Frontend: 5109
- Control HTTP: 8180
- Control gRPC: 9190
- Executor HTTP: 4280
- RabbitMQ: 5672

## 2. 目录结构

```text
Araneae/
├─ Backend/
│  ├─ cmd/
│  │  ├─ control/            # Control 入口
│  │  └─ executor/           # Executor 入口
│  ├─ internal/
│  │  ├─ control/            # Control 核心逻辑与路由
│  │  ├─ executor/           # Executor 核心逻辑与执行流程
│  │  └─ common/             # 共享配置与通用能力
│  ├─ proto/                 # gRPC 协议定义
│  ├─ gen/pb/                # gRPC 生成代码
│  └─ data/                  # 本地开发数据目录
├─ Frontend/
│  ├─ src/
│  └─ Dockerfile
├─ docker-compose.yml            # 单机一体化部署
├─ docker-compose.control.yml    # 控制端机器部署
├─ docker-compose.executor.yml   # 运行端机器部署
└─ README.md
```

## 3. 运行模式

### 3.1 单机一体化（推荐本地联调）

在仓库根目录执行：

```bash
docker compose up -d --build
```

验证：

```bash
curl http://127.0.0.1:8180/healthz
curl http://127.0.0.1:4280/healthz
curl http://127.0.0.1:5109
```

### 3.2 分机部署（控制端与运行端分离）

请优先参考部署文档：DEPLOYMENT 与 Backend/deploy/MULTI_NODE_DEPLOYMENT。

关键约束：

1. EXECUTION_CALLBACK_KEY 在 Control 与所有 Executor 上必须一致。
2. Executor 必须可访问 Control 的 8180 与 9190。
3. Executor 必须可访问共享 RabbitMQ（5672）。
4. 任务 node_queue 与 Executor 的 EXECUTOR_QUEUE 一致时才会命中目标节点。

## 4. 核心执行链路

一次任务执行的主路径：

1. 用户通过 Frontend 调用 Control API 创建任务或调度。
2. Control 将任务发布到 RabbitMQ 交换机（按 routing key 分发）。
3. 对应 queue 的 Executor 消费消息。
4. Executor 通过 gRPC 向 Control 拉取目标版本制品。
5. Executor 在本地工作目录解压并执行 entry_command。
6. Executor 回调 Control 的 runs callback 接口上报状态、输出、退出码。
7. Control 更新运行记录，并在需要时推进链式调度。

## 5. 关键配置项

Control 常用：

- CONTROL_HTTP_ADDR（默认 :8180）
- CONTROL_GRPC_ADDR（默认 :9190）
- CONTROL_DB_PATH
- ARTIFACT_ROOT
- RABBITMQ_URL
- EXECUTION_CALLBACK_KEY
- CONTROL_CORS_ALLOW_ORIGINS

Executor 常用：

- EXECUTOR_HTTP_ADDR（默认 :4280）
- EXECUTOR_DB_PATH
- EXECUTOR_WORKDIR
- EXECUTOR_QUEUE（默认 default）
- RABBITMQ_URL
- CONTROL_GRPC_TARGET
- CONTROL_HTTP_BASE
- EXECUTION_CALLBACK_KEY

前端构建时：

- FRONT_VITE_BACKEND_BASE_URL
- FRONT_VITE_API_FLAVOR（go）

说明：前端镜像是静态构建产物，分机部署时应将 FRONT_VITE_BACKEND_BASE_URL 指向用户可访问的 Control 地址，不建议用 localhost。

## 6. 本地开发建议

### 6.1 仅调后端

```bash
cd Backend
docker compose up -d rabbitmq
go run ./cmd/control
go run ./cmd/executor
```

### 6.2 前后端联调

```bash
cd Frontend
npm install
npm run dev -- --port 5109
```

并确保前端 API 地址指向 Control（例如 [http://localhost:8180](http://localhost:8180)）。

## 7. 排障速查

1. Executor 重启循环且日志出现 AMQP 403：
   - 通常是 RABBITMQ_URL 用户名/密码不匹配。
2. Frontend 可打开但接口全部失败：
   - 通常是前端构建时 BACKEND 地址配置错误。
3. 任务一直不被某节点消费：
   - 检查 node_queue 与 EXECUTOR_QUEUE 是否一致。
4. 回调不生效：
   - 检查 EXECUTION_CALLBACK_KEY 是否一致，以及 Control HTTP 地址是否可达。

## 8. 当前能力边界

当前实现已支持按队列路由的多 Executor 部署。

暂未内置完整的节点注册心跳与在线节点清单能力，节点管理更多依赖队列约定与运行监控。
