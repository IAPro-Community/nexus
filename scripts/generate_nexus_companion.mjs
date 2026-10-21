#!/usr/bin/env node
// Reproducible, dependency-free vector sprite generator for Nexus Companion.
// Run: node scripts/generate_nexus_companion.mjs
import { mkdir, writeFile } from 'node:fs/promises';
import { resolve } from 'node:path';
import { fileURLToPath } from 'node:url';

const repoRoot = resolve(fileURLToPath(new URL('.', import.meta.url)), '..');
const output = resolve(repoRoot, 'web/public/companion');
const size = 96;
const states = ['idle', 'working', 'attention', 'critical', 'offline', 'success'];
const colors = ['#59cdf5', '#59cdf5', '#ffb34d', '#ff785e', '#9babc1', '#5ddcaa'];
const faces = {
  idle: ['arc', 'arc', 'blink', 'arc', 'arc', 'look'],
  working: ['look', 'look', 'arc', 'blink', 'look', 'look'],
  attention: ['round', 'round', 'blink', 'round', 'round', 'round'],
  critical: ['cross', 'cross', 'blink', 'cross', 'cross', 'cross'],
  offline: ['sleep', 'sleep', 'sleep', 'blink', 'sleep', 'sleep'],
  success: ['arc', 'arc', 'blink', 'arc', 'arc', 'arc'],
};
const esc = (n) => Number(n).toString();
function frame(column, row) {
  const shift = [0, -2, -3, -2, 0, 1][column];
  const eye = faces[states[row]][column];
  const tint = colors[row];
  const look = column === 5 || eye === 'look' ? 2 : 0;
  const eyes = eye === 'blink'
    ? `<path d="M39 53h8M60 53h8" stroke="${tint}" stroke-width="3" stroke-linecap="round"/>`
    : eye === 'sleep'
      ? '<path d="M38 54q5 4 10 0M58 54q5 4 10 0" fill="none" stroke="#a3b7cc" stroke-width="3" stroke-linecap="round"/>'
      : eye === 'cross'
        ? `<path d="M39 48l8 8m0-8l-8 8m20-8l8 8m0-8l-8 8" stroke="${tint}" stroke-width="3" stroke-linecap="round"/>`
        : eye === 'round'
          ? `<ellipse cx="43" cy="52" rx="4" ry="6" fill="${tint}"/><ellipse cx="63" cy="52" rx="4" ry="6" fill="${tint}"/>`
          : `<path d="M39 ${52 + look}q4 -5 9 0M58 ${52 + look}q4 -5 9 0" fill="none" stroke="${tint}" stroke-width="3.7" stroke-linecap="round"/>`;
  return `<g transform="translate(${esc(column * size)} ${esc(row * size + shift)})">
<ellipse cx="51" cy="84" rx="33" ry="4" fill="#182234" opacity=".32"/>
<ellipse cx="50" cy="45" rx="35" ry="25" fill="none" stroke="${tint}" stroke-width="2" transform="rotate(-17 50 45)" opacity="${row === 4 ? '.35' : '.85'}"/>
<path d="M25 36L16 26l0 29 9-3M75 36L83 26v29l-9-3" fill="#3b72e1" stroke="#182234" stroke-width="3" stroke-linejoin="round"/>
<path d="M21 49q0-28 28-30h5q28 3 28 30v11q-3 25-30 26Q23 84 21 61z" fill="#70a4ea" stroke="#182234" stroke-width="3"/>
<path d="M22 51q5-25 28-25h7q20 4 24 25v12q-6 19-30 19Q25 79 22 61z" fill="#3b72e1"/>
<rect x="30" y="38" width="44" height="33" rx="15" fill="#182234"/>
${eyes}
<path d="M48 62l4 3 4-3" fill="none" stroke="${row === 4 ? '#94a3b8' : '#f8fafc'}" stroke-width="2" stroke-linecap="round"/>
<rect x="38" y="76" width="26" height="7" rx="3" fill="#182234"/>
<path d="M43 79l4-2 4 2-4 2M54 80h6" fill="none" stroke="#f8fafc" stroke-width="1.4" stroke-linecap="round"/>
${row === 2 || row === 3 ? `<path d="M73 18l3-7M81 22l7-3" stroke="${tint}" stroke-width="2.5" stroke-linecap="round"/>` : ''}
</g>`;
}
const atlas = `<svg xmlns="http://www.w3.org/2000/svg" width="576" height="576" viewBox="0 0 576 576"><title>Nexus Guardian 2D sprite atlas</title><desc>Six frame loops, six rows: idle, working, attention, critical, offline, success. Each frame is 96 square pixels.</desc>${states.flatMap((name, row) => Array.from({ length: 6 }, (_, col) => frame(col, row))).join('')}</svg>`;
const bucket = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 64 70" width="64" height="70">
<title>Cápsula de pendências do Nexus</title><path d="M12 20h41v35q0 8-8 8H20q-8 0-8-8z" fill="#182234" stroke="#3b72e1" stroke-width="3"/><path d="M19 18V7h28v11" fill="none" stroke="#70a4ea" stroke-width="3"/><path d="M21 12h24v22H21z" fill="#f8fafc" opacity=".8"/><path d="M17 20h30v21H17z" fill="#dbeafe" stroke="#70a4ea"/><path d="M12 35h41v19q0 9-8 9H20q-8 0-8-9z" fill="#182234" stroke="#70a4ea" stroke-width="2"/><path d="M22 48l6-4 6 4-6 4M36 52h10" stroke="#f8fafc" stroke-width="2" fill="none" stroke-linecap="round"/></svg>`;
await mkdir(output, { recursive: true });
await writeFile(resolve(output, 'nexus-guardian-sprites.svg'), atlas, 'utf8');
await writeFile(resolve(output, 'nexus-task-capsule.svg'), bucket, 'utf8');
console.log('Nexus Companion: generated 6x6 vector atlas and task capsule.');
