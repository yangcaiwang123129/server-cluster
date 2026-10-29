#!/usr/bin/env bash
# 构建并打包单个/多个服务器实例。
#
# 用法：
#   deploy/scripts/build.sh --env dev --version v1.2.3 --servers "gate-1 game-1"
#
# 约定：
#   - 服务器 id 前缀决定服务与入口包：gate-* / game-* / battle-* / social-* / login-* / admin-*
#   - 配置优先取 deploy/env/<env>/<服务器id>/etc，不存在时回退到该服务自身的 etc 目录
#   - 产物：dist/<env>/<version>/<服务器id>/（可执行文件 + etc + 启停脚本）与 <服务器id>.tar.gz
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

ENV=""
VERSION=""
SERVERS=""

usage() {
    cat <<'EOF'
用法: build.sh --env <环境> --version <版本号> --servers "<服务器id...>"

示例: build.sh --env dev --version v1.2.3 --servers "gate-1 game-1 social-1"
EOF
    exit 1
}

while [[ $# -gt 0 ]]; do
    case "$1" in
        --env) ENV="${2:-}"; shift 2 ;;
        --version) VERSION="${2:-}"; shift 2 ;;
        --servers) SERVERS="${2:-}"; shift 2 ;;
        -h | --help) usage ;;
        *) echo "未知参数: $1" >&2; usage ;;
    esac
done

[[ -n "$ENV" && -n "$VERSION" && -n "$SERVERS" ]] || usage

# 服务器 id 前缀 → 入口包
entry_of() {
    case "$1" in
        gate-*) echo "./gate/gate" ;;
        game-*) echo "./node/game" ;;
        battle-*) echo "./node/battle" ;;
        social-*) echo "./node/social" ;;
        login-*) echo "./mesh/login" ;;
        admin-*) echo "./mesh/admin" ;;
        *) return 1 ;;
    esac
}

DIST="$ROOT/dist/$ENV/$VERSION"
mkdir -p "$DIST"

export CGO_ENABLED=0
export GOOS="${GOOS:-linux}"
export GOARCH="${GOARCH:-amd64}"

for server in $SERVERS; do
    server="${server//,/}"
    [[ -n "$server" ]] || continue

    entry="$(entry_of "$server")" || {
        echo "[错误] 无法识别服务器 id: $server（需以 gate-/game-/battle-/social-/login-/admin- 开头）" >&2
        exit 1
    }

    out="$DIST/$server"
    rm -rf "$out"
    mkdir -p "$out"

    # 配置查找顺序：deploy/env/<env>/<服务器id>/etc → deploy/env/<env>/<服务名>/etc → 仓库内 <服务>/etc
    svc_dir="${entry#./}"
    svc_name="$(basename "$svc_dir")"
    for etc_src in "$ROOT/deploy/env/$ENV/$server/etc" "$ROOT/deploy/env/$ENV/$svc_name/etc" "$ROOT/$svc_dir/etc"; do
        if [[ -d "$etc_src" ]]; then
            cp -R "$etc_src" "$out/etc"
            echo "[配置] $server <- ${etc_src#"$ROOT/"}"
            break
        fi
        etc_src=""
    done
    if [[ -z "$etc_src" ]]; then
        echo "[错误] $server 缺少配置目录（deploy/env/$ENV/$server/etc、deploy/env/$ENV/$svc_name/etc、$svc_dir/etc 均不存在）" >&2
        exit 1
    fi

    echo "[构建] $server ($entry, $GOOS/$GOARCH)"
    (cd "$ROOT" && go build -buildvcs=false -trimpath \
        -ldflags "-s -w" -o "$out/$server" "$entry")

    cp "$ROOT/deploy/scripts/start.sh" "$ROOT/deploy/scripts/stop.sh" "$out/"
    chmod +x "$out/start.sh" "$out/stop.sh" "$out/$server"
    # swagger 文档（login/admin 存在 docs 目录时一并打包）
    if [[ -d "$ROOT/$svc_dir/docs" ]]; then
        cp -R "$ROOT/$svc_dir/docs" "$out/docs"
    fi
    printf '%s\n' "$VERSION" >"$out/VERSION"

    (cd "$DIST" && tar -czf "$server.tar.gz" "$server")
    echo "[完成] $DIST/$server.tar.gz"
done

echo "全部完成：$DIST"
