# -*- coding: utf-8 -*-

#  Copyright (c)   2025.5  Henry Zhao. All rights reserved.
#  From BJ.

# araneae_ControlNode - cron_verify.py
# Created by zhr62 at 2025/5/22 - 20:35


"""
中文：cron表达式验证模块
en：Cron expression verification module
"""

import re
from datetime import datetime
from dateutil import parser
from dateutil.rrule import rrulestr
from dateutil.rrule import rrulestr

def is_valid_cron_expression(cron_expression):
    """
    验证cron表达式的有效性 TODO: 待用
    Validate the cron expression
    :param cron_expression: cron表达式
    :return: bool
    """
    # 使用正则表达式验证cron表达式的格式
    cron_regex = re.compile(r'^\s*([0-5]?\d|\*)\s+([01]?\d|2[0-3]|\*)\s+([1-9]|[12]\d|3[01]|\*)\s+([1-9]|1[0-2]|\*)\s+([0-6]|\*)\s*$')
    return bool(cron_regex.match(cron_expression))
