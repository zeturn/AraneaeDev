#!/usr/bin/env bash
set -euo pipefail

cmd="${1:-up}"

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
RUNDIR="$ROOT/.araneae-dev"
LOGDIR="$RUNDIR/logs"

mkdir -p "$LOGDIR"

control_pid_file="$RUNDIR/controlnode.pid"
exec_pid_file="$RUNDIR/executionnode.pid"
front_pid_file="$RUNDIR/front.pid"

control_log="$LOGDIR/controlnode.log"
exec_log="$LOGDIR/executionnode.log"
front_log="$LOGDIR/front.log"

load_env_file() {
  local env_file="$ROOT/.env"
  if [[ ! -f "$env_file" ]]; then
    return 0
  fi

  while IFS= read -r line || [[ -n "$line" ]]; do
    line="${line#"${line%%[![:space:]]*}"}"
    line="${line%"${line##*[![:space:]]}"}"
    [[ -z "$line" || "$line" == \#* ]] && continue
    [[ "$line" != *=* ]] && continue

    local key="${line%%=*}"
    local val="${line#*=}"
    key="${key%"${key##*[![:space:]]}"}"
    key="${key#"${key%%[![:space:]]*}"}"
    val="${val#"${val%%[![:space:]]*}"}"
    val="${val%"${val##*[![:space:]]}"}"
    val="${val%\"}"
    val="${val#\"}"
    val="${val%\'}"
    val="${val#\'}"
    export "$key=$val"
  done <"$env_file"
}

ensure_runtime_paths() {
  local control_db="${DJANGO_DB_PATH:-}"
  local exec_db="${EXECUTION_DB_PATH:-}"
  local fallback_control="$RUNDIR/controlnode.sqlite3"
  local fallback_exec="$RUNDIR/executionnode.sqlite3"

  if [[ -n "$control_db" ]]; then
    local control_dir
    control_dir="$(dirname "$control_db")"
    if [[ ! -d "$control_dir" || ! -w "$control_dir" ]]; then
      export DJANGO_DB_PATH="$fallback_control"
    fi
  else
    export DJANGO_DB_PATH="$fallback_control"
  fi

  if [[ -n "$exec_db" ]]; then
    local exec_dir
    exec_dir="$(dirname "$exec_db")"
    if [[ ! -d "$exec_dir" || ! -w "$exec_dir" ]]; then
      export EXECUTION_DB_PATH="$fallback_exec"
    fi
  else
    export EXECUTION_DB_PATH="$fallback_exec"
  fi
}

is_pid_running() {
  local pid="$1"
  [[ -n "$pid" ]] && kill -0 "$pid" 2>/dev/null
}

pid_listens_on_port() {
  local pid="$1"
  local port="$2"
  [[ -z "$pid" ]] && return 1
  if command -v ss >/dev/null 2>&1; then
    ss -ltnpH "sport = :$port" 2>/dev/null | grep -Eq "pid=$pid(,|)"
    return $?
  fi
  if command -v lsof >/dev/null 2>&1; then
    lsof -tiTCP:"$port" -sTCP:LISTEN 2>/dev/null | head -n1 | grep -Eq "^$pid$"
    return $?
  fi
  return 1
}

read_pid() {
  local file="$1"
  [[ -f "$file" ]] && cat "$file" || true
}

pid_from_port() {
  local port="$1"
  if command -v ss >/dev/null 2>&1; then
    ss -ltnpH "sport = :$port" 2>/dev/null | sed -n 's/.*pid=\([0-9]\+\).*/\1/p' | head -n1 || true
    return 0
  fi

  if command -v lsof >/dev/null 2>&1; then
    lsof -tiTCP:"$port" -sTCP:LISTEN 2>/dev/null | head -n1 || true
    return 0
  fi
}

wait_for_port_pid() {
  local port="$1"
  local timeout_seconds="${2:-12}"
  local deadline=$((SECONDS + timeout_seconds))

  while (( SECONDS < deadline )); do
    local pid
    pid="$(pid_from_port "$port")"
    if [[ -n "$pid" ]] && is_pid_running "$pid"; then
      echo "$pid"
      return 0
    fi
    sleep 0.2
  done
  return 1
}

python_for_dir() {
  local dir="$1"
  if [[ -x "$dir/.venv/bin/python" ]]; then
    echo "$dir/.venv/bin/python"
    return 0
  fi
  if [[ -x "$ROOT/.venv/bin/python" ]]; then
    echo "$ROOT/.venv/bin/python"
    return 0
  fi
  if command -v python3 >/dev/null 2>&1; then
    echo "python3"
    return 0
  fi
  echo "python"
}

stop_port() {
  local name="$1"
  local port="$2"
  local pid_file="$3"

  local pid port_pid file_pid
  file_pid="$(read_pid "$pid_file")"
  port_pid="$(pid_from_port "$port")"
  pid="$port_pid"
  if [[ -z "$pid" && -n "$file_pid" ]] && is_pid_running "$file_pid" && pid_listens_on_port "$file_pid" "$port"; then
    pid="$file_pid"
  fi
  if [[ -z "$pid" ]]; then
    if [[ -n "$file_pid" ]]; then
      echo "Cleaning stale pid file for $name ($file_pid)"
    fi
    rm -f "$pid_file"
    return 0
  fi

  if is_pid_running "$pid"; then
    echo "Stopping $name on :$port (pid $pid)"
    kill "$pid" 2>/dev/null || true
    for _ in {1..25}; do
      if ! is_pid_running "$pid"; then
        break
      fi
      sleep 0.2
    done
    if is_pid_running "$pid"; then
      echo "Force killing $name (pid $pid)"
      kill -9 "$pid" 2>/dev/null || true
    fi
  fi

  rm -f "$pid_file"
}

adopt_or_running() {
  local name="$1"
  local port="$2"
  local pid_file="$3"

  local existing
  existing="$(read_pid "$pid_file")"
  if [[ -n "$existing" ]] && is_pid_running "$existing" && pid_listens_on_port "$existing" "$port"; then
    echo "$name already running (pid $existing)"
    return 0
  elif [[ -n "$existing" ]]; then
    echo "Ignoring stale pid in $pid_file: $existing"
    rm -f "$pid_file"
  fi

  local port_pid
  port_pid="$(pid_from_port "$port")"
  if [[ -n "$port_pid" ]] && is_pid_running "$port_pid"; then
    echo "$name already running on :$port (pid $port_pid)"
    echo "$port_pid" >"$pid_file"
    return 0
  fi
  return 1
}

start_controlnode() {
  adopt_or_running "controlnode" 8107 "$control_pid_file" && return 0

  local dir="$ROOT/ControlNode"
  local py
  py="$(python_for_dir "$dir")"
  echo "Starting controlnode on :8107"
  (
    cd "$dir"
    nohup "$py" manage.py runserver 0.0.0.0:8107 --noreload >"$control_log" 2>&1 &
    if listener_pid="$(wait_for_port_pid 8107 20)"; then
      echo "$listener_pid" >"$control_pid_file"
    else
      echo $! >"$control_pid_file"
    fi
  )
}

start_executionnode() {
  adopt_or_running "executionnode" 5001 "$exec_pid_file" && return 0

  local dir="$ROOT/ExecutionNode"
  local py
  py="$(python_for_dir "$dir")"
  echo "Starting executionnode on :5001"
  (
    cd "$dir"
    nohup env PYTHONPATH="$ROOT/ExecutionNode:${PYTHONPATH:-}" "$py" app.py >"$exec_log" 2>&1 &
    if listener_pid="$(wait_for_port_pid 5001 20)"; then
      echo "$listener_pid" >"$exec_pid_file"
    else
      echo $! >"$exec_pid_file"
    fi
  )
}

start_front() {
  adopt_or_running "front" 5109 "$front_pid_file" && return 0

  local dir="$ROOT/Front"
  echo "Starting front on :5109"
  (
    cd "$dir"
    nohup npm run dev -- --host 0.0.0.0 --port 5109 >"$front_log" 2>&1 &
    if listener_pid="$(wait_for_port_pid 5109 20)"; then
      echo "$listener_pid" >"$front_pid_file"
    else
      echo $! >"$front_pid_file"
    fi
  )
}

ensure_started() {
  local name="$1"
  local port="$2"
  local pid_file="$3"
  local log_file="$4"

  local pid
  pid="$(read_pid "$pid_file")"
  if [[ -n "$pid" ]] && is_pid_running "$pid" && pid_listens_on_port "$pid" "$port"; then
    return 0
  fi
  if [[ -n "$(pid_from_port "$port")" ]]; then
    return 0
  fi

  echo "$name failed to start."
  if [[ -f "$log_file" ]]; then
    echo "Recent $name log:"
    tail -n 20 "$log_file" || true
  fi
  return 1
}

print_proc_status() {
  local name="$1"
  local port="$2"
  local pid_file="$3"
  local pid
  pid="$(read_pid "$pid_file")"
  if [[ -z "$pid" ]] || ! is_pid_running "$pid" || ! pid_listens_on_port "$pid" "$port"; then
    pid="$(pid_from_port "$port")"
  fi
  printf "%-14s %s\n" "$name" "${pid:-<none>}"
  if [[ -n "${pid:-}" ]] && is_pid_running "$pid"; then
    echo "  - running on :$port"
  else
    echo "  - stopped"
  fi
}

status() {
  echo "Repo: $ROOT"
  echo "Run dir: $RUNDIR"
  echo
  print_proc_status "controlnode" 8107 "$control_pid_file"
  print_proc_status "executionnode" 5001 "$exec_pid_file"
  print_proc_status "front" 5109 "$front_pid_file"
  echo
  if command -v ss >/dev/null 2>&1; then
    ss -ltnp | grep -E ':(8107|5001|5109)\b' || true
  fi
}

logs() {
  echo "Tail logs (Ctrl+C to stop):"
  echo "- $control_log"
  echo "- $exec_log"
  echo "- $front_log"
  echo
  tail -n 80 -f "$control_log" "$exec_log" "$front_log"
}

probe_http() {
  local name="$1"
  local url="$2"
  local code=""
  local tries=10
  for _ in $(seq 1 "$tries"); do
    code="$(curl -sS -o /dev/null --max-time 3 -w "%{http_code}" "$url" || true)"
    if [[ -n "$code" && "$code" != "000" ]]; then
      echo "$name OK ($code)"
      return 0
    fi
    sleep 0.4
  done
  echo "$name FAIL"
  return 1
}

healthcheck() {
  local ok=0
  probe_http "controlnode" "http://127.0.0.1:8107/api/" || ok=1
  probe_http "executionnode" "http://127.0.0.1:5001/" || ok=1
  probe_http "front" "http://127.0.0.1:5109/" || ok=1
  return "$ok"
}

load_env_file
ensure_runtime_paths

case "$cmd" in
  up|start)
    start_controlnode
    start_executionnode
    start_front
    echo

    local_fail=0
    ensure_started "controlnode" 8107 "$control_pid_file" "$control_log" || local_fail=1
    ensure_started "executionnode" 5001 "$exec_pid_file" "$exec_log" || local_fail=1
    ensure_started "front" 5109 "$front_pid_file" "$front_log" || local_fail=1
    echo
    status
    echo
    echo "URLs:"
    echo "- ControlNode:   http://localhost:8107"
    echo "- ExecutionNode: http://localhost:5001"
    echo "- Front:         http://localhost:5109"
    echo
    echo "Logs: $LOGDIR (or run: scripts/dev.sh logs)"
    if command -v curl >/dev/null 2>&1; then
      echo
      echo "Health check:"
      healthcheck || true
    fi
    if [[ "$local_fail" -ne 0 ]]; then
      echo
      echo "One or more services did not start. Check dependency setup first."
      exit 1
    fi
    ;;
  down|stop)
    stop_port "front" 5109 "$front_pid_file"
    stop_port "executionnode" 5001 "$exec_pid_file"
    stop_port "controlnode" 8107 "$control_pid_file"
    echo "Stopped."
    ;;
  status)
    status
    ;;
  logs)
    logs
    ;;
  health)
    healthcheck
    ;;
  *)
    echo "Usage: scripts/dev.sh {up|down|status|logs|health}" >&2
    exit 2
    ;;
esac
