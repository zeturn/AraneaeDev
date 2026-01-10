"""
URL configuration for Araneae project.

The `urlpatterns` list routes URLs to views. For more information please see:
    https://docs.djangoproject.com/en/5.0/topics/http/urls/
Examples:
Function views
    1. Add an import:  from my_app import views
    2. Add a URL to urlpatterns:  path('', views.home, name='home')
Class-based views
    1. Add an import:  from other_app.views import Home
    2. Add a URL to urlpatterns:  path('', Home.as_view(), name='home')
Including another URLconf
    1. Import the include() function: from django.urls import include, path
    2. Add a URL to urlpatterns:  path('blog/', include('blog.urls'))
"""

from django.contrib import admin
from django.urls import path, include
from drf_spectacular.views import SpectacularRedocView, SpectacularSwaggerView, SpectacularAPIView
from rest_framework.routers import DefaultRouter
from rest_framework_simplejwt.views import TokenObtainPairView, TokenRefreshView
from django.conf import settings
from django.conf.urls.static import static

from Araneae_main.views import csrf_token_view, LogoutView, FileUploadViewSet
from Araneae_manager.views import SourceDistributeViewSet, TaskCallbackViewSet, TaskChainCreateView
from Araneae_manager import views

router = DefaultRouter()
urlpatterns = [
    path('admin/', admin.site.urls),
    path('api/token/', TokenObtainPairView.as_view(), name='token_obtain_pair'),
    path('api/token/refresh/', TokenRefreshView.as_view(), name='token_refresh'),
    path('api/csrf-token/', csrf_token_view),
    path('logout/', LogoutView.as_view(), name='logout'),

    # OpenAPI schema 生成
    path('api/schema/', SpectacularAPIView.as_view(), name='schema'),
    # Swagger UI
    path('api/doc/', SpectacularSwaggerView.as_view(url_name='schema'), name='swagger-ui'),
    # Redoc UI
    path('api/redoc/', SpectacularRedocView.as_view(url_name='schema'), name='redoc'),
    # JSON 格式
    path('api/schema/', SpectacularAPIView.as_view(), name='schema-json'),
    # YAML 格式
    path('api/schema/download/', SpectacularAPIView.as_view(), name='schema-yaml'),

    path('', include('Araneae_main.urls')),
    path('', include('Araneae_manager.urls')),
    path('', include('Araneae_repo.urls')),

    path('api/project/<project_id>/version/<version>/download/<file_name>/', views.download_file, name='download-file'),
    path('api/upload-script/', FileUploadViewSet.upload_script, name='upload-script'),
    path('api/distribute_source/', SourceDistributeViewSet.as_view({'post': 'distribute_source'}), name='distribute-source'),
    path('api/distribute_source/project/<int:project_id>/', SourceDistributeViewSet.as_view({'get': 'source_distribution_list'}), name='source_distribution_list'),
    path('api/task/callback/', TaskCallbackViewSet.as_view({'post': 'task_callback'}), name='task_callback'),

    path('api/create-task-chain/', TaskChainCreateView.as_view({'post': 'create'}), name='create-task-chain'),
] + static(settings.MEDIA_URL, document_root=settings.MEDIA_ROOT)
