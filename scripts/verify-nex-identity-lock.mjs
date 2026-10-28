#!/usr/bin/env node
// Identity provenance/immutability gate. Hash equality != perceptual similarity.
import { createHash } from 'node:crypto';
import { existsSync, readFileSync } from 'node:fs';
import { resolve, dirname } from 'node:path';
import { execFileSync } from 'node:child_process';
import { fileURLToPath } from 'node:url';

const root = resolve(dirname(fileURLToPath(import.meta.url)), '..');
const path = 'docs/design/nex/identity-lock.json';
const lock = JSON.parse(readFileSync(resolve(root, path), 'utf8'));
const baseArg = process.argv.indexOf('--base-ref');
const baseRef = baseArg >= 0 ? process.argv[baseArg + 1] : null;
const fail = (message) => { throw new Error('NEX identity lock: ' + message); };
const sha = (file) => createHash('sha256').update(readFileSync(resolve(root, file))).digest('hex');
const hex = (value) => typeof value === 'string' && /^[a-f0-9]{64}$/.test(value);

if (lock.subject !== 'IAPro Nexus / NEX visual character') fail('unexpected subject');
if (lock.selection_record) {
  const selection = lock.selection_record;
  const artwork = selection.selected_reference;
  if (selection.status !== 'CHARACTER_SELECTED_PRODUCTION_MASTER_PENDING') fail('unexpected character selection status');
  if (!artwork || !hex(artwork.sha256) || artwork.format !== 'PNG' || !artwork.width ||
      !artwork.height || !Number.isSafeInteger(artwork.bytes) || artwork.bytes <= 0) {
    fail('selected character reference metadata missing or invalid');
  }
  const idleContract = resolve(root, selection.idle_art_contract);
  if (!existsSync(idleContract)) fail('retro game idle contract is missing');
  if (artwork.repository_file) {
    if (!existsSync(resolve(root, artwork.repository_file)) ||
        sha(artwork.repository_file) !== artwork.sha256) fail('selected reference binary SHA mismatch');
  } else {
    console.warn('Selected NEX character reference binary NOT_VERSIONED; pixel art production master still pending.');
  }
}

if (!Array.isArray(lock.candidate_references) || lock.candidate_references.length < 3) fail('reference manifest incomplete');
for (const r of lock.candidate_references) {
  if (!hex(r.sha256) || !r.filename || !r.width || !r.height) fail('invalid reference provenance');
}
if (!['CANDIDATE_PENDING_APPROVAL', 'FROZEN'].includes(lock.status)) fail('invalid status');
if (lock.status === 'FROZEN') {
  if (!/^NEX-ID-v[1-9][0-9]*$/.test(lock.proposed_lock_version ?? '')) fail('version required');
  if (!lock.approved_by || !lock.approved_at_utc || !lock.approval_record) fail('explicit approval record required');
  for (const key of ['file', 'editable_source_file']) {
    const f = lock.master?.[key];
    const hash = lock.master?.[key === 'file' ? 'sha256' : 'editable_source_sha256'];
    if (!f || !hex(hash) || !existsSync(resolve(root, f)) || sha(f) !== hash) fail('missing/mutated master: ' + String(f));
  }
  if (!Array.isArray(lock.locked_assets) || lock.locked_assets.length === 0) fail('versioned derivative assets required');
  for (const asset of lock.locked_assets) {
    if (!asset.file || !hex(asset.sha256) || !existsSync(resolve(root,asset.file)) || sha(asset.file) !== asset.sha256) fail('missing/mutated asset: ' + String(asset.file));
  }
  if (!existsSync(resolve(root, lock.approval_record))) fail('approval record file missing');
} else {
  if (lock.master?.file || lock.approved_by || lock.approved_at_utc || lock.approval_record) fail('cannot silently treat candidate as approved');
  console.warn('NEX identity is a CANDIDATE: no final master frozen or approved. Structural gate only.');
}

if (baseRef) {
  let base;
  try { base = JSON.parse(execFileSync('git', ['show', `${baseRef}:${path}`], {cwd:root,encoding:'utf8',stdio:['ignore','pipe','ignore']})); }
  catch { base = null; }
  if (base?.status === 'FROZEN') {
    const protectedFields = ['master', 'invariants', 'locked_assets', 'proposed_lock_version'];
    const changed = protectedFields.some((field) => JSON.stringify(base[field]) !== JSON.stringify(lock[field]));
    if (changed) {
      if (base.proposed_lock_version === lock.proposed_lock_version) fail('frozen identity changed without bumping the lock version');
      const record = `docs/design/nex/change-requests/${lock.proposed_lock_version}.md`;
      if (!existsSync(resolve(root,record))) fail('identity change requires '+record);
      console.warn('IDENTITY CHANGE proposed; explicit GitHub owner review is still mandatory.');
    }
  }
}
console.log('NEX identity provenance gate OK, status:',lock.status);
