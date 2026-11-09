@echo off
rem Generate config Go structs + JSON data from Excel via Luban (Windows native).
rem
rem 步骤：
rem   1. 拉取 excelconfig 仓库到 cfg\excel\.src\（凭据由系统 git 配置 / SSH key 处理）
rem   2. 按 --env（和可选 --version）定位仓库里的子目录，把其中 xlsx 平铺复制到 cfg\excel\
rem   3. 调用 Luban 生成 share\tpl Go 代码 + cfg\data JSON
rem
rem Usage:
rem   gen-config.bat --env dev
rem   gen-config.bat --env prod --version v1.0.0
rem
rem 说明：
rem   bat 不处理凭据注入，直接用系统 git 配置（credential manager / .netrc / SSH key）。
rem   如果 excelconfig 仓库是私有的，请确保本机 git 已登录。
setlocal enabledelayedexpansion

cd /d "%~dp0..\.."

rem ---------- 参数解析 ----------
set "ENV="
set "VERSION="
:parse_arg
if "%~1"=="" goto done_parse
if /i "%~1"=="--env"     (set "ENV=%~2" & shift & shift & goto parse_arg)
if /i "%~1"=="--version" (set "VERSION=%~2" & shift & shift & goto parse_arg)
if /i "%~1"=="-h" goto help
if /i "%~1"=="--help" goto help
echo [gen-config] ERROR: 未知参数: %~1 1>&2
exit /b 2
:help
echo Usage: gen-config.bat --env ^<env^> [--version ^<version^>]
echo   --env     必填: dev / test / release / prod
echo   --version 可选: 例如 v1.0.0（prod/release 建议传）
echo.
echo excelconfig 仓库凭据由本机 git 配置处理（credential manager / SSH key）。
exit /b 0
:done_parse

if "%ENV%"=="" (
    echo [gen-config] ERROR: --env 必填 1>&2
    exit /b 2
)

rem ---------- 前置检查 ----------
where dotnet >nul 2>nul
if errorlevel 1 (
    echo [gen-config] ERROR: dotnet not found. Install .NET 8 Runtime: https://dotnet.microsoft.com/download 1>&2
    exit /b 1
)
where git >nul 2>nul
if errorlevel 1 (
    echo [gen-config] ERROR: git not found 1>&2
    exit /b 1
)

set "LUBAN_DLL=deploy\tools\Luban\Luban\Luban.dll"
if not exist "%LUBAN_DLL%" (
    echo [gen-config] ERROR: Luban.dll not found: %LUBAN_DLL% 1>&2
    exit /b 1
)
if not exist "cfg\luban.conf" (
    echo [gen-config] ERROR: luban.conf not found 1>&2
    exit /b 1
)

rem ---------- 1. 拉取 excelconfig ----------
if "%EXCEL_GIT_URL%"=="" set "EXCEL_GIT_URL=https://cnb.cool/nightcatgames/excelconfig.git"

set "EXCEL_DIR=cfg\excel"
set "EXCEL_SRC=cfg\excel\.src"

echo [gen-config] === Luban config generator ===
echo [gen-config] env        : %ENV%
if not "%VERSION%"=="" echo [gen-config] version   : %VERSION%
echo [gen-config] excel repo : %EXCEL_GIT_URL%

if not exist "%EXCEL_DIR%" mkdir "%EXCEL_DIR%"

if exist "%EXCEL_SRC%\.git" (
    echo [gen-config] excelconfig already cloned, pulling latest...
    git -C "%EXCEL_SRC%" pull --ff-only --depth 1 >nul 2>nul
    if errorlevel 1 (
        echo [gen-config] pull 失败，删了重 clone
        rmdir /s /q "%EXCEL_SRC%"
    )
)

if not exist "%EXCEL_SRC%\.git" (
    echo [gen-config] cloning excelconfig...
    git clone --depth 1 "%EXCEL_GIT_URL%" "%EXCEL_SRC%"
    if errorlevel 1 (
        echo [gen-config] ERROR: git clone failed 1>&2
        echo [gen-config]   excelconfig 仓库可能是私有的，请确认本机 git 已登录 1>&2
        exit /b 1
    )
)

rem ---------- 2. 平铺复制 xlsx ----------
set "ENV_PATH=%ENV%"
if not "%VERSION%"=="" set "ENV_PATH=%ENV%/%VERSION%"

set "SRC_DIR=%EXCEL_SRC%\%ENV_PATH%"
if not exist "%SRC_DIR%" (
    echo [gen-config] ERROR: excelconfig 里找不到目录: %SRC_DIR% 1>&2
    echo [gen-config]   excelconfig 顶层目录： 1>&2
    dir /b "%EXCEL_SRC%" 1>&2
    exit /b 1
)

echo [gen-config] picking xlsx from excelconfig/%ENV_PATH%

rem 清掉上次平铺的 xlsx
del /q "%EXCEL_DIR%\*.xlsx" 2>nul

rem 把目标目录（含所有子目录）里的 xlsx 平铺复制过来
for /r "%SRC_DIR%" %%f in (*.xlsx) do (
    copy /y "%%f" "%EXCEL_DIR%\" >nul
)

if not exist "%EXCEL_DIR%\__beans__.xlsx" (
    echo [gen-config] ERROR: cfg\excel\__beans__.xlsx 不存在！ 1>&2
    echo [gen-config]   excelconfig\%ENV_PATH% 目录内容： 1>&2
    dir "%SRC_DIR%" 1>&2
    exit /b 1
)

rem ---------- 3. Luban 生成 ----------
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
echo [gen-config] DONE -^> share\tpl + cfg\data  (env=%ENV%)
