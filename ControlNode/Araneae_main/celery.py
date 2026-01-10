# -*- coding: utf-8 -*-
"""
#  araneae_main - celery.py
"""
#  Copyright (c)   2025.2  Henry Zhao. All rights reserved.
#  From BJ.

# araneae_main - celery.py.py
# Created by zhr62 at 2025/2/16 - 下午5:07
import os
from celery import Celery

# 设置 Django 的默认 settings 模块
os.environ.setdefault("DJANGO_SETTINGS_MODULE", "araneae_main.settings")

# 创建 Celery 应用实例
app = Celery("araneae_main")

# 从 Django settings 读取 Celery 配置
app.config_from_object("django.conf:settings", namespace="CELERY")

# 自动发现任务
app.autodiscover_tasks()
