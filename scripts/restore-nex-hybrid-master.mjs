#!/usr/bin/env node
/** Reconstruct the original OpenRaster source without any image recoding. */
import { createHash } from 'node:crypto';
import { existsSync, mkdirSync, readFileSync, writeFileSync } from 'node:fs';
import { resolve, dirname, sep } from 'node:path';
import { fileURLToPath } from 'node:url';
const root = resolve(dirname(fileURLToPath(import.meta.url)), '..');
const dir = resolve(root, 'docs/design/nex/hybrid-v1/chunks');
const meta = JSON.parse(readFileSync(resolve(dir,'manifest.json'), 'utf8'));
const verify = (buffer,expected) => createHash('sha256').update(buffer).digest('hex')===expected;
const all = [];
for (const segment of meta.segments) {
 if (!/^nex_hybrid_master\.ora\.part-\d{3}\.b64$/.test(segment.file)) throw new Error('Unsafe chunk filename');
 const file=resolve(dir,segment.file);
 if(!file.startsWith(dir+sep)||!existsSync(file))throw new Error('Missing ORA segment: '+segment.file);
 const encoded=readFileSync(file,'utf8').trim();
 if(!/^[A-Za-z0-9+/]+={0,2}$/.test(encoded)) throw new Error('Invalid base64 chunk: '+segment.file);
 const bytes=Buffer.from(encoded,'base64');
 if(bytes.length!==segment.decoded_bytes||!verify(bytes,segment.decoded_sha256)) throw new Error('ORA segment checksum mismatch: '+segment.file);
 all.push(bytes);
}
const bytes=Buffer.concat(all);
if(bytes.length!==meta.size || !verify(bytes,meta.sha256) || bytes.subarray(0,2).toString('hex')!=='504b') throw new Error('Original OpenRaster integrity mismatch');
if(process.argv.includes('--restore')) {
 const output=resolve(root,'docs/design/nex/hybrid-v1/nex_hybrid_master.ora');
 mkdirSync(dirname(output),{recursive:true});
 if(existsSync(output) && !verify(readFileSync(output),meta.sha256)) throw new Error('Refusing to overwrite different OpenRaster source');
 writeFileSync(output,bytes);
 console.log('Reconstructed original editable NEX source: '+output);
}else {
 console.log('NEX source is stored in '+meta.segments.length+' exact Git chunks; original SHA-256 verified:',meta.sha256);
}
