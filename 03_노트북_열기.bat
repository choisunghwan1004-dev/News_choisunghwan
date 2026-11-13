@echo off
chcp 65001 >nul
setlocal
set "LABROOT=%~dp0original\ai-datajournalism-labs-main"
set "PYTHON=%~dp0_venv\Scripts\python.exe"
if not exist "%PYTHON%" (
 echo 노트북 환경이 없습니다. 02_노트북_환경설치.bat 를 먼저 실행하세요.
 pause
 exit /b 1
)
if not exist "%LABROOT%\README.md" (
 echo 원본 자료를 찾을 수 없습니다.
 pause
 exit /b 1
)
echo Jupyter Lab을 실행합니다. 브라우저 창이 열릴 때까지 확인하세요.
"%PYTHON%" -m jupyter lab --notebook-dir="%LABROOT%"
if errorlevel 1 (
 echo Jupyter가 종료되었거나 설치 오류가 있습니다.
 pause
)
