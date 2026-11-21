const {test,expect}=require('@playwright/test');
const fs=require('node:fs');
const axe=require('axe-core');
const base=process.env.CIPHERENC_TEST_URL;
const password='Synthetic correct horse battery staple!';
test.beforeEach(async({page,context})=>{
  await context.route('**/*',route=>{
    const url=new URL(route.request().url());
    return url.origin===new URL(base).origin?route.continue():route.abort();
  });
  await page.goto(base);
});

test('offline assets load without remote requests, storage, script errors or external links',async({page,context})=>{
  const requests=[],errors=[];
  context.on('request',request=>requests.push(request.url()));page.on('pageerror',error=>errors.push(error.message));
  await page.reload();
  await expect(page.locator('h1')).toContainText('Encrypt on your own device');
  await expect(page.locator('a[href^="http"]')).toHaveCount(0);
  await expect(page.locator('#internetToggle')).toHaveCount(0);
  const result=await page.evaluate(()=>({cookies:document.cookie,local:localStorage.length,session:sessionStorage.length,crypto:!!crypto.subtle,pgp:!!window.openpgp,argon:!!window.hashwasm}));
  expect(result).toEqual({cookies:'',local:0,session:0,crypto:true,pgp:true,argon:true});
  expect(requests.every(url=>new URL(url).origin===new URL(base).origin)).toBe(true);expect(errors).toEqual([]);
});

test('AES file UI saves .cen and restores exact binary data',async({page})=>{
  const bytes=Buffer.from([0,255,128,65,10,0,3]);
  await page.locator('#fileInput').setInputFiles({name:'synthetic.bin',mimeType:'application/octet-stream',buffer:bytes});
  await page.locator('#password').fill(password);await page.locator('#runButton').click();
  await expect(page.locator('#resultPanel')).toBeVisible();
  await expect(page.locator('#password')).toHaveValue('');
  const downloadPromise=page.waitForEvent('download');await page.locator('#downloadResult').click();const download=await downloadPromise;
  expect(download.suggestedFilename()).toBe('synthetic.bin.cen');
  const encrypted=fs.readFileSync(await download.path());expect(encrypted.subarray(0,8).toString()).toBe('CSPCS4!!');
  await page.locator('[data-op="decrypt"]').click();
  await page.locator('#fileInput').setInputFiles({name:'synthetic.bin.cen',mimeType:'application/octet-stream',buffer:encrypted});
  await page.locator('#password').fill('wrong password');await page.locator('#runButton').click();
  await expect(page.locator('#errorMessage')).toContainText('password is wrong');await expect(page.locator('#resultPanel')).toBeHidden();
  await page.locator('#password').fill(password);await page.locator('#runButton').click();await expect(page.locator('#resultPanel')).toBeVisible();
  const restoredPromise=page.waitForEvent('download');await page.locator('#downloadResult').click();const restored=await restoredPromise;
  expect(restored.suggestedFilename()).toBe('synthetic.bin');expect(fs.readFileSync(await restored.path())).toEqual(bytes);
});

test('encrypted text downloads use .cen and can be restored via file picker',async({page})=>{
  await page.locator('#textTab').click();await page.locator('#textInput').fill('Synthetic note · Español');await page.locator('#password').fill(password);await page.locator('#runButton').click();
  await expect(page.locator('#resultPanel')).toBeVisible();const promise=page.waitForEvent('download');await page.locator('#downloadResult').click();const download=await promise;
  expect(download.suggestedFilename()).toBe('encrypted-note.cen');const bytes=fs.readFileSync(await download.path());
  await page.locator('#fileTab').click();await page.locator('[data-op="decrypt"]').click();
  await page.locator('#fileInput').setInputFiles({name:'note.cen',mimeType:'application/octet-stream',buffer:bytes});await page.locator('#password').fill(password);await page.locator('#runButton').click();
  await expect(page.locator('#resultPanel')).toBeVisible();const openedPromise=page.waitForEvent('download');await page.locator('#downloadResult').click();const opened=await openedPromise;
  expect(fs.readFileSync(await opened.path(),'utf8')).toBe('Synthetic note · Español');
});

test('native XChaCha UI round trip and workspace clear',async({page})=>{
  await page.locator('#modernKind').selectOption('text');await page.locator('#modernText').fill('Synthetic XChaCha note');await page.locator('#modernPassword').fill(password);await page.locator('#modernConfirm').fill(password);
  await page.locator('#modernForm button[type=submit]').click();await expect(page.locator('#modernStatus')).toContainText('Encryption complete');
  await expect(page.locator('#modernPassword')).toHaveValue('');const promise=page.waitForEvent('download');await page.locator('#modernDownload').click();const download=await promise;
  expect(download.suggestedFilename()).toBe('encrypted.cen');const bytes=fs.readFileSync(await download.path());expect(bytes.subarray(0,8).toString()).toBe('CENCX01!');
  await page.locator('#modernOperation').selectOption('decrypt');await page.locator('#modernFile').setInputFiles({name:'encrypted.cen',mimeType:'application/octet-stream',buffer:bytes});await page.locator('#modernPassword').fill(password);
  await page.locator('#modernForm button[type=submit]').click();await expect(page.locator('#modernStatus')).toContainText('Authentication complete');
  const openedPromise=page.waitForEvent('download');await page.locator('#modernDownload').click();const opened=await openedPromise;expect(fs.readFileSync(await opened.path(),'utf8')).toBe('Synthetic XChaCha note');
  await page.locator('#clearWorkspace').click();await expect(page.locator('#modernDownload')).toBeHidden();
});

test('OpenPGP engine encrypts and decrypts locally',async({page})=>{
  const result=await page.evaluate(async()=>{
    const keys=await openpgp.generateKey({type:'ecc',curve:'curve25519Legacy',userIDs:[{name:'Synthetic Test'}],format:'armored'});
    const publicKey=await openpgp.readKey({armoredKey:keys.publicKey});const privateKey=await openpgp.readPrivateKey({armoredKey:keys.privateKey});
    const message=await openpgp.createMessage({text:'Synthetic PGP message'});const encrypted=await openpgp.encrypt({message,encryptionKeys:publicKey});
    const opened=await openpgp.decrypt({message:await openpgp.readMessage({armoredMessage:encrypted}),decryptionKeys:privateKey});return opened.data;
  });expect(result).toBe('Synthetic PGP message');
});

test('keyboard, narrow reflow, reduced motion and accessibility scan',async({page,browserName})=>{
  await page.keyboard.press('Tab');await expect(page.locator('.skip-link')).toBeFocused();await page.keyboard.press('Enter');
  await page.locator('#fileTab').focus();await page.keyboard.press('ArrowRight');await expect(page.locator('#textTab')).toBeFocused();
  await page.locator('#generatePassword').click();await expect(page.locator('#passwordGenerator')).toBeVisible();await page.keyboard.press('Escape');await expect(page.locator('#passwordGenerator')).toBeHidden();
  for(const width of [1280,640,320]){
    await page.setViewportSize({width,height:900});expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth+1)).toBe(true);
  }
  await page.emulateMedia({reducedMotion:'reduce'});
  await page.setViewportSize({width:1280,height:900});
  await page.evaluate(axe.source);
  const results=await page.evaluate(async()=>await axe.run(document,{runOnly:{type:'tag',values:['wcag2a','wcag2aa','wcag21aa','wcag22aa']}}));
  fs.mkdirSync('test-results',{recursive:true});fs.writeFileSync(`test-results/axe-${browserName}.json`,JSON.stringify(results.violations,null,2));
  expect(results.violations.map(v=>({id:v.id,nodes:v.nodes.map(n=>n.target)}))).toEqual([]);
  if(browserName==='chromium')await page.screenshot({path:'test-results/CipherEnc-desktop.png',fullPage:true});
});
