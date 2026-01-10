# -*- coding: utf-8 -*-
"""
#  Araneae_WorkNode - setting.py
"""
#  Copyright (c)   2025.2  Henry Zhao. All rights reserved.
#  From BJ.

# araneae_main - signals.py.py
# Created by zhr62 at 2025/2/16 - 下午5:04


import os
import signal
from django.core.signals import request_finished
from django.dispatch import receiver


@receiver(request_finished)
def stop_celery(sender, **kwargs):
    """
    停止 Celery worker 和 beat
    @param sender:
    @param kwargs:
    """
    print("Stopping Celery worker and beat...")
    os.system("pkill -f 'celery worker'")
    os.system("pkill -f 'celery beat'")
