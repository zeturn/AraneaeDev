"""
Araneae_manager/apps.py
"""
from django.apps import AppConfig


class AraneaeManagerConfig(AppConfig):
    default_auto_field = 'django.db.models.BigAutoField'
    name = 'Araneae_manager'

    def ready(self):
        """
        Django 启动完成后自动注册心跳定时任务。
        使用 django-celery-beat 的 IntervalSchedule + PeriodicTask，
        若已存在则跳过，避免重复创建。
        """
        self._register_heartbeat_task()

    @staticmethod
    def _register_heartbeat_task():
        try:
            from django_celery_beat.models import IntervalSchedule, PeriodicTask
            import json

            # 每 60 秒执行一次
            schedule, _ = IntervalSchedule.objects.get_or_create(
                every=60,
                period=IntervalSchedule.SECONDS,
            )

            PeriodicTask.objects.get_or_create(
                name='heartbeat_all_nodes',
                defaults={
                    'task': 'heartbeat_all_nodes',
                    'interval': schedule,
                    'args': json.dumps([]),
                    'kwargs': json.dumps({}),
                    'enabled': True,
                    'description': '每 60 秒探测所有启用节点的 /health 接口，更新 Node.status',
                },
            )
        except Exception:
            # 首次 migrate 前数据库表可能尚不存在，静默跳过
            pass
