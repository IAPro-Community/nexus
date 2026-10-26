#!/usr/bin/env node
/**
 * Version the approved NEX raster from an existing local copy.
 *
 * Usage:
 *   node scripts/import-nex-concept.mjs --source /path/to/guia_visual_oficial_do_mascote_nex.png
 *
 * The raster's SHA-256, dimensions and byte length must match the approved
 * manifest before it is copied. Run git add/commit after review.
 * Never fetch unknown URLs or convert/resize this master automatically.
 */
import { createHash } from 'node:crypto';
import { existsSync, mkdirSync, readFileSync, writeFileSync } from 'node:fs';
import { dirname, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';

const root = resolve(dirname(fileURLToPath(import.meta.url)), '..');
const manifestPath = resolve(root, 'docs/design/NEX_VISUAL_MANIFEST.json');
const argv = process.argv.slice(2);
const sourceIndex = argv.indexOf('--source');
if (sourceIndex === -1 || !argv[sourceIndex + 1]) {
  console.error('Usage: node scripts/import-nex-concept.mjs --source <approved-original.png>');
  process.exit(2);
}

const manifest = JSON.parse(readFileSync(manifestPath, 'utf8'));
const entry = manifest.references.find((item) => item.id === 'approved_high_fidelity_concept');
if (!entry) throw new Error('Approved visual source is missing from manifest');
const sourcePath = resolve(argv[sourceIndex + 1]);
const targetPath = resolve(root, entry.expected_repository_path);
const bytes = readFileSync(sourcePath);
const sha256 = createHash('sha256').update(bytes).digest('hex');
const signature = bytes.subarray(0, 8).toString('hex');
if (sha256 !== entry.sha256) throw new Error('Source SHA-256 differs from approved original');
if (bytes.length !== entry.bytes) throw new Error('Source length differs from approved original');
if (signature !== '89504e470d0a1a0a' || bytes.length < 24) {
  throw new Error('Source is not a valid PNG header');
}
const width = bytes.readUInt32BE(16);
const height = bytes.readUInt32BE(20);
if (width !== entry.width || height !== entry.height) {
  throw new Error('Source dimensions differ from approved original');
}
if (existsSync(targetPath)) {
  const existing = readFileSync(targetPath);
  if (createHash('sha256').update(existing).digest('hex') !== sha256) {
    throw new Error('Destination exists with different contents');
  }
} else {
  mkdirSync(dirname(targetPath), { recursive: true });
  writeFileSync(targetPath, bytes, { flag: 'wx' });
}
entry.file = entry.expected_repository_path;
entry.status = 'VERSIONED';
entry.ingested_at = new Date().toISOString();
manifest.last_contract_revision = '2026-10-26';
writeFileSync(manifestPath, JSON.stringify(manifest, null, 2) + '\n', 'utf8');
console.log('Original NEX concept imported and verified. Review, stage and commit both files:');
console.log('  ' + entry.file);
console.log('  docs/design/NEX_VISUAL_MANIFEST.json');
console.log('SHA-256: ' + sha256);
