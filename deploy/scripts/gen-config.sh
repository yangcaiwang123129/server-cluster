#!/usr/bin/env bash
# Generate config Go structs + JSON data from Excel via Luban.
#
# 步骤：
#   1. 拉取 excelconfig 仓库到 cfg/excel/.src/
#   2. 按 --env（和可选 --version）定位仓库里的子目录，把其中 xlsx 平铺复制到 cfg/excel/
#   3. 调用 Luban 生成 share/tpl Go 代码 + cfg/data JSON
#
# Usage:
#   开发 / 测试环境：  gen-config.sh --env dev
#   预发布 / 生产：    gen-config.sh --env prod --version v1.0.0
#
# 凭据（任选其一）：
#   - 环境变量 EXCEL_GIT_USER + EXCEL_GIT_PASS（Jenkins withCredentials 注入）
#   - 或本机已配置的 git 凭据 / SSH key
set -euo pipefail
shopt -s nullglob

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

EXCEL_GIT_URL="${EXCEL_GIT_URL:-https://cnb.cool/nightcatgames/excelconfig.git}"

LUBAN_BIN="$ROOT/deploy/tools/Luban/luban.sh"
CONF="$ROOT/cfg/luban.conf"
CODE_OUT="$ROOT/share/tpl"
DATA_OUT="$ROOT/cfg/data"
MODULE="server-cluster/share/tpl"
TPL_DIR="$ROOT/deploy/tools/Luban/templates"
EXCEL_DIR="$ROOT/cfg/excel"
EXCEL_SRC="$EXCEL_DIR/.src"

# ---------- 参数解析 ----------
ENV=""
VERSION=""
while [[ $# -gt 0 ]]; do
    case "$1" in
        --env)     ENV="$2"; shift 2 ;;
        --version) VERSION="$2"; shift 2 ;;
        -h|--help)
            sed -n '2,18p' "$0"
            exit 0
            ;;
        *) echo "[gen-config] ERROR: 未知参数: $1" >&2; exit 2 ;;
    esac
done

if [[ -z "$ENV" ]]; then
    echo "[gen-config] ERROR: --env 必填（dev / test / release / prod）" >&2
    exit 2
fi

# release / prod 这类环境通常需要锁定版本；dev / test 则用最新。
# 用户如果没显式传 --version，就不加子目录。
ENV_PATH="$ENV"
if [[ -n "$VERSION" ]]; then
    ENV_PATH="$ENV/$VERSION"
fi

# ---------- 前置检查 ----------
if [[ ! -f "$LUBAN_BIN" ]]; then
    echo "[gen-config] ERROR: Luban launcher not found: $LUBAN_BIN" >&2; exit 1
fi
chmod +x "$LUBAN_BIN"
if [[ ! -f "$CONF" ]]; then
    echo "[gen-config] ERROR: luban.conf not found: $CONF" >&2; exit 1
fi

cd "$ROOT"

# ---------- 1. 拉取 excelconfig ----------
echo "[gen-config] === Luban config generator ==="
echo "[gen-config] env         : $ENV${VERSION:+  version=$VERSION}"
echo "[gen-config] excel repo : $EXCEL_GIT_URL"

mkdir -p "$EXCEL_DIR"

# 如果 .src 已存在且 remote 没变，尝试 pull；否则删掉重 clone。
if [[ -d "$EXCEL_SRC/.git" ]]; then
    EXISTING_REMOTE="$(git -C "$EXCEL_SRC" remote get-url origin 2>/dev/null || true)"
    if [[ "$EXISTING_REMOTE" == "$EXCEL_GIT_URL" ]]; then
        echo "[gen-config] excelconfig already cloned, pulling latest..."
        git -C "$EXCEL_SRC" pull --ff-only --depth 1 || {
            echo "[gen-config] pull 失败，删了重 clone"
            rm -rf "$EXCEL_SRC"
        }
    else
        echo "[gen-config] remote 变了（$EXISTING_REMOTE → $EXCEL_GIT_URL），重 clone"
        rm -rf "$EXCEL_SRC"
    fi
fi

if [[ ! -d "$EXCEL_SRC/.git" ]]; then
    echo "[gen-config] cloning excelconfig..."
    if [[ -n "${EXCEL_GIT_USER:-}" && -n "${EXCEL_GIT_PASS:-}" ]]; then
        # Jenkins 注入了凭据，用 GIT_ASKPASS 方式（避免 URL 内嵌特殊字符）
        ASKPASS_SCRIPT="$(mktemp)"
        cat > "$ASKPASS_SCRIPT" <<ASKPASS_EOF
#!/bin/sh
case "\$1" in
    *Username*) printf '%s' "$EXCEL_GIT_USER" ;;
    *Password*) printf '%s' "$EXCEL_GIT_PASS" ;;
esac
ASKPASS_EOF
        chmod +x "$ASKPASS_SCRIPT"
        GIT_ASKPASS="$ASKPASS_SCRIPT" GIT_TERMINAL_PROMPT=0 \
            git clone --depth 1 "$EXCEL_GIT_URL" "$EXCEL_SRC"
        rm -f "$ASKPASS_SCRIPT"
    else
        git clone --depth 1 "$EXCEL_GIT_URL" "$EXCEL_SRC"
    fi
fi

# ---------- 2. 平铺复制 xlsx 到 cfg/excel ----------
SRC_DIR="$EXCEL_SRC/$ENV_PATH"
if [[ ! -d "$SRC_DIR" ]]; then
    echo "[gen-config] ERROR: excelconfig 里找不到目录: $SRC_DIR" >&2
    echo "[gen-config]   excelconfig 顶层目录：" >&2
    ls "$EXCEL_SRC" | sed 's/^/    /' >&2
    exit 1
fi

echo "[gen-config] picking xlsx from excelconfig/$ENV_PATH"

# 先清掉上次平铺的 xlsx（保留 .src 隐藏目录）
find "$EXCEL_DIR" -maxdepth 1 -name '*.xlsx' -delete

# 把目标目录（含所有子目录）里的 xlsx 平铺复制过来
find "$SRC_DIR" -name '*.xlsx' -print0 | while IFS= read -r -d '' f; do
    cp -f "$f" "$EXCEL_DIR/"
done

XLSX_COUNT="$(find "$EXCEL_DIR" -maxdepth 1 -name '*.xlsx' | wc -l)"
echo "[gen-config] copied $XLSX_COUNT xlsx files to cfg/excel/"

if [[ ! -f "$EXCEL_DIR/__beans__.xlsx" ]]; then
    echo "[gen-config] ERROR: cfg/excel/__beans__.xlsx 不存在！" >&2
    echo "[gen-config]   excelconfig/$ENV_PATH 目录内容：" >&2
    ls -la "$SRC_DIR" >&2
    exit 1
fi

# ---------- 路径转换（MSYS/Cygwin → Windows 形式）----------
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

echo "[gen-config] conf     : $CONF"
echo "[gen-config] code out : $CODE_OUT"
echo "[gen-config] data out : $DATA_OUT"
echo

# ---------- 3. Luban 生成 ----------
EMBED_BAK="$(mktemp -t server-cluster-embed.go.XXXXXX)"
if [[ -f "$CODE_OUT/embed.go" ]]; then
    cp "$CODE_OUT/embed.go" "$EMBED_BAK"
fi

mkdir -p "$DATA_OUT"

echo "[gen-config] running Luban..."
"$LUBAN_BIN" -t server -c go-json -d json \
    --conf "$CONF_WIN" \
    -x "outputCodeDir=$CODE_OUT_WIN" \
    -x "outputDataDir=$DATA_OUT_WIN" \
    -x "lubanGoModule=$MODULE" \
    --customTemplateDir "$TPL_DIR_WIN"

if [[ -f "$EMBED_BAK" ]]; then
    cp "$EMBED_BAK" "$CODE_OUT/embed.go"
    rm -f "$EMBED_BAK"
fi

echo "[gen-config] sync data to embed dir..."
mkdir -p "$CODE_OUT/data"
cp -f "$DATA_OUT"/*.json "$CODE_OUT/data/"

if command -v gofmt >/dev/null 2>&1; then
    echo "[gen-config] gofmt..."
    gofmt -w "$CODE_OUT"
else
    echo "[gen-config] gofmt not found, skip"
fi

if command -v go >/dev/null 2>&1; then
    echo "[gen-config] go build..."
    go build -buildvcs=false ./share/tpl/...
else
    echo "[gen-config] go not found, skip build check"
fi

echo
echo "[gen-config] DONE (env=$ENV${VERSION:+ version=$VERSION})"
