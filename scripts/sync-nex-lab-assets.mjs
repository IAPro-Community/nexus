#!/usr/bin/env node
/** Copy verified NEX candidate assets from this Nexus checkout into a Site lab. */
import { createHash } from 'node:crypto';
import { mkdirSync, readFileSync, writeFileSync } from 'node:fs';
import { dirname, resolve, sep } from 'node:path';
import { fileURLToPath } from 'node:url';
import { execFileSync } from 'node:child_process';

const nexusRoot = resolve(dirname(fileURLToPath(import.meta.url)), '..');
const siteArg = process.argv.find((arg) => arg.startsWith('--site-root='));
if (!siteArg) {
  console.error('Usage: node scripts/sync-nex-lab-assets.mjs --site-root=<Nexus-Site checkout>');
  process.exit(2);
}
const siteRoot = resolve(siteArg.slice('--site-root='.length));
const destinationRoot = resolve(siteRoot, 'public/nex/hybrid-v1');
if (!destinationRoot.startsWith(siteRoot + sep)) throw new Error('Site asset path escaped repository root');

const sourceManifestPath = resolve(nexusRoot, 'docs/design/nex/hybrid-v1/asset-manifest.json');
const sourceManifestBytes = readFileSync(sourceManifestPath);
const sourceManifest = JSON.parse(sourceManifestBytes.toString('utf8'));
const digest = (bytes) => createHash('sha256').update(bytes).digest('hex');
const canonicalText = (filename, bytes) => /\.(html|json)$/.test(filename)
  ? Buffer.from(bytes.toString('utf8').replace(/\r\n/g, '\n'), 'utf8')
  : bytes;
const sourceAssets = sourceManifest.assets.filter((asset) =>
  asset.status === 'VERSIONED' && asset.path.startsWith('web/public/nex/hybrid-v1/'));
if (sourceAssets.length !== 13) throw new Error(`Expected 13 versioned lab assets, found ${sourceAssets.length}`);

const copiedAssets = [];
for (const asset of sourceAssets) {
  const source = resolve(nexusRoot, asset.path);
  const destination = resolve(destinationRoot, asset.filename);
  if (!destination.startsWith(destinationRoot + sep)) throw new Error(`Unsafe asset filename: ${asset.filename}`);
  const bytes = canonicalText(asset.filename, readFileSync(source));
  const sha256 = digest(bytes);
  if (sha256 !== asset.sha256 || bytes.length !== asset.bytes) {
    throw new Error(`Nexus source checksum mismatch: ${asset.filename}`);
  }
  mkdirSync(dirname(destination), { recursive: true });
  writeFileSync(destination, bytes);
  const copied = readFileSync(destination);
  if (digest(copied) !== sha256 || copied.length !== bytes.length) {
    throw new Error(`Site copy checksum mismatch: ${asset.filename}`);
  }
  copiedAssets.push({ filename: asset.filename, sha256, bytes: bytes.length });
}

const sourceCommit = execFileSync('git', ['rev-parse', 'HEAD'], { cwd: nexusRoot, encoding: 'utf8' }).trim();
const sourceBranch = execFileSync('git', ['branch', '--show-current'], { cwd: nexusRoot, encoding: 'utf8' }).trim();
const labManifest = {
  schema_version: '1.0',
  character: 'NEX',
  status: 'CANDIDATE_PENDING_VISUAL_APPROVAL',
  source_repository: 'IAPro-Community/nexus',
  release_target: 'release/v1.0.0',
  source_branch: sourceBranch,
  source_commit: sourceCommit,
  source_sha256: sourceManifest.source_sha256,
  source_manifest_sha256: digest(sourceManifestBytes),
  assets: copiedAssets,
  approval: { human: false, production: false },
};
writeFileSync(resolve(destinationRoot, 'asset-manifest.json'), `${JSON.stringify(labManifest, null, 2)}\n`);
console.log(`Copied and verified ${copiedAssets.length} NEX lab assets from ${sourceCommit}. Candidate approval remains pending.`);
