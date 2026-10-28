#!/usr/bin/env node
/* Verify and optionally promote the binary status of the 2026-10-28 NEX reference pack. */
import { createHash } from 'node:crypto';
import { readFileSync, existsSync, writeFileSync } from 'node:fs';
import { dirname, resolve } from 'node:path';
import { resolveNexReferenceAsset } from './nex-reference-path.mjs';
import { fileURLToPath } from 'node:url';

const root=resolve(dirname(fileURLToPath(import.meta.url)),'..');
const manifestPath=resolve(root,'docs/design/nex/reference-pack/2026-10-28/manifest.json');
const manifest=JSON.parse(readFileSync(manifestPath,'utf8'));
const strict=process.argv.includes('--require-binaries')||process.argv.includes('--promote');
const promote=process.argv.includes('--promote');
const errors=[];
if(manifest.assets.length!==2)errors.push('Expected exactly two raster references');
for (const item of manifest.assets) {
  const path=resolveNexReferenceAsset(root,item);
  if(!path) {errors.push('Unsafe asset path: '+item.path);continue;}
  if(!/^[0-9a-f]{64}$/.test(item.sha256))errors.push('Invalid SHA-256: '+item.path);
  if (!existsSync(path)){
    if(strict || item.binary_status==='VERSIONED')errors.push('Missing PNG: '+item.path);
    continue;
  }
  const bytes=readFileSync(path);
  const hash=createHash('sha256').update(bytes).digest('hex');
  if(hash!==item.sha256)errors.push('SHA-256 mismatch: '+item.path);
  if(bytes.length!==item.bytes)errors.push('Byte-length mismatch: '+item.path);
  if(bytes.length<24 || bytes.subarray(0,8).toString('hex')!=='89504e470d0a1a0a'){
    errors.push('Not PNG: '+item.path);
  }else if(bytes.readUInt32BE(16)!==item.width || bytes.readUInt32BE(20)!==item.height){
    errors.push('PNG dimension mismatch: '+item.path);
  }
}
if(manifest.status==='VERSIONED' && manifest.assets.some(a=>a.binary_status!=='VERSIONED'))errors.push('Manifest version flags disagree');
if(errors.length){errors.forEach(e=>console.error('FAIL:',e));process.exitCode=1;}
else {
  if(promote){
    manifest.status='VERSIONED';
    manifest.files_uploaded_to_github=true;
    for(const item of manifest.assets)item.binary_status='VERSIONED';
    writeFileSync(manifestPath,JSON.stringify(manifest,null,2)+'\n');
    console.log('Asset references validated; stage PNGs and updated manifest in git.');
  }else console.log('NEX reference manifest verified; status:',manifest.status);
}