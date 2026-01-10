# -*- coding: utf-8 -*-
from django.contrib.auth import get_user_model

from Araneae import settings


#  Copyright (c)   2025.4  Henry Zhao. All rights reserved.
#  From BJ.

# araneae_ControlNode - user.py
# Created by zhr62 at 2025/4/6 - 11:56

# django 创建初始admin用户


def create_initial_superuser(
        username=None,
        email=None,
        password=None,
        verbosity=1
):
    """
    如果不存在指定用户名的用户，就创建一个超级用户。
    参数默认从 settings 中读取，也可以调用时传入。
    """
    User = get_user_model()
    username = username or getattr(settings, 'INIT_ADMIN_USERNAME', 'admin')
    email = email or getattr(settings, 'INIT_ADMIN_EMAIL', 'admin@example.com')
    password = password or getattr(settings, 'INIT_ADMIN_PASSWORD', 'changeme123')

    if User.objects.filter(username=username).exists():
        if verbosity:
            print(f"⚠️ [USER] 用户 `{username}` 已存在，跳过创建。")
    else:
        User.objects.create_superuser(username=username, email=email, password=password)
        if verbosity:
            print(f"✅ [USER] 超级用户 `{username}` 创建成功。")