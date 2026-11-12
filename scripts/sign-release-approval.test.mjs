import assert from "node:assert/strict";
import { test } from "node:test";
import {
  createPublicKey,
  generateKeyPairSync,
  verify as verifySignature,
} from "node:crypto";
import { mkdtempSync, writeFileSync, readFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { execFileSync } from "node:child_process";
import { canonicalEvidencePayload } from "./offline-release-evidence.mjs";

const SCRIPT = new URL("./sign-release-approval.mjs", import.meta.url).pathname;
const SHA = "0123456789abcdef0123456789abcdef01234567";

function run(args) {
  return execFileSync(process.execPath, [SCRIPT, ...args], {
    encoding: "utf8",
  });
}

// Keys are passed as file paths (the realistic flow), since PEM bodies start
// with "--" and would collide with option parsing.
function writeKeys(dir, privateKey) {
  const keyPath = join(dir, "signing-key.pem");
  writeFileSync(
    keyPath,
    privateKey.export({ type: "pkcs8", format: "pem" }).toString(),
  );
  return keyPath;
}

function writeTrust(dir, publicKey) {
  const trust = join(dir, "reviewers.json");
  writeFileSync(
    trust,
    JSON.stringify({
      schemaVersion: 1,
      reviewers: [
        {
          id: "owner",
          publicKeyPem: publicKey
            .export({ type: "spki", format: "pem" })
            .toString(),
        },
      ],
    }),
  );
  return trust;
}

test("sign then verify round-trip with an enrolled key", () => {
  const dir = mkdtempSync(join(tmpdir(), "approval-"));
  const { publicKey, privateKey } = generateKeyPairSync("ed25519");
  const pem = publicKey.export({ type: "spki", format: "pem" }).toString();
  const trust = writeTrust(dir, publicKey);
  const keyPath = writeKeys(dir, privateKey);
  const out = join(dir, "release-approval.json");
  run([
    "--approver",
    "maestro",
    "--approval-id",
    "GO-1",
    "--source-sha",
    SHA,
    "--key",
    keyPath,
    "--key-id",
    "owner",
    "--out",
    out,
    "--trust",
    trust,
  ]);
  const bundle = JSON.parse(readFileSync(out, "utf8"));
  assert.equal(bundle.schemaVersion, 1);
  assert.equal(bundle.approvalId, "GO-1");
  assert.equal(bundle.sourceSha, SHA);
  assert.equal(bundle.signature.keyId, "owner");

  // Independent verification against the canonical form.
  const { signature, ...payload } = bundle;
  assert.equal(
    verifySignature(
      null,
      Buffer.from(canonicalEvidencePayload(payload)),
      createPublicKey(pem),
      Buffer.from(signature.base64, "base64"),
    ),
    true,
  );
  // The script's own --verify path must accept the bundle too.
  const output = run([
    "--verify",
    "--approval",
    out,
    "--trust",
    trust,
    "--source-sha",
    SHA,
  ]);
  assert.match(output, /approval OK/);
});

test("verify rejects a sha mismatch and refuses to overwrite", () => {
  const dir = mkdtempSync(join(tmpdir(), "approval-bad-"));
  const { publicKey, privateKey } = generateKeyPairSync("ed25519");
  const trust = writeTrust(dir, publicKey);
  const keyPath = writeKeys(dir, privateKey);
  const out = join(dir, "release-approval.json");
  const signingArgs = [
    "--approver",
    "maestro",
    "--approval-id",
    "GO-2",
    "--source-sha",
    SHA,
    "--key",
    keyPath,
    "--key-id",
    "owner",
    "--out",
    out,
    "--trust",
    trust,
  ];
  run(signingArgs);
  const other = "ffffffffffffffffffffffffffffffffffffffff";
  assert.throws(
    () =>
      run([
        "--verify",
        "--approval",
        out,
        "--trust",
        trust,
        "--source-sha",
        other,
      ]),
    /sourceSha/,
  );
  // Overwriting an existing approval must abort (fail closed).
  assert.throws(() => run(signingArgs), /already exists/);
});
