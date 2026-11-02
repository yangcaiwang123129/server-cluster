@echo off
rem Generate config Go structs + JSON data from Excel via Luban (Windows native).
rem Usage:  deploy\scripts\gen-config.bat
rem Output: Go code -> share\tpl ; JSON -> cfg\data ; JSON copy -> share\tpl\data
setlocal enabledelayedexpansion

cd /d "%~dp0..\.."

rem The bundled Luban.exe may not be runnable on every Windows host; use dotnet + Luban.dll.
where dotnet >nul 2>nul
if errorlevel 1 (
    echo [gen-config] ERROR: dotnet not found. Install .NET 8 Runtime: https://dotnet.microsoft.com/download 1>&2
    exit /b 1
)

set "LUBAN_DLL=deploy\tools\Luban\Luban\Luban.dll"
if not exist "%LUBAN_DLL%" (
    echo [gen-config] ERROR: Luban.dll not found: %LUBAN_DLL% 1>&2
    exit /b 1
)
if not exist "cfg\luban.conf" (
    echo [gen-config] ERROR: luban.conf not found: cfg\luban.conf 1>&2
    exit /b 1
)
if not exist "deploy\tools\Luban\templates\go-json" (
    echo [gen-config] ERROR: custom template dir not found: deploy\tools\Luban\templates\go-json 1>&2
    exit /b 1
)

if not exist "cfg\data" mkdir "cfg\data"
if not exist "share\tpl\data" mkdir "share\tpl\data"

rem Luban removes unknown files in the output dir; back up hand-written embed.go first.
set "EMBED_BAK=%TEMP%\server-cluster-embed.go.bak"
if exist "share\tpl\embed.go" copy /y "share\tpl\embed.go" "%EMBED_BAK%" >nul

echo [gen-config] running Luban...
dotnet "%LUBAN_DLL%" -t server -c go-json -d json ^
    --conf "cfg\luban.conf" ^
    -x "outputCodeDir=%CD%\share\tpl" ^
    -x "outputDataDir=%CD%\cfg\data" ^
    -x "lubanGoModule=server-cluster/share/tpl" ^
    --customTemplateDir "deploy\tools\Luban\templates"
if errorlevel 1 (
    echo [gen-config] ERROR: Luban failed 1>&2
    exit /b 1
)

if exist "%EMBED_BAK%" (
    copy /y "%EMBED_BAK%" "share\tpl\embed.go" >nul
    del /q "%EMBED_BAK%"
)

rem Luban cleans unknown files in the code dir, including share\tpl\data, so recreate it.
if not exist "share\tpl\data" mkdir "share\tpl\data"

echo [gen-config] sync data to embed dir...
copy /y "cfg\data\*.json" "share\tpl\data\" >nul
if errorlevel 1 (
    echo [gen-config] ERROR: copy cfg\data\*.json failed 1>&2
    exit /b 1
)

where gofmt >nul 2>nul
if not errorlevel 1 (
    echo [gen-config] gofmt...
    gofmt -w share\tpl
)

where go >nul 2>nul
if not errorlevel 1 (
    echo [gen-config] go build...
    go build -buildvcs=false ./share/tpl/...
    if errorlevel 1 exit /b 1
)

echo.
echo [gen-config] DONE -^> share\tpl + cfg\data
