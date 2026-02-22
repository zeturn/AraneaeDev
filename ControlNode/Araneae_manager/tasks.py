# -*- coding: utf-8 -*-
"""
# Araneae_manager/tasks.py
"""
#  Copyright (c)  2025 INIT  2025.4 UPDATE  Henry Zhao. All rights reserved.
#  From CA.

import json
from datetime import timedelta

import grpc
from celery import shared_task, Celery
from django.utils import timezone

from Araneae import settings
from Araneae_manager.models import ChainedTask, Node, NodeCurrentStatus, NodeMetricArchive
from araneae_proto import resource_pb2_grpc, resource_pb2

################################################################
# Celery 实例（用于向 ExecutionNode 投递任务）                 #
################################################################

celery = Celery(
    "tasks",
    broker=f"amqp://{settings.RABBITMQ['USERNAME']}:{settings.RABBITMQ['PASSWORD']}"
           f"@{settings.RABBITMQ['HOST']}:{settings.RABBITMQ['PORT']}/{settings.RABBITMQ['VHOST']}",
)


################################################################
# 核心：精准路由分发                                           #
################################################################

def dispatch_task_to_nodes(task_kwargs: dict, node_list: list) -> list:
    """
    向 node_list 中每个节点的专属队列精准投递 execute_script 任务。

    - 每个 ExecutionNode 监听自己的独立队列 node_{node_hash}
    - ControlNode 逐一投递，不再广播
    - 多节点并行：node_list 中每个节点均收到独立消息

    :param task_kwargs: 传递给 execute_script 的完整 kwargs（无需包含 nodes 字段）
    :param node_list:   目标节点 hash 列表
    :return:            返回各节点的 Celery task_id 列表
    """
    task_ids = []
    for node_hash in node_list:
        queue_name = f"node_{node_hash}"
        task = celery.send_task(
            "execute_script",
            kwargs=task_kwargs,
            queue=queue_name,
            routing_key=f"node.{node_hash}",
        )
        print(f"✅ 已派发任务到节点 [{node_hash}]，queue={queue_name}，celery_id={task.id}")
        task_ids.append(task.id)
    return task_ids


################################################################
# 定时入口任务（Celery Beat 驱动）                             #
################################################################

@shared_task(name="schedule_task_execution")
def schedule_task_execution(*args, **kwargs):
    """
    Celery Beat 调用的定时任务入口。
    kwargs 需由 PeriodicTask 配置注入，包含：
      - project_hash  (str)
      - version_hash  (str)  可为 "LATEST"
      - nodes         (list) 目标节点 hash 列表
      - task_id       (int|str|None)
      - chain_id      (int|None)
    """
    print("📦 [schedule_task_execution] Triggered with kwargs:", kwargs)

    node_list = kwargs.get("nodes", [])
    if not node_list:
        print("⚠️ [schedule_task_execution] nodes 为空，跳过派发")
        return "No nodes specified."

    try:
        task_ids = dispatch_task_to_nodes(task_kwargs=kwargs, node_list=node_list)
        print(f"✅ 已成功派发到 {len(task_ids)} 个节点: {task_ids}")
    except Exception as e:
        print(f"❌ 派发失败: {e}")
        raise

    return f"Dispatched to {len(task_ids)} node(s)."


################################################################
# 节点资源轮询（ControlNode 内部定时任务）                     #
################################################################

@shared_task(name='poll_all_nodes_status')
def poll_all_nodes_status(*args, **kwargs):
    """
    轮询所有启用节点的资源状态（通过 gRPC）。
    """
    print("📦 [poll_all_nodes_status] Triggered")
    nodes = Node.objects.filter(is_enabled=True)
    for node in nodes:
        try:
            channel = grpc.insecure_channel(f"{node.ip_address}:50051")
            stub = resource_pb2_grpc.ResourceMonitorStub(channel)
            call = stub.Subscribe(
                resource_pb2.SubscribeRequest(node_id=str(node.id)),
                timeout=5
            )
            usage = next(call)
            call.cancel()

            NodeCurrentStatus.objects.update_or_create(
                node=node,
                defaults={
                    'cpu_percent': usage.cpu_percent,
                    'memory_used': usage.memory_used,
                    'memory_total': usage.memory_total,
                    'gpu_info': [
                        {
                            'index': g.index,
                            'mem_used': g.gpu_mem_used,
                            'util': g.gpu_util
                        } for g in usage.gpus
                    ],
                    'updated_at': timezone.now()
                }
            )

            now = timezone.now()
            last = NodeMetricArchive.objects.filter(node=node).order_by('-timestamp').first()
            if not last or now - last.timestamp >= timedelta(seconds=60):
                NodeMetricArchive.objects.create(
                    node=node,
                    cpu_percent=usage.cpu_percent,
                    memory_used=usage.memory_used,
                    memory_total=usage.memory_total,
                    gpu_info=[  # JSONField 自动序列化，无需 json.dumps()
                        {
                            'index': g.index,
                            'mem_used': g.gpu_mem_used,
                            'util': g.gpu_util
                        } for g in usage.gpus
                    ],
                    timestamp=now
                )

        except Exception as e:
            print(f"❌ 轮询节点 {node.ip_address} 失败: {e}")
            continue


################################################################
# 测试任务                                                     #
################################################################

@shared_task(name="print_task")
def print_task(*args, **kwargs):
    """测试：打印收到的参数"""
    print("📦 [print_task] args:", args)
    print("📦 [print_task] kwargs:", kwargs)
    return "Task executed successfully."


@shared_task(name="call_hello_world")
def call_hello_world():
    """测试：Hello World"""
    print("📦 [call_hello_world] Triggered")
    return "Hello, world!"


################################################################
# 节点心跳检测（ControlNode 内部定时任务，每 60 秒执行）       #
################################################################

@shared_task(name='heartbeat_all_nodes')
def heartbeat_all_nodes(*args, **kwargs):
    """
    轮询所有启用节点的存活状态（通过 HTTP GET /health）。
    - 响应 200 → Node.status = 'active'，更新 last_active_time
    - 超时 / 连接失败 → Node.status = 'unreachable'
    每 60 秒由 Celery Beat 触发一次。
    """
    import requests
    from django.utils import timezone

    print("💓 [heartbeat_all_nodes] Triggered")
    nodes = Node.objects.filter(is_enabled=True)

    for node in nodes:
        url = f"http://{node.ip_address}:{node.port}/health"
        try:
            resp = requests.get(url, timeout=3)
            alive = resp.status_code == 200
        except requests.RequestException:
            alive = False

        new_status = 'active' if alive else 'unreachable'
        update_fields = ['status']

        if node.status != new_status:
            node.status = new_status

        if alive:
            node.last_active_time = timezone.now()
            update_fields.append('last_active_time')

        node.save(update_fields=update_fields)

        icon = "✅" if alive else "❌"
        print(f"  {icon} Node [{node.name}] {node.ip_address} → {new_status}")

    print(f"💓 [heartbeat_all_nodes] Done, checked {nodes.count()} node(s)")
