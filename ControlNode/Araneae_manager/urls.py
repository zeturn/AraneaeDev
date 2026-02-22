"""
Araneae_manager/urls.py
"""
from django.urls import path, include
from rest_framework.routers import DefaultRouter
from .views import NodeViewSet, SourceDistributeViewSet, TaskViewSet, TaskCallbackViewSet

router = DefaultRouter()
router.register(r'tasks', TaskViewSet, basename='tasks')
router.register(r'nodes', NodeViewSet)
router.register(r'task', TaskCallbackViewSet, basename='task')

urlpatterns = [
    path('api/', include(router.urls)),
]
