"""
Araneae_manager/serializers.py
"""
# -*- coding: utf-8 -*-
#  Copyright (c) 2024 INIT  2025.4 UPDATE  Henry Zhao. All rights reserved.
#  From BJ.

from rest_framework import serializers

from Araneae_repo.models import NodeProjectVersion
from .models import Node, TaskRecord, Task, Schedule


class NodeSerializer(serializers.ModelSerializer):
    class Meta:
        """Node serializer"""
        model = Node
        fields = '__all__'

class ScheduleSerializer(serializers.ModelSerializer):
    class Meta:
        """Schedule serializer"""
        model = Schedule

        celery_label = serializers.CharField(read_only=True)

        fields = [
            "id",
            "name",
            "description",
            "enabled",
            "order",
            "workplace",
            "created_at",
            "updated_at",
        ]
        extra_kwargs = {}

class TaskSerializer(serializers.ModelSerializer):
    class Meta:
        """Task serializer"""
        model = Task
        fields = '__all__'

class TaskRecordSerializer(serializers.ModelSerializer):
    class Meta:
        """TaskRecord serializer"""
        model = TaskRecord
        fields = '__all__'  # Include all fields from the model

class NodeStatusSerializer(serializers.Serializer):
    cpu_percent = serializers.FloatField()
    memory_used = serializers.IntegerField()
    memory_total = serializers.IntegerField()