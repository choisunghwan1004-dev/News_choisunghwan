/* 뉴스 스튜디오: 가상 데이터 전용, 외부 API 없이 작동하는 규칙 기반 원고 엔진.
   AI 모델, 사실 검증, 영상 렌더링 기능을 가장하지 않습니다. */
(function(global){
'use strict';
const STORIES = {
 bike: {label:'따릉이',group:'도시·교통',fields:[{key:'rides',label:'연간 이용 건수 (건)',value:45000000,min:1,max:10000000000},{key:'ratio',label:'3년 전 대비 (배)',value:1.5,min:0.01,max:100}],title:'따릉이 이용 건수 변화',emoji:'🚲'},
 lunch:{label:'직장인 점심값',group:'생활·물가',fields:[{key:'price',label:'현재 평균 점심값 (원)',value:11000,min:1,max:100000000},{key:'increase',label:'5년 전보다 상승 (%)',value:32,min:-99,max:10000}],title:'직장인 평균 점심값',emoji:'🍱'},
 culture:{label:'공연 관람 20대 비율',group:'문화·세대',fields:[{key:'share',label:'전체 관람객 중 20대 (%)',value:41,min:0,max:100}],title:'20대 공연 관람객 비율',emoji:'🎭'}
};
const STYLES={standard:'정통 뉴스',daily:'생활 밀착',explainer:'데이터 해설'};
const TIMINGS={30:[0,6,14,23,30],60:[0,8,20,32,44,53,60],90:[0,10,24,38,52,66,78,90]};
function num(n){return Math.round(n).toLocaleString('ko-KR')}
function count(n){return n>=1000000 && Math.abs(n%10000)<.001?`${num(n/10000)}만`:num(n)}
function strNum(n){return Number(n).toLocaleString('ko-KR',{maximumFractionDigits:2})}
function sensible(v,f){
 const n=Number(v===undefined?f.value:v);
 return Number.isFinite(n)&&n>=f.min&&n<=f.max?n:f.value;
}
function defaults(story){const s=STORIES[story]||STORIES.bike;return Object.fromEntries(s.fields.map(f=>[f.key,f.value]))}
function payload(story,values={}){
 let d=Object.fromEntries(STORIES[story].fields.map(f=>[f.key,sensible(values[f.key],f)]));
 const text={};
 if(story==='bike'){
   const old=d.rides/d.ratio;
   text.stat=`공공자전거 따릉이의 연간 이용 건수는 ${count(d.rides)}건입니다. 이 숫자는 실제 발표치가 아닌 실습용 가상 데이터입니다.`;
   text.compare=`3년 전과 비교하면 ${strNum(d.ratio)}배입니다. 3년 전 이용 건수는 약 ${count(Math.round(old))}건으로 계산됩니다.`;
   text.method=`이용 건수는 자전거를 이용한 횟수입니다. 이용한 사람의 수와는 다릅니다. 집계 연도와 계산 기준도 확인해야 합니다.`;
   text.meaning=`가상 통계대로라면 3년 사이 이용 횟수가 변했다는 뜻입니다. 하지만 이 숫자만으로 이용 이유까지 설명할 수는 없습니다.`;
   text.caution=`왜 늘거나 줄었는지는 확인되지 않았습니다. 실제 기사로 쓰려면 운영기관의 원자료부터 대조해야 합니다.`;
   text.sum=`한 줄 정리. 따릉이 이용은 3년 전의 ${strNum(d.ratio)}배라는 가상 통계입니다.`;
   text.chart=[{name:'3년 전',value:old,label:`약 ${count(Math.round(old))}`},{name:'현재',value:d.rides,label:count(d.rides)}];
   text.caption=`3년 전 대비 ${strNum(d.ratio)}배`;text.main=`${count(d.rides)}|건`;
   text.hooks={standard:'서울 공공자전거 이용은 얼마나 달라졌을까요?',daily:'출퇴근길에 보이는 따릉이, 얼마나 많이 이용할까요?',explainer:'오늘은 따릉이 이용 건수를 숫자로 비교해 보겠습니다.'};
 }else if(story==='lunch'){
   const old=d.price/(1+d.increase/100);const diff=d.price-old;
   const direction=d.increase>=0?'올랐습니다':'내렸습니다';const abs=Math.abs(d.increase);
   text.stat=`직장인 평균 점심값은 ${num(d.price)}원입니다. 이 수치는 실제 조사 결과가 아닌 가상 데이터입니다.`;
   text.compare=`5년 전보다 ${strNum(abs)}퍼센트 ${direction} 5년 전 점심값은 약 ${num(old)}원으로 계산됩니다.`;
   text.method=`여기서 평균은 여러 점심값을 합쳐 나눈 값입니다. 지역과 조사 대상에 따라 실제 체감 가격은 달라질 수 있습니다.`;
   text.meaning=`계산상 5년 전과의 차이는 약 ${num(Math.abs(diff))}원입니다. 다만 식당 종류나 조사 방식은 아직 확인되지 않았습니다.`;
   text.caution=`어느 지역의 어떤 직장인을 조사했는지가 중요합니다. 출처 확인 전에는 실제 물가 기사로 보도하면 안 됩니다.`;
   text.sum=`한 줄 정리. 점심값 ${num(d.price)}원, 5년 전보다 ${strNum(abs)}퍼센트 ${direction} 단, 가상 수치입니다.`;
   text.chart=[{name:'5년 전',value:old,label:num(old)+'원'},{name:'현재',value:d.price,label:num(d.price)+'원'}];
   text.caption=`5년 전 대비 ${strNum(abs)}% ${direction.replace('습니다','')}`;text.main=`${num(d.price)}|원`;
   text.hooks={standard:'직장인들의 점심 한 끼 가격, 얼마나 달라졌을까요?',daily:'점심 먹고 계산할 때, 가격이 부담스럽다고 느끼시나요?',explainer:'평균 점심값과 5년간 변화율을 숫자로 보겠습니다.'};
 }else{
   text.stat=`국내 공연 관람객 중 20대 비율은 ${strNum(d.share)}퍼센트입니다. 오늘 사용하는 수치는 가상 데이터입니다.`;
   text.compare=`이 비율이 역대 최고라는 조건입니다. 다만 비교 대상 기간과 과거 연도별 수치는 제공되지 않았습니다.`;
   text.method=`41퍼센트는 전체 관람객 중 20대가 차지하는 비율입니다. 공연 횟수나 전체 인구의 41퍼센트라는 뜻은 아닙니다.`.replace('41',strNum(d.share));
   text.meaning=`20대 비중이 눈에 띄는 가상 사례입니다. 어떤 장르에서 높은지, 관람객을 어떻게 집계했는지는 별도 자료가 필요합니다.`;
   text.caution=`역대 최고라는 표현은 과거 자료를 모두 비교해야 확인할 수 있습니다. 원출처 검증 전 실제 보도를 보류합니다.`;
   text.sum=`한 줄 정리. 20대 비율 ${strNum(d.share)}퍼센트, 역대 최고라는 가상 조건입니다.`;
   text.chart=[{name:'20대',value:d.share,label:strNum(d.share)+'%'},{name:'그 외',value:100-d.share,label:strNum(100-d.share)+'%'}];
   text.caption=`관람객 중 20대 ${strNum(d.share)}%`;text.main=`${strNum(d.share)}|%`;
   text.hooks={standard:'국내 공연 관람객의 연령 비중이 달라졌다는 가상 통계입니다.',daily:'요즘 공연장, 20대 관객은 얼마나 될까요?',explainer:'공연 관람객의 연령 비중을 데이터로 살펴봅니다.'};
 }
 return {d,text};
}
function makeScript(story='bike',duration=60,style='standard',values={}){
 if(!STORIES[story])throw new Error('유효하지 않은 주제');
 if(!TIMINGS[duration])throw new Error('길이는 30·60·90초 중 하나여야 합니다.');
 if(!STYLES[style])throw new Error('알 수 없는 전달 방식');
 const p=payload(story,values),t=p.text,points=TIMINGS[duration];
 const scenes={bike:['시민과 자전거가 있는 거리','따릉이 사진과 큰 숫자','과거와 현재 비교 차트','통계 기준 표시','자료 검증 체크리스트','가상 데이터 고지','한 줄 요약 카드'],lunch:['직장인 점심시간 풍경','점심 메뉴와 가격 자막','5년 전과 현재 비교 차트','평균의 의미 설명','조사 범위 표시','원출처 확인 안내','한 줄 요약 카드'],culture:['공연장 외관 또는 저작권 확인된 이미지','20대 비율 큰 숫자','연령 비중 그래픽','분모·분자 설명','집계 기준 확인','과거 자료 검증 안내','한 줄 요약 카드']}[story];
 const items = duration===30?[['hook',0],['stat',1],['compare',2],['sum',6]]:duration===60?[['hook',0],['stat',1],['compare',2],['meaning',4],['caution',5],['sum',6]]:[['hook',0],['stat',1],['compare',2],['method',3],['meaning',4],['caution',5],['sum',6]];
 const lines=items.map(([k,sceneIndex],i)=>{
   let body=k==='hook'?t.hooks[style]:t[k];
   if(style==='daily'&&k==='meaning') body=`우리 생활과 이어지는 숫자죠. ${body}`;
   if(style==='explainer'&&k==='caution')body=`데이터를 해석하기 전에 확인할 점입니다. ${body}`;
   if(duration===30&&k==='stat')body=body.replace(' 이 숫자는 실제 발표치가 아닌 실습용 가상 데이터입니다.','').replace(' 이 수치는 실제 조사 결과가 아닌 가상 데이터입니다.','').replace(' 오늘 사용하는 수치는 가상 데이터입니다.','');
   if(duration===30&&k==='compare'&&story==='culture')body='가상 데이터에서는 역대 최고라고 설정했습니다. 실제 비교 자료는 아직 확인하지 않았습니다.';
   return {start:points[i],end:points[i+1],scene:scenes[sceneIndex],caption:k==='hook'?STORIES[story].label:k==='stat'?t.caption:k==='sum'?'한 줄 정리 · 가상 통계':k==='caution'?'출처 확인 전 보도 보류':k==='method'?'집계 기준 확인':k==='compare'?'수치 비교': '해석과 의미',narration:body};
 });
 return {story,duration,style,topic:STORIES[story].label,topicTitle:STORIES[story].title,styleName:STYLES[style],values:p.d,stat:t.main,subtitle:t.caption,charts:t.chart,oneLine:t.sum,segments:lines,isFictional:true,releaseStatus:'HOLD',sourceVerified:false,rightsVerified:false,approved:false};
}
function timecode(n,srt=false){
 const h=String(Math.floor(n/3600)).padStart(2,'0'),m=String(Math.floor(n/60)%60).padStart(2,'0'),s=String(Math.floor(n%60)).padStart(2,'0');
 return srt?`${h}:${m}:${s},000`:`${m}:${s}`;
}
function toTxt(doc){
 let out=`[가상 데이터 / 교육용] ${doc.topicTitle}\n조건: ${doc.duration}초 / ${doc.styleName}\n출처: 확인되지 않음 — 실제 보도 보류\n\n`;
 doc.segments.forEach(s=>{out+=`[${timecode(s.start)}–${timecode(s.end)}] 화면: ${s.scene}\n자막: ${s.caption}\n나레이션: ${s.narration}\n\n`});
 out+=`한 줄 정리: ${doc.oneLine}\n\n[편집·윤리 주의]\n- 가상 데이터이므로 사실로 보도할 수 없습니다.\n- 숫자·인용은 최초 공식 출처와 교차 확인하십시오.\n- 사진·음악·목소리 권리 및 사용 허락을 확인하십시오.\n- AI 초안을 사용한 경우 최종 검토 후 'AI 초안을 기자가 확인·수정' 표시를 적용하십시오.\n- 오류 정정 시 '바로잡습니다'를 남기십시오.\n`;return out;
}
function toSrt(doc){return doc.segments.map((s,i)=>`${i+1}\n${timecode(s.start,true)} --> ${timecode(s.end,true)}\n${s.narration}\n`).join('\n')}
function allCombinations(){let out=[];for(let s of Object.keys(STORIES))for(let d of [30,60,90])for(let style of Object.keys(STYLES))out.push(makeScript(s,d,style));return out}
const api={STORIES,STYLES,defaults,makeScript,toTxt,toSrt,timecode,allCombinations};
if(typeof module!=='undefined'&&module.exports)module.exports=api;
global.NewsEngine=api;
})(typeof window!=='undefined'?window:globalThis);
