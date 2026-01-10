#!/usr/bin/env python
"""Django's command-line utility for administrative tasks."""
import os
import sys
import signal
import multiprocessing
import subprocess
import time
import json
from django.db import OperationalError

'''
一个跨平台的爬虫部署平台，由管理端（Django Restful Framework）和执行端（Flask）组成，
前端对接VUE Araneae_front
管理端负责管理爬虫项目，执行端负责执行爬虫任务。两端之间使用RabbitMQ消息队列进行通信，使用Celery进行任务调度，使用Http协议进行文件传输。
管理端的：
Araneae_main包括Workplace、User、UserProfile等模型，面对对象，提供了对爬虫工作区的的管理功能，包括创建、编辑、删除、查看日志等。
Araneae_manager包括了Project、Task、Node等模型，面对业务核心逻辑，提供了对爬虫项目与任务的管理功能，包括创建、编辑、删除、启动、分发、停止、查看日志、回调接收等。
Araneae_repo，提供了对爬虫项目代码的分发和预处理。
执行端的：
Araneae_work包括Project、task等模型，负责接受任务并执行，执行后回调。
管理端会向执行端发送爬虫文件和执行命令，执行端会根据任务的要求执行爬虫任务，并将爬取的数据返回给管理端。
'''

# Celery 进程变量
celery_worker_process = None
celery_beat_process = None


def start_celery_worker():
    """
    启动 Celery Worker

    """
    global celery_worker_process
    retry_num = 0

    if celery_worker_process is None or celery_worker_process.poll() is not None:
        print("[INFO] 启动 Celery Worker...")
        multiprocessing.set_start_method('spawn', force=True)
        with open("celery_worker.log", "w") as f:
            celery_worker_process = subprocess.Popen(
                ["celery", "-A", "Araneae", "worker",
                 "-Q", "internal_channel",
                 "--loglevel=info", "--pool=solo"], # --pool=solo 解决了 Windows 下的权限问题
                stdout=f,
                stderr=f
            )
        sleep_time = 5
        time.sleep(sleep_time)
        if celery_worker_process.poll() is not None and retry_num < 3:
            print(f"[WARNING] Celery Worker 启动失败，等待 {sleep_time} 秒后重试")
            time.sleep(sleep_time)
            retry_num += 1
            start_celery_worker()
        else:
            print(f"[INFO] Celery Worker 已启动 {celery_worker_process.pid}")
    else:
        print("[WARNING] Celery Worker 已在运行")


def start_celery_beat():
    """
    启动 Celery Beat

    """
    global celery_beat_process
    retry_num = 0

    if celery_beat_process is None or celery_beat_process.poll() is not None:
        print("[INFO] 启动 Celery Beat...")
        multiprocessing.set_start_method('spawn', force=True)
        #celery_beat_process = subprocess.Popen(["celery", "-A", "Araneae", "beat", "--loglevel=info", "--pool=solo"])
        with open("celery_beat.log", "w") as f:
            celery_beat_process = subprocess.Popen(
                ["celery", "-A", "Araneae", "beat", "--loglevel=debug", "--scheduler", "django_celery_beat.schedulers:DatabaseScheduler"],
                stdout=f,
                stderr=f
            )
        sleep_time = 5
        time.sleep(sleep_time)
        if celery_beat_process.poll() is not None and retry_num < 3:
            print(f"[WARNING] Celery Beat 启动失败，等待 {sleep_time} 秒后重试")
            time.sleep(sleep_time)
            retry_num += 1
            start_celery_beat()
        else:
            print(f"[INFO] Celery Beat 已启动 {celery_beat_process.pid}")
    else:
        print("[WARNING] Celery Beat 已在运行")


def stop_celery():
    """
    停止 Celery Worker 和 Celery Beat

    """
    global celery_worker_process, celery_beat_process
    print("[INFO] 正在关闭 Celery...")
    if celery_worker_process:
        celery_worker_process.terminate()
        celery_worker_process.wait()
    if celery_beat_process:
        celery_beat_process.terminate()
        celery_beat_process.wait()


def signal_handler(sig, frame):
    """
    处理终止信号
    @param sig:
    @param frame:
    """
    print("[INFO] 收到终止信号，正在关闭Celery进程...")
    stop_celery()
    print("[INFO] Celery进程关闭...")
    print("[INFO] 本体进程关闭...")
    sys.exit(0)


def init_celery_beat_db():
    """
    执行python manage.py migrate django_celery_beat
    :return:
    """
    from django.db import OperationalError, ProgrammingError
    from django.core.management import call_command
    try:
        # 尝试迁移 celery-beat 所需的表结构
        call_command('migrate', 'django_celery_beat', interactive=False, verbosity=0)

        # 可选：初始化一个默认的 schedule，避免 scheduler 为 None
        from django_celery_beat.models import PeriodicTask, CrontabSchedule
        if not CrontabSchedule.objects.exists():
            CrontabSchedule.objects.create(minute='0', hour='0', day_of_week='0', day_of_month='0', month_of_year='0')

    except (OperationalError, ProgrammingError) as e:
        print(f"[INFO]Celery Beat 初始化跳过，数据库未准备好：{e}")

def start_node_status():
    """
    中文：应用就绪时，创建每5秒执行一次的 Celery Beat 任务
    English: On app ready, create a Celery Beat periodic task every 5 seconds
    """
    from django_celery_beat.models import PeriodicTask, IntervalSchedule
    try:
        schedule, _ = IntervalSchedule.objects.get_or_create(
            every=5,
            period=IntervalSchedule.SECONDS
        )
        PeriodicTask.objects.get_or_create(
            name='poll_all_nodes_status',
            defaults={
                'interval': schedule,
                'task': 'poll_all_nodes_status',
                'args': json.dumps([]),
                'enabled': True
            }
        )
    except OperationalError:
        # 数据表尚未创建（如 migrate 期间），忽略
        pass

if __name__ == "__main__":
    os.environ.setdefault("DJANGO_SETTINGS_MODULE", "Araneae.settings")
    print("HollowData Group")
    print("[INFO] Araneae Main Node is running...")

    import django
    django.setup()
    # 检查数据库连接
    from Araneae_main.initial.database import init_db, check_db_connection
    from Araneae_main.initial.user import create_initial_superuser
    from Araneae_main.initial.init_status_check import is_initial_state
    from Araneae_main.initial.identity import init_identity

    if not check_db_connection(retries=3, delay=1):
        print("[ERROR][START] ❌ 数据库连接失败，请检查数据库配置。")
        sys.exit(1)

    # 检查数据库初始状态
    if is_initial_state():
        print("[INFO][INIT] ⚠️ 数据库处于初始状态，正在创建表和初始用户...")
        init_db()
        init_celery_beat_db()
        create_initial_superuser()
        init_identity()
        print("[INFO][INIT] ⚠️ 数据库初始化完成。")
    else:
        print("[INFO][INIT] ⚠️ 数据库已初始化，无需再次初始化。")

    # 只在主进程执行 Celery（防止 `runserver` 自动重启时重复执行）
    if "RUN_MAIN" not in os.environ and multiprocessing.get_start_method() == "spawn":
        print("[INFO][START] ✨ 启动 Django 和 Celery...")
        signal.signal(signal.SIGINT, signal_handler)
        signal.signal(signal.SIGTERM, signal_handler)
        start_celery_worker() #如果管理端不执行任务，可以注释掉
        start_celery_beat()

        # 启动 工作节点状态轮询
        start_node_status()


    from django.core.management import execute_from_command_line

    try:
        execute_from_command_line(sys.argv)
    finally:
        stop_celery()
        print("[INFO] Django 进程已关闭")
