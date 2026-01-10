# Araneae

分布式任务调度与执行系统

目录结构（概览）
- ControlNode/ — Django 管理与 API
- ExecutionNode/ — 节点执行端，包含注册、任务执行逻辑
- Front/ — 前端 (Vite + Vue)

快速开始（开发环境）

1. 克隆仓库并进入项目根目录：
   - 本说明假设你在 Windows / PowerShell 下操作。

2. 后端（ControlNode）
   - 进入 `ControlNode`：
     ```powershell
     cd ControlNode
     python -m venv .venv
     .\.venv\Scripts\Activate.ps1
     pip install -r requirements.txt
     ```
   - 创建迁移并初始化数据库：
     ```powershell
     python manage.py migrate
     python manage.py createsuperuser
     ```
   - 启动开发服务器：
     ```powershell
     python manage.py runserver
     ```
   - 注意：数据库默认在 `ControlNode/db.sqlite3`。

    - 配置密钥与凭据：请在仓库根复制 `config.example.json` 为 `config.json` 并填入真实值（已在 `.gitignore` 中忽略）。Django 的 `SECRET_KEY`、RabbitMQ 的账号密码等会从该文件读取。

3. ExecutionNode（节点进程）
   - 进入 `ExecutionNode`：
     ```powershell
     cd ..\ExecutionNode
     python -m venv .venv
     .\.venv\Scripts\Activate.ps1
     pip install -r requirements.txt
     ```
   - 运行节点（示例）：
     ```powershell
     python app.py
     ```
   - ExecutionNode 的 sqlite 默认在 `ExecutionNode/instance/db.sqlite3`。

4. 前端（Front）
   - 进入 `Front`：
     ```powershell
     cd ..\Front
     npm install
     npm run dev
     ```

配置与环境变量
- 敏感配置集中在仓库根的 `config.json`（请从 `config.example.json` 复制再填写）。此文件已加入 `.gitignore`，不会被提交。
- 可选：`.env` 中的同名变量可覆盖部分配置（如 `DJANGO_SECRET_KEY`、`RABBITMQ_*`）。详见 `.env.example`。

常见文件说明
- `.gitignore` — 忽略不应提交的文件（虚拟环境、node_modules、数据库、日志等）。
- `.gitattributes` — 行结尾/编码策略。
- `.editorconfig` — 编辑器风格统一。
- `.pre-commit-config.yaml` — 可选，代码风格与静态检查钩子（需安装 `pre-commit`）。

开发建议
- 使用虚拟环境隔离 Python 依赖。
- 不要将生产 SECRET_KEY、数据库凭据提交到仓库。使用 `.env` 或 CI 密钥管理。

联系与贡献
- 若需帮助或想贡献，请在项目仓库中打开 issue 或 pull request。

