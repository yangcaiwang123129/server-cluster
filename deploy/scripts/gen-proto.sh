#!/usr/bin/env bash
# Generate protobuf Go code: share/proto/*.proto -> share/pb/*.pb.go
# Usage:  deploy/scripts/gen-proto.sh
set -euo pipefail
shopt -s nullglob

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT"

if ! command -v protoc >/dev/null 2>&1; then
    echo "[gen-proto] ERROR: protoc not found in PATH" >&2
    exit 1
fi
if ! command -v protoc-gen-go >/dev/null 2>&1; then
    echo "[gen-proto] ERROR: protoc-gen-go not found. Install: go install google.golang.org/protobuf/cmd/protoc-gen-go@latest" >&2
    exit 1
fi
if ! command -v protoc-gen-go-grpc >/dev/null 2>&1; then
    echo "[gen-proto] ERROR: protoc-gen-go-grpc not found. Install: go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest" >&2
    exit 1
fi

PROTOS=()
for f in share/proto/*.proto; do
    [[ -s "$f" ]] && PROTOS+=("$f")
done
if [[ ${#PROTOS[@]} -eq 0 ]]; then
    echo "[gen-proto] ERROR: no non-empty .proto file found in share/proto" >&2
    exit 1
fi

echo "[gen-proto] protoc ${PROTOS[*]}"
protoc -I share/proto \
    --go_out=. --go_opt=module=server-cluster \
    --go-grpc_out=. --go-grpc_opt=module=server-cluster \
    "${PROTOS[@]}"
echo "[gen-proto] DONE -> share/pb"
