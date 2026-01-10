#  -*- coding: utf-8 -*-
"""
Araneae ControlNode Araneae_repo models
"""
#  Copyright (c)   2025.5  Henry Zhao. All rights reserved.
#  From BJ.

from django.db import models

from Araneae_main.models import Workplace
from Araneae_manager.models import Node


# Create your models here.

class Project(models.Model):
    """
    项目类，表示一个代码项目。
    Attributes:
        workplace (Workplace): 关联的工作场所。
        name (str): 项目名称。
        description (str): 项目描述。
        language (str): 项目语言。
        command (str): 执行命令。
        entrance_file (str): 入口文件。
        mode (str): 执行模式，可选值：'manual', 'automatic'。
        created_at (datetime): 创建时间，自动生成。
        updated_at (datetime): 更新时间，自动生成。
    """
    MODE_CHOICES = [
        ('manual', 'Manual'),
        ('automatic', 'Automatic'),
    ]

    workplace = models.ForeignKey(Workplace, on_delete=models.CASCADE, related_name='projects')
    project_hash = models.CharField(max_length=50, unique=True)
    name = models.CharField(max_length=100)
    description = models.TextField(null=True, blank=True)
    language = models.CharField(max_length=50)
    command = models.TextField() # 执行命令
    mode = models.CharField(max_length=10, choices=MODE_CHOICES)
    changelog = models.TextField(blank=True, null=True)
    created_at = models.DateTimeField(auto_now_add=True)
    updated_at = models.DateTimeField(auto_now=True)


class Version(models.Model):
    """
    版本类，表示一个代码项目的版本。
    Attributes:
        project (Project): 关联的项目。
        version_hash (str): 版本号。
        release_date (datetime): 发布时间，自动生成。
        changelog (str): 更新日志。
        file (FileField): 版本文件。
    """
    project = models.ForeignKey(Project, on_delete=models.CASCADE, related_name="versions")
    version_hash = models.CharField(max_length=50, unique=True)
    release_date = models.DateTimeField(auto_now_add=True)
    entrance_file = models.CharField(max_length=100, null=True, blank=True)  # 入口文件
    create_at = models.DateTimeField(auto_now_add=True)  # 创建时间
    update_at = models.DateTimeField(auto_now=True)  # 更新时间

    class Meta:
        unique_together = ('project', 'version_hash')  # 确保同一个项目下版本号唯一
        ordering = ['-release_date']  # 默认按发布时间倒序排列

    def __str__(self):
        return f"{self.project.name} - {self.version_hash}"


class NodeProjectVersion(models.Model):
    """ 记录某个 Node 上存储的 Project 及其对应的 Version，关联表 """
    node = models.ForeignKey(Node, on_delete=models.CASCADE, related_name="node_versions")
    project = models.ForeignKey('Araneae_repo.Project', on_delete=models.CASCADE, related_name="node_projects")
    version = models.ForeignKey(Version, on_delete=models.CASCADE, related_name="node_versions")
    deployed_at = models.DateTimeField(auto_now_add=True)  # 记录部署时间
    is_active = models.BooleanField(default=True)  # 标记当前是否是激活的版本

    def __str__(self):
        return f"{self.project.id} - {self.version.id} on {self.node.name}"