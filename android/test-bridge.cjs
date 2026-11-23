// Development-only adapter checks in Chromium, not an Android/WebView device test.
const fs = require('node:fs');
const path = require('node:path');
const http = require('node:http');
const assert = require('node:assert/strict');
const {chromium} = require('@playwright/test');
const {execFileSync}=require('node:child_process');
const androidIndex=execFileSync('python3',[path.resolve(__dirname,'../scripts/openpgp_page.py')],{encoding:'utf8'});
const root = path.resolve(__dirname, '../docs/encrypt');
const types = {'.html':'text/html','.js':'application/javascript','.css':'text/css','.svg':'image/svg+xml','.png':'image/png','.json':'application/json'};
(async () => {
  const server = http.createServer((request, response) => {
    const name = decodeURIComponent(new URL(request.url, 'http://localhost').pathname).replace(/^\//, '') || 'index.html';
    const file = path.resolve(root, name);
    if (!file.startsWith(root + path.sep) || !fs.existsSync(file) || !fs.statSync(file).isFile()) { response.writeHead(404); response.end(); return; }
    response.setHeader('Content-Type', types[path.extname(file)] || 'application/octet-stream');
    response.end(name==='index.html'?androidIndex:fs.readFileSync(file));
  });
  await new Promise(resolve => server.listen(0, '127.0.0.1', resolve));
  let browser;
  try {
    browser = await chromium.launch();
    const page = await browser.newPage();
    await page.addInitScript(() => {
      window.androidCalls = [];
      window.CipherEncAndroid = {
        pickFolder: token => window.androidCalls.push({kind:'pick',token}),
        saveFile: (name, data) => window.androidCalls.push({kind:'save',name,data}),
        saveFolder: data => window.androidCalls.push({kind:'folder',data})
      };
    });
    await page.goto('http://127.0.0.1:' + server.address().port + '/');
    await page.evaluate(fs.readFileSync(path.join(__dirname, 'android-bridge.js'), 'utf8'));
    await page.locator('#chooseFolder').click();
    const token = await page.evaluate(() => window.androidCalls.at(-1).token);
    await page.locator('#clearWorkspace').click();
    assert.equal(await page.evaluate(token => {
      window.CipherEncAndroidFolder(token, [{path:'folder/note.txt',data:btoa('Synthetic note')}]);
      return selectedEntries.length;
    }, token), 0, 'Late folder callback resurrected cleared work');
    await page.locator('#chooseFolder').click();
    const current = await page.evaluate(() => window.androidCalls.at(-1).token);
    assert.equal(await page.evaluate(async token => {
      await window.CipherEncAndroidFolder(token, [{path:'folder/note.txt',data:btoa('Synthetic note')}]);
      return await selectedEntries[0].file.text();
    }, current), 'Synthetic note');
    await page.evaluate(() => {
      selectedEntries = [{path:'oversized.bin',file:{name:'oversized.bin',size:17*1048576}}];
      els.password.value = 'Synthetic adapter password'; sync();
    });
    await page.locator('#runButton').click();
    assert.match(await page.locator('#errorMessage').textContent(), /Android evaluation limit/);
    assert.equal(await page.evaluate(() => workspaceBusy), false, 'Size guard ran after crypto handler');
    await page.evaluate(() => {
      resultBlob = new Blob(['Synthetic save result']); resultName = 'encrypted.cen'; resultFiles = [];
      els.result.classList.remove('hidden');
    });
    await page.locator('#downloadResult').click();
    await page.waitForFunction(() => window.androidCalls.some(call => call.kind === 'save'));
    const saved = await page.evaluate(() => window.androidCalls.find(call => call.kind === 'save'));
    assert.equal(saved.name, 'encrypted.cen');
    assert.equal(Buffer.from(saved.data, 'base64').toString(), 'Synthetic save result');
    assert.equal(await page.locator('#clearWorkspace').isDisabled(), true);
    await page.evaluate(() => window.CipherEncAndroidResult(true, 'Synthetic Android save completed.'));
    assert.equal(await page.locator('#clearWorkspace').isDisabled(), false);
    const originalURL=page.url();
    await page.getByRole('link',{name:'Use OpenPGP',exact:true}).click();
    assert.equal(page.url(),originalURL,'Android key link navigated outside its permitted workspace');
    assert.equal(await page.evaluate(()=>cryptoMethod),'openpgp');
    assert.equal(await page.locator('#pgpKeyInput').isVisible(),true);
    assert.equal(await page.locator('#password').isVisible(),false);
    assert.equal(await page.locator('#pgpKeyInput').evaluate(e=>e===document.activeElement),true);
    console.log('Android JavaScript adapter: stale import, size guard ordering, explicit save bridge and inline OpenPGP link PASS in desktop Chromium. Android device behavior NOT RUN.');
  } finally { await browser?.close(); await new Promise(resolve => server.close(resolve)); }
})().catch(error => { console.error(error); process.exitCode = 1; });
