#!/usr/bin/env bash
# 关服：按 <服务器id>.pid 发送 SIGTERM 优雅停止，跟踪日志等待资源释放完成标记，之后再按需强杀。
#
# 流程：
#   1. 读取 pid 文件，记录 stdout 日志当前字节偏移（避免误匹配历史标记）
#   2. kill -15，实时输出该实例新增的停服日志
#   3. 等到日志出现 [shutdown] === DONE ===（资源释放完成）或进程自行退出
#   4. 标记出现后给进程少量时间自然退出；仍未退出或等待超时则 kill -9 兜底
#
# 环境变量：
#   SHUTDOWN_TIMEOUT  等待资源释放完成标记的秒数（默认 20）
#   EXIT_TIMEOUT      标记出现后等待进程自行退出的秒数（默认 5）
set -euo pipefail

APP_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
APP="$(basename "$APP_DIR")"
PID_FILE="$APP_DIR/$APP.pid"
LOG_FILE="$APP_DIR/log/stdout.log"
DONE_MARK="[shutdown] === DONE ==="
SHUTDOWN_TIMEOUT="${SHUTDOWN_TIMEOUT:-20}"
EXIT_TIMEOUT="${EXIT_TIMEOUT:-5}"

alive() { kill -0 "$1" 2>/dev/null; }

# 日志文件当前字节数；不存在时按 0 处理。
log_size() {
    if [[ -f "$LOG_FILE" ]]; then
        local size
        size="$(wc -c <"$LOG_FILE" 2>/dev/null | tr -d '[:space:]')"
        echo "${size:-0}"
    else
        echo 0
    fi
}

if [[ ! -f "$PID_FILE" ]]; then
    echo "[关服] $APP 未运行（无 pid 文件）"
    exit 0
fi

pid="$(cat "$PID_FILE")"
if [[ ! "$pid" =~ ^[0-9]+$ ]]; then
    rm -f "$PID_FILE"
    echo "[关服] $APP pid 文件内容非法（$pid），已清理"
    exit 1
fi
if ! alive "$pid"; then
    rm -f "$PID_FILE"
    echo "[关服] $APP 进程不存在，已清理 pid 文件"
    exit 0
fi

# 从 offset 之后增量读取日志并回显，命中完成标记时置 marked=1（只消费一次，不重复输出）。
offset="$(log_size)"
marked=0
drain_log() {
    [[ -f "$LOG_FILE" ]] || return 0
    local cur chunk
    cur="$(log_size)"
    (( cur > offset )) || return 0
    chunk="$(tail -c +$((offset + 1)) "$LOG_FILE")"
    offset="$cur"
    [[ -n "$chunk" ]] || return 0
    printf '%s\n' "$chunk"
    if grep -qF "$DONE_MARK" <<<"$chunk"; then
        marked=1
    fi
}

echo "[关服] $APP 发送 SIGTERM (pid=$pid)，等待资源释放完成标记…"
if ! kill -15 "$pid" 2>/dev/null; then
    rm -f "$PID_FILE"
    echo "[关服] $APP 信号发送失败（进程可能已退出），已清理 pid 文件"
    exit 0
fi

deadline=$((SECONDS + SHUTDOWN_TIMEOUT))
timed_out=0
while :; do
    drain_log
    (( marked )) && break
    if ! alive "$pid"; then
        drain_log
        break
    fi
    if (( SECONDS >= deadline )); then
        timed_out=1
        break
    fi
    sleep 0.5
done

if (( marked )); then
    echo "[关服] $APP 资源释放完成，等待进程自行退出…"
elif (( timed_out )); then
    echo "[关服] $APP 未在 ${SHUTDOWN_TIMEOUT}s 内输出完成标记"
else
    echo "[关服] $APP 进程已退出，未获取到完成标记"
fi

# 标记出现后还有极短收尾（日志落盘、连接关闭），给一段自然退出的时间。
deadline=$((SECONDS + EXIT_TIMEOUT))
while alive "$pid" && (( SECONDS < deadline )); do
    sleep 0.5
done

forced=0
if alive "$pid"; then
    echo "[关服] $APP 进程仍在，kill -9 强制结束 (pid=$pid)"
    kill -9 "$pid" 2>/dev/null || true
    forced=1
    for _ in 1 2 3 4 5; do
        alive "$pid" || break
        sleep 0.2
    done
fi

drain_log
rm -f "$PID_FILE"

if alive "$pid"; then
    echo "[关服] $APP 强制结束失败，请人工检查 (pid=$pid)"
    exit 1
fi

if (( marked )); then
    echo "[关服] $APP 已优雅停止"
elif (( forced )); then
    echo "[关服] $APP 已强制停止（未确认资源释放完成）"
else
    echo "[关服] $APP 已停止（未确认资源释放完成）"
fi
