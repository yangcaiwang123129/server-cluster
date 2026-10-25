@echo off
setlocal

rem Generate swagger docs: mesh\login\app\web\docs
rem Usage: deploy\scripts\gen-swagger.bat (works from any directory)

cd /d "%~dp0..\.."

where swag >nul 2>nul
if errorlevel 1 (
    echo [gen-swagger] swag not found: go install github.com/swaggo/swag/cmd/swag@latest
    pause
    exit /b 1
)

echo [gen-swagger] swag init -g mesh/login/main.go
swag init -g mesh/login/main.go -d ./ --parseInternal -o mesh/login/app/web/docs
if errorlevel 1 (
    echo [gen-swagger] generation failed
    pause
    exit /b 1
)
echo [gen-swagger] done: mesh\login\app\web\docs
pause
