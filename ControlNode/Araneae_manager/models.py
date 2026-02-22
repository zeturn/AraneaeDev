"""
Araneae_manager/models.py
"""
# -*- coding: utf-8 -*-
from django.db import models
from django.forms import JSONField
from django_celery_beat.models import PeriodicTask
from django.utils import timezone

from Araneae_main.models import Workplace




class Node(models.Model):
    """
    节点类，表示一个工作节点。
    Attributes:
        name (str): 节点名称。
        description (str): 节点描述。
        HDID (str): 节点的唯一标识符。
        status (str): 节点状态，可选值：'active', 'inactive', 'unreachable'。
        ip_address (str): 节点的 IP 地址。
        port (int): 节点的端口号，默认值为 5000。
        rpc_url (str): 节点的 RPC URL。
        auth_key (str): 节点的认证密钥。
        cpu_info (str): 节点的 CPU 信息。
        memory_info (str): 节点的内存信息。
        last_active_time (datetime): 上次活动时间。
        celery_queue (str): Celery 队列名称。
        created_at (datetime): 创建时间，自动生成。
        updated_at (datetime): 更新时间，自动生成。
    """
    STATUS_CHOICES = [
        ('active', 'Active'),
        ('inactive', 'Inactive'),
        ('unreachable', 'Unreachable'),
    ]

    name = models.CharField(max_length=100, unique=True)
    node_hash = models.CharField(max_length=100, unique=True, null=True)  # 节点唯一标识符
    description = models.TextField(blank=True, null=True)
    HDID = models.CharField(max_length=100, blank=True, null=True)
    status = models.CharField(max_length=20,choices=STATUS_CHOICES,default='inactive')

    # Communication info
    ip_address = models.GenericIPAddressField(null=True, blank=True)
    port = models.PositiveIntegerField(default=5000)
    rpc_url = models.URLField(null=True)
    auth_key = models.CharField(max_length=255, blank=True, null=True)

    # Performance info
    cpu_info = models.TextField(blank=True, null=True)
    memory_info = models.TextField(blank=True, null=True)

    # Task and scheduling
    last_active_time = models.DateTimeField(blank=True, null=True)
    celery_queue = models.CharField(max_length=100, null=True)

    # Other info
    created_at = models.DateTimeField(auto_now_add=True)
    updated_at = models.DateTimeField(auto_now=True)
    is_enabled = models.BooleanField(default=True)

    # Runtime capabilities (populated by refresh_capabilities action)
    # 存储从执行节点拉取的运行时环境列表 [{key, name, available, version, path}, ...]
    runtime_capabilities = models.JSONField(null=True, blank=True, default=list)

    def __str__(self):
        return self.name

class Schedule(models.Model):
    """
    任务调度类，定义了任务的调度信息。
    Attributes:
        workplace (Workplace): 关联的工作场所。
        name (str): 调度名称。
        description (str): 调度描述。
        mode (str): 调度模式，可选值：'once', 'recurring'。
        order (str): 调度顺序。
        enabled (bool): 是否启用调度，默认值为 True。
        created_at (datetime): 创建时间，自动生成。
        updated_at (datetime): 更新时间，自动生成。
    """
    MODE_CHOICES = [
        ('once', 'Once'),
        ('recurring', 'Recurring'),
    ]

    workplace = models.ForeignKey(Workplace, on_delete=models.CASCADE, related_name='schedules')
    name = models.CharField(max_length=100)
    description = models.TextField(null=True, blank=True)
    mode = models.CharField(max_length=10, choices=MODE_CHOICES)
    order = models.TextField()
    enabled = models.BooleanField(default=True)
    created_at = models.DateTimeField(auto_now_add=True)
    updated_at = models.DateTimeField(auto_now=True)

class Task(models.Model):
    """
    任务的抽象类，指代例如周期性可复用的任务。不是具体某一个被执行的任务示例。每一次被执行的任务由 TaskRecord 记录。
    Attributes:
        schedule (Schedule): 任务的调度信息。
        status (str): 任务状态，可选值：'pending', 'running', 'completed', 'failed'。
        created_at (datetime): 任务创建时间。
        updated_at (datetime): 任务更新时间，默认当前时间。

    Status:
        - pending: 任务待执行
        - running: 任务正在执行
        - completed: 任务已完成
        - failed: 任务执行失败

    """

    # 任务的调度信息
    name = models.CharField(max_length=100, unique=False, null=True)  # 任务名称
    node_hash = models.CharField(max_length=100, unique=False, null=True)  # 节点唯一标识符
    description = models.TextField(blank=True, null=True)  # 任务描述
    schedule = models.ForeignKey(Schedule, on_delete=models.CASCADE, related_name='tasks', null=True, blank=True)
    workplace = models.ForeignKey('Araneae_main.Workplace', on_delete=models.CASCADE, related_name='tasks', null=True, blank=True)
    celery_label = models.CharField(max_length=200, unique=False) # 任务标签名称
    periodic_task = models.OneToOneField(PeriodicTask, on_delete=models.CASCADE,  related_name='reflection_task_relation', null=True, blank=True) # 对应的 PeriodicTask 对象
    args = models.TextField(blank=True, null=True)  # 任务参数
    kwargs = models.TextField(blank=True, null=True)  # 任务参数
    enable = models.BooleanField(default=False)  # 是否启用
    created_at = models.DateTimeField(auto_now_add=True)
    updated_at = models.DateTimeField(auto_now=True)

    def __str__(self):
        return f"{self.name} - {self.status}"

class TaskRecord(models.Model):
    """
    记录 Task 的执行情况。
    Attributes:
        node (str): 节点名称。
        project (str): 项目名称。
        version (str): 版本号。
        task_status (str): 任务状态，可选值：'pending', 'running', 'finished', 'failed'。
        task_result (Any): 任务执行结果。
        task_created_at (datetime): 任务创建时间。
        task_updated_at (datetime): 任务更新时间，默认当前时间。
    """
    STATUS_CHOICES = [
        ('pending', 'Pending'),
        ('running', 'Running'),
        ('finished', 'Finished'),
        ('canceled', 'Canceled'),
        ('error', 'Error'),
        ('timeout', 'Timeout'),
        ('failed', 'Failed'),
        ('retry', 'Retry'),
    ]

    node = models.ForeignKey('Araneae_manager.Node', on_delete=models.CASCADE, related_name="task_records")
    project = models.ForeignKey('Araneae_repo.Project', on_delete=models.CASCADE, related_name="task_records")
    version = models.ForeignKey('Araneae_repo.Version', on_delete=models.CASCADE, related_name="task_records")
    workplace = models.ForeignKey('Araneae_main.Workplace', on_delete=models.CASCADE, related_name='task_records', null=True, blank=True)
    schedule = models.ForeignKey('Araneae_manager.Schedule', on_delete=models.CASCADE, related_name='task_records', null=True, blank=True)
    task_id = models.CharField(max_length=100, blank=True, null=True)
    task_status = models.CharField(max_length=20,choices=STATUS_CHOICES)
    task_result = models.TextField(blank=True, null=True)
    task_created_at = models.DateTimeField(auto_now_add=True)
    task_updated_at = models.DateTimeField(auto_now=True)

    def __str__(self):
        return f"{self.task_name} on {self.node.name}"

class TaskChain(models.Model):
    """
    任务链，用于将多个任务串联在一起。
    Attributes:
        name (str): 任务链名称。
        description (str): 任务链描述。
        enabled (bool): 是否启用任务链，默认值为 False。
        created_at (datetime): 创建时间，自动生成。
    """
    name = models.CharField(max_length=100, unique=False)
    description = models.TextField(blank=True, null=True)
    enabled = models.BooleanField(default=False)
    created_at = models.DateTimeField(auto_now_add=True)


class ChainedTask(models.Model):
    """
    任务链中的具体任务关联表。
    Attributes:
        task_id (Task): 当前任务的 ID。
        next_task_id (Task): 下一个任务的 ID。
        chain (TaskChain): 关联的任务链。
        order (int): 任务在链中的顺序，默认值为 0。TODO:作为优先级评标
    """
    chain = models.ForeignKey(TaskChain, related_name='tasks', on_delete=models.CASCADE)
    task = models.ForeignKey(Task, related_name='next_tasks', on_delete=models.CASCADE, null=True, blank=True)
    next_task = models.ForeignKey(Task, related_name='previous_tasks', on_delete=models.CASCADE, null=True, blank=True)
    order = models.PositiveIntegerField(default=0)  # 任务链顺序

    class Meta:
        ordering = ['order']

class NodeCurrentStatus(models.Model):
    """
    中文：节点当前状态模型
    英文：Node current status model
    """
    node = models.OneToOneField(Node, on_delete=models.CASCADE, primary_key=True)
    cpu_percent   = models.FloatField()
    memory_used   = models.BigIntegerField()
    memory_total  = models.BigIntegerField()
    gpu_info      = models.JSONField(null=True, blank=True, default=list)  # 存 [{"index":0,"mem_used":…,"util":…},…]，无 GPU 时为 []
    updated_at    = models.DateTimeField(auto_now=True)

    class Meta:
        verbose_name = '节点当前状态'
        verbose_name_plural = '节点当前状态列表'
        ordering = ['-updated_at']

    def __str__(self):
        """
        中文：返回简短的模型实例字符串
        英文：Return a concise string representation of the instance
        """
        return f'NodeCurrentStatus(node_id={self.node_id})'

class NodeMetricArchive(models.Model):
    """
    中文：节点指标归档模型
    英文：Node metric archive model
    """
    # 中文：关联的节点
    # 英文：Referenced node
    node = models.ForeignKey(
        'Araneae_manager.Node',
        on_delete=models.CASCADE,
        related_name='metric_archives'
    )

    # 中文：数据采集时间
    # 英文：Time when metrics were collected
    timestamp = models.DateTimeField(default=timezone.now)

    # 中文：CPU 使用率（百分比）
    # 英文：CPU usage percentage
    cpu_percent = models.FloatField()

    # 中文：已用内存（字节）
    # 英文：Memory used in bytes
    memory_used = models.BigIntegerField()

    # 中文：总内存（字节）
    # 英文：Total memory in bytes
    memory_total = models.BigIntegerField()

    # 中文：GPU 使用情况列表，每项格式 {"index": int, "mem_used": int, "util": float}
    # 英文：List of GPU usage info, each item format {"index": int, "mem_used": int, "util": float}
    gpu_info = models.JSONField(null=True, blank=True, default=list)

    class Meta:
        verbose_name = '节点指标归档'
        verbose_name_plural = '节点指标归档列表'
        ordering = ['-timestamp']

    def __str__(self):
        """
        中文：返回简短的模型实例字符串
        英文：Return a concise string representation of the instance
        """
        return f'NodeMetricArchive(node_id={self.node_id}, time={self.timestamp})'

class ProgrammingLanguage(models.Model):
    """
    Programming Language Model
    语言类，表示支持的编程语言。
    Attributes:
        name (str): 语言名称。
        created_at (datetime): 创建时间，自动生成。
        updated_at (datetime): 更新时间，自动生成。
    """
    name = models.CharField(max_length=50, unique=True)
    created_at = models.DateTimeField(auto_now_add=True)
    updated_at = models.DateTimeField(auto_now=True)

    def __str__(self):
        return self.name

class Identity(models.Model):
    """
    当前节点的身份
    """
    __tablename__ = 'identities'  # 确保表名正确
    id = models.AutoField(primary_key=True)
    identity_hash = models.CharField(max_length=100, unique=True, null=False)
    created_at = models.DateTimeField(auto_now_add=True)
    updated_at = models.DateTimeField(auto_now=True)

    def __str__(self):
        return self.identity_hash