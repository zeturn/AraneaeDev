import pika
import json


def callback(ch, method, properties, body):
    """
    消费者回调函数，处理任务。
    """
    try:
        message = json.loads(body)
        task_name = message['task_name']
        payload = message['payload']

        # 模拟执行任务
        print(f"Executing task: {task_name} with payload: {payload}")

        # 手动确认消息已处理
        ch.basic_ack(delivery_tag=method.delivery_tag)
    except Exception as e:
        print(f"Error processing message: {e}")
        # 如果失败，不确认消息（可以选择重试或丢弃）


def start_consumer():
    """
    启动 RabbitMQ 消费者。
    """
    connection = pika.BlockingConnection(pika.ConnectionParameters(
        host='199.7.140.120',
        port=5673,
        credentials=pika.PlainCredentials('guest', '54321Ssdlh!!')
    ))
    channel = connection.channel()

    # 声明队列（需与生产者一致）
    channel.queue_declare(queue='task_queue', durable=True)

    # 设置回调函数
    channel.basic_consume(queue='task_queue', on_message_callback=callback)

    print("[HollowData]Waiting for messages. To exit press CTRL+C")
    channel.start_consuming()


if __name__ == '__main__':
    start_consumer()
