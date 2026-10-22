#!/usr/bin/env node
// Validate the source-linked capability catalog and the generated public matrix.
// Run with --write to regenerate only the table, not status or certification claims.
import { existsSync, readFileSync, writeFileSync } from 'node:fs';
import { resolve } from 'node:path';
import process from 'node:process';

const root = process.cwd();
const catalogPath = resolve(root, 'docs/reference/features.json');
const matrixPath = resolve(root, 'docs/reference/feature-matrix.md');
const start = '<!-- FEATURE_CATALOG_START -->';
const end = '<!-- FEATURE_CATALOG_END -->';
const validSurfaces = new Set(['core', 'web', 'cli', 'api', 'desktop', 'hud']);
const statuses = new Set(['implemented', 'experimental', 'optional', 'internal']);
const errors = [];
const catalog = JSON.parse(readFileSync(catalogPath, 'utf8'));
if (!Array.isArray(catalog.features) || catalog.features.length === 0) errors.push('features must be a nonempty array');
const ids = new Set();
const features = Array.isArray(catalog.features) ? catalog.features : [];
const source = (p) => existsSync(resolve(root, p));
for (const f of features) {
  if (!f || typeof f !== 'object') { errors.push('non-object feature'); continue; }
  if (typeof f.id !== 'string' || !/^[a-z][a-z0-9-]+$/.test(f.id)) errors.push('invalid feature id: ' + f.id);
  if (ids.has(f.id)) errors.push('duplicate feature id: ' + f.id);
  ids.add(f.id);
  for (const field of ['name', 'implementation', 'documentation', 'requires', 'limits']) {
    if (typeof f[field] !== 'string' || !f[field].trim()) errors.push(f.id + ': missing ' + field);
  }
  if (!statuses.has(f.status)) errors.push(f.id + ': invalid status ' + f.status);
  if (!Array.isArray(f.surfaces) || !f.surfaces.length || f.surfaces.some((s) => !validSurfaces.has(s))) errors.push(f.id + ': invalid surfaces');
  if (!Array.isArray(f.tests) || !f.tests.length) errors.push(f.id + ': test references missing');
  if (f.evidence !== 'SOURCE_TEST_PRESENT_NOT_RUN') errors.push(f.id + ': evidence must remain unverified until external signed gate imports are implemented');
  if (f.nativeCertification !== 'PENDING_EXACT_SHA') errors.push(f.id + ': native certification requires separate exact-SHA evidence');
  for (const field of ['implementation', 'documentation']) {
    if (typeof f[field] === 'string' && (!f[field].startsWith(field === 'documentation' ? 'docs/' : '') || f[field].includes('..') || !source(f[field]))) errors.push(f.id + ': missing or unsafe ' + field + ' ' + f[field]);
  }
  for (const p of Array.isArray(f.tests) ? f.tests : []) {
    if (typeof p !== 'string' || p.includes('..') || !/(_test\.go|\.test\.[jt]sx?)$/.test(p) || !source(p)) errors.push(f.id + ': invalid or missing test: ' + p);
  }
}
const rows = features.map((f) => '| ' + f.id + ' | [' + f.name + '](../' + f.documentation.slice(5) + ') | ' + f.surfaces.map((s) => '`' + s + '`').join(', ') + ' | ' + f.status + ' | ' + f.evidence + ' | ' + f.nativeCertification + ' |');
const expected = [start, '| ID | Recurso e documentação | Superfícies no source | Estado no source | Evidência de testes | Certificação nativa |', '| --- | --- | --- | --- | --- | --- |', ...rows, end].join('\n');
let matrix = readFileSync(matrixPath, 'utf8');
const a = matrix.indexOf(start), b = matrix.indexOf(end);
if (a < 0 || b < a || matrix.indexOf(start, a + start.length) !== -1 || matrix.indexOf(end, b + end.length) !== -1) errors.push('matrix must contain exactly one bounded generated block');
if (errors.length) { for (const err of errors) console.error('catalog: ' + err); process.exitCode = 1; }
else if (process.argv.includes('--write')) {
  matrix = matrix.slice(0, a) + expected + matrix.slice(b + end.length);
  writeFileSync(matrixPath, matrix);
  console.log('feature matrix regenerated (' + features.length + ' features)');
} else if (matrix.slice(a, b + end.length) !== expected) {
  console.error('feature matrix out of date; run node scripts/verify-feature-catalog.mjs --write');
  process.exitCode = 1;
} else console.log('feature catalog verified (' + features.length + ' source/test/doc mappings)');
