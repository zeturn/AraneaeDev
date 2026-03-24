# Araneae 部署说明

本文档说明 Araneae 如何通过 `GHCR` 做镜像化部署，以及如何接入 `BasaltPass` 认证。

## 1. 部署目标

- ControlNode API: `8107`
- Frontend: `5109`
- ExecutionNode: `4107`
- RabbitMQ: `5672`, `15672`
- 建议镜像:
  - `ghcr.io/<owner>/araneae-controlnode:<tag>`
  - `ghcr.io/<owner>/araneae-executionnode:<tag>`
  - `ghcr.io/<owner>/araneae-front:<tag>`

## 2. BasaltPass 接入方式

Araneae 的 BasaltPass 接入由仓库根目录 `.env` 控制，后端使用 discovery 自动发现端点。

关键变量见 `.env.example`:

```env
BASALTPASS_BASE_URL=https://auth.example.com
BASALTPASS_OAUTH_ENABLED=true
BASALTPASS_OAUTH_DISCOVERY_URL=https://auth.example.com/api/v1/.well-known/openid-configuration
BASALTPASS_OAUTH_CLIENT_ID=<client-id>
BASALTPASS_OAUTH_CLIENT_SECRET=<client-secret>
BASALTPASS_OAUTH_REDIRECT_URI=https://api.example.com/api/auth/basaltpass/callback/
BASALTPASS_OAUTH_SCOPE=openid profile email offline_access
BASALTPASS_FRONTEND_CALLBACK_PATH=/oauth/callback
```

线上回调建议:

- 后端回调: `https://api.example.com/api/auth/basaltpass/callback/`
- 前端回跳页: `https://app.example.com/oauth/callback`

## 3. 需要在 BasaltPass 中创建的客户端

为 Araneae 创建 1 个 OAuth 客户端即可:

- `grant_types`: `authorization_code`, `refresh_token`
- `client_type`: `confidential`
- `redirect_uris`:
  - `https://api.example.com/api/auth/basaltpass/callback/`
- `scopes`: `openid profile email offline_access`
- `require_pkce`: `true`

## 4. 生产环境变量

从 `Araneae/.env.example` 复制 `.env`，至少填写:

```env
DJANGO_SECRET_KEY=<long-random-secret>
DJANGO_DEBUG=False
DJANGO_ALLOWED_HOSTS=api.example.com

FRONTEND_BASE_URL=https://app.example.com
CONTROLNODE_BASE_URL=https://api.example.com
VITE_BACKEND_BASE_URL=https://api.example.com

BASALTPASS_BASE_URL=https://auth.example.com
BASALTPASS_OAUTH_DISCOVERY_URL=https://auth.example.com/api/v1/.well-known/openid-configuration
BASALTPASS_OAUTH_CLIENT_ID=<client-id>
BASALTPASS_OAUTH_CLIENT_SECRET=<client-secret>
BASALTPASS_OAUTH_REDIRECT_URI=https://api.example.com/api/auth/basaltpass/callback/

ARANEAE_CALLBACK_SHARED_SECRET=<long-random-secret>
ARANEAE_NODE_API_TOKEN=<node-api-token>

RABBITMQ_USERNAME=<username>
RABBITMQ_PASSWORD=<password>
```

## 5. GHCR 自动部署建议

当前仓库已有 CI，但没有现成的 GHCR deploy workflow。建议补充一个 deploy workflow，流程如下:

1. 构建 `ControlNode`、`ExecutionNode`、`Front` 三个镜像并推送到 GHCR。
2. 将服务器上的 `.env` 保存在固定目录，例如 `/opt/araneae/.env`。
3. 服务器使用 compose 以预构建镜像启动。

建议 compose 生产版结构:

- `controlnode`
- `controlnode-worker`
- `controlnode-beat`
- `executionnode`
- `front`
- `rabbitmq`

注意:

- `controlnode-worker` 和 `controlnode-beat` 可复用 `controlnode` 镜像
- `front` 镜像需要在构建时注入前端 API 基础地址

## 6. 服务器落地步骤

1. 先确认 BasaltPass 已上线。
2. 在 BasaltPass 注册 Araneae OAuth 客户端。
3. 服务器准备 `/opt/araneae/.env`。
4. 通过 GHCR 拉取镜像并执行:

```bash
docker compose pull
docker compose up -d --remove-orphans
```

5. 配置反向代理:
  - `app.example.com` -> Front
  - `api.example.com` -> ControlNode

## 7. 验收

- 打开前端登录页后可跳转到 BasaltPass
- BasaltPass 登录完成后可回到 `/oauth/callback`
- `controlnode` 健康检查通过
- `executionnode` 健康检查通过
- `rabbitmq` 正常连接
