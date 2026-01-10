from __future__ import absolute_import, unicode_literals
import os
from celery import Celery
from kombu import Queue

# 设置默认的 Django settings 模块
os.environ.setdefault('DJANGO_SETTINGS_MODULE', 'Araneae.settings')

app = Celery('Araneae')

#broker_connection_retry_on_startup = True
# 从 Django 的 settings 中加载 Celery 配置
app.config_from_object('django.conf:settings', namespace='CELERY')
app.conf.task_default_queue = 'celery'
# 定义队列（频道）
app.conf.task_queues = (
    # 公共频道：供 Flask worker 使用
    Queue('public_channel', routing_key='public.#'),
    # 内部频道：供 Django 内部任务使用
    Queue('internal_channel', routing_key='internal.#'),
)

# 定义任务路由规则
app.conf.task_routes = {
    # 内部任务实例，发送到 internal_channel
    'Araneae_manager.tasks.internal_task': {
        'queue': 'internal_channel',
        'routing_key': 'internal.task'
    },
    # 内部任务，发送到 internal_channel，接受
    'schedule_task_execution': {
        'queue': 'internal_channel',
        'routing_key': 'schedule_task_execution'
    },
    # 内部任务，发送到 internal_channel
    'poll_all_nodes_status': {
        'queue': 'internal_channel',
        'routing_key': 'poll_all_nodes_status'
    },
    # 公共任务示例，发送到 public_channel
    'Araneae_manager.tasks.public_task': {
        'queue': 'public_channel',
        'routing_key': 'public.task'
    },
    # 公共任务，发送到 public_channel，接受
    'execute_script': {
        'queue': 'public_channel',
        'routing_key': 'execute_script'
    },
}

# 自动发现所有 Django apps 中的任务
app.autodiscover_tasks()


@app.task(bind=True)
def debug_task(self):
    """
    打印 request 对象
    @param self:
    """
    print(f'测试任务')
