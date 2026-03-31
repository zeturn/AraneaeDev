#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR=$(cd "$(dirname "$0")/.." && pwd)
cd "$ROOT_DIR"

probe_port() {
  if command -v nc >/dev/null 2>&1; then
    nc -z localhost 5672
    return $?
  fi
  timeout 1 bash -lc '</dev/tcp/localhost/5672' >/dev/null 2>&1
}

if ! command -v jq >/dev/null 2>&1; then
  echo "jq is required"
  exit 1
fi
if ! command -v zip >/dev/null 2>&1; then
  echo "zip is required"
  exit 1
fi

cleanup() {
  if [[ -n "${CONTROL_PID:-}" ]]; then kill "$CONTROL_PID" >/dev/null 2>&1 || true; fi
  if [[ -n "${EXECUTOR_PID:-}" ]]; then kill "$EXECUTOR_PID" >/dev/null 2>&1 || true; fi
}
trap cleanup EXIT

for port in 8180 9190 4280; do
  if command -v lsof >/dev/null 2>&1; then
    PIDS=$(lsof -ti tcp:"$port" || true)
    if [[ -n "$PIDS" ]]; then
      kill $PIDS >/dev/null 2>&1 || true
      sleep 1
    fi
  fi
done

if command -v docker >/dev/null 2>&1; then
  docker compose up -d rabbitmq
else
  if ! probe_port; then
    echo "docker not found and rabbitmq:5672 is unavailable"
    echo "please start RabbitMQ manually, then rerun this script"
    exit 1
  fi
fi

go run ./cmd/control >/tmp/araneae-go-control.log 2>&1 &
CONTROL_PID=$!
go run ./cmd/executor >/tmp/araneae-go-executor.log 2>&1 &
EXECUTOR_PID=$!

echo "waiting for services..."
for _ in $(seq 1 60); do
  if curl -sf http://localhost:8180/healthz >/dev/null && curl -sf http://localhost:4280/healthz >/dev/null; then
    break
  fi
  sleep 1
done

TOKEN=$(curl -s http://localhost:8180/api/v1/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"username":"admin","password":"admin123"}' | jq -r '.token')

if [[ -z "$TOKEN" || "$TOKEN" == "null" ]]; then
  echo "login failed"
  exit 1
fi

PROJECT_ID=$(curl -s http://localhost:8180/api/v1/projects \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"name":"demo-project"}' | jq -r '.id')

TMP_ZIP=$(mktemp /tmp/araneae-demo-XXXXXX.zip)
rm -f "$TMP_ZIP"
(
  cd "$ROOT_DIR/examples/simple-job"
  chmod +x run.sh
  zip -qr "$TMP_ZIP" .
)

VERSION_ID=$(curl -s -X POST "http://localhost:8180/api/v1/projects/$PROJECT_ID/upload" \
  -H "Authorization: Bearer $TOKEN" \
  -F "file=@$TMP_ZIP" | jq -r '.id')

TASK_ID=$(curl -s http://localhost:8180/api/v1/tasks \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d "{\"name\":\"demo-schedule\",\"project_id\":\"$PROJECT_ID\",\"version_id\":\"$VERSION_ID\",\"entry_command\":\"bash run.sh\",\"cron_expr\":\"*/15 * * * * *\",\"node_queue\":\"default\"}" | jq -r '.id')

echo "project=$PROJECT_ID"
echo "version=$VERSION_ID"
echo "task=$TASK_ID"
echo "waiting 20s for schedule trigger"
sleep 20

echo "task runs:"
curl -s "http://localhost:8180/api/v1/tasks/$TASK_ID/runs" \
  -H "Authorization: Bearer $TOKEN" | jq .

echo "done"
