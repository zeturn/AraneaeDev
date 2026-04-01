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
NODE_KEY=$(tr -d '\r\n' < Backend/data/executor.node.key)
curl -H "X-Node-Key: $NODE_KEY" http://127.0.0.1:4280/healthz
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

### 4.3 鉴权模式下添加工作节点（必须执行）

从当前版本开始，Control 与 Executor 之间默认采用节点密钥鉴权。工作节点只启动还不够，必须完成“注册绑定”。

#### 步骤 1：获取工作节点密钥

优先级如下：

1. 若你在 .env.executor 中显式配置了 EXECUTOR_NODE_KEY，直接使用该值。
2. 若未显式配置，Executor 启动后会自动生成密钥并写入 EXECUTOR_NODE_KEY_FILE。

默认文件位置（按本仓库 Compose）：

- 运行端机器仓库路径：Backend/data/executor.node.key

读取示例：

```bash
NODE_KEY=$(tr -d '\r\n' < Backend/data/executor.node.key)
echo "$NODE_KEY"
```

#### 步骤 2：确认工作节点 HTTP 已开启鉴权

不带密钥应返回 401：

```bash
curl -i http://<executor-ip>:4280/healthz
```

带密钥应返回 200：

```bash
curl -H "X-Node-Key: $NODE_KEY" http://<executor-ip>:4280/healthz
```

#### 步骤 3：在控制端注册该工作节点

先登录拿 token：

```bash
TOKEN=$(curl -s http://<control-ip>:8180/api/v1/auth/login \
	-H 'Content-Type: application/json' \
	-d '{"username":"admin","password":"admin123"}' | jq -r '.token')
```

注册节点（pair_key 必填）：

```bash
curl -s http://<control-ip>:8180/api/v1/nodes/register/ \
	-H "Authorization: Bearer $TOKEN" \
	-H 'Content-Type: application/json' \
	-d "{\"ip\":\"<executor-ip>\",\"name\":\"executor-a\",\"port\":4280,\"grpc_port\":9190,\"pair_key\":\"$NODE_KEY\"}"
```

说明：

- Control 会用 pair_key 主动调用 Executor 的 /node/verify 做握手。
- 握手成功后，Control 才会保存该节点，并允许该节点通过 gRPC 拉取制品。

#### 步骤 4：在前端添加节点（可选）

前端路径：Aprons -> 节点 -> 创建节点。

现在“创建节点”表单中必须填写“节点密钥”，其值与上面的 NODE_KEY 相同。

#### 步骤 5：验证任务能路由到该节点

创建任务时把 node_queue 设为该节点队列（通常与 EXECUTOR_QUEUE 一致），触发后检查任务运行记录为 success。

## 5. 分机部署关键约束

1. 所有运行端与控制端必须使用同一个 EXECUTION_CALLBACK_KEY。
2. 运行端必须能访问控制端的 9190 和 8180。
3. 运行端必须能访问共享 RabbitMQ 的 5672。
4. 任务下发时 node_queue 需与运行端 EXECUTOR_QUEUE 对齐。
5. 前端构建地址 FRONT_VITE_BACKEND_BASE_URL 应填写用户可访问的控制端地址，不建议在分机部署时使用 localhost。
6. 每个运行端都必须完成节点注册，并提供正确 pair_key（节点密钥）。
7. Executor 的开放 HTTP 端口请求需带 X-Node-Key，匿名请求会被拒绝。

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
- Executor /healthz 在带 X-Node-Key 时返回 200。
- Executor /healthz 在不带 X-Node-Key 时返回 401。
- 节点注册接口 /api/v1/nodes/register/ 在携带正确 pair_key 时成功。
- 创建任务后可按预期路由到目标 queue 并完成回调。
