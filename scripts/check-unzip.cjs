const {chromium,firefox,webkit}=require('@playwright/test'),assert=require('node:assert/strict');
const base=process.env.CIPHERENC_CHECK_BASE||'http://127.0.0.1:8941';
(async()=>{for(const[name,type]of [['chromium',chromium],['firefox',firefox],['webkit',webkit]]){
 const browser=await type.launch();try{const page=await browser.newPage();await page.goto(base+'/encrypt/');
 const result=await page.evaluate(async()=>{
  const make=async files=>{const z=new JSZip();for(const[name,value]of files){if(name.endsWith('/'))z.folder(name);else z.file(name,value)}return z.generateAsync({type:'uint8array',compression:'DEFLATE'})};
  const load=async bytes=>{resultBlob=new Blob([bytes],{type:ZIP_MEDIA_TYPE});resultFiles=[];resultDirectories=[];await unpackZipResult()};
  await load(await make([['original.txt','Exact unzipped bytes.\n\n']]));const single={name:resultName,text:await resultBlob.text(),files:resultFiles.length};
  const bad=[];
  for(const bytes of [await make([['../escape','bad']]),await make([['A.txt','one'],['a.txt','two']]),await make([['CON.txt','bad']])]){try{await load(bytes);bad.push(false)}catch{bad.push(true)}}
  const bomb=await make([['large.bin','A'.repeat(100000)]]);let central=-1;for(let p=0;p<bomb.length-4;p++)if(new DataView(bomb.buffer).getUint32(p,true)===0x02014b50){central=p;break}
  const lied=bomb.slice();new DataView(lied.buffer).setUint32(central+24,1,true);try{await load(lied);bad.push(false)}catch{bad.push(true)}
  const oversized=bomb.slice();new DataView(oversized.buffer).setUint32(central+24,0x7fffffff,true);try{await load(oversized);bad.push(false)}catch{bad.push(true)}
  const crc=bomb.slice();new DataView(crc.buffer).setUint32(central+16,1234,true);try{await load(crc);bad.push(false)}catch{bad.push(true)}
  const folder=await make([['Lessons/one.txt','First'],['Lessons/Nested/two.txt','Second'],['Lessons/Empty/','']]);await load(folder);
  const records={directories:[],files:{}};
  const directories=new Map();
  function dir(path){
   if(directories.has(path))return directories.get(path);
   const value={
    async getDirectoryHandle(name,opts={}){
     const next=path+'/'+name;
     if(!directories.has(next)&&!opts.create)throw new DOMException('Missing','NotFoundError');
     records.directories.push(next);return dir(next);
    },
    async getFileHandle(name){
     const file=path+'/'+name;
     return {async createWritable(){
      return {async write(blob){records.files[file]=await blob.text()},async close(){},async abort(){}};
     }};
    }
   };
   directories.set(path,value);return value;
  }
  const saved=await restoreToDirectory(dir('root'));return{single,bad,saved,records};
 });
 assert.deepEqual(result.single,{name:'original.txt',text:'Exact unzipped bytes.\n\n',files:0});assert(result.bad.every(Boolean));assert.match(result.saved,/^CipherEnc-restored-[a-f0-9]{24}$/);const files=Object.entries(result.records.files).map(([path,text])=>[path.replace('root/'+result.saved+'/',''),text]);assert.deepEqual(files,[['Lessons/one.txt','First'],['Lessons/Nested/two.txt','Second']]);assert(result.records.directories.some(d=>d.endsWith('/Lessons/Empty')));
 console.log(name+' original file restored unzipped; new-folder writing and empty folders; traversal, duplicates, reserved paths, size expansion and CRC rejection PASS');
 }finally{await browser.close()}
}})().catch(e=>{console.error(e);process.exitCode=1});
