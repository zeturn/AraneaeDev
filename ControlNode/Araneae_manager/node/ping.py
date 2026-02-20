# -*- coding: utf-8 -*-
"""
#  Araneae_WorkNode - ping.py
"""
import platform
import subprocess
import socket

#  Copyright (c)   2025.3  Henry Zhao. All rights reserved.
#  From CA.

import requests
from django.conf import settings

def get_system_info(ip, port=5001):
    """
    访问目标服务器的 /system_info 接口，并返回整理后的信息。

    :param ip: 目标服务器 IP 地址
    :param port: 目标端口，默认为 5001
    :return: 整理后的系统信息字典，如果失败返回错误信息
    """
    url = f"http://{ip}:{port}/system_info"
    headers = {}
    if getattr(settings, "NODE_API_TOKEN", ""):
        headers["X-Araneae-Node-Token"] = settings.NODE_API_TOKEN

    try:
        response = requests.get(url, headers=headers, timeout=5)
        response.raise_for_status()
        data = response.json()
    except requests.exceptions.RequestException as e:
        return {"error": f"Failed to fetch system info: {e}"}

    # 整理返回的数据
    formatted_info = {
        "CPU": {
            "Physical Cores": data["cpu_info"].get("cpu_physical_cores"),
            "Total Cores": data["cpu_info"].get("cpu_count"),
            "Frequency (MHz)": data["cpu_info"]["cpu_frequency"].get("current"),
            "Processor": data["system_info"].get("processor"),
        },
        "Memory": {
            "Total Memory (GB)": round(data["memory_info"].get("total_memory", 0) / (1024 ** 3), 2),
            "Used Memory (GB)": round(data["memory_info"].get("used_memory", 0) / (1024 ** 3), 2),
            "Memory Usage (%)": data["memory_info"].get("memory_percentage"),
        },
        "Disk": {
            "Total Disk (GB)": round(data["disk_info"].get("total_disk", 0) / (1024 ** 3), 2),
            "Used Disk (GB)": round(data["disk_info"].get("used_disk", 0) / (1024 ** 3), 2),
            "Disk Usage (%)": data["disk_info"].get("disk_percentage"),
        },
        "System": {
            "OS": f"{data['system_info'].get('system')} {data['system_info'].get('version')}",
            "Hostname": data["system_info"].get("hostname"),
            "Architecture": data["system_info"].get("architecture"),
            "Machine": data["system_info"].get("machine"),
        },
        "Network": {
            "Public IP": data["system_info"].get("public_ip"),
            "Internal IPs": data["system_info"].get("internal_ips"),
            "Location": data["system_info"]["geo_location"].get("city"),
            "Region": data["system_info"]["geo_location"].get("region"),
            "Country": data["system_info"]["geo_location"].get("country"),
            "Organization": data["system_info"]["geo_location"].get("org"),
        },
        "Timezone": {
            "Local Time": data["timezone"].get("local_time"),
            "Timezone": data["timezone"].get("name"),
        }
    }

    return formatted_info


def ping(ip, count=3, timeout=1):
    """
    Ping 目标 IP 地址，返回是否可达。

    :param ip: 目标 IP 地址
    :param count: 发送的 ping 包数量，默认为 3
    :param timeout: 每个 ping 请求的超时时间（秒）
    :return: {"reachable": True/False, "details": 输出信息}
    """
    system = platform.system()

    if system == "Windows":
        cmd = ["ping", "-n", str(count), "-w", str(timeout * 1000), ip]
    else:
        cmd = ["ping", "-c", str(count), "-W", str(timeout), ip]

    try:
        result = subprocess.run(cmd, stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True)
        success = result.returncode == 0
        return {"reachable": success, "details": result.stdout}
    except FileNotFoundError:
        # 部分容器/最小系统没有 ping 命令，降级为业务端口连通性检查
        sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        try:
            sock.settimeout(float(timeout))
            ok = sock.connect_ex((ip, 5001)) == 0
            if ok:
                return {"reachable": True, "details": "ping unavailable; tcp connect to 5001 ok"}
            return {"reachable": False, "details": "ping unavailable; tcp connect to 5001 failed"}
        finally:
            sock.close()
    except Exception as e:
        return {"error": f"Failed to execute ping: {e}"}
