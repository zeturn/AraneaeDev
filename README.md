# Araneae

分布式任务调度与执行平台，包含 ControlNode、ExecutionNode、前端与消息队列组件。

## Overview

- **定位**：任务调度编排与执行系统
- **核心组件**：`ControlNode`、`ExecutionNode`、`Front`、`RabbitMQ`
- **运行形态**：单机 Docker 编排或本地多进程开发

## Repository Structure

```text
Araneae/
├─ ControlNode/             # Django 控制节点
├─ ExecutionNode/           # 执行节点
├─ Front/                   # 前端
├─ scripts/                 # 开发脚本
├─ docker-compose.yml       # Docker 编排
├─ docs-site/               # Docusaurus 文档站（统一文档入口）
└─ README.md
```

## Quick Start

### Docker

```bash
cd Araneae
docker compose up -d --build
```

- Front: `http://localhost:5109`
- ControlNode API: `http://localhost:8107`
- ExecutionNode: `http://localhost:4107`
- RabbitMQ Console: `http://localhost:15672`

### Local Development

```bash
cd ControlNode
python -m venv .venv
.\.venv\Scripts\activate
pip install -r requirements.txt
python manage.py migrate
python manage.py runserver 0.0.0.0:8107
```

```bash
cd ../ExecutionNode
python -m venv .venv
.\.venv\Scripts\activate
pip install -r requirements.txt
python app.py
```

```bash
cd ../Front
npm install
npm run dev -- --port 5109
```

## Configuration

优先使用仓库根目录 `.env`（由 `.env.example` 拷贝）：

- `DJANGO_DB_PATH`
- `EXECUTION_DB_PATH`
- `RABBITMQ_USERNAME`
- `RABBITMQ_PASSWORD`
- `BASALTPASS_OAUTH_CLIENT_ID`
- `BASALTPASS_OAUTH_CLIENT_SECRET`
- `ARANEAE_CALLBACK_SHARED_SECRET`
- `ARANEAE_NODE_API_TOKEN`

## Persistence Mounts

`docker-compose.yml` 当前已挂载：

- `./data/controlnode -> /data`（ControlNode 数据，含 SQLite）
- `./data/controlnode/media -> /app/media`（媒体目录）
- `./data/controlnode/repo -> /app/Araneae_repo/repo`（仓库目录）
- `./data/executionnode -> /data`（ExecutionNode 数据，含 SQLite）
- `./data/executionnode/logs -> /app/logs`（执行日志）
- `./data/rabbitmq -> /var/lib/rabbitmq`（RabbitMQ 数据）

## Documentation

项目文档统一入口：`docs-site/`（Docusaurus）。

```bash
cd docs-site
npm install
npm run start
```

## Testing

- ControlNode：`python manage.py test`
- ExecutionNode：按模块运行 pytest/自带测试脚本

## Deployment

生产建议：

- 将 RabbitMQ 与数据库目录纳入备份策略
- 对外接口启用反向代理与 HTTPS
- 对节点通信密钥与 API Token 做轮换管理

---

发布门禁与流程请见文档站对应章节。