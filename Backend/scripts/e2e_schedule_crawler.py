#!/usr/bin/env python3
"""End-to-end API crawler test for GoRefactor Control + Executor.

Coverage:
1. Create project + upload artifact
2. Create task + manual trigger execution + result callback
3. Create cron schedule + verify automatic run
4. Create chained schedule + verify chain step execution
5. Print summarized execution results
"""

from __future__ import annotations

import json
import os
import socket
import subprocess
import sys
import tempfile
import time
import urllib.error
import urllib.request
import uuid
import zipfile
from pathlib import Path
from typing import Any, Dict, List, Optional, Tuple

ROOT = Path(__file__).resolve().parents[1]

CONTROL_PORT = 18280
GRPC_PORT = 19190
EXECUTOR_PORT = 14280
RABBIT_PORT = 5672

CONTROL_BASE = f"http://127.0.0.1:{CONTROL_PORT}"
API_BASE = f"{CONTROL_BASE}/api/v1"

TERMINAL_STATUSES = {"success", "failed"}


def is_port_open(host: str, port: int, timeout: float = 0.5) -> bool:
    try:
        with socket.create_connection((host, port), timeout=timeout):
            return True
    except OSError:
        return False


def ensure_rabbitmq() -> None:
    if is_port_open("127.0.0.1", RABBIT_PORT):
        return

    try:
        subprocess.run(
            ["docker", "compose", "up", "-d", "rabbitmq"],
            cwd=str(ROOT),
            check=True,
            stdout=subprocess.PIPE,
            stderr=subprocess.STDOUT,
            text=True,
        )
    except Exception as exc:  # pragma: no cover - environment dependent
        raise RuntimeError(
            "RabbitMQ is not running on 5672 and docker compose failed. "
            "Start RabbitMQ manually, then rerun this script."
        ) from exc

    deadline = time.time() + 30
    while time.time() < deadline:
        if is_port_open("127.0.0.1", RABBIT_PORT):
            return
        time.sleep(0.5)
    raise RuntimeError("RabbitMQ did not become ready on port 5672")


def wait_http_ok(url: str, timeout_seconds: int = 60) -> None:
    deadline = time.time() + timeout_seconds
    while time.time() < deadline:
        try:
            req = urllib.request.Request(url=url, method="GET")
            with urllib.request.urlopen(req, timeout=2) as resp:
                if resp.status == 200:
                    return
        except Exception:
            pass
        time.sleep(0.5)
    raise RuntimeError(f"Service did not become healthy: {url}")


def wait_port_open(host: str, port: int, timeout_seconds: int = 60) -> None:
    deadline = time.time() + timeout_seconds
    while time.time() < deadline:
        if is_port_open(host, port):
            return
        time.sleep(0.3)
    raise RuntimeError(f"Port did not become ready: {host}:{port}")


def kill_port_listener(port: int) -> None:
    if os.name == "nt":
        try:
            out = subprocess.check_output(["netstat", "-ano"], text=True, stderr=subprocess.DEVNULL)
        except Exception:
            return
        pids = set()
        needle = f":{port}"
        for line in out.splitlines():
            row = " ".join(line.split())
            if "LISTENING" not in row or needle not in row:
                continue
            parts = row.split(" ")
            if parts:
                pid = parts[-1].strip()
                if pid.isdigit():
                    pids.add(pid)
        for pid in pids:
            subprocess.run(
                ["taskkill", "/PID", pid, "/T", "/F"],
                stdout=subprocess.DEVNULL,
                stderr=subprocess.DEVNULL,
                check=False,
            )
        return

    for cmd in (["lsof", "-ti", f"tcp:{port}"], ["fuser", "-n", "tcp", str(port)]):
        try:
            out = subprocess.check_output(cmd, text=True, stderr=subprocess.DEVNULL).strip()
        except Exception:
            continue
        if not out:
            continue
        for token in out.replace("\n", " ").split():
            if token.isdigit():
                subprocess.run(["kill", "-9", token], stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL, check=False)
        return


def api_json(method: str, path: str, token: Optional[str] = None, payload: Any = None) -> Any:
    url = f"{API_BASE}{path}"
    headers = {"Accept": "application/json"}
    body = None
    if payload is not None:
        body = json.dumps(payload).encode("utf-8")
        headers["Content-Type"] = "application/json"
    if token:
        headers["Authorization"] = f"Bearer {token}"

    req = urllib.request.Request(url=url, data=body, headers=headers, method=method)
    try:
        with urllib.request.urlopen(req, timeout=20) as resp:
            raw = resp.read().decode("utf-8")
            return json.loads(raw) if raw else {}
    except urllib.error.HTTPError as exc:
        detail = exc.read().decode("utf-8", errors="replace")
        raise RuntimeError(f"{method} {path} failed: HTTP {exc.code} {detail}") from exc


def api_multipart_upload(
    path: str,
    token: str,
    file_field: str,
    file_name: str,
    file_bytes: bytes,
) -> Any:
    url = f"{API_BASE}{path}"
    boundary = f"----araneae-{uuid.uuid4().hex}"

    parts: List[bytes] = []
    parts.append(f"--{boundary}\r\n".encode("utf-8"))
    parts.append(
        (
            f'Content-Disposition: form-data; name="{file_field}"; filename="{file_name}"\r\n'
            "Content-Type: application/octet-stream\r\n\r\n"
        ).encode("utf-8")
    )
    parts.append(file_bytes)
    parts.append(b"\r\n")
    parts.append(f"--{boundary}--\r\n".encode("utf-8"))
    body = b"".join(parts)

    headers = {
        "Accept": "application/json",
        "Authorization": f"Bearer {token}",
        "Content-Type": f"multipart/form-data; boundary={boundary}",
        "Content-Length": str(len(body)),
    }

    req = urllib.request.Request(url=url, data=body, headers=headers, method="POST")
    try:
        with urllib.request.urlopen(req, timeout=30) as resp:
            raw = resp.read().decode("utf-8")
            return json.loads(raw) if raw else {}
    except urllib.error.HTTPError as exc:
        detail = exc.read().decode("utf-8", errors="replace")
        raise RuntimeError(f"POST {path} upload failed: HTTP {exc.code} {detail}") from exc


def create_test_zip() -> Tuple[str, bytes]:
    readme = "Araneae E2E artifact package for API crawler test.\n"
    with tempfile.TemporaryDirectory(prefix="araneae-e2e-zip-") as tmp:
        zip_path = Path(tmp) / "crawler-test.zip"
        with zipfile.ZipFile(zip_path, "w", zipfile.ZIP_DEFLATED) as zf:
            zf.writestr("README.txt", readme)
        return zip_path.name, zip_path.read_bytes()


def wait_for_task_run(token: str, task_id: str, run_id: str, timeout_seconds: int = 90) -> Dict[str, Any]:
    deadline = time.time() + timeout_seconds
    while time.time() < deadline:
        runs = api_json("GET", f"/tasks/{task_id}/runs", token=token)
        for run in runs:
            if run.get("id") == run_id and run.get("status") in TERMINAL_STATUSES:
                return run
        time.sleep(1)
    raise RuntimeError(f"Timed out waiting task run terminal state: task={task_id} run={run_id}")


def wait_for_schedule_runs(
    token: str,
    schedule_id: str,
    min_terminal_runs: int,
    timeout_seconds: int = 120,
) -> List[Dict[str, Any]]:
    deadline = time.time() + timeout_seconds
    while time.time() < deadline:
        runs = api_json("GET", f"/schedules/{schedule_id}/runs", token=token)
        terminal = [r for r in runs if r.get("status") in TERMINAL_STATUSES]
        if len(terminal) >= min_terminal_runs:
            return runs
        time.sleep(1)
    raise RuntimeError(
        f"Timed out waiting schedule runs: schedule={schedule_id}, expected_terminal={min_terminal_runs}"
    )


def start_services(workdir: Path, queue_name: str) -> Tuple[subprocess.Popen, subprocess.Popen, Any, Any]:
    logs_dir = workdir / "logs"
    logs_dir.mkdir(parents=True, exist_ok=True)
    control_log = (logs_dir / "control.log").open("w", encoding="utf-8")
    executor_log = (logs_dir / "executor.log").open("w", encoding="utf-8")

    callback_key = "e2e-callback-key"

    control_env = os.environ.copy()
    control_env.update(
        {
            "CONTROL_HTTP_ADDR": f":{CONTROL_PORT}",
            "CONTROL_GRPC_ADDR": f":{GRPC_PORT}",
            "CONTROL_DB_PATH": str(workdir / "control.db"),
            "ARTIFACT_ROOT": str(workdir / "artifacts"),
            "EXECUTION_CALLBACK_KEY": callback_key,
            "RABBITMQ_URL": "amqp://guest:guest@127.0.0.1:5672/",
        }
    )

    executor_env = os.environ.copy()
    executor_env.update(
        {
            "EXECUTOR_HTTP_ADDR": f":{EXECUTOR_PORT}",
            "EXECUTOR_DB_PATH": str(workdir / "executor.db"),
            "EXECUTOR_WORKDIR": str(workdir / "workdir"),
            "EXECUTOR_QUEUE": queue_name,
            "CONTROL_GRPC_TARGET": f"127.0.0.1:{GRPC_PORT}",
            "CONTROL_HTTP_BASE": CONTROL_BASE,
            "EXECUTION_CALLBACK_KEY": callback_key,
            "RABBITMQ_URL": "amqp://guest:guest@127.0.0.1:5672/",
        }
    )

    control_proc = subprocess.Popen(
        ["go", "run", "./cmd/control"],
        cwd=str(ROOT),
        env=control_env,
        stdout=control_log,
        stderr=subprocess.STDOUT,
    )
    executor_proc = subprocess.Popen(
        ["go", "run", "./cmd/executor"],
        cwd=str(ROOT),
        env=executor_env,
        stdout=executor_log,
        stderr=subprocess.STDOUT,
    )

    return control_proc, executor_proc, control_log, executor_log


def stop_process(proc: subprocess.Popen) -> None:
    if proc.poll() is not None:
        return
    if os.name == "nt":
        subprocess.run(
            ["taskkill", "/PID", str(proc.pid), "/T", "/F"],
            stdout=subprocess.DEVNULL,
            stderr=subprocess.DEVNULL,
            check=False,
        )
        try:
            proc.wait(timeout=5)
        except subprocess.TimeoutExpired:
            proc.kill()
        return

    proc.terminate()
    try:
        proc.wait(timeout=8)
    except subprocess.TimeoutExpired:
        proc.kill()


def run_e2e() -> Dict[str, Any]:
    ensure_rabbitmq()

    for port in (CONTROL_PORT, GRPC_PORT, EXECUTOR_PORT):
        kill_port_listener(port)

    runtime_dir = Path(tempfile.mkdtemp(prefix="araneae-go-e2e-"))
    queue_name = f"e2e-{uuid.uuid4().hex[:8]}"
    control_proc, executor_proc, control_log, executor_log = start_services(runtime_dir, queue_name)
    try:
        wait_http_ok(f"{CONTROL_BASE}/healthz", timeout_seconds=90)
        wait_http_ok(f"http://127.0.0.1:{EXECUTOR_PORT}/healthz", timeout_seconds=90)
        wait_port_open("127.0.0.1", GRPC_PORT, timeout_seconds=60)
        time.sleep(1.5)

        login = api_json("POST", "/auth/login", payload={"username": "admin", "password": "admin123"})
        token = login.get("token")
        if not token:
            raise RuntimeError("Login did not return token")

        project = api_json("POST", "/projects", token=token, payload={"name": f"e2e-project-{uuid.uuid4().hex[:8]}"})
        project_id = project["id"]

        zip_name, zip_data = create_test_zip()
        version = api_multipart_upload(
            path=f"/projects/{project_id}/upload",
            token=token,
            file_field="file",
            file_name=zip_name,
            file_bytes=zip_data,
        )
        version_id = version["id"]

        manual_task = api_json(
            "POST",
            "/tasks",
            token=token,
            payload={
                "name": "e2e-manual-task",
                "project_id": project_id,
                "version_id": version_id,
                "entry_command": "echo RUN_MARKER:manual",
                "cron_expr": "",
                "node_queue": queue_name,
            },
        )

        chain_task_1 = api_json(
            "POST",
            "/tasks",
            token=token,
            payload={
                "name": "e2e-chain-task-1",
                "project_id": project_id,
                "version_id": version_id,
                "entry_command": "echo RUN_MARKER:chain-1",
                "cron_expr": "",
                "node_queue": queue_name,
            },
        )

        chain_task_2 = api_json(
            "POST",
            "/tasks",
            token=token,
            payload={
                "name": "e2e-chain-task-2",
                "project_id": project_id,
                "version_id": version_id,
                "entry_command": "echo RUN_MARKER:chain-2",
                "cron_expr": "",
                "node_queue": queue_name,
            },
        )

        manual_trigger = api_json("POST", f"/tasks/{manual_task['id']}/trigger", token=token)
        manual_run = wait_for_task_run(token, manual_task["id"], manual_trigger["id"], timeout_seconds=90)

        cron_schedule = api_json(
            "POST",
            "/schedules",
            token=token,
            payload={
                "name": "e2e-cron-schedule",
                "description": "cron auto run verification",
                "enabled": True,
                "order": {
                    "name": "e2e-cron-schedule",
                    "schedule": [
                        {
                            "task_id": chain_task_1["id"],
                            "trigger": "crons",
                            "crons": "*/8 * * * * *",
                            "node": [queue_name],
                        }
                    ],
                },
            },
        )

        cron_runs = wait_for_schedule_runs(token, cron_schedule["id"], min_terminal_runs=1, timeout_seconds=120)
        cron_latest = cron_runs[0]

        chain_schedule = api_json(
            "POST",
            "/schedules",
            token=token,
            payload={
                "name": "e2e-chain-schedule",
                "description": "chain trigger verification",
                "enabled": False,
                "order": {
                    "name": "e2e-chain-schedule",
                    "schedule": [
                        {
                            "task_id": chain_task_1["id"],
                            "trigger": "api",
                            "node": [queue_name],
                        },
                        {
                            "task_id": chain_task_2["id"],
                            "trigger": "previous",
                            "node": [queue_name],
                        },
                    ],
                },
            },
        )

        api_json("POST", f"/schedules/{chain_schedule['id']}/trigger", token=token)
        chain_runs = wait_for_schedule_runs(token, chain_schedule["id"], min_terminal_runs=2, timeout_seconds=120)

        chain_indices = sorted({int(r.get("chain_index", -1)) for r in chain_runs if r.get("chain_id")})
        if not ({0, 1}.issubset(set(chain_indices))):
            raise RuntimeError(f"Chain schedule did not produce both chain steps, got indices={chain_indices}")

        summary = {
            "runtime_dir": str(runtime_dir),
            "queue": queue_name,
            "project_id": project_id,
            "version_id": version_id,
            "manual_task": {
                "task_id": manual_task["id"],
                "run_id": manual_run.get("id"),
                "status": manual_run.get("status"),
                "output": (manual_run.get("output") or "")[:300],
            },
            "cron_schedule": {
                "schedule_id": cron_schedule["id"],
                "latest_run_id": cron_latest.get("id"),
                "latest_status": cron_latest.get("status"),
                "latest_output": (cron_latest.get("output") or "")[:300],
            },
            "chain_schedule": {
                "schedule_id": chain_schedule["id"],
                "terminal_run_count": len([r for r in chain_runs if r.get("status") in TERMINAL_STATUSES]),
                "chain_indices": chain_indices,
                "runs": [
                    {
                        "id": r.get("id"),
                        "status": r.get("status"),
                        "chain_index": r.get("chain_index"),
                        "output": (r.get("output") or "")[:180],
                    }
                    for r in chain_runs[:3]
                ],
            },
        }
        return summary
    finally:
        stop_process(executor_proc)
        stop_process(control_proc)
        control_log.close()
        executor_log.close()


def main() -> int:
    try:
        result = run_e2e()
        print(json.dumps({"ok": True, "result": result}, ensure_ascii=False, indent=2))
        return 0
    except Exception as exc:
        print(json.dumps({"ok": False, "error": str(exc)}, ensure_ascii=False, indent=2))
        return 1


if __name__ == "__main__":
    sys.exit(main())
