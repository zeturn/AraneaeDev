import socket
import platform
import psutil
import time
from datetime import datetime
import pytz

from machine.network import get_internal_ips, get_public_ip


def get_system_info():
    # 获取主机名和内网 IP
    hostname = socket.gethostname()
    internal_ips = get_internal_ips()

    # 获取公网 IP 和地理位置信息
    public_ip, geo_info = get_public_ip()

    # 获取时区
    local_time = time.time()
    time_zone = time.tzname

    # 获取系统信息
    system_info = {
        "hostname": hostname,
        "internal_ips": internal_ips,
        "public_ip": public_ip,
        "geo_location": geo_info,
        "system": platform.system(),
        "node": platform.node(),
        "release": platform.release(),
        "version": platform.version(),
        "machine": platform.machine(),
        "processor": platform.processor(),
        "architecture": platform.architecture(),
    }

    # 获取 CPU 信息
    cpu_info = {
        "cpu_count": psutil.cpu_count(logical=True),
        "cpu_physical_cores": psutil.cpu_count(logical=False),
        "cpu_frequency": psutil.cpu_freq()._asdict() if psutil.cpu_freq() else {},
    }

    # 获取内存信息
    memory = psutil.virtual_memory()
    memory_info = {
        "total_memory": memory.total,
        "available_memory": memory.available,
        "used_memory": memory.used,
        "memory_percentage": memory.percent,
    }

    # 获取磁盘信息
    disk = psutil.disk_usage('/')
    disk_info = {
        "total_disk": disk.total,
        "used_disk": disk.used,
        "free_disk": disk.free,
        "disk_percentage": disk.percent,
    }

    # 汇总信息
    info = {
        "system_info": system_info,
        "cpu_info": cpu_info,
        "memory_info": memory_info,
        "disk_info": disk_info,
        "timezone": {
            "name": time_zone,
            "local_time": datetime.now(pytz.timezone("UTC")).isoformat()
        },
    }

    return info