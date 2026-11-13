const assert=require('node:assert/strict');
const E=require('../src/web/engine.js');
const cases=E.allCombinations();
assert.equal(cases.length,27);
for(const d of cases){
 assert.equal(d.segments[0].start,0);
 assert.equal(d.segments.at(-1).end,d.duration);
 assert.equal(d.isFictional,true);
 assert.equal(d.releaseStatus,'HOLD');
 assert.match(E.toTxt(d),/가상 데이터/);
 assert.equal(E.toSrt(d).split('-->').length-1,d.segments.length);
 for(let i=0;i<d.segments.length;i++){
  assert.ok(d.segments[i].end>d.segments[i].start);
  if(i) assert.equal(d.segments[i].start,d.segments[i-1].end);
 }
}
assert.match(E.toTxt(E.makeScript('bike',60)),/4,500만건/);
assert.match(E.toTxt(E.makeScript('bike',60)),/3,000만건/);
assert.match(E.toTxt(E.makeScript('lunch',60)),/11,000원/);
assert.match(E.toTxt(E.makeScript('lunch',60)),/8,333원/);
assert.match(E.toTxt(E.makeScript('culture',60)),/41퍼센트/);
assert.notEqual(E.toTxt(E.makeScript('lunch',60,'standard',{price:10000})),E.toTxt(E.makeScript('lunch',60,'standard')));
let changed=E.makeScript('culture',60,'daily',{share:35});
assert.match(E.toTxt(changed),/35퍼센트/);
assert.equal(E.makeScript('bike',30).segments.length,4);
console.log('PASS: 27 variants, fictional status, timeline, calculations, editability, export');
