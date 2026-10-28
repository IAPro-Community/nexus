#!/usr/bin/env node
// Structural SVG contract: does not certify perceptual similarity or browser rendering.
import { readFileSync } from 'node:fs';
import { resolve, dirname } from 'node:path';
import { fileURLToPath } from 'node:url';
const root=resolve(dirname(fileURLToPath(import.meta.url)),'..');
const svg=readFileSync(resolve(root,'web/public/nex/nex-site-hero.svg'),'utf8');
const failures=[];
const test=(ok,msg)=>{if(!ok)failures.push(msg)};
test(svg.trimStart().startsWith('<svg') && svg.trimEnd().endsWith('</svg>'),'Invalid SVG outer element');
test(svg.includes('viewBox="0 0 1024 1024"'),'Hero viewBox drift');
test(svg.includes('data-facing="front"'),'Hero default pose must be front');
test(svg.includes('data-nex-art="site-hero-layered-v1"'),'Hero art marker missing');
test(!/<image[\s/>]/i.test(svg),'Hero should remain genuine vector paths, not embed a PNG');
test(!/<script[\s/>]/i.test(svg),'SVG must not run scripts');
test(svg.includes('prefers-reduced-motion'),'Reduced motion support missing');
test(svg.includes('data-motion="on"'),'Motion control missing');
for(const id of ['nex-figure','nex-yaw-stage','nex-shadow','nex-head','nex-face','nex-fin-left','nex-fin-right','nex-ear-left','nex-ear-right','nex-body','nex-reactor','nex-orbit-back','nex-orbit-front','nex-module-1','nex-module-2','nex-module-3','nex-module-4']){
  test(svg.includes(`id="${id}"`),`Missing animation-ready layer ${id}`);
}
const ids=[...svg.matchAll(/\sid="([^"]+)"/g)].map(x=>x[1]);
test(new Set(ids).size===ids.length,'Duplicate SVG IDs cause broken gradients and layer selectors');
for(const ref of [...svg.matchAll(/url\(#([^\)]+)\)/g)].map(x=>x[1])){
  test(ids.includes(ref),`Broken SVG gradient/clip-path reference: ${ref}`);
}
const markup=svg.replace(/<!\[CDATA\[[\s\S]*?\]\]>/g,'').replace(/<!--[\s\S]*?-->/g,'');
const stack=[];
for(const token of markup.matchAll(/<\/?([A-Za-z][\w:-]*)\b[^>]*>/g)){
 const [raw,name]=token;
 if(raw.startsWith('</')){
   const last=stack.pop();
   if(last!==name)failures.push(`Mismatched close: ${name} expected ${last}`);
 } else if(!raw.endsWith('/>'))stack.push(name);
}
test(stack.length===0,`Unclosed SVG tags: ${stack.join(', ')}`);
if(failures.length){
 for(const failure of failures)console.error('FAIL:',failure);
 process.exitCode=1;
} else console.log('NEX site SVG: structural parity PASS (visual/rendering/360° still need review)');
