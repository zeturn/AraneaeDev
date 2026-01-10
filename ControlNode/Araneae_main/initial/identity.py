# -*- coding: utf-8 -*-

#  Copyright (c)   2025.5  Henry Zhao. All rights reserved.
#  From BJ.

# araneae_ControlNode - identity.py
# Created by zhr62 at 2025/5/17 - 13:52

import os
import sys
import json
import uuid

from Araneae_manager.models import Identity


def get_identity_hash():
    """
    获取当前节点的身份哈希值
    :return: 身份哈希值
    """
    # 使用 UUID 生成一个唯一的身份哈希值
    identity_hash = str(uuid.uuid4())
    return identity_hash

def store_identity(identity_hash):
    """
    存储当前节点的身份哈希值到数据库
    :param identity_hash: 身份哈希值
    """
    identity = Identity.objects.create(identity_hash=identity_hash)
    identity.save()
    print(f"[INFO] Identity hash {identity_hash} created and stored in database.")

def init_identity():
    """
    初始化节点身份
    :return: None
    """
    identity_hash = get_identity_hash()
    store_identity(identity_hash)
    print(f"[INFO] Node identity initialized with hash: {identity_hash}")