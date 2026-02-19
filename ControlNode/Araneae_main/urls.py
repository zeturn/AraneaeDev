"""
#  Araneae_main/urls.py
"""
#  Copyright (c)   2024.12  Henry Zhao. All rights reserved.
#  From BJ.

from django.contrib import admin
from django.urls import path, include
from rest_framework.routers import DefaultRouter

from . import views
from .oauth_views import basaltpass_oauth_login, basaltpass_oauth_callback

router = DefaultRouter()

router.register(r'workplaces', views.WorkplaceViewSet)
router.register(r'projects', views.ProjectViewSet)
router.register(r'schedules', views.ScheduleViewSet)
router.register(r'users', views.UserViewSet)
router.register(r'profile', views.ProfileViewSet, basename='profile')
router.register(r'teams', views.TeamViewSet)

urlpatterns = [
    path('api/', include(router.urls)),
    path('api/auth/basaltpass/login/', basaltpass_oauth_login, name='basaltpass_oauth_login'),
    path('api/auth/basaltpass/callback/', basaltpass_oauth_callback, name='basaltpass_oauth_callback'),
    path('webrtc/session/', views.webrtc_session, name='webrtc_session'),
]
