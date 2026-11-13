@echo off
setlocal EnableExtensions
cd /d "%~dp0"
where gh >nul 2>&1
if errorlevel 1 (
  echo [ERROR] GitHub CLI is not installed. Install it or use GITHUB_SETUP_KR.md method B.
  echo https://cli.github.com/
  pause
  exit /b 1
)
where git >nul 2>&1
if errorlevel 1 (
  echo [ERROR] Git for Windows is not installed.
  echo https://git-scm.com/downloads/win
  pause
  exit /b 1
)
gh auth status >nul 2>&1
if errorlevel 1 (
  echo [INFO] Sign in to GitHub first.
  gh auth login
  if errorlevel 1 goto :fail
)
gh repo view choisunghwan1004-dev/News_choisunghwan >nul 2>&1
if not errorlevel 1 (
  echo [ERROR] Repository already exists. Use PUSH_EXISTING_REPO.bat carefully.
  pause
  exit /b 1
)
git rev-parse --is-inside-work-tree >nul 2>&1
if errorlevel 1 git init
git checkout -B main
if errorlevel 1 goto :fail
git add --all
if errorlevel 1 goto :fail
git diff --cached --quiet
if errorlevel 1 (
  git commit -m "Initial import of News_choisunghwan V2.0.1"
  if errorlevel 1 goto :fail
)
gh repo create choisunghwan1004-dev/News_choisunghwan --private --source=. --remote=origin --push
if errorlevel 1 goto :fail
echo [SUCCESS] Repository: https://github.com/choisunghwan1004-dev/News_choisunghwan
pause
exit /b 0
:fail
echo [ERROR] Could not create or upload repository. Check authentication and output above.
pause
exit /b 1
