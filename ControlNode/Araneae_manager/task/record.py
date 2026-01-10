"""
Araneae_manager.task.record
"""
# -*- coding: utf-8 -*-

#  Copyright (c)   2025.3  Henry Zhao. All rights reserved.
#  From BJ.

# araneae_ControlNode - record.py
# Created by zhr62 at 2025/3/28 - 12:58

import os
import subprocess
import time

from Araneae_repo.models import Project, Version
from Araneae_manager.models import TaskRecord, Node


def create_task_record(node, project, version, task_status, task_result=None):
    """
    创建任务记录
    :param node_id: 节点ID
    :param project_id: 项目ID
    :param version_id: 版本ID
    :param task_status: 任务状态
    :param task_result: 任务结果
    :return: TaskRecord任务记录
    """

    task_record = TaskRecord(
        node = node,
        project = project,
        version = version,
        task_status = task_status,
        task_result = task_result,
        task_created_at = time.time(),
        task_updated_at = time.time()
    )
    task_record.save()
    return task_record