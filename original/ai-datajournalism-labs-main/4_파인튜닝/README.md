# 4_파인튜닝 — 내 모델 만들기

내가 모은 데이터로 작은 AI 모델(Qwen2.5-3B)에게 **MBC 기사 말투**를 가르치고,
원본 모델 · GPT와 누가 더 잘 쓰는지 비교해요.

> **용어 정리**
> - **Studio** = Unsloth Studio. 브라우저에서 `127.0.0.1:8888`로 여는 학습 프로그램. **버튼을 눌러** 학습해요.
> - **Jupyter 노트북** = VS Code에서 여는 `.ipynb` 파일. **코드 칸을 실행해서** 학습해요.
> - 학습은 둘 중 **하나만** 쓰면 돼요. 결과는 같아요.

---

## 이 폴더의 파일

| 파일 | 하는 일 |
|---|---|
| `1_split_data.ipynb` | 검수한 데이터 살펴보기 → 학습 · 평가 · 시험으로 나누기 → 확인 |
| `2_train_unsloth_gpu.ipynb` | (선택) Studio 대신 코드로 학습 · GGUF로 내보내기 |
| `3_gpu_speed_and_monitor.ipynb` | GPU·CPU 속도 비교, 학습 중 GPU 지켜보기 |
| `4_compare_results.ipynb` | 시험 문제로 원본 · 내 모델 · GPT 비교 → 웹 비교표 |
| `5_weather_tone_test.ipynb` | 스포츠로 배운 모델에 날씨를 넣어 톤이 따라오는지 시험 |
| `compare_tools.py` | 4·5번이 함께 쓰는 비교 도구 (지우지 마세요) |
| `setup_ollama.ps1` | Ollama 설치 확인 → 설치 → 원본 모델 받기를 한 번에 |

---

## 한눈에 보는 순서

```
① 나누기 → ② 학습 → ③ 내보내기(GGUF) → ④ Ollama 준비 → ⑤ 내 모델 등록 → ⑥ 비교 → ⑦ 날씨로 톤 시험
```

| 단계 | 어디서 | 만들어지는 것 |
|---|---|---|
| ① 나누기 | `1_split_data.ipynb` | `train.jsonl` `eval.jsonl` `test.jsonl` |
| ② 학습 | Studio **또는** `2_train_unsloth_gpu.ipynb` | 학습한 모델 |
| ③ 내보내기 | Studio의 Export 버튼 **또는** 2번 노트북 「5.」 칸 | `.gguf` 파일 + `Modelfile` |
| ④ Ollama 준비 | 터미널 `setup_ollama.ps1` | 원본 모델 `qwen2.5:3b` |
| ⑤ 내 모델 등록 | 터미널 `ollama create` | 내 모델 (예: `mbcsports`) |
| ⑥ 비교 | `4_compare_results.ipynb` | `compare_report.html` |
| ⑦ 날씨로 톤 시험 | `5_weather_tone_test.ipynb` | `weather_report.html` 등 |

> 터미널 여는 법: VS Code에서 `Ctrl` + `` ` `` (숫자 1 왼쪽 키)

---

## ① 나누기 — `1_split_data.ipynb`

1. `pairs_checked.csv`(칸: `본문`, `리드`, `판정`)를 이 폴더에 넣어요.
2. 설정 칸에서 데이터 종류를 골라요. 지시문이 자동으로 정해져요.
   ```python
   TASK = '보도자료'      # '보도자료' 또는 '방송원고'
   ```
3. 위에서부터 실행해요. 마지막 칸에서 **겹치는 쌍 0개**, **예상 학습 스텝**이 나오면 성공 ✅

---

## ② 학습 · ③ 내보내기

**Studio로 할 때** — `127.0.0.1:8888` → 학습 → 모델 `Qwen2.5-3B-Instruct (4bit)` · QLoRA →
데이터 `train.jsonl` · `eval.jsonl` (Local) → 에포크 2 · 컨텍스트 2048 · 배치 2 × 누적 4 · 학습률 2e-4 · rank 16 / alpha 32 →
끝나면 Export → **GGUF · Q4_K_M**

**Jupyter 노트북으로 할 때** — `2_train_unsloth_gpu.ipynb`를 위에서부터 실행해요. 「0. 확인」 칸이 `+cu128 True`를 보여 줘야 해요.
끝나면 `lead_model_gguf` 폴더에 GGUF 파일과 `Modelfile`이 생겨요.

---

## ④ Ollama 준비 — 명령 한 줄

```powershell
cd 4_파인튜닝
powershell -ExecutionPolicy Bypass -File .\setup_ollama.ps1
```

- 계정 만들기 창이 뜨면 → **로그인하지 말고** `No thanks, I'll use Ollama locally`
- 마지막에 `qwen2.5:3b ... 1.9 GB`와 `준비 끝!`이 나오면 성공 ✅
- 이미 설치돼 있으면 알아서 건너뛰어요. 여러 번 실행해도 괜찮아요.

---

## ⑤ 내 모델 등록 — 3단계

```
lead_model_gguf
├─ Modelfile                         ← Ollama에게 주는 설명서
└─ Qwen2.5-3B-Instruct.Q4_K_M.gguf   ← 내 모델 파일 (약 2GB)
```

**1.** `Modelfile`을 열어 **첫 줄**이 GGUF 파일을 가리키는지 확인해요. 다르면 첫 줄만 이렇게 바꾸고 저장해요. 둘째 줄부터는 건드리지 마세요.
```
FROM ./Qwen2.5-3B-Instruct.Q4_K_M.gguf
```

**2.** 터미널에서 등록해요.
```powershell
cd lead_model_gguf
ollama create mbcsports -f Modelfile
ollama list
```
목록에 `mbcsports:latest`와 `qwen2.5:3b` **두 개**가 보이면 성공 ✅

> 🔒 `ollama create`는 **내 PC 안에만** 등록해요. 인터넷에 올라가지 않아요.
> 온라인에 올리는 명령은 `ollama push`뿐이고, **이 수업에서는 쓰지 않아요.**

---

## ⑥ 비교 — `4_compare_results.ipynb`

1. 첫 칸에서 `MY_MODEL = 'mbcsports'`(⑤에서 등록한 이름)를 확인해요.
2. 위에서부터 실행해요. **실행할 때마다 세 모델에 새로 물어요** (1~2분).

| 칸 | 결과 |
|---|---|
| 2. 세 모델에 묻기 | `results/compare_날짜시각.json` (실행 기록) |
| 3-1. 표로 보기 | 정답 리드와 세 모델 답 전체 |
| 3-2. 자동 체크 | 끝인사 없음 · 기자에게 넘김 · 2문장 이하 · 길이 · 입력에 없는 숫자 |
| 4. 웹 비교표 | `compare_report.html` — 원본(회색) · 내 모델(초록) · GPT(파랑) |
| 5. 블라인드 평가 | `compare_blind.html` — 이름을 가리고 문제마다 A·B·C로 섞어요 |

> 💡 **A · B · C가 뭔가요?** 블라인드 비교표에서만 나와요. 누가 쓴 답인지 모르고 고르도록 **문제마다 순서를 새로 섞어요.**
> 1번 문제의 A와 2번 문제의 A는 다른 모델일 수 있어요. 다 고른 뒤 맨 아래 **🔑 정답지**로 확인해요.

> 💡 **먼저 주황 상자(정답 리드)를 읽고** 세 모델을 보세요. 볼 곳은 세 가지예요.
> ① 끝 문장: "○○○ 기자입니다"(앵커가 넘김) vs "MBC뉴스 ○○○입니다"(기자 끝인사) ② 길이 ③ "~는데요" 말투

자동 체크는 **형식**만 봐요. **사실이 맞는지**는 본문을 펼쳐서 사람이 직접 대조해요.

---

## ⑦ 날씨로 톤 시험 — `5_weather_tone_test.ipynb`

스포츠 리포트로만 배운 모델에 **한 번도 안 본 날씨**를 넣어요. 준비물 `test_weather.jsonl`(MBC 날씨 리포트 5건)은
기사 원문이라 GitHub에 없어요. **강사에게 받아** 이 폴더에 넣으세요.

| 부분 | 넣는 것 | 결과 | 보는 점 |
|---|---|---|---|
| ① 날씨 리포트 | 날씨 리포트 5건 → 앵커 리드 | `weather_report.html` | 처음 보는 주제에도 앵커 리드 형식이 나오나 |
| ② 긴 글 지시 | 같은 5건 + "700자 리포트를 써라" | `weather_long_report.html` | 지시가 바뀌어도 배운 습관(짧게)이 남나 |
| ③ 날씨 데이터 | Open-Meteo 내일 예보 숫자 (가입 필요 없음) | `meteo_report.html` | 숫자만 줘도 사실대로 쓰나 |

---

## Ollama 명령어 한눈에

| 명령 | 하는 일 |
|---|---|
| `ollama list` | 내 PC에 있는 모델 목록 |
| `ollama ps` | 지금 GPU에 올라와 있는 모델 (`100% GPU`면 가장 빨라요) |
| `ollama run mbcsports` | 내 모델과 직접 대화 (끝내기: `/bye`) |
| `ollama stop mbcsports` | GPU에서 내리기 (지워지지는 않아요) |
| `ollama rm mbcsports` | 내 모델 지우기 (다시 등록하고 싶을 때) |
| ~~`ollama push`~~ | 온라인 업로드 — **이 수업에서는 쓰지 않아요** |

---

## 막혔을 때

| 이런 메시지가 나오면 | 이렇게 해요 |
|---|---|
| `'ollama' 용어가 … 인식되지 않습니다` | ④ 스크립트 실행. 그래도 안 되면 VS Code를 껐다 켜기 |
| `SyntaxError` (Jupyter 칸에 `ollama` 명령) | 터미널에서 실행하기. Jupyter 칸에서는 앞에 `!` 붙이기 (`!ollama list`) |
| `NameError` | VS Code를 다시 켜면 노트북 기억이 지워져요. 맨 위 칸부터 다시 실행 |
| `ModuleNotFoundError: compare_tools` | `compare_tools.py`가 노트북과 같은 폴더에 있는지 확인 |
| `(실패: APIConnectionError)` | Ollama가 꺼져 있어요. 시작 메뉴에서 Ollama 실행 → `ollama list` |
| `(실패: BadRequestError)` | ⑤ 등록을 안 했거나 `MY_MODEL` 이름이 `ollama list`와 달라요 |
| `torch.cuda.is_available()`가 `False` | 아래 「꼭 지킬 것」 첫 번째 |

---

## ⚠ 꼭 지킬 것

- **`pip install unsloth`를 그냥 실행하지 마세요.** Windows에서는 PyTorch가 CPU판(`+cpu`)으로 바뀌어 GPU를 못 쓰게 돼요.
  바뀌었다면 Jupyter 칸에서 아래를 실행하고 **Restart** 하세요.
  ```python
  %pip install --force-reinstall --no-deps torch torchvision --index-url https://download.pytorch.org/whl/cu128
  ```
- **학습하는 동안에는 비교 칸을 실행하지 마세요.** GPU 메모리(8GB)를 나눠 쓰면 느려지거나 오류가 나요.

---

## GitHub에 올리지 않는 것

아래는 `.gitignore`로 막혀 있어요. **직접 `git add` 하지 마세요.**

| 종류 | 파일 | 이유 |
|---|---|---|
| 데이터 | `*.csv` `*.jsonl` | 기사 원문 (내부 학습용으로만 허락받음) |
| 모델 답 · 비교표 | `results/` `answers*.json` `*_report.html` `compare_blind.html` | 기사 내용이 들어 있어요 |
| 모델 | `lead_model*/` `*.gguf` `outputs/` `unsloth_compiled_cache/` | 용량이 크고, 공개하지 않아요 |
| 키 | `.env` | OpenAI API 키 |

> Jupyter 노트북을 커밋할 때는 먼저 **Clear All Outputs**(모든 출력 지우기) → 저장.
> 출력 칸에 기사 본문과 모델 답이 남아 있어요.
> 학습한 모델(`.gguf`, LoRA)은 GitHub, Hugging Face, ollama.com 같은 공개된 곳에 올리지 않아요.
