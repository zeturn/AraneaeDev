# -*- coding: utf-8 -*-
# araneae_worknode - tasks.py
# ExecutionNode 任务执行模块

import os
import json
import hmac
import hashlib
import time
from datetime import datetime

import requests
import subprocess

from celery import shared_task

from config import create_app
from models import TaskRecord, Project, Version, db, Identity
import setting

flask_app = create_app()
celery_app = flask_app.extensions["celery"]

# 脚本仓库根目录，优先从配置/环境变量读取
_basedir = os.path.abspath(os.path.dirname(__file__))
REPO_BASE_PATH = (
    setting.REPO.get("BASE_PATH")
    if hasattr(setting, "REPO") and setting.REPO
    else os.path.join(_basedir, "repo")
)


################################################################
# 测试任务                                                     #
################################################################

@shared_task(ignore_result=False)
def long_running_task(iterations) -> int:
    """长时间运行的测试任务"""
    from time import sleep
    result = 0
    for i in range(iterations):
        result += i
        sleep(2)
    return result


################################################################
# 核心：接收并执行脚本                                         #
################################################################

@celery_app.task(name="execute_script")
def execute_script(*args, **kwargs):
    """
    接收 ControlNode 下发的任务并执行脚本。

    ControlNode 精准路由确保此消息只投递到目标节点，
    因此无需再进行节点白名单过滤。

    kwargs 格式：
      task_id      (str|int|None)  任务 ID
      chain_id     (int|None)      任务链 ID
      project_hash (str)           项目 hash
      version_hash (str)           版本 hash（支持 "LATEST"）
    """
    print("[execute_script] Received kwargs:", kwargs)

    task_id      = kwargs.get("task_id")
    chain_id     = kwargs.get("chain_id")
    project_hash = kwargs.get("project_hash")
    version_hash = kwargs.get("version_hash")

    # 1) 查找项目记录
    with flask_app.app_context():
        project = Project.query.filter_by(project_hash=project_hash).first()
        if not project:
            print(f"[execute_script] ❌ 项目不存在: project_hash={project_hash}")
            return None

        # 2) 解析版本 hash
        if version_hash == "LATEST":
            latest_version = Version.query.filter_by(
                project_id=project.id
            ).order_by(Version.id.desc()).first()
            if not latest_version:
                print(f"[execute_script] ❌ 项目 {project_hash} 无任何版本")
                return None
            version_hash = latest_version.version_hash

        # 3) 获取本节点 identity
        identity = Identity.query.first()
        node_hash = identity.identity_hash if identity else "unknown"

        # 4) 动态拼接脚本路径（从本节点 repo 目录读取）
        script_path = os.path.join(REPO_BASE_PATH, project_hash, version_hash, "main.py")
        print(f"[execute_script] 脚本路径: {script_path}")

        if not os.path.exists(script_path):
            _send_callback(node_hash, project_hash, version_hash, task_id, chain_id,
                           status="failed", result=f"Script not found: {script_path}")
            return None

        # 5) 执行前写 running 状态
        _update_or_create_task_record(
            node_hash=node_hash, project_hash=project_hash,
            version_hash=version_hash, task_id=task_id,
            task_chain_id=chain_id, task_status="running"
        )

    # 6) 在 flask 上下文外执行脚本（避免长阻塞持有 DB 连接）
    try:
        result = subprocess.run(
            ["python", script_path],
            capture_output=True, text=True, timeout=3600
        )
        stdout = result.stdout
        stderr = result.stderr
        exit_ok = (result.returncode == 0)
        final_status = "finished" if exit_ok else "failed"
        final_result = stdout if exit_ok else stderr
        print(f"[execute_script] 执行完毕，returncode={result.returncode}")
    except subprocess.TimeoutExpired:
        final_status = "timeout"
        final_result = "Script execution timed out."
        print("[execute_script] ❌ 执行超时")
    except Exception as e:
        final_status = "error"
        final_result = str(e)
        print(f"[execute_script] ❌ 执行异常: {e}")

    # 7) 回调 ControlNode 汇报结果（ExecutionNode 不负责任何链推进逻辑）
    _send_callback(
        node_hash=node_hash,
        project_hash=project_hash,
        version_hash=version_hash,
        task_id=task_id,
        chain_id=chain_id,
        status=final_status,
        result=final_result,
    )

    return final_result


################################################################
# 内部辅助函数                                                 #
################################################################

def _update_or_create_task_record(node_hash, project_hash, version_hash,
                                  task_id, task_chain_id, task_status, task_result=None):
    """写入本地 TaskRecord（ExecutionNode 侧的执行流水）"""
    with flask_app.app_context():
        record = TaskRecord(
            node_hash=node_hash,
            project_hash=project_hash,
            version_hash=version_hash,
            task_id=task_id,
            task_status=task_status,
            task_chain_id=task_chain_id,
            task_result=task_result,
            task_created_at=datetime.now(),
            task_updated_at=datetime.now(),
        )
        db.session.add(record)
        db.session.commit()
        return record.id


def _send_callback(node_hash, project_hash, version_hash,
                   task_id, chain_id, status, result=None):
    """
    向 ControlNode 发送 HMAC 签名回调，汇报执行结果。
    ControlNode 负责：更新 TaskRecord + 任务链推进。
    ExecutionNode 不携带任何链推进逻辑。
    """
    payload = {
        "node":         node_hash,
        "project":      project_hash,
        "version":      version_hash,
        "task_id":      task_id,
        "task_chain_id": chain_id,
        "task_status":  status,
        "task_result":  result,
    }

    body = json.dumps(payload, separators=(",", ":"), ensure_ascii=False)
    headers = {"Content-Type": "application/json"}

    secret = setting.SECURITY.get("CALLBACK_SHARED_SECRET", "")
    if secret:
        timestamp = str(int(time.time()))
        signature = hmac.new(
            secret.encode("utf-8"),
            f"{timestamp}.{body}".encode("utf-8"),
            hashlib.sha256,
        ).hexdigest()
        headers["X-Araneae-Timestamp"] = timestamp
        headers["X-Araneae-Signature"] = signature

    callback_url = f"{setting.CONTROLNODE['BASE_URL']}/api/task/callback/"
    try:
        resp = requests.post(
            callback_url,
            data=body.encode("utf-8"),
            headers=headers,
            timeout=10,
        )
        resp.raise_for_status()
        print(f"[callback] ✅ 回调成功: {resp.status_code}")
        return resp
    except requests.exceptions.HTTPError as e:
        print(f"[callback] ❌ HTTP 错误: {e}")
    except Exception as e:
        print(f"[callback] ❌ 回调失败: {e}")
