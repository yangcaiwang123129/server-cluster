@echo off
rem Generate Luban config tables: Go code (go-json) + JSON data.
rem Output: share/cfg (package cfg) + share/cfg/data/*.json
rem
rem Note: Luban cleans outputCodeDir (deletes any file it did not generate),
rem so hand-written files live in cfg\tpl and are copied back after generation.
setlocal
cd /d "%~dp0..\.."

set "DOTNET=%USERPROFILE%\.dotnet\dotnet.exe"
set LUBAN_DLL=deploy\tools\Luban\Luban\Luban.dll

if not exist "%DOTNET%" (
    echo [ERROR] dotnet not found: %DOTNET%
    echo         please install .NET 8 SDK first
    pause
    exit /b 1
)
if not exist "%LUBAN_DLL%" (
    echo [ERROR] Luban.dll not found: %LUBAN_DLL%
    pause
    exit /b 1
)

rem Step 1: code only (code target cleans share/cfg, so it runs first)
"%DOTNET%" "%LUBAN_DLL%" ^
    -t server ^
    -c go-json ^
    --conf cfg\luban.conf ^
    -x outputCodeDir=share/cfg ^
    -x lubanGoModule=server-cluster/share/cfg
if errorlevel 1 (
    echo [ERROR] luban code generation failed
    pause
    exit /b 1
)

rem Step 2: data only (runs after code cleanup so JSON files survive)
"%DOTNET%" "%LUBAN_DLL%" ^
    -t server ^
    -d json ^
    --conf cfg\luban.conf ^
    -x outputDataDir=share/cfg/data
if errorlevel 1 (
    echo [ERROR] luban data generation failed
    pause
    exit /b 1
)

rem Step 3: merge generated files by table -> item.go (Item + TbItem), weapon.go ... + tables.go
go run ./deploy/tools/cfgmerge share\cfg
if errorlevel 1 (
    echo [ERROR] merge generated config files failed
    pause
    exit /b 1
)

rem Step 4: restore hand-written files (loader entry + package doc + smoke test)
copy /y cfg\tpl\cfg.go.txt share\cfg\cfg.go >nul
copy /y cfg\tpl\doc.go.txt share\cfg\doc.go >nul
copy /y cfg\tpl\cfg_smoke_test.go.txt share\cfg\cfg_smoke_test.go >nul

rem Step 5: format generated Go code
gofmt -w share\cfg 2>nul

echo [OK] config generated: share/cfg (Go code + data/*.json)
pause
endlocal
