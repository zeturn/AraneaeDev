"""
machine/installer.py
zh-CN: 操作系统感知的运行时软件安装器
en-US: OS-aware runtime installer for the Araneae ExecutionNode
"""
# -*- coding: utf-8 -*-
import platform
import shutil
import subprocess
import threading
import time
import uuid
from typing import Optional


# ---------------------------------------------------------------------------
# 运行时安装命令定义表
# 每种运行时针对三大平台给出安装命令（列表形式，执行时用 shell=True 拼接为字符串）
# ---------------------------------------------------------------------------

# 平台常量
_OS = platform.system()  # "Linux" / "Darwin" / "Windows"


def _detect_pkg_manager() -> str:
    """检测 Linux 发行版包管理器：apt / yum / dnf / pacman / apk"""
    for pm in ("apt-get", "dnf", "yum", "pacman", "apk"):
        if shutil.which(pm):
            return pm
    return "apt-get"


# 安装命令模板
# key → {linux: [...], darwin: [...], windows: [...]}
# 命令列表：每项是一个完整的 shell 命令，按顺序执行，任意一步失败则中止
INSTALL_RECIPES: dict[str, dict[str, list[str]]] = {
    "node": {
        "linux": [
            "curl -fsSL https://deb.nodesource.com/setup_lts.x | bash -",
            "{pm} install -y nodejs",
        ],
        "darwin": ["brew install node"],
        "windows": ["winget install --id OpenJS.NodeJS -e --accept-source-agreements --accept-package-agreements"],
    },
    "npm": {
        # npm 随 Node.js 一起安装，单独安装只是升级
        "linux": ["{pm} install -y npm"],
        "darwin": ["brew install node"],
        "windows": ["winget install --id OpenJS.NodeJS -e --accept-source-agreements --accept-package-agreements"],
    },
    "bun": {
        "linux": ["curl -fsSL https://bun.sh/install | bash"],
        "darwin": ["brew install bun"],
        "windows": ["powershell -Command \"irm bun.sh/install.ps1 | iex\""],
    },
    "python": {
        "linux": ["{pm} install -y python3 python3-pip"],
        "darwin": ["brew install python"],
        "windows": ["winget install --id Python.Python.3 -e --accept-source-agreements --accept-package-agreements"],
    },
    "pip": {
        "linux": ["{pm} install -y python3-pip"],
        "darwin": ["brew install python"],
        "windows": ["python -m ensurepip --upgrade"],
    },
    "java": {
        "linux": ["{pm} install -y default-jdk"],
        "darwin": ["brew install openjdk"],
        "windows": ["winget install --id Microsoft.OpenJDK.21 -e --accept-source-agreements --accept-package-agreements"],
    },
    "javac": {
        "linux": ["{pm} install -y default-jdk"],
        "darwin": ["brew install openjdk"],
        "windows": ["winget install --id Microsoft.OpenJDK.21 -e --accept-source-agreements --accept-package-agreements"],
    },
    "go": {
        "linux": ["{pm} install -y golang-go"],
        "darwin": ["brew install go"],
        "windows": ["winget install --id GoLang.Go -e --accept-source-agreements --accept-package-agreements"],
    },
    "ruby": {
        "linux": ["{pm} install -y ruby-full"],
        "darwin": ["brew install ruby"],
        "windows": ["winget install --id RubyInstallerTeam.Ruby -e --accept-source-agreements --accept-package-agreements"],
    },
    "php": {
        "linux": ["{pm} install -y php php-cli php-mbstring"],
        "darwin": ["brew install php"],
        "windows": ["winget install --id PHP.PHP -e --accept-source-agreements --accept-package-agreements"],
    },
    "dotnet": {
        "linux": ["{pm} install -y dotnet-sdk-8.0"],
        "darwin": ["brew install dotnet"],
        "windows": ["winget install --id Microsoft.DotNet.SDK.8 -e --accept-source-agreements --accept-package-agreements"],
    },
    "rustc": {
        "linux": ["curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh -s -- -y"],
        "darwin": ["brew install rust"],
        "windows": ["winget install --id Rustlang.Rustup -e --accept-source-agreements --accept-package-agreements"],
    },
    "gcc": {
        "linux": ["{pm} install -y build-essential gcc g++"],
        "darwin": ["xcode-select --install"],
        "windows": ["winget install --id GnuWin32.GCC -e --accept-source-agreements --accept-package-agreements"],
    },
    "conda": {
        "linux": [
            "curl -fsSL https://repo.anaconda.com/miniconda/Miniconda3-latest-Linux-x86_64.sh -o /tmp/miniconda.sh",
            "bash /tmp/miniconda.sh -b -p $HOME/miniconda",
            "eval \"$($HOME/miniconda/bin/conda shell.bash hook)\" && conda init",
        ],
        "darwin": ["brew install --cask miniconda"],
        "windows": ["winget install --id Anaconda.Miniconda3 -e --accept-source-agreements --accept-package-agreements"],
    },
    "perl": {
        "linux": ["{pm} install -y perl"],
        "darwin": ["brew install perl"],
        "windows": ["winget install --id StrawberryPerl.StrawberryPerl -e --accept-source-agreements --accept-package-agreements"],
    },
    "swift": {
        "linux": [
            "{pm} install -y swift",
        ],
        "darwin": ["xcode-select --install"],
        "windows": ["winget install --id Swift.Toolchain -e --accept-source-agreements --accept-package-agreements"],
    },
    "kotlin": {
        "linux": [
            "curl -s https://get.sdkman.io | bash",
            "bash -c 'source $HOME/.sdkman/bin/sdkman-init.sh && sdk install kotlin'",
        ],
        "darwin": ["brew install kotlin"],
        "windows": ["winget install --id JetBrains.Kotlin -e --accept-source-agreements --accept-package-agreements"],
    },
    "lua": {
        "linux": ["{pm} install -y lua5.4"],
        "darwin": ["brew install lua"],
        "windows": ["winget install --id DEVCOM.Lua -e --accept-source-agreements --accept-package-agreements"],
    },
    "R": {
        "linux": ["{pm} install -y r-base"],
        "darwin": ["brew install r"],
        "windows": ["winget install --id RProject.R -e --accept-source-agreements --accept-package-agreements"],
    },
}

# 每个运行时的友好显示名
RUNTIME_NAMES: dict[str, str] = {
    "node": "Node.js",
    "npm": "npm",
    "bun": "Bun",
    "python": "Python",
    "pip": "pip",
    "java": "Java (JDK)",
    "javac": "Java Compiler",
    "go": "Go",
    "ruby": "Ruby",
    "php": "PHP",
    "dotnet": ".NET SDK",
    "rustc": "Rust",
    "gcc": "GCC (C/C++)",
    "conda": "Conda (Miniconda)",
    "perl": "Perl",
    "swift": "Swift",
    "kotlin": "Kotlin",
    "lua": "Lua",
    "R": "R",
}

# ---------------------------------------------------------------------------
# 安装任务注册表（内存）
# job_id → {status, log, exit_code, key, started_at}
# ---------------------------------------------------------------------------
_jobs: dict[str, dict] = {}
_jobs_lock = threading.Lock()

# 作业状态枚举
STATUS_PENDING = "pending"
STATUS_RUNNING = "running"
STATUS_SUCCESS = "success"
STATUS_FAILED = "failed"


def get_installable_runtimes() -> list[dict]:
    """
    返回当前 OS 下支持安装的运行时元信息列表
    [{"key", "name", "supported": True/False, "platform"}]
    """
    os_key = _get_os_key()
    result = []
    for key, recipes in INSTALL_RECIPES.items():
        supported = os_key in recipes
        result.append({
            "key": key,
            "name": RUNTIME_NAMES.get(key, key),
            "supported": supported,
            "platform": os_key,
        })
    return result


def _get_os_key() -> str:
    """返回 'linux' / 'darwin' / 'windows'"""
    s = _OS.lower()
    if s == "darwin":
        return "darwin"
    if s == "windows":
        return "windows"
    return "linux"


def _build_commands(key: str) -> Optional[list[str]]:
    """
    根据 key 和当前 OS 构建要执行的命令列表。
    Linux 下自动替换 {pm} 占位符为检测到的包管理器。
    Returns None if key or OS not supported.
    """
    os_key = _get_os_key()
    recipes = INSTALL_RECIPES.get(key)
    if not recipes:
        return None
    cmds = recipes.get(os_key)
    if not cmds:
        return None

    if os_key == "linux":
        pm = _detect_pkg_manager()
        cmds = [c.replace("{pm}", pm) for c in cmds]

    return cmds


def _run_job(job_id: str, key: str, cmds: list[str]) -> None:
    """
    后台线程：依次执行安装命令，将输出追加到 job 日志中。
    """
    def _append(text: str) -> None:
        with _jobs_lock:
            _jobs[job_id]["log"] += text

    with _jobs_lock:
        _jobs[job_id]["status"] = STATUS_RUNNING

    _append(f"=== [{RUNTIME_NAMES.get(key, key)}] 安装开始 ===\n")
    _append(f"[OS: {_OS}]\n\n")

    overall_success = True
    for cmd in cmds:
        _append(f"$ {cmd}\n")
        try:
            proc = subprocess.Popen(
                cmd,
                shell=True,
                stdout=subprocess.PIPE,
                stderr=subprocess.STDOUT,
                text=True,
                errors="replace",
            )
            if proc.stdout is not None:
                for line in proc.stdout:
                    _append(line)
            proc.wait()
            if proc.returncode != 0:
                _append(f"\n[错误] 命令退出码: {proc.returncode}\n")
                overall_success = False
                break
            else:
                _append(f"\n[OK] 命令成功\n\n")
        except Exception as exc:
            _append(f"\n[异常] {exc}\n")
            overall_success = False
            break

    final_status = STATUS_SUCCESS if overall_success else STATUS_FAILED
    _append(f"\n=== 安装{'成功' if overall_success else '失败'} ===\n")
    with _jobs_lock:
        _jobs[job_id]["status"] = final_status
        _jobs[job_id]["exit_code"] = 0 if overall_success else 1
        _jobs[job_id]["finished_at"] = time.time()


def start_install(key: str) -> dict:
    """
    发起后台安装任务。
    Returns:
        {"job_id": str, "error": str|None}
    """
    cmds = _build_commands(key)
    if cmds is None:
        return {"job_id": None, "error": f"运行时 '{key}' 不支持或当前 OS 不支持自动安装"}

    job_id = str(uuid.uuid4())
    with _jobs_lock:
        _jobs[job_id] = {
            "job_id": job_id,
            "key": key,
            "name": RUNTIME_NAMES.get(key, key),
            "status": STATUS_PENDING,
            "log": "",
            "exit_code": None,
            "started_at": time.time(),
            "finished_at": None,
        }

    t = threading.Thread(target=_run_job, args=(job_id, key, cmds), daemon=True)
    t.start()
    return {"job_id": job_id, "error": None}


def get_job_status(job_id: str) -> Optional[dict]:
    """
    获取安装任务状态快照。
    Returns None if job_id not found.
    """
    with _jobs_lock:
        job = _jobs.get(job_id)
        if job is None:
            return None
        return dict(job)  # 返回副本避免线程竞争
