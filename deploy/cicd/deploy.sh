#!/usr/bin/env bash
# 分发：把构建产物推送到目标机并重启。主机清单来自 deploy/cicd/<env>/servers.conf。
#
# 用法：
#   deploy/cicd/deploy.sh --env prod --version v1.2.3 --servers "gate-1 game-1"
#   deploy/cicd/deploy.sh --env prod --version v1.2.3            # 不传 --servers 表示该环境全部
#
# 可选：
#   --user <用户>      默认 game（可被 servers.conf 第 3 列覆盖）
#   --remote <目录>    默认 /data/game（可被 servers.conf 第 4 列覆盖）
#   --dry-run          只打印将执行的命令
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

ENV=""
VERSION=""
SERVERS=""
USER_DEFAULT="game"
REMOTE_DEFAULT="/data/game"
DRY_RUN=0

usage() {
    cat <<'EOF'
用法: deploy.sh --env <环境> --version <版本号> [--servers "<服务器id...>"] [--user <用户>] [--remote <目录>] [--dry-run]
EOF
    exit 1
}

while [[ $# -gt 0 ]]; do
    case "$1" in
        --env) ENV="${2:-}"; shift 2 ;;
        --version) VERSION="${2:-}"; shift 2 ;;
        --servers) SERVERS="${2:-}"; shift 2 ;;
        --user) USER_DEFAULT="${2:-}"; shift 2 ;;
        --remote) REMOTE_DEFAULT="${2:-}"; shift 2 ;;
        --dry-run) DRY_RUN=1; shift ;;
        -h | --help) usage ;;
        *) echo "未知参数: $1" >&2; usage ;;
    esac
done

[[ -n "$ENV" && -n "$VERSION" ]] || usage

CONF="$ROOT/deploy/cicd/$ENV/servers.conf"
[[ -f "$CONF" ]] || { echo "[错误] 缺少主机清单：deploy/cicd/$ENV/servers.conf" >&2; exit 1; }

DIST="$ROOT/dist/$ENV/$VERSION"

run() {
    if [[ $DRY_RUN -eq 1 ]]; then
        echo "  [dry-run] $*"
    else
        "$@"
    fi
}

# 选中服务器集合（空格/逗号分隔）
in_scope() {
    local id=$1 s
    [[ -z "$SERVERS" ]] && return 0
    for s in ${SERVERS//,/ }; do
        [[ "$s" == "$id" ]] && return 0
    done
    return 1
}

failed=0

while read -r id host user remote _; do
    [[ -z "$id" || "$id" == "#"* ]] && continue
    in_scope "$id" || continue

    user="${user:-$USER_DEFAULT}"
    remote="${remote:-$REMOTE_DEFAULT}"
    pkg="$DIST/$id.tar.gz"

    if [[ ! -f "$pkg" ]]; then
        echo "[错误] $id 缺少构建产物：dist/$ENV/$VERSION/$id.tar.gz" >&2
        failed=1
        continue
    fi

    echo "[分发] $id -> $user@$host:$remote"
    run ssh -o BatchMode=yes "$user@$host" "mkdir -p '$remote'"
    run scp -q -o BatchMode=yes "$pkg" "$user@$host:$remote/"

    # 先停旧进程，再解包覆盖（保留 log/pid/data 运行期目录），最后拉起。
    # 远端命令用 POSIX 写法，兼容 sh/bash 登录 shell。
    remote_cmd="cd '$remote' || exit 1
if [ -x '$id/stop.sh' ]; then './$id/stop.sh' || true; fi
tar -xzf '$id.tar.gz' && bash './$id/start.sh'"
    run ssh -o BatchMode=yes "$user@$host" "$remote_cmd" || {
        echo "[错误] $id 远端部署失败（$host）" >&2
        failed=1
    }
done <"$CONF"

exit $failed
