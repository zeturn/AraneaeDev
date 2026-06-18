# Araneae

分布式任务调度与执行平台，包含 Go 控制端、Go 运行端、前端与消息队列组件。

## Overview

- 定位：任务调度编排与执行系统
- 核心组件：Backend (Control + Executor)、Frontend、RabbitMQ
- 运行形态：
	- 单机 Docker 一体化部署
	- 多机拆分部署（控制端在 A 机，运行端在 B 机）

## Repository Structure

```text
Araneae/
├─ Backend/                     # Go 控制端 + 运行端
├─ Frontend/                    # 前端
├─ docker-compose.yml           # 单机一体化部署
├─ docker-compose.control.yml   # 控制端机器部署（control + rabbitmq + front）
├─ docker-compose.executor.yml  # 运行端机器部署（executor）
├─ .env.control.example         # 控制端机器环境示例
├─ .env.executor.example        # 运行端机器环境示例
└─ README.md
```

## Quick Start

### 单机一体化（前端 + 控制端 + 运行端）

```bash
cd Araneae
docker compose up -d --build
```

- Front: http://localhost:5109
- Control API: http://localhost:8180
- Executor: http://localhost:4280
- RabbitMQ Console: http://localhost:15672

## Multi Machine Deployment

适用于控制端在一台机器、运行端在另一台机器的场景。

### 机器 A（控制端机器）

1. 准备环境文件：复制 .env.control.example 为 .env.control，并按实际 IP/域名修改。
2. 启动：

```bash
cd Araneae
docker compose -f docker-compose.control.yml --env-file .env.control up -d --build
```

### 机器 B（运行端机器）

1. 准备环境文件：复制 .env.executor.example 为 .env.executor，并填写控制端机器地址。
2. 启动：

```bash
cd Araneae
docker compose -f docker-compose.executor.yml --env-file .env.executor up -d --build
```

### 多机部署关键点

- EXECUTION_CALLBACK_KEY 必须在控制端和所有运行端一致。
- 运行端必须能访问控制端：
	- gRPC 端口 9190
	- HTTP 端口 8180
- 运行端必须能访问共享 RabbitMQ 5672。
- 任务下发时 node_queue 需要和运行端 EXECUTOR_QUEUE 对齐。

## Frontend API Address

前端镜像构建时使用 FRONT_VITE_BACKEND_BASE_URL 作为 API 目标地址。
在多机部署时，建议设置为用户可访问的控制端地址（例如 http://control.example.com:8180），不要默认使用 localhost。

## Useful Commands

```bash
# 单机查看状态
docker compose ps

# 控制端机器查看状态
docker compose -f docker-compose.control.yml --env-file .env.control ps

# 运行端机器查看状态
docker compose -f docker-compose.executor.yml --env-file .env.executor ps
```

## Health Check

- Control: GET /healthz
- Executor: GET /healthz

## More Docs

后端多节点部署细节：Backend/deploy/MULTI_NODE_DEPLOYMENT.md

## Security and Quality

- CodeQL is configured to scan GitHub Actions, Go, Python, and JavaScript/TypeScript when those languages are present.
- Keep secrets out of the repository. Use `.env` files locally and GitHub Actions secrets in CI.
- Report vulnerabilities privately through the process in `SECURITY.md`.

## Contributing

Please read `CONTRIBUTING.md` before opening issues or pull requests. Contributions should include a clear description, relevant tests or manual verification, and updates to documentation when behavior changes.

## Code of Conduct

This project follows the community expectations in `CODE_OF_CONDUCT.md`.

## License

This project is licensed under the ISC License. See `LICENSE` for details.
