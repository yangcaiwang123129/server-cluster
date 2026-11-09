#!/usr/bin/env bash
# Generate config Go structs + JSON data from Excel via Luban.
# Usage:  deploy/scripts/gen-config.sh
# Output: Go code  -> share/tpl/
#         JSON     -> cfg/data/
set -euo pipefail
shopt -s nullglob  # 防止 *.json 未匹配时被当作字面量路径

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

LUBAN_BIN="$ROOT/deploy/tools/Luban/luban.sh"
CONF="$ROOT/cfg/luban.conf"
CODE_OUT="$ROOT/share/tpl"
DATA_OUT="$ROOT/cfg/data"
MODULE="server-cluster/share/tpl"
TPL_DIR="$ROOT/deploy/tools/Luban/templates"

# Linux clone 下来的 .sh 可能没有 +x（git 默认不保留 Windows 上设置的 POSIX 位），
# 这里在执行前补上，避免 [[ ! -x ]] 直接拒绝。
if [[ ! -f "$LUBAN_BIN" ]]; then
    echo "[gen-config] ERROR: Luban launcher not found: $LUBAN_BIN" >&2
    exit 1
fi
chmod +x "$LUBAN_BIN"
if [[ ! -f "$CONF" ]]; then
    echo "[gen-config] ERROR: luban.conf not found: $CONF" >&2
    exit 1
fi

cd "$ROOT"

# MSYS/Cygwin 下 ROOT 形如 /c/Users/...，原生 Windows 版 Luban（dotnet）无法识别；
# 传给 Luban 的路径统一转成 Windows 形式。Linux/macOS 上原样返回。
win_path() {
    if command -v cygpath >/dev/null 2>&1; then
        cygpath -w "$1"
    else
        printf '%s' "$1"
    fi
}

CONF_WIN="$(win_path "$CONF")"
CODE_OUT_WIN="$(win_path "$CODE_OUT")"
DATA_OUT_WIN="$(win_path "$DATA_OUT")"
TPL_DIR_WIN="$(win_path "$TPL_DIR")"

echo "[gen-config] === Luban config generator ==="
echo "[gen-config] conf     : $CONF"
echo "[gen-config] code out : $CODE_OUT"
echo "[gen-config] data out : $DATA_OUT"
echo

# --- 0. Backup hand-written files in code dir (Luban removes unknown files there) ---
EMBED_BAK="$(mktemp -t server-cluster-embed.go.XXXXXX)"
if [[ -f "$CODE_OUT/embed.go" ]]; then
    cp "$CODE_OUT/embed.go" "$EMBED_BAK"
fi

# --- 1. Clean data dir (code dir is overwritten by Luban, no need to pre-clean) ---
mkdir -p "$DATA_OUT"

# --- 2. Run Luban: code + data in one shot ---
echo "[gen-config] running Luban..."
"$LUBAN_BIN" -t server -c go-json -d json \
    --conf "$CONF_WIN" \
    -x "outputCodeDir=$CODE_OUT_WIN" \
    -x "outputDataDir=$DATA_OUT_WIN" \
    -x "lubanGoModule=$MODULE" \
    --customTemplateDir "$TPL_DIR_WIN"

# --- 2.5 Restore hand-written files removed by Luban's output-dir cleanup ---
if [[ -f "$EMBED_BAK" ]]; then
    cp "$EMBED_BAK" "$CODE_OUT/embed.go"
    rm -f "$EMBED_BAK"
fi

# --- 2.6 Sync data into embed dir (share/tpl/data, loaded via go:embed) ---
echo "[gen-config] sync data to embed dir..."
mkdir -p "$CODE_OUT/data"
cp -f "$DATA_OUT"/*.json "$CODE_OUT/data/"

# --- 3. gofmt ---
if command -v gofmt >/dev/null 2>&1; then
    echo "[gen-config] gofmt..."
    gofmt -w "$CODE_OUT"
else
    echo "[gen-config] gofmt not found, skip"
fi

# --- 4. go build sanity check ---
if command -v go >/dev/null 2>&1; then
    echo "[gen-config] go build..."
    go build -buildvcs=false ./share/tpl/...
else
    echo "[gen-config] go not found, skip build check"
fi

echo
echo "[gen-config] DONE (code=$CODE_OUT  data=$DATA_OUT)"
