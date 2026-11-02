@echo off
rem Generate protobuf Go code: share\proto\*.proto -> share\pb\*.pb.go
rem Usage:  deploy\scripts\gen-proto.bat
setlocal enabledelayedexpansion

cd /d "%~dp0..\.."

where protoc >nul 2>nul
if errorlevel 1 (
    echo [gen-proto] ERROR: protoc not found in PATH 1>&2
    exit /b 1
)
where protoc-gen-go >nul 2>nul
if errorlevel 1 (
    echo [gen-proto] ERROR: protoc-gen-go not found. Install: go install google.golang.org/protobuf/cmd/protoc-gen-go@latest 1>&2
    exit /b 1
)
where protoc-gen-go-grpc >nul 2>nul
if errorlevel 1 (
    echo [gen-proto] ERROR: protoc-gen-go-grpc not found. Install: go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest 1>&2
    exit /b 1
)

set "PROTOS="
for %%f in (share\proto\*.proto) do (
    if %%~zf NEQ 0 set "PROTOS=!PROTOS! share/proto/%%~nxf"
)
if not defined PROTOS (
    echo [gen-proto] ERROR: no non-empty .proto file found in share/proto 1>&2
    exit /b 1
)

echo [gen-proto] protoc!PROTOS!
protoc -I share/proto --go_out=. --go_opt=module=server-cluster --go-grpc_out=. --go-grpc_opt=module=server-cluster !PROTOS!
if errorlevel 1 exit /b 1
echo [gen-proto] DONE -^> share/pb
