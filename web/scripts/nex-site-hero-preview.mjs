/**
 * Offline, self-contained Hero SVG preview generated from ONE official file.
 * No scene redraws, no 3D engine, no network or runtime dependencies.
 * Run: node web/scripts/nex-site-hero-preview.mjs
 * Output: web/dist/nex-site-hero-preview.html
 */
import { readFile, mkdir, writeFile } from 'node:fs/promises';
import { dirname, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';

const web = resolve(dirname(fileURLToPath(import.meta.url)), '..');
const file = resolve(web, 'public/nex/nex-site-hero.svg');
const svg = await readFile(file, 'utf8');
const mandatory = [
  'data-nex-art="site-hero-layered-v1"',
  'id="nex-yaw-stage"', 'id="nex-head"', 'id="nex-fin-left"',
  'id="nex-fin-right"', 'id="nex-reactor"', 'id="nex-face"',
  'id="nex-module-1"', 'id="nex-module-2"',
  'id="nex-module-3"', 'id="nex-module-4"',
];
for (const marker of mandatory) {
  if (!svg.includes(marker)) throw new Error(`Missing NEX canonical part: ${marker}`);
}
const safeSvg = svg.replace(/<\?xml[^>]*\?>\s*/i, '');
const html = `<!doctype html>
<html lang="pt-BR"><head><meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>NEX · Hero SVG Animation QA</title>
<style>
*{box-sizing:border-box}
body{margin:0;background:#061021;color:#e8f4ff;font:15px system-ui,-apple-system,sans-serif}
main{width:min(1160px,100%);margin:auto;padding:24px}
h1{font-size:clamp(22px,4vw,36px);margin:0 0 6px}
p{color:#9cb9d7;line-height:1.5;max-width:850px}
.stage{display:grid;grid-template-columns:minmax(0,1fr) minmax(250px,310px);gap:20px}
.figure{display:grid;place-items:center;overflow:hidden;border:1px solid #1D4C78;
border-radius:16px;min-height:490px;background:radial-gradient(ellipse at 50% 42%,#173656,#091728 70%,#07111F)}
.figure svg{width:min(100%,660px);height:auto;max-height:650px}
.controls{padding:20px;background:#0D1E37;border:1px solid #1B487A;border-radius:16px;display:flex;flex-direction:column;gap:14px}
button{border:1px solid #376A9C;background:#15375D;color:#e7f6ff;padding:10px 12px;border-radius:7px;font:inherit;cursor:pointer}
button[aria-pressed="true"]{border-color:#54DBFF;background:#235582}
button:focus-visible,input:focus-visible{outline:2px solid #69DFFF;outline-offset:3px}
.buttons{display:flex;gap:8px;flex-wrap:wrap}
label{display:flex;gap:8px;align-items:center;flex-wrap:wrap}
input[type=range]{width:100%;accent-color:#31D5FF}
.small{font-size:12px;line-height:1.4;color:#8DAFCC}
.info{border-top:1px solid #214B72;padding-top:12px}
#nex-yaw-stage{transform-origin:512px 500px;transition:transform 220ms ease-out}
#nex-site-hero[data-motion=off] #nex-yaw-stage{transition:none}
@media (max-width:770px){.stage{grid-template-columns:1fr}.figure{min-height:330px}}
@media (prefers-reduced-motion:reduce){#nex-yaw-stage{transition:none}}
</style></head>
<body><main>
<h1>NEX / SVG Hero — movimento leve e giro simulado</h1>
<p>Esta página usa o <b>SVG vetorial real</b> com camadas independentes. A inclinação abaixo é uma aproximação 2D de poucos graus; não é uma vista traseira nem um giro real de 360°.</p>
<div class="stage">
<div class="figure">${safeSvg}</div>
<section class="controls" aria-label="Controles de demonstração">
<strong>Estado visual</strong>
<div class="buttons" role="group" aria-label="Moods canônicos">
${['idle','working','warning','critical','offline'].map(m=>`<button data-mood="${m}" aria-pressed="${m==='idle'}">${m}</button>`).join('')}
</div>
<label><input id="motion" type="checkbox" checked> Ativar animações</label>
<label for="yaw">Inclinação simulada <output id="yaw-value">0°</output></label>
<input id="yaw" type="range" min="-12" max="12" step="1" value="0" aria-label="Inclinação horizontal simulada, não rotação real">
<button type="button" id="reset">Centralizar personagem</button>
<div class="info"><strong>Este preview demonstra</strong>
<p class="small">Flutuação, piscada, aletas independentes, quatro módulos, núcleo, anel e respostas visuais aos moods. Sem motor 3D, timers externos, provider ou Core.</p>
<strong>Este preview NÃO demonstra</strong>
<p class="small">Vista real 3/4, costas, spritesheet com rotação completa, atlas PNG pronto, drag nativo do HUD ou homologação. Para 360° são necessárias 12–24 vistas autorais consistentes, sob o mesmo master.</p></div>
</section>
</div></main>
<script>
const art=document.getElementById('nex-site-hero');
const motion=document.getElementById('motion');
const yaw=document.getElementById('yaw');
const stage=document.getElementById('nex-yaw-stage');
const value=document.getElementById('yaw-value');
function update(){
 art.dataset.motion=motion.checked?'on':'off';
 art.dataset.state=document.querySelector('button[data-mood][aria-pressed=true]')?.dataset.mood||'idle';
 const angle=Number(yaw.value);
 const squash=1-Math.abs(angle)/240;
 stage.style.transform='skewY('+(-angle/7).toFixed(2)+'deg) scaleX('+squash.toFixed(3)+')';
 value.textContent=angle+'°';
}
document.querySelectorAll('button[data-mood]').forEach(button=>button.addEventListener('click',()=>{
 document.querySelectorAll('button[data-mood]').forEach(other=>other.setAttribute('aria-pressed',String(button===other)));
 update();
}));
motion.addEventListener('change',update);
yaw.addEventListener('input',update);
document.getElementById('reset').addEventListener('click',()=>{yaw.value='0';update()});
update();
</script></body></html>`;
const output = resolve(web, 'dist/nex-site-hero-preview.html');
await mkdir(dirname(output), {recursive:true});
await writeFile(output, html, 'utf8');
console.log(`NEX Hero animation preview: ${output}`);
