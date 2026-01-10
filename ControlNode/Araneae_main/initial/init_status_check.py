# -*- coding: utf-8 -*-
import sys

#  Copyright (c)   2025.4  Henry Zhao. All rights reserved.
#  From BJ.

# araneae_ControlNode - init_status_check.py.py
# Created by zhr62 at 2025/4/6 - 12:12
from django.db import connection
from django.db.migrations.recorder import MigrationRecorder
from django.db.utils import OperationalError
from django.db import connections, DEFAULT_DB_ALIAS
from django.contrib.auth import get_user_model

from Araneae_main.initial.database import check_db_connection, create_tables
from Araneae_main.initial.user import create_initial_superuser


def is_initial_migrations():
    """
    If the database has no migrations applied, return True.
    :return: Boolean
    """
    try:
        recorder = MigrationRecorder(connection)
        applied = recorder.applied_migrations()
        return len(applied) == 0
    except OperationalError:
        # 表不存在，也算初始状态
        return True


def is_db_empty():
    """
    If the database is empty (no tables), return True.
    :return: Boolean
    """
    tables = connections[DEFAULT_DB_ALIAS].introspection.table_names()
    # 过滤掉系统表
    user_tables = [t for t in tables if not t.startswith('django_')]
    return len(user_tables) == 0


def is_initial_user():
    """
    If there are no superusers, return True.
    :return: Boolean
    """
    User = get_user_model()
    return not User.objects.filter(is_superuser=True).exists()


def is_initial_state():
    """
    check if the database is in initial state
    :return: Boolean
    """
    # 只要满足下面任意一种，就认为是“初始状态”
    return (
            is_initial_migrations() or
            is_db_empty() or
            is_initial_user()
    )
