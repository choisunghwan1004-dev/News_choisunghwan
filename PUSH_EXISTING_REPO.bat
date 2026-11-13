@echo off
setlocal EnableExtensions EnableDelayedExpansion
cd /d "%~dp0"
set "REPO=https://github.com/choisunghwan1004-dev/News_choisunghwan.git"
where git >nul 2>&1
if errorlevel 1 (
  echo [ERROR] Git is not installed. Install Git for Windows first.
  echo https://git-scm.com/downloads/win
  pause
  exit /b 1
)
git rev-parse --is-inside-work-tree >nul 2>&1
if errorlevel 1 (
  git init
  if errorlevel 1 goto :fail
)
git checkout -B main
if errorlevel 1 goto :fail
git add --all
if errorlevel 1 goto :fail
git diff --cached --quiet
if errorlevel 1 (
  git commit -m "Initial import of News_choisunghwan V2.0.1"
  if errorlevel 1 (
    echo [ERROR] Git commit failed. Set git user.name and user.email.
    echo git config --global user.name "Your GitHub Name"
    echo git config --global user.email "your-email@example.com"
    pause
    exit /b 1
  )
)
git remote get-url origin >nul 2>&1
if errorlevel 1 (
  git remote add origin "%REPO%"
  if errorlevel 1 goto :fail
) else (
  for /f "delims=" %%A in ('git remote get-url origin') do set "CURRENT_REMOTE=%%A"
  if /I not "!CURRENT_REMOTE!"=="%REPO%" (
    echo [ERROR] Existing git origin is different. Please check manually.
    pause
    exit /b 1
  )
)
echo [INFO] Uploading to %REPO%
git push -u origin main
if errorlevel 1 goto :fail
echo [SUCCESS] GitHub repository: https://github.com/choisunghwan1004-dev/News_choisunghwan
pause
exit /b 0
:fail
echo [ERROR] Upload failed. Check repository creation, access, login, and Git output.
pause
exit /b 1
