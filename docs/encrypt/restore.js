'use strict';
// Only authenticated CipherEnc ZIP wrappers are unpacked automatically. The
// original file may itself be a ZIP; its contents remain the original file.
const RESTORE_LIMIT=(window.CipherEncAndroid?16:256)*1048576,RESTORE_ENTRIES=window.CipherEncAndroid?1000:10000;
const zipCrcTable=Uint32Array.from({length:256},(_,n)=>{let c=n;for(let i=0;i<8;i++)c=c&1?0xedb88320^(c>>>1):c>>>1;return c>>>0});
function validateRestoreTree(items){
 const seen=new Map();let total=0;
 for(const item of items){
  const path=zipPath(item.path),parts=path.split('/');
  for(let i=1;i<parts.length;i++){
   const name=parts.slice(0,i).join('/'),key=name.normalize('NFC').toLowerCase(),old=seen.get(key);
   if(old&&(!old.dir||old.name!==name))throw Error('Conflicting folder paths were found. Nothing was saved.');
   if(!old)seen.set(key,{name,dir:true,explicit:false});
  }
  const key=path.normalize('NFC').toLowerCase(),old=seen.get(key);
  if(old&&(old.explicit||!item.dir||!old.dir||old.name!==path))throw Error('Duplicate folder paths were found. Nothing was saved.');
  seen.set(key,{name:path,dir:!!item.dir,explicit:true});
  if(!item.dir){total+=item.size;if(!Number.isSafeInteger(item.size)||item.size<0||total>RESTORE_LIMIT)throw Error('Restored files exceed this device’s restoration limit.');}
 }
 if(items.length>RESTORE_ENTRIES)throw Error('Too many entries in the restored archive.');
}
function inspectZip(bytes){
 const view=new DataView(bytes.buffer,bytes.byteOffset,bytes.byteLength),u16=p=>view.getUint16(p,true),u32=p=>view.getUint32(p,true);
 let end=-1;
 for(let p=bytes.length-22;p>=Math.max(0,bytes.length-65557);p--)if(u32(p)===0x06054b50&&p+22+u16(p+20)===bytes.length){end=p;break}
 if(end<0||u16(end+4)||u16(end+6)||u16(end+8)!==u16(end+10))throw Error('Invalid or unsupported ZIP directory.');
 const count=u16(end+10),length=u32(end+12),start=u32(end+16);
 if(count<1||count>RESTORE_ENTRIES||count===65535||start+length>end)throw Error('Invalid or excessive ZIP directory.');
 const items=[];let p=start;
 for(let i=0;i<count;i++){
  if(p+46>start+length||u32(p)!==0x02014b50)throw Error('Invalid ZIP entry.');
  const flags=u16(p+8),method=u16(p+10),compressed=u32(p+20),size=u32(p+24),namesize=u16(p+28),extra=u16(p+30),comment=u16(p+32),mode=u32(p+38)>>>16,local=u32(p+42),next=p+46+namesize+extra+comment;
  if(next>start+length||flags&1||![0,8].includes(method)||u16(p+34)||local+30>start||u32(local)!==0x04034b50)throw Error('Unsupported ZIP entry.');
  const raw=new TextDecoder('utf-8',{fatal:true}).decode(bytes.subarray(p+46,p+46+namesize)),dir=raw.endsWith('/'),path=dir?raw.slice(0,-1):raw;
  if(raw.includes('\\')||mode&0xf000&&(mode&0xf000)!==(dir?0x4000:0x8000)||dir&&size)throw Error('ZIP links and special files are not supported.');
  const localNameSize=u16(local+26),dataStart=local+30+localNameSize+u16(local+28);
  if(dataStart+compressed>start||localNameSize!==namesize||u16(local+8)!==method||!equalBytes(bytes.subarray(local+30,local+30+localNameSize),bytes.subarray(p+46,p+46+namesize)))throw Error('Inconsistent ZIP entry.');
  items.push({raw,path,dir,size,crc:u32(p+16)});p=next;
 }
 if(p!==start+length)throw Error('Unexpected ZIP directory data.');
 validateRestoreTree(items);return items;
}
function inflateBounded(entry,item){
 return new Promise((resolve,reject)=>{
  const chunks=[];let size=0,crc=0xffffffff,failed=false;
  const stream=entry.internalStream('uint8array');
  const stop=error=>{if(failed)return;failed=true;stream.pause();chunks.length=0;reject(error)};
  stream.on('data',chunk=>{
   if(failed)return;size+=chunk.length;
   if(size>item.size||size>RESTORE_LIMIT){stop(Error('ZIP expansion exceeded the declared size. Nothing was saved.'));return}
   for(const byte of chunk)crc=zipCrcTable[(crc^byte)&255]^(crc>>>8);
   chunks.push(chunk);
  }).on('error',stop).on('end',()=>{
   if(failed)return;
   if(size!==item.size||((crc^0xffffffff)>>>0)!==item.crc){stop(Error('ZIP contents failed verification. Nothing was saved.'));return}
   resolve(new Blob(chunks,{type:'application/octet-stream'}));
  }).resume();
 });
}
async function unpackZipResult(){
 if(!resultBlob||resultBlob.type!==ZIP_MEDIA_TYPE)return;
 const bytes=new Uint8Array(await resultBlob.arrayBuffer()),items=inspectZip(bytes),zip=await JSZip.loadAsync(bytes,{createFolders:false});
 const files=[],dirs=[];
 for(const item of items){
  if(item.dir){dirs.push(item.path);continue}
  const entry=zip.file(item.raw);if(!entry)throw Error('A ZIP entry could not be read.');
  files.push({path:item.path,blob:await inflateBounded(entry,item)});
 }
 if(files.length===1&&!dirs.length&&!files[0].path.includes('/')){
  resultBlob=files[0].blob;resultName=files[0].path;resultFiles=[];resultDirectories=[];
  $('#resultMeta').textContent='Authenticated, unzipped and restored: '+resultName;
 }else{
  resultFiles=files;resultDirectories=dirs;resultBlob=new Blob([]);
  $('#resultMeta').textContent='Authenticated and unzipped '+files.length+' file(s).';
 }
}
async function restoreToDirectory(parent){
 const items=[...resultDirectories.map(path=>({path,dir:true,size:0})),...resultFiles.map(file=>({...file,dir:false,size:file.blob.size}))];
 validateRestoreTree(items);
 let folder,name;
 for(let n=0;n<10;n++){
  name='CipherEnc-restored-'+Array.from(crypto.getRandomValues(new Uint8Array(12)),b=>b.toString(16).padStart(2,'0')).join('');
  try{await parent.getDirectoryHandle(name);continue}catch(error){if(error.name!=='NotFoundError')throw error}
  folder=await parent.getDirectoryHandle(name,{create:true});break;
 }
 if(!folder)throw Error('A new restoration folder could not be created.');
 for(const item of items){
  const parts=item.path.split('/');let dir=folder;
  for(const part of item.dir?parts:parts.slice(0,-1))dir=await dir.getDirectoryHandle(part,{create:true});
  if(item.dir)continue;
  const handle=await dir.getFileHandle(parts.at(-1),{create:true}),writer=await handle.createWritable();
  try{await writer.write(item.blob);await writer.close()}catch(error){try{await writer.abort()}catch{};throw Error('Restoration stopped in '+name+'. Some restored files may remain: '+error.message)}
 }
 $('#statusCopy').textContent='Unzipped files restored in the new folder '+name+'. Existing files were not replaced.';
 return name;
}
function showRestoredFileList(){
 const area=$('#restoredFilesArea'),list=$('#restoredFilesList');list.replaceChildren();area.classList.remove('hidden');
 for(const item of resultFiles){
  const row=document.createElement('li'),name=document.createElement('span'),button=document.createElement('button');
  name.textContent=item.path;button.type='button';button.className='secondary';button.textContent='Save file';button.setAttribute('aria-label','Save '+item.path);button.addEventListener('click',()=>downloadBlob(item.blob,item.path.split('/').at(-1)));row.append(name,button);list.append(row);
 }
 $('#statusCopy').textContent='Your files are authenticated and unzipped. This browser cannot write a folder directly. Save the files below, or use the native app to restore their complete folder structure.';
 $('#downloadResult').textContent='Download ZIP instead';
}
