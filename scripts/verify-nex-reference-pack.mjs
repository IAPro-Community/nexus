#!/usr/bin/env node
import {createHash} from 'node:crypto';
import {existsSync,readFileSync} from 'node:fs';
import {dirname,resolve} from 'node:path';
import {fileURLToPath} from 'node:url';
const root=resolve(dirname(fileURLToPath(import.meta.url)),'..');
const m=JSON.parse(readFileSync(resolve(root,'docs/design/nex/reference-pack/manifest.json'),'utf8'));
const issues=[];
if(m.assets.length!==10)issues.push('Expected 10 visual concept references');
const hash=(buf)=>createHash('sha256').update(buf).digest('hex');
for(const a of m.assets){
 const path=resolve(root,a.expected_repository_path);
 const shouldExist=a.status==='VERSIONED';
 if(!existsSync(path)){if(shouldExist)issues.push('VERSIONED reference missing: '+a.filename);continue;}
 const b=readFileSync(path);
 if(b.length!==a.bytes||hash(b)!==a.sha256)issues.push('Reference integrity mismatch: '+a.filename);
 if(b.length<24||b.subarray(0,8).toString('hex')!=='89504e470d0a1a0a'||b.readUInt32BE(16)!==a.width||b.readUInt32BE(20)!==a.height)issues.push('Reference PNG metadata mismatch: '+a.filename);
}
if(m.commit_status==='VERSIONED'&&m.assets.some(a=>a.status!=='VERSIONED'))issues.push('Manifest version status inconsistent');
if(issues.length){for(const x of issues)console.error('FAIL: '+x);process.exitCode=1;}
else console.log('NEX reference manifest structurally valid; binary status:',m.commit_status);
