import { test } from 'node:test';
import assert from 'node:assert/strict';
import { spawnSync } from 'node:child_process';
import { mkdtempSync, readFileSync, writeFileSync, rmSync, mkdirSync } from 'node:fs';
import { join, resolve } from 'node:path';
import { tmpdir } from 'node:os';
import { createHash, generateKeyPairSync, sign } from 'node:crypto';
import { deflateSync } from 'node:zlib';
import { verifyOfflineReleaseEvidence, canonicalEvidencePayload } from './offline-release-evidence.mjs';

const root = resolve(import.meta.dirname, '..');
const script = join(root, 'scripts/verify-agent-workspace-v1.mjs');
const manifest = join(root, 'docs/reference/agent-workspace-v1.json');
function run(args) {
  return spawnSync(process.execPath, [script, ...args], {cwd: root, encoding: 'utf8'});
}

test('source contract is structurally complete without pretending to certify it', () => {
  const p = run([]);
  assert.equal(p.status, 0, p.stderr);
  assert.match(p.stdout, /certification still pending/);
});

test('release refuses unverified contract without externally produced evidence', () => {
  const p = run(['--release', '--sha', '0'.repeat(40), '--ci-run-id', '123']);
  assert.notEqual(p.status, 0);
  assert.match(p.stderr, /external exact-SHA evidence|Checkout SHA differs/);
});

test('contract cannot silently drop, duplicate or waive required features', () => {
  const temp = mkdtempSync(join(tmpdir(), 'nexus-v1-scope-'));
  try {
    const doc = JSON.parse(readFileSync(manifest, 'utf8'));
    doc.features[0].releaseRequired = false;
    doc.features[1].id = doc.features[0].id;
    const mutated = join(temp, 'altered.json');
    writeFileSync(mutated, JSON.stringify(doc));
    const p = run(['--manifest', mutated]);
    assert.notEqual(p.status, 0);
    assert.match(p.stderr, /cannot drop mandatory scope/);
    assert.match(p.stderr, /Duplicated feature/);
  } finally { rmSync(temp, {recursive: true, force: true}); }
});

const sha256 = (bytes) => createHash('sha256').update(bytes).digest('hex');
function crc32(data) {
  let crc = 0xffffffff;
  for (const byte of data) {
    crc ^= byte;
    for (let bit = 0; bit < 8; bit++) crc = (crc >>> 1) ^ ((crc & 1) ? 0xedb88320 : 0);
  }
  return (crc ^ 0xffffffff) >>> 0;
}
function pngChunk(type, data) {
  const kind = Buffer.from(type, 'ascii');
  const length = Buffer.alloc(4);
  length.writeUInt32BE(data.length);
  const crc = Buffer.alloc(4);
  crc.writeUInt32BE(crc32(Buffer.concat([kind, data])));
  return Buffer.concat([length, kind, data, crc]);
}
function syntheticPngFixture() {
  const width = 320, height = 240;
  const ihdr = Buffer.alloc(13);
  ihdr.writeUInt32BE(width, 0);
  ihdr.writeUInt32BE(height, 4);
  ihdr[8] = 8; ihdr[9] = 6;
  const rows = Buffer.alloc(height * (1 + width * 4));
  for (let y = 0; y < height; y++) {
    for (let x = 0; x < width; x++) {
      const o = y * (1 + width * 4) + 1 + x * 4;
      rows[o] = (x * 73 + y) & 255;
      rows[o + 1] = (x + y * 37) & 255;
      rows[o + 2] = (x ^ y) & 255;
      rows[o + 3] = 255;
    }
  }
  return Buffer.concat([
    Buffer.from([137, 80, 78, 71, 13, 10, 26, 10]),
    pngChunk('IHDR', ihdr),
    pngChunk('IDAT', deflateSync(rows)),
    pngChunk('IEND', Buffer.alloc(0)),
  ]);
}
function offlineFixture(dir) {
  const sourceSha = 'a'.repeat(40), maestroSha = 'b'.repeat(40);
  const ids = Array.from({ length: 48 }, (_, i) => 'feature-' + (i + 1));
  const signer = generateKeyPairSync('ed25519');
  const trustFile = join(dir, 'authorized-reviewers.json');
  writeFileSync(trustFile, JSON.stringify({
    schemaVersion: 1, reviewers: [{
      id: 'release-owner', publicKeyPem: signer.publicKey.export({ type: 'spki', format: 'pem' }),
    }],
  }));
  const platforms = ['linux-amd64', 'windows-amd64', 'macos-arm64'];
  const png = syntheticPngFixture();
  const rows = platforms.map((platform) => {
    const folder = join(dir, platform);
    mkdirSync(folder, { recursive: true });
    const image = join(folder, 'image.png'), log = join(folder, 'tests.log'), binary = join(folder, 'nexus.fixture');
    const binaryBytes = Buffer.from('Synthetic release evidence fixture only; never a distributable binary.');
    writeFileSync(image, png);
    writeFileSync(log, 'Synthetic test fixture only. NEVER certify this as a device run.');
    writeFileSync(binary, binaryBytes);
    return {
      platform, status: 'passed', runnerType: 'physical',
      sourceSha, maestroSha, binarySha256: sha256(binaryBytes),
      deviceIdHash: sha256('synthetic-device-' + platform),
      osVersion: 'synthetic-fixture', reviewerId: 'release-owner',
      reviewedAtUtc: new Date().toISOString(),
      artifacts: [
        { kind: 'screenshot', path: platform + '/image.png', sha256: sha256(png) },
        {
          kind: 'test-log', path: platform + '/tests.log',
          sha256: sha256('Synthetic test fixture only. NEVER certify this as a device run.'),
        },
        { kind: 'binary', path: platform + '/nexus.fixture', sha256: sha256(binaryBytes) },
      ],
    };
  });
  const payload = {
    schemaVersion: 2, sourceSha, maestroSha, issuedAtUtc: new Date().toISOString(),
    platforms: rows,
    features: ids.map((id) => ({
      id, sourceSha, status: 'passed', tests: ['synthetic fixture assertion'],
      clients: 2, platforms: [...platforms],
    })),
  };
  const evidenceFile = join(dir, 'bundle.json');
  function writeSigned(value = payload) {
    const base64 = sign(null, Buffer.from(canonicalEvidencePayload(value)), signer.privateKey).toString('base64');
    writeFileSync(evidenceFile, JSON.stringify({
      ...value, signature: { keyId: 'release-owner', base64 },
    }));
  }
  writeSigned();
  return {
    sourceSha, maestroSha, ids, trustFile, evidenceFile,
    payload, writeSigned,
    check: () => verifyOfflineReleaseEvidence({
      sourceSha, maestroSha, featureIds: ids,
      twoClientIds: [ids[0]], evidenceFile, trustFile,
    }),
  };
}

test('signed offline proof checks all 48 features, exact SHAs and external binary artifacts', () => {
  const temp = mkdtempSync(join(tmpdir(), 'nexus-offline-release-'));
  try {
    const fixture = offlineFixture(temp);
    const result = fixture.check();
    assert.equal(result.verified, true);
    assert.equal(result.reviewerId, 'release-owner');
    const forged = JSON.parse(readFileSync(fixture.evidenceFile, 'utf8'));
    forged.platforms[0].binarySha256 = sha256('different');
    writeFileSync(fixture.evidenceFile, JSON.stringify(forged));
    assert.throws(fixture.check, /signature failed/);
    fixture.writeSigned();
    writeFileSync(join(temp, 'linux-amd64', 'tests.log'), 'tampered');
    assert.throws(fixture.check, /SHA-256 mismatch/);
  } finally { rmSync(temp, { recursive: true, force: true }); }
});

test('offline verification fails closed on unknown reviewer, missing device and incomplete two-client proof', () => {
  const temp = mkdtempSync(join(tmpdir(), 'nexus-offline-release-'));
  try {
    const fixture = offlineFixture(temp);
    writeFileSync(fixture.trustFile, JSON.stringify({ schemaVersion: 1, reviewers: [] }));
    assert.throws(fixture.check, /No owner-authorized reviewers/);
    const signer = generateKeyPairSync('ed25519');
    const trust = {
      schemaVersion: 1, reviewers: [{
        id: 'release-owner', publicKeyPem: signer.publicKey.export({ type: 'spki', format: 'pem' }),
      }],
    };
    writeFileSync(fixture.trustFile, JSON.stringify(trust));
    assert.throws(fixture.check, /signature failed/);
  } finally { rmSync(temp, { recursive: true, force: true }); }
});
