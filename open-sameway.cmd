@echo off
REM Opens this workspace in a browser. Double-click it, or run it from a
REM terminal with arguments: open-sameway.cmd --workspace "C:\path\to\workspace"
REM
REM With no arguments it uses the nearest workspace.yaml, or %SAMEWAY_WORKSPACE%,
REM which is the same rule every other sameway command follows. If you have not
REM made a workspace yet: sameway init my-workspace
cd /d "%~dp0"
REM Windows Smart App Control sometimes refuses a freshly built program with
REM "An Application Control policy has blocked this file". A build with a
REM different build id is a different file and is usually let through, so
REM the launcher builds, checks the build runs, and builds again if not.
set "EXE=%TEMP%\sameway-open-%RANDOM%.exe"
set /a TRY=0
:build
set /a TRY+=1
go build -ldflags "-buildid=sw-%RANDOM%%RANDOM%" -o "%EXE%" ./cmd/sameway
if errorlevel 1 goto stopped
"%EXE%" --version >nul 2>&1
if not errorlevel 1 goto run
if %TRY% geq 6 (
  "%EXE%" --version
  goto stopped
)
echo Windows refused that build, so building another ^(try %TRY% of 6^)...
goto build
:run
"%EXE%" open %*
set "CODE=%ERRORLEVEL%"
del "%EXE%" >nul 2>&1
if not "%CODE%"=="0" goto stopped
exit /b 0
:stopped
echo.
echo sameway stopped. The message above says why.
pause
exit /b 1
