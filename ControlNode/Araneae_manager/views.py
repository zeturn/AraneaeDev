"""
Araneae_worknode/views.py
"""
import traceback
import hashlib
import hmac
import time
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
from Araneae_manager.tasks import schedule_task_execution
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
            # Allow local development without shared secret.
            return settings.MODE == "dev"

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
        任务回调
        api: /task/callback
        :param request:

        :return: JsonResponse
        """
        if not self._verify_callback_signature(request):
            return JsonResponse({"error": "Invalid callback signature"}, status=403)

        print("[SUCCESS][TASK]Received task callback")
        data = request.data

        print(data)

        node_hash = data.get('node')
        project_hash = data.get('project')
        version_hash = data.get('version')
        task_status = data.get('task_status', 'pending')
        task_result = data.get('task_result', None)

        print(f'[DEBUG] Node: {node_hash}, Project: {project_hash}, Version: {version_hash}, Status: {task_status}, Result: {task_result}')

        try:
            node = Node.objects.get(node_hash=node_hash) if node_hash else None
        except Node.DoesNotExist:
            # === 以下功能：记录 Node 未找到情况 ===
            print(f"[WARNING] Node with hash {node_hash} not found.")
            node = None

        try:
            project = Project.objects.get(project_hash=project_hash) if project_hash else None
        except Project.DoesNotExist:
            print(f"[WARNING] Project with hash {project_hash} not found.")
            project = None

        try:
            version = Version.objects.get(version_hash=version_hash) if version_hash else None
        except Version.DoesNotExist:
            print(f"[WARNING] Version with hash {version_hash} not found.")
            version = None

        create_task_record(node, project, version, task_status, task_result)
        # 检查任务元参数
        current_task_id = data.get('task_id')
        task_chain_id = data.get('task_chain_id')

        if task_chain_id:
            # 任务链回调
            try:
                task_chain = TaskChain.objects.get(id=task_chain_id)
            except TaskChain.DoesNotExist:
                print(f"[Task][CallBack]⚠️ 任务链不存在: id={task_chain_id}")
                task_chain = None
            if task_chain:

                self.continue_task_chain(current_task_id,task_chain_id)  # 调用检查任务链的方法
                print(f"🔗 任务链 {task_chain.name} 回调完毕")
        else:
            print("⚠️ 任务链不存在, 执行结束")

        return JsonResponse({"message": "Callback received, task record created.", "status": 200})


    def continue_task_chain(self, current_task_id, task_chain_id):
        """
        检查任务链是否存在
        :return:
        """
        try:
            # 获取 Task 实例
            try:
                current_task = Task.objects.get(id=current_task_id)
            except Task.DoesNotExist:
                print(f"⚠️ 未找到 Task 对象: id={current_task_id}")
                return JsonResponse({"error": "Task not found"}, status=404)

            print(f"🔍 检索到当前任务id: {current_task.id}")

            # 获取 PeriodicTask 实例
            try:
                # Adjust this query as needed. If current_task. is an ID, use:
                current_pt = PeriodicTask.objects.get(id=current_task.periodic_task.id)
            except PeriodicTask.DoesNotExist:
                print(f"⚠️ 未找到 PeriodicTask: id={current_task.periodic_task}")
                return JsonResponse({"error": "PeriodicTask not found"}, status=404)

            print(f"🔗 当前任务对应的 PeriodicTask id: {current_pt.id}")

            # 获取 ChainedTask 实例
            try:
                current_chain_task = ChainedTask.objects.get(id=current_task.id)
            except ChainedTask.DoesNotExist:
                print(f"⚠️ 未找到 ChainedTask 与 PeriodicTask id={current_pt.id} 对应的任务")
                return JsonResponse({"error": "ChainedTask not found"}, status=404)

            print(f"🔗 当前任务对应的 ChainedTask: {current_chain_task.id}")

            # 查找所有后续任务
            next_chained_tasks = ChainedTask.objects.filter(
                chain=current_chain_task.chain,
                task=current_task
            )
            print(f"🔗 共找到 {next_chained_tasks.count()} 个次级后续任务")

            for next_chained_task in next_chained_tasks:
                try:
                    next_task = Task.objects.get(id=next_chained_task.next_task_id)
                    next_period_task = PeriodicTask.objects.get(id=next_task.periodic_task.id)
                except Task.DoesNotExist:
                    print(f"⚠️ 未找到后续任务对象: {next_chained_task}")
                    continue

                args = json.loads(next_period_task.args or "[]")
                if isinstance(args, dict):
                    args = [args]
                kwargs = json.loads(next_period_task.kwargs or "{}")
                print(f"➡️ 触发后续任务: {next_period_task.name} | args={args} kwargs={kwargs}")
                schedule_task_execution.apply_async(args=args, kwargs=kwargs)

        except Exception as e:
            print(f"❌ 执行失败: {e}")

        return "Task dispatched and chain checked."


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
        执行任务
        api: /task/execute
        :param request:
        :return:
        """
        data = request.data
        project_id = data.get("project_id")
        version_hash = data.get("version_hash")
        node_list = data.get("node_list")
        task_id = data.get("task_id")

        # 1) 获取项目和版本信息
        try:
            project = Project.objects.get(id=project_id)
            version = Version.objects.get(version_hash=version_hash)
            print(f"Project: {project}, Version: {version}")
        except Project.DoesNotExist:
            return JsonResponse({"error": "Project not found"}, status=status.HTTP_404_NOT_FOUND)


