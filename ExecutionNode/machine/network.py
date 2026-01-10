import psutil
import requests
import socket



def get_internal_ips():
    """获取所有内网 IP"""
    internal_ips = []
    for interface, addrs in psutil.net_if_addrs().items():
        for addr in addrs:
            if addr.family == socket.AF_INET:
                internal_ips.append(addr.address)
    return internal_ips


def get_public_ip():
    """获取公网 IP"""
    try:
        response = requests.get("https://ipinfo.io/json")
        if response.status_code == 200:
            data = response.json()
            return data.get("ip"), data
        else:
            return None, {}
    except Exception as e:
        return None, {}