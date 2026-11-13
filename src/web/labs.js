/* V2 standalone lab interfaces; does not replace or alter V1 editor state. */
(()=>{'use strict';
const $=q=>document.querySelector(q);let vData=null,bData=null,cData=null,tData=null,lastAlert=null;
const $id=id=>document.getElementById(id);
const pop=s=>{const x=$id('toast');x.textContent=s;x.classList.add('show');setTimeout(()=>x.classList.remove('show'),3500)};
const save=(filename,data,mime='text/plain;charset=utf-8')=>{const blob=new Blob([data],{type:mime});const u=URL.createObjectURL(blob),a=document.createElement('a');a.href=u;a.download=filename;document.body.append(a);a.click();a.remove();setTimeout(()=>URL.revokeObjectURL(u),1800)};
const jsonSave=(name,obj)=>save(name,JSON.stringify(obj,null,2),'application/json;charset=utf-8');
const kv=(tag,text,cls)=>{let node=document.createElement(tag);if(cls)node.className=cls;node.textContent=text;return node};
const clear=node=>node.replaceChildren();
const formatted=n=>Number(n).toLocaleString('ko-KR',{maximumFractionDigits:2});
const error=err=>pop(err.message||String(err));
async function sendJSON(endpoint,body){const r=await fetch(endpoint,{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify(body)});let val=await r.json().catch(()=>({error:`응답 코드 ${r.status}`}));if(!r.ok)throw Error(val.error||`HTTP ${r.status}`);return val}
async function sendFile(endpoint,file){if(!file)throw Error('CSV/JSONL 파일을 선택하거나 샘플을 불러오세요.');let fd=new FormData();fd.append('file',file,file.name);const r=await fetch(endpoint,{method:'POST',body:fd});const val=await r.json().catch(()=>({error:`응답 코드 ${r.status}`}));if(!r.ok)throw Error(val.error||`HTTP ${r.status}`);return val}
function kpis(node,values){clear(node);for(const [name,val] of values){let c=kv('div','','kpibox');c.append(kv('b',String(val)),kv('span',name));node.append(c)}}
function table(node,heads,rows){clear(node);const t=document.createElement('table'),the=document.createElement('thead'),tr=document.createElement('tr');for(const h of heads)tr.append(kv('th',h));the.append(tr);t.append(the);const tbody=document.createElement('tbody');for(const row of rows){const r=document.createElement('tr');for(const x of row)r.append(kv('td',String(x)));tbody.append(r)}t.append(tbody);node.append(t)}
const COLORS=['#39876f','#9fbea6','#d6b98d','#719cb3','#b5a4c8','#d89783','#8bb8aa','#819fa1','#9eae76','#b0a1a1'];
const svgNS='http://www.w3.org/2000/svg';
function svgNode(tag,attrs,txt){let n=document.createElementNS(svgNS,tag);for(let [k,v] of Object.entries(attrs||{}))n.setAttribute(k,String(v));if(txt!==undefined)n.textContent=txt;return n}
function drawChart(host,data,type='bar',title='집계 그래프'){
 clear(host);if(!data||!data.length){host.append(kv('p','그래프로 표시할 데이터가 없습니다.','hint'));return}
 const arr=data.slice(0,10),total=arr.reduce((s,x)=>s+Number(x.value),0),W=740,H=365;
 const s=svgNode('svg',{viewBox:`0 0 ${W} ${H}`,role:'img','aria-label':title});s.append(svgNode('title',{},title));s.append(svgNode('text',{x:24,y:29,'font-size':16,'font-weight':700,fill:'#265947'},title));
 if(type==='pie'){
  let angle=-Math.PI/2;const sum=total||1,cx=230,cy=185,r=124;
  for(let i=0;i<arr.length;i++){let entry=arr[i];if(entry.value<=0)continue;let sweep=Number(entry.value)/sum*Math.PI*2,next=angle+sweep;
   if(sweep>Math.PI*2-0.00001){s.append(svgNode('circle',{cx,cy,r,fill:COLORS[i%COLORS.length]}))}
   else{let x1=cx+r*Math.cos(angle),y1=cy+r*Math.sin(angle),x2=cx+r*Math.cos(next),y2=cy+r*Math.sin(next);s.append(svgNode('path',{d:`M ${cx} ${cy} L ${x1} ${y1} A ${r} ${r} 0 ${sweep>Math.PI?1:0} 1 ${x2} ${y2} Z`,fill:COLORS[i%COLORS.length]}))}angle=next;
  }
  arr.forEach((e,i)=>{const y=60+i*28;s.append(svgNode('rect',{x:410,y:y-12,width:12,height:12,rx:3,fill:COLORS[i%COLORS.length]}));s.append(svgNode('text',{x:431,y,'font-size':12,fill:'#416757'},`${String(e.label).slice(0,16)} — ${formatted(e.value)}`))});
 } else {
  let max=Math.max(...arr.map(e=>e.value),1),left=125,right=686,top=56,bottom=315;
  for(let j=0;j<=4;j++){let y=bottom-(bottom-top)*j/4;s.append(svgNode('line',{x1:left,y1:y,x2:right,y2:y,stroke:'#e5eee8'}));s.append(svgNode('text',{x:left-11,y:y+4,'font-size':11,'text-anchor':'end',fill:'#718b7f'},formatted(max*j/4)))}
  if(type==='line'){
   let coords=arr.map((e,i)=>[left+10+(right-left-20)*(i+(arr.length===1?0.5:0))/(Math.max(1,arr.length-1)),bottom-e.value/max*(bottom-top-12)]);
   s.append(svgNode('polyline',{points:coords.map(a=>a.join(',')).join(' '),fill:'none',stroke:'#30836a','stroke-width':3}));coords.forEach((c,i)=>{s.append(svgNode('circle',{cx:c[0],cy:c[1],r:5,fill:'#30836a'}));s.append(svgNode('text',{x:c[0],y:c[1]-12,'text-anchor':'middle','font-size':10,fill:'#316c59'},formatted(arr[i].value)))});
   coords.forEach((c,i)=>s.append(svgNode('text',{x:c[0],y:bottom+18,'text-anchor':'middle','font-size':11,fill:'#547b6a'},String(arr[i].label).slice(0,6))))
  }else{
   const gap=(right-left)/arr.length,width=Math.min(48,gap*.72);arr.forEach((e,i)=>{let ht=(Number(e.value)/max)*(bottom-top-10),x=left+gap*i+(gap-width)/2,y=bottom-ht;s.append(svgNode('rect',{x,y,width,height:ht,rx:4,fill:COLORS[i%COLORS.length]}));s.append(svgNode('text',{x:x+width/2,y:y-9,'text-anchor':'middle','font-size':11,fill:'#316c59'},formatted(e.value)));s.append(svgNode('text',{x:x+width/2,y:bottom+18,'text-anchor':'middle','font-size':11,fill:'#547b6a'},String(e.label).slice(0,7)))});
  }
 }
 s.append(svgNode('text',{x:24,y:350,'font-size':10,fill:'#80968a'},'원자료 검증 및 출처 확인 필요 · 실습 결과'));host.append(s);
}
function optionList(sel,items){clear(sel);for(let [val,name] of items){let o=document.createElement('option');o.value=val;o.textContent=name;sel.append(o)}}
function demoFile(name,content){return new File([content],name,{type:'text/csv;charset=utf-8'})}
function demoTraffic(){let rows=['연도,교통수단,이용건수,지역,승인여부'];const types=['공공자전거','시내버스','지하철'];for(let year=2021;year<=2026;year++){types.forEach((t,i)=>rows.push(`${year},${t},${Math.round((year-2020)*(i+2)*2300+i*1700)},${['서울','부산','인천'][i]},가상`))}return demoFile('교통수단_가상샘플.csv',rows.join('\n'))}
function demoNews(){let rows=['날짜,분야,지역,기사제목,조회수,유형'];const topics={교통:['공공자전거 이용 증가','버스 노선 개편 논의','지하철 혼잡도 변화'],경제:['점심값 부담 확대','전통시장 소비 증가','지역 물가 비교'],문화:['청년 공연 관람 확대','지역 축제 관객 증가','작은 도서관 운영']};for(let i=0;i<180;i++){let type=Object.keys(topics)[i%3],title=topics[type][Math.floor(i/3)%3];rows.push(`2026-11-${String(i%28+1).padStart(2,'0')},${type},${['서울','부산','대전','광주'][i%4]},${title},${80+(i*37)%800},가상`)}return demoFile('가상_뉴스_180건.csv',rows.join('\n'))}
let vFile=null,bFile=null,tExamples=null;let visType=null;
let stagedCrawlFile=null,crawlRequestId=0; // Explicit two-step handoff; latest crawl request wins.
async function runVisual(){try{const val=await sendFile('/api/analyze',vFile||$id('visFile').files[0]);vData=val;renderVisual();pop(`${val.totalRows}개 행의 통계를 계산했습니다.`)}catch(e){error(e)}}
function renderVisual(){if(!vData)return;const cols=vData.columns;
 kpis($id('visStats'),[['데이터 행',formatted(vData.totalRows)],['변수 수',cols.length],['결측 셀',formatted(cols.reduce((s,c)=>s+c.missing,0))]]);
 let options=cols.map(c=>[c.name,c.name]);const selected=$id('visColumn').value;optionList($id('visColumn'),options);if(cols.some(c=>c.name===selected))$id('visColumn').value=selected;
 optionList($id('visMetric'),[['count','범주별 개수']]);renderVChart();
 table($id('visTable'),['열','유효 값','결측','고유값','평균','중앙값','최소','최대'],cols.map(c=>[c.name,formatted(c.count),c.missing,c.unique,c.numericCount?formatted(c.mean):'—',c.numericCount?formatted(c.median):'—',c.numericCount?formatted(c.min):'—',c.numericCount?formatted(c.max):'—']));
}
function renderVChart(){if(!vData)return;let column=vData.columns.find(c=>c.name===$id('visColumn').value);if(!column)return;const vals=column.top.map(e=>({label:e.value,value:e.count}));drawChart($id('visChart'),vals,$id('visType').value,`${column.name} · 값별 빈도`)}
$id('visRun').onclick=runVisual;$id('visDemo').onclick=()=>{vFile=demoTraffic();runVisual()};$id('visFile').onchange=()=>{vFile=null};$id('visColumn').onchange=renderVChart;$id('visType').onchange=renderVChart;$id('visMetric').onchange=renderVChart;
$id('visSvg').onclick=()=>{let v=$('#visChart svg');if(!v)return pop('먼저 차트를 생성하세요.');save('통계시각화_가상데이터.svg',new XMLSerializer().serializeToString(v),'image/svg+xml;charset=utf-8')};$id('visSummary').onclick=()=>vData?jsonSave('통계분석_결과.json',vData):pop('먼저 분석하세요.');
async function runBig(){try{bData=await sendFile('/api/analyze',bFile||$id('bigFile').files[0]);renderBig();pop(`${bData.totalRows}건 데이터 분석 완료`)}catch(e){error(e)}}
function renderBig(){if(!bData)return;const cols=bData.columns;kpis($id('bigStats'),[['뉴스 건수',formatted(bData.totalRows)],['필드 수',cols.length],['결측 수',formatted(cols.reduce((s,c)=>s+c.missing,0))]]);
 let saved=$id('bigGroup').value,savedText=$id('bigText').value;optionList($id('bigGroup'),cols.map(c=>[c.name,c.name]));optionList($id('bigText'),cols.map(c=>[c.name,c.name]));
 $id('bigGroup').value=cols.some(c=>c.name===saved)?saved:(cols.find(c=>/분야|지역|category|section|type/i.test(c.name))||cols[0]).name;
 $id('bigText').value=cols.some(c=>c.name===savedText)?savedText:(cols.find(c=>/제목|본문|title|text/i.test(c.name))||cols[0]).name;
 renderBigDetail();table($id('bigTable'),['열','고유값','결측','수치형 개수'],cols.map(c=>[c.name,c.unique,c.missing,c.numericCount]));
}
function renderBigDetail(){if(!bData)return;const group=bData.columns.find(c=>c.name===$id('bigGroup').value),t=bData.columns.find(c=>c.name===$id('bigText').value);if(group)drawChart($id('bigChart'),group.top.map(x=>({label:x.value,value:x.count})),'bar',`${group.name} · 기사 건수`);let host=$id('bigTerms');clear(host);for(let x of (t?t.terms:[]).slice(0,14))host.append(kv('span',`${x.value} ${x.count}`,'kw'));if(!host.children.length)host.append(kv('p','키워드 추출 결과 없음','hint'))}
$id('bigDemo').onclick=()=>{bFile=demoNews();runBig()};$id('bigRun').onclick=runBig;$id('bigFile').onchange=()=>{bFile=null};$id('bigGroup').onchange=renderBigDetail;$id('bigText').onchange=renderBigDetail;$id('bigExport').onclick=()=>bData?jsonSave('뉴스빅데이터_분석결과.json',bData):pop('먼저 데이터를 분석하세요.');
$id('bigToVis').onclick=()=>{const f=bFile||$id('bigFile').files[0];if(!f)return pop('분석 데이터를 먼저 선택하세요.');vFile=f;document.querySelector('.nav[data-page="visualize"]').click();runVisual()};
async function crawl(demo){
 const currentRequest=++crawlRequestId;
 // Never reuse old crawl results after a new request or failed URL. Keep the user on the crawl tab.
 cData=null;stagedCrawlFile=null;$id('crawlOpenBig').disabled=true;
 $id('crawlTransferStatus').textContent='새 수집을 시작했습니다. 이전 전달 준비는 취소되었습니다.';
 $id('crawlInfo').textContent='수집 중...';clear($id('crawlOutput'));
 try{
  const req=demo?{demo:true}:{url:$id('crawlURL').value.trim(),maxPages:Number($id('crawlPages').value)};
  const result=await sendJSON('/api/crawl',req);
  if(currentRequest!==crawlRequestId)return; // An older request must not overwrite a newer result.
  if(!Array.isArray(result.pages)||!result.pages.length)throw Error('수집한 페이지가 없습니다.');
  cData=result;
  $id('crawlInfo').textContent=`${cData.pages.length}개 페이지 추출 · ${cData.note}`;
  const out=$id('crawlOutput');
  for(let p of cData.pages){const card=kv('div','','crawlItem');card.append(kv('small',p.url),kv('h3',p.title||p.heading||'제목 없음'),kv('p',p.text||'본문이 추출되지 않았습니다.'));out.append(card)}
  $id('crawlTransferStatus').textContent='수집 완료. JSONL 저장 또는 빅데이터 분석용 데이터 준비를 선택하세요. 화면은 이동하지 않습니다.';
  pop(`${cData.pages.length}개 페이지 처리 완료 · 크롤링 화면 유지`)
 }catch(e){if(currentRequest!==crawlRequestId)return;$id('crawlInfo').textContent=`수집 실패: ${e.message||String(e)}`;error(e)}
}
$id('crawlRun').onclick=()=>crawl(false);$id('crawlDemo').onclick=()=>crawl(true);
$id('crawlSave').onclick=()=>cData?save('웹수집_교육용.jsonl',cData.pages.map(p=>JSON.stringify(p)).join('\n')+'\n','application/x-ndjson;charset=utf-8'):pop('수집 결과가 없습니다.');
$id('crawlToBig').onclick=()=>{
 if(!cData)return pop('먼저 웹 자료를 수집하세요.');
 // Stage only. Navigation is a separate explicit action so that crawler results remain visible.
 stagedCrawlFile=new File([cData.pages.map(p=>JSON.stringify(p)).join('\n')+'\n'],'웹수집_교육용.jsonl',{type:'application/x-ndjson'});
 $id('crawlOpenBig').disabled=false;
 $id('crawlTransferStatus').textContent=`${cData.pages.length}개 페이지의 JSONL을 분석용으로 준비했습니다. 현재 화면은 그대로 유지됩니다. 오른쪽의 ‘빅데이터 화면에서 분석 실행’을 누르면 이동합니다.`;
 pop('분석용 데이터 준비 완료 · 화면 유지');
};
$id('crawlOpenBig').onclick=()=>{
 if(!stagedCrawlFile||!cData)return pop('크롤링 후 분석용 데이터를 먼저 준비하세요.');
 bFile=stagedCrawlFile;
 document.querySelector('.nav[data-page="bigdata"]').click();
 runBig();
};
function parseCSV(input){let rows=[],row=[],v='',quoted=false;for(let i=0;i<input.length;i++){let c=input[i];if(c==='"'){if(quoted&&input[i+1]==='"'){v+='"';i++}else quoted=!quoted}else if(c===','&&!quoted){row.push(v);v=''}else if((c==='\n'||c==='\r')&&!quoted){if(c==='\r'&&input[i+1]==='\n')i++;row.push(v);v='';if(row.some(e=>e.trim()))rows.push(row);row=[]}else v+=c}if(v||row.length){row.push(v);rows.push(row)}return rows}
function trainingDemo(){let out=[],labels={교통:['자전거 이용이 늘었습니다','버스 정류장 혼잡을 살핍니다','지하철 승객이 증가했습니다','교통량 변화가 나타났습니다','시내버스 환승이 편해졌습니다','통근 시간대 승차 인원 조사'],경제:['점심값 상승을 조사합니다','물가 부담이 커집니다','소비자 지출이 늘었습니다','시장 매출이 회복됐습니다','지역 경제 지표를 분석합니다','상점 가격이 오르고 있습니다'],문화:['공연 관람객이 늘었습니다','지역 축제가 열렸습니다','영화 관객 수를 조사합니다','박물관 전시가 인기를 모읍니다','음악회 관람 열기가 높습니다','예술 공연의 표가 팔립니다']};
 for(let label of Object.keys(labels)){let list=labels[label];for(let i=0;i<20;i++)out.push({label,text:`${list[i%list.length]} ${['오늘','이번 달','올해','지역에서'][i%4]}`})}return out}
async function loadTrainFile(){let f=$id('trainFile').files[0];if(!f)throw Error('학습 CSV를 선택하세요.');let rows=parseCSV(await f.text()),header=(rows.shift()||[]).map(s=>s.replace(/^\uFEFF/,'').trim().toLowerCase());let li=header.indexOf('label'),ti=header.indexOf('text');if(li<0||ti<0)throw Error('label,text 헤더가 필요합니다.');return rows.filter(r=>r[li]&&r[ti]).map(r=>({label:r[li].trim(),text:r[ti].trim()}))}
async function runTrain(){try{if(!tExamples)tExamples=await loadTrainFile();tData=await sendJSON('/api/train',{examples:tExamples,predict:$id('predictText').value});const b=tData.baseline.test,tt=tData.tuned.test;kpis($id('trainStats'),[['훈련 행',tData.split.train],['검증 행',tData.split.validation],['시험 행',tData.split.test],['기준 정확도',b.accuracy+'%'],['튜닝 정확도',tt.accuracy+'%']]);
 drawChart($id('trainChart'),[{label:'기준 모델',value:b.accuracy},{label:'튜닝 모델',value:tt.accuracy}],'bar','미사용 테스트 데이터 정확도 (%)');
 table($id('trainDetails'),['구분','설정','검증 정확도','시험 정답'],[['기준',tData.baseline.parameter,tData.baseline.validation.accuracy+'%',`${b.correct}/${b.total}`],['튜닝',tData.tuned.parameter,tData.tuned.validation.accuracy+'%',`${tt.correct}/${tt.total}`],['예측',tData.prediction||'입력 없음','—','—']]);pop('실제 분류 모델 학습과 비교를 마쳤습니다.')
 }catch(e){error(e)}}
$id('trainDemo').onclick=()=>{tExamples=trainingDemo();pop(`${tExamples.length}개 가상 뉴스 분류 샘플을 불러왔습니다.`)};$id('trainRun').onclick=runTrain;$id('trainFile').onchange=()=>{tExamples=null};$id('trainExport').onclick=()=>tData?jsonSave('모델학습_평가결과.json',tData):pop('학습을 먼저 실행하세요.');$id('trainNotebook').onclick=()=>{document.querySelector('.nav[data-page="notebooks"]').click();setTimeout(()=>{const rows=[...document.querySelectorAll('#notebookTable .notebookRow')],target=rows.find(x=>x.textContent.includes('train_unsloth'));if(target){target.style.outline='3px solid #7ab195';target.scrollIntoView({behavior:'smooth',block:'center'})}else pop('원본 목록에서 2_train_unsloth_gpu.ipynb를 찾아 실행하세요.')},250)};
function calcAlert(){let subj=$id('alertSubject').value.trim(),raw=[$id('alertPrevious').value,$id('alertCurrent').value,$id('alertThreshold').value],p=Number(raw[0]),c=Number(raw[1]),th=Number(raw[2]);if(!subj||raw.some(s=>!s.trim())||![p,c,th].every(Number.isFinite)||th<=0)throw Error('지표명과 유효한 기준값을 입력하세요.');let change=c-p,trigger=Math.abs(change)>=th,summary=`[가상 데이터 / 미검증 / 자동 게시 금지]\n${subj} ${formatted(p)}% → ${formatted(c)}%, ${change>=0?'+':''}${formatted(change)}%p 변화.\n${trigger?'경보 기준에 도달해 사람이 확인해야 합니다.':'지정한 경보 기준 미달입니다.'}\n한 줄 정리: 숫자를 원자료에서 확인한 뒤 기사화 여부를 판단합니다.`;lastAlert={subject:subj,previous:p,current:c,threshold:th,change,trigger,summary,notice:'FICTIONAL TRAINING DATA / HOLD'};let el=$id('alertOutput');el.textContent=summary;el.classList.toggle('trigger',trigger);return trigger}
$id('alertRun').onclick=()=>{try{pop(calcAlert()?'경보 조건 충족 · 출고 보류':'경보 기준 미달')}catch(e){error(e)}};
$id('alertDemo').onclick=()=>{let curr=Number($id('alertCurrent').value);$id('alertCurrent').value=curr===57?46:57;$id('alertRun').click()};$id('alertSave').onclick=()=>lastAlert?jsonSave('속보시뮬레이션_미검증.json',lastAlert):pop('먼저 속보를 실행하세요.');
let recognition=null;const constructor=window.SpeechRecognition||window.webkitSpeechRecognition;
$id('micStart').onclick=()=>{if(!constructor){pop('이 브라우저에서 음성 인식이 지원되지 않습니다. Edge/Chrome에서 확인하세요.');return}try{if(recognition)recognition.stop();recognition=new constructor();recognition.lang='ko-KR';recognition.interimResults=true;recognition.continuous=true;let fixed=$id('transcript').value;recognition.onresult=e=>{let interim='',final='';for(let i=0;i<e.results.length;i++){const line=e.results[i][0].transcript;if(e.results[i].isFinal)final+=line+' ';else interim+=line};$id('transcript').value=fixed+'\n'+final+interim};recognition.onerror=e=>pop(`마이크 오류: ${e.error}`);recognition.start();pop('마이크 음성 인식을 시작했습니다.')}catch(e){error(e)}};
$id('micStop').onclick=()=>{if(recognition){recognition.stop();recognition=null}pop('마이크 인식을 중지했습니다.')};
$id('audioRun').onclick=async()=>{try{let file=$id('audioFile').files[0];if(!file)throw Error('녹음 파일을 선택하세요.');const res=await sendFile('/api/transcribe',file);$id('transcript').value=[ $id('transcript').value,res.text||'' ].filter(Boolean).join('\n');pop('로컬 전사가 완료되었습니다.')}catch(e){error(e)}};
$id('transcriptSave').onclick=()=>save('음성받아쓰기_검토필요.txt',`[자동 음성 인식 결과: 사람이 재검토해야 합니다]\n\n${$id('transcript').value}`);
$id('startJupyterBtn').onclick=async()=>{if(window.NEWS_STANDALONE)return pop('Windows EXE에서만 원본 Jupyter를 실행할 수 있습니다.');try{const r=await fetch('/api/start-jupyter',{method:'POST'});if(!r.ok)throw Error(await r.text());pop('Jupyter 실행을 요청했습니다. 별도 창을 확인하세요.')}catch(e){error(e)}};
})();
