from __future__ import absolute_import, unicode_literals
import os
from celery import Celery
from kombu import Queue, Exchange

# 设置默认的 Django settings 模块
os.environ.setdefault('DJANGO_SETTINGS_MODULE', 'Araneae.settings')

app = Celery('Araneae')

# 从 Django 的 settings 中加载 Celery 配置
app.config_from_object('django.conf:settings', namespace='CELERY')
app.conf.task_default_queue = 'celery'

################################################################
# 队列定义                                                     #
################################################################
# node_direct exchange：用于精准路由到各 ExecutionNode 专属队列
node_exchange = Exchange('node_direct', type='direct')

app.conf.task_queues = (
    # ControlNode 内部任务队列（定时任务、轮询等）
    Queue('internal_channel', routing_key='internal.#'),
    # 兼容旧版广播队列（保留，不删）
    Queue('public_channel', routing_key='public.#'),
    # 注意：每个节点的专属队列 node_{hash} 由 ControlNode 在派发时动态声明，
    # 无需在此静态列举。
)

################################################################
# 任务路由规则                                                 #
################################################################
app.conf.task_routes = {
    # ControlNode 内部调度任务
    'schedule_task_execution': {
        'queue': 'internal_channel',
        'routing_key': 'internal.schedule',
    },
    # 节点资源轮询
    'poll_all_nodes_status': {
        'queue': 'internal_channel',
        'routing_key': 'internal.poll',
    },
    # 节点心跳检测（每 60 秒）
    'heartbeat_all_nodes': {
        'queue': 'internal_channel',
        'routing_key': 'internal.heartbeat',
    },
    # ExecutionNode 执行任务（精准路由）
    # queue 和 routing_key 由 dispatch_task_to_nodes() 在调用时动态指定，
    # 此处规则作为默认 fallback。
    'execute_script': {
        'queue': 'public_channel',
        'routing_key': 'public.task',
    },
}

# 自动发现所有 Django apps 中的任务
app.autodiscover_tasks()


@app.task(bind=True)
def debug_task(self):
    """打印 request 对象（测试任务）"""
    print(f'测试任务')
