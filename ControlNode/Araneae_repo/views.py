import os

from django.shortcuts import render
from django.http import JsonResponse
from django.shortcuts import get_object_or_404
from Araneae_repo.models import Project


# Create your views here.


def check_project_code_files(project_id):
    """

    @param project_id:
    @return:
    """
    project_path = os.path.join('./Araneae_repo/repo/', str(project_id))
    if not os.path.exists(project_path):
        return False
    for root, dirs, files in os.walk(project_path):
        for file in files:
            if file.endswith(('.py', '.java', '.js', '.html', '.css', '.go', '.cpp', '.cs', '.rb', '.php', '.txt')):
                return True
    return False
