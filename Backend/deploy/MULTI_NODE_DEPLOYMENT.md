# Multi Node Deployment (1 Control + N Executors)

This guide describes deployment where one control node manages multiple executor nodes across different servers.

## 1. Topology

- Control node server:
  - HTTP API for frontend and callbacks (default port 8180)
  - gRPC artifact service for executors (default port 9190)
- RabbitMQ server:
  - Shared broker for control and executors (default port 5672)
- Executor node servers (one or many):
  - Consume tasks from RabbitMQ
  - Pull artifact from control gRPC
  - Callback run result to control HTTP

## 2. Build binaries

Run on your build host:

- go build -o bin/araneae-control ./cmd/control
- go build -o bin/araneae-executor ./cmd/executor

Copy bin/araneae-control and bin/araneae-executor to deployment hosts.

## 2.1 Docker Compose quick path (recommended for this repo)

If you deploy directly from this repository with Docker:

- Control machine:
  - Copy `.env.control.example` to `.env.control` and set real addresses.
  - Run:
    - `docker compose -f docker-compose.control.yml --env-file .env.control up -d --build`

- Executor machine:
  - Copy `.env.executor.example` to `.env.executor` and set control/rabbit addresses.
  - Run:
    - `docker compose -f docker-compose.executor.yml --env-file .env.executor up -d --build`

This model keeps control side and executor side independently scalable.

## 3. Configure control node

Use template:
- deploy/control.env.example

Key points:
- CONTROL_GRPC_ADDR should listen on an address reachable by executors.
- EXECUTION_CALLBACK_KEY must be shared with all executors.
- RABBITMQ_URL should point to shared broker.

## 4. Configure each executor node

Use template:
- deploy/executor.env.example

Key points:
- CONTROL_GRPC_TARGET and CONTROL_HTTP_BASE must point to control server reachable addresses.
- EXECUTION_CALLBACK_KEY must equal control side value.
- EXECUTOR_QUEUE decides routing behavior:
  - Dedicated routing: each executor uses unique queue (node-a, node-b, ...)
  - Load balancing: multiple executors share same queue name

## 5. Route tasks to executors

When creating task in control API:
- node_queue field should match target queue.

Examples:
- node_queue=node-a routes to executors using EXECUTOR_QUEUE=node-a
- node_queue=default routes to executors using EXECUTOR_QUEUE=default

## 6. Systemd deployment

Systemd unit templates:
- deploy/systemd/araneae-control.service
- deploy/systemd/araneae-executor.service

Typical steps on each host:

- create user/group araneae
- place app under /opt/araneae/GoRefactor
- place env file under /etc/araneae/
- place unit file under /etc/systemd/system/
- systemctl daemon-reload
- systemctl enable --now araneae-control (or araneae-executor)

## 7. Network and security baseline

- Open control HTTP (8180) for frontend and executor callbacks.
- Open control gRPC (9190) only for executor source ranges.
- RabbitMQ (5672) only for control/executor source ranges.
- Keep callback key and JWT secret long and random.
- Prefer private network/VPN between nodes.

## 8. Health checks

- Control: GET /healthz
- Executor: GET /healthz

## 9. Current limitation

Current implementation routes by queue name and supports multi-executor deployment,
but does not yet include built-in executor registration, heartbeat, and online node inventory.
If needed, add a control-side node registry API as the next step.
