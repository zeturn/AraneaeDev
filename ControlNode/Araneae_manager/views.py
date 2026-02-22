"""
Araneae_worknode/views.py
"""
import traceback
import hashlib
import hmac
import time
import socket
import ipaddress
from concurrent.futures import ThreadPoolExecutor, as_completed
#  Copyright (c) 2024 INIT  2025.4 UPDATE Henry Zhao. All rights reserved.
#  From BJ.

from datetime import datetime
import os
import tempfile

import grpc
import requests
import json

from django.http import FileResponse, JsonResponse
from django.views.decorators.csrf import csrf_exempt
from django_celery_beat.models import PeriodicTask, CrontabSchedule
from rest_framework import status, viewsets
from rest_framework.decorators import action
from rest_framework.permissions import AllowAny
from rest_framework.viewsets import ModelViewSet

from Araneae import settings
from Araneae_manager.source.send import send_file, zip_folder
from Araneae_manager.tasks import dispatch_task_to_nodes
from Araneae_manager.task.record import create_task_record
from Araneae_manager.management.commands.register_node import get_control_hash
from Araneae_manager.models import Node, Schedule, TaskRecord, ChainedTask, TaskChain, Task, NodeCurrentStatus
from Araneae_manager.serializers import NodeSerializer, ScheduleSerializer, TaskRecordSerializer, TaskSerializer
from Araneae_manager.node.ping import ping, get_system_info
from Araneae_manager.node.register import register_node

from Araneae_repo.models import Project, Version, NodeProjectVersion
from Araneae_repo.serializers import NodeProjectVersionSerializer

from araneae_proto import node_registration_pb2, node_registration_pb2_grpc




################################################################
# Araneae_worknode Node                                        #
################################################################
class NodeViewSet(ModelViewSet):
    """工作节点相关视图集"""
    queryset = Node.objects.all()
    serializer_class = NodeSerializer

    @staticmethod
    def _discover_candidate_networks():
        """构造默认扫描网段（基于当前机器的 IPv4 地址）。"""
        networks = set()
        extra_hosts = set()
        ips = set()

        try:
            host_name = socket.gethostname()
            for ip in socket.gethostbyname_ex(host_name)[2]:
                ips.add(ip)
        except Exception:
            pass

        try:
            for info in socket.getaddrinfo(socket.gethostname(), None, socket.AF_INET):
                ip = info[4][0]
                if ip:
                    ips.add(ip)
        except Exception:
            pass

        # 兼容本机测试场景：仅探测 127.0.0.1，避免 127.0.0.0/24 全命中同一服务
        extra_hosts.add("127.0.0.1")

        for ip in ips:
            try:
                addr = ipaddress.ip_address(ip)
            except ValueError:
                continue

            if addr.version != 4:
                continue
            if not (addr.is_private or addr.is_loopback):
                continue

            if addr.is_loopback:
                continue

            networks.add(ipaddress.ip_network(f"{addr}/24", strict=False))

        return sorted(networks, key=lambda n: str(n)), sorted(extra_hosts)

    @staticmethod
    def _probe_worknode(ip, http_port, grpc_port, timeout=1.2):
        """探测单个地址是否为可注册的 worknode。"""
        sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        try:
            sock.settimeout(0.35)
            if sock.connect_ex((ip, http_port)) != 0:
                return None
        except Exception:
            return None
        finally:
            sock.close()

        headers = {}
        if getattr(settings, "NODE_API_TOKEN", ""):
            headers["X-Araneae-Node-Token"] = settings.NODE_API_TOKEN

        try:
            resp = requests.get(
                f"http://{ip}:{http_port}/system_info",
                headers=headers,
                timeout=timeout,
            )
        except requests.RequestException:
            return None

        if resp.status_code != 200:
            return None

        try:
            payload = resp.json()
        except ValueError:
            return None

        sys_info = payload.get("system_info", {}) or {}
        hostname = sys_info.get("hostname") or ip

        existing = Node.objects.filter(ip_address=ip).first()
        return {
            "ip": ip,
            "name": hostname,
            "port": http_port,
            "grpc_port": grpc_port,
            "already_registered": bool(existing),
            "registered_node_id": existing.id if existing else None,
            "machine": sys_info.get("machine"),
            "os": sys_info.get("system"),
            "internal_ips": sys_info.get("internal_ips", []) or [],
        }

    @staticmethod
    def fetch_system_info(ip_address, port):
        """向目标节点的 system_info API 发送请求并解析返回数据"""
        url = f"http://{ip_address}:{port}/system_info"
        try:
            response = requests.get(url, timeout=5)
            if response.status_code == 200:
                return response.json()
            else:
                return None
        except requests.RequestException as e:
            return {"error": f"Failed to fetch system info: {str(e)}"}

    @action(detail=False, methods=["post"], url_path="register")
    @csrf_exempt
    def register_new_node(self, request):
        """
        zh-CN: 注册新节点
        en-US: Register a new node
        :param request:
        :return:
        """
        print("Registering new node...")
        if request.method != "POST":
            return JsonResponse({"error": "Only POST requests are allowed"}, status=405)

        # 1) 解析输入
        try:
            data = request.data
            print("Parsed data:", data)
            ip = data['ip']
            name = data['name']
        except Exception as e:
            tb = traceback.format_exc()
            print("❌ Error parsing request data:\n", tb)
            return JsonResponse({"error": "Invalid input", "detail": str(e)}, status=400)

        # 2) Ping 检查
        try:
            ping_result = ping(ip)
            if not ping_result.get("reachable", False):
                return JsonResponse({"error": f"Ping failed: {ping_result.get('details')}"}, status=400)
        except Exception as e:
            tb = traceback.format_exc()
            print("❌ Error during ping:\n", tb)
            return JsonResponse({"error": "Ping error", "detail": str(e)}, status=500)

        # 3) 获取系统信息
        try:
            system_info = get_system_info(ip)
            if "error" in system_info:
                return JsonResponse(system_info, status=400)
        except Exception as e:
            tb = traceback.format_exc()
            print("❌ Error fetching system_info:\n", tb)
            return JsonResponse({"error": "System info error", "detail": str(e)}, status=500)

        # 4) gRPC 握手 + 确认
        try:
            channel = grpc.insecure_channel(f"{ip}:50051")
            grpc.channel_ready_future(channel).result(timeout=3)
            stub = node_registration_pb2_grpc.NodeRegistrationStub(channel)

            print("[INFO]🔗 Handshake …")
            ctrl = get_control_hash()
            print("DEBUG control_hash:", ctrl, "type:", type(ctrl))
            print("DEBUG greeting type:", type('👋'))

            resp = stub.Handshake(
                node_registration_pb2.HandshakeRequest(
                    control_hash=get_control_hash(),
                    greeting='👋'
                ), timeout=5
            )
            print(f"[INFO]✅ Handshake response: node_hash={resp.node_hash}, greeting={resp.greeting}")

            print("[INFO]🔐 Confirm …")
            ack = stub.Confirm(
                node_registration_pb2.ConfirmRequest(
                    node_hash=resp.node_hash,
                    greeting='👋'
                ), timeout=5
            )
            print(f"[INFO]🎉 Confirm response: {ack.greeting}")
        except grpc.RpcError as e:
            tb = traceback.format_exc()
            print("❌ gRPC error:\n", tb)
            return JsonResponse({"error": "gRPC failure", "code": e.code().name, "details": e.details()}, status=502)
        except Exception as e:
            tb = traceback.format_exc()
            print("❌ Unexpected error in gRPC handshake:\n", tb)
            return JsonResponse({"error": "Handshake error", "detail": str(e)}, status=500)

        # 5) 整理数据并调用 register_node
        node_data = {
            "name": name,
            "node_hash": resp.node_hash,
            "description": f"Registered from {system_info['Network']['Public IP']}",
            "ip_address": ip,
            "port": 5001,
            "rpc_url": f"http://{ip}:5001/rpc",
            "auth_key": None,
            "hdid": system_info["System"]["Machine"],
            "status": "active",
            "celery_queue": system_info["System"]["Hostname"],
        }
        print("[DEBUG] node_data:", node_data)
        print("[DEBUG] types:", {k: type(v).__name__ for k, v in node_data.items()})

        try:
            node = register_node(**node_data)
        except TypeError as e:
            tb = traceback.format_exc()
            print("❌ TypeError in register_node:\n", tb)
            return JsonResponse({"error": "TypeError", "detail": str(e)}, status=500)
        except Exception as e:
            tb = traceback.format_exc()
            print("❌ Unexpected error in register_node:\n", tb)
            return JsonResponse({"error": "Registration error", "detail": str(e)}, status=500)

        # 6) 返回成功
        return JsonResponse(node, status=201)

    @action(detail=False, methods=["get"], url_path="discover")
    def discover_worknodes(self, request):
        """
        扫描本地/内网可用 worknode。
        GET /api/nodes/discover/?scope=local|custom&cidr=192.168.1.0/24
        """
        scope = request.query_params.get("scope", "local")
        cidr_raw = (request.query_params.get("cidr") or "").strip()
        try:
            http_port = int(request.query_params.get("port", 5001))
            grpc_port = int(request.query_params.get("grpc_port", 50051))
        except ValueError:
            return JsonResponse({"error": "invalid port"}, status=400)

        networks = []
        if scope == "custom":
            if not cidr_raw:
                return JsonResponse({"error": "cidr is required when scope=custom"}, status=400)
            try:
                custom_net = ipaddress.ip_network(cidr_raw, strict=False)
            except ValueError:
                return JsonResponse({"error": f"invalid cidr: {cidr_raw}"}, status=400)

            if custom_net.num_addresses > 1024:
                return JsonResponse(
                    {"error": "cidr too large, please use <=1024 addresses"},
                    status=400
                )
            networks = [custom_net]
        else:
            networks, extra_hosts = self._discover_candidate_networks()

        candidates = []
        for net in networks:
            candidates.extend([str(ip) for ip in net.hosts()])
        if scope != "custom":
            candidates.extend(extra_hosts)

        # 默认模式下控制扫描规模，避免请求阻塞过久
        if scope != "custom":
            candidates = candidates[:768]

        discovered = []
        with ThreadPoolExecutor(max_workers=36) as executor:
            futures = [
                executor.submit(self._probe_worknode, ip, http_port, grpc_port)
                for ip in candidates
            ]
            for future in as_completed(futures):
                try:
                    result = future.result()
                except Exception:
                    continue
                if result:
                    discovered.append(result)

        # 去重：同一台主机可能通过多个地址被探测到，优先保留非 loopback 地址
        dedup_map = {}
        for item in discovered:
            key = (item.get("name"), item.get("machine"))
            old = dedup_map.get(key)
            if not old:
                dedup_map[key] = item
                continue
            old_loop = str(old.get("ip", "")).startswith("127.")
            new_loop = str(item.get("ip", "")).startswith("127.")
            if old_loop and not new_loop:
                dedup_map[key] = item

        discovered = list(dedup_map.values())
        discovered.sort(key=lambda item: (item["already_registered"], item["ip"]))
        return JsonResponse(
            {
                "scope": scope,
                "networks": [str(n) for n in networks],
                "scanned": len(candidates),
                "count": len(discovered),
                "candidates": discovered,
            },
            status=200
        )

    @action(detail=True, methods=['get'], url_path='status')
    def status(self, request, pk=None):
        """
        获取单个节点的状态
        GET /api/nodes/{pk}/status/
        """
        node = self.get_object()  # 会自动根据 pk 抛出 404
        try:
            stat = NodeCurrentStatus.objects.get(node=node)
        except NodeCurrentStatus.DoesNotExist:
            return JsonResponse({'detail': 'Status not available'}, status=404)

        data = {
            'cpu_percent': stat.cpu_percent,
            'memory_used': stat.memory_used,
            'memory_total': stat.memory_total,
        }
        return JsonResponse(data, status=200)

    @action(detail=True, methods=['get'], url_path='ping')
    def ping_node(self, request, pk=None):
        """
        实时探测单个执行节点是否存活（HTTP GET /health）。
        GET /api/nodes/{pk}/ping/
        Returns: {node_id, alive, status, latency_ms}
        """
        import time as _time
        node = self.get_object()
        url = f"http://{node.ip_address}:{node.port}/health"
        t0 = _time.monotonic()
        try:
            resp = requests.get(url, timeout=3)
            alive = resp.status_code == 200
        except requests.RequestException:
            alive = False
        latency_ms = round((_time.monotonic() - t0) * 1000, 1)

        new_status = 'active' if alive else 'unreachable'
        update_fields = ['status']
        node.status = new_status
        if alive:
            from django.utils import timezone
            node.last_active_time = timezone.now()
            update_fields.append('last_active_time')
        node.save(update_fields=update_fields)

        return JsonResponse({
            'node_id': node.id,
            'alive': alive,
            'status': new_status,
            'latency_ms': latency_ms if alive else None,
        }, status=200)

    @action(detail=True, methods=['get'], url_path='capabilities')
    def capabilities(self, request, pk=None):
        """
        读取节点已存储的运行时能力列表（不会重新探测执行节点）
        GET /api/nodes/{pk}/capabilities/
        """
        node = self.get_object()
        caps = node.runtime_capabilities or []
        return JsonResponse({'node_id': node.id, 'capabilities': caps}, status=200)

    @action(detail=True, methods=['post'], url_path='refresh_capabilities')
    def refresh_capabilities(self, request, pk=None):
        """
        主动拉取执行节点的运行时能力，更新数据库并返回结果
        POST /api/nodes/{pk}/refresh_capabilities/

        Flow:
          ControlNode → ExecutionNode GET /capabilities → store in Node.runtime_capabilities
        """
        node = self.get_object()
        url = f"http://{node.ip_address}:{node.port}/capabilities"

        headers = {}
        token = getattr(settings, 'NODE_API_TOKEN', '')
        if token:
            headers['X-Araneae-Node-Token'] = token

        try:
            resp = requests.get(url, headers=headers, timeout=10)
            resp.raise_for_status()
            capabilities_list = resp.json()
        except requests.Timeout:
            return JsonResponse(
                {'error': f'Request to node {node.ip_address}:{node.port} timed out'},
                status=504
            )
        except requests.ConnectionError:
            return JsonResponse(
                {'error': f'Cannot connect to node at {node.ip_address}:{node.port}'},
                status=502
            )
        except Exception as exc:
            return JsonResponse({'error': str(exc)}, status=500)

        # 持久化到 Node 模型
        node.runtime_capabilities = capabilities_list
        node.save(update_fields=['runtime_capabilities', 'updated_at'])

        return JsonResponse(
            {
                'node_id': node.id,
                'capabilities': capabilities_list,
                'count': len(capabilities_list),
                'available_count': sum(1 for c in capabilities_list if c.get('available')),
            },
            status=200
        )

    @action(detail=True, methods=['get'], url_path='installers')
    def installers(self, request, pk=None):
        """
        代理：获取执行节点支持安装的运行时列表
        GET /api/nodes/{pk}/installers/
        """
        node = self.get_object()
        url = f"http://{node.ip_address}:{node.port}/installers"
        headers = {}
        token = getattr(settings, 'NODE_API_TOKEN', '')
        if token:
            headers['X-Araneae-Node-Token'] = token
        try:
            resp = requests.get(url, headers=headers, timeout=10)
            resp.raise_for_status()
            return JsonResponse({'node_id': node.id, 'installers': resp.json()}, status=200)
        except requests.Timeout:
            return JsonResponse({'error': '节点请求超时'}, status=504)
        except requests.ConnectionError:
            return JsonResponse({'error': f'无法连接到节点 {node.ip_address}:{node.port}'}, status=502)
        except Exception as exc:
            return JsonResponse({'error': str(exc)}, status=500)

    @action(detail=True, methods=['post'], url_path='install_runtime')
    def install_runtime(self, request, pk=None):
        """
        代理：向执行节点发起后台安装任务
        POST /api/nodes/{pk}/install_runtime/   body: {"key": "node"}
        Returns: {"job_id": str}
        """
        node = self.get_object()
        key = request.data.get('key', '').strip()
        if not key:
            return JsonResponse({'error': "缺少 'key' 参数"}, status=400)

        url = f"http://{node.ip_address}:{node.port}/install"
        headers = {'Content-Type': 'application/json'}
        token = getattr(settings, 'NODE_API_TOKEN', '')
        if token:
            headers['X-Araneae-Node-Token'] = token
        try:
            resp = requests.post(url, json={'key': key}, headers=headers, timeout=15)
            resp.raise_for_status()
            return JsonResponse(resp.json(), status=resp.status_code)
        except requests.Timeout:
            return JsonResponse({'error': '节点请求超时'}, status=504)
        except requests.ConnectionError:
            return JsonResponse({'error': f'无法连接到节点 {node.ip_address}:{node.port}'}, status=502)
        except Exception as exc:
            return JsonResponse({'error': str(exc)}, status=500)

    @action(detail=True, methods=['get'], url_path=r'install_status/(?P<job_id>[^/.]+)')
    def install_status(self, request, pk=None, job_id=None):
        """
        代理：轮询执行节点的安装任务进度
        GET /api/nodes/{pk}/install_status/{job_id}/
        Returns: {job_id, key, name, status, log, exit_code, ...}
        """
        node = self.get_object()
        url = f"http://{node.ip_address}:{node.port}/install/{job_id}"
        headers = {}
        token = getattr(settings, 'NODE_API_TOKEN', '')
        if token:
            headers['X-Araneae-Node-Token'] = token
        try:
            resp = requests.get(url, headers=headers, timeout=10)
            resp.raise_for_status()
            return JsonResponse(resp.json(), status=200)
        except requests.Timeout:
            return JsonResponse({'error': '节点请求超时'}, status=504)
        except requests.ConnectionError:
            return JsonResponse({'error': f'无法连接到节点 {node.ip_address}:{node.port}'}, status=502)
        except Exception as exc:
            return JsonResponse({'error': str(exc)}, status=500)

################################################################
# Araneae_worknode SourceDistribute                            #
################################################################
class SourceDistributeViewSet(ModelViewSet):
    """分发文件视图集"""
    serializer_class = NodeSerializer
    @action(detail=False, methods=["post"], url_path="order")
    @csrf_exempt
    def distribute_source(self, request):
        """
        分发文件到各个节点
        api: /source_distribute/order
        :param request:
        """
        targets = request.data.get("targets", [])
        project_id = request.data.get("project_id")
        version = request.data.get("version")

        print(f"Distributing file to targets: {targets}")

        for target in targets:
            print(f"Distributing file to target: {target}")
            send_file(project_id, version, target['node_id'])

        #  创建version记录 NodeProjectVersion
        project_instance = Project.objects.get(id=project_id)
        version_instance = Version.objects.get(version_hash=version)
        NodeProjectVersion.objects.create(
            node_id=targets[0]['node_id'],
            project=project_instance,
            version=version_instance,
            deployed_at=datetime.now(),
            is_active=True
        )

        return JsonResponse({"message": "Distribution initiated"})

    @action(detail=False, methods=["get"], url_path="list")
    @csrf_exempt
    def source_distribution_list(self, request, *args, **kwargs):
        """
        api: /source_distribute/list
        获取所有分发记录
        """
        project_id = kwargs.get('project_id')
        queryset = NodeProjectVersion.objects.filter(project_id=project_id)
        serializer = NodeProjectVersionSerializer(queryset, many=True)
        return JsonResponse(serializer.data, safe=False)


################################################################
# Araneae_worknode TaskCallback                                #
################################################################
class TaskCallbackViewSet(ModelViewSet):
    """任务回调视图集"""
    queryset = TaskRecord.objects.all()
    serializer_class = TaskRecordSerializer
    permission_classes = [AllowAny]  # 允许任何人访问

    @staticmethod
    def _verify_callback_signature(request):
        secret = getattr(settings, "CALLBACK_SHARED_SECRET", "")
        if not secret:
            # Allow insecure callbacks only when explicitly enabled.
            return bool(getattr(settings, "ALLOW_INSECURE_CALLBACKS", False))

        timestamp = request.headers.get("X-Araneae-Timestamp", "")
        signature = request.headers.get("X-Araneae-Signature", "")
        if not timestamp or not signature:
            return False
        try:
            ts_int = int(timestamp)
        except ValueError:
            return False

        # 5-minute replay window
        if abs(int(time.time()) - ts_int) > 300:
            return False

        body = request.body.decode("utf-8")
        expected = hmac.new(
            secret.encode("utf-8"),
            f"{timestamp}.{body}".encode("utf-8"),
            hashlib.sha256,
        ).hexdigest()
        return hmac.compare_digest(expected, signature)

    @action(detail=False, methods=["post"], url_path="callback")
    @csrf_exempt
    def task_callback(self, request):
        """
        任务回调 — ExecutionNode 执行完毕后调用此接口汇报结果。
        ControlNode 负责更新 TaskRecord 并（如有任务链）触发下一步任务。
        api: POST /api/task/callback/
        """
        if not self._verify_callback_signature(request):
            return JsonResponse({"error": "Invalid callback signature"}, status=403)

        print("[SUCCESS][TASK] Received task callback")
        data = request.data
        print("[DEBUG] Callback payload:", data)

        node_hash    = data.get('node')
        project_hash = data.get('project')
        version_hash = data.get('version')
        task_status  = data.get('task_status', 'finished')
        task_result  = data.get('task_result', None)
        current_task_id  = data.get('task_id')
        task_chain_id    = data.get('task_chain_id')

        print(f'[DEBUG] node={node_hash} project={project_hash} version={version_hash} '
              f'status={task_status} chain={task_chain_id}')

        # --- 解析关联对象（容错：找不到则置 None）---
        try:
            node = Node.objects.get(node_hash=node_hash) if node_hash else None
        except Node.DoesNotExist:
            print(f"[WARNING] Node not found: {node_hash}")
            node = None

        try:
            project = Project.objects.get(project_hash=project_hash) if project_hash else None
        except Project.DoesNotExist:
            print(f"[WARNING] Project not found: {project_hash}")
            project = None

        try:
            version = Version.objects.get(version_hash=version_hash) if version_hash else None
        except Version.DoesNotExist:
            print(f"[WARNING] Version not found: {version_hash}")
            version = None

        # --- 写入执行记录 ---
        create_task_record(node, project, version, task_status, task_result)

        # --- 任务链推进（完全由 ControlNode 负责）---
        if task_chain_id and task_status in ('finished', 'completed'):
            try:
                task_chain = TaskChain.objects.get(id=task_chain_id)
                self.continue_task_chain(current_task_id, task_chain_id)
                print(f"🔗 任务链 [{task_chain.name}] 推进完毕")
            except TaskChain.DoesNotExist:
                print(f"[WARNING] TaskChain not found: id={task_chain_id}")
        elif task_chain_id and task_status in ('failed', 'error'):
            print(f"⚠️ 任务 {current_task_id} 执行失败，任务链 {task_chain_id} 停止推进")
        else:
            print("ℹ️ 无任务链，执行结束")

        return JsonResponse({"message": "Callback received.", "status": 200})


    def continue_task_chain(self, current_task_id, task_chain_id):
        """
        任务链推进（完全由 ControlNode 驱动）。

        查询当前任务在 ChainedTask 中的所有后续任务，
        逐一通过 dispatch_task_to_nodes 精准路由派发。
        """
        try:
            # 1) 获取当前 Task 实例
            try:
                current_task = Task.objects.get(id=current_task_id)
            except Task.DoesNotExist:
                print(f"⚠️ 未找到 Task: id={current_task_id}")
                return

            print(f"🔍 当前任务: id={current_task.id} name={current_task.name}")

            # 2) 在 ChainedTask 中查找「当前任务 → 下一任务」的关联记录
            #    ChainedTask.task = 当前任务，ChainedTask.next_task = 后续任务
            next_entries = ChainedTask.objects.filter(
                chain_id=task_chain_id,
                task=current_task
            ).select_related('next_task')

            print(f"🔗 找到 {next_entries.count()} 个后续任务")

            for entry in next_entries:
                next_task = entry.next_task
                if not next_task:
                    print("⚠️ ChainedTask.next_task 为空，跳过")
                    continue

                # 3) 从关联的 PeriodicTask 读取 kwargs（含 nodes、project_hash 等）
                try:
                    pt = next_task.periodic_task
                    if not pt:
                        print(f"⚠️ 任务 {next_task.id} 没有关联 PeriodicTask，跳过")
                        continue
                    task_kwargs = json.loads(pt.kwargs or "{}")
                except Exception as e:
                    print(f"⚠️ 读取后续任务 kwargs 失败: {e}，跳过")
                    continue

                node_list = task_kwargs.get("nodes", [])
                if not node_list:
                    print(f"⚠️ 后续任务 {next_task.name} nodes 为空，跳过")
                    continue

                print(f"➡️ 触发后续任务: [{next_task.name}] → 节点: {node_list}")
                try:
                    task_ids = dispatch_task_to_nodes(
                        task_kwargs=task_kwargs,
                        node_list=node_list,
                    )
                    print(f"✅ 后续任务已派发，celery task_ids={task_ids}")
                except Exception as e:
                    print(f"❌ 派发后续任务失败: {e}")

        except Exception as e:
            print(f"❌ continue_task_chain 执行异常: {e}")

        return "Chain check completed."


def download_file(request, file_name, project_id, version):
    """提供文件下载"""
    current_folder_path = os.path.join(settings.BASE_DIR, 'Araneae_repo', 'repo', str(project_id), version)
    compressed_file = os.path.join(tempfile.gettempdir(), f"{project_id}_{version}.zip")

    # 先压缩文件夹
    zip_folder(current_folder_path, compressed_file)

    response = FileResponse(open(compressed_file, 'rb'), as_attachment=True)

    # 设置回调，在请求结束后删除文件
    response['Content-Disposition'] = f'attachment; filename="{file_name}"'

    def cleanup_file(path):
        """
        清理临时文件
        :param path:
        :return: Response
        """
        if os.path.exists(path):
            os.remove(path)
            print(f"Deleted temporary file: {path}")

    response.close = lambda: cleanup_file(compressed_file)

    return response


def create_cron_task(project_id, cron_expression, task_details):
    """
    Create a periodic task based on the given cron expression.
    """
    # Parse cron expression
    minute, hour, day_of_month, month, day_of_week = cron_expression.split(' ')

    # Create or get Crontab schedule
    schedule, _ = CrontabSchedule.objects.get_or_create(
        minute=minute,
        hour=hour,
        day_of_month=day_of_month,
        month_of_year=month,
        day_of_week=day_of_week
    )

    # Create PeriodicTask
    task_name = f'run_project_{project_id}'
    PeriodicTask.objects.create(
        crontab=schedule,
        name=task_name,
        task='Araneae_manager.tasks.send_crawl_message',
        args=json.dumps([project_id, task_details])
    )


def delete_cron_task(task_name):
    """
    Delete a periodic task and its associated crontab schedule.
    """
    try:
        task = PeriodicTask.objects.get(name=task_name)
        schedule = task.crontab

        # Delete the task
        task.delete()

        # Delete schedule if no longer in use
        if not PeriodicTask.objects.filter(crontab=schedule).exists():
            schedule.delete()

        print(f"Task '{task_name}' and its schedule deleted successfully.")
    except PeriodicTask.DoesNotExist:
        print(f"Task '{task_name}' does not exist.")


################################################################
# Araneae_worknode TaskChain                                   #
################################################################
class TaskChainCreateView(viewsets.ModelViewSet):

    queryset = Schedule.objects.all()
    serializer_class = ScheduleSerializer
    def create(self, request):
        """
        创建任务链
        Status: 使用中
        :param request:
        :return: JsonResponse
        """
        serializer = self.get_serializer(data=request.data)
        serializer.is_valid(raise_exception=True)
        self.perform_create(serializer)
        instance = serializer.instance

        order = instance.order
        parsed_data = json.loads(order)  # 解析 JSON
        name = parsed_data["name"]
        schedules = parsed_data["schedule"]

        if not name or not schedules:
            return JsonResponse({"error": "缺少 name 或 schedule"}, status=status.HTTP_400_BAD_REQUEST )

        try:
            # 创建 TaskChain 实例
            chain = TaskChain.objects.create(name=name, enabled=True)

            task_map = {}  # 存储任务名到 PeriodicTask 的映射

            for idx, schedule in enumerate(schedules):

                # 获取任务配置参数
                # Get task configuration parameters
                task_id = schedule.get("task_id")
                task_status = schedule.get("task_status")
                new_task_name = schedule.get("name")
                project_id = schedule.get("project_id")
                node_list = schedule.get("node", [])
                trigger = schedule.get("trigger")
                crons = schedule.get("crons")
                previous_name = schedule.get("previous")

                # 处理参数
                args = json.dumps([project_id, node_list])

                if task_status == "exist":
                    # 使用现有 Task
                    # Use existing Task
                    try:
                        task = Task.objects.get(id=task_id)
                    except Task.DoesNotExist:
                        return JsonResponse(
                            {"error": f"未找到任务 ID '{task_id}'，请确保任务存在"},
                            status=status.HTTP_400_BAD_REQUEST
                        )
                    task_name = task.name
                    args = task.args
                    kwargs = task.kwargs
                else:
                    # 创建新 Task
                    # Create new Task
                    task_args = {"project_id": project_id, "nodes": node_list}
                    task = Task.objects.create(
                        name=new_task_name,
                        args=json.dumps(task_args),
                        kwargs=json.dumps({}),
                        enable=True,
                        celery_label="schedule_task_execution",
                    )
                    task_name = task.name
                    args = task.args
                    kwargs = task.kwargs

                # 创建定时任务
                if trigger == "crons":
                    cron_expr = schedule["crons"]
                    minute, hour, dom, month, dow = cron_expr.split()

                    crontab, _ = CrontabSchedule.objects.get_or_create(
                        minute=minute,
                        hour=hour,
                        day_of_month=dom,
                        month_of_year=month,
                        day_of_week=dow,
                    )

                    ChainedTask.objects.create(
                        chain=chain,
                        order=idx
                    )

                    pt = PeriodicTask.objects.create(
                        name=task_name,
                        task="schedule_task_execution", # TODO:task名称传入，等待前端修改
                        args=json.dumps([project_id, node_list]),
                        kwargs=json.dumps({"chain_id":chain.id, "project_id": project_id, "version_hash": "LATEST", "nodes": node_list, "task_id": task.id}),
                        crontab=crontab,
                        enabled=True
                    )

                    task_map[task_name] = pt

                elif trigger == "previous":
                    if task_status == "exist":
                        # 使用现有 Task
                        # Use existing Task
                        try:
                            task = Task.objects.get(id=task_id)
                        except Task.DoesNotExist:
                            return JsonResponse(
                                {"error": f"未找到任务 ID '{task_id}'，请确保任务存在"},
                                status=status.HTTP_400_BAD_REQUEST
                            )
                    else:
                        previous_name = schedule["previous"]
                        if previous_name not in task_map:
                            return JsonResponse(
                                {"error": f"未找到前置任务 '{previous_name}'，请确保定义顺序正确"},
                                status=status.HTTP_400_BAD_REQUEST
                            )

                    # 设置不执行的 crontab
                    crontab, _ = CrontabSchedule.objects.get_or_create(
                        minute=0,
                        hour=0,
                        day_of_month=0,
                        month_of_year=0,
                        day_of_week=0,
                    )

                    previous_task = Task.objects.get(name=previous_name)

                    # 创建 ChainedTask 记录
                    ChainedTask.objects.create(
                        chain=chain,
                        task = previous_task, # 这里是前置任务的 ID
                        next_task = task,  # 这里是当前任务的 ID，在使用时使用task寻找当前task值，即可获得后续task_id
                        order=idx
                    )
                    # TODO:[Delete]创建celery_beat周期性任务，虽然不执行，但保持一下一致，以后删
                    pt = PeriodicTask.objects.create(
                        name=task_name,
                        task="schedule_task_execution",
                        args=json.dumps([project_id, node_list]),
                        kwargs=json.dumps({"chain_id":chain.id, "task_id": task.id, "project_id": project_id, "version_hash": "LATEST", "nodes": node_list}),
                        crontab=crontab,
                        enabled=False  # 后续任务不由 crontab 启动
                    )

                    task_map[task_name] = pt

                task.periodic_task = pt
                task.save()

            return JsonResponse({"message": f"任务链 '{name}' 创建成功，id '{chain.id}'"}, status=status.HTTP_201_CREATED)

        except Exception as e:
            return JsonResponse({"error": f"任务链创建失败: {str(e)}"}, status=status.HTTP_500_INTERNAL_SERVER_ERROR)

################################################################
# Araneae_worknode Task                                        #
################################################################
class TaskViewSet(viewsets.ModelViewSet):
    """
    Task 视图集
    """
    queryset = Task.objects.all()
    serializer_class = TaskSerializer

    @action(detail=False, methods=["post"], url_path="execute")
    @csrf_exempt
    def execute_task(self, request):
        """
        手动触发任务执行，向指定节点列表精准派发。
        api: POST /api/tasks/execute/

        Body:
          project_id   (int)       项目 ID
          version_hash (str)       版本 hash，传 "LATEST" 自动解析
          node_list    (list[str]) 目标节点 hash 列表
          task_id      (int|null)  可选，关联已有 Task 记录
          chain_id     (int|null)  可选，关联任务链
        """
        data = request.data
        project_id   = data.get("project_id")
        version_hash = data.get("version_hash", "LATEST")
        node_list    = data.get("node_list", [])
        task_id      = data.get("task_id")
        chain_id     = data.get("chain_id")

        # 1) 参数校验
        if not project_id:
            return JsonResponse({"error": "project_id is required"}, status=400)
        if not node_list:
            return JsonResponse({"error": "node_list must not be empty"}, status=400)

        # 2) 获取项目和版本信息
        try:
            project = Project.objects.get(id=project_id)
        except Project.DoesNotExist:
            return JsonResponse({"error": "Project not found"}, status=404)

        try:
            if version_hash == "LATEST":
                version = Version.objects.filter(project=project).order_by('-created_at').first()
                if not version:
                    return JsonResponse({"error": "No version found for project"}, status=404)
                version_hash = version.version_hash
            else:
                version = Version.objects.get(version_hash=version_hash)
        except Version.DoesNotExist:
            return JsonResponse({"error": f"Version '{version_hash}' not found"}, status=404)

        print(f"[execute_task] project={project.project_hash} version={version_hash} nodes={node_list}")

        # 3) 构造任务 kwargs 并精准派发
        task_kwargs = {
            "task_id":      task_id,
            "chain_id":     chain_id,
            "project_hash": project.project_hash,
            "version_hash": version_hash,
        }

        try:
            celery_task_ids = dispatch_task_to_nodes(
                task_kwargs=task_kwargs,
                node_list=node_list,
            )
        except Exception as e:
            return JsonResponse({"error": f"Dispatch failed: {str(e)}"}, status=500)

        # 4) 为每个目标节点创建 pending TaskRecord
        records_created = []
        for node_hash in node_list:
            try:
                node = Node.objects.get(node_hash=node_hash)
                record = create_task_record(node, project, version, "pending")
                records_created.append(node_hash)
            except Node.DoesNotExist:
                print(f"[WARNING] Node not found for record: {node_hash}")

        return JsonResponse({
            "message": "Task dispatched.",
            "nodes":   node_list,
            "celery_task_ids": celery_task_ids,
            "records_created": records_created,
        }, status=202)
