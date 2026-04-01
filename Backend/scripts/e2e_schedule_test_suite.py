#!/usr/bin/env python3
"""Comprehensive schedule E2E test suite for Araneae Go backend.

Coverage:
1) Single-task schedule create/update/trigger/delete
2) Multi-task chain schedule trigger and chain progression
3) Cron schedule timed auto-start and run completion
4) Run started_at/finished_at validation
5) Sample crawler artifact execution (crawler.py)

This suite starts isolated control/executor instances on dedicated ports,
uses RabbitMQ on localhost:5672, and validates the API end-to-end.
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
from datetime import datetime
from pathlib import Path
from typing import Any, Dict, List, Optional, Tuple
from urllib.parse import quote

ROOT = Path(__file__).resolve().parents[1]

CONTROL_PORT = 18280
GRPC_PORT = 19190
EXECUTOR_PORT = 14280
RABBIT_PORT = 5672

CONTROL_BASE = f"http://127.0.0.1:{CONTROL_PORT}"
API_BASE = f"{CONTROL_BASE}/api/v1"
CALLBACK_KEY = "suite-callback-key"
TERMINAL_STATUSES = {"success", "failed"}


def load_repo_env() -> Dict[str, str]:
    env_file = ROOT.parent / ".env"
    if not env_file.exists():
        return {}
    parsed: Dict[str, str] = {}
    for raw in env_file.read_text(encoding="utf-8", errors="ignore").splitlines():
        line = raw.strip()
        if not line or line.startswith("#") or "=" not in line:
            continue
        key, value = line.split("=", 1)
        parsed[key.strip()] = value.strip().strip('"').strip("'")
    return parsed


REPO_ENV = load_repo_env()


def env_or_repo_env(key: str, default: str) -> str:
    value = os.getenv(key, "").strip()
    if value:
        return value
    repo_value = REPO_ENV.get(key, "").strip()
    if repo_value:
        return repo_value
    return default


def rabbitmq_url() -> str:
    explicit = env_or_repo_env("RABBITMQ_URL", "")
    if explicit:
        return explicit
    user = quote(env_or_repo_env("RABBITMQ_USERNAME", "guest"), safe="")
    password = quote(env_or_repo_env("RABBITMQ_PASSWORD", "guest"), safe="")
    port = env_or_repo_env("RABBITMQ_PORT", "5672")
    return f"amqp://{user}:{password}@127.0.0.1:{port}/"


class SuiteError(RuntimeError):
    """Suite assertion error with context."""


def is_port_open(host: str, port: int, timeout: float = 0.5) -> bool:
    try:
        with socket.create_connection((host, port), timeout=timeout):
            return True
    except OSError:
        return False


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

    deadline = time.time() + 40
    while time.time() < deadline:
        if is_port_open("127.0.0.1", RABBIT_PORT):
            return
        time.sleep(0.4)
    raise SuiteError("RabbitMQ did not become ready on 5672")


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
            if parts and parts[-1].isdigit():
                pids.add(parts[-1])

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


def wait_http_ok(url: str, timeout_seconds: int = 80, headers: Optional[Dict[str, str]] = None) -> None:
    deadline = time.time() + timeout_seconds
    while time.time() < deadline:
        try:
            req = urllib.request.Request(url=url, method="GET", headers=headers or {})
            with urllib.request.urlopen(req, timeout=2) as resp:
                if resp.status == 200:
                    return
        except Exception:
            pass
        time.sleep(0.5)
    raise SuiteError(f"Service did not become healthy: {url}")


def wait_port_open(host: str, port: int, timeout_seconds: int = 80) -> None:
    deadline = time.time() + timeout_seconds
    while time.time() < deadline:
        if is_port_open(host, port):
            return
        time.sleep(0.3)
    raise SuiteError(f"Port did not become ready: {host}:{port}")


def api_raw(method: str, path: str, token: Optional[str] = None, payload: Any = None) -> Tuple[int, str]:
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
        with urllib.request.urlopen(req, timeout=30) as resp:
            return resp.status, resp.read().decode("utf-8")
    except urllib.error.HTTPError as exc:
        return exc.code, exc.read().decode("utf-8", errors="replace")


def api_json(method: str, path: str, token: Optional[str] = None, payload: Any = None) -> Any:
    status, body = api_raw(method, path, token=token, payload=payload)
    if status < 200 or status >= 300:
        raise SuiteError(f"{method} {path} failed: HTTP {status} {body}")
    if not body:
        return {}
    return json.loads(body)


def api_multipart_upload(path: str, token: str, file_name: str, file_bytes: bytes) -> Any:
    url = f"{API_BASE}{path}"
    boundary = f"----araneae-{uuid.uuid4().hex}"

    parts: List[bytes] = []
    parts.append(f"--{boundary}\r\n".encode("utf-8"))
    parts.append(
        (
            f'Content-Disposition: form-data; name="file"; filename="{file_name}"\r\n'
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
        with urllib.request.urlopen(req, timeout=40) as resp:
            raw = resp.read().decode("utf-8")
            return json.loads(raw) if raw else {}
    except urllib.error.HTTPError as exc:
        detail = exc.read().decode("utf-8", errors="replace")
        raise SuiteError(f"POST {path} upload failed: HTTP {exc.code} {detail}") from exc


def register_executor_node(token: str, node_key: str, queue_name: str) -> Dict[str, Any]:
    node = api_json(
        "POST",
        "/nodes/register/",
        token=token,
        payload={
            "ip": "127.0.0.1",
            "name": f"suite-executor-{queue_name}",
            "port": EXECUTOR_PORT,
            "grpc_port": 9190,
            "pair_key": node_key,
        },
    )
    if not node.get("id"):
        raise SuiteError(f"node registration returned invalid payload: {node}")
    return node


def create_crawler_zip() -> Tuple[str, bytes]:
    crawler_code = f'''#!/usr/bin/env python3
import json
import sys
import time
import urllib.error
import urllib.request
from datetime import datetime, timezone


def ts():
    return datetime.now(timezone.utc).isoformat()

marker = sys.argv[1] if len(sys.argv) > 1 else "default"
url = "http://127.0.0.1:{CONTROL_PORT}/healthz"

print(f"CRAWLER_MARKER={{marker}}")
print(f"CRAWLER_START={{ts()}}")
status = "ERR"
body = ""
try:
    with urllib.request.urlopen(url, timeout=4) as resp:
        status = str(resp.status)
        body = resp.read().decode("utf-8", errors="ignore")
except Exception as exc:
    body = str(exc)

# Make sure run has observable duration so started/finished checks are meaningful.
time.sleep(1.2)

print(f"CRAWLER_FETCH_STATUS={{status}}")
print("CRAWLER_FETCH_BODY=" + body[:180].replace("\\n", " "))
print(f"CRAWLER_END={{ts()}}")
'''

    readme = "Araneae schedule suite crawler artifact.\n"

    with tempfile.TemporaryDirectory(prefix="araneae-suite-zip-") as tmp:
        zip_path = Path(tmp) / "suite-crawler.zip"
        with zipfile.ZipFile(zip_path, "w", zipfile.ZIP_DEFLATED) as zf:
            zf.writestr("crawler.py", crawler_code)
            zf.writestr("README.txt", readme)
        return zip_path.name, zip_path.read_bytes()


def parse_time(value: Optional[str]) -> Optional[datetime]:
    if not value or not isinstance(value, str):
        return None
    normalized = value.replace("Z", "+00:00")
    try:
        return datetime.fromisoformat(normalized)
    except ValueError:
        return None


def assert_run_time_fields(run: Dict[str, Any], label: str) -> None:
    started = parse_time(run.get("started_at"))
    finished = parse_time(run.get("finished_at"))
    if started is None or finished is None:
        raise SuiteError(f"{label}: started_at/finished_at missing or invalid: {run}")
    if finished < started:
        raise SuiteError(f"{label}: finished_at earlier than started_at")


def wait_for_task_run(token: str, task_id: str, run_id: str, timeout_seconds: int = 120) -> Dict[str, Any]:
    deadline = time.time() + timeout_seconds
    while time.time() < deadline:
        runs = api_json("GET", f"/tasks/{task_id}/runs", token=token)
        for run in runs:
            if run.get("id") == run_id and run.get("status") in TERMINAL_STATUSES:
                return run
        time.sleep(1)
    raise SuiteError(f"Timed out waiting task run terminal state: task={task_id} run={run_id}")


def wait_for_schedule_terminal_runs(
    token: str,
    schedule_id: str,
    min_terminal_runs: int,
    timeout_seconds: int = 140,
) -> List[Dict[str, Any]]:
    deadline = time.time() + timeout_seconds
    while time.time() < deadline:
        runs = api_json("GET", f"/schedules/{schedule_id}/runs", token=token)
        terminal = [r for r in runs if r.get("status") in TERMINAL_STATUSES]
        if len(terminal) >= min_terminal_runs:
            return runs
        time.sleep(1)
    raise SuiteError(
        f"Timed out waiting schedule runs: schedule={schedule_id}, expected_terminal={min_terminal_runs}"
    )


def start_services(
    workdir: Path,
    queue_name: str,
    node_key: str,
) -> Tuple[subprocess.Popen, subprocess.Popen, Any, Any]:
    logs_dir = workdir / "logs"
    logs_dir.mkdir(parents=True, exist_ok=True)
    control_log = (logs_dir / "control.log").open("w", encoding="utf-8")
    executor_log = (logs_dir / "executor.log").open("w", encoding="utf-8")

    control_env = os.environ.copy()
    control_env.update(
        {
            "CONTROL_HTTP_ADDR": f":{CONTROL_PORT}",
            "CONTROL_GRPC_ADDR": f":{GRPC_PORT}",
            "CONTROL_DB_PATH": str(workdir / "control.db"),
            "ARTIFACT_ROOT": str(workdir / "artifacts"),
            "EXECUTION_CALLBACK_KEY": CALLBACK_KEY,
            "RABBITMQ_URL": rabbitmq_url(),
        }
    )

    executor_env = os.environ.copy()
    executor_env.update(
        {
            "EXECUTOR_HTTP_ADDR": f":{EXECUTOR_PORT}",
            "EXECUTOR_DB_PATH": str(workdir / "executor.db"),
            "EXECUTOR_WORKDIR": str(workdir / "workdir"),
            "EXECUTOR_QUEUE": queue_name,
            "EXECUTOR_NODE_KEY": node_key,
            "EXECUTOR_NODE_KEY_FILE": str(workdir / "executor.node.key"),
            "CONTROL_GRPC_TARGET": f"127.0.0.1:{GRPC_PORT}",
            "CONTROL_HTTP_BASE": CONTROL_BASE,
            "EXECUTION_CALLBACK_KEY": CALLBACK_KEY,
            "RABBITMQ_URL": rabbitmq_url(),
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


def create_task(token: str, project_id: str, version_id: str, queue_name: str, name: str, marker: str) -> Dict[str, Any]:
    cmd = f"python crawler.py {marker} || python3 crawler.py {marker}"
    return api_json(
        "POST",
        "/tasks",
        token=token,
        payload={
            "name": name,
            "project_id": project_id,
            "version_id": version_id,
            "entry_command": cmd,
            "cron_expr": "",
            "node_queue": queue_name,
        },
    )


def run_suite() -> Dict[str, Any]:
    ensure_rabbitmq()

    for port in (CONTROL_PORT, GRPC_PORT, EXECUTOR_PORT):
        kill_port_listener(port)

    runtime_dir = Path(tempfile.mkdtemp(prefix="araneae-go-schedule-suite-"))
    queue_name = f"suite-{uuid.uuid4().hex[:8]}"
    node_key = f"suite-node-key-{uuid.uuid4().hex}"

    control_proc, executor_proc, control_log, executor_log = start_services(runtime_dir, queue_name, node_key)
    try:
        wait_http_ok(f"{CONTROL_BASE}/healthz", timeout_seconds=100)
        wait_http_ok(
            f"http://127.0.0.1:{EXECUTOR_PORT}/healthz",
            timeout_seconds=100,
            headers={"X-Node-Key": node_key},
        )
        wait_port_open("127.0.0.1", GRPC_PORT, timeout_seconds=90)
        time.sleep(1.5)

        login = api_json("POST", "/auth/login", payload={"username": "admin", "password": "admin123"})
        token = login.get("token")
        if not token:
            raise SuiteError("Login did not return token")

        node = register_executor_node(token, node_key, queue_name)
        node_queue = node.get("celery_queue") or queue_name

        project = api_json("POST", "/projects", token=token, payload={"name": f"suite-project-{uuid.uuid4().hex[:8]}"})
        project_id = project["id"]

        zip_name, zip_data = create_crawler_zip()
        version = api_multipart_upload(path=f"/projects/{project_id}/upload", token=token, file_name=zip_name, file_bytes=zip_data)
        version_id = version["id"]

        task_single = create_task(token, project_id, version_id, node_queue, "suite-single-task", "single")
        task_chain_1 = create_task(token, project_id, version_id, node_queue, "suite-chain-task-1", "chain1")
        task_chain_2 = create_task(token, project_id, version_id, node_queue, "suite-chain-task-2", "chain2")
        task_cron = create_task(token, project_id, version_id, node_queue, "suite-cron-task", "cron")

        # 1) Single-task schedule create + update + trigger
        single_schedule = api_json(
            "POST",
            "/schedules",
            token=token,
            payload={
                "name": "suite-single-schedule",
                "description": "single schedule create/update/delete test",
                "enabled": False,
                "order": {
                    "name": "suite-single-schedule",
                    "schedule": [
                        {
                            "task_id": task_single["id"],
                            "trigger": "api",
                            "node": [node_queue],
                        }
                    ],
                },
            },
        )

        single_schedule_id = single_schedule["id"]

        updated_single = api_json(
            "PUT",
            f"/schedules/{single_schedule_id}",
            token=token,
            payload={
                "name": "suite-single-schedule-updated",
                "description": "updated description",
            },
        )
        if updated_single.get("name") != "suite-single-schedule-updated":
            raise SuiteError("single schedule update did not persist name")

        single_trigger = api_json("POST", f"/schedules/{single_schedule_id}/trigger", token=token)
        single_run = wait_for_task_run(token, task_single["id"], single_trigger["id"], timeout_seconds=120)
        if single_run.get("status") != "success":
            raise SuiteError(f"single schedule run failed: {single_run}")
        if "CRAWLER_MARKER=single" not in (single_run.get("output") or ""):
            raise SuiteError("single schedule run output missing crawler marker")
        assert_run_time_fields(single_run, "single_schedule_run")

        # 2) Multi-task chain schedule
        chain_schedule = api_json(
            "POST",
            "/schedules",
            token=token,
            payload={
                "name": "suite-chain-schedule",
                "description": "chain schedule test",
                "enabled": False,
                "order": {
                    "name": "suite-chain-schedule",
                    "schedule": [
                        {
                            "task_id": task_chain_1["id"],
                            "trigger": "api",
                            "node": [node_queue],
                        },
                        {
                            "task_id": task_chain_2["id"],
                            "trigger": "previous",
                            "node": [node_queue],
                        },
                    ],
                },
            },
        )

        chain_schedule_id = chain_schedule["id"]
        api_json("POST", f"/schedules/{chain_schedule_id}/trigger", token=token)
        chain_runs = wait_for_schedule_terminal_runs(token, chain_schedule_id, min_terminal_runs=2, timeout_seconds=140)
        chain_terminal = [r for r in chain_runs if r.get("status") in TERMINAL_STATUSES]

        chain_indices = sorted({int(r.get("chain_index", -1)) for r in chain_terminal if r.get("chain_id")})
        if not ({0, 1}.issubset(set(chain_indices))):
            raise SuiteError(f"chain schedule missing expected chain steps: {chain_indices}")

        outputs = "\n".join((r.get("output") or "") for r in chain_terminal)
        if "CRAWLER_MARKER=chain1" not in outputs or "CRAWLER_MARKER=chain2" not in outputs:
            raise SuiteError("chain run outputs missing expected markers")

        for idx, run in enumerate(chain_terminal[:2], start=1):
            assert_run_time_fields(run, f"chain_run_{idx}")

        # 3) Cron schedule timed auto start + finish
        cron_schedule = api_json(
            "POST",
            "/schedules",
            token=token,
            payload={
                "name": "suite-cron-schedule",
                "description": "cron timed start/end test",
                "enabled": True,
                "order": {
                    "name": "suite-cron-schedule",
                    "schedule": [
                        {
                            "task_id": task_cron["id"],
                            "trigger": "crons",
                            "crons": "*/8 * * * * *",
                            "node": [node_queue],
                        }
                    ],
                },
            },
        )

        cron_schedule_id = cron_schedule["id"]
        cron_runs = wait_for_schedule_terminal_runs(token, cron_schedule_id, min_terminal_runs=1, timeout_seconds=160)
        cron_terminal = [r for r in cron_runs if r.get("status") in TERMINAL_STATUSES]
        cron_latest = cron_terminal[0]
        if cron_latest.get("status") != "success":
            raise SuiteError(f"cron schedule latest run is not success: {cron_latest}")
        if "CRAWLER_MARKER=cron" not in (cron_latest.get("output") or ""):
            raise SuiteError("cron schedule run output missing crawler marker")
        assert_run_time_fields(cron_latest, "cron_schedule_run")

        # Disable cron before deletion to avoid new runs during cleanup.
        api_json("POST", f"/schedules/{cron_schedule_id}/disable", token=token)

        # 4) Schedule delete validation
        for sid in (single_schedule_id, chain_schedule_id, cron_schedule_id):
            api_json("DELETE", f"/schedules/{sid}", token=token)
            status, body = api_raw("GET", f"/schedules/{sid}", token=token)
            if status != 404:
                raise SuiteError(f"expected schedule {sid} to be deleted, got status={status}, body={body}")

        return {
            "ok": True,
            "runtime_dir": str(runtime_dir),
            "queue": node_queue,
            "node_id": node.get("id"),
            "checks": {
                "single_schedule_create": "PASS",
                "single_schedule_update": "PASS",
                "single_schedule_trigger": "PASS",
                "multi_task_chain": "PASS",
                "cron_timed_start_finish": "PASS",
                "schedule_delete": "PASS",
                "run_started_finished_fields": "PASS",
                "sample_crawler_execution": "PASS",
            },
            "artifacts": {
                "project_id": project_id,
                "version_id": version_id,
                "single_schedule_id": single_schedule_id,
                "chain_schedule_id": chain_schedule_id,
                "cron_schedule_id": cron_schedule_id,
            },
            "run_samples": {
                "single": {
                    "id": single_run.get("id"),
                    "status": single_run.get("status"),
                    "started_at": single_run.get("started_at"),
                    "finished_at": single_run.get("finished_at"),
                },
                "chain": [
                    {
                        "id": r.get("id"),
                        "status": r.get("status"),
                        "chain_index": r.get("chain_index"),
                        "started_at": r.get("started_at"),
                        "finished_at": r.get("finished_at"),
                    }
                    for r in chain_terminal[:2]
                ],
                "cron": {
                    "id": cron_latest.get("id"),
                    "status": cron_latest.get("status"),
                    "started_at": cron_latest.get("started_at"),
                    "finished_at": cron_latest.get("finished_at"),
                },
            },
        }
    finally:
        stop_process(executor_proc)
        stop_process(control_proc)
        control_log.close()
        executor_log.close()


def main() -> int:
    try:
        result = run_suite()
        print(json.dumps(result, ensure_ascii=False, indent=2))
        return 0
    except Exception as exc:
        print(json.dumps({"ok": False, "error": str(exc)}, ensure_ascii=False, indent=2))
        return 1


if __name__ == "__main__":
    raise SystemExit(main())
