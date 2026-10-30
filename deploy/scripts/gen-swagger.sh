#!/usr/bin/env bash
# Per-service swagger docs generator.
# Annotations live in each service's own source (mesh/<svc>/app/handler),
# main.go header defines @title/@host; docs output to mesh/<svc>/docs.
# Usage:  deploy/scripts/gen-swagger.sh [service ...]
#         no args = all services; e.g. gen-swagger.sh login admin
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT"

SERVICES=("$@")
if [[ ${#SERVICES[@]} -eq 0 ]]; then
    SERVICES=(login admin)
fi

if ! command -v swag >/dev/null 2>&1; then
    echo "[gen-swagger] ERROR: swag not found. Install: go install github.com/swaggo/swag/cmd/swag@latest" >&2
    exit 1
fi

echo "[gen-swagger] === per-service swagger generator ==="
echo

for svc in "${SERVICES[@]}"; do
    if [[ ! -f "mesh/$svc/main.go" ]]; then
        echo "[gen-swagger] ERROR: mesh/$svc/main.go not found, skip" >&2
        exit 1
    fi
    echo "[gen-swagger] ---- $svc ----"
    (
        cd "mesh/$svc"
        swag init -g main.go -o docs --parseDependency --parseInternal
    )
    echo "[gen-swagger] $svc OK -> mesh/$svc/docs/swagger.json"
    echo
done

echo "[gen-swagger] DONE"
