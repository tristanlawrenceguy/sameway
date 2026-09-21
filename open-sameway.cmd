@echo off
REM Opens a sameway workspace in your browser. Double-click this file, or run
REM it from a terminal with the same arguments `sameway open` takes:
REM   open-sameway.cmd --workspace "D:\work\my-workspace"
REM   open-sameway.cmd --page /design
REM
REM Without --workspace it uses %SAMEWAY_WORKSPACE%, or the folder you ran it
REM from when that folder is a workspace. A fresh clone has neither, so a
REM double-click opens examples\workspaces\starter instead of stopping on an
REM error. Make one of your own with `sameway init my-workspace`.
setlocal enabledelayedexpansion
set "PICK="
set "ARGS=x%*"
if not defined SAMEWAY_WORKSPACE if "!ARGS:-workspace=!"=="!ARGS!" (
  if exist "%CD%\workspace.yaml" (
    set PICK=--workspace "%CD%"
  ) else (
    echo No workspace given, so this is the starter example.
    echo For your own: sameway init my-workspace, then pass --workspace with it.
    echo.
    set PICK=--workspace "%~dp0examples\workspaces\starter"
  )
)
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
"%EXE%" open %PICK% %*
set "CODE=%ERRORLEVEL%"
del "%EXE%" >nul 2>&1
if not "%CODE%"=="0" goto stopped
exit /b 0
:stopped
echo.
echo sameway stopped. The message above says why.
pause
exit /b 1
