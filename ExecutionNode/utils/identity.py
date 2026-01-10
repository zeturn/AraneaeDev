"""
Araneae_ExecutionNode - identity.py
"""
# -*- coding: utf-8 -*-
# Araneae_ExecutionNode - identity.py
# Created by zhr62 at 2025/5/17 - 16:25

from models import Identity
from config import create_app

app = create_app()

def get_hash():
    """
    获取当前节点的身份哈希值
    TODO: 可用性
    flask_sqlalchemy
    :return: 身份哈希值
    """
    try:
        with app.app_context():
            identity_hash = Identity.query.first()
            print("Identity Hash:", identity_hash)  # 调试输出，查看 identity_hash 的内容
            if identity_hash and hasattr(identity_hash, "identity_hash"):
                return identity_hash.identity_hash
            else:
                return None  # 如果没有找到身份哈希值，返回 None
    except Exception as e:
        # 可以根据需要记录日志
        return None


def get_hash_by_id(identity_id):
    """
    根据身份 ID 获取身份哈希值
    :param identity_id: 身份 ID
    :return: 身份哈希值
    """
    identity_hash = Identity.query.filter_by(id=identity_id).first()
    if identity_hash:
        return identity_hash.hash
    else:
        return None  # 如果没有找到身份哈希值，返回 None

if __name__ == "__main__":
    # 测试获取当前节点的身份哈希值
    print(get_hash())
    # 测试根据身份 ID 获取身份哈希值
    print(get_hash_by_id(1))  # 假设 ID 为 1 的身份存在