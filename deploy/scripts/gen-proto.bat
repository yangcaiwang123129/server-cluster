@echo off
setlocal enabledelayedexpansion

rem Generate protobuf Go code: share\proto\*.proto -> share\pb\*.pb.go
rem Usage: deploy\scripts\gen-proto.bat (works from any directory)

cd /d "%~dp0..\.."

where protoc >nul 2>nul
if errorlevel 1 (
    echo [gen-proto] protoc not found in PATH
    pause
    exit /b 1
)
where protoc-gen-go >nul 2>nul
if errorlevel 1 (
    echo [gen-proto] protoc-gen-go not found: go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
    pause
    exit /b 1
)
where protoc-gen-go-grpc >nul 2>nul
if errorlevel 1 (
    echo [gen-proto] protoc-gen-go-grpc not found: go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
    pause
    exit /b 1
)

set "PROTOS="
for %%f in (share\proto\*.proto) do (
    if %%~zf GTR 0 set "PROTOS=!PROTOS! %%f"
)
if "!PROTOS!"=="" (
    echo [gen-proto] no non-empty .proto file found in share\proto
    pause
    exit /b 1
)

echo [gen-proto] protoc!PROTOS!
protoc -I share/proto --go_out=. --go_opt=module=server-cluster --go-grpc_out=. --go-grpc_opt=module=server-cluster!PROTOS!
if errorlevel 1 (
    echo [gen-proto] generation failed
    pause
    exit /b 1
)
echo [gen-proto] done: share\pb
pause
