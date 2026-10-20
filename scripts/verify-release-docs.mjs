#!/usr/bin/env node
// Fast consistency checks for documentation tied to source manifests.
import { readFileSync } from 'node:fs';
import process from 'node:process';
const read = (p) => readFileSync(p, 'utf8');
const go = read('go.mod');
const web = JSON.parse(read('web/package.json'));
const notices = read('docs/legal/THIRD_PARTY_NOTICES.md');
const platform = read('docs/operations/platform-support.md');
const hud = read('docs/product/hud.md');
const errors = [];
const wails = go.match(/github\.com\/wailsapp\/wails\/v3\s+(\S+)/);
if (!wails) errors.push('source lacks expected Wails v3 module; re-audit desktop');
else if (!notices.includes('github.com/wailsapp/wails/v3 ' + wails[1])) errors.push('third party notices stale Wails version');
if (/github\.com\/wailsapp\/wails\/v2/.test(notices)) errors.push('legacy Wails v2 in notices');
if (!notices.includes('NÃO AUDITADO COMO NOTICE JURÍDICO FINAL')) errors.push('third party notice must not be claimed as audited');
if (!platform.includes('PENDING_EXACT_SHA') && !platform.includes('PENDING')) errors.push('platform evidence not explicitly pending');
const windows = read('.github/workflows/windows-beta10-homologation.yml').match(/go-version:\s*'?([\d.]+)'?/);
if (!windows || !platform.includes('`' + windows[1] + '`')) errors.push('Windows workflow Go version is not documented');
const linux = read('.github/workflows/ci.yml').match(/go-version:\s*'([\d.]+)'/);
if (!linux || !platform.includes('`' + linux[1] + '`')) errors.push('Linux CI Go version is not documented');
if (!hud.includes('release/v1.0.0')) errors.push('HUD doc must reference canonical release branch');
if (web.version !== read('VERSION').trim()) errors.push('Web/source VERSION mismatch');
if (errors.length) { errors.forEach((e) => console.error('release docs: ' + e)); process.exitCode = 1; }
else console.log('release source/documentation manifest checks passed');
