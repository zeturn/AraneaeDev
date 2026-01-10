# -*- coding: utf-8 -*-

#  Copyright (c)   2025.4  Henry Zhao. All rights reserved.
#  From BJ.

# araneae_ControlNode - serializers.py.py
# Created by zhr62 at 2025/4/5 - 18:01

from rest_framework import serializers

from utils.uuid.generate6uuid import generate_6_uuid
from .models import Project, Version, NodeProjectVersion

class NodeProjectVersionSerializer(serializers.ModelSerializer):
    version_hash = serializers.SerializerMethodField()
    project_name = serializers.SerializerMethodField()

    class Meta:
        """NodeProjectVersion serializer"""
        model = NodeProjectVersion
        fields = '__all__'  # Include all fields from the model

    def get_version_hash(self, obj):
        """Get the version hash from the related Version model"""
        return obj.version.version_hash

    def get_project_name(self, obj):
        """Get the project name from the related Project model"""
        return obj.project.name

    def to_representation(self, instance):
        representation = super().to_representation(instance)
        representation['version_hash'] = self.get_version_hash(instance)
        representation['project_name'] = self.get_project_name(instance)
        return representation

class ProjectSerializer(serializers.ModelSerializer):
    class Meta:
        """Project serializer"""
        model = Project
        fields = '__all__'  # Include all fields from the model
        read_only_fields = ('id', 'project_hash')

    def create(self, validated_data):
        """
        Create a new Project instance with a unique project_hash.
        :param validated_data: The validated data for the project.
        :return: The created Project instance.
        """
        # Generate a unique project_hash
        project_hash = generate_6_uuid()
        validated_data['project_hash'] = project_hash
        return super().create(validated_data)

class VersionSerializer(serializers.ModelSerializer):
    class Meta:
        """Version serializer"""
        model = Version
        fields = '__all__'  # Include all fields from the model
        read_only_fields = ('id', 'version_hash')

    def create(self, validated_data):
        """
        Create a new Version instance with a unique version_hash.
        :param validated_data: The validated data for the version.
        :return: The created Version instance.
        """
        # Generate a unique version_hash
        version_hash = generate_6_uuid()
        validated_data['version_hash'] = version_hash
        return super().create(validated_data)