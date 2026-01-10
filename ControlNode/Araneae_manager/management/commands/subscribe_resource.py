# -*- coding: utf-8 -*-

#  Copyright (c)   2025.5  Henry Zhao. All rights reserved.
#  From BJ.

# araneae_ControlNode - subscribe_resource.py.py
# Created by zhr62 at 2025/5/3 - 19:33

# === 以下功能：gRPC Client 订阅执行端状态 ===
"""
中文：管理端 gRPC 客户端，从执行端拉取 ResourceUsage 并保存
英文：Management-side gRPC client that pulls ResourceUsage from exec side and persists
# TODO: CHECK AND MAYBE DELETE
"""

# === 以下功能：管理端 gRPC 客户端流式订阅并归档节点资源 ===
"""
中文：管理端 gRPC 客户端，从执行端实时拉取节点资源状态，并按间隔归档与更新当前状态。
英文：Management-side gRPC client that streams ResourceUsage from exec side, archives at intervals and updates current status.
"""

import grpc
from datetime import datetime, timedelta
from django.core.management.base import BaseCommand
from araneae_proto import resource_pb2, resource_pb2_grpc
from Araneae_manager.models import Node, NodeCurrentStatus, NodeMetricArchive

# === 以下功能：判断是否需要归档 ===
"""
中文：判断与上次归档时间相比是否已超过归档间隔。
英文：Determine if archive interval has passed since last archive.
"""
def should_archive(last_archive, current_time, interval):
    if last_archive is None:
        return True
    return (current_time - last_archive) >= interval


class Command(BaseCommand):  # pylint: disable=too-few-public-methods
    # === 以下功能：命令帮助信息 ===
    """
    中文：订阅指定 worknode 的资源状态，更新最新状态并按间隔归档。
    英文：Subscribe to a worknode's ResourceUsage, update current status and archive at interval.
    """
    help = "订阅节点资源状态，更新当前状态并按指定间隔归档"

    def add_arguments(self, parser):
        # === 以下功能：添加命令行参数 ===
        """
        中文：添加节点 ID 和归档间隔（秒）参数。
        英文：Add node-id and archive-interval (seconds) arguments.
        """
        parser.add_argument(
            '--node-id', '-n', type=int, required=True,
            help='要订阅的 Node 对象的 ID'
        )
        parser.add_argument(
            '--interval', '-i', type=int, default=60,
            help='归档间隔，单位秒，默认60秒'
        )

    def handle(self, *args, **options):
        node_id = options['node_id']
        interval_seconds = options['interval']
        archive_interval = timedelta(seconds=interval_seconds)

        # === 以下功能：获取 Node 实例 ===
        """
        中文：根据 node_id 获取 Node 对象。
        英文：Retrieve Node object by node_id.
        """
        try:
            node = Node.objects.get(id=node_id)
        except Node.DoesNotExist:
            self.stderr.write(f"Node (ID={node_id}) 不存在")
            return

        # === 以下功能：建立 gRPC 通道与 stub ===
        """
        中文：连接执行端 gRPC 服务并创建 stub。
        英文：Connect to exec-side gRPC service and create stub.
        """
        grpc_target = f"{node.ip_address}:50051"
        channel = grpc.insecure_channel(grpc_target)
        stub = resource_pb2_grpc.ResourceMonitorStub(channel)

        last_archive_time = None

        # === 以下功能：订阅并处理流式 ResourceUsage ===
        """
        中文：循环订阅资源状态，更新当前状态并按间隔归档。
        英文：Loop subscribe ResourceUsage, update current status and archive at interval.
        """
        try:
            for usage in stub.Subscribe(resource_pb2.SubscribeRequest(node_id=str(node.id))):
                current_time = datetime.fromtimestamp(usage.timestamp / 1000.0)
                # 更新当前状态
                NodeCurrentStatus.objects.update_or_create(
                    node=node,
                    defaults={
                        'cpu_percent': usage.cpu_percent,
                        'memory_used': usage.memory_used,
                        'memory_total': usage.memory_total,
                        'gpu_info': [
                            {'index': g.index, 'mem_used': g.gpu_mem_used, 'util': g.gpu_util}
                            for g in usage.gpus
                        ],
                        'updated_at': current_time
                    }
                )
                # 归档历史数据
                if should_archive(last_archive_time, current_time, archive_interval):
                    NodeMetricArchive.objects.create(
                        node=node,
                        timestamp=current_time,
                        cpu_percent=usage.cpu_percent,
                        memory_used=usage.memory_used,
                        memory_total=usage.memory_total,
                        gpu_info=[
                            {'index': g.index, 'mem_used': g.gpu_mem_used, 'util': g.gpu_util}
                            for g in usage.gpus
                        ]
                    )
                    last_archive_time = current_time

                # 输出日志
                self.stdout.write(
                    f"[{current_time:%Y-%m-%d %H:%M:%S}] "
                    f"Node {node.id} CPU {usage.cpu_percent}% "
                    f"Mem {usage.memory_used}/{usage.memory_total}"
                )
        except grpc.RpcError as e:
            self.stderr.write(f"gRPC 错误: {e}")
        except Exception as e:
            self.stderr.write(f"处理异常: {e}")
