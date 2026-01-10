# -*- coding: utf-8 -*-
# Araneae_ExecutionNode - database.py
# Created by zhr62 at 2025/5/17 - 12:42
# TODO: CHECK
from datetime import time
from sqlite3 import OperationalError


def check_db_connection(retries=2, delay=2):
    """
    检查flask_sqlalchemy数据库连接是否可用，最多重试 retries 次，每次间隔 delay 秒。
    检查数据库连接是否可用，最多重试 retries 次，每次间隔 delay 秒。
    返回 True 表示连接成功，False 表示全部重试失败。
    """

    from flask_sqlalchemy import SQLAlchemy
    from config import create_app

    app = create_app()
    db = SQLAlchemy(app)

    for attempt in range(1, retries + 1):
        try:
            db.engine.connect()
            print(f"[SUCCESS][INIT][DB] ✅ 第 {attempt} 次尝试：连接成功")
            return True
        except OperationalError as e:
            print(f"[ERROR][INIT][DB] ❌ 第 {attempt} 次尝试失败：{e}，{delay} 秒后重试…")
            time.sleep(delay)
    print(f"🚨 [DB] {retries} 次重试后仍无法连接数据库。")
    return False

def create_tables():
    """
    调用 Flask-SQLAlchemy 的 create_all 来创建/更新所有表。
    """
    from flask_sqlalchemy import SQLAlchemy
    from config import create_app

    app = create_app()
    db = SQLAlchemy(app)

    with app.app_context():
        db.create_all()
        print("[INFO][INIT]🎉 数据库表创建完成。")
