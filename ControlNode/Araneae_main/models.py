"""
Araneae_main/models.py
"""
#  Copyright (c)   2025.1  Henry Zhao. All rights reserved.
#  From CA.

from django.conf import settings
from django.contrib.auth import get_user_model
from django.db import models
from django.db.models.signals import post_save
from django.dispatch import receiver

User = get_user_model()  # respects AUTH_USER_MODEL

class Profile(models.Model):
    user = models.OneToOneField(User, on_delete=models.CASCADE)
    avatar = models.ImageField(upload_to='avatars/', blank=True, null=True)

    def __str__(self):
        return self.user.username

@receiver(post_save, sender=User)
def create_user_profile(sender, instance, created, **kwargs):
    if created:
        Profile.objects.create(user=instance)


class Team(models.Model):
    name = models.CharField(max_length=100, unique=True)
    description = models.TextField(blank=True, null=True)
    join_able = models.BooleanField(default=True)
    created_at = models.DateTimeField(auto_now_add=True)
    updated_at = models.DateTimeField(auto_now=True)

    # reference User model directly, through TeamMember
    members = models.ManyToManyField(
        User,
        through='TeamMember',
        related_name='teams'
    )

    def __str__(self):
        return self.name

class TeamMember(models.Model):
    class Role(models.TextChoices):
        OWNER = 'owner', 'Owner'
        ADMIN = 'admin', 'Admin'
        MEMBER = 'member', 'Member'

    team = models.ForeignKey(Team, on_delete=models.CASCADE)
    user = models.ForeignKey(User, on_delete=models.CASCADE)
    role = models.CharField(max_length=20, choices=Role.choices, default=Role.MEMBER)
    joined_at = models.DateTimeField(auto_now_add=True)

    class Meta:
        unique_together = ('team', 'user')


@receiver(post_save, sender=User)
def create_user_team(sender, instance, created, **kwargs):
    if created:
        team = Team.objects.create(
            name=f"{instance.username}'s Team",
            join_able=False,
        )
        TeamMember.objects.create(
            team=team,
            user=instance,
            role=TeamMember.Role.ADMIN,
        )


class Workplace(models.Model):
    STATUS_CHOICES = [
        ('active', 'Active'),
        ('inactive', 'Inactive'),
    ]

    workplace_hash = models.CharField(max_length=50, unique=True)
    name = models.CharField(max_length=100)
    description = models.TextField(null=True, blank=True)
    status = models.CharField(max_length=10, choices=STATUS_CHOICES, default='active')

    # correct Team M2M through the permission model
    teams = models.ManyToManyField(
        Team,
        through='WorkplaceTeamPermission',
        related_name='workplaces'
    )

    created_at = models.DateTimeField(auto_now_add=True)
    updated_at = models.DateTimeField(auto_now=True)

    def __str__(self):
        return self.name


class WorkplaceTeamPermission(models.Model):
    class Permission(models.TextChoices):
        OWNER  = 'owner',  'Owner'
        EDITOR = 'editor', 'Editor'
        VIEWER = 'viewer', 'Viewer'

    workplace = models.ForeignKey(Workplace, on_delete=models.CASCADE)
    team = models.ForeignKey(Team, on_delete=models.CASCADE)
    permission = models.CharField(max_length=20, choices=Permission.choices, default=Permission.VIEWER)
    assigned_at = models.DateTimeField(auto_now_add=True)

    class Meta:
        unique_together = ('workplace', 'team')
        verbose_name = 'Workplace–Team permission'
        verbose_name_plural = 'Workplace–Team permissions'


class WorkplaceUserPermission(models.Model):
    class Permission(models.TextChoices):
        OWNER  = 'owner',  'Owner'
        EDITOR = 'editor', 'Editor'
        VIEWER = 'viewer', 'Viewer'

    workplace   = models.ForeignKey(Workplace, on_delete=models.CASCADE)
    user        = models.ForeignKey(User, on_delete=models.CASCADE)
    permission  = models.CharField(max_length=20, choices=Permission.choices, default=Permission.VIEWER)
    assigned_at = models.DateTimeField(auto_now_add=True)

    class Meta:
        unique_together = ('workplace', 'user')
        verbose_name = 'Workplace–User permission'
        verbose_name_plural = 'Workplace–User permissions'
