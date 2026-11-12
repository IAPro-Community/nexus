#!/usr/bin/env node
// Signs a release approval payload for the Release Desk
// (capability release-desk). The signed bundle is consumed by
// internal/nexus/release_approval.go through
// docs/reference/release-approval.json in the project repo.
//
// Usage:
//   node scripts/sign-release-approval.mjs \
//     --approver "maestro" --approval-id GO-2026-001 \
//     --source-sha <40-char-sha> --key <ed25519-private.pem> \
//     --key-id maestro-owner --out docs/reference/release-approval.json
//   node scripts/sign-release-approval.mjs --verify \
//     --approval docs/reference/release-approval.json \
//     --trust docs/reference/release-reviewers.json --source-sha <sha>
//
// Fail-closed: missing fields, non-hex SHA, already-signed payload or an
// unenrolled key id abort without writing anything.
import {
  sign as cryptoSign,
  createPublicKey,
  verify as verifySignature,
} from "node:crypto";
import { readFileSync, writeFileSync, existsSync } from "node:fs";
import { canonicalEvidencePayload } from "./offline-release-evidence.mjs";

const SHA = /^[0-9a-f]{40}$/;

function option(name) {
  const at = process.argv.indexOf(name);
  if (at < 0) return null;
  const value = process.argv[at + 1];
  if (!value || value.startsWith("--"))
    throw new Error("Missing value for " + name);
  return value;
}

function die(message) {
  console.error("sign-release-approval: " + message);
  process.exit(2);
}

const trustFile = option("--trust") || "docs/reference/release-reviewers.json";

const verifyOnly = process.argv.includes("--verify");
if (verifyOnly) {
  const approvalFile = option("--approval");
  const sourceSha = (option("--source-sha") || "").toLowerCase();
  if (!approvalFile || !SHA.test(sourceSha))
    die("--verify needs --approval and a 40-char --source-sha");
  const bundle = JSON.parse(readFileSync(approvalFile, "utf8"));
  const trust = JSON.parse(readFileSync(trustFile, "utf8"));
  const reviewer = (trust.reviewers || []).find(
    (r) => r.id === bundle.signature?.keyId,
  );
  if (!reviewer) die("signer is not enrolled in the trust store");
  const { signature, ...payload } = bundle;
  if (payload.sourceSha !== sourceSha)
    die("approval is bound to a different sourceSha");
  const ok = verifySignature(
    null,
    Buffer.from(canonicalEvidencePayload(payload)),
    createPublicKey(reviewer.publicKeyPem),
    Buffer.from(signature.base64, "base64"),
  );
  if (!ok) die("approval signature verification FAILED");
  console.log(
    "approval OK: " +
      payload.approvalId +
      " by " +
      payload.approver +
      " for " +
      payload.sourceSha,
  );
  process.exit(0);
}

const approver = option("--approver");
const approvalId = option("--approval-id");
const sourceSha = (option("--source-sha") || "").toLowerCase();
const keyFile = option("--key");
const keyId = option("--key-id");
const outFile = option("--out");
for (const [name, value] of [
  ["--approver", approver],
  ["--approval-id", approvalId],
  ["--source-sha", sourceSha && SHA.test(sourceSha) ? sourceSha : null],
  ["--key", keyFile],
  ["--key-id", keyId],
  ["--out", outFile],
]) {
  if (!value) die("missing or invalid required option " + name);
}
if (existsSync(outFile)) {
  die(
    outFile +
      " already exists; move or revoke it before issuing a new approval",
  );
}

const approvedAtUtc = new Date().toISOString().replace(/\.\d{3}Z$/, "Z");
const payload = {
  schemaVersion: 1,
  approver,
  approvalId,
  sourceSha,
  approvedAtUtc,
};
const canonical = Buffer.from(canonicalEvidencePayload(payload));
const signature = cryptoSign(null, canonical, readFileSync(keyFile));
const bundle = {
  ...payload,
  signature: { keyId, base64: signature.toString("base64") },
};
writeFileSync(outFile, JSON.stringify(bundle, null, 2) + "\n");
console.log(
  "signed approval written to " +
    outFile +
    " (self-check: node scripts/sign-release-approval.mjs --verify --approval " +
    outFile +
    " --source-sha " +
    sourceSha +
    ")",
);
