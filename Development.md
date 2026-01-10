# Araneae 分布式任务调度与执行系统开发手册

## 一、系统架构与核心理念

Araneae 是一个分布式爬虫/任务调度与执行平台，采用**前后端分离**、**主控-执行分布式**架构，支持多节点协作、任务链调度、代码分发、回调追踪等功能。

- **ControlNode（管理端）**：基于 Django + DRF，负责用户、工作区、项目、任务、节点等的管理、API 提供、任务调度、代码分发、回调处理。
- **ExecutionNode（执行端）**：基于 Flask，负责节点注册、项目代码接收、任务执行、状态上报、回调。
- **Front（前端）**：基于 Vite + Vue3，提供可视化管理界面。
- **消息通信**：管理端与执行端通过 RabbitMQ 消息队列通信，任务调度采用 Celery。
- **代码分发**：项目代码通过 HTTP/文件分发到各执行节点，节点自动拉取、解压、注册项目。
- **数据库**：管理端和执行端各自维护独立 SQLite 数据库。

---

## 二、目录结构与模块分工

```
Araneae/
├─ ControlNode/         # 管理端（Django项目）
│  ├─ Araneae/          # Django主配置
│  ├─ Araneae_main/     # 工作区、用户、团队等模型与API
│  ├─ Araneae_manager/  # 节点、项目、任务、调度、分发、回调等业务核心
│  ├─ Araneae_repo/     # 项目代码分发与管理
│  ├─ araneae_proto/    # gRPC协议文件
│  ├─ manage.py         # 启动脚本
│  └─ db.sqlite3        # 管理端数据库
├─ ExecutionNode/       # 执行端（Flask项目）
│  ├─ app.py            # 主入口，API与服务启动
│  ├─ tasks.py          # Celery任务与回调
│  ├─ source/           # 代码接收、处理、repo管理
│  ├─ models.py         # ORM模型
│  ├─ instance/db.sqlite3 # 执行端数据库
│  └─ ...               # 其他工具与子模块
├─ Front/               # 前端（Vue3+Vite）
│  ├─ src/              # 前端源码
│  └─ ...
├─ config.example.json  # 配置模板
├─ config.json          # 敏感配置（需手动创建）
└─ README.md            # 项目说明
```

---

## 三、核心运作流程

### 1. 节点注册与管理

- 执行端启动后，通过 gRPC/HTTP 向管理端注册自身（`/api/nodes/register`），管理端校验节点可达性、系统信息、握手确认，登记节点信息。
- 管理端可通过 API 查询、监控所有节点状态（心跳、资源占用等）。

### 2. 项目与代码分发

- 用户在前端创建项目，上传代码包（zip），管理端保存至 `Araneae_repo/repo/{project_id}/{version_hash}/`，并生成新版本记录。
- 管理端通过 `/source_distribute/order` API，分发指定项目版本到目标节点，触发 RabbitMQ 消息。
- 执行端监听消息队列，收到分发通知后自动下载代码包、解压到本地 repo，并在本地数据库注册项目与版本。

### 3. 任务调度与执行

- 用户可在前端为项目创建任务/调度（支持定时、链式等），管理端通过 Celery + django_celery_beat 维护调度计划。
- 任务调度时，管理端通过消息队列下发任务到目标节点。
- 执行端收到任务后，查找本地项目代码，按指定命令（如 `python main.py`）启动子进程执行脚本。
- 执行端实时记录任务状态，执行完毕后通过 HTTP 回调 `/api/task/callback/` 上报结果。

### 4. 回调与任务链

- 管理端收到回调后，记录任务结果，并根据任务链配置自动触发下一个任务（如有）。
- 支持任务链（ChainedTask）、定时任务（PeriodicTask）、单次任务等多种调度模式。

---

## 四、主要数据模型与API说明

### 1. 管理端（ControlNode）

- **Workplace/Team/User**：多用户多团队协作，支持权限分配。
- **Project/Version**：项目与多版本管理，支持代码上传、分发、版本追踪。
- **Node**：执行节点注册、状态监控。
- **Task/Schedule/ChainedTask**：任务、调度、任务链，支持定时、依赖、链式执行。
- **NodeProjectVersion**：记录项目版本在各节点的分发与部署状态。
- **API**：RESTful 风格，详见 `views.py`，如 `/api/projects/`、`/api/nodes/`、`/api/source_distribute/order`、`/api/task/callback/` 等。

### 2. 执行端（ExecutionNode）

- **Project/Version/TaskRecord**：本地项目、版本、任务执行记录。
- **任务执行流程**：
  - 监听消息队列，接收分发/任务通知。
  - 下载并解压项目代码，注册到本地数据库。
  - 按命令启动脚本，记录执行日志。
  - 任务完成后回调管理端，附带状态与结果。
- **API**：如 `/projects`（创建项目）、`/run_task`（运行任务）、`/system_info`（节点信息）、`/check_status/<pid>`（任务状态）等。

---

## 五、开发与部署建议

1. **环境配置**
   - Python 3.10+，Node.js 18+，RabbitMQ，SQLite（可扩展为 MySQL/PostgreSQL）。
   - 配置文件 `config.json` 必须手动创建，包含密钥、RabbitMQ 等敏感信息。
   - 推荐使用虚拟环境（venv）隔离依赖。

2. **启动流程**
   - 先启动 RabbitMQ 服务。
   - 启动 ControlNode（`python manage.py runserver`），初始化数据库、超级用户。
   - 启动 ExecutionNode（`python app.py`），节点自动注册。
   - 启动前端（`npm run dev`）。

3. **调试与扩展**
   - 日志详见 `logs/` 目录，便于排查问题。
   - Celery 任务与调度可通过 Django Admin 管理。
   - 支持多节点横向扩展，节点可动态上下线。
   - 代码分发、任务调度、回调等流程均有详细日志与异常处理。

4. **安全与权限**
   - 敏感配置不应提交仓库，生产环境请使用安全的密钥与凭据。
   - API 权限控制基于 DRF，需合理配置用户、团队、节点权限。

---

## 六、常见问题与排查

- **节点无法注册**：检查网络连通性、gRPC 端口、防火墙、RabbitMQ 配置。
- **任务未执行/无回调**：检查 Celery、RabbitMQ、节点日志，确认任务是否下发、脚本路径是否正确。
- **代码分发失败**：检查文件路径、权限、网络、压缩包格式。
- **数据库异常**：确认 SQLite 文件权限、路径、迁移状态。

---

## 七、参考与贡献

- 详细 API、模型、调度机制请参考各模块 `views.py`、`tasks.py`、`models.py` 注释。
- 欢迎通过 issue/pull request 反馈问题或贡献代码。

---

如需更细致的流程图、时序图、接口文档或具体代码解读，可随时补充！

---

## 八、Playwright + WebRTC 人在回路（HITL）

本节介绍如何在执行端集成 Playwright 与 WebRTC（SDP 信令），让控制端实时看到远端页面并注入鼠标/键盘事件，从而在验证码、人为干预等场景下，用户可暂时接管页面，操作完成后点击“已完成（Resume）”继续脚本执行。

### 组件

- 执行端：`ExecutionNode/webrtc.py`（新增）基于 `aiortc` + `playwright` 实现 WebRTC 终端，提供 SDP 应答、视频推流（从 Playwright 页面截图生成帧）、DataChannel('ctrl') 输入事件注入。
- 执行端 Flask 路由（新增，见 `ExecutionNode/app.py`）：
   - `POST /webrtc/session`：创建 Playwright 会话，参数 `{url, headless}`，返回 `{session_id}`。
   - `POST /webrtc/offer`：接收 SDP Offer，参数 `{session_id, sdp, type}`，返回 SDP Answer。
   - `POST /webrtc/resume/<session_id>`：恢复执行，后端设置 `resume_event`。
- 控制端 Django 页面（新增）：`ControlNode/templates/webrtc/session.html` 与视图 `Araneae_main.views.webrtc_session`，路由 `/webrtc/session/`。该页面通过 WebRTC 建立连接，接收视频，采集并发送鼠标/键盘事件到 DataChannel('ctrl')。

### 依赖安装

执行端（ExecutionNode）新增依赖：

```powershell
cd ExecutionNode
pip install -r requirements.txt
python -m playwright install chromium
```

requirements.txt 新增了：`aiortc`, `av`, `playwright`。

注意：`aiortc/av` 依赖系统的编解码（libav/ffmpeg），Windows 下请确保能正常安装 PyAV 对应的轮子；若编码异常，优先检查本地 ffmpeg/codec 支持。

### 使用步骤（开发调试）

1) 启动执行端（确保 Flask、webrtc 路由可用）

```powershell
cd ExecutionNode
python app.py
```

2) 在控制端打开 WebRTC 页面并创建远端会话

- 浏览器访问（示例）：
  
   ```
   http://127.0.0.1:8000/webrtc/session/?exec=http://127.0.0.1:5001&url=https://example.com&headless=true
   ```

- 页面加载后会自动调用执行端 `/webrtc/session` 创建会话，并将 `session_id` 填入页面；点击“连接”将：
   - 创建 RTCPeerConnection（recvonly video + datachannel 'ctrl'）。
   - `createOffer()` 并将 SDP Offer 发送到执行端 `/webrtc/offer`。
   - 设置远端 Answer，开始接收视频、发送控制。

3) 控制交互

- 视频区域的鼠标移动/点击/滚轮事件会被缩放映射到 1280×720 的 Playwright 页面坐标：
   - 鼠标消息示例：`{"type":"mouse","action":"move|down|up|click|wheel","x":..,"y":..,"button":"left|right|middle","deltaX":..,"deltaY":..}`
- 键盘事件：`{"type":"keyboard","action":"down|up|press","key":"Enter"}`
- “我已完成（Resume）”按钮：
   - 通过 DataChannel 发送 `{type:"resume"}`，并调用执行端 `POST /webrtc/resume/<session_id>`；服务端设置 `resume_event`，供业务脚本继续执行。

### 后端注入点（脚本暂停/恢复）

- 业务脚本中当检测到需要人工干预（如验证码）时，可：
   1. 调用 `POST /webrtc/session` 创建会话并导航到相关 URL；
   2. 在业务协程中 `await session.resume_event.wait()`（参考 `webrtc.py` 中的事件），待用户点击“完成”后继续。

### 安全与资源

- 对 `/webrtc/*` 路由建议在生产中增加鉴权（JWT/Session/签名 URL）。
- Playwright 浏览器与上下文需妥善关闭（`Session.close_browser()` 已在会话销毁时调用）。
- 视频帧生成基于截图 + PyAV 解码开销较大，默认 10 FPS；可在 `PlaywrightVideoTrack(fps=...)` 调整。

### 已知限制

- 基于截图的推流不如原生捕获高效，适合“应急/介入”场景；如需更高帧率/低延迟，建议切换到原生屏幕捕获（需要更复杂的管线）。
- PyAV/编解码依赖在不同平台可能存在安装差异；编码失败时，客户端可能无法收到视频。

