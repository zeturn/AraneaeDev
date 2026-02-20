"""
Araneae_main/serializers.py
"""
#  Copyright (c)   2025.4  Henry Zhao. All rights reserved.
#  From BJ.

from rest_framework import serializers
from drf_spectacular.utils import extend_schema_field

from utils.uuid.generate6uuid import generate_6_uuid
from .models import Workplace, Team, TeamMember
from django.contrib.auth.models import User

from .models import Profile


class UserSerializer(serializers.ModelSerializer):
    class Meta:
        """User serializer"""
        model = User
        fields = ['id', 'username', 'email']


class ProfileSerializer(serializers.ModelSerializer):
    user = UserSerializer(read_only=True)  # 嵌套UserSerializer

    class Meta:
        """Profile serializer"""
        model = Profile
        fields = ['avatar', 'user']

    def create(self, validated_data):
        # 将user从上下文中获取，并传递给Profile
        validated_data['user'] = self.context['request'].user
        return super().create(validated_data)

    def update(self, instance, validated_data):
        """
        个人资料更新
        :param instance:
        :param validated_data:
        :return:
        """
        # 保持user字段不变
        validated_data['user'] = instance.user
        return super().update(instance, validated_data)


class WorkplaceSerializer(serializers.ModelSerializer):
    class Meta:
        """Workplace serializer"""
        model = Workplace
        fields = ('id', 'name', 'description', 'status', 'workplace_hash','created_at', 'updated_at')
        read_only_fields = ('id', 'workplace_hash')

    def create(self, validated_data):
        request = self.context.get('request', None)
        if request is not None:

            # 创建 6 位随机 workplace_hash
            workplace_hash = generate_6_uuid()

            workplace = Workplace.objects.create(
                name = validated_data['name'],
                description = validated_data['description'],
                workplace_hash = workplace_hash,
                status = validated_data['status']
            )
            return workplace
        raise serializers.ValidationError("Request context not provided.")


class TeamSerializer(serializers.ModelSerializer):

    class Meta:
        """Team serializer"""
        model = Team
        fields = ['id', 'name', 'description', 'join_able', 'created_at', 'updated_at']
        read_only_fields = ['id', 'created_at', 'updated_at']

    def create(self, validated_data):
        """
        中文: 在创建时自动设置 created_by 字段
        English: Automatically set created_by on creation.
        """
        request = self.context.get('request')
        # 创建 Team 实例
        team = Team.objects.create(**validated_data)
        # 如果用户已认证，则加入 members
        if request and hasattr(request, 'user') and request.user.is_authenticated:
            team.members.add(request.user)
        # **务必返回 team 实例**
        return team


class TeamWithRoleSerializer(serializers.ModelSerializer):
    """
    Team 序列化器，附加当前请求用户在对应 Team 中的角色字段。
    """
    role = serializers.SerializerMethodField()

    class Meta:
        model = Team
        # 根据需要返回的字段自行调整
        fields = [
            'id', 'name', 'description', 'created_at', 'updated_at', 'role'
        ]

    @extend_schema_field(serializers.CharField(allow_null=True))
    def get_role(self, obj) -> str | None:
        """
        获取当前请求用户在此 Team 中的 role。
        如果用户不在 Team 中，则返回 None。
        """
        request = self.context.get('request')
        user = getattr(request, 'user', None)
        if not user or not user.is_authenticated:
            return None
        try:
            membership = TeamMember.objects.get(team=obj, user=user)
            return membership.role
        except TeamMember.DoesNotExist:
            return None

class TeamMemberSerializer(serializers.ModelSerializer):
    """
    中文: 序列化 TeamMember，用于输出 user 和 role
    English: Serializer for TeamMember, outputting user info and role.
    """
    user = UserSerializer(read_only=True)

    class Meta:
        model = TeamMember
        fields = ['user', 'role', 'joined_at']

class TeamWithMembersSerializer(serializers.ModelSerializer):
    """
    中文: 序列化 Team，并附加所有成员及其角色
    English: Serializer for Team including all members with roles.
    """
    members = TeamMemberSerializer(
        source='teammember_set',  # 使用模型默认的 related_name
        many=True,
        read_only=True
    )

    class Meta:
        model = Team
        fields = ['id', 'name', 'description', 'join_able', 'members']
