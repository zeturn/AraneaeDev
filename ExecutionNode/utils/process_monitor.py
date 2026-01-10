import psutil


def is_process_alive(pid):
    """
    Check if a process is alive by PID
    @param pid:
    @return:
    """
    try:
        process = psutil.Process(pid)
        return process.is_running()
    except psutil.NoSuchProcess:
        return False
