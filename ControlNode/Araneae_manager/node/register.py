# -*- coding: utf-8 -*-
"""
#  Araneae_WorkNode - register.py
"""
#  Copyright (c)   2025.3  Henry Zhao. All rights reserved.
#  From CA.

# araneae_ControlNode - register.py.py
# Created by zhr62 at 2025/3/21 - 23:54

import time
from django.utils.timezone import now
from Araneae_manager.models import Node
from Araneae_manager.serializers import NodeSerializer


def register_node(name, node_hash = None, ip_address=None, port=5000, rpc_url=None, auth_key=None, hdid=None, description=None, celery_queue=None, status='inactive'):
    """
    注册一个新的 Node，如果 name 已存在则返回错误信息。

    :param name: Node 名称（唯一）
    :param node_hash: 节点唯一标识（可选）
    :param ip_address: 节点 IP 地址（可选）
    :param port: 端口号，默认 5000
    :param rpc_url: RPC 访问地址（可选）
    :param auth_key: 认证密钥（可选）
    :param hdid: 硬件 ID（可选）
    :param description: 描述信息（可选）
    :param celery_queue: Celery 队列名称（可选）
    :return: 新创建的 Node 实例或错误信息
    """
    if Node.objects.filter(name=name).exists():
        return {"error": "Node with this name already exists"}

    if Node.objects.filter(node_hash=node_hash).exists():
        return {"error": "Node with this hash already exists"}

    node = Node.objects.create(
        name=name,
        node_hash=node_hash,
        ip_address=ip_address,
        port=port,
        rpc_url=rpc_url,
        auth_key=auth_key,
        HDID=hdid,
        description=description,
        celery_queue=celery_queue,
        last_active_time=now(),
        status=status
    )
    serializer = NodeSerializer(node)
    return serializer.data

