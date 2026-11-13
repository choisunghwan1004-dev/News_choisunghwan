"""세 모델 비교 도구 — 4_compare_results.ipynb, 5_weather_tone_test.ipynb 에서 불러 써요.

    from compare_tools import connect, ask_models, auto_summary, make_report

- connect()       : 원본(Ollama) · 내 모델(Ollama) · gpt-4.1-mini(OpenAI) 연결
- ask_models()    : 같은 문제를 세 모델에 묻고, results/ 폴더에 기록으로 저장
- auto_summary()  : 리드 형식 자동 체크 요약표 (참고용, 최종 판단은 사람이)
- make_report()   : 세 모델 답을 나란히 놓은 웹 비교표(HTML) 만들기
"""
import html
import json
import os
import random
import re
from datetime import datetime
from pathlib import Path

import pandas as pd
from openai import OpenAI

ORIGINAL = '원본 Qwen2.5-3B'
MINE = '내가 학습한 모델'
GPT = 'gpt-4.1-mini'
COLORS = {ORIGINAL: '#9aa5b1', MINE: '#1f8a70', GPT: '#2563a8'}
RESULTS = Path('results')          # 실행 기록 폴더 (GitHub에 올리지 않아요)


# ── 1. 연결 ─────────────────────────────────────────────
def connect(my_model, openai_key=None, ollama_url='http://localhost:11434/v1'):
    """세 모델의 (연결, 모델 이름)을 돌려줘요. 키가 없으면 GPT 칸은 '(키 없음)'으로 나와요."""
    local = OpenAI(base_url=ollama_url, api_key='ollama')       # 내 PC의 Ollama
    cloud = OpenAI(api_key=openai_key) if openai_key else None
    return {ORIGINAL: (local, 'qwen2.5:3b'),
            MINE:     (local, my_model),
            GPT:      (cloud, 'gpt-4.1-mini')}


# ── 2. 묻기 ─────────────────────────────────────────────
def _ask(client, model, instruction, text):
    try:
        r = client.chat.completions.create(
            model=model, temperature=0,
            messages=[{'role': 'user', 'content': instruction + '\n\n' + text}])
        return r.choices[0].message.content.strip()
    except Exception as e:
        return f'(실패: {type(e).__name__})'


def ask_models(models, df, name):
    """df의 instruction · input 으로 세 모델에 묻고 {모델: [답, ...]} 을 돌려줘요.
    매번 새로 물어요. 결과는 results/<name>_<날짜시각>.json 에 기록으로 남아요."""
    answers = {}
    for label, (client, model) in models.items():
        print(f'묻는 중: {label} ({len(df)}문제)')
        answers[label] = [_ask(client, model, r['instruction'], r['input']) if client else '(키 없음)'
                          for _, r in df.iterrows()]
    RESULTS.mkdir(exist_ok=True)
    path = RESULTS / f'{name}_{datetime.now():%Y%m%d_%H%M}.json'
    record = {'만든 시각': f'{datetime.now():%Y-%m-%d %H:%M}',
              '지시문': sorted(set(df['instruction'])),
              '모델': {k: v[1] for k, v in models.items()},
              '답': answers}
    path.write_text(json.dumps(record, ensure_ascii=False, indent=1), encoding='utf-8')
    print('✅ 기록 저장:', path)
    return answers


def failed(a):
    return (not a) or a.startswith('(실패') or a.startswith('(키')


# ── 3. 자동 체크 ─────────────────────────────────────────
def checks(ans, src, gold=None):
    """리드 형식을 지켰나? — 형식만 봐요. 사실이 맞는지는 사람이 본문과 대조해요."""
    a = ans.replace('\\n', '\n')
    sentences = [s for s in re.split(r'(?<=[.!?다요])\s+', a.strip()) if s.strip()]
    extra = sorted(n for n in set(re.findall(r'\d[\d,.]*', a)) if n.strip('.,') not in src)
    greet = bool(re.search(r'MBC\s*뉴스\s*\S{2,4}입니다|안녕하세요', a))
    handoff = bool(re.search(r'\S{2,4}\s*(기자|기상분석관|캐스터)(입니다|가 보도합니다|가 전해드립니다|가 전합니다)', a))
    c = {'끝인사 없음': not greet,
         '기자에게 넘김': handoff,
         '2문장 이하': len(sentences) <= 2,
         '입력에 없는 숫자 없음': not extra,
         '_문장수': len(sentences), '_숫자': extra, '_인사': greet, '_글자': len(a)}
    if gold:
        c['_배율'] = len(a) / max(len(gold), 1)
        c['길이 비슷(0.5~2배)'] = 0.5 <= c['_배율'] <= 2
    return c


def auto_summary(df, answers, gold_col='output'):
    """모델별 자동 체크 요약표"""
    rows = []
    for label, outs in answers.items():
        res = [checks(a, r['input'], r.get(gold_col) if gold_col in df.columns else None)
               for a, (_, r) in zip(outs, df.iterrows()) if not failed(a)]
        n = len(res)
        row = {'모델': label, '답한 문제': f'{n}/{len(outs)}'}
        for k in ['끝인사 없음', '기자에게 넘김', '2문장 이하', '길이 비슷(0.5~2배)', '입력에 없는 숫자 없음']:
            if res and k in res[0]:
                row[k] = f'{sum(x[k] for x in res)}/{n}'
        row['평균 글자 수'] = round(sum(x['_글자'] for x in res) / max(n, 1))
        rows.append(row)
    return pd.DataFrame(rows).set_index('모델')


# ── 4. 웹 비교표 ─────────────────────────────────────────
def _badges(c):
    items = [('기자 끝인사·인사말 있음' if c['_인사'] else '끝인사 없음', c['끝인사 없음']),
             ('기자에게 넘김' if c['기자에게 넘김'] else '넘김 없음', c['기자에게 넘김']),
             (f"{c['_문장수']}문장", c['2문장 이하'])]
    if '_배율' in c:
        items.append(('길이 비슷' if c['길이 비슷(0.5~2배)'] else f"길이 {c['_배율']:.1f}배", c['길이 비슷(0.5~2배)']))
    items.append(('입력에 없는 숫자 없음' if not c['_숫자'] else '입력에 없는 숫자: ' + ', '.join(c['_숫자'][:3]),
                  c['입력에 없는 숫자 없음']))
    return ''.join(f'<span class="b {"ok" if ok else "no"}">{html.escape(t)}</span>' for t, ok in items)


def make_report(df, answers, path, title, note='', gold_col='output', gold_label='정답 리드',
                src_label='리포트 본문', blind=False, open_browser=True):
    """세 모델 답을 나란히 놓은 HTML. blind=True 면 문제마다 A·B·C로 섞고 맨 아래 정답지를 붙여요."""
    names = list(answers)
    blocks, key_rows = [], []
    for i, (_, r) in enumerate(df.iterrows()):
        gold = r.get(gold_col) if gold_col in df.columns else None
        order = names[:]
        if blind:
            random.shuffle(order)
        heads = ['ABC'[k] if blind else n for k, n in enumerate(order)]
        key_rows.append(f'<tr><td>{i+1}번</td>' + ''.join(f'<td>{"ABC"[k]} = {html.escape(n)}</td>'
                                                          for k, n in enumerate(order)) + '</tr>')
        cards = ''
        for head, n in zip(heads, order):
            a = answers[n][i]
            body = (f'<div class="ans empty">{html.escape(a or "답 없음")}</div>' if failed(a) else
                    f'<div class="ans">{html.escape(a.replace(chr(92) + "n", chr(10)))}</div>'
                    f'<div class="meta">{_badges(checks(a, r["input"], gold))}</div>')
            color = '#18232f' if blind else COLORS.get(n, '#18232f')
            cards += (f'<div class="card"><div class="h" style="background:{color}">{html.escape(head)}</div>{body}'
                      '<label class="human"><input type="checkbox"> 사실 오류 있음</label>'
                      '<label class="human"><input type="checkbox"> 가장 좋은 답</label></div>')
        head = html.escape(str(r['title'])) if 'title' in df.columns else f'문제 {i+1}'
        gold_html = f'<div class="gold"><b>{gold_label}</b> {html.escape(gold)}</div>' if gold else ''
        blocks.append(f'<h2>{i+1}. {head}</h2>'
                      f'<details><summary>{src_label} 보기 ({len(r["input"])}자)</summary>'
                      f'<pre class="q">{html.escape(r["input"])}</pre></details>'
                      f'{gold_html}<div class="row">{cards}</div>')

    mode = ('블라인드: 문제마다 A·B·C 순서를 새로 섞었어요. 같은 글자라도 문제마다 다른 모델이에요. 다 고른 뒤 맨 아래 정답지를 여세요.'
            if blind else '순서 고정: 원본(회색) → 내가 학습한 모델(초록) → gpt-4.1-mini(파랑)')
    key_html = ('<details class="key"><summary>🔑 정답지 열기 (다 고른 뒤에)</summary><table class="k">'
                + ''.join(key_rows) + '</table></details>') if blind else ''
    sum_html = auto_summary(df, answers, gold_col).reset_index().to_html(index=False, classes='sum', border=0)
    instr = ' / '.join(html.escape(s) for s in sorted(set(df['instruction'])))

    page = f'''<!doctype html><html lang="ko"><head><meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1"><title>{html.escape(title)}</title><style>
body{{font-family:"Malgun Gothic",sans-serif;margin:24px auto;max-width:1500px;background:#f7f7f5;color:#222;padding:0 16px}}
h1{{font-size:26px}} h2{{font-size:20px;margin-top:40px;border-top:2px solid #ddd;padding-top:20px}}
.box{{background:#fff;border:1px solid #ddd;border-radius:8px;padding:10px 14px;margin:8px 0;line-height:1.6}}
.sum{{border-collapse:collapse;background:#fff;margin:10px 0}} .sum th,.sum td{{border:1px solid #ddd;padding:6px 12px;text-align:center}}
.sum th{{background:#18232f;color:#fff}}
.q{{background:#fff;border:1px solid #ddd;padding:12px;white-space:pre-wrap;border-radius:8px;line-height:1.6}}
.gold{{background:#fff7ed;border-left:6px solid #e8590c;padding:10px 14px;margin:10px 0;line-height:1.7}}
.row{{display:flex;gap:12px;align-items:stretch}}
.card{{flex:1;background:#fff;border:1px solid #ddd;border-radius:10px;overflow:hidden;display:flex;flex-direction:column}}
.h{{color:#fff;font-weight:bold;padding:8px 12px;font-size:17px}}
.ans{{white-space:pre-wrap;line-height:1.7;padding:12px;flex:1}} .empty{{color:#aaa;text-align:center}}
.meta{{padding:0 12px 8px;font-size:12px}}
.b{{display:inline-block;padding:2px 8px;border-radius:10px;margin:2px 4px 2px 0}}
.ok{{background:#d8f3e8;color:#16664c}} .no{{background:#f6dede;color:#8a2a2a}}
.human{{display:block;font-size:13px;color:#555;padding:4px 12px;border-top:1px dashed #eee}}
details summary{{cursor:pointer;color:#2563a8;margin:6px 0}}
.key{{margin-top:40px;background:#fff;border:2px dashed #999;padding:12px;border-radius:8px}} .k td{{padding:4px 10px}}
</style></head><body>
<h1>{html.escape(title)}</h1>
<p>만든 시각: {datetime.now():%Y-%m-%d %H:%M} · 초록 배지 = 리드 형식을 지킴, 빨강 = 확인 필요 (자동 체크는 참고용)</p>
<p class="box"><b>지시문</b> {instr}</p>
<p class="box">{mode}</p>
{f'<p class="box">{note}</p>' if note else ''}
<h2 style="border:none;margin-top:16px">자동 체크 요약</h2>{sum_html}
<p>자동 체크는 <b>형식</b>만 봐요. <b>사실이 맞는지</b>는 {src_label}을 펼쳐서 직접 대조하고, 카드 아래 체크박스에 표시하세요.</p>
{''.join(blocks)}
{key_html}
</body></html>'''
    Path(path).write_text(page, encoding='utf-8')
    print('✅ 비교표 완성:', Path(path).resolve())
    if open_browser and os.name == 'nt':
        os.startfile(Path(path).resolve())       # 윈도우에서 브라우저로 바로 열기
