import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { resolve } from "node:path";
import test from "node:test";
import {
  PLATFORM_IDS,
  PRODUCER,
  assembleProof,
  buildPlan,
  detectTwoClient,
  readJson,
  runPlan,
} from "./workspace-acceptance-e2e.mjs";

const root = resolve(import.meta.dirname, "..");
const contract = readJson(
  resolve(root, "docs/reference/agent-workspace-v1.json"),
);
const map = readJson(
  resolve(root, "docs/reference/agent-workspace-v1-evidence-map.json"),
);

const SHA = "a".repeat(40);
const JOB_URL = "https://github.com/IAPro-Community/nexus/actions/runs/123";

function planFor(platform = "linux-amd64") {
  return buildPlan({ contract, map, root, platform });
}

test("plan covers every contract capability and classifies test references", () => {
  const plan = planFor();
  assert.equal(plan.producer, PRODUCER);
  assert.equal(plan.features.length, contract.features.length);
  const ids = plan.features.map((feature) => feature.id);
  assert.equal(new Set(ids).size, contract.features.length);
  const kinds = new Set(plan.steps.map((step) => step.kind));
  assert.ok(kinds.has("go"));
  assert.ok(kinds.has("vitest"));
  const goStep = plan.steps.find((step) => step.id.startsWith("go:"));
  assert.match(goStep.args[0], /^test$/);
  assert.match(goStep.args[2], /^\.\//);
});

test("plan groups Go references by package and keeps feature attribution", () => {
  const plan = planFor();
  const packages = plan.steps
    .filter((step) => step.kind === "go")
    .map((step) => step.id);
  assert.equal(new Set(packages).size, packages.length);
  const feature = plan.features.find((item) => item.id === "terminal-studio");
  assert.ok(feature.stepIds.some((id) => id.startsWith("go:internal/control/host")));
});

test("run marks capability passed only when every planned step passes", () => {
  const plan = planFor();
  const failing = new Set(["./internal/control/host"]);
  const record = runPlan(plan, {
    runner: ({ args }) => ({
      status: failing.has(args.at(-1)) ? 1 : 0,
      stdout: "",
      stderr: "",
    }),
  });
  assert.equal(record.platform, "linux-amd64");
  const host = record.features.find((item) => item.id === "terminal-studio");
  assert.equal(host.status, "failed");
  for (const step of host.steps) {
    if (step.id === "go:internal/control/host") {
      assert.equal(step.status, "failed");
      assert.equal(step.exitCode, 1);
    } else {
      assert.equal(step.status, "passed");
    }
  }
  const affected = new Set(
    plan.features
      .filter((feature) => feature.stepIds.includes("go:internal/control/host"))
      .map((feature) => feature.id),
  );
  assert.ok(affected.has("terminal-studio"));
  assert.ok(affected.has("remote-companion"));
  for (const feature of record.features) {
    if (feature.id === "cross-platform-exact-sha") {
      assert.equal(feature.status, "blocked");
      continue;
    }
    assert.equal(
      feature.status,
      affected.has(feature.id) ? "failed" : "passed",
      `${feature.id} status`,
    );
  }
});

test("run reports blocked when capability has no executable reference", () => {
  const plan = planFor();
  const withoutSteps = plan.features.find(
    (feature) => feature.id === "cross-platform-exact-sha",
  );
  assert.ok(withoutSteps);
  assert.equal(withoutSteps.stepIds.length, 0);
  const record = runPlan(plan, { runner: () => ({ status: 0, stdout: "", stderr: "" }) });
  const blocked = record.features.find(
    (feature) => feature.id === "cross-platform-exact-sha",
  );
  assert.equal(blocked.status, "blocked");
  assert.equal(record.status, "failed");
});

test("two-client evidence requires a real two-client test declaration", () => {
  assert.equal(
    detectTwoClient(root, "internal/control/web/multi_runtime_attention_e2e_test.go"),
    true,
  );
  assert.equal(
    detectTwoClient(root, "internal/control/web/structured_multiclient_process_e2e_test.go"),
    true,
  );
  assert.equal(
    detectTwoClient(root, "internal/control/host/host_test.go"),
    true,
  );
  assert.equal(
    detectTwoClient(root, "internal/nexus/agent_application_test.go"),
    false,
  );
});

test("assemble requires three native platforms with matching SHA", () => {
  const records = PLATFORM_IDS.map((platform) => ({
    schemaVersion: 1,
    producer: PRODUCER,
    platform,
    sourceSha: SHA,
    jobUrl: JOB_URL,
    status: "passed",
    features: [
      { id: "terminal-studio", category: "experience", status: "passed", tests: ["internal/control/host/host_test.go"], clients: 2 },
    ],
  }));
  const proof = assembleProof({ records, sourceSha: SHA, ciRunId: "123" });
  assert.equal(proof.producer, PRODUCER);
  assert.equal(proof.features.length, 1);
  assert.equal(proof.features[0].status, "passed");
  assert.equal(proof.features[0].clients, 2);
  assert.equal(proof.platforms["macos-arm64"].status, "passed");
  assert.throws(
    () => assembleProof({ records: records.slice(0, 2), sourceSha: SHA, ciRunId: "123" }),
    /three platform records/,
  );
  assert.throws(
    () => assembleProof({ records: records.slice(1), sourceSha: SHA, ciRunId: "123" }),
    /three platform records/,
  );
  const mismatched = records.map((record) => ({ ...record, sourceSha: "b".repeat(40) }));
  assert.throws(
    () => assembleProof({ records: mismatched, sourceSha: SHA, ciRunId: "123" }),
    /sourceSha mismatch/,
  );
  const badUrl = records.map((record) => ({
    ...record,
    jobUrl: "https://example.com/not-github",
  }));
  assert.throws(
    () => assembleProof({ records: badUrl, sourceSha: SHA, ciRunId: "123" }),
    /jobUrl/,
  );
});

test("native certification closes only the capability it names", () => {
  const records = PLATFORM_IDS.map((platform) => ({
    schemaVersion: 1,
    producer: PRODUCER,
    platform,
    sourceSha: SHA,
    jobUrl: JOB_URL,
    status: "failed",
    features: [
      {
        id: "cross-platform-exact-sha",
        category: "foundation",
        status: "blocked",
        tests: [],
        clients: 1,
      },
    ],
  }));
  const withoutCertification = assembleProof({
    records,
    sourceSha: SHA,
    ciRunId: "123",
  });
  assert.equal(withoutCertification.features[0].status, "failed");
  assert.deepEqual(withoutCertification.features[0].tests, []);

  const withCertification = assembleProof({
    records,
    sourceSha: SHA,
    ciRunId: "123",
    nativeCertifications: [
      {
        id: "cross-platform-exact-sha",
        sourceSha: SHA,
        jobUrl: JOB_URL,
        tests: ["native-certification-" + SHA],
        clients: 1,
      },
    ],
  });
  assert.equal(withCertification.features[0].status, "passed");
  assert.deepEqual(withCertification.features[0].tests, [
    "native-certification-" + SHA,
  ]);

  const forged = assembleProof({
    records,
    sourceSha: SHA,
    ciRunId: "123",
    nativeCertifications: [
      {
        id: "cross-platform-exact-sha",
        sourceSha: SHA,
        jobUrl: "https://example.com/not-a-run",
        tests: ["native-certification-" + SHA],
        clients: 1,
      },
    ],
  });
  assert.equal(forged.features[0].status, "failed");
});

test("native certification cannot approve another capability or another SHA", () => {
  const records = PLATFORM_IDS.map((platform) => ({
    schemaVersion: 1,
    producer: PRODUCER,
    platform,
    sourceSha: SHA,
    jobUrl: JOB_URL,
    status: "failed",
    features: [
      {
        id: "cross-platform-exact-sha",
        category: "foundation",
        status: "blocked",
        tests: [],
        clients: 1,
      },
    ],
  }));
  const wrongCapability = assembleProof({
    records,
    sourceSha: SHA,
    ciRunId: "123",
    nativeCertifications: [
      {
        id: "other-capability",
        sourceSha: SHA,
        jobUrl: JOB_URL,
        tests: ["native-certification-" + SHA],
        clients: 1,
      },
    ],
  });
  assert.equal(wrongCapability.features[0].status, "failed");
  const wrongSha = assembleProof({
    records,
    sourceSha: SHA,
    ciRunId: "123",
    nativeCertifications: [
      {
        id: "cross-platform-exact-sha",
        sourceSha: "b".repeat(40),
        jobUrl: JOB_URL,
        tests: ["native-certification-" + SHA],
        clients: 1,
      },
    ],
  });
  assert.equal(wrongSha.features[0].status, "failed");
});

test("assemble fails capability when one native platform lacks evidence", () => {
  const records = PLATFORM_IDS.map((platform, index) => ({
    schemaVersion: 1,
    producer: PRODUCER,
    platform,
    sourceSha: SHA,
    jobUrl: JOB_URL,
    status: index === 2 ? "failed" : "passed",
    features: [
      {
        id: "terminal-studio",
        category: "experience",
        status: index === 2 ? "failed" : "passed",
        tests: index === 2 ? [] : ["internal/control/host/host_test.go"],
        clients: 1,
      },
    ],
  }));
  const proof = assembleProof({ records, sourceSha: SHA, ciRunId: "123" });
  assert.equal(proof.features[0].status, "failed");
  assert.equal(proof.platforms["macos-arm64"].status, "failed");
});
