@echo off
chcp 65001 >nul
cd /d "%~dp0"
if not exist "DataJournalism_NewsStudio.exe" (
 echo 뉴스 스튜디오 실행 파일이 없습니다.
 pause
 exit /b 1
)
start "" "%~dp0DataJournalism_NewsStudio.exe"
exit /b 0
