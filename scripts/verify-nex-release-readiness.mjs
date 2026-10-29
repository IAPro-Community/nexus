#!/usr/bin/env node
/**
 * NEX joint release gate. Ordinary check only validates the honesty of status.
 * --require-ready fails until ALL 11 deliverables and explicit owner approval
 * have verifiable evidence. Does not deploy or change Nexus/HUD itself.
 */
import { readFileSync, existsSync } from 'node:fs';
import { dirname, resolve, sep } from 'node:path';
import { fileURLToPath } from 'node:url';
const root=resolve(dirname(fileURLToPath(import.meta.url)), '..');
const manifestPath=resolve(root,'docs/design/nex/RELEASE_READINESS.json');
const m=JSON.parse(readFileSync(manifestPath,'utf8'));
const fail=[];
const check=(ok,msg)=>{if(!ok)fail.push(msg)};
const ids=new Set();
check(m.publication_policy==='ATOMIC_HUD_AND_SITE','Must release HUD and Site atomically');
check(m.status==='BLOCKED'||m.status==='READY','Invalid release readiness status');
check(Array.isArray(m.requirements)&&m.requirements.length===11,'Expected 11 hard release gates');
for(const item of m.requirements||[]){
 check(!ids.has(item.id),'Duplicate gate '+item.id);ids.add(item.id);
 check(item.status==='BLOCKED'||item.status==='PASSED','Invalid gate status '+item.id);
 check(Array.isArray(item.evidence),'Missing evidence array '+item.id);
 const path=resolve(root,item.contract||'');
 check(path.startsWith(root+sep)&&existsSync(path),'Missing normative document: '+item.contract);
 if(item.status==='PASSED'){
   check(item.evidence.length>0,'Gate PASSED without linked evidence: '+item.id);
   for(const e of item.evidence)check(typeof e==='string'&&e.trim().length>8,'Unusable evidence: '+item.id);
 }
}
const allPassed=(m.requirements||[]).every(x=>x.status==='PASSED');
if(m.status==='READY'){
 check(allPassed,'Release marked READY with pending gates');
 check(m.public_site_allowed===true&&m.replace_hud_allowed===true,'READY state must activate both surfaces');
 check(typeof m.reference_commit==='string'&&/^[a-f0-9]{40}$/.test(m.reference_commit),'Missing 40-char source SHA');
 check(typeof m.release_artifact_sha256==='string'&&/^[a-f0-9]{64}$/.test(m.release_artifact_sha256),'Missing release artifact SHA-256');
 check(Boolean(m.approval?.approved_by&&m.approval.approved_at&&m.approval.approved_commit),'Missing explicit human approval');
 check(m.approval?.approved_commit===m.reference_commit,'Approval must bind to exact source SHA');
} else {
 check(m.public_site_allowed===false,'Site publication must be false while BLOCKED');
 check(m.replace_hud_allowed===false,'HUD replacement must be false while BLOCKED');
 check(!allPassed,'All requirements PASSED while release still BLOCKED; resolve state explicitly');
}
if(process.argv.includes('--require-ready'))check(m.status==='READY','NEX full HUD + Site release is BLOCKED');
if(fail.length){for(const error of fail)console.error('FAIL:',error);process.exitCode=1;}
else console.log('NEX joint release:',m.status,'('+m.requirements.filter(x=>x.status==='PASSED').length+'/'+m.requirements.length+' gates)','; public site:',m.public_site_allowed);
