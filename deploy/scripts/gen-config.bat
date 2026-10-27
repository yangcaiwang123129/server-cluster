@echo off
setlocal enabledelayedexpansion

rem Generate config Go structs + JSON data from Excel via Luban.
rem Usage: deploy\scripts\gen-config.bat (works from any directory)

cd /d "%~dp0..\.."

set "LUBAN_DLL=deploy\tools\Luban\Luban\Luban.dll"
set "CONF=cfg\luban.conf"
set "CODE_OUT=tmp_luban_code"
set "DATA_OUT=share\cfg\data"
set "CFG_DIR=share\cfg"
set "TPL_DIR=deploy\tools\Luban\templates"

if not exist "%LUBAN_DLL%" (
    echo [gen-config] Luban.dll not found: %LUBAN_DLL%
    pause
    exit /b 1
)

echo [gen-config] generating Go structs...
dotnet "%LUBAN_DLL%" -t server -c go-json --conf "%CONF%" -x outputCodeDir="%CODE_OUT%" -x lubanGoModule=server-cluster/share/cfg --customTemplateDir "%TPL_DIR%"
if errorlevel 1 (
    echo [gen-config] code generation failed
    pause
    exit /b 1
)

echo [gen-config] generating JSON data...
dotnet "%LUBAN_DLL%" -t server -d json --conf "%CONF%" -x outputDataDir="%DATA_OUT%"
if errorlevel 1 (
    echo [gen-config] data generation failed
    pause
    exit /b 1
)

echo [gen-config] copying struct files to %CFG_DIR%...
for %%f in ("%CODE_OUT%\*.go") do (
    set "name=%%~nxf"
    rem skip Tb*.go and Tables.go (only keep bean structs)
    echo !name! | findstr /b /i "Tb" >nul
    if errorlevel 1 (
        if /i not "!name!"=="Tables.go" (
            copy /y "%%f" "%CFG_DIR%\" >nul
        )
    )
)

echo [gen-config] gofmt...
where gofmt >nul 2>nul
if not errorlevel 1 (
    gofmt -w "%CFG_DIR%"
) else (
    echo [gen-config] gofmt not found, skipping
)

echo [gen-config] cleaning up temp dir...
if exist "%CODE_OUT%" rmdir /s /q "%CODE_OUT%"

echo [gen-config] done
pause
