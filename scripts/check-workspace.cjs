const {chromium,firefox,webkit}=require('@playwright/test'),assert=require('node:assert/strict'),axe=require('axe-core');
const base=process.env.CIPHERENC_CHECK_BASE||'http://127.0.0.1:8941';
(async()=>{for(const[name,type]of [['chromium',chromium],['firefox',firefox],['webkit',webkit]]){
 const browser=await type.launch();try{
  const page=await browser.newPage({viewport:{width:1440,height:1000}}),errors=[];page.on('pageerror',error=>errors.push(error.message));await page.goto(base+'/encrypt/');
  assert.equal(await page.locator('#advanced-keys,#pgpCredentials,.browser-overview,#browser-help').count(),0);
  assert.equal(await page.locator('#workspace .workspace-status').count(),1);assert.equal(await page.locator('footer').count(),1);
  await page.locator('#password').fill('Synthetic help navigation password');await page.locator('#textInput').fill('Synthetic note kept while reading help');
  const opened=page.waitForEvent('popup');await page.getByRole('link',{name:'Help & privacy',exact:true}).click();const help=await opened;await help.waitForLoadState();
  assert.equal(await page.locator('#textInput').inputValue(),'Synthetic note kept while reading help');assert.equal(await page.locator('#password').inputValue(),'Synthetic help navigation password');
  assert(await help.locator('#browser-help details').count()>=8);assert(await help.locator('#openpgp-keys details').count()>=6);assert.equal(await help.locator('script[src],#runButton').count(),0);
  await help.evaluate(axe.source);const scan=await help.evaluate(async()=>axe.run(document,{runOnly:{type:'tag',values:['wcag2a','wcag2aa','wcag21aa','wcag22aa']}}));assert.deepEqual(scan.violations.map(v=>v.id),[]);
  for(const width of [1440,640,390,320]){await help.setViewportSize({width,height:900});assert.equal(await help.evaluate(()=>document.documentElement.scrollWidth>innerWidth+1),false,'help overflow '+width)}
  await help.close();await page.locator('#textInput').fill('-----BEGIN PGP MESSAGE-----\nSynthetic placeholder');assert.equal(await page.locator('#runButton').isDisabled(),true);assert.match(await page.locator('#errorMessage').textContent(),/Use OpenPGP/);
  await page.locator('#clearWorkspace').click();await page.getByRole('link',{name:'Use OpenPGP',exact:true}).click();await page.locator('#pgpKeyInput').waitFor();assert.equal(await page.locator('#advanced-keys,.browser-overview,#browser-help').count(),0);assert.equal(await page.locator('footer').count(),1);assert.deepEqual(errors,[]);
  console.log(name+': focused workspace, separate Help preserving input, help FAQs/accessibility/reflow, dedicated OpenPGP and footer PASS');
 }finally{await browser.close()}
}})().catch(error=>{console.error(error);process.exitCode=1});
