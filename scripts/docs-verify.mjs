#!/usr/bin/env node
import { existsSync, readFileSync, readdirSync } from 'node:fs';
import { spawnSync } from 'node:child_process';
import path from 'node:path';
import process from 'node:process';

const root = process.cwd();
const required = [
  'README.md', 'README.en.md', 'README.es.md', 'docs/README.md',
  'docs/quick-start.md', 'docs/features/index.md', 'docs/cli/README.md',
  'docs/api/README.md', 'docs/reference/configuration.md',
  'docs/reference/feature-matrix.md', 'docs/faq.md', 'docs/GLOSSARY.md',
  'docs/internal/FEATURE_INVENTORY.md', 'docs/internal/DOCUMENTATION_COVERAGE.md',
  'docs/internal/DOCUMENTATION_EVIDENCE.md', 'DOCUMENTATION_COMPLETE.md',
  'docs/getting-started/installation.md', 'docs/product/overview.md',
  'docs/product/visual-tour.md', 'docs/assets/screenshots/manifest.json',
  'docs/assets/screenshots/workspace-overview.png',
  'docs/assets/demos/direct-workflow.webm',
  'docs/release-status.md',
  'docs/product/how-it-works.md',
  'docs/product/agent-coordination.md',
  'docs/architecture/mission-runtime.md',
  'docs/architecture/guardrails.md',
  'docs/assets/diagrams/nexus-v1-overview.svg',
  'docs/assets/diagrams/nexus-v1-flow.svg',
  'docs/assets/diagrams/nexus-v1-components.svg',
  'docs/assets/diagrams/nexus-v1-agent-coordination.svg',
  'docs/assets/diagrams/nexus-v1-mission-lifecycle.svg',
  'docs/assets/diagrams/nexus-v1-guardrails.svg',
];
const errors = [];
for (const file of required) if (!existsSync(path.join(root, file))) errors.push(`missing required file: ${file}`);

const markdownFiles = [];
const publicMarkdownFiles = [];
function walk(dir) {
  if (!existsSync(dir)) return;
  for (const entry of readdirSync(dir, { withFileTypes: true })) {
    if (entry.name.startsWith('.') || entry.name === 'node_modules' || entry.name === 'DEV') continue;
    const full = path.join(dir, entry.name);
    if (entry.isDirectory()) walk(full);
    else if (entry.isFile() && full.endsWith('.md')) {
      markdownFiles.push(full);
      publicMarkdownFiles.push(full);
    }
  }
}
for (const file of ['README.md', 'README.en.md', 'README.es.md']) markdownFiles.push(path.join(root, file));
walk(path.join(root, 'docs'));
const linkPattern = /!?(?:\[[^\]]*\])\(([^)]+)\)/g;
for (const file of markdownFiles) {
  const text = readFileSync(file, 'utf8');
  let match;
  while ((match = linkPattern.exec(text))) {
    const target = match[1].trim().split(/\s+['"]?/)[0];
    if (!target || /^(?:https?:|mailto:|file:|#)/i.test(target)) continue;
    const clean = target.split('#')[0].replace(/:\d+$/, '');
    const resolved = path.resolve(path.dirname(file), clean);
    if (!existsSync(resolved)) errors.push(`broken local link: ${path.relative(root, file)} -> ${target}`);
    if (/^(?:\.\.\/)*\.?(?:DEV|\.omx)(?:\/|$)/i.test(clean)) errors.push(`public doc links internal path: ${path.relative(root, file)} -> ${target}`);
  }
}
const forbiddenDocumentationData = [
  /\/projetos\//i,
  /escolanet-futura/i,
  /sistemas2/i,
  /proxy-nginx/i,
  /real-local-bootstrap/i,
];
for (const file of publicMarkdownFiles) {
  const text = readFileSync(file, 'utf8');
  for (const pattern of forbiddenDocumentationData) {
    if (pattern.test(text)) errors.push(`public documentation contains non-synthetic data or stale capture marker: ${path.relative(root, file)} -> ${pattern}`);
  }
}
// Link-only mode is a genuine pre-capture preflight. It must not evaluate
// source-SHA visual evidence: screenshots are recaptured *after* this step.
// Full docs verification (default mode) still requires immutable provenance.
if (process.argv.includes('--links-only')) {
  if (errors.length) {
    console.error(errors.map((error) => `✗ ${error}`).join('\\n'));
    process.exit(1);
  }
  console.log(`documentation link preflight passed: ${markdownFiles.length} Markdown files`);
  process.exit(0);
}

const manifestPath = path.join(root, 'docs/assets/screenshots/manifest.json');
if (existsSync(manifestPath)) {
  const manifest = JSON.parse(readFileSync(manifestPath, 'utf8'));
  if (manifest.scenario !== 'isolated-synthetic-fixture') errors.push('visual manifest must use isolated-synthetic-fixture');
  if (manifest.data_classification !== 'SYNTHETIC') errors.push('visual manifest must declare SYNTHETIC data classification');
  const items = Array.isArray(manifest) ? manifest : manifest.captures || [];
  const requiredVisualIds = Array.from({ length: 11 }, (_, index) => `VIS-${String(index + 1).padStart(3, '0')}`);
  const itemById = new Map();
  for (const item of items) {
    if (!item.id) errors.push('visual manifest item has no id');
    else if (itemById.has(item.id)) errors.push(`visual manifest duplicate id: ${item.id}`);
    else itemById.set(item.id, item);
    const historicalNative = item.id === 'VIS-011' &&
      item.status === 'HISTORICAL' &&
      item.source_sha !== manifest.source_sha;
    if (item.status !== 'PASS' && !historicalNative) {
      errors.push(`visual manifest item is not verified: ${item.id || '(unknown)'}`);
    }
    if (item.id !== 'VIS-011' && item.source_sha !== manifest.source_sha) {
      errors.push(`visual capture has a different source SHA: ${item.id || '(unknown)'}`);
    }
    if (item.id === 'VIS-011' && item.source_sha !== manifest.source_sha && !historicalNative) {
      errors.push('historical Desktop capture cannot be marked PASS');
    }
    for (const field of ['surface', 'scenario', 'viewport', 'capture_command', 'source_sha', 'data_classification']) {
      if (!item[field]) errors.push(`visual manifest item ${item.id || '(unknown)'} missing ${field}`);
    }
    if (typeof item.scenario !== 'string' || !item.scenario.startsWith('isolated-synthetic-')) errors.push(`visual manifest item uses unsafe scenario: ${item.id || '(unknown)'}`);
    if (item.data_classification !== 'SYNTHETIC') errors.push(`visual manifest item is not SYNTHETIC: ${item.id || '(unknown)'}`);
    if (item.id === 'VIS-005' && (!item.terminal_evidence || item.terminal_evidence.verified !== true || !item.terminal_evidence.marker)) {
      errors.push('VIS-005 requires verified terminal evidence and marker');
    }
    if (item.source_sha && !/^[0-9a-f]{40}$/i.test(item.source_sha)) {
      errors.push(`visual manifest item has invalid source_sha: ${item.id || '(unknown)'}`);
    }
  }
  for (const id of requiredVisualIds) if (!itemById.has(id)) errors.push(`visual manifest missing required capture: ${id}`);
  for (const item of items) {
    if (!item.file) errors.push('visual manifest item has no file');
    else if (!existsSync(path.join(root, 'docs/assets/screenshots', item.file))) errors.push(`manifest file missing: ${item.file}`);
  }
  if (items.some((item) => item.status === 'PASS' && item.scenario === 'real-local-bootstrap')) {
    errors.push('visual manifest contains PASS evidence from real-local-bootstrap');
  }

  if (manifest.source_sha && !/^[0-9a-f]{40}$/i.test(manifest.source_sha)) {
    errors.push('visual manifest has invalid source_sha');
  }
  if (manifest.source_sha) {
    // Visual capture provenance must track rendered UI, not a new inactive
    // scheduler module and test-only code. Test/spec files do not ship to the
    // renderer. GuardianMascot, NexPixelSprite, styles and other runtime UI
    // still invalidate captures exactly as before.
    const isRenderingChange = (name) =>
      !/\.(?:test|spec)\.(?:ts|tsx)$/.test(name) &&
      name !== 'web/src/features/hud/nexIdleDirector.ts' &&
      // These source candidates are not imported by the live HUD or current
      // screenshot scenarios. Their own hash + browser QA gate is separate.
      !name.startsWith('web/public/nex/hybrid-v1/');
    const visualPaths = [
      'web/src',
      'web/public',
      'cmd/nexus-desktop',
      'internal/desktop',
      'web/scripts/docs-capture.mjs',
      'web/scripts/maestro-visual-verify.mjs',
      'web/scripts/task-preparation-visual-verify.mjs',
    ];
    const diff = spawnSync('git', ['diff', '--name-only', `${manifest.source_sha}..HEAD`, '--', ...visualPaths], {
      cwd: root,
      encoding: 'utf8',
    });
    if (diff.status !== 0) {
      errors.push(
        'visual manifest provenance cannot be verified; fetch full Git history ' +
        '(checkout fetch-depth: 0) and retry source SHA ' + manifest.source_sha,
      );
    } else {
      const actualRenderChanges = (diff.stdout || '')
        .split(/\r?\n/)
        .map((item) => item.trim())
        .filter(Boolean)
        .filter(isRenderingChange);
      if (actualRenderChanges.length) {
        errors.push(`visual manifest is stale after visual changes since ${manifest.source_sha}: ${actualRenderChanges.join(', ')}`);
      }
    }
    // A manifest is evidence for an immutable candidate.  Comparing only
    // against HEAD would incorrectly pass while visual files are dirty in
    // the worktree, so include both staged and unstaged changes as well.
    const dirty = spawnSync('git', ['diff', '--name-only', '--', ...visualPaths], {
      cwd: root,
      encoding: 'utf8',
    });
    const staged = spawnSync('git', ['diff', '--cached', '--name-only', '--', ...visualPaths], {
      cwd: root,
      encoding: 'utf8',
    });
    const dirtyPaths = [...new Set(
      `${dirty.stdout || ''}\n${staged.stdout || ''}`
        .split(/\r?\n/)
        .map((item) => item.trim())
        .filter(Boolean)
        .filter(isRenderingChange),
    )];
    if (dirty.status === 0 && staged.status === 0 && dirtyPaths.length) {
      errors.push(`visual manifest cannot certify uncommitted visual changes for ${manifest.source_sha}: ${dirtyPaths.join(', ')}`);
    }
  }
}
for (const file of ['README.md', 'README.en.md', 'README.es.md']) {
  const text = readFileSync(path.join(root, file), 'utf8');
  for (const phrase of ['TL;DR', 'docs/README.md', 'docs/assets/screenshots/workspace-overview.png']) {
    if (!text.includes(phrase)) errors.push(`${file} missing required phrase: ${phrase}`);
  }
}
const nexVisualGate = spawnSync(process.execPath, [path.join(root, 'scripts/verify-nex-visual.mjs')], { cwd: root, encoding: 'utf8' });
if (nexVisualGate.status !== 0) errors.push('NEX official mascot visual contract failed: ' + (nexVisualGate.stderr || nexVisualGate.stdout || 'unknown error').trim());
const nexHeroGate = spawnSync(process.execPath, [path.join(root, 'scripts/verify-nex-site-hero.mjs')], { cwd: root, encoding: 'utf8' });
if (nexHeroGate.status !== 0) errors.push('NEX site Hero SVG structural gate failed: ' + (nexHeroGate.stderr || nexHeroGate.stdout || 'unknown error').trim());
const nexJointGate = spawnSync(process.execPath, [path.join(root, 'scripts/verify-nex-release-readiness.mjs')], { cwd: root, encoding: 'utf8' });
if (nexJointGate.status !== 0) errors.push('NEX atomic HUD+Site release declaration invalid: ' + (nexJointGate.stderr || nexJointGate.stdout || 'unknown error').trim());
const nexHybridGate = spawnSync(process.execPath, [path.join(root, 'scripts/verify-nex-hybrid-assets.mjs')], { cwd: root, encoding: 'utf8' });
if (nexHybridGate.status !== 0) errors.push('NEX layered Hero raster manifest failed: ' + (nexHybridGate.stderr || nexHybridGate.stdout || 'unknown error').trim());
const nexVisualAuditGate = spawnSync(process.execPath, [path.join(root, 'scripts/verify-nex-image-references.mjs')], { cwd: root, encoding: 'utf8' });
if (nexVisualAuditGate.status !== 0) errors.push('NEX Oct 2026 visual audit manifest failed: ' + (nexVisualAuditGate.stderr || nexVisualAuditGate.stdout || 'unknown error').trim());
const nexPoseGate = spawnSync(process.execPath, [path.join(root, 'scripts/verify-nex-turnaround-candidates.mjs')], { cwd: root, encoding: 'utf8' });
if (nexPoseGate.status !== 0) errors.push('NEX 256px reference poses failed integrity check: ' + (nexPoseGate.stderr || nexPoseGate.stdout || 'unknown error').trim());
const nexReferenceGate = spawnSync(process.execPath, [path.join(root, 'scripts/verify-nex-reference-pack.mjs')], { cwd: root, encoding: 'utf8' });
if (nexReferenceGate.status !== 0) errors.push('NEX pixel-art reference pack failed: ' + (nexReferenceGate.stderr || nexReferenceGate.stdout || 'unknown error').trim());
const catalogGate = spawnSync(process.execPath, [path.join(root, 'scripts/verify-feature-catalog.mjs')], { cwd: root, encoding: 'utf8' });
if (catalogGate.status !== 0) errors.push('feature catalog gate failed: ' + (catalogGate.stderr || catalogGate.stdout || 'unknown error').trim());
const releaseDocsGate = spawnSync(process.execPath, [path.join(root, 'scripts/verify-release-docs.mjs')], { cwd: root, encoding: 'utf8' });
if (releaseDocsGate.status !== 0) errors.push('release docs contract gate failed: ' + (releaseDocsGate.stderr || releaseDocsGate.stdout || 'unknown error').trim());
if (errors.length) {
  console.error(errors.map((error) => `✗ ${error}`).join('\n'));
  process.exitCode = 1;
} else {
  console.log(`docs verification passed: ${markdownFiles.length} Markdown files, ${required.length} required assets/files`);
}
