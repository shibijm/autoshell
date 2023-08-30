@echo off
setlocal
for /F "tokens=*" %%i in ('type .env') do set %%i
set CGO_ENABLED=0
set GOOS=windows
set GOARCH=amd64
call :build
set GOOS=linux
call :build
set GOARCH=arm64
call :build
exit /b 0
:build
echo Building %GOOS%-%GOARCH%
go build -ldflags "-s -w -X autoshell/utils.devicePassSeed=%DEVICE_PASS_SEED%" -trimpath -o out/%GOOS%-%GOARCH%/
