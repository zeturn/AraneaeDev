"""
# Araneae Execution Node
"""
# -*- coding: utf-8 -*-
# araneae_worknode - config.py.py
# Created by zhr62 at 2025/2/20 - 下午3:38
import os

from celery import Celery, Task
from flask import Flask
from kombu import Queue

import setting  # Ensure this module exists and contains the required RabbitMQ configuration
from models import db


def celery_init_app(app: Flask) -> Celery:
    """
    初始化 Celery
    @param app:
    @return:
    """
    class FlaskTask(Task):
        def __call__(self, *args: object, **kwargs: object) -> object:
            with app.app_context():
                return self.run(*args, **kwargs)

    celery_app = Celery(app.name, task_cls=FlaskTask)
    celery_app.config_from_object(app.config["CELERY"])
    celery_app.conf.task_queues = (
        Queue('public_channel', routing_key='public.#'),
        Queue('internal_channel', routing_key='internal.#'),
    )

    # 定义任务路由规则（这里只需要处理公共任务即可）
    celery_app.conf.task_routes = {
        'execute_script': {
            'queue': 'public_channel',
            'routing_key': 'execute_script'
        },
    }

    # celery_app.set_default()  # Ensure this method exists or remove it
    app.extensions["celery"] = celery_app
    return celery_app


def create_app() -> Flask:
    """
    创建 Flask App (for Celery)
    @return:
    """
    app = Flask(__name__)

    # 数据库配置
    basedir = os.path.abspath(os.path.dirname(__file__))
    app.config['SQLALCHEMY_DATABASE_URI'] = f'sqlite:///{os.path.join(basedir, "db.sqlite")}'
    app.config['SQLALCHEMY_TRACK_MODIFICATIONS'] = False
    app.config['SQLALCHEMY_ECHO'] = True

    # 初始化数据库
    db.init_app(app)

    app.config.from_mapping(
        CELERY=dict(
            broker_url=f"amqp://{setting.RABBITMQ['USERNAME']}:{setting.RABBITMQ['PASSWORD']}@{setting.RABBITMQ['HOST']}:{setting.RABBITMQ['PORT']}/{setting.RABBITMQ['VHOST']}",
            # 使用 RabbitMQ 作为消息中间件
            result_backend="rpc://",  # 使用 Redis 存储任务结果
            task_ignore_result=True,  # 默认情况下忽略任务结果
            broker_connection_retry_on_startup=True  # 在启动时重试连接
        ),
    )
    # app.config.from_prefixed_env()  # Ensure this method exists or remove it
    celery_init_app(app)
    return app