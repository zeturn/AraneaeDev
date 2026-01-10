# -*- coding: utf-8 -*-
"""
# Araneae_manager/tasks.py
"""
#  Copyright (c)  2025 INIT  2025.4 UPDATE  Henry Zhao. All rights reserved.
#  From CA.

import json
from datetime import timedelta

import grpc
import pika
import requests
from celery import shared_task, Celery
from django.utils import timezone

from Araneae import settings
from Araneae_manager.models import ChainedTask, Node, NodeCurrentStatus, NodeMetricArchive
from araneae_proto import resource_pb2_grpc, resource_pb2

################################################################
# Araneae_worknode new                                         #
################################################################

celery = Celery("tasks",
                broker=f"amqp://{settings.RABBITMQ['USERNAME']}:{settings.RABBITMQ['PASSWORD']}@{settings.RABBITMQ['HOST']}:{settings.RABBITMQ['PORT']}/{settings.RABBITMQ['VHOST']}", )


def dispatch_to_flask(*args, **kwargs):
    """
    发送任务给 Flask 端（Flask 的 Celery worker 会监听这个任务名），并同步等待 Flask 返回结果
    """
    task = celery.send_task( # 指定路由键和队列，确保任务路由到公共频道
        "execute_script",
        args=args,
        kwargs=kwargs,
        routing_key="public.task",
        queue="public_channel"
    )
    print(f"👋任务已派发:", task)
    return task.id



@shared_task(name="schedule_task_execution")
def schedule_task_execution(*args, **kwargs):
    """
    Celery Beat 调用的定时任务入口
    """

    print("📦 [schedule_task_execution] Triggered with args:", args)

    try:
        print("Executing script...")
        args = [{
            "script_path": r"C:\Users\zhr62\PycharmProjects\djangoProject\Araneae_worknode\repo\5\Wn5hV0\helloworld.py",
            "task_id": "",
        }]

        task_result = dispatch_to_flask( *args, **kwargs) # 发送任务给 Flask
        print(f"✅ 已派发给 Flask 端 Celery，开始执行，返回结果: {task_result}")

    except Exception as e:
        print(f"❌ 执行失败: {e}")

    return f"Task dispatched and checked."


@shared_task(name="print_task")
def print_task(*args, **kwargs):
    """
    测试任务
    """
    print("📦 [print_task] Triggered with args:", args)
    print("📦 [print_task] Triggered with kwargs:", kwargs)
    return "Task executed successfully."

@shared_task(name="call_hello_world")
def call_hello_world():
    """
    测试任务
    """
    print("📦 [call_hello_world] Triggered")
    return "Hello, world!"

@shared_task(name='poll_all_nodes_status')
def poll_all_nodes_status(*args, **kwargs):
    """
    中文：轮询所有启用节点的资源状态
    English: Poll all enabled nodes for resource status
    """
    print("📦 [poll_all_nodes_status] Triggered")
    nodes = Node.objects.filter(is_enabled=True)
    for node in nodes:
        try:
            # 建立 gRPC 通道并获取一个状态快照
            # Establish gRPC channel and fetch one snapshot
            channel = grpc.insecure_channel(f"{node.ip_address}:50051")
            stub = resource_pb2_grpc.ResourceMonitorStub(channel)
            call = stub.Subscribe(
                resource_pb2.SubscribeRequest(node_id=str(node.id)),
                timeout=5
            )
            usage = next(call)
            call.cancel()  # 取消流以释放资源

            # 更新或创建“当前状态”记录
            # Update or create the NodeCurrentStatus record
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

            # 每分钟归档一次历史数据
            # Archive one data point per minute
            now = timezone.now()
            last = NodeMetricArchive.objects.filter(node=node).order_by('-timestamp').first()
            if not last or now - last.timestamp >= timedelta(seconds=60):
                NodeMetricArchive.objects.create(
                    node=node,
                    cpu_percent=usage.cpu_percent,
                    memory_used=usage.memory_used,
                    memory_total=usage.memory_total,
                    gpu_info=json.dumps([
                        {
                            'index': g.index,
                            'mem_used': g.gpu_mem_used,
                            'util': g.gpu_util
                        } for g in usage.gpus
                    ]),
                    timestamp=now
                )

        except Exception as e:
            print(f"❌ 轮询节点 {node.ip_address} 失败: {e}")
            continue

################################################################
# Araneae_main old ver                                         #
################################################################
@shared_task
def publish_to_rabbitmq(task_name, payload):
    """
    将任务发布到 RabbitMQ 队列。[abandoned]
    :param task_name: 任务名称
    :param payload: 任务数据
    """
    import pika
    try:
        # RabbitMQ 连接配置
        connection = pika.BlockingConnection(pika.ConnectionParameters(
            host='199.7.140.120',
            port=15673,
            credentials=pika.PlainCredentials('guest', '54321Ssdlh!!')
        ))
        channel = connection.channel()

        # 声明队列
        channel.queue_declare(queue='task_queue', durable=True)

        # 发布消息
        message = {
            "task_name": task_name,
            "payload": payload
        }
        channel.basic_publish(
            exchange='',
            routing_key='task_queue',
            body=json.dumps(message),
            properties=pika.BasicProperties(delivery_mode=2)  # 消息持久化
        )
        connection.close()
        return f"Task {task_name} published to RabbitMQ"
    except Exception as e:
        return f"Failed to publish task {task_name}: {str(e)}"


@shared_task
def send_crawl_message(project_id, task_details):
    """
    Send a crawl message to RabbitMQ for the specified project.[abandoned]
    """
    connection = pika.BlockingConnection(pika.ConnectionParameters(
        host='199.7.140.120',
        port=15673,
        credentials=pika.PlainCredentials('guest', '54321Ssdlh!!')
    ))
    channel = connection.channel()
    channel.queue_declare(queue='crawl_tasks')

    # Message to send
    message = {
        'project_id': project_id,
        'task_details': task_details
    }

    channel.basic_publish(exchange='',
                          routing_key='crawl_tasks',
                          body=str(message))
    connection.close()


@shared_task(bind=True)
def start_crawler(project_path, script_name):
    """
    Start a crawler process.[abandoned]
    :param project_path:
    :param script_name:
    :return:
    """
    response = requests.post(
        'http://localhost:5000/run_task',
        json={'project_path': project_path, 'script_name': script_name}
    )
    return response.json()

if __name__ == '__main__':
    recipe = {
        "script_path": r"C:\Users\zhr62\PycharmProjects\djangoProject\Araneae_worknode\repo\5\6RzZR6\helloworld.py",
        "args": "args",
    }
    dispatch_to_flask(recipe)