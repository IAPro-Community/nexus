import { readFile, writeFile } from "node:fs/promises";

/** Preserve the existing release policy: beta increments stay on the same minor. */
export function nextBetaVersion(current) {
  const beta = /^(\d+)\.(\d+)\.(\d+)-beta\.(\d+)$/.exec(current);
  if (beta) {
    const [, major, minor, patch, number] = beta;
    return `${major}.${minor}.${patch}-beta.${Number(number) + 1}`;
  }
  const stable = /^(\d+)\.(\d+)\.(\d+)$/.exec(current);
  if (stable) {
    const [, major, minor] = stable;
    return `${major}.${Number(minor) + 1}.0-beta.0`;
  }
  throw new Error(`VERSION inválida ou prerelease não suportada: ${current}`);
}

/**
 * Keep VERSION and web/package.json in sync. The rollback restores original
 * bytes (not reserialized JSON), including on a failed build.
 */
export async function stageBetaVersionFiles(versionPath, packagePath) {
  const [previousVersionText, previousPackageText] = await Promise.all([
    readFile(versionPath, "utf8"),
    readFile(packagePath, "utf8"),
  ]);
  const previous = previousVersionText.trim();
  const pkg = JSON.parse(previousPackageText);
  if (pkg.version !== previous) {
    throw new Error(
      `Contrato de versão inválido: VERSION=${previous}, web/package.json=${pkg.version}`,
    );
  }
  const next = nextBetaVersion(previous);
  // Replace only the version field in-place to preserve original formatting
  // (tabs, spacing, key order). JSON.stringify would reformat the whole file.
  const versionFieldRegex = /("version"\s*:\s*")[^"]*(")/;
  if (!versionFieldRegex.test(previousPackageText)) {
    throw new Error(
      `web/package.json não contém campo "version" válido`,
    );
  }
  const newPackageText = previousPackageText.replace(
    versionFieldRegex,
    `$1${next}$2`,
  );
  const rollback = async () => {
    const results = await Promise.allSettled([
      writeFile(versionPath, previousVersionText),
      writeFile(packagePath, previousPackageText),
    ]);
    const failures = results.filter((result) => result.status === "rejected");
    if (failures.length) {
      throw new AggregateError(
        failures.map((result) => result.reason),
        "Não foi possível restaurar ambos os arquivos de versão",
      );
    }
  };
  try {
    await writeFile(versionPath, `${next}\n`);
    await writeFile(packagePath, newPackageText);
  } catch (error) {
    await rollback();
    throw error;
  }
  return { previous, next, rollback };
}
