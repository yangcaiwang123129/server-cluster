@echo off
rem Generate config Go structs + JSON data from excel via Luban (local dev).
rem Excel files are expected to already be in cfg\excel\ (Luban reads cfg\luban.conf -> dataDir=excel).
rem This script is the Windows counterpart of gen-config.sh, but skips the
rem excelconfig repo clone / env-subdir / flatten steps - that is CI-only.
rem
rem Usage (double-click or from cmd):
rem   gen-config.bat
rem   gen-config.bat --no-pause
setlocal

cd /d "%~dp0..\.."

set "NOPAUSE=0"
if /i "%~1"=="--no-pause" set "NOPAUSE=1"

rem ---------- Pre-flight checks ----------
set "LUBAN_EXE=deploy\tools\Luban\Luban\Luban.exe"
set "LUBAN_DLL=deploy\tools\Luban\Luban\Luban.dll"

if not exist "cfg\luban.conf" (
    echo [gen-config] ERROR: luban.conf not found
    goto :end 1
)
if not exist "%LUBAN_EXE%" if not exist "%LUBAN_DLL%" (
    echo [gen-config] ERROR: Luban not found: %LUBAN_EXE% / %LUBAN_DLL%
    goto :end 1
)
if not exist "%LUBAN_EXE%" (
    where dotnet >nul 2>nul
    if errorlevel 1 (
        echo [gen-config] ERROR: dotnet not found. Install .NET 8 Runtime: https://dotnet.microsoft.com/download
        goto :end 1
    )
)

echo [gen-config] === Luban config generator (local) ===
echo [gen-config] excel dir : cfg\excel
echo [gen-config] code  out : share\tpl
echo [gen-config] data  out : cfg\data
echo.

rem ---------- Backup hand-written embed.go ----------
set "EMBED_BAK=%TEMP%\server-cluster-embed.go.bak"
if exist "share\tpl\embed.go" copy /y "share\tpl\embed.go" "%EMBED_BAK%" >nul

rem ---------- Run Luban ----------
set DOTNET_SYSTEM_GLOBALIZATION_INVARIANT=1

if not exist "cfg\data" mkdir "cfg\data"

echo [gen-config] running Luban...
if exist "%LUBAN_EXE%" (
    "%LUBAN_EXE%" -t server -c go-json -d json ^
        --conf "cfg\luban.conf" ^
        -x "outputCodeDir=%CD%\share\tpl" ^
        -x "outputDataDir=%CD%\cfg\data" ^
        -x "lubanGoModule=server-cluster/share/tpl" ^
        --customTemplateDir "deploy\tools\Luban\templates"
    if errorlevel 1 (
        echo [gen-config] Luban exe failed, falling back to dotnet dll
        dotnet "%LUBAN_DLL%" -t server -c go-json -d json ^
            --conf "cfg\luban.conf" ^
            -x "outputCodeDir=%CD%\share\tpl" ^
            -x "outputDataDir=%CD%\cfg\data" ^
            -x "lubanGoModule=server-cluster/share/tpl" ^
            --customTemplateDir "deploy\tools\Luban\templates"
    )
) else (
    dotnet "%LUBAN_DLL%" -t server -c go-json -d json ^
        --conf "cfg\luban.conf" ^
        -x "outputCodeDir=%CD%\share\tpl" ^
        -x "outputDataDir=%CD%\cfg\data" ^
        -x "lubanGoModule=server-cluster/share/tpl" ^
        --customTemplateDir "deploy\tools\Luban\templates"
)
if errorlevel 1 (
    echo [gen-config] ERROR: Luban failed
    goto :end 1
)

rem ---------- Restore embed.go (Luban wipes unknown files from output dir) ----------
if exist "%EMBED_BAK%" (
    copy /y "%EMBED_BAK%" "share\tpl\embed.go" >nul
    del /q "%EMBED_BAK%"
)

rem ---------- Copy JSON data into embed dir ----------
echo [gen-config] sync data to embed dir...
if not exist "share\tpl\data" mkdir "share\tpl\data"
copy /y "cfg\data\*.json" "share\tpl\data\" >nul
if errorlevel 1 (
    echo [gen-config] ERROR: copy cfg\data\*.json failed
    goto :end 1
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
    if errorlevel 1 goto :end 1
)

echo.
echo [gen-config] DONE -^> share\tpl + cfg\data

:end
if "%EXIT_CODE%"=="" set "EXIT_CODE=0"
if not "%EXIT_CODE%"=="0" echo [gen-config] FAILED ^(exit=%EXIT_CODE%^)
if not "%NOPAUSE%"=="1" (
    echo.
    pause
)
exit /b %EXIT_CODE%
