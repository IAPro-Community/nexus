#!/usr/bin/env node
// Keep the legacy global workspace cascade explicit and bounded until
// individual selectors can be migrated safely into feature SCSS Modules.
import { createHash } from 'node:crypto';
import { readFileSync, readdirSync } from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const appDir = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '../src/app');
const stylesDir = path.join(appDir, 'workspace-styles');
const entry = readFileSync(path.join(appDir, 'workspace-os.css'), 'utf8');
const sections = [
  '01-foundation.css',
  '02-shell-and-rail.css',
  '03-workspace-layout.css',
  '04-shared-controls.css',
  '05-terminal-and-panels.css',
  '06-responsive-core.css',
  '07-topbar-and-dialogs.css',
  '08-welcome-and-modals.css',
  '09-project-management.css',
  '10-capacity-and-attention.css',
  '11-notifications-and-composer.css',
  '12-zen-and-flow.css',
  '13-communication-and-task-ui.css',
  '14-desktop-windows.css',
  '15-context-and-responsive.css',
  '16-splash-and-settings.css',
];

const errors = [];
const actualImports = Array.from(
  entry.matchAll(/^@import "\.\/workspace-styles\/([^"\n]+)";$/gm),
  (match) => match[1],
);
const unexpectedContent = entry
  .replace(/\/\*[\s\S]*?\*\//g, '')
  .replace(/^@import "\.\/workspace-styles\/[^"\n]+";$/gm, '')
  .trim();
if (unexpectedContent) errors.push('workspace-os.css must contain only comments and ordered imports');
if (JSON.stringify(actualImports) !== JSON.stringify(sections)) {
  errors.push('workspace-os.css import sequence differs from the reviewed section manifest');
}
const actualFiles = readdirSync(stylesDir).filter((name) => name.endsWith('.css')).sort();
if (JSON.stringify(actualFiles) !== JSON.stringify([...sections].sort())) {
  errors.push('workspace-styles contains missing or unexpected global CSS files');
}

const css = sections.map((section) => {
  const content = readFileSync(path.join(stylesDir, section), 'utf8');
  const lines = content.split(/\r?\n/).length;
  if (lines > 1000) errors.push(`${section}: ${lines} lines exceeds 1000-line ceiling`);
  if (!content.trim()) errors.push(`${section}: empty stylesheet`);
  return content;
}).join('');

// Optional one-time parity proof: the exact ordered file contents must match
// the Git blob of workspace-os.css before the mechanical refactor.
// DO NOT run as a regular gate: future intentional CSS edits will change the hash.
if (process.argv.includes('--baseline')) {
  const sourceSha = '877d00b0ec666c4db498682effe4235e298d3166';
  const bytes = Buffer.from(css, 'utf8');
  const actualSha = createHash('sha1')
    .update(`blob ${bytes.length}\0`)
    .update(bytes)
    .digest('hex');
  if (actualSha !== sourceSha) {
    errors.push(`baseline CSS content differs: expected ${sourceSha}, got ${actualSha}`);
  }
}

if (errors.length > 0) {
  for (const error of errors) console.error(`[FAIL] ${error}`);
  process.exitCode = 1;
} else {
  console.log(`[OK] Workspace CSS: ${sections.length} ordered sections with bounded size`);
  if (process.argv.includes('--baseline')) {
    console.log('[OK] Original pre-refactor CSS Git blob preserved byte-for-byte');
  }
}
