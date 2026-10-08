import { readFile, writeFile } from "node:fs/promises";
import { spawn } from "node:child_process";

const versionPath = new URL("../VERSION", import.meta.url);
const packagePath = new URL("../web/package.json", import.meta.url);
const originalVersion = await readFile(versionPath, "utf8");
const current = originalVersion.trim();
const originalPackage = await readFile(packagePath, "utf8");
const pkg = JSON.parse(originalPackage);

const match = /^(\d+)\.(\d+)\.(\d+)(?:-([0-9A-Za-z-]+)\.(\d+))?$/.exec(current);
if (!match) {
  throw new Error(`VERSION inválida: ${current}`);
}
if (pkg.version !== current) {
  throw new Error(
    `Contrato de versão inconsistente antes do bump: VERSION=${current}, web/package.json=${pkg.version}`,
  );
}

const [, major, minor, patch, pre, preNumber] = match;
const next =
  pre === "beta"
    ? `${major}.${minor}.${patch}-beta.${Number(preNumber) + 1}`
    : `${major}.${Number(minor) + 1}.0-beta.0`;

try {
  await writeFile(versionPath, `${next}\n`);
  pkg.version = next;
  await writeFile(packagePath, JSON.stringify(pkg, null, 2) + "\n");
  console.log(`Gerando versão beta ${next} (anterior: ${current})`);

  const child = spawn("make", ["build"], {
    stdio: "inherit",
    cwd: new URL("..", import.meta.url),
  });
  const exitCode = await new Promise((resolve) => {
    child.on("close", (code) => resolve(code ?? 1));
    child.on("error", () => resolve(1));
  });
  if (exitCode !== 0) {
    throw new Error(`build exited with code ${exitCode}`);
  }
} catch (error) {
  // A failed bump must not leave the version contract half-updated.
  await Promise.all([
    writeFile(versionPath, originalVersion),
    writeFile(packagePath, originalPackage),
  ]);
  console.error(`Build/bump falhou; VERSION e web/package.json restaurados para ${current}`);
  console.error(error);
  process.exitCode = 1;
}

if (process.exitCode !== 1) {
  console.log(`Versão beta ${next} compilada e instalada.`);
}
