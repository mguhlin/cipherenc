const {chromium,firefox,webkit}=require('@playwright/test'),assert=require('node:assert/strict');
const base=process.env.CIPHERENC_CHECK_BASE||'http://127.0.0.1:8941';
(async()=>{for(const[name,type]of [['chromium',chromium],['firefox',firefox],['webkit',webkit]]){
 const browser=await type.launch();try{const page=await browser.newPage({viewport:{width:1440,height:1000}});let staleRequested=false;
 await page.route('**/browser.css',route=>{staleRequested=true;return route.fulfill({contentType:'text/css',body:'label,p,input,button{font-size:9px!important}'})});
 await page.goto(base+'/encrypt/');
 const result=await page.evaluate(()=>({sizes:Object.fromEntries(['label[for=password]','.step-hint','#password','.password-note','.step-heading','#dropTitle','#dropSub','#chooseFiles','label[for=textInput]','#textInput','#runLabel','#textResultLabel','#strengthText','.char-count'].map(s=>[s,getComputedStyle(document.querySelector(s)).fontSize])),directions:document.querySelector('.workspace-directions').textContent,icons:document.querySelectorAll('.step-icon').length,assets:[...document.querySelectorAll('link[rel=stylesheet],script[src]')].map(e=>e.href||e.src)}));
 for(const[selector,size]of Object.entries(result.sizes))assert.equal(size,'20px',name+' '+selector);
 assert.equal(result.directions,'Directions: Follow these three steps to safeguard your data.Prefer OpenPGP private/public key encryption? Use OpenPGP or learn how they work.');assert.equal(result.icons,3);assert(result.assets.every(url=>/\?v=[a-f0-9]{12}$/.test(url)));assert.equal(staleRequested,false);
 for(const width of [1440,1120,640,390,320]){await page.setViewportSize({width,height:900});assert.equal(await page.evaluate(()=>document.documentElement.scrollWidth>innerWidth+1),false,name+' clipped at '+width)}
 console.log(name+' uniform 20px workspace type, three step icons, exact directions, content-versioned assets and 320–1440px reflow PASS');
 }finally{await browser.close()}
}})().catch(e=>{console.error(e);process.exitCode=1});
