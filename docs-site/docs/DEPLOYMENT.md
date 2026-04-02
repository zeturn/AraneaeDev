# Araneae 部署与使用指南

本指南面向当前仓库结构（Backend + Frontend），提供从测试环境到生产环境的完整部署路径，并给出验收与排障方法。

## 1. 组件与端口

- Frontend: 5109
- Control HTTP API: 8180
- Control gRPC: 9190
- Executor HTTP: 4280
- RabbitMQ AMQP: 5672
- RabbitMQ 管理台: 15672

## 2. 部署模式选择

仓库根目录提供以下 Compose 文件：

- docker-compose.yml: 单机一体化（本地联调/测试）
- docker-compose.control.yml: 多机模式的控制端机器（control + rabbitmq + front）
- docker-compose.executor.yml: 多机模式的执行端机器（executor）
- docker-compose.prod.yml: 单机生产模板（强约束、必须显式安全参数）

建议：

- 本地功能验证: 使用 docker-compose.yml
- 正式生产发布（单机）: 使用 docker-compose.prod.yml
- 正式生产发布（多机）: 使用 docker-compose.control.yml + docker-compose.executor.yml

## 3. 先决条件

- Docker 24+ 与 Docker Compose v2
- 机器时间同步（NTP）
- 可访问镜像仓库
- 如果启用 TLS：提前准备证书文件

## 4. 本地单机快速启动（测试环境）

### 4.1 准备最小环境变量

Linux/macOS:

```bash
cat > .env.local <<'EOF'
EXECUTION_CALLBACK_KEY=dev-callback-key-change-this
EOF
```

PowerShell:

```powershell
@"
EXECUTION_CALLBACK_KEY=dev-callback-key-change-this
"@ | Set-Content -Path .env.local
```

### 4.2 启动

```bash
docker compose --env-file .env.local up -d --build
```

### 4.3 验活

```bash
curl http://127.0.0.1:8180/healthz
curl http://127.0.0.1:5109
NODE_KEY=$(tr -d '\r\n' < Backend/data/executor.node.key)
curl -H "X-Node-Key: $NODE_KEY" http://127.0.0.1:4280/healthz
```

## 5. 单机生产部署（推荐使用 docker-compose.prod.yml）

### 5.1 创建生产环境文件

创建 .env.prod，至少包含：

```env
RABBITMQ_USERNAME=<strong-user>
RABBITMQ_PASSWORD=<strong-password>

INIT_ADMIN_PASSWORD=<strong-admin-password>
CONTROL_JWT_SECRET=<long-random-secret>
EXECUTION_CALLBACK_KEY=<long-random-shared-key>

CONTROL_CORS_ALLOW_ORIGINS=https://<frontend-domain>
FRONT_VITE_BACKEND_BASE_URL=https://<frontend-domain-or-api-domain>
FRONT_VITE_API_FLAVOR=go

CONTROL_GRPC_TLS_CERT_FILE=/data/certs/control-grpc.crt
CONTROL_GRPC_TLS_KEY_FILE=/data/certs/control-grpc.key

EXECUTOR_CONTROL_HTTP_BASE=https://<control-domain>
EXECUTOR_CONTROL_GRPC_TLS_SERVER_NAME=<control-domain>
```

说明：

- docker-compose.prod.yml 会强制要求关键变量，不满足会拒绝启动。
- 生产模式默认启用 gRPC TLS，且节点配对默认使用 HTTPS。

### 5.2 启动

```bash
docker compose -f docker-compose.prod.yml --env-file .env.prod up -d --build
```

### 5.3 观察状态

```bash
docker compose -f docker-compose.prod.yml --env-file .env.prod ps
docker compose -f docker-compose.prod.yml --env-file .env.prod logs -f controlnode executionnode front rabbitmq
```

## 6. 多机生产部署（控制端 + 执行端）

## 6.1 机器 A（控制端）

1. 复制模板：

```bash
cp .env.control.example .env.control
```

2. 修改关键参数：

- RABBITMQ_USERNAME / RABBITMQ_PASSWORD
- INIT_ADMIN_PASSWORD
- CONTROL_JWT_SECRET
- EXECUTION_CALLBACK_KEY
- CONTROL_GRPC_TLS_CERT_FILE / CONTROL_GRPC_TLS_KEY_FILE
- CONTROL_CORS_ALLOW_ORIGINS
- FRONT_VITE_BACKEND_BASE_URL

3. 启动：

```bash
docker compose -f docker-compose.control.yml --env-file .env.control up -d --build
```

## 6.2 机器 B（执行端）

1. 复制模板：

```bash
cp .env.executor.example .env.executor
```

2. 修改关键参数：

- EXECUTOR_RABBITMQ_URL
- EXECUTOR_CONTROL_GRPC_TARGET
- EXECUTOR_CONTROL_GRPC_TLS_SERVER_NAME
- EXECUTOR_CONTROL_HTTP_BASE
- EXECUTION_CALLBACK_KEY（必须与控制端一致）
- EXECUTOR_TASK_TIMEOUT_SECONDS

3. 启动：

```bash
docker compose -f docker-compose.executor.yml --env-file .env.executor up -d --build
```

## 6.3 注册执行节点（必做）

控制端和执行端启动后，还需要把执行节点注册到控制端。

步骤 1：读取执行节点密钥

```bash
NODE_KEY=$(tr -d '\r\n' < Backend/data/executor.node.key)
echo "$NODE_KEY"
```

步骤 2：确认执行端健康检查鉴权生效

```bash
curl -i http://<executor-ip>:4280/healthz
curl -H "X-Node-Key: $NODE_KEY" http://<executor-ip>:4280/healthz
```

步骤 3：登录控制端并注册节点

```bash
TOKEN=$(curl -s http://<control-ip>:8180/api/v1/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"username":"admin","password":"<INIT_ADMIN_PASSWORD>"}' | jq -r '.token')

curl -s http://<control-ip>:8180/api/v1/nodes/register/ \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d "{\"ip\":\"<executor-ip>\",\"name\":\"executor-a\",\"port\":4280,\"grpc_port\":9190,\"pair_key\":\"$NODE_KEY\"}"
```

## 7. 发布后验收清单

- Frontend 首页可访问
- Control 健康检查返回 200
- Executor 健康检查不带 X-Node-Key 返回 401
- Executor 健康检查带 X-Node-Key 返回 200
- /api/v1/nodes/register/ 可以成功注册节点
- 创建任务后可正常触发、执行并回调成功

## 8. 常用运维命令

单机生产：

```bash
docker compose -f docker-compose.prod.yml --env-file .env.prod ps
docker compose -f docker-compose.prod.yml --env-file .env.prod logs -f
docker compose -f docker-compose.prod.yml --env-file .env.prod pull
docker compose -f docker-compose.prod.yml --env-file .env.prod up -d
```

控制端机器：

```bash
docker compose -f docker-compose.control.yml --env-file .env.control ps
docker compose -f docker-compose.control.yml --env-file .env.control logs -f controlnode front rabbitmq
```

执行端机器：

```bash
docker compose -f docker-compose.executor.yml --env-file .env.executor ps
docker compose -f docker-compose.executor.yml --env-file .env.executor logs -f executionnode
```

## 9. 常见问题排查

1. 节点注册失败（401 / pair_key rejected）

- 检查执行端 NODE_KEY 是否读取正确
- 检查注册请求中的 pair_key 是否有换行符
- 检查执行端是否对外可达（4280）

2. 任务触发后一直 queued

- 检查 node_queue 与 EXECUTOR_QUEUE 是否一致
- 检查执行端与 RabbitMQ 连通性
- 检查执行端日志是否存在消费报错

3. 回调失败（callback failed with status ...）

- 检查 EXECUTION_CALLBACK_KEY 两端是否一致
- 检查 EXECUTOR_CONTROL_HTTP_BASE 地址和协议是否正确
- 检查控制端 8180/443 防火墙与反向代理规则

4. gRPC 拉制品失败

- 检查控制端 9190 端口开放
- 检查 TLS 证书、SNI（EXECUTOR_CONTROL_GRPC_TLS_SERVER_NAME）配置
- 检查节点是否已注册且处于 enabled/active
