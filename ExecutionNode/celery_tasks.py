from celery import Celery
import subprocess
import setting

app = Celery('tasks', broker='pyamqp://{username}:{password}@{host}:{port}/{vhost}'.format(
    username=setting.RABBITMQ['USERNAME'],
    password=setting.RABBITMQ['PASSWORD'],
    host=setting.RABBITMQ['HOST'],
    port=setting.RABBITMQ['PORT'],
    vhost=setting.RABBITMQ['VHOST']
))


@app.task
def run_crawler(project_path, script_name):
    """

    @param project_path:
    @param script_name:
    @return:
    """
    result = subprocess.run(
        ['python', f'{project_path}/{script_name}'],
        capture_output=True,
        text=True
    )
    return result.stdout
