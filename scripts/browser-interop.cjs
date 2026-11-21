// Generate independent browser-engine fixtures and verify native Go CS4 output.
const {chromium}=require('@playwright/test');
const http=require('node:http');const fs=require('node:fs');const path=require('node:path');const {execFileSync}=require('node:child_process');
(async()=>{
 const root=path.resolve('docs/encrypt');
 const server=http.createServer((req,res)=>{let name=decodeURIComponent(req.url.split('?')[0]);if(name.endsWith('/'))name+='index.html';const file=path.join(root,name);if(!file.startsWith(root+path.sep)){res.writeHead(404);res.end();return};try{const data=fs.readFileSync(file);const ext=path.extname(file);res.setHeader('Content-Type',ext==='.js'?'text/javascript':ext==='.css'?'text/css':ext==='.html'?'text/html':'application/octet-stream');res.end(data)}catch{res.writeHead(404);res.end()}});
 await new Promise(resolve=>server.listen(0,'127.0.0.1',resolve));let browser;
 try{
  browser=await chromium.launch();const page=await browser.newPage();await page.goto(`http://127.0.0.1:${server.address().port}/`);
  const fixtures=await page.evaluate(async()=>{
   const password='Synthetic independent passphrase!';els.password.value=password;const plain='Independent CipherEnc browser fixture';
   selectedEntries=[{file:new File([plain],'fixture.txt',{type:'text/plain'}),path:'fixture.txt'}];
   const cs4=await encryptFilesChunked();const text=await encryptData(enc.encode(plain),'text','fixture.txt',[]);
   const salt=crypto.getRandomValues(new Uint8Array(16)),iv=crypto.getRandomValues(new Uint8Array(12)),key=await derive(password,salt);
   const header=enc.encode(JSON.stringify({v:2,alg:'AES-256-GCM',kdf:'PBKDF2-SHA-256',iter:ITERATIONS,salt:b64(salt),iv:b64(iv)}));
   const aad=containerAad(MAGIC_TEXT2,header);const cipher=new Uint8Array(await crypto.subtle.encrypt({name:'AES-GCM',iv,additionalData:aad,tagLength:128},key,makePlain('text','legacy.txt',enc.encode(plain),[])));
   const extras=[];
   const h1=enc.encode(JSON.stringify({v:1,alg:'AES-256-GCM',kdf:'PBKDF2-SHA-256',iter:ITERATIONS,salt:b64(salt),iv:b64(iv)}));
   const c1=new Uint8Array(await crypto.subtle.encrypt({name:'AES-GCM',iv,tagLength:128},key,makePlain('text','legacy.txt',enc.encode(plain),[])));
   extras.push({name:'browser-text-v1',password,data:b64(concat(MAGIC,u32(h1.length),h1,c1)),expected:plain});
   for(const format of [2,3]){
    const prefix=crypto.getRandomValues(new Uint8Array(8));const header=enc.encode(JSON.stringify({v:format,alg:'AES-256-GCM-CHUNKED',kdf:'PBKDF2-SHA-256',iter:ITERATIONS,chunkSize:CHUNK_SIZE,salt:b64(salt),prefix:b64(prefix)}));
    const data=enc.encode(plain);const meta=enc.encode(JSON.stringify({kind:'file',name:'legacy.txt',files:[{path:'legacy.txt',size:data.length,type:'text/plain',chunks:1}]}));
    const seal=async(counter,bytes)=>{const cipher=new Uint8Array(await crypto.subtle.encrypt({name:'AES-GCM',iv:ivFor(prefix,counter),additionalData:recordAad(format,header,counter),tagLength:128},key,bytes));return concat(u32(bytes.length),u32(cipher.length),cipher)};
    const packed=concat(format===2?MAGIC2:MAGIC3,u32(header.length),header,await seal(0,meta),await seal(1,data));
    extras.push({name:'browser-CS'+format,password,data:b64(packed),expected:plain});
   }
   return [{name:'browser-CS4',password,data:b64(new Uint8Array(await cs4.blob.arrayBuffer())),expected:plain},{name:'browser-text-v3',password,data:b64(text),expected:plain},{name:'browser-text-v2',password,data:b64(concat(aad,cipher)),expected:plain},...extras];
  });fs.writeFileSync('reference/compatibility/browser-fixtures.json',JSON.stringify(fixtures,null,2)+'\n');
  execFileSync('build/CipherEnc-native',['--self-test','/tmp/cipherenc-interop-native'],{stdio:'inherit'});
  const native=fs.readFileSync('/tmp/cipherenc-interop-native/native-cs4.cen');
  const result=await page.evaluate(async bytes=>{els.password.value='Synthetic independent passphrase!';const result=await decryptFilesChunked(new File([new Uint8Array(bytes)],'native.cen'));return {paths:result.files.map(f=>f.path),text:await result.files[0].blob.text()};},[...native]);
  if(result.text!=='Synthetic sample'||result.paths.length!==2)throw Error('Browser could not restore native CS4 output');
  console.log('Interop PASS: browser CS4/text-v3/legacy-v2 fixtures generated; browser restored native Go CS4 nested folder.');
 }finally{await browser?.close();server.close()}
})().catch(error=>{console.error(error);process.exitCode=1});
