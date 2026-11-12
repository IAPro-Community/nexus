#!/usr/bin/env node
// Signs an offline release evidence payload so that
// `node scripts/verify-agent-workspace-v1.mjs --release --offline --sha <sha>
//  --maestro-sha <sha> --evidence <bundle>` accepts it.
//
// Usage:
//   node scripts/sign-offline-release-evidence.mjs \
//     --payload <raw-payload.json> --key <ed25519-private.pem> \
//     --key-id <reviewer-id> --out <signed-bundle.json>
//   node scripts/sign-offline-release-evidence.mjs --hash <artifact-file>
//
// The payload must already contain: schemaVersion=2, sourceSha, maestroSha,
// issuedAtUtc (UTC), platforms (exactly 3 physical attestations) and features
// (exactly 48 passed entries). Structural validation happens when the release
// verifier consumes the signed bundle; this tool only canonicalizes and signs.
import { createHash, sign as cryptoSign } from "node:crypto";
import { readFileSync, writeFileSync } from "node:fs";
import { canonicalEvidencePayload } from "./offline-release-evidence.mjs";

function option(name) {
  const at = process.argv.indexOf(name);
  if (at < 0) return null;
  const value = process.argv[at + 1];
  if (!value || value.startsWith("--"))
    throw new Error("Missing value for " + name);
  return value;
}

const hashTarget = option("--hash");
if (hashTarget) {
  const digest = createHash("sha256")
    .update(readFileSync(hashTarget))
    .digest("hex");
  console.log(digest + "  " + hashTarget);
  process.exit(0);
}

const payloadFile = option("--payload");
const keyFile = option("--key");
const keyId = option("--key-id");
const outFile = option("--out");
for (const [name, value] of [
  ["--payload", payloadFile],
  ["--key", keyFile],
  ["--key-id", keyId],
  ["--out", outFile],
]) {
  if (!value) {
    console.error(
      "sign-offline-release-evidence: missing required option " + name,
    );
    process.exit(2);
  }
}

let payload;
try {
  payload = JSON.parse(readFileSync(payloadFile, "utf8"));
} catch (error) {
  console.error(
    "sign-offline-release-evidence: payload is not valid JSON: " +
      error.message,
  );
  process.exit(2);
}
if (payload.schemaVersion !== 2 || !payload.sourceSha || !payload.maestroSha) {
  console.error(
    "sign-offline-release-evidence: payload must be schemaVersion 2 with sourceSha and maestroSha",
  );
  process.exit(2);
}
if (payload.signature) {
  console.error(
    "sign-offline-release-evidence: payload already carries a signature; strip it first",
  );
  process.exit(2);
}

const canonical = Buffer.from(canonicalEvidencePayload(payload));
let signature;
try {
  signature = cryptoSign(null, canonical, readFileSync(keyFile));
} catch (error) {
  console.error(
    "sign-offline-release-evidence: signing failed: " + error.message,
  );
  process.exit(2);
}

const bundle = {
  ...payload,
  signature: { keyId, base64: signature.toString("base64") },
};
writeFileSync(outFile, JSON.stringify(bundle, null, 2) + "\n");
console.log(
  "signed bundle written to " +
    outFile +
    " (verify with: node scripts/verify-agent-workspace-v1.mjs --release --offline --sha " +
    payload.sourceSha +
    " --maestro-sha " +
    payload.maestroSha +
    " --evidence " +
    outFile +
    ")",
);
