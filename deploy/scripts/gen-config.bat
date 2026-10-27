@echo off
setlocal enabledelayedexpansion

rem ================================================================
rem Generate config Go structs + JSON data from Excel via Luban.
rem Usage:  deploy\scripts\gen-config.bat
rem Output: Go code  -> share\tpl\
rem         JSON     -> cfg\data\
rem ================================================================

cd /d "%~dp0..\.."

set "LUBAN_DLL=deploy\tools\Luban\Luban\Luban.dll"
set "CONF=cfg\luban.conf"
set "CODE_OUT=share\tpl"
set "DATA_OUT=cfg\data"
set "MODULE=server-cluster/share/tpl"
set "TPL_DIR=deploy\tools\Luban\templates"

if not exist "%LUBAN_DLL%" (
    echo [gen-config] ERROR: Luban.dll not found: %LUBAN_DLL%
    pause
    exit /b 1
)
if not exist "%CONF%" (
    echo [gen-config] ERROR: luban.conf not found: %CONF%
    pause
    exit /b 1
)

echo [gen-config] === Luban config generator ===
echo [gen-config] conf     : %CONF%
echo [gen-config] code out : %CODE_OUT%
echo [gen-config] data out : %DATA_OUT%
echo.

rem --- 1. Clean data dir (code dir is overwritten by Luban, no need to pre-clean) ---
if not exist "%DATA_OUT%" mkdir "%DATA_OUT%"

rem --- 2. Run Luban: code + data in one shot ---
echo [gen-config] running Luban...
dotnet "%LUBAN_DLL%" -t server -c go-json -d json ^
    --conf "%CONF%" ^
    -x outputCodeDir="%CODE_OUT%" ^
    -x outputDataDir="%DATA_OUT%" ^
    -x lubanGoModule=%MODULE% ^
    --customTemplateDir "%TPL_DIR%"
if errorlevel 1 (
    echo [gen-config] ERROR: Luban failed
    pause
    exit /b 1
)

rem --- 3. gofmt ---
echo [gen-config] gofmt...
where gofmt >nul 2>nul
if not errorlevel 1 (
    gofmt -w "%CODE_OUT%"
) else (
    echo [gen-config] gofmt not found, skip
)

rem --- 4. go build sanity check ---
echo [gen-config] go build...
where go >nul 2>nul
if not errorlevel 1 (
    go build ./share/tpl/...
    if errorlevel 1 (
        echo [gen-config] ERROR: go build failed
        pause
        exit /b 1
    )
) else (
    echo [gen-config] go not found, skip build check
)

echo.
echo [gen-config] DONE ^(code=%CODE_OUT%  data=%DATA_OUT%^)
pause
