# Araneae 部署说明

本文档基于当前仓库结构（Backend + Frontend）提供可直接执行的 Docker 部署方法，覆盖以下两类场景：

- 单机一体化部署：前端 + 控制端 + 运行端 + RabbitMQ
- 多机拆分部署：控制端机器（可含前端）与运行端机器分离

## 1. 端口与组件

- Frontend: 5109
- Control HTTP: 8180
- Control gRPC: 9190
- Executor HTTP: 4280
- RabbitMQ: 5672
- RabbitMQ 管理台: 15672

## 2. 部署方式总览

仓库根目录提供三份 Compose 文件：

- docker-compose.yml: 单机一体化
- docker-compose.control.yml: 控制端机器（controlnode + rabbitmq + front）
- docker-compose.executor.yml: 运行端机器（executionnode）

## 3. 单机一体化部署

在仓库根目录执行：

```bash
docker compose up -d --build
```

验活：

```bash
curl http://127.0.0.1:5109
curl http://127.0.0.1:8180/healthz
curl http://127.0.0.1:4280/healthz
```

## 4. 多机拆分部署

### 4.1 机器 A（控制端机器）

1. 在仓库根目录复制环境模板：

```bash
cp .env.control.example .env.control
```

2. 编辑 .env.control，至少确认：

- RABBITMQ_USERNAME
- RABBITMQ_PASSWORD
- EXECUTION_CALLBACK_KEY
- FRONT_VITE_BACKEND_BASE_URL
- CONTROL_CORS_ALLOW_ORIGINS

3. 启动控制端机器服务：

```bash
docker compose -f docker-compose.control.yml --env-file .env.control up -d --build
```

### 4.2 机器 B（运行端机器）

1. 在仓库根目录复制环境模板：

```bash
cp .env.executor.example .env.executor
```

2. 编辑 .env.executor，至少确认：

- EXECUTOR_RABBITMQ_URL
- EXECUTOR_CONTROL_GRPC_TARGET
- EXECUTOR_CONTROL_HTTP_BASE
- EXECUTION_CALLBACK_KEY
- EXECUTOR_QUEUE

3. 启动运行端：

```bash
docker compose -f docker-compose.executor.yml --env-file .env.executor up -d --build
```

## 5. 分机部署关键约束

1. 所有运行端与控制端必须使用同一个 EXECUTION_CALLBACK_KEY。
2. 运行端必须能访问控制端的 9190 和 8180。
3. 运行端必须能访问共享 RabbitMQ 的 5672。
4. 任务下发时 node_queue 需与运行端 EXECUTOR_QUEUE 对齐。
5. 前端构建地址 FRONT_VITE_BACKEND_BASE_URL 应填写用户可访问的控制端地址，不建议在分机部署时使用 localhost。

## 6. 推荐网络与安全策略

- 控制端机器仅对必要网段开放 8180 与 9190。
- RabbitMQ 5672 只对控制端与运行端网段开放。
- EXECUTION_CALLBACK_KEY 使用高强度随机值并定期轮换。
- 生产环境建议在控制端前放置反向代理与 HTTPS。

## 7. 常用运维命令

单机：

```bash
docker compose ps
docker compose logs -f controlnode executionnode front rabbitmq
docker compose down
```

控制端机器：

```bash
docker compose -f docker-compose.control.yml --env-file .env.control ps
docker compose -f docker-compose.control.yml --env-file .env.control logs -f controlnode front rabbitmq
```

运行端机器：

```bash
docker compose -f docker-compose.executor.yml --env-file .env.executor ps
docker compose -f docker-compose.executor.yml --env-file .env.executor logs -f executionnode
```

## 8. 验收清单

- Frontend 可访问。
- Control /healthz 返回 200。
- Executor /healthz 返回 200。
- 创建任务后可按预期路由到目标 queue 并完成回调。
