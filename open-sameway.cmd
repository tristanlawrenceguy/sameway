@echo off
REM Opens this workspace in a browser. Double-click it, or run it from a
REM terminal with arguments: open-sameway.cmd --workspace "C:\path\to\workspace"
REM
REM With no arguments it uses the nearest workspace.yaml, or %SAMEWAY_WORKSPACE%,
REM which is the same rule every other sameway command follows. If you have not
REM made a workspace yet: sameway init my-workspace
cd /d "%~dp0"
go run ./cmd/sameway open %*
if errorlevel 1 (
  echo.
  echo sameway stopped. The message above says why.
  pause
)
