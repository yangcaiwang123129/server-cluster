#!/usr/bin/env bash
# Luban launcher (dual entry, Linux/macOS)
#   Prefer Luban/Luban (native host from official package).
#   Fallback to dotnet Luban/Luban.dll if exe is missing/broken.
#
# Official package: https://github.com/focus-creative-games/luban/releases
#   Extract Luban.7z into deploy/tools/Luban/Luban/ to get the native binary.
#
# Note: Windows counterpart is luban.exe (kept as .cmd so it does not
# clash with the Luban/ subdirectory on case-insensitive filesystems).
set -euo pipefail

LUBAN_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BIN="$LUBAN_DIR/Luban/Luban"
DLL="$LUBAN_DIR/Luban/Luban.dll"

# Windows 包内的可执行文件是 Luban.exe，不是 Linux 的 Luban。
if [[ ! -x "$BIN" && -x "$BIN.exe" ]]; then
    BIN="$BIN.exe"
fi

# MSYS/Cygwin 下 POSIX 路径不能直接交给原生 dotnet，需转换为 Windows 路径。
DLL_RUN="$DLL"
if command -v cygpath >/dev/null 2>&1; then
    DLL_RUN="$(cygpath -w "$DLL")"
fi

if [[ ! -f "$DLL" ]]; then
    echo "[luban] ERROR: Luban.dll not found: $DLL" >&2
    exit 1
fi

# Fast path: native host present and runnable.
if [[ -x "$BIN" ]]; then
    "$BIN" "$@" && exit 0
    echo "[luban] Luban binary failed (exit=$?), fallback to dotnet Luban.dll" >&2
fi

# Fallback: dotnet + dll.
if ! command -v dotnet >/dev/null 2>&1; then
    echo "[luban] ERROR: dotnet not found. Install .NET 8 Runtime: https://dotnet.microsoft.com/download" >&2
    exit 1
fi
exec dotnet "$DLL_RUN" "$@"
