/**
 * Build an offline, single-file NEX animation QA preview directly from the
 * canonical static exports and the real HUD SCSS. No duplicate robot drawing.
 *
 * Run npm run art:nex:preview (exports artwork and builds this QA page).
 * Output: web/dist/nex-motion-preview.html
 */
import { mkdir, readFile, writeFile } from 'node:fs/promises';
import { dirname, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';
import * as sass from 'sass';

const webDir = resolve(dirname(fileURLToPath(import.meta.url)), '..');
const moods = [
  { id: 'idle', label: 'Idle', description: 'Flutuação discreta, olhos alegres e núcleo lento' },
  { id: 'working', label: 'Working', description: 'Movimento curto e núcleo pulsando durante execução' },
  { id: 'warning', label: 'Warning', description: 'Atenção real, alerta laranja e sinal luminoso' },
  { id: 'critical', label: 'Critical', description: 'Atenção crítica, pulso controlado e aletas estáveis' },
  { id: 'offline', label: 'Offline', description: 'Sem atividade aparente; sinais desativados' },
];

const scss = sass.compile(resolve(webDir, 'src/features/hud/_hudGuardian.scss')).css;
const sprites = await Promise.all(moods.map(async ({ id }) => {
  const svg = await readFile(resolve(webDir, 'public/nex', `nex-${id}.svg`), 'utf8');
  if (!svg.includes('data-nex-art="32bit-official"') ||
      !svg.includes('data-nex-facing="front"')) {
    throw new Error(`NEX official SVG export is missing/invalid: nex-${id}.svg`);
  }
  return { id, svg };
}));

const optionButtons = moods.map((mood, index) =>
  `<button type="button" class="qa-choice" data-select="${mood.id}" aria-pressed="${index === 0}">${mood.label}</button>`
).join('');
const panels = sprites.map(({ id, svg }, index) => `
  <div class="qa-panel" data-panel="${id}" ${index > 0 ? 'hidden' : ''}>
    <div class="qa-scale">
      <span class="guardian" data-mood="${id}" data-motion="${id === 'offline' ? 'off' : 'on'}" aria-hidden="true">
        ${svg}
      </span>
    </div>
  </div>`
).join('');
const html = `<!doctype html>
<html lang="pt-BR"><head><meta charset="utf-8"/>
<meta name="viewport" content="width=device-width,initial-scale=1"/>
<title>NEX — QA Motion Preview v1</title>
<style>${scss}</style>
<style>
*{box-sizing:border-box}
body{margin:0;min-height:100vh;background:#080f1f;color:#edf5ff;font:14px system-ui, sans-serif}
main{width:min(860px,100%);margin:auto;padding:32px 18px}
h1{margin:0;font-size:26px}p{color:#9fb6d5;line-height:1.5}
.qa-note{font-size:12px;color:#bed4ed}
.qa-choices{display:flex;flex-wrap:wrap;gap:8px;margin:22px 0}
.qa-choice{padding:10px 12px;border:1px solid #456da3;border-radius:8px;background:#142a47;color:#eaf5ff;font:inherit;cursor:pointer}
.qa-choice[aria-pressed="true"]{background:#24528b;border-color:#67cfff}
.qa-choice:focus-visible,input:focus-visible{outline:2px solid #9be9ff;outline-offset:3px}
.qa-stage{height:320px;display:grid;place-items:center;border:1px solid #284466;border-radius:16px;background:radial-gradient(ellipse at 50% 42%,#15355a,#081326 80%)}
.qa-panel[hidden]{display:none}
.qa-panel{width:100%;height:100%;display:grid;place-items:center}
.qa-scale{width:92px;height:92px;transform:scale(2.25)}
.qa-settings{display:flex;align-items:center;gap:10px;margin-top:16px}
</style>
</head><body><main>
<h1>NEX — prévia técnica de movimentos</h1>
<p>Renderiza os cinco SVGs derivados do componente oficial e os keyframes do HUD.
Isto é QA visual local, <strong>não</strong> captura do HUD nativo nem certificação de release.</p>
<div class="qa-choices" role="group" aria-label="Estado do NEX">${optionButtons}</div>
<div class="qa-stage">${panels}</div>
<p id="qa-description" aria-live="polite">${moods[0].description}</p>
<label class="qa-settings"><input type="checkbox" id="qa-reduced"/> Desativar animações</label>
<p class="qa-note">Os movimentos não alteram telemetria, roteamento, geometria nativa ou foco. 
Em sistema configurado para reduzir movimentos, os elementos já devem aparecer imóveis.</p>
</main><script>
const states=${JSON.stringify(moods)};
let selected='idle';
const freeze=document.getElementById('qa-reduced');
const render=()=>{
  document.querySelectorAll('[data-select]').forEach(button=>{
    button.setAttribute('aria-pressed',String(button.dataset.select===selected));
  });
  document.querySelectorAll('[data-panel]').forEach(panel=>{
    const active=panel.dataset.panel===selected;
    panel.hidden=!active;
    panel.querySelector('.guardian').dataset.motion=
      freeze.checked||selected==='offline'?'off':'on';
  });
  document.getElementById('qa-description').textContent=
    states.find(state=>state.id===selected)?.description||'';
};
document.querySelectorAll('[data-select]').forEach(button=>
  button.addEventListener('click',()=>{selected=button.dataset.select;render();})
);
freeze.addEventListener('change',render);
render();
</script></body></html>`;
const output = resolve(webDir, 'dist/nex-motion-preview.html');
await mkdir(dirname(output), { recursive: true });
await writeFile(output, html, 'utf8');
console.log(`NEX visual motion QA preview ready: ${output}`);
