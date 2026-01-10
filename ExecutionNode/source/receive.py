# -*- coding: utf-8 -*-
# source/receive.py
import ast
import json
import re
import sys
import logging
from logging.handlers import RotatingFileHandler

import pika
import requests
import os
import zipfile

from models import ControlNode, Project, db, Version

#from app import app
#from process import create_project

sys.path.append(os.path.abspath(os.path.join(os.path.dirname(__file__), "..")))

import setting  # 现在可以正确导入

RABBITMQ_HOST = 'localhost'
QUEUE_NAME = 'file_transfer'
LOCAL_REPO = setting.LOCAL_REPO


def ensure_directory_exists(directory):
    """确保目录存在"""
    if not os.path.exists(directory):
        os.makedirs(directory, exist_ok=True)


def process_notification(ch, method, properties, body):
    """
    处理 RabbitMQ 通知
    Handle RabbitMQ notification
    """
    try:
        # === 以下功能：解码消息体 ===
        try:
            text = body.decode('utf-8')
        except UnicodeDecodeError as e:
            logging.error(f"[RECEIVE][ERROR] 无法解码消息体: {e}")
            return

        logging.info(f"[RECEIVE] Received notification: {text}")
        print(f"[RECEIVE] Received notification: {text}")

        # === 以下功能：预处理 Python 字典格式 ===
        # 将 <Identity: uuid> 替换为纯字符串形式
        # Replace <Identity: uuid> patterns with pure string
        text = re.sub(r"<Identity:\s*([0-9a-f\-]+)>", r'"\1"', text)

        # === 以下功能：将 Python None、True、False 转为 JSON 格式 ===
        # Replace Python literals None/True/False with JSON null/true/false
        text = re.sub(r"\bNone\b", "null", text)
        text = re.sub(r"\bTrue\b", "true", text)
        text = re.sub(r"\bFalse\b", "false", text)

        # === 以下功能：单引号转双引号 ===
        # Replace single quotes with double quotes for JSON compatibility
        text = text.replace("'", '"')

        # === 以下功能：JSON 解析 ===
        try:
            notification = json.loads(text)
        except json.JSONDecodeError as e:
            logging.error(f"[RECEIVE][ERROR] JSON 解析失败: {e}")
            return

        if not isinstance(notification, dict):
            logging.error(f"[RECEIVE][ERROR] 解析结果不是字典: {notification!r}")
            return

        action = notification.get('action')
        node_identity = notification.get('node_identity')
        file_hash = notification.get('file_hash')
        file_url = notification.get('file_url')
        project_id = notification.get('project_hash')
        project_hash = notification.get('project_hash')
        project_name = notification.get('project_name')
        project_description = notification.get('project_description')
        project_programming_language = notification.get('project_programming_language')
        project_command = notification.get('project_command')
        project_mode = notification.get('project_mode')
        project_entrance_file = notification.get('project_entrance_file')
        version = notification.get('version')

        # === 以下功能：根据 node_identity 查找控制节点 ===
        control_node = ControlNode.query.filter_by(node_hash=node_identity).first()
        if not control_node:
            logging.error(f"[RECEIVE][ERROR] Node with identity {node_identity} not found.")
            return

        logging.info(f"[RECEIVE] Project {project_hash}({project_id}), Version {version}, File URL: {file_url}")

        # === 以下功能：下载并解压项目文件 ===
        project_path = os.path.join(LOCAL_REPO, str(project_hash))
        version_path = os.path.join(project_path, version)
        ensure_directory_exists(version_path)

        compressed_file = os.path.join(project_path, f"{version}.zip")
        response = requests.get(file_url, stream=True)
        if response.status_code != 200:
            logging.error(f"[RECEIVE][ERROR] 下载失败: {file_url}, 状态码: {response.status_code}")
            return

        with open(compressed_file, 'wb') as f:
            for chunk in response.iter_content(chunk_size=1024):
                f.write(chunk)
        logging.info(f"[RECEIVE] Downloaded file: {compressed_file}")

        with zipfile.ZipFile(compressed_file, 'r') as zipf:
            zipf.extractall(version_path)
        logging.info(f"[RECEIVE] Extracted files to: {version_path}")


        # Using the project_hash to find or create a project
        existing_project = Project.query.filter_by(project_hash=project_hash).first()

        if existing_project:
            project = existing_project
            print(f"[RECEIVE] Project already exists: {project}")
        else:
            project = Project(
                name=project_name,
                description=project_description,
                project_hash = project_hash,
                source_node=control_node.id,
                source_project=project_id,
                source_workplace=control_node.id,
                language=project_programming_language,
                command=project_command,
                mode=project_mode,
            )
            logging.info(f"[RECEIVE] Project create: {project}")
            db.session.add(project)
            db.session.commit()

        version = Version(
            project_id=project.id,
            version_hash=version,
            source_node=control_node.id,
            description=project_description,
        )
        db.session.add(version)
        db.session.commit()

        logging.info(f"[RECEIVE] Project and Version created successfully with ID: {project.id}")

        # delete temporary compressed file
        os.remove(compressed_file)
        logging.info(f"[RECEIVE] Deleted compressed file: {compressed_file}")

    except Exception as e:
        logging.error(f"[RECEIVE][ERROR] unknown error: {e}")
        return



def listen_for_notifications():
    """
    监听 RabbitMQ 队列
    :return: None

    """
    print("[RECEIVE] Initializing RabbitMQ listener...")
    logging.info("[RECEIVE] Initializing RabbitMQ listener...")
    """监听 RabbitMQ 队列"""
    connection = pika.BlockingConnection(pika.ConnectionParameters(
        host='199.7.140.120',
        port=5673,
        credentials=pika.PlainCredentials('guest', '54321Ssdlh!!')
    ))
    channel = connection.channel()
    channel.queue_declare(queue=QUEUE_NAME)
    channel.basic_consume(queue=QUEUE_NAME, on_message_callback=process_notification, auto_ack=True)
    print("[RECEIVE] Listening for file transfer notifications...")
    logging.info("[RECEIVE] Listening for file transfer notifications...")
    channel.start_consuming()


def main():
    """
    启动接收服务
    """
    print("[RECEIVE] Starting file transfer listener...")
    listen_for_notifications()


if __name__ == "__main__":
    main()
