"""Optional UI regression test: python -m pip install playwright; python tests/ui_navigation_regression.py

Renders the real HTML/CSS/JS in headless Chromium without external-network access.
Mocks only API transport. Back-end contracts are verified separately by Go tests.
"""
from pathlib import Path
from playwright.sync_api import sync_playwright
import re

WEB=Path(__file__).resolve().parents[1]/'src'/'web'
html=(WEB/'index.html').read_text(encoding='utf-8')
html=re.sub(r'<script src="/[^\"]+"\s*></script>','',html)
html=html.replace('<link rel="stylesheet" href="/style.css">','')

mock_api=r'''() => {
 window.__calls=[];
 const good=data=>({ok:true,status:200,json:async()=>data});
 const bad=(status,error)=>({ok:false,status,json:async()=>({error})});
 const terms=[{value:'관람객',count:4},{value:'공연',count:3}];
 window.fetch=async (endpoint,opts={})=>{
   window.__calls.push(endpoint);
   if(endpoint==='/api/crawl'){
     const req=JSON.parse(opts.body);
     if(!req.demo)return bad(400,'검증용 URL 접속 실패');
     return good({pages:[{url:'demo://fictional',title:'가상 뉴스',heading:'교육용 실습',text:'가상 뉴스 본문입니다.'}],note:'오프라인 샘플'});
   }
   if(endpoint==='/api/analyze')return good({totalRows:12,columns:[
      {name:'분야',count:12,missing:0,unique:2,numericCount:0,top:[{value:'경제',count:8},{value:'문화',count:4}],terms:[]},
      {name:'기사제목',count:12,missing:0,unique:3,numericCount:0,top:[{value:'공연 관람객',count:4}],terms},
      {name:'조회수',count:12,missing:0,unique:10,numericCount:12,mean:45,median:42,min:8,max:98,top:[{value:'42',count:2}],terms:[]}
   ]});
   if(endpoint==='/api/train')return good({split:{train:30,validation:10,test:10},baseline:{parameter:'a=1',validation:{accuracy:80},test:{accuracy:70,correct:7,total:10}},tuned:{parameter:'a=0.1',validation:{accuracy:90},test:{accuracy:90,correct:9,total:10}},prediction:'교통'});
   if(endpoint==='/api/labs')return good({notebooks:[]});
   return bad(404,'unexpected endpoint');
 };
}'''

def current(page):return page.locator('.page.show').get_attribute('id')

def check():
 with sync_playwright() as p:
  browser=p.chromium.launch(headless=True,executable_path='/usr/bin/chromium',args=['--no-sandbox'])
  page=browser.new_page(viewport={'width':1280,'height':840},accept_downloads=True)
  errors=[]
  page.on('pageerror', lambda e: errors.append(str(e)))
  page.set_content(html)
  page.add_style_tag(content=(WEB/'style.css').read_text(encoding='utf-8'))
  page.evaluate(mock_api)
  for name in ('engine.js','app.js','labs.js'):
   page.add_script_tag(content=(WEB/name).read_text(encoding='utf-8'))
  assert current(page)=='editor'
  assert page.locator('#sampleGrid .sampleCard').count()==27
  # Editor state is preserved across all lab menus.
  page.locator('.story[data-id="lunch"]').click()
  assert '점심' in page.locator('#resultTitle').inner_text()
  # Crawl must stay on its own tab. Import and navigation are two separate actions.
  page.locator('.nav[data-page="crawl"]').click()
  assert current(page)=='crawl' and not page.locator('#crawlOpenBig').is_enabled()
  page.locator('#crawlDemo').click()
  page.wait_for_function("document.querySelector('#crawlInfo').textContent.includes('1개 페이지')")
  assert current(page)=='crawl'
  assert page.locator('.crawlItem').count()==1
  assert page.evaluate("window.__calls.filter(x=>x==='/api/analyze').length")==0
  page.locator('#crawlToBig').click()
  assert current(page)=='crawl' and page.locator('#crawlOpenBig').is_enabled()
  assert page.evaluate("window.__calls.filter(x=>x==='/api/analyze').length")==0
  # A failed new crawl must invalidate the stale result and prepared handoff.
  page.locator('#crawlURL').fill('https://unavailable.example/news')
  page.locator('#crawlRun').click()
  page.wait_for_function("document.querySelector('#crawlInfo').textContent.includes('실패')")
  assert current(page)=='crawl'
  assert page.locator('.crawlItem').count()==0
  assert not page.locator('#crawlOpenBig').is_enabled()
  page.locator('#crawlToBig').click()
  assert not page.locator('#crawlOpenBig').is_enabled()
  # Restart and intentionally continue to big data.
  page.locator('#crawlDemo').click()
  page.wait_for_function("document.querySelector('#crawlInfo').textContent.includes('1개 페이지')")
  page.locator('#crawlToBig').click()
  page.locator('#crawlOpenBig').click()
  page.wait_for_function("document.querySelector('#bigStats').textContent.includes('12')")
  assert current(page)=='bigdata'
  assert page.evaluate("window.__calls.filter(x=>x==='/api/analyze').length")==1
  # Big-data -> visualization is also an explicit link only.
  page.locator('#bigToVis').click()
  page.wait_for_function("document.querySelector('#visStats').textContent.includes('12')")
  assert current(page)=='visualize'
  page.locator('#visType').select_option('pie')
  assert page.locator('#visChart svg').count()==1
  # Training remains isolated.
  page.locator('.nav[data-page="training"]').click()
  page.locator('#trainDemo').click()
  page.locator('#trainRun').click()
  page.wait_for_function("document.querySelector('#trainStats').textContent.includes('90')")
  assert current(page)=='training'
  # Alerts do not change page; invalid blank values are rejected.
  page.locator('.nav[data-page="automation"]').click()
  page.locator('#alertRun').click()
  assert current(page)=='automation'
  assert '경보 기준에 도달' in page.locator('#alertOutput').inner_text()
  page.locator('#alertThreshold').fill('')
  page.locator('#alertRun').click()
  assert '유효한 기준값' in page.locator('#toast').inner_text()
  # Return to editor; existing story selection unchanged.
  page.locator('.nav[data-page="editor"]').click()
  assert current(page)=='editor'
  assert '점심' in page.locator('#resultTitle').inner_text()
  assert not errors, 'JS errors: '+str(errors)
  browser.close()
  print('PASS: 27 samples, no spontaneous crawler navigation, explicit handoff, stale-data reset, 5 lab menus, old editor state, no JS exceptions')

if __name__=='__main__':check()
