#!/usr/bin/env bash
# 关服：按 pid 文件优雅停止，超时后强制结束。
set -euo pipefail

APP_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
APP="$(basename "$APP_DIR")"
PID_FILE="$APP_DIR/$APP.pid"

if [[ ! -f "$PID_FILE" ]]; then
    echo "[关服] $APP 未运行（无 pid 文件）"
    exit 0
fi

pid="$(cat "$PID_FILE")"
if ! kill -0 "$pid" 2>/dev/null; then
    rm -f "$PID_FILE"
    echo "[关服] $APP 进程不存在，已清理 pid 文件"
    exit 0
fi

kill "$pid"
for _ in $(seq 1 10); do
    if ! kill -0 "$pid" 2>/dev/null; then
        rm -f "$PID_FILE"
        echo "[关服] $APP 已停止"
        exit 0
    fi
    sleep 1
done

kill -9 "$pid" 2>/dev/null || true
rm -f "$PID_FILE"
echo "[关服] $APP 未在 10s 内退出，已强制结束"
