# -*- coding: utf-8 -*-

#  Copyright (c)   2025.5  Henry Zhao. All rights reserved.
#  From BJ.

# araneae_ControlNode - generate6uuid.py
# Created by zhr62 at 2025/5/24 - 12:24
import random
import string

def generate_6_uuid():
    """
    生成 6 位随机版本号（字母+数字）
    @return: str
    """
    return ''.join(random.choices(string.ascii_letters + string.digits, k=6))