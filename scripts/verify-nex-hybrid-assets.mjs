#!/usr/bin/env node
/**
 * NEX Hero layered-raster integrity gate.
 *
 * Typical workflow:
 *   unzip -o nex_hybrid_github_assets_v1.zip -d .
 *   node scripts/verify-nex-hybrid-assets.mjs --require-binaries
 *   node scripts/verify-nex-hybrid-assets.mjs --promote
 *
 * --promote marks the binary files as VERSIONED for the next Git commit.
 * This NEVER approves the design or publishes a runtime feature.
 */
import { createHash } from 'node:crypto';
import { existsSync, readFileSync, writeFileSync } from 'node:fs';
import { dirname, resolve, sep } from 'node:path';
import { fileURLToPath } from 'node:url';

const root=resolve(dirname(fileURLToPath(import.meta.url)),'..');
const path=resolve(root,'docs/design/nex/hybrid-v1/asset-manifest.json');
const manifest=JSON.parse(readFileSync(path,'utf8'));
const requireBinaries=process.argv.includes('--require-binaries')||process.argv.includes('--promote');
const promote=process.argv.includes('--promote');
const issues=[];
const safe=(file)=>{const resolved=resolve(root,file);if(!resolved.startsWith(root+sep))throw new Error('Unsafe NEX file path: '+file);return resolved};
const sha=(bytes)=>createHash('sha256').update(bytes).digest('hex');
if(manifest.assets.length!==14)issues.push('Expected 14 NEX hybrid asset entries');
if(manifest.source_sha256!=='ef7a4aeb7b24d2ab5f199f913d351414b8fafb0b6208b3ab2b665013adcb9118')issues.push('Original source SHA drift');
for(const a of manifest.assets){
  if(!/^[a-f0-9]{64}$/.test(a.sha256))issues.push('Invalid SHA: '+a.filename);
  const file=safe(a.path);
  if(!existsSync(file)){
    if(requireBinaries||a.status==='VERSIONED')issues.push('Missing asset: '+a.path);
    continue;
  }
  const bytes=readFileSync(file);
  if(bytes.length!==a.bytes||sha(bytes)!==a.sha256)issues.push('Asset SHA-256/size mismatch: '+a.path);
  if(a.filename.endsWith('.png')&&(bytes.subarray(0,8).toString('hex')!=='89504e470d0a1a0a'))issues.push('Invalid PNG signature: '+a.path);
  if(a.filename.endsWith('.ora') && bytes.subarray(0,2).toString('hex')!=='504b')issues.push('Invalid ORA zip header: '+a.path);
}
if(manifest.generator){
  const generatorPath=safe(manifest.generator.path||manifest.generator.file);
  if(!existsSync(generatorPath)){
    if(requireBinaries)issues.push('Missing portable art generator: '+generatorPath);
  }else{
    const bytes=readFileSync(generatorPath);
    if(sha(bytes)!==manifest.generator.sha256 || bytes.length!==manifest.generator.bytes)
      issues.push('Portable art generator SHA-256 mismatch');
    if(bytes.toString('utf8').includes('/mnt/data/'))
      issues.push('Art generator still contains temporary container paths');
  }
}
if(issues.length){for(const error of issues)console.error('FAIL:',error);process.exitCode=1;}
else if(promote){
  manifest.status='ASSETS_VERSIONED_PENDING_VISUAL_APPROVAL';
  for(const a of manifest.assets)a.status='VERSIONED';
  writeFileSync(path,JSON.stringify(manifest,null,2)+'\n','utf8');
  console.log('NEX assets pass checksum checks. Stage the PNGs/ORA/HTML plus manifest in a Git commit; visual approval is still pending.');
}else{
  console.log('NEX hybrid manifest: structurally valid; status:',manifest.status);
}
