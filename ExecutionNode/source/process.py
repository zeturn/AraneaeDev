# -*- coding: utf-8 -*-
# araneae_worknode - process.py
# Created by zhr62 at 2025/2/13 - 下午5:10

from models import db, Project


def create_project(workplace_id, name, language, command, description=None, mode='manual'):
    """
    创建一个新的 Project 记录并保存到数据库
    :param workplace_id: 工作区 ID（必填）
    :param name: 项目名称（必填）
    :param language: 编程语言（必填）
    :param command: 启动命令（必填）
    :param description: 项目描述（可选）
    :param mode: 运行模式（默认 'manual'）
    :return: 创建成功返回 Project 对象，失败返回 None
    """
    try:
        new_project = Project(
            workplace_id=workplace_id,
            name=name,
            description=description,
            language=language,
            command=command,
            mode=mode
        )
        db.session.add(new_project)
        db.session.commit()
        return new_project  # 返回创建的对象
    except Exception as e:
        db.session.rollback()
        print(f"Error creating project: {e}")
        return None  # 失败时返回 None
