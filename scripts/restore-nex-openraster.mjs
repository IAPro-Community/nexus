#!/usr/bin/env node
/**
 * Restore NEX's exact approved-source OpenRaster project from tracked Git shards.
 * Splitting as base64 text makes the 5.34 MB original reproducibly versionable
 * where the GitHub connector cannot transport one large binary request.
 */
import { createHash } from 'node:crypto';
import { existsSync, mkdirSync, readFileSync, writeFileSync } from 'node:fs';
import { dirname, resolve, sep } from 'node:path';
import { fileURLToPath } from 'node:url';
const root=resolve(dirname(fileURLToPath(import.meta.url)), '..');
const manifest=JSON.parse(readFileSync(resolve(root,'docs/design/nex/hybrid-v1/asset-manifest.json'),'utf8'));
const archive=manifest.ora_archive;
if(!archive||archive.mode!=='GIT_TRACKED_BASE64_SHARDS'||archive.parts.length!==6)
 throw new Error('Unrecognized NEX OpenRaster archive');
function safe(rel){const p=resolve(root,rel);if(!p.startsWith(root+sep))throw new Error('Unsafe archive path: '+rel);return p}
const encoded=archive.parts.map(p=>readFileSync(safe(p),'utf8').trim()).join('');
if (!/^[A-Za-z0-9+/]*={0,2}$/.test(encoded))throw new Error('Invalid archive encoding');
const original=Buffer.from(encoded,'base64');
const sha256=createHash('sha256').update(original).digest('hex');
if(original.length!==archive.bytes||sha256!==archive.source_sha256)throw new Error('OpenRaster SHA-256 mismatch');
if(original.subarray(0,2).toString('hex')!=='504b')throw new Error('Not a ZIP/OpenRaster file');
const destination=safe(archive.final_path);
if(existsSync(destination)){
 const b=readFileSync(destination);
 if(b.length!==original.length||!b.equals(original))throw new Error('Existing OpenRaster differs from canonical bytes');
}else{
 mkdirSync(dirname(destination),{recursive:true});
 writeFileSync(destination,original);
}
console.log('NEX source restored byte-for-byte:',archive.final_path,sha256);
