#!/usr/bin/env node
/** NEX frontal hover: integrity gate for 16 native 256x256 RGBA frames. */
import { createHash } from 'node:crypto';
import { existsSync, readFileSync } from 'node:fs';
import { resolve, dirname } from 'node:path';
import { fileURLToPath } from 'node:url';

const root=resolve(dirname(fileURLToPath(import.meta.url)),'..');
const dir=resolve(root,'docs/design/nex/hover256-v1');
const manifest=JSON.parse(readFileSync(resolve(dir,'manifest.json'),'utf8'));
const errors=[];
const test=(v,message)=>{if(!v)errors.push(message);};
const sha=buffer=>createHash('sha256').update(buffer).digest('hex');
const read=(rel)=>{if(!/^(?:frames\/hover-\d\d\.png|nex-hover-256-atlas\.(?:png|webp))$/.test(rel))throw Error('Invalid path '+rel);const path=resolve(dir,rel);if(!existsSync(path)){errors.push('Missing '+rel);return null;}return readFileSync(path);};
const png=(data,width,height,name)=>{
  if(!data)return;
  test(data.subarray(0,8).toString('hex')==='89504e470d0a1a0a','Bad PNG signature '+name);
  test(data.readUInt32BE(16)===width && data.readUInt32BE(20)===height,'Bad PNG size '+name);
  test(data[25]===6,'PNG must be true RGBA (color type 6): '+name);
};
test(manifest.status==='CANDIDATE_PENDING_VISUAL_APPROVAL','Do not assert artistic approval for hover candidate');
test(manifest.publication_allowed===false,'Do not publish incomplete animation');
test(manifest.source?.sha256==='ef7a4aeb7b24d2ab5f199f913d351414b8fafb0b6208b3ab2b665013adcb9118','Source identity drift');
const source=resolve(root,'web/public/nex/hybrid-v1/nex_source_original.png');
test(existsSync(source) && sha(readFileSync(source))===manifest.source?.sha256,'Original source not versioned or altered');
test(manifest.export?.frame_count===16 && manifest.export?.fps===8 && manifest.export?.duration_ms===2000,'Timing mismatch');
test(Array.isArray(manifest.frames) && manifest.frames.length===16,'Need 16 timeline frames');
const unique=new Set();
for(let i=0;i<(manifest.frames||[]).length;i++){
  const f=manifest.frames[i];
  test(f.index===i && f.file===`frames/hover-${String(i).padStart(2,'0')}.png`,'Frame index/path mismatch '+i);
  test(f.duration_ms===125 && f.alpha===true,'Frame metadata mismatch '+i);
  const data=read(f.file);
  png(data,256,256,f.file);
  if(data){
    test(data.length===f.bytes,'Byte length mismatch '+f.file);
    test(sha(data)===f.sha256,'SHA mismatch '+f.file);
    test(!unique.has(f.sha256),'Duplicate frame; use unique timeline poses '+i);
    unique.add(f.sha256);
  }
}
const atlas=read('nex-hover-256-atlas.png');
png(atlas,1024,1024,'atlas PNG');
if(atlas)test(sha(atlas)===manifest.export?.atlas_png_sha256,'Atlas PNG SHA mismatch');
const webp=read('nex-hover-256-atlas.webp');
if(webp){
  test(webp.toString('ascii',0,4)==='RIFF'&&webp.toString('ascii',8,16)==='WEBPVP8L','WebP must be lossless VP8L');
  const width=1+(webp[21]|((webp[22]&0x3f)<<8));
  const height=1+((webp[22]>>>6)|(webp[23]<<2)|((webp[24]&0x0f)<<10));
  test(width===1024&&height===1024,'WebP atlas size mismatch');
  test(sha(webp)===manifest.export?.atlas_webp_sha256,'WebP atlas SHA mismatch');
}
if(process.argv.includes('--require-ready'))errors.push('Frontal bob alone does not certify completed HUD, module rig or 360 rotation');
if(errors.length){for(const e of errors)console.error('FAIL:',e);process.exitCode=1;}
else console.log('PASS: NEX 16 unique transparent 256px hover frames/atlases verified; CANDIDATE ONLY');
