#!/usr/bin/env node
/**
 * Frontend verification report for IAPro Nexus.
 *
 * Runs typecheck → lint → test → build and writes a durable markdown report
 * under DEV/validation/. Fail the process if any hard gate fails.
 *
 * Usage:
 *   node web/scripts/verify-report.mjs
 *   npm --prefix web run verify
 *   make web-verify
 */
import { spawnSync } from 'node:child_process';
import { existsSync, mkdirSync, readFileSync, readdirSync, writeFileSync } from 'node:fs';
import { dirname, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';

const webDir = resolve(dirname(fileURLToPath(import.meta.url)), '..');
const repoDir = resolve(webDir, '..');
const outDir = resolve(repoDir, 'DEV/validation');
const latestPath = resolve(outDir, 'FRONTEND_LATEST.md');
const historyPath = resolve(outDir, 'FRONTEND_HISTORY.md');

const now = new Date();
const stamp = now.toISOString().replace(/\.\d{3}Z$/, 'Z');
const shortStamp = stamp.replace(/[:.]/g, '-');

/** @typedef {{ id: string, title: string, hard: boolean, command?: string[], cwd?: string, check?: () => GateResult }} Gate */
/** @typedef {{ ok: boolean, detail: string, durationMs: number }} GateResult */

function bin(name) {
  const candidates = [
    resolve(webDir, 'node_modules/.bin', name),
    resolve(webDir, 'node_modules/.bin', `${name}.js`),
    resolve(webDir, 'node_modules/.bin', `${name}.cjs`),
  ];
  for (const candidate of candidates) {
    if (existsSync(candidate)) return candidate;
  }
  return name;
}

function runNodeScript(scriptRelative, args = []) {
  return run(process.execPath, [resolve(webDir, scriptRelative), ...args], webDir);
}

function runLocal(name, args) {
  const executable = bin(name);
  // Prefer invoking .bin shims through node when they are JS entrypoints.
  if (executable.endsWith('.js') || executable.endsWith('.cjs') || executable.endsWith('.mjs')) {
    return run(process.execPath, [executable, ...args], webDir);
  }
  // Many npm .bin shims are shell scripts; run via PATH lookup with explicit bin dir.
  return run(executable, args, webDir, {
    PATH: `${resolve(webDir, 'node_modules/.bin')}:${process.env.PATH || ''}`,
  });
}

function run(command, args, cwd, extraEnv = {}) {
  const started = Date.now();
  const result = spawnSync(command, args, {
    cwd,
    encoding: 'utf8',
    env: { ...process.env, FORCE_COLOR: '0', NO_COLOR: '1', ...extraEnv },
    maxBuffer: 8 * 1024 * 1024,
  });
  const durationMs = Date.now() - started;
  const stdout = (result.stdout || '').trim();
  const stderr = (result.stderr || '').trim();
  const combined = [stdout, stderr].filter(Boolean).join('\n');
  const ok = result.status === 0 && !result.error;
  const tail = combined
    .split('\n')
    .slice(-40)
    .join('\n')
    .trim();
  const errorNote = result.error ? `spawn error: ${result.error.message}` : '';
  return {
    ok,
    detail: ok
      ? tail || 'ok'
      : [errorNote, `exit ${result.status ?? 'null'}`, tail || '(sem saída)'].filter(Boolean).join('\n'),
    durationMs,
  };
}

function typecheck() {
  return runLocal('tsc', ['--noEmit']);
}

function lint() {
  return runLocal('eslint', ['src']);
}

function testAll() {
  return runLocal('vitest', ['run']);
}

function stylelintCheck() {
  return runLocal('stylelint', ['src/**/*.css']);
}

function prettierCheck() {
  return runLocal('prettier', ['--check', 'src/**/*.{ts,tsx,css,json}']);
}

function build() {
  return runNodeScript('scripts/build.mjs');
}

function checkEmbeddedSync() {
  const started = Date.now();
  const webBundle = resolve(webDir, 'dist/bundle.js');
  const embeddedBundle = resolve(repoDir, 'internal/control/web/embedded/bundle.js');
  if (!existsSync(webBundle) || !existsSync(embeddedBundle)) {
    return {
      ok: false,
      detail: 'bundle.js ausente em web/dist ou internal/control/web/embedded — rode o build',
      durationMs: Date.now() - started,
    };
  }
  const a = readFileSync(webBundle);
  const b = readFileSync(embeddedBundle);
  if (a.length !== b.length || !a.equals(b)) {
    return {
      ok: false,
      detail: 'web/dist e internal/control/web/embedded divergem — o binário nexus pode servir UI antiga',
      durationMs: Date.now() - started,
    };
  }
  return {
    ok: true,
    detail: `bundles idênticos (${a.length} bytes)`,
    durationMs: Date.now() - started,
  };
}

function checkCriticalUiMarkers() {
  const started = Date.now();
  const cssPath = resolve(webDir, 'dist/bundle.css');
  const jsPath = resolve(webDir, 'dist/bundle.js');
  if (!existsSync(cssPath) || !existsSync(jsPath)) {
    return {
      ok: false,
      detail: 'dist incompleto após build',
      durationMs: Date.now() - started,
    };
  }
  const css = readFileSync(cssPath, 'utf8');
  const js = readFileSync(jsPath, 'utf8');
  /** Markers that catch common “UI quebrada” regressions after embed/build. */
  const required = [
    { where: 'css', hay: css, needle: 'nx-os-shell', label: 'shell layout' },
    { where: 'css', hay: css, needle: 'nx-attention-radar', label: 'attention radar styles' },
    { where: 'css', hay: css, needle: 'xterm-helper-textarea', label: 'xterm input isolation' },
    { where: 'css', hay: css, needle: '--nx-spacing-8', label: 'canonical spacing tokens' },
    { where: 'js', hay: js, needle: 'nx-attention-radar', label: 'radar class wiring' },
    { where: 'js', hay: js, needle: 'esperando input', label: 'document title wait copy' },
    { where: 'js', hay: js, needle: 'Project Shell', label: 'overview/project shell CTA' },
  ];
  const missing = required.filter((item) => !item.hay.includes(item.needle));
  if (missing.length) {
    return {
      ok: false,
      detail: `marcadores ausentes: ${missing.map((m) => `${m.where}:${m.label}`).join(', ')}`,
      durationMs: Date.now() - started,
    };
  }
  return {
    ok: true,
    detail: `marcadores críticos presentes (${required.length})`,
    durationMs: Date.now() - started,
  };
}

function checkI18nParityQuick() {
  // Soft gate: vitest already covers full parity; this is a fast fail signal in the report.
  return runLocal('vitest', ['run', 'src/i18n/i18n.test.ts']);
}

/**
 * Static guard against the recurring Go-JSON null array crash:
 * `Cannot read properties of null (reading 'length'|'map'|...)`
 *
 * Flags bare field access like `pkg.dependencies.length` without `?.` or
 * prior normalization via `(x || [])` / `asArray(x)`.
 */
function checkNullSafeArrayAccess() {
  const started = Date.now();
  const srcRoot = resolve(webDir, 'src');
  // Nested Go JSON arrays that often arrive as `null`. Top-level lists
  // (agents/runtimes/projects/phases/packages) must be normalized at the API/
  // model edge instead — see useNexusData + normalizeWorkPlan.
  const fields =
    'dependencies|changed_files|commands|generations|acceptance_criteria|remaining_issues|dependency_receipts|skills|package_runs|maestro_gates|maestro_skills|relevant_paths|verification_requirements|shared_artifacts|changed_fields|warnings|added_packages|removed_packages|changed_packages|checks';
  const methods = 'length|map|filter|forEach|flatMap|reduce|find|some|every|includes|join|slice';
  // Negative lookbehind for optional chaining: foo?.dependencies.length is OK.
  const risky = new RegExp(`(?<!\\?)\\.(?:${fields})\\.(?:${methods})\\b`, 'g');
  const hits = [];

  function walk(dir) {
    for (const entry of readdirSync(dir, { withFileTypes: true })) {
      if (entry.name === 'node_modules' || entry.name === 'dist') continue;
      const full = resolve(dir, entry.name);
      if (entry.isDirectory()) {
        walk(full);
        continue;
      }
      if (!/\.(ts|tsx)$/.test(entry.name)) continue;
      if (entry.name.endsWith('.test.ts') || entry.name.endsWith('.test.tsx')) continue;
      const text = readFileSync(full, 'utf8');
      const lines = text.split('\n');
      for (let i = 0; i < lines.length; i++) {
        const line = lines[i];
        if (line.trimStart().startsWith('//') || line.trimStart().startsWith('*')) continue;
        // Allow normalized forms on the same expression line.
        if (/\|\|\s*\[\]/.test(line) || /asArray\s*\(/.test(line) || /asStringArray\s*\(/.test(line) || /Array\.isArray\s*\(/.test(line)) {
          continue;
        }
        // Truthy short-circuit on the same line: `x.deps && x.deps.length`
        const hasTruthyGuard = new RegExp(`(?:${fields})\\s*&&`).test(line);
        risky.lastIndex = 0;
        if (risky.test(line)) {
          if (hasTruthyGuard) continue;
          hits.push(`${full.replace(`${webDir}/`, '')}:${i + 1}: ${line.trim().slice(0, 160)}`);
        }
      }
    }
  }

  walk(srcRoot);

  if (hits.length) {
    return {
      ok: false,
      detail: `acesso inseguro a arrays da API (use asArray / || [] / ?.):\n${hits.slice(0, 20).join('\n')}`,
      durationMs: Date.now() - started,
    };
  }
  return {
    ok: true,
    detail: 'sem .length/.map direto em campos nullable conhecidos',
    durationMs: Date.now() - started,
  };
}

const gates = [
  { id: 'format', title: 'Prettier (`prettier --check`)', hard: true, check: () => prettierCheck() },
  { id: 'typecheck', title: 'TypeScript (`tsc --noEmit`)', hard: true, check: () => typecheck() },
  { id: 'lint', title: 'ESLint (`eslint src`)', hard: true, check: () => lint() },
  { id: 'stylelint', title: 'Stylelint (`stylelint "src/**/*.css"`)', hard: true, check: () => stylelintCheck() },
  { id: 'null-arrays', title: 'Null-safe API array access', hard: true, check: () => checkNullSafeArrayAccess() },
  { id: 'test', title: 'Vitest (`vitest run`)', hard: true, check: () => testAll() },
  { id: 'i18n', title: 'i18n catalog parity', hard: true, check: () => checkI18nParityQuick() },
  { id: 'build', title: 'Build + embed (`node scripts/build.mjs`)', hard: true, check: () => build() },
  { id: 'embed-sync', title: 'Embed sync (web/dist ≡ internal/.../embedded)', hard: true, check: () => checkEmbeddedSync() },
  { id: 'ui-markers', title: 'Critical UI markers in bundle', hard: true, check: () => checkCriticalUiMarkers() },
];

function gitMeta() {
  const branch = spawnSync('git', ['rev-parse', '--abbrev-ref', 'HEAD'], { cwd: repoDir, encoding: 'utf8' });
  const commit = spawnSync('git', ['rev-parse', '--short', 'HEAD'], { cwd: repoDir, encoding: 'utf8' });
  const dirty = spawnSync('git', ['status', '--porcelain', '--', 'web', 'internal/control/web/embedded'], {
    cwd: repoDir,
    encoding: 'utf8',
  });
  return {
    branch: (branch.stdout || '').trim() || 'unknown',
    commit: (commit.stdout || '').trim() || 'unknown',
    dirtyWeb: Boolean((dirty.stdout || '').trim()),
    dirtyList: (dirty.stdout || '').trim(),
  };
}

function renderReport(results, meta, hardFailed, softNotes) {
  const passCount = results.filter((r) => r.result.ok).length;
  const failCount = results.length - passCount;
  const verdict = hardFailed.length === 0 ? 'PASS' : 'FAIL';
  const lines = [
    `# Frontend verification report`,
    ``,
    `- Generated: \`${stamp}\``,
    `- Branch: \`${meta.branch}\` @ \`${meta.commit}\``,
    `- Verdict: **${verdict}** (${passCount} pass / ${failCount} fail)`,
    `- Dirty web/dist tree: ${meta.dirtyWeb ? '**yes**' : 'no'}`,
    ``,
    `## Gates`,
    ``,
    `| Gate | Hard | Status | Duration | Detail |`,
    `| --- | --- | --- | --- | --- |`,
  ];
  for (const item of results) {
    const status = item.result.ok ? 'PASS' : 'FAIL';
    const detail = item.result.detail.replace(/\n/g, '<br>').slice(0, 500);
    lines.push(
      `| ${item.gate.title} | ${item.gate.hard ? 'yes' : 'no'} | ${status} | ${item.result.durationMs}ms | ${detail} |`
    );
  }
  lines.push('', '## Residual risks / next operator steps', '');
  if (hardFailed.length) {
    lines.push('- Hard gates failed — do not claim frontend delivery until green.');
    for (const id of hardFailed) lines.push(`  - Fix \`${id}\` then re-run \`make web-verify\`.`);
  } else {
    lines.push('- Automated gates green.');
    lines.push('- If UI still looks broken in the browser: restart `nexus web` so the new embedded bundle is loaded (`make build`).');
    lines.push('- Manual smoke (not automated here): open Project Overview, Radar, one Agent terminal, and a second Project focus switch.');
  }
  if (meta.dirtyWeb) {
    lines.push('', '### Dirty paths', '', '```', meta.dirtyList || '(none)', '```');
  }
  if (softNotes.length) {
    lines.push('', '### Notes', '');
    for (const note of softNotes) lines.push(`- ${note}`);
  }
  lines.push(
    '',
    '## How to regenerate',
    '',
    '```bash',
    'make web-verify',
    '# or',
    'npm --prefix web run verify',
    '```',
    ''
  );
  return lines.join('\n');
}

mkdirSync(outDir, { recursive: true });

const meta = gitMeta();
const softNotes = [];
const results = [];
const hardFailed = [];

for (const gate of gates) {
  process.stderr.write(`→ ${gate.id}… `);
  const result = gate.check();
  results.push({ gate, result });
  process.stderr.write(`${result.ok ? 'PASS' : 'FAIL'} (${result.durationMs}ms)\n`);
  if (!result.ok && gate.hard) hardFailed.push(gate.id);
}

if (!existsSync(resolve(repoDir, 'nexus')) && !existsSync(resolve(process.env.HOME || '', '.local/bin/nexus'))) {
  softNotes.push('Binário `nexus` local não encontrado neste check — após PASS, rode `make build` para instalar.');
}

const markdown = renderReport(results, meta, hardFailed, softNotes);
writeFileSync(latestPath, markdown);
const datedPath = resolve(outDir, `FRONTEND_${shortStamp}.md`);
writeFileSync(datedPath, markdown);

const historyLine = `- ${stamp} · ${hardFailed.length ? 'FAIL' : 'PASS'} · ${meta.branch}@${meta.commit} · failed=[${hardFailed.join(',') || '-'}] · ${datedPath.replace(`${repoDir}/`, '')}\n`;
if (!existsSync(historyPath)) {
  writeFileSync(historyPath, `# Frontend verification history\n\n${historyLine}`);
} else {
  writeFileSync(historyPath, readFileSync(historyPath, 'utf8') + historyLine);
}

// Keep DEV/VERIFY.md pointer fresh without rewriting the whole log.
const verifyPath = resolve(repoDir, 'DEV/VERIFY.md');
if (existsSync(verifyPath)) {
  const marker = '<!-- frontend-verify:latest -->';
  const block = [
    marker,
    `## Frontend gate — ${stamp}`,
    '',
    `Verdict: **${hardFailed.length ? 'FAIL' : 'PASS'}**. Relatório completo: [\`DEV/validation/FRONTEND_LATEST.md\`](validation/FRONTEND_LATEST.md).`,
    '',
  ].join('\n');
  let verify = readFileSync(verifyPath, 'utf8');
  // Drop any previous frontend-verify blocks (marker + following Frontend gate sections).
  verify = verify.replace(/\n*<!-- frontend-verify:latest -->[\s\S]*$/m, '\n');
  verify = `${verify.trimEnd()}\n\n${block}\n`;
  writeFileSync(verifyPath, verify);
}

process.stderr.write(`\nReport: ${latestPath}\n`);
process.stderr.write(`Dated:  ${datedPath}\n`);
process.exit(hardFailed.length ? 1 : 0);
