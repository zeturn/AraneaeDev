# -*- coding: utf-8 -*-
from Araneae_manager.models import Identity


#  Copyright (c)   2025.5  Henry Zhao. All rights reserved.
#  From BJ.

# araneae_ControlNode - identity.py
# Created by zhr62 at 2025/5/17 - 14:00

def get_hash():
    """
    获取当前节点的身份哈希值
    :return: 身份哈希值
    """
    # 使用 UUID 生成一个唯一的身份哈希值
    identity_hash = Identity.objects.first()
    return identity_hash