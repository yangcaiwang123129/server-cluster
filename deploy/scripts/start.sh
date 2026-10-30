#!/usr/bin/env bash
# 开服：在包目录内后台启动可执行文件（包目录名即服务器 id / 可执行文件名）。
# 跨平台：
#   - Linux/macOS: nohup &, $! 就是子进程，kill -0 可直接探测
#   - Windows Git Bash: nohup 包一层；用 tasklist 抓真正的 Windows PID，用 tasklist 探测存活
set -euo pipefail

APP_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
APP="$(basename "$APP_DIR")"
PID_FILE="$APP_DIR/$APP.pid"
LOG_DIR="$APP_DIR/log"

mkdir -p "$LOG_DIR" "$APP_DIR/pid"

is_windows() {
    case "$(uname -s)" in
        *MINGW*|*CYGWIN*|*MSYS*) return 0 ;;
        *) return 1 ;;
    esac
}

pid_alive() {
    local pid="$1"
    if is_windows; then
        # Git Bash kill -0 不认 Windows PID；用 tasklist 查
        tasklist 2>/dev/null | awk -v p="$pid" '$2==p {found=1; exit} END {exit !found}'
    else
        kill -0 "$pid" 2>/dev/null
    fi
}

if [[ -f "$PID_FILE" ]] && pid_alive "$(cat "$PID_FILE")"; then
    echo "[开服] $APP 已在运行 (pid=$(cat "$PID_FILE"))" >&2
    exit 0
fi

cd "$APP_DIR"

# 后台启动
nohup "./$APP" >>"$LOG_DIR/stdout.log" 2>&1 &

# 取真正的进程 PID
sleep 0.5
REAL_PID="$!"
if is_windows; then
    # Windows Git Bash: tasklist 列 ImageName PID ...，$1=进程名 $2=PID
    REAL_PID="$(tasklist 2>/dev/null | awk -v app="$APP" '$1==app {print $2; exit}')"
    if [[ -z "$REAL_PID" ]]; then
        sleep 1
        REAL_PID="$(tasklist 2>/dev/null | awk -v app="$APP" '$1==app {print $2; exit}')"
    fi
fi

if [[ -z "$REAL_PID" ]] || ! pid_alive "$REAL_PID"; then
    echo "[开服] ERROR: 进程启动后立即退出，$LOG_DIR/stdout.log 末尾：" >&2
    tail -n 20 "$LOG_DIR/stdout.log" >&2 || true
    exit 1
fi

echo "$REAL_PID" >"$PID_FILE"
echo "[开服] $APP 已启动 (pid=$REAL_PID)" >&2
