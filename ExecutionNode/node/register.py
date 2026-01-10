# -*- coding: utf-8 -*-
# Araneae_ExecutionNode - register.py
# Created by zhr62 at 2025/5/17 - 15:04

"""
Araneae_WorkNode - register.py
创建control_node的注册函数
"""
from sqlalchemy.sql.functions import now

from models import ControlNode, db


def register_control_node(name, ip_address=None, port=5000, rpc_url=None, auth_key=None, hdid=None, description=None, celery_queue=None):
    """
    注册一个新的 Node，如果 name 已存在则返回错误信息。

    :param name: Node 名称（唯一）
    :param ip_address: 节点 IP 地址（可选）
    :param port: 端口号，默认 5000
    :param rpc_url: RPC 访问地址（可选）
    :param auth_key: 认证密钥（可选）
    :param hdid: 硬件 ID（可选）
    :param description: 描述信息（可选）
    :param celery_queue: Celery 队列名称（可选）
    :return: 新创建的 Node 实例或错误信息
    """

    node = ControlNode(
        name=name,
        ip_address=ip_address,
        port=port,
        rpc_url=rpc_url,
        auth_key=auth_key,
        HDID=hdid,
        description=description,
        celery_queue=celery_queue,
        last_active_time=now(),
        status='inactive'
    )

    # 检查是否存在同名节点
    existing_node = ControlNode.query.filter_by(name=name).first()
    if existing_node:
        return {"error": "Node with this name already exists"}

    # 如果不存在同名节点，则添加新节点
    db.session.add(node)
    db.session.commit()
    # 更新节点的状态为 'active'
    node.status = 'active'
    node.last_active_time = now()
    db.session.commit()

    # 返回新创建的节点
    return node