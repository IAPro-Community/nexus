import { test } from "node:test";
import assert from "node:assert/strict";
import { spawnSync } from "node:child_process";
import { mkdtempSync, writeFileSync, rmSync } from "node:fs";
import { join } from "node:path";
import { tmpdir } from "node:os";
import { generateKeyPairSync } from "node:crypto";
import { verifyOfflineReleaseEvidence } from "./offline-release-evidence.mjs";

const script = join(import.meta.dirname, "sign-offline-release-evidence.mjs");

function run(args) {
  return spawnSync(process.execPath, [script, ...args], { encoding: "utf8" });
}

function basePayload() {
  return {
    schemaVersion: 2,
    sourceSha: "a".repeat(40),
    maestroSha: "b".repeat(40),
    issuedAtUtc: new Date().toISOString().replace(/\.\d{3}Z$/, "Z"),
    platforms: [
      { platform: "linux-amd64", status: "passed", runnerType: "physical" },
    ],
    features: [],
  };
}

function makeKey() {
  const { privateKey, publicKey } = generateKeyPairSync("ed25519");
  return {
    privatePem: privateKey.export({ type: "pkcs8", format: "pem" }).toString(),
    publicPem: publicKey.export({ type: "spki", format: "pem" }).toString(),
  };
}

test("signed bundle passes signature verification and reaches structural checks", () => {
  const folder = mkdtempSync(join(tmpdir(), "nexus-sign-"));
  try {
    const { privatePem, publicPem } = makeKey();
    const keyFile = join(folder, "private.pem");
    const payloadFile = join(folder, "payload.json");
    const outFile = join(folder, "bundle.json");
    writeFileSync(keyFile, privatePem);
    writeFileSync(payloadFile, JSON.stringify(basePayload()));

    const result = run([
      "--payload",
      payloadFile,
      "--key",
      keyFile,
      "--key-id",
      "test-reviewer",
      "--out",
      outFile,
    ]);
    assert.equal(result.status, 0, result.stderr);
    assert.match(result.stdout, /signed bundle written/);

    // Trust store must contain the test key to verify the signature here.
    const trust = {
      schemaVersion: 1,
      reviewers: [{ id: "test-reviewer", publicKeyPem: publicPem }],
    };
    const trustFile = join(folder, "trust.json");
    writeFileSync(trustFile, JSON.stringify(trust));

    // One platform is structurally invalid on purpose: if the error mentions
    // attestations instead of the signature, the Ed25519 verification passed.
    let structuralError = null;
    try {
      verifyOfflineReleaseEvidence({
        evidenceFile: outFile,
        trustFile,
        sourceSha: "a".repeat(40),
        maestroSha: "b".repeat(40),
        featureIds: Array.from({ length: 48 }, (_, i) => "feature-" + i),
      });
    } catch (error) {
      structuralError = error;
    }
    assert.ok(
      structuralError,
      "bundle with one platform must fail structurally",
    );
    assert.match(
      structuralError.message,
      /Exactly three distinct native platform attestations/,
    );
  } finally {
    rmSync(folder, { recursive: true, force: true });
  }
});

test("signature from a different key is rejected by the trust store contract", () => {
  const folder = mkdtempSync(join(tmpdir(), "nexus-sign-"));
  try {
    const signer = makeKey();
    const enrolled = makeKey();
    const keyFile = join(folder, "signer.pem");
    const payloadFile = join(folder, "payload.json");
    const outFile = join(folder, "bundle.json");
    writeFileSync(keyFile, signer.privatePem);
    writeFileSync(payloadFile, JSON.stringify(basePayload()));

    const result = run([
      "--payload",
      payloadFile,
      "--key",
      keyFile,
      "--key-id",
      "other-reviewer",
      "--out",
      outFile,
    ]);
    assert.equal(result.status, 0);

    const trust = {
      schemaVersion: 1,
      reviewers: [{ id: "other-reviewer", publicKeyPem: enrolled.publicPem }],
    };
    const trustFile = join(folder, "trust.json");
    writeFileSync(trustFile, JSON.stringify(trust));

    assert.throws(
      () =>
        verifyOfflineReleaseEvidence({
          evidenceFile: outFile,
          trustFile,
          sourceSha: "a".repeat(40),
          maestroSha: "b".repeat(40),
          featureIds: Array.from({ length: 48 }, (_, i) => "feature-" + i),
        }),
      /signature failed/,
    );
  } finally {
    rmSync(folder, { recursive: true, force: true });
  }
});

test("refuses to sign payloads that already carry a signature", () => {
  const folder = mkdtempSync(join(tmpdir(), "nexus-sign-"));
  try {
    const keyFile = join(folder, "private.pem");
    const payloadFile = join(folder, "payload.json");
    writeFileSync(keyFile, makeKey().privatePem);
    writeFileSync(
      payloadFile,
      JSON.stringify({
        ...basePayload(),
        signature: { keyId: "x", base64: "y" },
      }),
    );
    const result = run([
      "--payload",
      payloadFile,
      "--key",
      keyFile,
      "--key-id",
      "r",
      "--out",
      join(folder, "out.json"),
    ]);
    assert.equal(result.status, 2);
    assert.match(result.stderr, /already carries a signature/);
  } finally {
    rmSync(folder, { recursive: true, force: true });
  }
});

test("--hash prints the sha256 of an artifact", () => {
  const folder = mkdtempSync(join(tmpdir(), "nexus-sign-"));
  try {
    const file = join(folder, "artifact.bin");
    writeFileSync(file, "nexus-evidence-artifact");
    const result = run(["--hash", file]);
    assert.equal(result.status, 0);
    const digest = result.stdout.split(/\s/)[0];
    assert.match(digest, /^[0-9a-f]{64}$/);
  } finally {
    rmSync(folder, { recursive: true, force: true });
  }
});
