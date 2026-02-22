"""
machine/capabilities.py
zh-CN: 检测当前系统中可用的运行时环境（Python / Node.js / Java 等）
en-US: Detect available runtime environments on the current system
"""
# -*- coding: utf-8 -*-
import platform
import shutil
import subprocess
import sys


# 运行时的定义表：
# key        - 可执行文件名（PATH 中查找）
# name       - 显示名称
# version_args - 用于获取版本的命令行参数
# version_flag - 有些工具版本输出在 stdout（True），其余在 stderr（False, None=自动）
RUNTIMES = [
    {
        "key": "python",
        "name": "Python",
        "bins": ["python3", "python"],
        "version_args": ["--version"],
        "stderr": False,
    },
    {
        "key": "node",
        "name": "Node.js",
        "bins": ["node"],
        "version_args": ["--version"],
        "stderr": False,
    },
    {
        "key": "npm",
        "name": "npm",
        "bins": ["npm"],
        "version_args": ["--version"],
        "stderr": False,
    },
    {
        "key": "bun",
        "name": "Bun",
        "bins": ["bun"],
        "version_args": ["--version"],
        "stderr": False,
    },
    {
        "key": "java",
        "name": "Java",
        "bins": ["java"],
        "version_args": ["-version"],
        "stderr": True,   # java -version 输出到 stderr
    },
    {
        "key": "javac",
        "name": "Java Compiler",
        "bins": ["javac"],
        "version_args": ["-version"],
        "stderr": True,
    },
    {
        "key": "go",
        "name": "Go",
        "bins": ["go"],
        "version_args": ["version"],
        "stderr": False,
    },
    {
        "key": "ruby",
        "name": "Ruby",
        "bins": ["ruby"],
        "version_args": ["--version"],
        "stderr": False,
    },
    {
        "key": "php",
        "name": "PHP",
        "bins": ["php"],
        "version_args": ["--version"],
        "stderr": False,
    },
    {
        "key": "dotnet",
        "name": ".NET",
        "bins": ["dotnet"],
        "version_args": ["--version"],
        "stderr": False,
    },
    {
        "key": "rustc",
        "name": "Rust",
        "bins": ["rustc"],
        "version_args": ["--version"],
        "stderr": False,
    },
    {
        "key": "gcc",
        "name": "GCC (C/C++)",
        "bins": ["gcc"],
        "version_args": ["--version"],
        "stderr": False,
    },
    {
        "key": "conda",
        "name": "Conda",
        "bins": ["conda"],
        "version_args": ["--version"],
        "stderr": False,
    },
    {
        "key": "pip",
        "name": "pip",
        "bins": ["pip3", "pip"],
        "version_args": ["--version"],
        "stderr": False,
    },
    {
        "key": "perl",
        "name": "Perl",
        "bins": ["perl"],
        "version_args": ["--version"],
        "stderr": False,
    },
    {
        "key": "swift",
        "name": "Swift",
        "bins": ["swift"],
        "version_args": ["--version"],
        "stderr": True,
    },
    {
        "key": "kotlin",
        "name": "Kotlin",
        "bins": ["kotlinc"],
        "version_args": ["-version"],
        "stderr": True,
    },
    {
        "key": "lua",
        "name": "Lua",
        "bins": ["lua", "lua5.4", "lua5.3"],
        "version_args": ["-v"],
        "stderr": True,
    },
    {
        "key": "R",
        "name": "R",
        "bins": ["Rscript"],
        "version_args": ["--version"],
        "stderr": True,
    },
]


def _resolve_bin(bins: list[str]) -> tuple[str | None, str | None]:
    """
    从候选可执行文件列表中，找到 PATH 内第一个可用的。
    Returns:
        (bin_name, bin_path) or (None, None)
    """
    for b in bins:
        path = shutil.which(b)
        if path:
            return b, path
    return None, None


def _get_version(bin_path: str, args: list[str], use_stderr: bool, timeout: int = 3) -> str:
    """
    调用 bin_path + args 获取版本字符串，最多取前两行。
    """
    try:
        result = subprocess.run(
            [bin_path] + args,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            timeout=timeout,
        )
        if use_stderr:
            raw = result.stderr.decode("utf-8", errors="replace").strip()
        else:
            raw = result.stdout.decode("utf-8", errors="replace").strip()
            if not raw:
                raw = result.stderr.decode("utf-8", errors="replace").strip()
        # 只取第一行，避免多行输出
        first_line = raw.splitlines()[0] if raw else ""
        return first_line
    except Exception:
        return ""


def detect_runtime_capabilities() -> list[dict]:
    """
    zh-CN: 检测系统中所有已定义运行时的可用状态
    en-US: Detect all defined runtimes and their availability

    Returns:
        list of dicts:
          {
            "key": str,      # 运行时唯一键
            "name": str,     # 显示名称
            "available": bool,
            "version": str,  # 版本字符串，不可用时为 ""
            "path": str,     # 可执行文件路径，不可用时为 ""
          }
    """
    results = []
    for rt in RUNTIMES:
        bins: list[str] = rt["bins"]  # type: ignore[assignment]
        version_args: list[str] = rt["version_args"]  # type: ignore[assignment]
        use_stderr: bool = bool(rt.get("stderr", False))

        bin_name, bin_path = _resolve_bin(bins)
        if bin_path:
            version = _get_version(bin_path, version_args, use_stderr)
            results.append({
                "key": rt["key"],
                "name": rt["name"],
                "available": True,
                "version": version,
                "path": bin_path,
            })
        else:
            results.append({
                "key": rt["key"],
                "name": rt["name"],
                "available": False,
                "version": "",
                "path": "",
            })
    return results
