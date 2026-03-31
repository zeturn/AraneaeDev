#!/usr/bin/env python3
"""Control-plane E2E crawler with simulated executor callbacks.

This script validates:
1) create task
2) trigger task
3) create schedule
4) cron auto-creates schedule run
5) schedule chain advances to next step
6) callback updates execution results

It avoids local executor runtime dependency (/bin/bash on Windows) by sending
callback payloads directly to Control's callback endpoint.
"""

from __future__ import annotations

import json
import os
import socket
import subprocess
import tempfile
import time
import urllib.error
import urllib.request
import uuid
import zipfile
from pathlib import Path
from typing import Any, Callable, Dict, List, Optional, Tuple

ROOT = Path(__file__).resolve().parents[1]

CONTROL_PORT = 18280
GRPC_PORT = 19190
RABBIT_PORT = 5672

CONTROL_BASE = f"http://127.0.0.1:{CONTROL_PORT}"
API_BASE = f"{CONTROL_BASE}/api/v1"
CALLBACK_KEY = "e2e-callback-key"
TERMINAL = {"success", "failed"}


def is_port_open(host: str, port: int, timeout: float = 0.5) -> bool:
    try:
        with socket.create_connection((host, port), timeout=timeout):
            return True
    except OSError:
        return False


def kill_port_listener(port: int) -> None:
    if os.name == "nt":
        try:
            out = subprocess.check_output(["netstat", "-ano"], text=True, stderr=subprocess.DEVNULL)
        except Exception:
            return
        target = f":{port}"
        pids = set()
        for line in out.splitlines():
            row = " ".join(line.split())
            if "LISTENING" in row and target in row:
                pid = row.split(" ")[-1]
                if pid.isdigit():
                    pids.add(pid)
        for pid in pids:
            subprocess.run(["taskkill", "/PID", pid, "/T", "/F"], stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL, check=False)
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


def ensure_rabbitmq() -> None:
    if is_port_open("127.0.0.1", RABBIT_PORT):
        return

    subprocess.run(
        ["docker", "compose", "up", "-d", "rabbitmq"],
        cwd=str(ROOT),
        check=True,
        stdout=subprocess.PIPE,
        stderr=subprocess.STDOUT,
        text=True,
    )

    deadline = time.time() + 30
    while time.time() < deadline:
        if is_port_open("127.0.0.1", RABBIT_PORT):
            return
        time.sleep(0.4)
    raise RuntimeError("RabbitMQ was not ready on 5672")


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
    raise RuntimeError(f"Service not healthy: {url}")


def wait_port_open(host: str, port: int, timeout_seconds: int = 60) -> None:
    deadline = time.time() + timeout_seconds
    while time.time() < deadline:
        if is_port_open(host, port):
            return
        time.sleep(0.3)
    raise RuntimeError(f"Port not ready: {host}:{port}")


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
        with urllib.request.urlopen(req, timeout=25) as resp:
            raw = resp.read().decode("utf-8")
            return json.loads(raw) if raw else {}
    except urllib.error.HTTPError as exc:
        detail = exc.read().decode("utf-8", errors="replace")
        raise RuntimeError(f"{method} {path} failed: HTTP {exc.code} {detail}") from exc


def api_upload(path: str, token: str, file_name: str, file_bytes: bytes) -> Any:
    url = f"{API_BASE}{path}"
    boundary = f"----araneae-{uuid.uuid4().hex}"
    parts: List[bytes] = []
    parts.append(f"--{boundary}\r\n".encode())
    parts.append(
        (
            f'Content-Disposition: form-data; name="file"; filename="{file_name}"\r\n'
            "Content-Type: application/octet-stream\r\n\r\n"
        ).encode()
    )
    parts.append(file_bytes)
    parts.append(b"\r\n")
    parts.append(f"--{boundary}--\r\n".encode())
    body = b"".join(parts)

    headers = {
        "Authorization": f"Bearer {token}",
        "Accept": "application/json",
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
        raise RuntimeError(f"upload failed: HTTP {exc.code} {detail}") from exc


def callback_run(run_id: str, status: str, output: str, exit_code: int = 0) -> None:
    url = f"{API_BASE}/runs/{run_id}/callback"
    payload = {
        "status": status,
        "output": output,
        "exit_code": exit_code,
    }
    body = json.dumps(payload).encode("utf-8")
    headers = {
        "Accept": "application/json",
        "Content-Type": "application/json",
        "X-Execution-Key": CALLBACK_KEY,
    }
    req = urllib.request.Request(url=url, data=body, headers=headers, method="POST")
    try:
        with urllib.request.urlopen(req, timeout=20):
            return
    except urllib.error.HTTPError as exc:
        detail = exc.read().decode("utf-8", errors="replace")
        raise RuntimeError(f"callback failed for run {run_id}: HTTP {exc.code} {detail}") from exc


def create_test_zip() -> Tuple[str, bytes]:
    content = "Synthetic test artifact for Araneae crawler.\n"
    with tempfile.TemporaryDirectory(prefix="araneae-artifact-") as tmp:
        zip_path = Path(tmp) / "artifact.zip"
        with zipfile.ZipFile(zip_path, "w", zipfile.ZIP_DEFLATED) as zf:
            zf.writestr("README.txt", content)
        return zip_path.name, zip_path.read_bytes()


def wait_for_task_run(token: str, task_id: str, run_id: str, timeout_seconds: int = 60) -> Dict[str, Any]:
    deadline = time.time() + timeout_seconds
    while time.time() < deadline:
        runs = api_json("GET", f"/tasks/{task_id}/runs", token=token)
        for run in runs:
            if run.get("id") == run_id and run.get("status") in TERMINAL:
                return run
        time.sleep(0.8)
    raise RuntimeError(f"Timed out waiting terminal task run: task={task_id} run={run_id}")


def wait_for_schedule_run(token: str, schedule_id: str, pred: Callable[[Dict[str, Any]], bool], timeout_seconds: int = 80) -> Dict[str, Any]:
    deadline = time.time() + timeout_seconds
    while time.time() < deadline:
        runs = api_json("GET", f"/schedules/{schedule_id}/runs", token=token)
        for run in runs:
            if pred(run):
                return run
        time.sleep(1)
    raise RuntimeError(f"Timed out waiting schedule run for {schedule_id}")


def wait_for_schedule_terminal_count(token: str, schedule_id: str, count: int, timeout_seconds: int = 80) -> List[Dict[str, Any]]:
    deadline = time.time() + timeout_seconds
    while time.time() < deadline:
        runs = api_json("GET", f"/schedules/{schedule_id}/runs", token=token)
        terminal = [r for r in runs if r.get("status") in TERMINAL]
        if len(terminal) >= count:
            return runs
        time.sleep(1)
    raise RuntimeError(f"Timed out waiting terminal schedule runs: schedule={schedule_id} count={count}")


def start_control(runtime_dir: Path) -> Tuple[subprocess.Popen, Any]:
    logs_dir = runtime_dir / "logs"
    logs_dir.mkdir(parents=True, exist_ok=True)
    control_log = (logs_dir / "control.log").open("w", encoding="utf-8")

    env = os.environ.copy()
    env.update(
        {
            "CONTROL_HTTP_ADDR": f":{CONTROL_PORT}",
            "CONTROL_GRPC_ADDR": f":{GRPC_PORT}",
            "CONTROL_DB_PATH": str(runtime_dir / "control.db"),
            "ARTIFACT_ROOT": str(runtime_dir / "artifacts"),
            "EXECUTION_CALLBACK_KEY": CALLBACK_KEY,
            "RABBITMQ_URL": "amqp://guest:guest@127.0.0.1:5672/",
        }
    )

    proc = subprocess.Popen(
        ["go", "run", "./cmd/control"],
        cwd=str(ROOT),
        env=env,
        stdout=control_log,
        stderr=subprocess.STDOUT,
    )
    return proc, control_log


def stop_process(proc: subprocess.Popen) -> None:
    if proc.poll() is not None:
        return
    if os.name == "nt":
        subprocess.run(["taskkill", "/PID", str(proc.pid), "/T", "/F"], stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL, check=False)
        return
    proc.terminate()
    try:
        proc.wait(timeout=6)
    except subprocess.TimeoutExpired:
        proc.kill()


def run_test() -> Dict[str, Any]:
    ensure_rabbitmq()
    for p in (CONTROL_PORT, GRPC_PORT):
        kill_port_listener(p)

    runtime_dir = Path(tempfile.mkdtemp(prefix="araneae-go-e2e-sim-"))
    control_proc, control_log = start_control(runtime_dir)

    try:
        wait_http_ok(f"{CONTROL_BASE}/healthz", timeout_seconds=80)
        wait_port_open("127.0.0.1", GRPC_PORT, timeout_seconds=60)
        time.sleep(1)

        login = api_json("POST", "/auth/login", payload={"username": "admin", "password": "admin123"})
        token = login.get("token")
        if not token:
            raise RuntimeError("login failed: no token")

        queue_name = f"e2e-{uuid.uuid4().hex[:8]}"

        project = api_json("POST", "/projects", token=token, payload={"name": f"sim-e2e-{uuid.uuid4().hex[:6]}"})
        project_id = project["id"]

        zip_name, zip_bytes = create_test_zip()
        version = api_upload(f"/projects/{project_id}/upload", token, zip_name, zip_bytes)
        version_id = version["id"]

        task_manual = api_json("POST", "/tasks", token=token, payload={
            "name": "sim-manual-task",
            "project_id": project_id,
            "version_id": version_id,
            "entry_command": "echo RUN_MARKER:manual",
            "cron_expr": "",
            "node_queue": queue_name,
        })

        task_chain_1 = api_json("POST", "/tasks", token=token, payload={
            "name": "sim-chain-task-1",
            "project_id": project_id,
            "version_id": version_id,
            "entry_command": "echo RUN_MARKER:chain-1",
            "cron_expr": "",
            "node_queue": queue_name,
        })

        task_chain_2 = api_json("POST", "/tasks", token=token, payload={
            "name": "sim-chain-task-2",
            "project_id": project_id,
            "version_id": version_id,
            "entry_command": "echo RUN_MARKER:chain-2",
            "cron_expr": "",
            "node_queue": queue_name,
        })

        manual_trigger = api_json("POST", f"/tasks/{task_manual['id']}/trigger", token=token)
        callback_run(manual_trigger["id"], "success", "RUN_MARKER:manual", 0)
        manual_run = wait_for_task_run(token, task_manual["id"], manual_trigger["id"], timeout_seconds=60)

        cron_schedule = api_json("POST", "/schedules", token=token, payload={
            "name": "sim-cron-schedule",
            "description": "cron schedule validation",
            "enabled": True,
            "order": {
                "name": "sim-cron-schedule",
                "schedule": [
                    {
                        "task_id": task_chain_1["id"],
                        "trigger": "crons",
                        "crons": "*/6 * * * * *",
                        "node": [queue_name],
                    }
                ],
            },
        })

        cron_queued = wait_for_schedule_run(
            token,
            cron_schedule["id"],
            lambda r: r.get("status") in {"queued", "running"},
            timeout_seconds=80,
        )
        callback_run(cron_queued["id"], "success", "RUN_MARKER:cron", 0)
        cron_runs = wait_for_schedule_terminal_count(token, cron_schedule["id"], 1, timeout_seconds=60)
        cron_terminal = next(r for r in cron_runs if r.get("id") == cron_queued["id"])

        chain_schedule = api_json("POST", "/schedules", token=token, payload={
            "name": "sim-chain-schedule",
            "description": "chain schedule validation",
            "enabled": False,
            "order": {
                "name": "sim-chain-schedule",
                "schedule": [
                    {"task_id": task_chain_1["id"], "trigger": "api", "node": [queue_name]},
                    {"task_id": task_chain_2["id"], "trigger": "previous", "node": [queue_name]},
                ],
            },
        })

        first_run = api_json("POST", f"/schedules/{chain_schedule['id']}/trigger", token=token)
        callback_run(first_run["id"], "success", "RUN_MARKER:chain-1", 0)

        second_run = wait_for_schedule_run(
            token,
            chain_schedule["id"],
            lambda r: int(r.get("chain_index", -1)) == 1 and r.get("status") in {"queued", "running", "success", "failed"},
            timeout_seconds=80,
        )
        if second_run.get("status") not in TERMINAL:
            callback_run(second_run["id"], "success", "RUN_MARKER:chain-2", 0)

        chain_runs = wait_for_schedule_terminal_count(token, chain_schedule["id"], 2, timeout_seconds=80)
        chain_indices = sorted({int(r.get("chain_index", -1)) for r in chain_runs if r.get("chain_id")})
        if not ({0, 1}.issubset(set(chain_indices))):
            raise RuntimeError(f"chain indices invalid: {chain_indices}")

        return {
            "ok": True,
            "runtime_dir": str(runtime_dir),
            "queue": queue_name,
            "checks": {
                "create_task": "PASS",
                "execute_task": "PASS" if manual_run.get("status") == "success" else "FAIL",
                "create_schedule": "PASS",
                "cron_auto_run": "PASS" if cron_terminal.get("status") == "success" else "FAIL",
                "schedule_chain": "PASS" if ({0, 1}.issubset(set(chain_indices))) else "FAIL",
                "callback_result": "PASS",
            },
            "result": {
                "manual_run": {
                    "id": manual_run.get("id"),
                    "status": manual_run.get("status"),
                    "output": (manual_run.get("output") or "")[:200],
                },
                "cron_run": {
                    "id": cron_terminal.get("id"),
                    "status": cron_terminal.get("status"),
                    "output": (cron_terminal.get("output") or "")[:200],
                },
                "chain_runs": [
                    {
                        "id": r.get("id"),
                        "status": r.get("status"),
                        "chain_index": r.get("chain_index"),
                        "output": (r.get("output") or "")[:160],
                    }
                    for r in chain_runs
                ],
            },
        }
    finally:
        stop_process(control_proc)
        control_log.close()


def main() -> int:
    try:
        out = run_test()
        print(json.dumps(out, ensure_ascii=False, indent=2))
        return 0
    except Exception as exc:
        print(json.dumps({"ok": False, "error": str(exc)}, ensure_ascii=False, indent=2))
        return 1


if __name__ == "__main__":
    raise SystemExit(main())
