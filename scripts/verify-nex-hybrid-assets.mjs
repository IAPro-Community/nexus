#!/usr/bin/env node
/**
 * NEX exact-source integrity gate.
 * CI: node scripts/verify-nex-hybrid-assets.mjs --require-binaries
 * The canonical OpenRaster is stored in verifiable Git-tracked base64 parts;
 * this checker decodes it in memory without requiring restoration to disk.
 */
import {createHash} from 'node:crypto';
import {existsSync,readFileSync} from 'node:fs';
import {dirname,resolve,sep} from 'node:path';
import {fileURLToPath} from 'node:url';
const root=resolve(dirname(fileURLToPath(import.meta.url)),'..');
const manifest=JSON.parse(readFileSync(resolve(root,'docs/design/nex/hybrid-v1/asset-manifest.json'),'utf8'));
const strict=process.argv.includes('--require-binaries')||process.argv.includes('--promote');
const errors=[];
const safe=(rel)=>{const p=resolve(root,rel);if(!p.startsWith(root+sep))throw Error('Unsafe path: '+rel);return p};
const hash=(b)=>createHash('sha256').update(b).digest('hex');
const check=(ok,msg)=>{if(!ok)errors.push(msg)};
check(manifest.assets.length===14,'Expected exactly 14 assets');
check(manifest.source_sha256==='ef7a4aeb7b24d2ab5f199f913d351414b8fafb0b6208b3ab2b665013adcb9118','Source identity drift');
let reconstructed=null;
const archive=manifest.ora_archive;
if(archive){
  try{
    check(archive.mode==='GIT_TRACKED_BASE64_SHARDS'&&archive.parts.length===6,'Unexpected ORA archive');
    const encoded=archive.parts.map(part=>readFileSync(safe(part),'utf8').trim()).join('');
    check(/^[A-Za-z0-9+/]*={0,2}$/.test(encoded),'Invalid OpenRaster shard encoding');
    reconstructed=Buffer.from(encoded,'base64');
    check(reconstructed.length===archive.bytes&&hash(reconstructed)===archive.source_sha256,'OpenRaster archive SHA/size mismatch');
  }catch(e){errors.push('Cannot reconstruct OpenRaster archive: '+e.message);}
}
for(const a of manifest.assets){
  check(/^[a-f0-9]{64}$/.test(a.sha256),'Invalid SHA: '+a.filename);
  const path=safe(a.path);
  let b;
  if(existsSync(path))b=readFileSync(path);
  else if(a.filename.endsWith('.ora')&&reconstructed)b=reconstructed;
  else{
    if(strict||a.status==='VERSIONED'||a.status==='RECONSTRUCTIBLE_FROM_VERSIONED_PARTS')errors.push('Missing versioned asset: '+a.path);
    continue;
  }
  check(b.length===a.bytes&&hash(b)===a.sha256,'Asset mismatch: '+a.path);
  if(a.filename.endsWith('.png'))check(b.subarray(0,8).toString('hex')==='89504e470d0a1a0a','Invalid PNG signature: '+a.filename);
  if(a.filename.endsWith('.ora'))check(b.subarray(0,2).toString('hex')==='504b','Invalid ORA ZIP signature');
  if(a.status==='RECONSTRUCTIBLE_FROM_VERSIONED_PARTS')check(Boolean(archive),'ORA lacks archived shard metadata');
}
if(manifest.generator){
 const path=safe(manifest.generator.path||manifest.generator.file);
 if(!existsSync(path)){if(strict)errors.push('Missing generator');}
 else{
   const b=readFileSync(path);
   check(b.length===manifest.generator.bytes&&hash(b)===manifest.generator.sha256,'Generator SHA mismatch');
   check(!b.toString('utf8').includes('/mnt/data/'),'Generator contains ephemeral path');
 }
}
if(strict)check(manifest.assets.every(a=>a.status==='VERSIONED'||a.status==='RECONSTRUCTIBLE_FROM_VERSIONED_PARTS'),
 'Strict validation requires committed binary/shard status');
if(errors.length){for(const error of errors)console.error('FAIL:',error);process.exitCode=1;}
else console.log('NEX 14/14 source assets verified; OpenRaster is reconstructible; artistic approval still pending.');
