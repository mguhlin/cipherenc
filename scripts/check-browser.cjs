const CHECK_BASE=process.env.CIPHERENC_CHECK_BASE||'http://127.0.0.1:8941';
const {chromium,firefox,webkit,devices}=require('@playwright/test');
const axe=require('axe-core');
const fs=require('node:fs');
(async()=>{for(const [name,type,mobile] of [['chromium',chromium,false],['firefox',firefox,false],['webkit',webkit,false],['android-size',chromium,true],['ios-size',webkit,true]]){
 const browser=await type.launch();const context=await browser.newContext(mobile?{...(name==='ios-size'?devices['iPhone 13']:devices['Pixel 5'])}:{});const page=await context.newPage();const errors=[];page.on('pageerror',e=>errors.push(e.message));
 await context.route('**/*',r=>new URL(r.request().url()).origin===CHECK_BASE?r.continue():r.abort());
 await page.goto(CHECK_BASE+'/encrypt/');
 if(await page.locator('h1').textContent()!=='Protect a file, folder, or private note.')throw Error(name+' h1');
 await page.locator('#textTab').click();await page.locator('#textInput').fill('Synthetic mobile note · Español');await page.locator('#password').fill('Synthetic correct horse battery staple!');await page.locator('#runButton').click();await page.locator('#resultPanel').waitFor();
 const blob=await page.evaluate(async()=>({text:await resultBlob.text(),name:resultName}));if(blob.name!=='encrypted-note.cen')throw Error(name+' extension');
 await page.locator('[data-op="decrypt"]').click();await page.locator('#textInput').fill(blob.text);await page.locator('#password').fill('Synthetic correct horse battery staple!');await page.locator('#runButton').click();await page.locator('#resultPanel').waitFor();
 if(await page.evaluate(async()=>await resultBlob.text())!=='Synthetic mobile note · Español')throw Error(name+' text restore');
 await page.locator('#clearWorkspace').click();if(await page.locator('#password').inputValue()!=='')throw Error(name+' clear');
 await page.locator('#fileTab').click();await page.locator('[data-op="encrypt"]').click();await page.locator('#fileInput').setInputFiles({name:'audit.bin',mimeType:'application/octet-stream',buffer:Buffer.from([0,255,128,1,3])});await page.locator('#password').fill('Synthetic correct horse battery staple!');await page.locator('#runButton').click();await page.locator('#resultPanel').waitFor();
 const encrypted=await page.evaluate(async()=>Array.from(new Uint8Array(await resultBlob.arrayBuffer())));await page.locator('[data-op="decrypt"]').click();await page.locator('#fileInput').setInputFiles({name:'audit.bin.cen',mimeType:'application/octet-stream',buffer:Buffer.from(encrypted)});await page.locator('#password').fill('wrong');await page.locator('#runButton').click();await page.waitForFunction(()=>document.querySelector('#errorMessage').textContent.includes('password is wrong'));
 await page.locator('#password').fill('Synthetic correct horse battery staple!');await page.locator('#runButton').click();await page.locator('#resultPanel').waitFor();if(String(await page.evaluate(async()=>Array.from(new Uint8Array(await resultBlob.arrayBuffer()))))!=='0,255,128,1,3')throw Error(name+' file restore');
 await page.locator('#clearWorkspace').click();await page.locator('[data-op="encrypt"]').click();
 await page.evaluate(axe.source);const findings=await page.evaluate(async()=>await axe.run(document,{runOnly:{type:'tag',values:['wcag2a','wcag2aa','wcag21aa','wcag22aa']}}));
 fs.writeFileSync('/tmp/cipherenc-axe-browser-'+name+'.json',JSON.stringify(findings.violations,null,2));
 if(findings.violations.length)console.log(name+' axe '+findings.violations.map(x=>x.id));
 if(errors.length)throw Error(name+' page errors '+errors.join(';'));
 if(await page.evaluate(()=>document.documentElement.scrollWidth>innerWidth+1))throw Error(name+' overflow');
 console.log(name+' file/text/wrong password/clear/local-requests PASS; axe violations '+findings.violations.length);
 if(name==='chromium'){await page.setViewportSize({width:1440,height:1000});await page.goto(CHECK_BASE+'/');await page.screenshot({path:'/tmp/cipherenc-site-desktop.png',fullPage:true});await page.evaluate(axe.source);const landing=await page.evaluate(async()=>await axe.run(document,{runOnly:{type:'tag',values:['wcag2a','wcag2aa','wcag21aa','wcag22aa']}}));fs.writeFileSync('/tmp/cipherenc-axe-landing.json',JSON.stringify(landing.violations,null,2));console.log('landing axe violations '+landing.violations.length);await page.setViewportSize({width:320,height:800});if(await page.evaluate(()=>document.documentElement.scrollWidth>innerWidth+1))throw Error('landing320 overflow');await page.screenshot({path:'/tmp/cipherenc-site-mobile.png',fullPage:true})}
 await browser.close();
}})().catch(e=>{console.error(e);process.exitCode=1});
