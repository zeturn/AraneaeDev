"""
# Araneae Execution Node
"""
# -*- coding: utf-8 -*-
# app.py
# flask run --host=0.0.0.0 --port=5000

import os
import signal
import sys
import threading
import time
from functools import wraps

import subprocess
import multiprocessing

import logging
from logging.handlers import RotatingFileHandler

from flask import Flask, jsonify, request, send_from_directory
import asyncio
from models import db, Project, Version, ControlNode, TaskRecord, Identity
import setting
from machine.info import get_system_info

from source.repo.repo_scaner import get_project_versions

from grpc_server import serve as grpc_serve

sys.path.append(os.path.abspath(os.path.join(os.path.dirname(__file__), "")))
from webrtc import api_create_session, api_offer, api_resume  # WebRTC service

################################################################
# Araneae_worknode Flask 应用                                  #
################################################################
app = Flask(__name__)

# 数据库配置
basedir = os.path.abspath(os.path.dirname(__file__))
_db_path = os.environ.get('EXECUTION_DB_PATH') or os.path.join(basedir, 'db.sqlite')
app.config['SQLALCHEMY_DATABASE_URI'] = f'sqlite:///{_db_path}'
app.config['SQLALCHEMY_TRACK_MODIFICATIONS'] = False
app.config['SQLALCHEMY_ECHO'] = True

# 初始化数据库
db.init_app(app)

# 设置日志记录器
log_dir = os.path.join(basedir, 'logs')
os.makedirs(log_dir, exist_ok=True)
log_file = os.path.join(log_dir, 'app.log')

file_handler = RotatingFileHandler(log_file, maxBytes=10 * 1024 * 1024, backupCount=5)
file_handler.setLevel(logging.INFO)
formatter = logging.Formatter('%(asctime)s [%(levelname)s] %(message)s')
file_handler.setFormatter(formatter)

app.logger.addHandler(file_handler)
app.logger.setLevel(logging.INFO)


def require_node_token(view_func):
    @wraps(view_func)
    def _wrapped(*args, **kwargs):
        expected = setting.SECURITY.get("NODE_API_TOKEN", "")
        if expected:
            provided = request.headers.get("X-Araneae-Node-Token", "")
            if not provided or provided != expected:
                return jsonify({"error": "Unauthorized"}), 401
        return view_func(*args, **kwargs)
    return _wrapped


@app.before_request
def log_request_info():
    """
    zh-cn: 记录请求信息
    en: Log request information
    """
    app.logger.info(f"Incoming request: {request.method} {request.url}")
    app.logger.info(f"Request body size: {len(request.data or b'')}")


@app.after_request
def log_response_info(response):
    """
    zh-cn: 记录响应信息
    en: Log response information
    @param response:
    @return:
    """
    app.logger.info(f"Response: {response.status_code}")
    return response


@app.errorhandler(Exception)
def handle_exception(e):
    """

    @param e:
    @return:
    """
    app.logger.error(f"Error: {str(e)}", exc_info=True)
    return jsonify({"error": "Internal Server Error"}), 500


################################################################
# Araneae_worknode 业务                                        #
################################################################
@app.route('/', methods=['GET'])
def index():
    """
    Index page
    @return: str
    """
    result = add_together(3, 5)
    return f"Task sent! Task result: {result}"


def add_together(a, b):
    """
    Add two numbers
    @param a:
    @param b:
    @return:
    """
    return a + b


# 路由
@app.route('/projects', methods=['POST'])
@require_node_token
def create_project():
    """
    [Araneae]创建项目
    @return:
    """
    data = request.json
    try:
        project = Project(
            workplace_id=data['workplace_id'],
            name=data['name'],
            description=data.get('description'),
            language=data['language'],
            command=data['command'],
            mode=data['mode']
        )
        db.session.add(project)
        db.session.commit()
        app.logger.info(f"Project created: {project.id}")
        return jsonify({'message': 'Project created!', 'project': project.id}), 201
    except Exception as e:
        db.session.rollback()
        app.logger.error(f"Failed to create project: {str(e)}")
        return jsonify({'error': 'Failed to create project'}), 500


@app.route('/projects/<int:project_id>', methods=['GET'])
def get_project(project_id):
    """
    [Araneae]获取项目
    @param project_hash:
    @return:
    """
    project = Project.query.get_or_404(project_id)
    app.logger.info(f"Fetched project: {project.id}")
    return jsonify({
        'id': project.id,
        'workplace_id': project.workplace_id,
        'name': project.name,
        'description': project.description,
        'language': project.language,
        'command': project.command,
        'mode': project.mode,
        'created_at': project.created_at,
        'updated_at': project.updated_at
    })

@app.route('/projects/versions/list', methods=['GET'])
def list_projects():
    """
    [Araneae]列出所有项目
    """
    return jsonify(get_project_versions())

@app.route('/projects/versions/<project_id>', methods=['GET'])
def list_project_versions(project_id):
    """
    [Araneae] 列出指定项目的所有版本
    """
    versions = get_project_versions(project_id)
    return jsonify({"project_hash": project_id, "versions": versions})

@app.route('/favicon.ico')
def favicon():
    """
    [Araneae]获取 favicon.ico
    :return:
    """
    return send_from_directory('static', 'favicon.ico', mimetype='image/vnd.microsoft.icon')

@app.route('/system_info', methods=['GET'])
def system_info():
    """
    [Araneae]获取系统信息
    @return:
    """
    info = get_system_info()
    app.logger.info("System info fetched")
    return jsonify(info)


process_registry = {}


@app.route('/run_task', methods=['POST'])
@require_node_token
def run_task():
    """
    [Araneae]运行任务
    未测试
    @return:
    """
    data = request.json
    project_path = data['project_path']
    script_name = data['script_name']

    process = subprocess.Popen(
        ['python', f'{project_path}/{script_name}'],
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE
    )

    process_registry[process.pid] = process
    return jsonify({"status": "running", "pid": process.pid}), 200


@app.route('/check_status/<int:pid>', methods=['GET'])
def check_status(pid):
    """
    [Araneae]检查任务状态
    未测试

    @param pid:
    @return:
    """
    if pid in process_registry:
        process = process_registry[pid]
        if process.poll() is None:
            return jsonify({"status": "running"}), 200
        else:
            return jsonify({"status": "completed"}), 200
    else:
        return jsonify({"status": "not_found"}), 404


################################################################
# WebRTC + Playwright Assisted Control                         #
################################################################
@app.route('/webrtc/session', methods=['POST'])
@require_node_token
def webrtc_create_session():
    """
    Create a Playwright session and return session_id
    Body: {"url": "https://example.com", "headless": true}
    """
    data = request.get_json(force=True) or {}
    url = data.get('url')
    headless = bool(data.get('headless', True))
    if not url:
        return jsonify({"error": "url is required"}), 400
    try:
        result = asyncio.run(api_create_session(url=url, headless=headless))
        return jsonify(result), 201
    except Exception as e:
        app.logger.error(f"webrtc_create_session failed: {e}")
        return jsonify({"error": str(e)}), 500


@app.route('/webrtc/offer', methods=['POST'])
@require_node_token
def webrtc_offer():
    """
    Accept SDP offer and return SDP answer.
    Body: {"session_id": "abc", "sdp": "...", "type": "offer"}
    """
    data = request.get_json(force=True) or {}
    session_id = data.get('session_id')
    sdp = data.get('sdp')
    sdp_type = data.get('type', 'offer')
    if not session_id or not sdp:
        return jsonify({"error": "session_id and sdp are required"}), 400
    try:
        answer = asyncio.run(api_offer(session_id=session_id, offer_sdp=sdp, offer_type=sdp_type))
        return jsonify(answer), 200
    except Exception as e:
        app.logger.error(f"webrtc_offer failed: {e}")
        return jsonify({"error": str(e)}), 500


@app.route('/webrtc/resume/<session_id>', methods=['POST'])
@require_node_token
def webrtc_resume(session_id: str):
    try:
        res = asyncio.run(api_resume(session_id=session_id))
        return jsonify(res), 200
    except Exception as e:
        app.logger.error(f"webrtc_resume failed: {e}")
        return jsonify({"error": str(e)}), 500


################################################################
# Araneae_worknode 启动                                         #
################################################################
# 运行 receive.py
def run_receive():
    """启动 receive.py，并在失败时重启（最多 3 次）"""
    max_retries = 3
    retry_count = 0
    # 确保可以导入 receive.py
    source_dir = os.path.join(os.path.dirname(os.path.abspath(__file__)), "source")
    if source_dir not in sys.path:
        sys.path.append(source_dir)

    try:
        from receive import main  # 假设 receive.py 有 main 函数
        while retry_count < max_retries:
            try:
                ctx = app.app_context()
                ctx.push()
                main()
                break  # 如果 main() 成功执行，退出循环
            except Exception as e:
                retry_count += 1
                app.logger.error(f"Error in receive.py: {e}. Retrying {retry_count}/{max_retries}...")
                time.sleep(5)  # 等待 5 秒后重试
    except ImportError as e:
        print(f"[FATAL] 无法导入 receive.py: {e}")
        return


# Araneae_worknode
# 启动 Celery

# Celery 进程变量
celery_worker_process = None
celery_beat_process = None


def start_celery_worker():
    """
    启动 Celery Worker

    """
    global celery_worker_process
    if celery_worker_process is None or celery_worker_process.poll() is not None:
        print("[INFO] 启动 Celery Worker...")
        multiprocessing.set_start_method('spawn', force=True)
        celery_worker_process = subprocess.Popen(
            ["celery", "-A", "tasks", "worker",
             "-Q", "public_channel",
             "--loglevel=info",
             "--pool=solo"
             ])  # --pool=solo 解决了 Windows 下的权限问题
    else:
        print("Celery Worker 已在运行")


def start_celery_beat():
    """
    启动 Celery Beat

    """
    global celery_beat_process
    if celery_beat_process is None or celery_beat_process.poll() is not None:
        print("[INFO] 启动 Celery Beat...")
        multiprocessing.set_start_method('spawn', force=True)
        celery_beat_process = subprocess.Popen(
            ["celery", "-A", "tasks", "beat", "--loglevel=info", "--pool=solo"])
    else:
        print("[INFO]Celery Beat 已在运行")


def stop_celery():
    """
    停止 Celery Worker 和 Celery Beat

    """
    global celery_worker_process, celery_beat_process
    print("[INFO] 正在关闭 Celery...")
    if celery_worker_process:
        celery_worker_process.terminate()
        celery_worker_process.wait()
    if celery_beat_process:
        celery_beat_process.terminate()
        celery_beat_process.wait()
    print("[INFO] Celery 已关闭")


def signal_handler(sig, frame):
    """
    处理终止信号
    @param sig:
    @param frame:
    """
    print("\n[INFO] 收到终止信号，正在关闭 celery 进程...")
    stop_celery()
    sys.exit(0)

def start_grpc_services():
    """
    启动 gRPC 服务
    """
    def _serve_with_ctx():
        with app.app_context():
            grpc_serve()

    grpc_thread = threading.Thread(target=_serve_with_ctx, daemon=True)
    grpc_thread.start()


def stop_grpc_services():
    """
        停止 gRPC 服务
    """
    try:
        grpc_thread = threading.Thread(target=grpc_serve, daemon=True)
        grpc_thread.join(timeout=1)
    except Exception as e:
        # 中文：gRPC 服务停止异常处理
        # 英文：Handle exceptions when stopping gRPC server
        app.logger.error(f'Failed to stop gRPC server: {e}')

################################################################
# Araneae_worknode 主函数                                       #
################################################################
if __name__ == '__main__':
    print("HollowData Group - Araneae Work Node Server")
    print("[INFO] Araneae Work Node Server is running...")

    app.logger.info("Server starting...")

    flask_port = 5001
    # TODO: 检查 Flask 端口是否被占用
    # TODO: 检查控制节点是否在线
    '''
    TODO: 检查 Flask 端口是否被占用
    # 预留
    try:
        import socket
        test_socket = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        test_socket.bind(("0.0.0.0", flask_port))
        test_socket.close()
    except OSError:
        print(f"[ERROR] Port {flask_port} is already in use. Flask cannot start.")
        sys.exit(1)  # 终止程序，防止 receive.py 运行
    '''

    # 启动 receive.py
    try:
        print("[INFO] Starting receive.py...")
        receive_process = multiprocessing.Process(target=run_receive, daemon=True)
        receive_process.start()
    except Exception as e:
        print(f"[ERROR] receive.py failed to start: {e}")
        app.logger.error(f"receive.py failed to start: {e}")
        sys.exit(1)  # 终止程序

    # 创建数据库表（如果需要）
    """with app.app_context():
        db.drop_all()
        db.create_all()
        app.logger.info("Database tables recreated")

        print("[INFO] Create node identity UUID")
        import uuid
        uuid = uuid.uuid4()
        print(f"[INFO] Node UUID: {uuid}")
        # 创建控制节点记录
        identity = Identity(identity_hash=str(uuid))
        db.session.add(identity)
        db.session.commit()"""

    # 启动 Celery

    try:
        print("[INFO] Starting Celery...")
        if "RUN_MAIN" not in os.environ and multiprocessing.get_start_method() == "spawn":
            print("[INFO] 启动 Celery...")
            signal.signal(signal.SIGINT, signal_handler)
            signal.signal(signal.SIGTERM, signal_handler)
            start_celery_worker()
            start_celery_beat()

    except Exception as e:
        print(f"[ERROR] Celery failed to start: {e}")
        app.logger.error(f"Celery failed to start: {e}")
        sys.exit(1)  # 终止程序

    try:
        print("[INFO] Starting gRPC services...")
        start_grpc_services()
    except Exception as e:
        print(f"[ERROR] gRPC services failed to start: {e}")
        app.logger.error(f"gRPC services failed to start: {e}")
        sys.exit(1)

    try:
        print("[INFO] Starting Flask...")
        app.run(host="0.0.0.0", port=flask_port, debug=False)
        app.logger.info("Flask started successfully")


    except Exception as e:
        print(f"[ERROR] Flask failed to start: {e}")
    finally:
        # Flask 退出时终止 receive.py
        receive_process.terminate()
        receive_process.join()
        print("[INFO] receive.py has been terminated.")
        # 停止 Celery
        stop_celery()
        print("[INFO] Celery has been terminated.")
        # 停止 gRPC 服务
        stop_grpc_services()
        print("[INFO] gRPC services have been terminated.")

        print("[INFO] Server stopped.")
        app.logger.info("Server stopped")
