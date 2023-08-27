@echo off
setlocal
set CGO_ENABLED=0
set GOARCH=amd64
set GOOS=windows
echo Building Windows binary...
go build -ldflags "-s -w" -trimpath -o out/
call :checkBuildStatus
set GOOS=linux
echo Building Linux binary...
go build -ldflags "-s -w" -trimpath -o out/
call :checkBuildStatus
exit /b 0
:checkBuildStatus
if not %ERRORLEVEL%==0 (
	echo Build failed
) else (
	echo Build succeeded
)
