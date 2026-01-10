"""
Araneae_manager/urls.py
"""
from django.contrib import admin
from django.urls import path, include
from rest_framework.routers import DefaultRouter
from .views import NodeViewSet, SourceDistributeViewSet, TaskViewSet

router = DefaultRouter()
router.register(r'tasks', TaskViewSet, basename='tasks')
router.register(r'nodes', NodeViewSet)

urlpatterns = [
    path('admin/', admin.site.urls),
    path('api/', include(router.urls)),
]
