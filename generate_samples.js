const fs=require('node:fs');const path=require('node:path');const E=require('./src/web/engine.js');const out=path.join(__dirname,'samples');fs.mkdirSync(out,{recursive:true});
let manifest=['file,topic,duration,style,fictional,publication_status'];
let grouped=[];
for(const doc of E.allCombinations()){
 const name=`sample_${doc.story}_${String(doc.duration)}s_${doc.style}.txt`;
 fs.writeFileSync(path.join(out,name),'\ufeff'+E.toTxt(doc),'utf8');
 manifest.push(`${name},${doc.topic},${doc.duration},${doc.styleName},TRUE,HOLD`);
 grouped.push(`\n===== ${doc.topic} / ${doc.duration}초 / ${doc.styleName} =====\n${E.toTxt(doc)}`);
}
fs.writeFileSync(path.join(out,'all_27_samples.txt'),'\ufeff'+grouped.join('\n'),'utf8');
fs.writeFileSync(path.join(out,'sample_conditions.csv'),'\ufeff'+manifest.join('\r\n'),'utf8');
fs.writeFileSync(path.join(out,'three_60s_default_samples.json'),JSON.stringify(['bike','lunch','culture'].map(s=>E.makeScript(s,60,'daily')),null,2),'utf8');
console.log(`생성: 27가지 원고 + 전체 묶음 + 조건표 + 구조화 JSON`)
