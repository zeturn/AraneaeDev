# -*- coding: utf-8 -*-
# Araneae_ExecutionNode - grpc_server.py.py
# Created by zhr62 at 2025/5/3 - 19:06
# === 以下功能：gRPC Server for resource monitoring ===

"""
中文：执行端 gRPC 服务，持续采集并推送节点资源状态
英文：Execution-side gRPC server that collects and streams node resource usage
"""

import time
import threading
from concurrent import futures

import psutil
import grpc

from araneae_proto import resource_pb2, resource_pb2_grpc, node_registration_pb2_grpc, node_registration_pb2
from utils.identity import get_hash
from models import ControlNode  # 假定已在 models.py 定义 ControlNode

_ONE_DAY_IN_SECONDS = 60 * 60 * 24

class ResourceMonitorServicer(resource_pb2_grpc.ResourceMonitorServicer):
    # === 以下功能：处理 Subscribe 请求 ===
    """
    中文：收到订阅请求后，循环采集并推送 ResourceUsage
    英文：On Subscribe, continuously collect and send ResourceUsage
    """
    def Subscribe(self, request, context):
        # Fix: use correct proto field name (node_id, not node_hash)
        node_id = request.node_id
        while True:
            # 内存
            vm = psutil.virtual_memory()
            # GPU — 延迟导入，避免在无 GPU 的 Docker 容器中顶层 import 崩溃
            gpus = []
            try:
                import gpustat
                for gpu in gpustat.new_query().gpus:
                    gpus.append(resource_pb2.GpuUsage(
                        index=gpu.index,
                        gpu_mem_used=gpu.memory_used * 1024 * 1024,
                        gpu_mem_total=gpu.memory_total * 1024 * 1024,
                        gpu_util=gpu.utilization
                    ))
            except Exception as e:
                print(f"[INFO] GPU stats not available: {e}")  # 无 GPU 或驱动异常，正常跳过

            usage = resource_pb2.ResourceUsage(
                node_id=node_id,
                cpu_percent=psutil.cpu_percent(),
                memory_used=vm.used,
                memory_total=vm.total,
                gpus=gpus,
                timestamp=int(time.time() * 1000)
            )
            yield usage
            time.sleep(5)  # 每 5 秒推送一次

class NodeRegistrationServicer(node_registration_pb2_grpc.NodeRegistrationServicer):
    # === 以下功能：第一步握手，记录控制节点 hash ===
    """
    中文：接收控制节点的 hash 和问候，存库并返回自身节点 hash
    English: Receive control node hash + greeting, persist and return this node’s hash
    """

    def get_control_hash(self):
        """
        控制节点 hash 读取逻辑
        :return:
        """
        node_hash = get_hash()
        if isinstance(node_hash, (bytes, bytearray)):
            node_hash = node_hash.decode('utf-8')
        else:
            node_hash= str(node_hash)
        print(f"Control node hash: {node_hash}")
        return node_hash

    def Handshake(self, request, context):
        # 记录控制节点 hash noid_uuid=request.control_hash flask
        from app import app, db
        print("Received handshake request from control node.")
        try:
            with app.app_context():
                control_node = ControlNode(
                    node_hash=request.control_hash,
                    status = 'active',
                    is_main = False,
                    is_enabled = True,
                )
                # 保存到数据库
                db.session.add(control_node)
                db.session.commit()
                print("Control node saved successfully.")
        except Exception as e:
            with app.app_context():
                db.session.rollback()
            print(f"Error saving control node: {e}")
            return node_registration_pb2.HandshakeResponse(
                node_hash=get_hash(),
                greeting='👋'
            )

        # 获取本节点 hash
        from app import app
        with app.app_context():
            node_hash = self.get_control_hash()
        print(f"Node hash: {node_hash}")
        return node_registration_pb2.HandshakeResponse(
            node_hash=node_hash,
            greeting='👋'
        )

    # === 以下功能：第二步确认，更新工作节点 hash 确认 ===
    """
    中文：接收客户端确认的工作节点 hash，更新 ControlNode 记录，并回应问候
    English: Receive client-confirmed node hash, update ControlNode record, and reply greeting
    """
    def Confirm(self, request, context):
        return node_registration_pb2.ConfirmResponse(greeting='👋')



def serve():
    """
    注册 gRPC 服务
    """
    server = grpc.server(futures.ThreadPoolExecutor(max_workers=4))
    # 注册两个服务
    from araneae_proto import resource_pb2_grpc
    resource_pb2_grpc.add_ResourceMonitorServicer_to_server(
        ResourceMonitorServicer(), server
    )
    node_registration_pb2_grpc.add_NodeRegistrationServicer_to_server(
        NodeRegistrationServicer(), server
    )
    server.add_insecure_port('0.0.0.0:50051')
    server.start()
    try:
        while True:
            time.sleep(_ONE_DAY_IN_SECONDS)
    except KeyboardInterrupt:
        server.stop(0)

if __name__ == '__main__':
    # 若要与 Flask 并行运行，可用 Thread
    threading.Thread(target=serve, daemon=True).start()

