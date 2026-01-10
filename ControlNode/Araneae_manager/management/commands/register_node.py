# -*- coding: utf-8 -*-

#  Copyright (c)   2025.5  Henry Zhao. All rights reserved.
#  From BJ.

# araneae_ControlNode - register_node.py.py
# Created by zhr62 at 2025/5/17 - 16:40

# === 以下功能：管理命令，通过 gRPC 向工作节点注册 ===
"""
中文：Django 管理命令，手动向指定工作节点 IP 发起注册握手
English: Django management command to manually initiate registration handshake with a work node
"""
# TODO: CHECK AND MAYBE DELETE

import grpc
from django.core.management.base import BaseCommand
from araneae_proto import node_registration_pb2, node_registration_pb2_grpc
from Araneae_manager.utils.identity import get_hash

def get_control_hash():
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

class Command(BaseCommand):
    help = "通过 gRPC 向工作节点注册（输入 --ip 参数）"

    def add_arguments(self, parser):
        """
        中文：添加命令行参数
        英文：Add command line arguments
        :param parser:
        """
        parser.add_argument(
            '--ip', '-i',
            required=True,
            help='工作节点的 IP 地址，例如 192.168.1.100'
        )

    def handle(self, *args, **options):
        """

        :param args:
        :param options:
        """
        ip = options['ip']
        channel = grpc.insecure_channel(f"{ip}:50051")
        stub = node_registration_pb2_grpc.NodeRegistrationStub(channel)

        # === 第一步：握手 ===
        self.stdout.write("🔗 正在向节点发起 Handshake …")
        resp = stub.Handshake(
            node_registration_pb2.HandshakeRequest(
                control_hash=get_control_hash(),
                greeting='👋'
            ),
            timeout=5
        )
        self.stdout.write(f"✅ 收到节点回应：node_hash={resp.node_hash}, greeting={resp.greeting}")

        # === 第二步：确认 ===
        self.stdout.write("🔐 正在发送 Confirm …")
        ack = stub.Confirm(
            node_registration_pb2.ConfirmRequest(
                node_hash=resp.node_hash,
                greeting='👋'
            ),
            timeout=5
        )
        self.stdout.write(f"🎉 注册完成，收到确认：{ack.greeting}")
