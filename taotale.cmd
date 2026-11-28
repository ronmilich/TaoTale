@echo off
setlocal

if not "%~2"=="" goto usage
if "%~1"=="" (
    set "action=up --build --wait"
) else if /i "%~1"=="stop" (
    set "action=stop"
) else (
    goto usage
)

if not exist "%~dp0infra\.env" (
    >&2 echo Create infra\.env from infra\.env.example and set POSTGRES_PASSWORD.
    exit /b 1
)

docker compose --env-file "%~dp0infra\.env" --file "%~dp0infra\compose.yaml" %action%
exit /b %errorlevel%

:usage
>&2 echo Usage: taotale.cmd [stop]
exit /b 2
