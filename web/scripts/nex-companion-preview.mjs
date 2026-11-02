/**
 * Self-contained QA preview: the SAME five official SVG exports plus the
 * compiled HUD CSS and GuardianMascot SCSS. No new mascot art or engine.
 * cd web && npm run art:nex:companion:preview
 * Opens: dist/nex-companion-preview.html (local file, offline).
 * The manual switches below are TEST FIXTURES, not Core telemetry.
 */
import { mkdir, readFile, writeFile } from 'node:fs/promises';
import { resolve, dirname } from 'node:path';
import { fileURLToPath } from 'node:url';
import * as sass from 'sass';

const web = resolve(dirname(fileURLToPath(import.meta.url)), '..');
const source = resolve(web, 'public/nex');
const moods = ['idle', 'working', 'warning', 'critical', 'offline'];
const assets = Object.fromEntries(await Promise.all(moods.map(async (mood) => {
  const svg = await readFile(resolve(source, `nex-${mood}.svg`), 'utf8');
  if (!svg.includes('data-nex-art="32bit-official"')) {
    throw new Error(`Unverified NEX art: nex-${mood}.svg`);
  }
  return [mood, svg];
})));
const css = [
  sass.compile(resolve(web, 'src/features/hud/_hudGuardian.scss')).css,
  sass.compile(resolve(web, 'src/features/hud/GuardianMascot.module.scss')).css,
].join('\n');
const behaviorList = [
  ['ambient', 'Repouso', 'idle'],
  ['greet', 'Cumprimento', 'idle'],
  ['look', 'Olhar', 'idle'],
  ['thinking', 'Trabalhando', 'working'],
  ['attention', 'Precisa de você', 'warning'],
  ['error', 'Falha crítica', 'critical'],
  ['resolved', 'Missão concluída', 'idle'],
  ['rest', 'Sonolência', 'idle'],
  ['still', 'Desconectado', 'offline'],
];
const buttons = behaviorList.map(([id, label], index) =>
  `<button type="button" data-behavior-choice="${id}" aria-pressed="${index === 0}">${label}</button>`
).join('\n');
const html = `<!doctype html>
<html lang="pt-BR"><head><meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<meta name="robots" content="noindex,nofollow">
<title>NEX Companion 2D — QA offline</title>
<style>${css}</style>
<style>
*{box-sizing:border-box}
body{margin:0;background:#081326;color:#e6f2ff;font:14px system-ui}
main{max-width:1080px;margin:auto;padding:24px}
h1{font-size:26px}
p{line-height:1.6;color:#bdd2ec}
.controls{display:flex;flex-wrap:wrap;gap:8px;margin:24px 0}
.controls button{font:inherit;color:inherit;background:#183555;border:1px solid #3b6b9f;border-radius:9px;padding:10px 13px;cursor:pointer}
.controls button[aria-pressed="true"]{background:#25649a;border-color:#69d9ff}
.controls button:focus-visible{outline:2px solid #b4eeff;outline-offset:3px}
.stage{height:370px;display:grid;place-items:center;border:1px solid #335172;border-radius:18px;background:radial-gradient(circle,#153a5c,#061224)}
.stage .root{width:240px;height:240px;align-items:center;justify-content:center}
.stage .group{align-items:center;min-height:180px}
.stage .guardianButton{width:180px;height:180px;flex-basis:180px}
.stage .guardian{width:180px;height:180px}
.stage .guardianSprite{width:100%;height:100%}
fieldset{border:0;padding:0;display:flex;gap:12px;align-items:center}
small{display:block;margin-top:16px;color:#a7bddb}
@media(max-width:620px){main{padding:15px}.stage{height:290px}.controls button{min-height:44px}}
</style></head>
<body><main>
<h1>NEX Companion 2D — laboratório de 8 comportamentos</h1>
<p>Prévia técnica com a arte SVG atual do HUD. Os botões simulam reações
para revisão visual; em produção, trabalho, erro, atenção e conclusão exigem evidência real.
Nenhuma ação aqui executa uma missão.</p>
<div class="controls" role="group" aria-label="Comportamentos para revisão">${buttons}</div>
<div class="stage">
  <div class="root" id="companion" data-mood="idle" data-motion="on"
    data-nex-behavior="ambient" data-nex-look="right">
    <div class="group"><button class="guardianButton" type="button"
      aria-label="NEX visual, interações simuladas" id="mascot">
      <span class="guardian" id="figure" data-mood="idle" data-motion="on">
        ${assets.idle}
      </span>
    </button></div>
  </div>
</div>
<p id="status" role="status">Simulação de repouso. Nenhuma telemetria conectada.</p>
<fieldset><label><input type="checkbox" id="motion" checked> Permitir movimento</label>
<label><input type="checkbox" id="look-left"> Olhar à esquerda</label></fieldset>
<small>Saída de QA isolada. Não substitui testes nativos, aprovação artística nem release.</small>
</main>
<script>
const sprites=${JSON.stringify(assets)};
const choices=${JSON.stringify(behaviorList)};
const companion=document.getElementById('companion');
const figure=document.getElementById('figure');
const status=document.getElementById('status');
const motion=document.getElementById('motion');
const left=document.getElementById('look-left');
let choice='ambient';
function update() {
  const entry=choices.find(x=>x[0]===choice);
  const mood=entry[2];
  // Reset once so a repeated manual gesture can replay; never changes Core state.
  companion.removeAttribute('data-nex-behavior');
  void companion.offsetWidth;
  companion.dataset.nexBehavior=motion.checked?choice:'still';
  companion.dataset.nexLook=left.checked?'left':'right';
  companion.dataset.motion=motion.checked?'on':'off';
  figure.dataset.motion=motion.checked?'on':'off';
  figure.dataset.mood=mood;
  if(figure.dataset.assetMood!==mood){
    figure.innerHTML=sprites[mood];
    figure.dataset.assetMood=mood;
  }
  status.textContent='Demonstração (não operacional): '+entry[1];
  document.querySelectorAll('[data-behavior-choice]').forEach(button=>
    button.setAttribute('aria-pressed', String(button.dataset.behaviorChoice===choice)));
}
document.querySelectorAll('[data-behavior-choice]').forEach(button=>
  button.addEventListener('click',()=>{choice=button.dataset.behaviorChoice;update();}));
motion.addEventListener('change',update);
left.addEventListener('change',update);
update();
</script></body></html>`;
const destination = resolve(web, 'dist/nex-companion-preview.html');
await mkdir(dirname(destination), { recursive: true });
await writeFile(destination, html, 'utf8');
console.log(`NEX 2D Companion QA preview: ${destination}`);
