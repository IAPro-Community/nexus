#!/usr/bin/env node
// Importa imagens de referência sem alterar bytes nem promover master artístico.
import { createHash } from 'node:crypto';
import { copyFileSync, existsSync, mkdirSync, readFileSync, writeFileSync } from 'node:fs';
import { dirname, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';
const root=resolve(dirname(fileURLToPath(import.meta.url)),'..');
const manifestFile=resolve(root,'docs/design/nex/reference-pack/manifest.json');
const manifest=JSON.parse(readFileSync(manifestFile,'utf8'));
const n=process.argv.indexOf('--source');
if(n<0||!process.argv[n+1])throw Error('Usage: node scripts/import-nex-reference-pack.mjs --source <extracted-directory>');
const src=resolve(process.argv[n+1]);
const actual=(p)=>createHash('sha256').update(readFileSync(p)).digest('hex');
for(const item of manifest.assets){
 const from=resolve(src,item.filename);
 const bytes=readFileSync(from);
 const sig=bytes.subarray(0,8).toString('hex');
 if(sig!=='89504e470d0a1a0a'||bytes.length<24)throw Error('Not a PNG: '+item.filename);
 if(bytes.length!==item.bytes||actual(from)!==item.sha256)throw Error('Reference hash/size mismatch: '+item.filename);
 if(bytes.readUInt32BE(16)!==item.width||bytes.readUInt32BE(20)!==item.height)throw Error('Reference dimensions mismatch: '+item.filename);
}
for(const item of manifest.assets){
 const from=resolve(src,item.filename);
 const to=resolve(root,item.expected_repository_path);
 if(existsSync(to)){if(actual(to)!==item.sha256)throw Error('Existing repository asset differs: '+item.filename)}
 else{mkdirSync(dirname(to),{recursive:true});copyFileSync(from,to);}
 item.status='VERSIONED';
}
manifest.commit_status='VERSIONED';
writeFileSync(manifestFile,JSON.stringify(manifest,null,2)+'\n','utf8');
console.log('NEX reference PNGs verified and copied: 10. Stage PNGs + manifest in a Git commit.');
