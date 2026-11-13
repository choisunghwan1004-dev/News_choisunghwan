# 데이터 시각화 → 크롤링 → 파인튜닝 → 자동화 실습 코드

**Colab과 VS Code(내 컴퓨터) 어디서든** 그대로 돌아가요. 모든 노트북 맨 위의 **▶ 환경 준비** 칸이 실행 위치·GPU·한글 글꼴을 알아서 맞춰요.

| 폴더 | 내용 | GPU |
|---|---|---|
| 0_환경확인 | 패키지·GPU·Node·Ollama·Studio 상태 한 번에 확인 | 확인만 |
| 1_시각화기초 | Matplotlib · Seaborn 차트 5종 | 필요 없음 |
| 2_빅데이터_차트문법 | 기사 수 vs 미세먼지, 정직한 차트 규칙 / 3 **[GPU] 기사 제목 뜻으로 묶기** | 1·2 필요 없음 · 3 GPU 권장 |
| 3_크롤링 | 연습 사이트 크롤링, 보도자료 → 리드 JSONL | 필요 없음 |
| 4_파인튜닝 | 1 데이터 나누기 / 2 코드로 학습(선택) / 3 **[GPU] 속도 비교·학습 중 GPU 지켜보기** / 4 세 모델 비교(Ollama) / 5 날씨로 톤 시험 — 자세한 순서는 폴더 안 README | 2·3 GPU 필요 · 4·5 Ollama |
| 5_n8n_카오스보도 | 개표 알림 시뮬레이션, n8n 프롬프트·설정 메모 / 2 **[GPU] 인터뷰 녹음 받아쓰기 → 속보 초안 → 인용 확인** | 2 GPU 권장 |

## VS Code에서 시작하기 (처음 한 번) — Python 3.12 가상환경 하나
가상환경 = 이 수업 전용 파이썬 상자. 다른 프로젝트 패키지와 섞이지 않아요. GPU 실습까지 한 환경에서 해요.
```bash
cd ai-datajournalism-labs          # 내려받은(clone) 폴더
py -3.12 -m venv .venv            # 3.12가 없으면 python.org에서 3.12 설치
.venv\Scripts\activate
python -m pip install --upgrade pip
pip install -r requirements.txt

# [GPU] 실습용 — RTX 50 시리즈는 CUDA 12.8 PyTorch
pip install torch --index-url https://download.pytorch.org/whl/cu128
pip install sentence-transformers scikit-learn openai-whisper
python -c "import torch; print(torch.cuda.is_available())"   # True 이면 성공
```
Whisper는 ffmpeg도 필요해요: `winget install ffmpeg` (설치 후 VS Code 다시 열기)

**VS Code에서 고르기:** .ipynb 열기 → 오른쪽 위 **커널 선택 → Python 환경** → `.venv` → 0_환경확인/check_env.ipynb부터 실행

- 데이터 파일(빅카인즈 엑셀, data_air_2022.csv 등)은 **노트북과 같은 폴더**에 두기
- OpenAI 키: 같은 폴더에 `.env` 파일을 만들고 `OPENAI_API_KEY=sk-...` 한 줄 (Colab은 🔑 보안 비밀)

## GPU는 어디서 쓰나요?
- **pandas·그래프·크롤링**: CPU로 충분해요. GPU가 있어도 빨라지지 않아요.
- **pandas를 GPU로(선택)**: 환경 준비 칸의 `USE_GPU_PANDAS = True` — Colab GPU 런타임 또는 Linux/WSL + cuDF에서만 동작, 아니면 자동으로 일반 pandas
- **모델 실행**: Ollama는 GPU가 있으면 알아서 GPU를 써요 (4_파인튜닝/4_compare_results · 5_weather_tone_test). 설치는 `4_파인튜닝/setup_ollama.ps1`
- **임베딩·음성 인식**: 기사 제목 묶기(sentence-transformers), 받아쓰기(Whisper) — GPU면 몇 초~1분, CPU면 수 분~수십 분
- **모델 학습**: 강의장은 Unsloth Studio(화면), 코드로는 4_파인튜닝/2_train_unsloth_gpu.ipynb (Colab T4 또는 GPU PC)

## 올리지 않을 것 (.gitignore에 들어 있음)
크롤링한 원문(.csv, .jsonl), 빅카인즈 엑셀, 녹음·녹취록, 학습한 모델(.gguf, lead_model*/, outputs/), 모델 답·비교표(results/, *_report.html), API 키(.env)
