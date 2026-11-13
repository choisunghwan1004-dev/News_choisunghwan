@echo off
chcp 65001 >nul
setlocal
cd /d "%~dp0"
echo [선택 기능] 음성 받아쓰기 faster-whisper CPU 환경을 설치합니다.
echo 인터넷 연결이 필요하며, 최초 전사 시 tiny 모델을 받을 수 있습니다.
echo 이 기능은 원본 Jupyter 환경 및 뉴스 제작 기능과 별개로 설치됩니다.
if exist "_voice_venv\Scripts\python.exe" goto install
where py >nul 2>&1
if errorlevel 1 goto try_python
py -3.12 -V >nul 2>&1
if not errorlevel 1 (
 py -3.12 -m venv "_voice_venv"
 goto install
)
:try_python
where python >nul 2>&1
if errorlevel 1 goto fail
python -m venv "_voice_venv"
:install
if not exist "_voice_venv\Scripts\python.exe" goto fail
"_voice_venv\Scripts\python.exe" -m pip install --upgrade pip
if errorlevel 1 goto fail
"_voice_venv\Scripts\python.exe" -m pip install "faster-whisper>=1.1,<2.0"
if errorlevel 1 goto fail
echo 설치 완료. 뉴스 스튜디오 음성 자동화 화면에서 녹음파일을 선택하세요.
pause
exit /b 0
:fail
echo 설치가 완료되지 않았습니다. Python 3.12 환경과 인터넷을 확인하세요.
pause
exit /b 1
