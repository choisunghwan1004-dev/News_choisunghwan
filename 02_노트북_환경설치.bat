@echo off
chcp 65001 >nul
setlocal
cd /d "%~dp0"
set "LABROOT=%~dp0original\ai-datajournalism-labs-main"
set "VENV=%~dp0_venv"
if not exist "%LABROOT%\requirements.txt" (
 echo ERROR: 원본 requirements.txt를 찾지 못했습니다.
 pause
 exit /b 1
)
if exist "%VENV%\Scripts\python.exe" goto install
where py >nul 2>&1
if not errorlevel 1 (
 py -3.12 -V >nul 2>&1
 if not errorlevel 1 (
  echo Python 3.12 기반 가상환경을 만드는 중...
  py -3.12 -m venv "%VENV%"
  goto install
 )
)
where python >nul 2>&1
if errorlevel 1 (
 echo Python이 설치되지 않았습니다. Python 3.12를 먼저 설치하세요.
 echo https://www.python.org/downloads/
 pause
 exit /b 1
)
python -m venv "%VENV%"
:install
if not exist "%VENV%\Scripts\python.exe" (
 echo 가상환경 생성 실패
 pause
 exit /b 1
)
echo 노트북 수업 패키지를 설치합니다. 인터넷이 필요할 수 있습니다.
"%VENV%\Scripts\python.exe" -m pip install --upgrade pip
"%VENV%\Scripts\python.exe" -m pip install -r "%LABROOT%\requirements.txt"
if errorlevel 1 (
 echo 패키지 설치 중 오류가 발생했습니다.
 pause
 exit /b 1
)
echo 설치 완료. 03_노트북_열기.bat 를 실행하세요.
pause
