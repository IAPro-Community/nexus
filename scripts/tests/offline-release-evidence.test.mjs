import assert from 'node:assert/strict';
import {
  createHash, generateKeyPairSync, sign,
} from 'node:crypto';
import {
  mkdtempSync, mkdirSync, writeFileSync, rmSync,
} from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { afterEach, test } from 'node:test';
import {
  canonicalEvidencePayload, verifyOfflineReleaseEvidence,
} from '../offline-release-evidence.mjs';

const sourceSha = 'a'.repeat(40);
const maestroSha = 'b'.repeat(40);
const platforms = ['linux-amd64', 'windows-amd64', 'macos-arm64'];
const featureIds = Array.from({ length: 48 }, (_, i) => 'feature-' + String(i + 1));
const created = [];
const digest = (bytes) => createHash('sha256').update(bytes).digest('hex');

afterEach(() => {
  for (const dir of created.splice(0)) rmSync(dir, { recursive: true, force: true });
});

function fixture() {
  const root = mkdtempSync(path.join(os.tmpdir(), 'nexus-evidence-test-'));
  created.push(root);
  const { publicKey, privateKey } = generateKeyPairSync('ed25519');
  const trustFile = path.join(root, 'reviewers.json');
  const evidenceFile = path.join(root, 'evidence.json');
  writeFileSync(trustFile, JSON.stringify({
    schemaVersion: 1,
    reviewers: [{ id: 'trusted-release-owner', publicKeyPem: publicKey.export({
      type: 'spki', format: 'pem',
    }) }],
  }));
  const artifacts = new Map();
  function save(relativePath, bytes, kind) {
    const destination = path.join(root, relativePath);
    mkdirSync(path.dirname(destination), { recursive: true });
    writeFileSync(destination, bytes);
    return { path: relativePath, sha256: digest(bytes), kind };
  }
  const records = platforms.map((platform) => {
    const binary = save(platform + '/nexus', Buffer.alloc(1024 * 1024 + 47, 0x43), 'binary');
    const png = Buffer.alloc(512);
    Buffer.from([137, 80, 78, 71, 13, 10, 26, 10]).copy(png);
    png.write('IHDR', 12, 4, 'ascii');
    png.writeUInt32BE(1440, 16);
    png.writeUInt32BE(900, 20);
    const screenshot = save(platform + '/screenshot.png', png, 'screenshot');
    const log = save(platform + '/tests.log', Buffer.from('Local device tests: PASS\n'), 'test-log');
    artifacts.set(platform, { binary, screenshot, log });
    return {
      platform, status: 'passed', runnerType: 'physical',
      sourceSha, maestroSha,
      binarySha256: binary.sha256,
      deviceIdHash: digest(Buffer.from(platform)),
      osVersion: 'test physical fixture only',
      reviewerId: 'trusted-release-owner',
      reviewedAtUtc: new Date().toISOString(),
      artifacts: [binary, screenshot, log],
    };
  });
  const payload = {
    schemaVersion: 2, sourceSha, maestroSha, issuedAtUtc: new Date().toISOString(),
    platforms: records,
    features: featureIds.map((id) => ({
      id, status: 'passed', sourceSha, platforms, tests: ['local-native-test'],
      clients: 2,
    })),
  };
  const signPayload = () => writeFileSync(evidenceFile, JSON.stringify({
    ...payload,
    signature: {
      keyId: 'trusted-release-owner',
      base64: sign(null, Buffer.from(canonicalEvidencePayload(payload)), privateKey).toString('base64'),
    },
  }));
  const verify = () => verifyOfflineReleaseEvidence({
    evidenceFile, trustFile, sourceSha, maestroSha, featureIds,
    twoClientIds: [featureIds[0]],
  });
  signPayload();
  return { root, payload, signPayload, verify, artifacts };
}

test('accepts signed physical proof only when referenced executable bytes match', () => {
  const f = fixture();
  assert.equal(f.verify().verified, true);
});

test('rejects tampered executable even with unchanged valid Ed25519 attestation', () => {
  const f = fixture();
  writeFileSync(path.join(f.root, f.artifacts.get('linux-amd64').binary.path),
    Buffer.from('malicious replacement'));
  assert.throws(f.verify, /SHA-256 mismatch/);
});

test('rejects signed binary digest claim not bound to any actual artifact', () => {
  const f = fixture();
  f.payload.platforms[0].binarySha256 = '0'.repeat(64);
  f.signPayload();
  assert.throws(f.verify, /exact binary artifacts required/);
});

test('rejects physically missing executable artifact even if signed', () => {
  const f = fixture();
  f.payload.platforms[0].artifacts = f.payload.platforms[0].artifacts.filter(
    (artifact) => artifact.kind !== 'binary');
  f.signPayload();
  assert.throws(f.verify, /exact binary artifacts required/);
});

test('rejects ambiguous duplicate paths in signed evidence', () => {
  const f = fixture();
  f.payload.platforms[0].artifacts.push({
    ...f.payload.platforms[0].artifacts[1], kind: 'report',
  });
  f.signPayload();
  assert.throws(f.verify, /Duplicated evidence artifact path/);
});

