"""
Araneae_manager/admin.py
"""
from django.contrib import admin

# Register your models here.

from .models import Schedule,Task, Node, TaskRecord, ChainedTask, TaskChain

admin.site.register(Schedule)
admin.site.register(Task)

admin.site.register(Node)
admin.site.register(TaskRecord)

admin.site.register(ChainedTask)
admin.site.register(TaskChain)
