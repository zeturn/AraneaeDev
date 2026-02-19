# -*- coding: utf-8 -*-
# araneae_worknode - tasks.py.py
# Created by zhr62 at 2025/2/20 - 下午3:40
import os
import json
import hmac
import hashlib
import time
from datetime import datetime
from time import sleep
import requests
import subprocess

from celery import shared_task
from kombu import Queue

from config import create_app
from models import TaskRecord, Project, Version, db, Identity
import setting
from utils.identity import get_hash

flask_app = create_app()
celery_app = flask_app.extensions["celery"]

@shared_task(ignore_result=False)
def long_running_task(iterations) -> int:
    """
    长时间运行的任务 [测试]
    @param iterations:
    @return:
    """
    result = 0
    for i in range(iterations):
        result += i
        sleep(2)
    return result  # -Line 6


@celery_app.task(name="execute_script")
def execute_script(*args, **kwargs):
    """
    执行 Python 脚本
    :param args:
    :param kwargs:
    :return:
    """
    print("Received args:", args)
    print("Received kwargs:", kwargs)

    # Continue with the rest of your code...
    task_id   = kwargs.get("task_id")
    node_list = kwargs.get("nodes")
    project_hash = kwargs.get("project_hash")
    project_hash = kwargs.get("project_hash")
    version_hash = kwargs.get("version_hash")
    chain_id = kwargs.get("chain_id")

    #current_node = get_hash()
    current_node = Identity.query.first().identity_hash  # For testing, replace with actual node identity retrieval

    if current_node not in node_list:
        print("Current node is not in the node list, aborting task execution.")
        return None
    else:
        print("Current node is in the node list, proceeding with task execution.")

        project = Project.query.filter_by(project_hash=project_hash).first()
        if not project:
            raise ValueError(f"Project with ID {project_hash} does not exist.")
            return None
        else:
            project_hash = project.project_hash

        project_id = project.id

        if version_hash == "LATEST":
            latest_version = Version.query.filter_by(project_id=project_id).first()
            if not latest_version:
                raise ValueError(f"No latest version found for project ID {project_hash}.")
            version_hash = latest_version.version_hash

        # For testing, you force a specific path (this may be temporary)
        script_path = fr"C:\Users\zhr62\PycharmProjects\djangoProject\Araneae_worknode\repo\{project_hash}\{version_hash}\helloworld.py"
        print("Script path:", script_path)

        if not script_path:
            raise ValueError("The script path must be provided in kwargs.")

        command = ["python", script_path]
        print(f"Command to run: {command}")

        try:
            result = subprocess.run(command, capture_output=True, text=True, check=True)

            # Update task record and send callback within flask_app context...
            with flask_app.app_context():
                task_record_id = create_task_record(
                    node_hash = node_list[0],
                    project_hash = project.project_hash,
                    version_hash = version_hash,
                    task_id=task_id,
                    task_chain_id=chain_id,
                    task_status="running"
                )

                task_record = TaskRecord.query.get(task_record_id)
                task_record.task_status = "finished"
                print("Task record created:", task_record)

                send_callback(task_record)

            return result.stdout
        except subprocess.CalledProcessError as e:
            return f"Script execution failed: {e}"




def create_task_record(node_hash, project_hash, version_hash, task_id, task_status, task_chain_id, task_result=None):
    """
    创建任务记录
    :param node_hash:  节点哈希
    :param project_hash: 项目哈希
    :param version_hash: 版本哈希
    :param task_id: 任务ID
    :param task_status: 任务状态
    :param task_result: 任务结果
    :return: TaskRecord任务记录
    """

    task_record = TaskRecord(
        node_hash = node_hash,
        project_hash = project_hash,
        version_hash = version_hash,
        task_id = task_id,
        task_status = task_status,
        task_chain_id = task_chain_id,
        task_result = task_result,
        task_created_at = datetime.now(),
        task_updated_at = datetime.now()
    )
    db.session.add(task_record)
    db.session.commit()

    return task_record.id

def send_callback(task_record):
    """
    向http://127.0.0.1:8000/api/task/callback/ 发送回调request
    :param task_record:
    :return: response
    """
    node = task_record.node_hash
    project = task_record.project_hash
    version = task_record.version_hash
    task_status = task_record.task_status
    task_id = task_record.task_id
    task_result = task_record.task_result
    task_chain_id = task_record.task_chain_id

    request = {
        "node": node,
        "project": project,
        "version": version,
        "task_status": task_status,
        "task_id": task_id,
        "task_result": task_result,
        "task_chain_id": task_chain_id,
    }

    body = json.dumps(request, separators=(",", ":"), ensure_ascii=False)
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
        response = requests.post(callback_url, data=body.encode("utf-8"), headers=headers, timeout=10)
        response.raise_for_status()  # 如果响应状态码不是200，抛出HTTPError
    except requests.exceptions.HTTPError as http_err:
        print(f"HTTP error occurred: {http_err}")
    except Exception as err:
        print(f"Other error occurred: {err}")
    else:
        print("Callback response:", response)
        return response


if __name__ == "__main__":
    node_hash = identity_hash = get_hash()  # For testing, replace with actual node identity retrieval
    args = []
    kwargs = {
        "task_id": "12345",
        "nodes": [node_hash],
        "project_hash": 1,
        "project_hash": "wgewjD",
        "version_hash": "LATEST",
        "chain_id": "123"
    }

    print(execute_script(*args, **kwargs))
