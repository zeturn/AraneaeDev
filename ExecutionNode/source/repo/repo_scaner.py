# -*- coding: utf-8 -*-
# Araneae_ExecutionNode - repo_scaner.py
# Created by zhr62 at 2025/3/21 - 22:55
import os

BASE_DIR = os.path.dirname(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))  # 获取上一级目录
REPO_PATH = os.path.join(BASE_DIR, 'repo')
def get_project_versions(project_id=None):
    """
    获取所有项目及其版本，或特定项目的版本
    """
    print(f"Scanning repository at: {REPO_PATH}")
    if not os.path.exists(REPO_PATH):
        return REPO_PATH if project_id is None else []

    if project_id:
        project_path = os.path.join(REPO_PATH, project_id)
        if os.path.isdir(project_path) and project_id.isdigit():
            versions = [v for v in os.listdir(project_path) if len(v) == 6 and os.path.isdir(os.path.join(project_path, v))]
            return versions
        return []

    projects = {}
    for project in os.listdir(REPO_PATH):
        project_path = os.path.join(REPO_PATH, project)
        if os.path.isdir(project_path) and project.isdigit():
            versions = [v for v in os.listdir(project_path) if len(v) == 6 and os.path.isdir(os.path.join(project_path, v))]
            projects[project] = versions

    return projects
