# Ollama 준비 스크립트 (강의장 노트북용)
# 실행:  powershell -ExecutionPolicy Bypass -File .\setup_ollama.ps1
# 하는 일: 설치 확인 → 없으면 설치 → 서버 확인 → 원본 모델 받기 → 목록 보기

$Model = "qwen2.5:3b"

function Refresh-Path {
    # 설치 직후에도 VS Code를 다시 켜지 않고 ollama 명령을 쓸 수 있게 경로를 새로 읽어요
    $env:Path = [Environment]::GetEnvironmentVariable("Path", "Machine") + ";" +
                [Environment]::GetEnvironmentVariable("Path", "User")
}

Write-Host "`n[1/4] Ollama 설치 확인" -ForegroundColor Cyan
Refresh-Path
if (-not (Get-Command ollama -ErrorAction SilentlyContinue)) {
    Write-Host "  설치되어 있지 않아요. 설치를 시작합니다 (약 1.7GB, 몇 분)..."
    winget install --id Ollama.Ollama -e --accept-source-agreements --accept-package-agreements
    Refresh-Path
    if (-not (Get-Command ollama -ErrorAction SilentlyContinue)) {
        Write-Host "  설치 후에도 ollama를 찾지 못했어요. VS Code를 껐다 켜고 다시 실행하세요." -ForegroundColor Red
        exit 1
    }
    Write-Host "  설치 완료. 계정 만들기 창이 뜨면 'No thanks, I'll use Ollama locally'를 누르세요." -ForegroundColor Yellow
}
ollama --version

Write-Host "`n[2/4] Ollama 서버(11434번 문) 확인" -ForegroundColor Cyan
$ok = $false
for ($i = 0; $i -lt 15; $i++) {
    try { Invoke-RestMethod -Uri "http://localhost:11434" -TimeoutSec 2 | Out-Null; $ok = $true; break }
    catch {
        if ($i -eq 0) {
            Write-Host "  꺼져 있어요. 켜는 중..."
            Start-Process "ollama" -ArgumentList "serve" -WindowStyle Hidden
        }
        Start-Sleep -Seconds 2
    }
}
if (-not $ok) { Write-Host "  서버가 켜지지 않아요. 시작 메뉴에서 Ollama 앱을 실행한 뒤 다시 시도하세요." -ForegroundColor Red; exit 1 }
Write-Host "  Ollama is running"

Write-Host "`n[3/4] 원본 모델 받기: $Model (약 1.9GB)" -ForegroundColor Cyan
if ((ollama list) -match [regex]::Escape($Model)) { Write-Host "  이미 받아 두었어요. 건너뜁니다." }
else { ollama pull $Model }

Write-Host "`n[4/4] 받은 모델 목록" -ForegroundColor Cyan
ollama list

Write-Host "`n준비 끝! 이제 1_split_data.ipynb의 비교 칸을 실행하세요." -ForegroundColor Green
