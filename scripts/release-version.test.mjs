import assert from "node:assert/strict";
import { mkdtemp, readFile, rm, writeFile } from "node:fs/promises";
import { tmpdir } from "node:os";
import path from "node:path";
import { test } from "node:test";
import { nextBetaVersion, stageBetaVersionFiles } from "./release-version.mjs";

test("beta increments only the prerelease counter", () => {
  assert.equal(nextBetaVersion("1.0.0-beta.9"), "1.0.0-beta.10");
  assert.equal(nextBetaVersion("1.0.0"), "1.1.0-beta.0");
  assert.throws(() => nextBetaVersion("1.0.0-rc.2"), /prerelease/);
  assert.throws(() => nextBetaVersion("broken"), /inválida/);
});

async function withFixture(version, packageVersion, callback) {
  const dir = await mkdtemp(path.join(tmpdir(), "nexus-release-test-"));
  const versionPath = path.join(dir, "VERSION");
  const packagePath = path.join(dir, "package.json");
  const originalVersion = `${version}\n`;
  const originalPackage = JSON.stringify(
    { name: "iapro-nexus-web", version: packageVersion, scripts: { test: "vitest run" } },
    null,
    2,
  ) + "\n";
  try {
    await writeFile(versionPath, originalVersion);
    await writeFile(packagePath, originalPackage);
    await callback({ versionPath, packagePath, originalVersion, originalPackage });
  } finally {
    await rm(dir, { recursive: true, force: true });
  }
}

test("staging synchronizes both files and rollback restores their original bytes", async () => {
  await withFixture("1.0.0-beta.9", "1.0.0-beta.9", async (fixture) => {
    const staged = await stageBetaVersionFiles(fixture.versionPath, fixture.packagePath);
    assert.equal(staged.previous, "1.0.0-beta.9");
    assert.equal(staged.next, "1.0.0-beta.10");
    assert.equal((await readFile(fixture.versionPath, "utf8")).trim(), staged.next);
    const pkg = JSON.parse(await readFile(fixture.packagePath, "utf8"));
    assert.equal(pkg.version, staged.next);
    assert.equal(pkg.scripts.test, "vitest run");
    await staged.rollback();
    assert.equal(await readFile(fixture.versionPath, "utf8"), fixture.originalVersion);
    assert.equal(await readFile(fixture.packagePath, "utf8"), fixture.originalPackage);
  });
});

test("mismatched version contract fails before changing files", async () => {
  await withFixture("1.0.0-beta.9", "1.0.0-beta.8", async (fixture) => {
    await assert.rejects(
      stageBetaVersionFiles(fixture.versionPath, fixture.packagePath),
      /Contrato de versão inválido/,
    );
    assert.equal(await readFile(fixture.versionPath, "utf8"), fixture.originalVersion);
    assert.equal(await readFile(fixture.packagePath, "utf8"), fixture.originalPackage);
  });
});
