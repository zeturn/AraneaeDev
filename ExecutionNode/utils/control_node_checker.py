# -*- coding: utf-8 -*-
# Araneae_WorkerNode - control_node_checker.py
# Created by zhr62 at 2025/2/21 - 17:15

import os
import http

def check_control_node():
    """
    检查控制节点是否正常运行
    :return: True if control node is running, False otherwise
    """
    try:
        response = http.client.HTTPConnection('localhost', 8000, timeout=5)
        response.request('GET', '/')
        response.getresponse()
        return True
    except Exception as e:
        print(f"Control node is not running: {e}")
        return False
