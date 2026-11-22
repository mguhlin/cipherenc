const {chromium,firefox,webkit,devices}=require('@playwright/test');
const base=process.env.CIPHERENC_CHECK_BASE||'http://127.0.0.1:8941';
(async()=>{for(const [name,type,device]of [['chromium',chromium],['firefox',firefox],['webkit',webkit],['android-size',chromium,devices['Pixel 5']],['ios-size',webkit,devices['iPhone 13']]]){
 const browser=await type.launch(),context=await browser.newContext(device||{}),page=await context.newPage(),errors=[];page.on('pageerror',e=>errors.push(e.message));
 await page.goto(base+'/encrypt/#password-generator');await page.locator('#passwordGenerator').waitFor({state:'visible'});
 const password=await page.locator('#generatedPassword').textContent();if(!password)throw Error(name+' missing generated password');
 // An asynchronous success reproduced the expired-event-reference bug.
 await page.evaluate(()=>Object.defineProperty(navigator,'clipboard',{configurable:true,value:{writeText:async value=>{await Promise.resolve();window.testCopied=value}}}));
 await page.locator('#copyGenerated').click();await page.waitForFunction(()=>document.querySelector('#copyGenerated').textContent.includes('Copied'));
 if(await page.locator('#generatorError').textContent())throw Error(name+' false blocked warning');if(await page.evaluate(()=>window.testCopied)!==password)throw Error(name+' wrong copied value');
 await page.waitForFunction(()=>document.querySelector('#copyGenerated').textContent==='Copy');
 for(const denied of [true,false]){
  await page.evaluate(denied=>Object.defineProperty(navigator,'clipboard',{configurable:true,value:denied?{writeText:async()=>{throw new DOMException('Denied','NotAllowedError')}}:undefined}),denied);
  await page.locator('#copyGenerated').click();await page.waitForFunction(()=>document.querySelector('#generatorError').textContent.includes('Automatic copy is unavailable'));
  if(await page.evaluate(()=>window.getSelection().toString())!==password)throw Error(name+' manual selection failed');
 }
 await page.locator('#selectGenerated').click();if(await page.evaluate(()=>window.getSelection().toString())!==password)throw Error(name+' Select password failed');
 await page.locator('#useGenerated').click();if(await page.locator('#password').inputValue()!==password)throw Error(name+' direct fill failed');
 if(errors.length)throw Error(name+' '+errors.join(';'));console.log(name+' clipboard async success/denied/unavailable/selection/direct fill PASS');await browser.close();
}})().catch(e=>{console.error(e);process.exitCode=1});
