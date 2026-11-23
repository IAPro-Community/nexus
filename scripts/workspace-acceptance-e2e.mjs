#!/usr/bin/env node
import { spawnSync } from "node:child_process";
import { existsSync, mkdirSync, readFileSync, writeFileSync } from "node:fs";
import { dirname, relative, resolve } from "node:path";
import process from "node:process";

export const PRODUCER = "nexus-v1-agent-workspace-e2e";
export const PLATFORM_IDS = ["linux-amd64", "windows-amd64", "macos-arm64"];
const SHA = /^[0-9a-f]{40}$/;
const ACTION_URL = /^https:\/\/github\.com\/[A-Za-z0-9_.-]+\/[A-Za-z0-9_.-]+\/actions\/runs\/\d+$/;
const GO_TEST = /func Test\w*(TwoClient|TwoClients|TwoAuthenticated|TwoSessions|TwoAgents|SecondClient)/;
const JS_TEST = /\b(test|it)\s*\(\s*[`'"][^`'"]*two[ _-]?client/i;

export function readJson(file) {
  return JSON.parse(readFileSync(file, "utf8"));
}

function classify(ref) {
  if (ref.startsWith("web/scripts/e2e-")) return "e2e";
  if (/\.test\.[jt]sx?$/.test(ref)) return "vitest";
  if (/_test\.go$/.test(ref)) return "go";
  return null;
}

export function detectTwoClient(root, ref) {
  const absolute = resolve(root, ref);
  if (!existsSync(absolute)) return false;
  const source = readFileSync(absolute, "utf8");
  return ref.endsWith("_test.go") ? GO_TEST.test(source) : JS_TEST.test(source);
}

export function buildPlan({
  contract,
  map,
  root,
  platform,
  includeBrowserE2E = false,
}) {
  if (!PLATFORM_IDS.includes(platform)) {
    throw new Error(`unsupported platform ${platform}`);
  }
  const categoryById = new Map(
    contract.features.map((feature) => [feature.id, feature.category]),
  );
  const steps = new Map();
  const features = [];

  const addStep = (step) => {
    if (!steps.has(step.id)) steps.set(step.id, step);
    return step.id;
  };

  for (const entry of map.features) {
    const refs = new Set();
    for (const criterion of entry.criteria || []) {
      for (const ref of criterion.testRefs || []) refs.add(ref);
    }
    const stepIds = [];
    const goPackages = new Map();
    for (const ref of [...refs].sort()) {
      const kind = classify(ref);
      if (!kind) continue;
      if (kind === "e2e" && !includeBrowserE2E) continue;
      if (kind === "go") {
        const pkg = dirname(ref);
        if (!goPackages.has(pkg)) goPackages.set(pkg, []);
        goPackages.get(pkg).push(ref);
        continue;
      }
      const id = `${kind}:${ref}`;
      addStep({
        id,
        kind,
        cwd: kind === "vitest" ? resolve(root, "web") : root,
        command: kind === "vitest" ? "vitest" : "node",
        args:
          kind === "vitest"
            ? ["run", relative(resolve(root, "web"), resolve(root, ref))]
            : [ref],
        tests: [ref],
        twoClient: detectTwoClient(root, ref),
      });
      stepIds.push(id);
    }
    for (const [pkg, tests] of [...goPackages.entries()].sort()) {
      const id = `go:${pkg}`;
      addStep({
        id,
        kind: "go",
        cwd: root,
        command: "go",
        args: ["test", "-count=1", `./${pkg}`],
        tests: tests.sort(),
        twoClient: tests.some((ref) => detectTwoClient(root, ref)),
      });
      stepIds.push(id);
    }
    features.push({
      id: entry.id,
      category: categoryById.get(entry.id) || "unknown",
      stepIds,
      plannedTests: [...refs].sort(),
    });
  }

  features.sort((a, b) => a.id.localeCompare(b.id));
  return {
    schemaVersion: 1,
    producer: PRODUCER,
    platform,
    steps: [...steps.values()].sort((a, b) => a.id.localeCompare(b.id)),
    features,
  };
}

const defaultRunner = ({ command, args, cwd }) => {
  if (command === "vitest") {
    const bunx = spawnSync("bunx", ["vitest", ...args], {
      cwd,
      encoding: "utf8",
      maxBuffer: 64 * 1024 * 1024,
    });
    if (bunx.error?.code !== "ENOENT") {
      return {
        status: bunx.status ?? 1,
        stdout: bunx.stdout || "",
        stderr: bunx.stderr || "",
      };
    }
    const npx = spawnSync("npx", ["vitest", ...args], {
      cwd,
      encoding: "utf8",
      maxBuffer: 64 * 1024 * 1024,
    });
    return {
      status: npx.status ?? 1,
      stdout: npx.stdout || "",
      stderr: npx.stderr || "",
    };
  }
  const result = spawnSync(command, args, {
    cwd,
    encoding: "utf8",
    maxBuffer: 64 * 1024 * 1024,
  });
  return {
    status: result.status ?? 1,
    stdout: result.stdout || "",
    stderr: result.stderr || "",
  };
};

export function runPlan(plan, { runner = defaultRunner, logDir } = {}) {
  const stepResults = new Map();
  for (const step of plan.steps) {
    const started = Date.now();
    const outcome = runner({ command: step.command, args: step.args, cwd: step.cwd });
    const status = outcome.status === 0 ? "passed" : "failed";
    const result = {
      id: step.id,
      kind: step.kind,
      status,
      exitCode: outcome.status,
      durationMs: Date.now() - started,
      tests: step.tests,
    };
    stepResults.set(step.id, result);
    if (logDir) {
      mkdirSync(logDir, { recursive: true });
      const safe = step.id.replace(/[^A-Za-z0-9._-]+/g, "_");
      writeFileSync(
        resolve(logDir, `${safe}.log`),
        `${step.command} ${step.args.join(" ")}\n\n${outcome.stdout || ""}${outcome.stderr || ""}`,
      );
    }
  }

  const features = plan.features.map((feature) => {
    const results = feature.stepIds.map((id) => stepResults.get(id)).filter(Boolean);
    const failed = results.filter((result) => result.status !== "passed");
    const passedTests = results
      .filter((result) => result.status === "passed")
      .flatMap((result) => result.tests);
    const twoClient = results.some(
      (result) => result.status === "passed" && result.twoClient,
    );
    const status =
      results.length === 0 ? "blocked" : failed.length === 0 ? "passed" : "failed";
    return {
      id: feature.id,
      category: feature.category,
      status,
      tests: [...new Set(passedTests)].sort(),
      clients: twoClient ? 2 : 1,
      steps: results.map((result) => ({
        id: result.id,
        status: result.status,
        exitCode: result.exitCode,
        durationMs: result.durationMs,
      })),
    };
  });

  return {
    schemaVersion: 1,
    producer: PRODUCER,
    platform: plan.platform,
    status: features.some((feature) => feature.status !== "passed")
      ? "failed"
      : "passed",
    features,
  };
}

export function assembleProof({
  records,
  sourceSha,
  ciRunId,
  nativeCertifications = [],
}) {
  if (!SHA.test(sourceSha || "")) {
    throw new Error("sourceSha must be 40 lowercase hex characters");
  }
  if (!/^[1-9][0-9]*$/.test(ciRunId || "")) {
    throw new Error("ciRunId must be a positive integer");
  }
  if (!Array.isArray(records) || records.length !== PLATFORM_IDS.length) {
    throw new Error("exactly three platform records are required");
  }
  const byPlatform = new Map();
  for (const record of records) {
    if (!PLATFORM_IDS.includes(record.platform)) {
      throw new Error(`unexpected platform ${record.platform}`);
    }
    if (byPlatform.has(record.platform)) {
      throw new Error(`duplicated platform ${record.platform}`);
    }
    if (record.sourceSha !== sourceSha) {
      throw new Error(`platform ${record.platform} sourceSha mismatch`);
    }
    if (!ACTION_URL.test(record.jobUrl || "")) {
      throw new Error(`platform ${record.platform} jobUrl is not a GitHub Actions run URL`);
    }
    byPlatform.set(record.platform, record);
  }
  for (const platform of PLATFORM_IDS) {
    if (!byPlatform.has(platform)) throw new Error(`missing platform ${platform}`);
  }

  const ids = [...new Set(records.flatMap((record) => record.features.map((f) => f.id)))].sort();
  const features = ids.map((id) => {
    const perPlatform = PLATFORM_IDS.map((platform) => {
      const record = byPlatform.get(platform);
      return record.features.find((feature) => feature.id === id);
    });
    const certification = nativeCertifications.find((entry) => entry.id === id);
    const coveredByPlatform = perPlatform.every(
      (feature) => feature?.status === "passed",
    );
    const coveredByCertification =
      certification &&
      certification.sourceSha === sourceSha &&
      certification.jobUrl &&
      ACTION_URL.test(certification.jobUrl) &&
      Array.isArray(certification.tests) &&
      certification.tests.length > 0;
    const status =
      coveredByPlatform || coveredByCertification ? "passed" : "failed";
    const tests = [
      ...new Set([
        ...perPlatform.flatMap((feature) => feature?.tests || []),
        ...(coveredByCertification ? certification.tests : []),
      ]),
    ].sort();
    const clients = Math.min(
      ...perPlatform.map((feature) => feature?.clients || 1),
      coveredByCertification ? certification.clients || 1 : Infinity,
    );
    const jobUrl =
      perPlatform.find((feature) => feature?.status === "passed")?.jobUrl ||
      (coveredByCertification ? certification.jobUrl : undefined) ||
      byPlatform.get(PLATFORM_IDS[0]).jobUrl;
    return { id, status, sourceSha, jobUrl, tests, clients };
  });

  return {
    schemaVersion: 1,
    sourceSha,
    ciRunId,
    producer: PRODUCER,
    platforms: Object.fromEntries(
      PLATFORM_IDS.map((platform) => {
        const record = byPlatform.get(platform);
        return [
          platform,
          {
            status: record.status === "passed" ? "passed" : "failed",
            sourceSha,
            jobUrl: record.jobUrl,
          },
        ];
      }),
    ),
    features,
  };
}

function option(name, fallback = null) {
  const index = process.argv.indexOf(name);
  if (index < 0) return fallback;
  const value = process.argv[index + 1];
  if (!value || value.startsWith("--")) {
    throw new Error(`missing value for ${name}`);
  }
  return value;
}

function optionAll(name) {
  const values = [];
  for (let index = 0; index < process.argv.length; index++) {
    if (process.argv[index] !== name) continue;
    const value = process.argv[index + 1];
    if (!value || value.startsWith("--")) {
      throw new Error(`missing value for ${name}`);
    }
    values.push(value);
  }
  return values;
}

function loadRoot() {
  return resolve(import.meta.dirname, "..");
}

function main() {
  const mode = process.argv[2];
  const root = loadRoot();
  const contract = readJson(
    resolve(root, "docs/reference/agent-workspace-v1.json"),
  );
  const map = readJson(
    resolve(root, "docs/reference/agent-workspace-v1-evidence-map.json"),
  );

  if (mode === "plan") {
    const platform = option("--platform", "linux-amd64");
    const plan = buildPlan({
      contract,
      map,
      root,
      platform,
      includeBrowserE2E: process.argv.includes("--include-browser-e2e"),
    });
    process.stdout.write(`${JSON.stringify(plan, null, 2)}\n`);
    return;
  }

  if (mode === "run") {
    const platform = option("--platform");
    const sourceSha = option("--sha");
    const jobUrl = option("--job-url");
    if (!SHA.test(sourceSha || "")) {
      throw new Error("--sha must be 40 lowercase hex characters");
    }
    if (!ACTION_URL.test(jobUrl || "")) {
      throw new Error("--job-url must be a GitHub Actions run URL");
    }
    const plan = buildPlan({
      contract,
      map,
      root,
      platform,
      includeBrowserE2E: process.argv.includes("--include-browser-e2e"),
    });
    const record = runPlan(plan, { logDir: option("--log-dir") });
    const payload = {
      ...record,
      sourceSha,
      jobUrl,
      generatedAtUtc: new Date().toISOString(),
    };
    const out = option("--out");
    if (out) {
      mkdirSync(dirname(resolve(out)), { recursive: true });
      writeFileSync(resolve(out), `${JSON.stringify(payload, null, 2)}\n`);
    }
    process.stdout.write(`${JSON.stringify(payload, null, 2)}\n`);
    if (payload.status !== "passed" && !process.argv.includes("--allow-incomplete")) {
      process.exitCode = 1;
    }
    return;
  }

  if (mode === "assemble") {
    const sourceSha = option("--sha");
    const ciRunId = option("--ci-run-id");
    const records = PLATFORM_IDS.map((platform) => {
      const file = option(`--${platform}`);
      if (!file) throw new Error(`missing --${platform}`);
      return readJson(resolve(file));
    });
    const nativeIds = optionAll("--native-certified");
    const certificationJobUrl = option("--native-certification-job-url");
    const certificationTests = optionAll("--native-certification-test");
    const nativeCertifications = nativeIds.map((id) => ({
      id,
      sourceSha,
      jobUrl: certificationJobUrl || "",
      tests:
        certificationTests.length > 0
          ? certificationTests
          : [`native-certification-${sourceSha}`],
      clients: 1,
    }));
    const proof = assembleProof({
      records,
      sourceSha,
      ciRunId,
      nativeCertifications,
    });
    const out = option("--out");
    if (out) {
      mkdirSync(dirname(resolve(out)), { recursive: true });
      writeFileSync(resolve(out), `${JSON.stringify(proof, null, 2)}\n`);
    }
    process.stdout.write(`${JSON.stringify(proof, null, 2)}\n`);
    return;
  }

  throw new Error(
    "usage: workspace-acceptance-e2e.mjs plan|run|assemble [options]",
  );
}

if (process.argv[1] && resolve(process.argv[1]) === resolve(import.meta.filename)) {
  try {
    main();
  } catch (error) {
    console.error(`workspace-acceptance-e2e: ${error?.message || error}`);
    process.exit(1);
  }
}
