#!/usr/bin/env python3
"""Araneae -> HashSlip sink integration E2E.

Flow:
1) Ensure RabbitMQ + HashSlip docker services are up
2) Start Araneae control/executor (local go run)
3) Upload a crawler artifact that uses `araneae_sink.emit_timeseries`
4) Trigger task
5) Verify HashSlip contains the emitted timeseries point
"""

from __future__ import annotations

import json
import os
import socket
import subprocess
import tempfile
import time
import urllib.error
import urllib.parse
import urllib.request
import uuid
import zipfile
from pathlib import Path
from typing import Any, Dict, Optional, Tuple

ROOT = Path(__file__).resolve().parents[1]
WORKSPACE = ROOT.parent.parent
HASHSLIP_ROOT = WORKSPACE / "HashSlip"

CONTROL_PORT = 18280
GRPC_PORT = 19190
EXECUTOR_PORT = 14280
RABBIT_PORT = 5672
HASHSLIP_PORT = 8106

CONTROL_BASE = f"http://127.0.0.1:{CONTROL_PORT}"
API_BASE = f"{CONTROL_BASE}/api/v1"
HASHSLIP_BASE = f"http://127.0.0.1:{HASHSLIP_PORT}"


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
    user = urllib.parse.quote(env_or_repo_env("RABBITMQ_USERNAME", "guest"), safe="")
    password = urllib.parse.quote(env_or_repo_env("RABBITMQ_PASSWORD", "guest"), safe="")
    port = env_or_repo_env("RABBITMQ_PORT", "5672")
    return f"amqp://{user}:{password}@127.0.0.1:{port}/"


def is_port_open(host: str, port: int, timeout: float = 0.5) -> bool:
    try:
        with socket.create_connection((host, port), timeout=timeout):
            return True
    except OSError:
        return False


def ensure_rabbitmq() -> None:
    if is_port_open("127.0.0.1", RABBIT_PORT):
        return
    subprocess.run(["docker", "compose", "up", "-d", "rabbitmq"], cwd=str(ROOT), check=True)
    deadline = time.time() + 40
    while time.time() < deadline:
        if is_port_open("127.0.0.1", RABBIT_PORT):
            return
        time.sleep(0.5)
    raise RuntimeError("rabbitmq not ready")


def ensure_hashslip() -> None:
    if is_port_open("127.0.0.1", HASHSLIP_PORT):
        return
    subprocess.run(
        ["docker", "compose", "up", "-d", "--build", "timescaledb", "textdb", "hashslip"],
        cwd=str(HASHSLIP_ROOT),
        check=True,
    )
    deadline = time.time() + 180
    while time.time() < deadline:
        try:
            with urllib.request.urlopen(f"{HASHSLIP_BASE}/healthz", timeout=3) as resp:
                if resp.status == 200:
                    return
        except Exception:
            pass
        time.sleep(1)
    raise RuntimeError("hashslip not ready")


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
            if "LISTENING" not in row or target not in row:
                continue
            pid = row.split(" ")[-1].strip()
            if pid.isdigit():
                pids.add(pid)
        for pid in pids:
            subprocess.run(["taskkill", "/PID", pid, "/T", "/F"], stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
        return


def wait_http_ok(url: str, timeout: int = 80, headers: Optional[Dict[str, str]] = None) -> None:
    deadline = time.time() + timeout
    while time.time() < deadline:
        try:
            req = urllib.request.Request(url=url, method="GET", headers=headers or {})
            with urllib.request.urlopen(req, timeout=3) as resp:
                if resp.status == 200:
                    return
        except Exception:
            pass
        time.sleep(0.5)
    raise RuntimeError(f"service not healthy: {url}")


def wait_port_open(host: str, port: int, timeout: int = 60) -> None:
    deadline = time.time() + timeout
    while time.time() < deadline:
        if is_port_open(host, port):
            return
        time.sleep(0.3)
    raise RuntimeError(f"port not open: {host}:{port}")


def api_json(method: str, path: str, token: str = "", payload: Any = None) -> Any:
    headers = {"Accept": "application/json"}
    body = None
    if payload is not None:
        body = json.dumps(payload).encode("utf-8")
        headers["Content-Type"] = "application/json"
    if token:
        headers["Authorization"] = f"Bearer {token}"
    req = urllib.request.Request(f"{API_BASE}{path}", method=method, headers=headers, data=body)
    try:
        with urllib.request.urlopen(req, timeout=20) as resp:
            raw = resp.read().decode("utf-8")
            return json.loads(raw) if raw else {}
    except urllib.error.HTTPError as exc:
        detail = exc.read().decode("utf-8", errors="replace")
        raise RuntimeError(f"{method} {path} failed: {exc.code} {detail}") from exc


def api_upload(path: str, token: str, file_name: str, content: bytes) -> Any:
    boundary = f"----araneae-{uuid.uuid4().hex}"
    body = (
        f"--{boundary}\r\n"
        f'Content-Disposition: form-data; name="file"; filename="{file_name}"\r\n'
        f"Content-Type: application/octet-stream\r\n\r\n"
    ).encode("utf-8") + content + f"\r\n--{boundary}--\r\n".encode("utf-8")
    req = urllib.request.Request(
        f"{API_BASE}{path}",
        method="POST",
        data=body,
        headers={
            "Authorization": f"Bearer {token}",
            "Accept": "application/json",
            "Content-Type": f"multipart/form-data; boundary={boundary}",
            "Content-Length": str(len(body)),
        },
    )
    try:
        with urllib.request.urlopen(req, timeout=30) as resp:
            raw = resp.read().decode("utf-8")
            return json.loads(raw) if raw else {}
    except urllib.error.HTTPError as exc:
        detail = exc.read().decode("utf-8", errors="replace")
        raise RuntimeError(f"upload failed: {exc.code} {detail}") from exc


def create_sink_artifact() -> Tuple[str, bytes, str]:
    hash_key = "fed_overnight_rate_daily_e2e"
    script = f"""#!/usr/bin/env python3
from datetime import datetime, timezone
import araneae_sink

now = datetime.now(timezone.utc)
araneae_sink.emit_timeseries(
    source="araneae.e2e",
    metric="fed_overnight_rate",
    timestamp=now.isoformat(),
    value=5.23,
    tags={{"series":"SOFR"}},
    payload={{"mode":"e2e"}},
    hash_key="{hash_key}",
    bucket_date=now.strftime("%Y-%m-%d"),
)
print("sink emitted")
"""
    with tempfile.TemporaryDirectory(prefix="araneae-sink-e2e-zip-") as tmp:
        p = Path(tmp) / "sink-e2e.zip"
        with zipfile.ZipFile(p, "w", zipfile.ZIP_DEFLATED) as zf:
            zf.writestr("crawler.py", script)
        return p.name, p.read_bytes(), hash_key


def start_services(workdir: Path, queue: str, node_key: str):
    logs = workdir / "logs"
    logs.mkdir(parents=True, exist_ok=True)
    control_log = (logs / "control.log").open("w", encoding="utf-8")
    executor_log = (logs / "executor.log").open("w", encoding="utf-8")

    callback_key = "sink-e2e-callback-key-abcdefghijklmnopqrstuvwxyz12345"
    admin_password = "SinkE2E_AdminPassword_2026!"
    control_env = os.environ.copy()
    control_env.update(
        {
            "CONTROL_HTTP_ADDR": f":{CONTROL_PORT}",
            "CONTROL_GRPC_ADDR": f":{GRPC_PORT}",
            "CONTROL_DB_PATH": str(workdir / "control.db"),
            "ARTIFACT_ROOT": str(workdir / "artifacts"),
            "EXECUTION_CALLBACK_KEY": callback_key,
            "INIT_ADMIN_PASSWORD": admin_password,
            "CONTROL_JWT_SECRET": "abcdefghijklmnopqrstuvwxyz123456",
            "RABBITMQ_URL": rabbitmq_url(),
        }
    )
    executor_env = os.environ.copy()
    executor_env.update(
        {
            "EXECUTOR_HTTP_ADDR": f":{EXECUTOR_PORT}",
            "EXECUTOR_DB_PATH": str(workdir / "executor.db"),
            "EXECUTOR_WORKDIR": str(workdir / "workdir"),
            "EXECUTOR_QUEUE": queue,
            "EXECUTOR_NODE_KEY": node_key,
            "EXECUTOR_NODE_KEY_FILE": str(workdir / "executor.node.key"),
            "CONTROL_GRPC_TARGET": f"127.0.0.1:{GRPC_PORT}",
            "CONTROL_HTTP_BASE": CONTROL_BASE,
            "EXECUTION_CALLBACK_KEY": callback_key,
            "RABBITMQ_URL": rabbitmq_url(),
            "HASHSLIP_BASE_URL": HASHSLIP_BASE,
            "EXECUTOR_SINK_ENABLED": "true",
            "EXECUTOR_SINK_STRICT": "true",
        }
    )

    control = subprocess.Popen(
        ["go", "run", "./cmd/control"], cwd=str(ROOT), env=control_env, stdout=control_log, stderr=subprocess.STDOUT
    )
    executor = subprocess.Popen(
        ["go", "run", "./cmd/executor"], cwd=str(ROOT), env=executor_env, stdout=executor_log, stderr=subprocess.STDOUT
    )
    return control, executor, control_log, executor_log, admin_password


def stop_proc(proc: subprocess.Popen):
    if proc.poll() is not None:
        return
    if os.name == "nt":
        subprocess.run(["taskkill", "/PID", str(proc.pid), "/T", "/F"], stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
    else:
        proc.terminate()


def wait_task_run(task_id: str, run_id: str, token: str, timeout: int = 120):
    deadline = time.time() + timeout
    while time.time() < deadline:
        runs = api_json("GET", f"/tasks/{task_id}/runs", token=token)
        for run in runs:
            if run.get("id") == run_id and run.get("status") in {"success", "failed"}:
                return run
        time.sleep(1)
    raise RuntimeError("task run timeout")


def main() -> int:
    ensure_rabbitmq()
    ensure_hashslip()
    for p in (CONTROL_PORT, GRPC_PORT, EXECUTOR_PORT):
        kill_port_listener(p)

    runtime_dir = Path(tempfile.mkdtemp(prefix="araneae-sink-e2e-"))
    queue = f"sink-{uuid.uuid4().hex[:8]}"
    node_key = f"sink-node-{uuid.uuid4().hex}"
    control, executor, control_log, executor_log, admin_password = start_services(runtime_dir, queue, node_key)
    try:
        wait_http_ok(f"{CONTROL_BASE}/healthz", timeout=90)
        wait_http_ok(f"http://127.0.0.1:{EXECUTOR_PORT}/healthz", timeout=90, headers={"X-Node-Key": node_key})
        wait_port_open("127.0.0.1", GRPC_PORT, timeout=60)

        login = api_json("POST", "/auth/login", payload={"username": "admin", "password": admin_password})
        token = login["token"]

        node = api_json(
            "POST",
            "/nodes/register/",
            token=token,
            payload={"ip": "127.0.0.1", "name": "sink-e2e-node", "port": EXECUTOR_PORT, "grpc_port": 9190, "pair_key": node_key},
        )
        node_queue = node.get("celery_queue") or queue

        project = api_json("POST", "/projects", token=token, payload={"name": f"sink-e2e-{uuid.uuid4().hex[:6]}"})
        zip_name, zip_data, hash_key = create_sink_artifact()
        version = api_upload(f"/projects/{project['id']}/upload", token, zip_name, zip_data)

        task = api_json(
            "POST",
            "/tasks",
            token=token,
            payload={
                "name": "sink-e2e-task",
                "project_id": project["id"],
                "version_id": version["id"],
                "entry_command": "python crawler.py || python3 crawler.py",
                "node_queue": node_queue,
            },
        )
        run = api_json("POST", f"/tasks/{task['id']}/trigger", token=token)
        run_result = wait_task_run(task["id"], run["id"], token)
        if run_result.get("status") != "success":
            raise RuntimeError(f"task failed: {run_result.get('output')}")

        query = urllib.parse.urlencode({"hash_key": hash_key, "limit": "5"})
        with urllib.request.urlopen(f"{HASHSLIP_BASE}/api/v1/timeseries/records?{query}", timeout=20) as resp:
            rows = json.loads(resp.read().decode("utf-8"))
        items = rows.get("items") or []
        if not items:
            raise RuntimeError("hashslip has no sink data")

        print(json.dumps({"ok": True, "hash_key": hash_key, "rows": len(items), "latest": items[0]}, ensure_ascii=False, indent=2))
        return 0
    finally:
        stop_proc(executor)
        stop_proc(control)
        control_log.close()
        executor_log.close()


if __name__ == "__main__":
    raise SystemExit(main())

