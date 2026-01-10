# -*- coding: utf-8 -*-
# Araneae_WorkNode - setting.py.py
# Created by zhr62 at 2025/2/2 - 下午9:44

from pathlib import Path
import os
from config_loader import load_config

# Flask 项目根目录
FLASK_PROJECT_ROOT = os.path.dirname(os.path.abspath(__file__))  # 确保是你的 Flask 根目录
LOCAL_REPO = os.path.join(FLASK_PROJECT_ROOT, "repo")  # 本地存储目录

# rabbitmq 配置（从共享 config.json 读取）。请在仓库根创建 config.json，参考 config.example.json。
_cfg = load_config().get('rabbitmq', {})
RABBITMQ = {
    'HOST': _cfg.get('host', 'localhost'),             # RabbitMQ 服务地址 调用：RABBITMQ['HOST']
    'PORT': int(_cfg.get('port', 5672)),               # RabbitMQ 服务端口（默认 5672） 调用：RABBITMQ['PORT']
    'USERNAME': _cfg.get('username', 'guest'),         # RabbitMQ 用户名 调用：RABBITMQ['USERNAME']
    'PASSWORD': os.environ.get('RABBITMQ_PASSWORD', _cfg.get('password', '')),  # RabbitMQ 密码
    'VHOST': _cfg.get('vhost', '/'),                   # 虚拟主机（默认 /） 调用：RABBITMQ['VHOST']
}

