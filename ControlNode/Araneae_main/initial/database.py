# -*- coding: utf-8 -*-
"""
Araneae_main - database.py
"""
#  Copyright (c)   2025.4  Henry Zhao. All rights reserved.
#  From BJ.

# araneae_ControlNode - database.py
# Created by zhr62 at 2025/4/6 - 11:56
import time
from django.db import connections, DEFAULT_DB_ALIAS, OperationalError
from django.core.management import call_command




def check_db_connection(retries=2, delay=2):
    """
    检查数据库连接是否可用，最多重试 retries 次，每次间隔 delay 秒。
    返回 True 表示连接成功，False 表示全部重试失败。
    """
    alias = DEFAULT_DB_ALIAS
    for attempt in range(1, retries + 1):
        try:
            # Django 2.0+ 推荐用 ensure_connection()
            connections[alias].ensure_connection()
            print(f"[SUCCESS][INIT][DB] ✅ 第 {attempt} 次尝试：连接成功")
            return True
        except OperationalError as e:
            print(f"[ERROR][INIT][DB] ❌ 第 {attempt} 次尝试失败：{e}，{delay} 秒后重试…")
            time.sleep(delay)
    print(f"🚨 [DB] {retries} 次重试后仍无法连接数据库。")
    return False

def create_tables():
    """
    调用 Django 的 migrate 来创建/更新所有表。
    """
    print("[INFO][INIT]📦 正在执行 migrate …")
    # migration
    call_command('makemigrations', interactive=False, verbosity=1)
    call_command('migrate', interactive=False, verbosity=1)
    print("[INFO][INIT]🎉 migrate 完成。")

def init_db(retries=5, delay=2):
    """
    先检查数据库连接，连接成功后再创建表。
    """
    if check_db_connection(retries=retries, delay=delay):
        create_tables()
    else:
        print("[ERROR][INIT][DB] ❌ 数据库连接失败，无法创建表。")
        # 也可以在这里 sys.exit(1) 强制退出
        pass
