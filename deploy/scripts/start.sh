#!/usr/bin/env bash
# 开服：在包目录内后台启动可执行文件（包目录名即服务器 id / 可执行文件名）。
set -euo pipefail

APP_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
APP="$(basename "$APP_DIR")"
PID_FILE="$APP_DIR/$APP.pid"
LOG_DIR="$APP_DIR/log"

mkdir -p "$LOG_DIR" "$APP_DIR/pid"

if [[ -f "$PID_FILE" ]] && kill -0 "$(cat "$PID_FILE")" 2>/dev/null; then
    echo "[开服] $APP 已在运行 (pid=$(cat "$PID_FILE"))"
    exit 0
fi

cd "$APP_DIR"
nohup "./$APP" >>"$LOG_DIR/stdout.log" 2>&1 &
echo $! >"$PID_FILE"
echo "[开服] $APP 已启动 (pid=$(cat "$PID_FILE"))"
