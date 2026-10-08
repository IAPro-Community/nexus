import { spawn } from "node:child_process";
import { stageBetaVersionFiles } from "./release-version.mjs";

const versionPath = new URL("../VERSION", import.meta.url);
const packagePath = new URL("../web/package.json", import.meta.url);
const staged = await stageBetaVersionFiles(versionPath, packagePath);
console.log(`Gerando versão beta ${staged.next} (anterior: ${staged.previous})`);

let buildSucceeded = false;
try {
  const child = spawn("make", ["build"], {
    stdio: "inherit",
    cwd: new URL("..", import.meta.url),
  });
  const exitCode = await new Promise((resolve) => {
    child.on("close", (code) => resolve(code ?? 1));
    child.on("error", () => resolve(1));
  });
  if (exitCode !== 0) {
    throw new Error(`Build falhou (código ${exitCode})`);
  }
  buildSucceeded = true;
  console.log(`Versão beta ${staged.next} compilada; nenhum instalador ou tag foi publicado.`);
} finally {
  if (!buildSucceeded) {
    await staged.rollback();
    console.error(`Build não concluído; VERSION e web/package.json restaurados para ${staged.previous}`);
  }
}
