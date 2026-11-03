import {
  createHash,
  createPublicKey,
  verify as verifySignature,
} from 'node:crypto';
import { readFileSync, realpathSync, statSync } from 'node:fs';
import { dirname, isAbsolute, relative, resolve, sep } from 'node:path';

const PLATFORM_IDS = Object.freeze(['linux-amd64', 'windows-amd64', 'macos-arm64']);
const HEX = /^[0-9a-f]{64}$/;
const SHA = /^[0-9a-f]{40}$/;
const MAX_BUNDLE_BYTES = 2 * 1024 * 1024;

const canonical = (value) => {
  if (Array.isArray(value)) return '[' + value.map(canonical).join(',') + ']';
  if (value && typeof value === 'object') {
    return '{' + Object.keys(value).sort().map(
      (key) => JSON.stringify(key) + ':' + canonical(value[key]),
    ).join(',') + '}';
  }
  return JSON.stringify(value);
};
function fail(message) {
  const err = new Error(message);
  err.code = 'INVALID_OFFLINE_RELEASE_EVIDENCE';
  throw err;
}
function jsonFile(file) {
  if (statSync(file).size > MAX_BUNDLE_BYTES) fail('Evidence exceeds 2 MiB metadata limit');
  return JSON.parse(readFileSync(file, 'utf8'));
}
function date(value) {
  if (typeof value !== 'string' || !/^\d{4}-\d\d-\d\dT\d\d:\d\d:\d\d(?:\.\d+)?Z$/.test(value)) {
    fail('Evidence timestamp must be explicit UTC');
  }
  const ms = Date.parse(value);
  if (!Number.isFinite(ms) || ms > Date.now() + 300000 || ms < Date.now() - 90 * 86400000) {
    fail('Evidence timestamp is invalid, in the future or stale');
  }
}
function safeFile(base, relativePath, digest, kind) {
  if (typeof relativePath !== 'string' || !relativePath.length ||
      isAbsolute(relativePath) || relativePath.includes('\\') ||
      relativePath.split('/').some((part) => part === '..' || part === '.' || !part)) {
    fail('Unsafe evidence file path');
  }
  if (!HEX.test(digest || '')) fail('Evidence artifact lacks SHA-256');
  const realBase = realpathSync(base);
  const target = realpathSync(resolve(base, relativePath));
  const rel = relative(realBase, target);
  if (!rel || rel.startsWith('..' + sep) || rel === '..' || isAbsolute(rel)) {
    fail('Evidence file escapes external bundle directory');
  }
  const bytes = readFileSync(target);
  if (!bytes.length || bytes.length > 64 * 1024 * 1024 ||
      createHash('sha256').update(bytes).digest('hex') !== digest) {
    fail('Evidence file missing or SHA-256 mismatch');
  }
  if (kind === 'screenshot') {
    const png = Buffer.from([137, 80, 78, 71, 13, 10, 26, 10]);
    if (bytes.length < 400 || !bytes.subarray(0, 8).equals(png) ||
        bytes.toString('ascii', 12, 16) !== 'IHDR' ||
        bytes.readUInt32BE(16) < 320 || bytes.readUInt32BE(20) < 200) {
      fail('Screenshot artifact must be a nontrivial PNG with usable dimensions');
    }
  }
}
export function verifyOfflineReleaseEvidence({
  evidenceFile, trustFile, sourceSha, maestroSha,
  featureIds, twoClientIds = [],
} = {}) {
  if (!SHA.test(sourceSha || '') || !SHA.test(maestroSha || '')) {
    fail('Nexus and Maestro immutable 40-character SHAs required');
  }
  if (!Array.isArray(featureIds) || featureIds.length !== 48 ||
      new Set(featureIds).size !== 48) {
    fail('Exactly 48 distinct release features are required');
  }
  const trust = jsonFile(trustFile);
  const reviewers = Array.isArray(trust.reviewers) ? trust.reviewers : [];
  if (trust.schemaVersion !== 1 || reviewers.length === 0) {
    fail('No owner-authorized reviewers registered in release trust store');
  }
  const doc = jsonFile(evidenceFile);
  const { signature, ...payload } = doc;
  if (payload.schemaVersion !== 2 ||
      payload.sourceSha !== sourceSha || payload.maestroSha !== maestroSha) {
    fail('Offline proof does not match both candidate SHAs');
  }
  date(payload.issuedAtUtc);
  if (!signature || typeof signature.keyId !== 'string' ||
      typeof signature.base64 !== 'string' ||
      !/^[A-Za-z0-9+/]+={0,2}$/.test(signature.base64)) {
    fail('Offline proof requires a detached Ed25519 signature');
  }
  const reviewer = reviewers.find((entry) => entry.id === signature.keyId);
  if (!reviewer || typeof reviewer.publicKeyPem !== 'string') {
    fail('Signer is not enrolled as an authorized release reviewer');
  }
  let authorized = false;
  try {
    const key = createPublicKey(reviewer.publicKeyPem);
    authorized = key.asymmetricKeyType === 'ed25519' &&
      verifySignature(null, Buffer.from(canonical(payload)), key, Buffer.from(signature.base64, 'base64'));
  } catch { /* fail closed for invalid PEM/signature */ }
  if (!authorized) fail('Offline evidence attestation signature failed');
  if (!Array.isArray(payload.platforms) || payload.platforms.length !== PLATFORM_IDS.length ||
      new Set(payload.platforms.map((p) => p?.platform)).size !== PLATFORM_IDS.length) {
    fail('Exactly three distinct native platform attestations required');
  }
  const artifactsBase = dirname(realpathSync(evidenceFile));
  for (const platform of PLATFORM_IDS) {
    const p = payload.platforms.find((item) => item.platform === platform);
    if (!p || p.status !== 'passed' || p.runnerType !== 'physical' ||
        p.sourceSha !== sourceSha || p.maestroSha !== maestroSha ||
        !HEX.test(p.binarySha256 || '') || !HEX.test(p.deviceIdHash || '') ||
        typeof p.osVersion !== 'string' || !p.osVersion.trim() ||
        p.reviewerId !== signature.keyId ||
        !Array.isArray(p.artifacts) || p.artifacts.length < 2) {
      fail('Missing physical-device evidence: ' + platform);
    }
    date(p.reviewedAtUtc);
    if (!p.artifacts.some((item) => item.kind === 'screenshot') ||
        !p.artifacts.some((item) => item.kind === 'test-log')) {
      fail('Visual and executable-test artifacts required for ' + platform);
    }
    for (const artifact of p.artifacts) {
      if (!['screenshot', 'test-log', 'report'].includes(artifact.kind)) {
        fail('Unrecognized evidence artifact kind');
      }
      safeFile(artifactsBase, artifact.path, artifact.sha256, artifact.kind);
    }
  }
  if (!Array.isArray(payload.features) || payload.features.length !== 48 ||
      new Set(payload.features.map((item) => item?.id)).size !== 48) {
    fail('Incomplete or duplicate feature evidence');
  }
  for (const id of featureIds) {
    const item = payload.features.find((record) => record.id === id);
    if (!item || item.status !== 'passed' || item.sourceSha !== sourceSha ||
        !Array.isArray(item.tests) || !item.tests.length ||
        item.tests.some((name) => typeof name !== 'string' || !name.trim() || name.length > 180) ||
        !Array.isArray(item.platforms) || item.platforms.length !== 3 ||
        PLATFORM_IDS.some((platform) => !item.platforms.includes(platform))) {
      fail('Missing native acceptance or test evidence for ' + id);
    }
    if (twoClientIds.includes(id) && (!Number.isInteger(item.clients) || item.clients < 2)) {
      fail('Two authenticated client evidence missing for ' + id);
    }
  }
  return Object.freeze({ verified: true, sourceSha, maestroSha, reviewerId: reviewer.id });
}
export { canonical as canonicalEvidencePayload };
