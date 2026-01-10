"""
ASGI config for Araneae project.

It exposes the ASGI callable as a module-level variable named ``application``.

For more information on this file, see
https://docs.djangoproject.com/en/5.0/howto/deployment/asgi/
"""

import os

from django.core.asgi import get_asgi_application

os.environ.setdefault('DJANGO_SETTINGS_MODULE', 'Araneae.settings')

application = get_asgi_application()

'''
通道：

更新通道
+项目通道
-项目更新通道
-项目删除通道
-项目创建通道
-项目详情通道
-项目列表通道
-项目成员通道

+任务通道
-任务发布通道
-任务检查通道

+日志通道
-INFO通道
-ERROR通道
-DEBUG通道
-WARNING通道

+节点监控通道
-心跳通道


'''