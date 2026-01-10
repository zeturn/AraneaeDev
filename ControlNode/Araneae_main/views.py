"""
Araneae_main/views.py
"""
import json
import os
import zipfile
from celery import chain

from django.views import View
from django.conf import settings
from django.http import JsonResponse
from django.middleware.csrf import get_token
from django_celery_beat.models import PeriodicTask, CrontabSchedule

from rest_framework.decorators import action
from rest_framework import viewsets
from rest_framework.permissions import IsAuthenticated
from rest_framework_simplejwt.tokens import RefreshToken
from rest_framework.views import APIView
from rest_framework.decorators import api_view
from rest_framework.response import Response
from rest_framework import status

from django.contrib.auth.models import User
from Araneae_repo.views import check_project_code_files
from utils.uuid.generate6uuid import generate_6_uuid
from .models import Profile, Workplace, Team, WorkplaceUserPermission, WorkplaceTeamPermission
from Araneae_manager.models import Task,  Schedule
from Araneae_repo.models import Project, Version
from .serializers import UserSerializer, ProfileSerializer, WorkplaceSerializer, TeamSerializer, TeamWithRoleSerializer, \
    TeamWithMembersSerializer
from Araneae_manager.serializers import ScheduleSerializer, TaskRecordSerializer, TaskSerializer
from Araneae_repo.serializers import ProjectSerializer
from Araneae_manager.tasks import publish_to_rabbitmq, schedule_task_execution


@api_view(['GET'])
def csrf_token_view(request):
    """
    获取 CSRF Token
    @param request: request
    @return:JsonResponse
    """
    return JsonResponse({'csrfToken': get_token(request)})

class UserViewSet(viewsets.ModelViewSet):
    """
    用户视图集，提供用户的基本信息和权限。
    """
    queryset = User.objects.all()
    serializer_class = UserSerializer
    permission_classes = [IsAuthenticated]

    def get_queryset(self):
        # 列表时只显示当前用户的所有 User（通常只有一个）
        return User.objects.filter(pk=self.request.user.pk)

class ProfileViewSet(viewsets.ModelViewSet):
    queryset = Profile.objects.all()
    serializer_class = ProfileSerializer
    permission_classes = [IsAuthenticated]

    def get_queryset(self):
        # 列表时只显示当前用户的所有 Profile（通常只有一个）
        return Profile.objects.filter(user=self.request.user)

    def get_object(self):
        # 如果不存在则自动创建
        profile, created = Profile.objects.get_or_create(user=self.request.user)
        return profile

    def update(self, request, *args, **kwargs):
        """
        重写 update 方法，使其只能更新当前用户的 Profile
        """
        partial = kwargs.pop('partial', False)
        instance = self.get_object()   # 此处保证 instance 一定存在
        serializer = self.get_serializer(instance, data=request.data, partial=partial)
        serializer.is_valid(raise_exception=True)
        self.perform_update(serializer)
        return Response(serializer.data)



class WorkplaceViewSet(viewsets.ModelViewSet):
    queryset = Workplace.objects.all()
    serializer_class = WorkplaceSerializer

    def create(self, request, *args, **kwargs):
        """
        重写创建 Workplace 实例，并可选择性地创建 Team ↔ Workplace 的权限。
        Override Create a Workplace instance and optionally create Team ↔ Workplace permissions.
        :param request:
        :param args:
        :param kwargs:
        :return:
        """
        # 1. 验证并反序列化
        serializer = self.get_serializer(data=request.data)
        serializer.is_valid(raise_exception=True)

        # 2. 保存 Workplace 实例
        workplace = serializer.save()

        # 3. 如果传了 team_id，则创建 Team ↔ Workplace 的权限
        team_id = request.data.get('team_id')
        if team_id is not None:
            try:
                team = Team.objects.get(id=team_id)
            except Team.DoesNotExist:
                return Response({'error': 'Team not found'}, status=status.HTTP_400_BAD_REQUEST)

            # 角色可根据需要改成 OWNER / EDITOR
            WorkplaceTeamPermission.objects.create(
                workplace=workplace,
                team=team,
                permission=WorkplaceTeamPermission.Permission.OWNER
            )

        # 4. 重新序列化输出
        out_serializer = self.get_serializer(workplace)
        headers = self.get_success_headers(out_serializer.data)
        return Response(out_serializer.data, status=status.HTTP_201_CREATED, headers=headers)

    @action(detail=True, methods=['post'], url_path='add_teams')
    def add_teams(self, request, *args, **kwargs):
        """
        向 Workplace 添加合作 Team
        :param request:
        :param args:
        :param kwargs:
        :return:

        api: /api/workplaces/{workplace_id}/add_teams/
        调用示例：
        {
            "team_ids": [1, 2, 3],
        }
        """
        # 1. 获取 Workplace 实例
        workplace = self.get_object()

        # 2. 获取 team_ids
        team_ids = request.data.get('team_ids')
        if not team_ids:
            return Response({'error': 'No team IDs provided'}, status=status.HTTP_400_BAD_REQUEST)

        # 3. 创建 Workplace ↔ Team 的权限
        for team_id in team_ids:
            try:
                team = Team.objects.get(id=team_id)
                WorkplaceTeamPermission.objects.create(
                    workplace=workplace,
                    team=team,
                    permission=WorkplaceTeamPermission.Permission.OWNER
                )
            except Team.DoesNotExist:
                return Response({'error': f'Team with ID {team_id} not found'}, status=status.HTTP_400_BAD_REQUEST)

        return Response({'message': 'Teams added successfully'}, status=status.HTTP_200_OK)


    @action(detail=True, methods=['post'], url_path='add_people')
    def add_people(self, request, *args, **kwargs):
        """
        向 Workplace 添加个体合作者
        :param request:
        :param args:
        :param kwargs:
        :return:

        api: /api/workplaces/{workplace_id}/add_people/
        调用示例：
        {
            "user_ids": [1, 2, 3],
        }
        """
        # 1. 获取 Workplace 实例
        workplace = self.get_object()

        # 2. 获取 user_ids
        user_ids = request.data.get('user_ids')
        if not user_ids:
            return Response({'error': 'No user IDs provided'}, status=status.HTTP_400_BAD_REQUEST)

        # 3. 创建 Workplace ↔ User 的权限
        for user_id in user_ids:
            try:
                user = Profile.objects.get(id=user_id)
                WorkplaceUserPermission.objects.create(
                    workplace=workplace,
                    user=user,
                    permission=WorkplaceUserPermission.Permission.OWNER
                )
            except Profile.DoesNotExist:
                return Response({'error': f'User with ID {user_id} not found'}, status=status.HTTP_400_BAD_REQUEST)

        return Response({'message': 'Users added successfully'}, status=status.HTTP_200_OK)

    def get_serializer_context(self):
        """
        获取序列化器上下文
        :return:
        """
        context = super().get_serializer_context()
        context.update({
            'request': self.request
        })
        return context

    @action(detail=False, methods=['get'], url_path='my_workplaces')
    def my_workplaces(self, request):
        """
        列出当前用户相关的所有 Workplace，包括：
        1. 该用户所在团队拥有权限的 Workplace
        2. 该用户个人拥有权限的 Workplace
        """
        user = request.user

        # 1. 获取用户所属的所有 Team
        teams = user.teams.all()

        # 2. 查询这些 Team 对应的 Workplace
        wp_by_team = Workplace.objects.filter(
            workplaceteampermission__team__in=teams
        )

        # 3. 查询用户个人对 Workplace 的权限
        wp_by_user = Workplace.objects.filter(
            workplaceuserpermission__user=user
        )

        # 4. 合并结果并去重
        qs = (wp_by_team | wp_by_user).distinct()

        # 5. 分页并返回序列化数据
        page = self.paginate_queryset(qs)
        if page is not None:
            serializer = self.get_serializer(page, many=True)
            return self.get_paginated_response(serializer.data)

        serializer = self.get_serializer(qs, many=True)
        return Response(serializer.data)

    @action(detail=True, methods=['get'], url_path='workplaces_projects')
    # 获取当前 Workplace 的所有 Project
    def workplaces_projects(self, request, pk=None):
        """
        获取当前 Workplace 的所有 Project
        @param request:
        @param pk:
        @return:
        """
        workplace = self.get_object()
        projects = workplace.projects.all()
        serializer = ProjectSerializer(projects, many=True)
        return Response(serializer.data)

    @action(detail=True, methods=['get'], url_path='workplaces_schedules')
    def workplaces_schedules(self, request, pk=None):
        """
        获取当前 Workplace 的所有 Schedule
        :param request:
        :param pk:
        :return:
        """
        workplace = self.get_object()
        schedules = workplace.schedules.all()
        serializer = ScheduleSerializer(schedules, many=True)
        return Response(serializer.data)

    @action(detail=True, methods=['get'], url_path='workplaces_taskrecords')
    def workplace_taskrecords(self, request, pk=None):
        """
        返回指定工作区的 TaskRecord 列表及其数量。
        Return the list of TaskRecords and their count.
        """
        try:
            # 获取工作区实例
            # Retrieve the workplace instance
            workplace = self.get_object()

            # 获取关联的任务记录
            # Get associated TaskRecord queryset
            records = workplace.task_records.all()

            # 序列化任务记录
            # Serialize TaskRecord instances
            serializer = TaskRecordSerializer(records, many=True)
            data = serializer.data

            # 计算记录数量
            # Count the serialized records
            count = len(data)

        except Exception as e:
            # 记录错误并返回默认响应
            # Log error and return default response
            print(f"Error fetching TaskRecords for Workplace ID {pk}: {e}")
            return Response({'records': [], 'count': 0}, status=status.HTTP_500_INTERNAL_SERVER_ERROR)

        # 返回序列化数据和数量
        # Return serialized data and count
        return Response({'records': data, 'count': count}, status=status.HTTP_200_OK)

    @action(detail=True, methods=['get'], url_path='workplaces_tasks')
    def workplace_tasks(self, request, pk=None):
        """
        返回指定工作区的 Task 列表及其数量。
        Return the list of Tasks and their count.
        """
        try:
            # 获取工作区实例
            workplace = self.get_object()

            # 获取关联的任务记录
            tasks = workplace.tasks.all()

            # 序列化任务记录
            serializer = TaskSerializer(tasks, many=True)
            data = serializer.data

            # 计算记录数量
            count = len(data)

        except Exception as e:
            # 记录错误并返回默认响应
            print(f"Error fetching Tasks for Workplace ID {pk}: {e}")
            return Response({'tasks': [], 'count': 0}, status=status.HTTP_500_INTERNAL_SERVER_ERROR)

        # 返回序列化数据和数量
        return Response({'tasks': data, 'count': count}, status=status.HTTP_200_OK)


class ProjectViewSet(viewsets.ModelViewSet):
    queryset = Project.objects.all()
    serializer_class = ProjectSerializer

    def create(self, request, *args, **kwargs):
        """
        重写 create 方法
        override create method
        :param request:
        :param args:
        :param kwargs:
        :return:
        """
        # 1. 验证并反序列化
        serializer = self.get_serializer(data=request.data)
        serializer.is_valid(raise_exception=True)

        # 2. 保存实例
        project = serializer.save()

        # 3. 重新序列化，用于响应
        out_serializer = self.get_serializer(project)

        # 4. 构造 Location header 指向新资源
        headers = self.get_success_headers(out_serializer.data)
        # （headers 默认会包含 'Location': '/api/projects/{id}/' ）

        # 5. 返回 201 CREATED
        return Response(
            out_serializer.data,
            status=status.HTTP_201_CREATED,
            headers=headers
        )

    @action(detail=False, methods=['get'], url_path='my_projects')
    def my_projects(self, request):
        """
        api: /api/projects/my_projects/
        获取当前用户作为 owner 或 editor 的所有 Project
        @param request:
        @return:
        """
        user = request.user
        # 获取当前用户作为 owner 或 editor 的所有 Project
        projects = Project.objects.filter(workplace__owners=user) | Project.objects.filter(workplace__editors=user)

        # 对 projects 进行去重
        projects = projects.distinct()

        # 序列化 projects
        serializer = self.get_serializer(projects, many=True)

        # 返回序列化的数据
        return Response(serializer.data)

    @action(detail=True, methods=['get'], url_path='versions')
    def get_versions(self, request, pk=None):
        """
        api: /api/projects/{project_id}/versions/
        获取指定项目的所有版本
        @param request:
        @param pk:
        @return: JsonResponse
        """
        project = self.get_object()
        versions = Version.objects.filter(project=project)
        version_data = [{'version_hash': v.version_hash, 'release_date': v.release_date} for v in versions]
        return Response({'versions': version_data})

    @action(detail=True, methods=['get'], url_path='check_code_files')
    def check_code_files(self, request, pk=None):
        """
        api: /api/projects/{project_id}/check_code_files/
        检查指定项目是否有代码文件
        @param request:
        @param pk:
        @return: JsonResponse
        """
        project = self.get_object()
        has_files = check_project_code_files(project.id)
        return JsonResponse({'has_code_files': has_files})

    def perform_create(self, serializer):
        """
        创建 Project ，并设置状态为 'Running'
        @param serializer:
        """
        project = serializer.save()
        project.status = 'Running'
        project.save()

    @action(detail=True, methods=['get'], url_path='get_repo')
    # url path: /api/projects/{project_id}/get_repo/
    def get_repo(self, request, pk=None):
        """
        获取指定项目的所有版本
        """
        # 根据 project_id 获取项目
        project = self.get_object()
        # 获取项目的所有版本
        versions = Version.objects.filter(project=project)
        # 获取版本的版本号和发布日期
        version_data = [{'version_hash': v.version_hash, 'release_date': v.release_date} for v in versions]
        return Response({'versions': version_data})

    @action(detail=True, methods=['get'], url_path='ultimateAGI')
    def ultimate_artificial_general_intelligence(self, request, *args, **kwargs):
        """
        终极 AGI - 符号主义
        The ultimate answer
        to the ultimate question
        of Life,
        the Universe,
        and Everything.
        """
        return Response({'answer': 42})


class FileUploadViewSet(View):
    """
    处理文件上传的视图集
    """
    @action(detail=False, methods=['post'], url_path='upload-script')
    def upload_script(request, *args, **kwargs):
        """
        Handles file uploads and associates them with a project version.
        @param request: HttpRequest
        @param args:
        @param kwargs:
        @return: JsonResponse
        """
        # 获取 project_id
        project_id = request.POST.get('project_id')
        if not project_id:
            return JsonResponse({'error': 'Project ID is required'}, status=400)

        # 获取上传的文件
        uploaded_file = request.FILES.get('file')
        if not uploaded_file:
            return JsonResponse({'error': 'No file uploaded'}, status=400)

        try:
            # 验证项目是否存在
            project = Project.objects.get(id=project_id)
        except Project.DoesNotExist:
            return JsonResponse({'error': 'Project does not exist'}, status=404)

        version_hash = generate_6_uuid()
        # 创建新的 Version 记录
        version = Version.objects.create(
            project = project,
            version_hash = version_hash,
        )

        # 确定目标路径：repo/{project_id}/{version_hash}/
        version_path = os.path.join(settings.BASE_DIR, 'Araneae_repo', 'repo', str(project_id), version_hash)
        os.makedirs(version_path, exist_ok=True)

        # 保存上传的压缩包到版本目录
        temp_zip_path = os.path.join(version_path, uploaded_file.name)
        with open(temp_zip_path, 'wb+') as destination:
            for chunk in uploaded_file.chunks():
                destination.write(chunk)

        # 解压缩文件到版本目录
        try:
            with zipfile.ZipFile(temp_zip_path, 'r') as zip_ref:
                zip_ref.extractall(version_path)
            os.remove(temp_zip_path)  # 删除压缩包
        except zipfile.BadZipFile:
            return JsonResponse({'error': 'Invalid ZIP file'}, status=400)

        return JsonResponse({
            'message': 'File uploaded and extracted successfully',
            'version': {
                'version_hash': version.version_hash,
                'release_date': version.release_date.strftime('%Y-%m-%d %H:%M:%S')
            }
        }, status=201)


############################################################
# WebRTC Assisted Control - Django page                     #
############################################################
from django.shortcuts import render
import requests as httpx

def webrtc_session(request):
    """
    Render a simple WebRTC page to control a remote Playwright session.
    Query params:
      - exec: ExecutionNode base URL (e.g., http://127.0.0.1:5001)
      - url:  Target page URL to open in Playwright
      - headless: true/false
    """
    exec_host = request.GET.get('exec', 'http://127.0.0.1:5001')
    target_url = request.GET.get('url')
    headless = request.GET.get('headless', 'true').lower() == 'true'
    session_id = request.GET.get('session_id') or None
    error = None
    if target_url and not session_id:
        try:
            resp = httpx.post(f"{exec_host}/webrtc/session", json={"url": target_url, "headless": headless}, timeout=10)
            if resp.status_code in (200, 201):
                session_id = resp.json().get('session_id')
            else:
                error = f"Session create failed: {resp.status_code} {resp.text}"
        except Exception as e:
            error = f"Failed to create session: {e}"
    ctx = {
        'exec_host': exec_host,
        'session_id': session_id or '',
        'error': error,
    }
    return render(request, 'webrtc/session.html', ctx)


class ScheduleViewSet(viewsets.ModelViewSet):
    queryset = Schedule.objects.all()
    serializer_class = ScheduleSerializer

    def create(self, request, *args, **kwargs):
        """
        创建 Schedule，并支持链式任务调度。
        @param request:
        @param args:
        @param kwargs:
        @return: Response
        """
        serializer = self.get_serializer(data=request.data)
        serializer.is_valid(raise_exception=True)
        self.perform_create(serializer)
        instance = serializer.instance

        if instance.enabled:
            order = instance.order
            try:
                parsed_data = json.loads(order)  # 解析 JSON
                name = parsed_data["name"]
                schedules = parsed_data["schedule"]
                enabled = parsed_data.get("enabled", False)

                print(f"创建链式任务: {name}")
                task_chain = []

                task_map = {}  # 存储任务名称与 Celery 任务的映射

                for schedule in schedules:
                    schedule_name = schedule["name"]
                    project_id = schedule["project_id"]
                    nodes = schedule["node"]
                    trigger = schedule["trigger"]

                    # 创建 Task 对象
                    task = Task.objects.create(
                        name=schedule_name,
                        celery_label=f"{name}-{schedule_name}",
                        args=json.dumps({"project_id": project_id, "nodes": nodes}),
                        kwargs=json.dumps({}),
                        enable=enabled,
                    )

                    if trigger == "crons":
                        # 解析 Cron 表达式
                        cron_expr = schedule["crons"]
                        minute, hour, day_of_month, month, day_of_week = cron_expr.split()

                    elif trigger == "previous":
                        minute, hour, day_of_month, month, day_of_week = 0

                    # 创建或获取 CrontabSchedule
                    crontab, _ = CrontabSchedule.objects.get_or_create(
                        minute=minute,
                        hour=hour,
                        day_of_month=day_of_month,
                        month_of_year=month,
                        day_of_week=day_of_week,
                    )



                    # 创建 Celery Beat 定时任务并传 task.id 到 kwargs
                    periodic_task = PeriodicTask.objects.create(
                        crontab=crontab,
                        name=f"{name}-{schedule_name}",
                        task="schedule_task_execution",
                        kwargs=json.dumps({"project_id": project_id, "version_hash": "LATEST", "nodes": nodes, "task_id": task.id}),
                        enabled=enabled,
                    )

                    task.periodic_task = periodic_task
                    task.save()

                    task_map[schedule_name] = periodic_task

                    if trigger == "crons":# 检查定时任务
                        print(f"创建 Celery Beat 定时任务: {schedule_name}，执行时间: {cron_expr}")

                    if trigger == "previous":
                        previous_task_nameE = schedule.get("previous")
                        print(f"创建 链式任务: {schedule_name}，前序任务: {previous_task_nameE}")
                        if previous_task_nameE in task_map:
                            prev_params = json.loads(task_map[previous_task_nameE].kwargs)
                            prev_task = schedule_task_execution.s(
                                project_id=prev_params.get("project_id"),
                                nodes=prev_params.get("nodes")
                            ).set(immutable=True)
                            current_task = schedule_task_execution.s(
                                project_id=project_id,
                                nodes=nodes
                            ).set(immutable=True)
                            # 将当前任务的签名保存到 task_map 中
                            task_map[schedule_name] = current_task
                            task_chain.append(prev_task | current_task)
                        else:
                            print(f"错误: 找不到前置任务 {previous_task_nameE}")

                        if task_chain:
                            chain(*task_chain).apply_async()

            except json.JSONDecodeError as e:
                print("JSON 解析错误:", e)

        else:
            print("Schedule created but not enabled")

        headers = self.get_success_headers(serializer.data)
        return Response(serializer.data, status=status.HTTP_201_CREATED, headers=headers)


class LogoutView(APIView):
    """
    注销会话
    """
    permission_classes = [IsAuthenticated]

    def post(self, request):
        """
        注销会话
        :param request:
        :return: Response
        """
        try:
            refresh_token = request.data.get("refresh")
            if refresh_token:
                token = RefreshToken(refresh_token)
                token.blacklist()  # 将刷新令牌加入黑名单
                return Response({"message": "Logged out successfully."}, status=status.HTTP_200_OK)
            return Response({"error": "Refresh token not provided."}, status=status.HTTP_400_BAD_REQUEST)
        except Exception as e:
            return Response({"error": str(e)}, status=status.HTTP_400_BAD_REQUEST)


def schedule_task(request):
    """
    废止
    接收任务调度请求并发布到 RabbitMQ。
    需要参数：task_name, payload
    """
    task_name = request.POST.get('task_name')
    payload = request.POST.get('payload')

    if not all([task_name, payload]):
        return JsonResponse({"error": "Missing parameters"}, status=400)

    # 调用 Celery 任务发布到 RabbitMQ
    result = publish_to_rabbitmq.delay(task_name, payload)
    return JsonResponse({"status": f"Task {task_name} scheduled successfully", "task_id": result.id})


################################################################
# Araneae_worknode Team                                        #
################################################################
class TeamViewSet(viewsets.ModelViewSet):
    """
    中文: 创建、查询、更新、删除 Team 视图，并提供 my_teams 接口
    English: Team viewset for CRUD operations with an extra my_teams endpoint.
    """
    queryset = Team.objects.all()
    permission_classes = [IsAuthenticated]

    def get_serializer_class(self):
        """
        中文: 根据 action 选择序列化器，my_teams 使用带角色信息的序列化器
        English: Choose serializer per action; use TeamWithRoleSerializer for my_teams.
        """
        if self.action == 'my_teams':
            return TeamWithRoleSerializer
        elif self.action == 'members':
            return TeamWithMembersSerializer
        return TeamSerializer

    def create(self, request, *args, **kwargs):
        """
        中文: 创建 Team，并捕获异常返回友好错误
        English: Create a Team instance, catching exceptions for user-friendly errors.
        """
        try:
            return super().create(request, *args, **kwargs)
        except Exception as e:
            return Response(
                {'detail': str(e)},
                status=status.HTTP_400_BAD_REQUEST
            )

    @action(detail=True, methods=['get'], url_path='members')
    def members(self, request, pk=None):
        """
        中文: 列出指定 Team 的所有成员及其角色
        English: List all members of this Team, including their roles.
        """
        team = self.get_object()
        serializer = self.get_serializer(team)
        # 返回整个 members 列表
        return Response(serializer.data, status=status.HTTP_200_OK)

    @action(detail=False, methods=['get'], url_path='my_teams')
    def my_teams(self, request):
        """
        中文: 返回当前用户所属的所有 Team 及其角色
        English: Return all Teams the current user belongs to, including role info.
        """
        user = request.user
        # 中文: 筛选当前用户所属的 Team，去重
        # English: Filter teams the user is a member of and distinct them
        teams = Team.objects.filter(members=user).distinct()

        # 中文: 支持分页
        # English: Support pagination
        page = self.paginate_queryset(teams)
        if page is not None:
            serializer = self.get_serializer(page, many=True)
            return self.get_paginated_response(serializer.data)

        serializer = self.get_serializer(teams, many=True)
        return Response(serializer.data)