'use strict';
let workspaceBusy = false;
const controls = () => [...document.querySelectorAll('button,input,textarea,select')];
async function runWorkspaceTask(task) {
  if (workspaceBusy) return;
  workspaceBusy = true;
  const list = controls(), previous = list.map(control => control.disabled);
  list.forEach(control => control.disabled = true);
  const monitor = document.querySelector('.status-panel');
  monitor.setAttribute('aria-busy', 'true');
  try { await task(); }
  catch (error) { els.error.textContent = error.message || 'The operation could not be completed.'; }
  finally {
    list.forEach((control, index) => control.disabled = previous[index]);
    workspaceBusy = false;
    monitor.setAttribute('aria-busy', 'false');
    sync();
  }
}
document.getElementById('runButton').addEventListener('click', async event => {
  event.stopImmediatePropagation();
  await runWorkspaceTask(runAndSave);
}, true);
document.getElementById('downloadResult').addEventListener('click', async event => {
  event.stopImmediatePropagation();
  await runWorkspaceTask(downloadResult);
}, true);
document.addEventListener('drop', event => {
  if (workspaceBusy) { event.preventDefault(); event.stopImmediatePropagation(); }
}, true);
document.getElementById('clearWorkspace').addEventListener('click', () => {
  if (!workspaceBusy) { clearSensitiveMaterial(); document.getElementById('clearWorkspace').focus(); }
});

const ZIP_MEDIA_TYPE='application/vnd.cipherenc.zip';
function equalBytes(a,b){return a.length===b.length&&a.every((value,index)=>value===b[index])}
function detectTextIntent(){
 mode='text';clearSelection();
 const value=els.text.value.trim(),pgp=value.startsWith('-----BEGIN PGP MESSAGE-----');
 if(pgp&&cryptoMethod!=='openpgp')setCryptoMethod('openpgp');
 if(/^PRISMVAULT[123]\./.test(value)&&cryptoMethod!=='password')setCryptoMethod('password');
 setOperation(pgp||/^PRISMVAULT[123]\./.test(value)?'decrypt':'encrypt');
 document.getElementById('inputIntent').textContent=operation==='decrypt'?'Encrypted text detected. Enter its password or private key to unlock.':'Text ready to encrypt. Enter a password and choose Encrypt text.';
}
async function identifyFile(file){
 const bytes=new Uint8Array(await file.slice(0,64).arrayBuffer()),head=dec.decode(bytes);
 if(['CSPCS4!!','CSPCS3!!','CSPCS2!!','NIVORA1!','PRISMVAULT1.','PRISMVAULT2.','PRISMVAULT3.'].some(m=>head.startsWith(m)))return 'password';
 if(head.startsWith('CENCX01!'))throw Error('This XChaCha20 package needs the native desktop app.');
 if(head.startsWith('-----BEGIN PGP MESSAGE-----'))return 'openpgp';
 const packet=bytes[0]&64?bytes[0]&63:(bytes[0]>>2)&15;
 if(/\.(cen|gpg|pgp|asc)$/i.test(file.name)||(bytes[0]&128)&&[1,3,9,18,20].includes(packet)){
  try{await openpgp.readMessage({binaryMessage:file.stream()});return 'openpgp'}catch{}
 }
 if(/\.cen$/i.test(file.name))throw Error('This .cen file is not a recognized encrypted package. Check that the file is complete.');
 return null;
}
async function autoSelectEntries(entries){
 if(workspaceBusy||!entries.length)return;
 const token=++selectionEpoch;selectedEntries=[];resetResult();els.run.disabled=true;
 try{
  if(entries.length>10000)throw Error('Choose at most 10,000 files per archive.');
  const kinds=await Promise.all(entries.map(e=>identifyFile(e.file)));
  if(token!==selectionEpoch)return;
  const encrypted=kinds.find(Boolean);
  if(encrypted&&entries.length!==1)throw Error('Choose one encrypted package at a time. Original files and folders can be bundled together.');
  mode='file';els.text.value='';$('#charCount').textContent='0';
  if(encrypted&&cryptoMethod!==encrypted)setCryptoMethod(encrypted);
  setOperation(encrypted?'decrypt':'encrypt');storeEntries(entries);
  $('#inputIntent').textContent=encrypted?'Encrypted package detected. It will be authenticated before anything is restored.':'Files ready. They will be ZIP-compressed, encrypted and verified before saving.';
  if(cryptoMethod==='openpgp'){$('#protectionMethod').value='openpgp';$('.advanced-tools').open=true;}
  // Dropping after entering credentials starts the action. Selecting files first
  // leaves the same action button available after credentials are entered.
  const ready=cryptoMethod==='password'?Boolean(els.password.value):Boolean($('#pgpKeyInput').value.trim()||pgpKeyBinary);
  if(ready)await runWorkspaceTask(runAndSave);
 }catch(error){if(token===selectionEpoch){els.error.textContent=error.message;clearSelection();sync()}}
}
async function encryptCompressedFiles(){
 const entries=selectedEntries.slice(),count=entries.length;
 if(!count||count>10000)throw Error('Choose between 1 and 10,000 files.');
 const total=entries.reduce((n,e)=>n+e.file.size,0);
 if(total>256*1048576)throw Error('Choose at most 256 MiB per compressed archive.');
 const zip=new JSZip(),seen=new Set();
 for(const entry of entries){
  const path=zipPath(entry.path),key=path.toLowerCase();
  if(seen.has(key))throw Error('Duplicate folder paths are not supported. Rename files that differ only by letter case.');
  seen.add(key);zip.file(path,entry.file,{date:new Date(1980,0,1),createFolders:true});
 }
 progress(2,'Compressing files into an in-memory ZIP');
 const compressed=await zip.generateAsync({type:'blob',compression:'DEFLATE',compressionOptions:{level:6},streamFiles:true});
 if(compressed.size>256*1048576)throw Error('The ZIP exceeds the 256 MiB archive limit.');
 const base=count===1&&!entries[0].path.includes('/')?entries[0].file.name:entries[0].path.split('/')[0];
 const name=base.slice(0,200)+'.zip',file=new File([compressed],name,{type:ZIP_MEDIA_TYPE});
 const packed=await encryptFilesChunked([{path:name,file}]);
 progress(98,'Verifying encrypted archive');
 const checked=await decryptFilesChunked(new File([packed.blob],'verify.cen'));
 const expected=new Uint8Array(await crypto.subtle.digest('SHA-256',await compressed.arrayBuffer()));
 const actual=checked.files.length===1?new Uint8Array(await crypto.subtle.digest('SHA-256',await checked.files[0].blob.arrayBuffer())):new Uint8Array();
 if(!equalBytes(expected,actual))throw Error('Encryption verification failed. No download was started.');
 return {...packed,name:base.slice(0,200)+'.cen',count};
}
async function runAndSave(){
 const encrypting=operation==='encrypt',inputMode=mode;
 if(window.CipherEncAndroid){const total=mode==='text'?enc.encode(els.text.value).length:selectedEntries.reduce((n,e)=>n+e.file.size,0);if(total>(encrypting?16:24)*1048576||selectedEntries.length>1000){els.error.textContent='Android evaluation limit: 16 MiB to encrypt, 24 MiB encrypted input, and 1,000 files.';return}}
 let handle=null;
 if(encrypting&&typeof window.showSaveFilePicker==='function'&&!window.CipherEncAndroid){
  const base=inputMode==='text'?(cryptoMethod==='openpgp'?'encrypted-message':'encrypted-note'):selectedEntries[0]?.path.split('/')[0]||'encrypted';
  try{handle=await window.showSaveFilePicker({suggestedName:base+'.cen',types:[{description:'CipherEnc encrypted file',accept:{'application/octet-stream':['.cen']}}]})}
  catch(error){if(error.name==='AbortError'){$('#statusCopy').textContent='Save cancelled. Your input is still ready; no encryption was started.';return}}
 }
 await runRequested();
 if(!resultBlob||els.result.classList.contains('hidden'))return;
 if(!encrypting&&inputMode==='text'){
  $('#textResult').value=resultText||await resultBlob.text();$('#textResultArea').classList.remove('hidden');return;
 }
 try{
  if(handle){
   const writable=await handle.createWritable();
   try{await writable.write(resultBlob);await writable.close()}catch(error){try{await writable.abort()}catch{};throw error}
   $('#statusCopy').textContent='Saved '+handle.name+'. Encryption and integrity checks completed on this device.';
  }else if(typeof window.CipherEncAndroidSave==='function'){
   await window.CipherEncAndroidSave();
  }else{
   await downloadResult(true);
   $('#statusCopy').textContent=encrypting?'Verified encrypted file sent to your browser’s downloads. Your browser controls the save location.':'Authenticated result sent to your browser’s downloads. The restored ZIP is unencrypted.';
  }
  $('#downloadResult').textContent='Save another copy';
 }catch(error){els.error.textContent='Saving did not finish: '+error.message+'. Your result is still available. Choose Save result to try again.';$('#downloadResult').textContent='Save result'}
}
$('#protectionMethod').addEventListener('change',event=>{setCryptoMethod(event.target.value);sync()});
els.password.addEventListener('keydown',event=>{if(event.key==='Enter'&&!els.run.disabled){event.preventDefault();els.run.click()}});
els.text.addEventListener('keydown',event=>{if(event.key==='Enter'&&(event.ctrlKey||event.metaKey)&&!els.run.disabled){event.preventDefault();els.run.click()}});

async function readInputOpenPgpMessage(engine){if(mode==='text')return engine.readMessage({armoredMessage:els.text.value.trim()});const file=selectedEntries[0].file;if(dec.decode(await file.slice(0,64).arrayBuffer()).startsWith('-----BEGIN PGP MESSAGE-----'))return engine.readMessage({armoredMessage:await file.text()});return engine.readMessage({binaryMessage:file.stream()})}
