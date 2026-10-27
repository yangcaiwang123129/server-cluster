@echo off
setlocal enabledelayedexpansion

rem ================================================================
rem Per-service swagger docs generator.
rem Annotations live in each service's own source (mesh\<svc>\app\handler),
rem main.go header defines its own @title/@host; docs output to mesh\<svc>\docs.
rem Usage:  deploy\scripts\gen-swagger.bat [service ...]
rem         no args = all services; or: gen-swagger.bat login admin
rem ================================================================

cd /d "%~dp0..\.."

set "SERVICES=%*"
if "%SERVICES%" == "" set "SERVICES=login admin"

echo [gen-swagger] === per-service swagger generator ===
echo.

for %%S in (%SERVICES%) do call :gen %%S
if errorlevel 1 goto :fail

echo [gen-swagger] DONE
pause
exit /b 0

:gen
if not exist "mesh\%1\main.go" (
    echo [gen-swagger] ERROR: mesh\%1\main.go not found, skip
    exit /b 1
)
echo [gen-swagger] ---- %1 ----
pushd "mesh\%1"
swag init -g main.go -o docs --parseDependency --parseInternal
if errorlevel 1 (
    popd
    echo [gen-swagger] ERROR: %1 swagger generate failed
    exit /b 1
)
popd
echo [gen-swagger] %1 OK ^> mesh\%1\docs\swagger.json
echo.
exit /b 0

:fail
pause
exit /b 1
