import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);
const srcDir = path.resolve(__dirname, '../src');
const baselinePath = path.resolve(__dirname, 'token-parity-baseline.json');

const FOUNDATION = path.join(srcDir, 'app/workspace-styles/01-foundation.css');
const TYPOGRAPHY = path.join(srcDir, 'styles/_typography.scss');

export const SPACING_PROPS = new Set([
  'padding',
  'padding-top',
  'padding-right',
  'padding-bottom',
  'padding-left',
  'padding-block',
  'padding-block-start',
  'padding-block-end',
  'padding-inline',
  'padding-inline-start',
  'padding-inline-end',
  'margin',
  'margin-top',
  'margin-right',
  'margin-bottom',
  'margin-left',
  'margin-block',
  'margin-block-start',
  'margin-block-end',
  'margin-inline',
  'margin-inline-start',
  'margin-inline-end',
  'gap',
  'row-gap',
  'column-gap',
]);

export const RADIUS_PROPS = new Set([
  'border-radius',
  'border-top-left-radius',
  'border-top-right-radius',
  'border-bottom-left-radius',
  'border-bottom-right-radius',
  'border-start-start-radius',
  'border-start-end-radius',
  'border-end-start-radius',
  'border-end-end-radius',
]);

export const FONT_PROPS = new Set(['font-size']);

const LENGTH = /^(-?\d+(?:\.\d+)?)(px|rem|em)$/;

function stripComments(text) {
  return text.replace(/\/\*[\s\S]*?\*\//g, '').replace(/(^|[^:])\/\/[^\n]*/g, '$1');
}

function normalize(length) {
  return `${Number(length[1])}${length[2]}`;
}

export function parseTokens(text, family) {
  const tokens = new Map();
  const pattern =
    family === 'spacing'
      ? /--nx-spacing-\d+\s*:\s*([^;}]+)[;}]/g
      : family === 'radius'
        ? /--nx-radius(?:-[a-z]+)?\s*:\s*([^;}]+)[;}]/g
        : /--nx-font-size-[a-z0-9]+\s*:\s*([^;}]+)[;}]/g;
  for (const match of text.matchAll(pattern)) {
    const length = match[1].trim().match(LENGTH);
    if (!length) continue;
    const unit = length[2];
    if (!tokens.has(unit)) tokens.set(unit, new Set());
    tokens.get(unit).add(normalize(length));
  }
  return tokens;
}

export function loadTokenTables() {
  const foundation = stripComments(fs.readFileSync(FOUNDATION, 'utf8'));
  const typography = stripComments(fs.readFileSync(TYPOGRAPHY, 'utf8'));
  return {
    spacing: parseTokens(foundation, 'spacing'),
    radius: parseTokens(foundation, 'radius'),
    font: parseTokens(typography, 'font'),
  };
}

function tableFor(prop, tables) {
  if (SPACING_PROPS.has(prop)) return tables.spacing;
  if (RADIUS_PROPS.has(prop)) return tables.radius;
  if (FONT_PROPS.has(prop)) return tables.font;
  return null;
}

function walk(dir, files = []) {
  for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
    const full = path.join(dir, entry.name);
    if (entry.isDirectory()) walk(full, files);
    else if (/\.(css|scss)$/i.test(entry.name)) files.push(full);
  }
  return files;
}

export function collectViolations(tables, files) {
  const violations = [];
  for (const file of files) {
    const rel = path.relative(path.dirname(srcDir), file).split(path.sep).join('/');
    const lines = stripComments(fs.readFileSync(file, 'utf8')).split('\n');
    lines.forEach((line, index) => {
      const match = line.match(/^\s*([a-z-]+)\s*:\s*([^;]+);/);
      if (!match) return;
      const [, prop, raw] = match;
      const table = tableFor(prop, tables);
      if (!table) return;
      const literal = raw.trim().match(LENGTH);
      if (!literal) return;
      const byUnit = table.get(literal[2]);
      if (!byUnit) return;
      const value = normalize(literal);
      if (!byUnit.has(value)) return;
      violations.push({ file: rel, line: index + 1, prop, value });
    });
  }
  return violations;
}

export function keyOf(violation) {
  return `${violation.file}|${violation.prop}|${violation.value}`;
}

export function tally(violations) {
  const counts = {};
  for (const violation of violations) {
    const key = keyOf(violation);
    counts[key] = (counts[key] ?? 0) + 1;
  }
  return counts;
}

export function diffAgainstBaseline(counts, baseline) {
  const added = [];
  const reduced = [];
  for (const [key, count] of Object.entries(counts)) {
    const accepted = baseline[key] ?? 0;
    if (count > accepted) added.push({ key, count, accepted });
  }
  for (const [key, accepted] of Object.entries(baseline)) {
    const count = counts[key] ?? 0;
    if (count < accepted) reduced.push({ key, count, accepted });
  }
  return { added, reduced };
}

function readBaseline() {
  if (!fs.existsSync(baselinePath)) return {};
  return JSON.parse(fs.readFileSync(baselinePath, 'utf8')).entries ?? {};
}

export function writeBaseline(counts) {
  const payload = {
    version: 1,
    note: 'Accepted literal-to-token parity debt. Counts are per file|property|value and may only go down.',
    entries: Object.fromEntries(Object.entries(counts).sort(([a], [b]) => a.localeCompare(b))),
  };
  fs.writeFileSync(baselinePath, `${JSON.stringify(payload, null, 2)}\n`);
}

function main() {
  const tables = loadTokenTables();
  const files = walk(srcDir);
  const violations = collectViolations(tables, files);
  const counts = tally(violations);

  if (process.argv.includes('--update-baseline')) {
    writeBaseline(counts);
    console.log(`[OK] Token parity baseline written: ${violations.length} accepted literals.`);
    return;
  }

  const baseline = readBaseline();
  const { added, reduced } = diffAgainstBaseline(counts, baseline);

  for (const entry of added) {
    console.error(`  ${entry.key}  (${entry.count} now, ${entry.accepted} accepted)`);
  }
  if (added.length > 0) {
    console.error(
      `[FAIL] Token parity regressed: ${added.length} file/property/value keys exceed the accepted baseline.`,
    );
    console.error('       Use the semantic token instead, or run --update-baseline after review.');
    process.exitCode = 1;
    return;
  }

  if (reduced.length > 0) {
    console.log(
      `[OK] Token parity improved: ${reduced.length} baseline keys now have fewer literals. Run --update-baseline to lock the ratchet.`,
    );
  }
  console.log(
    `[OK] Token parity check passed: ${violations.length} accepted literals, 0 regressions.`,
  );
}

if (process.argv[1] && path.resolve(process.argv[1]) === __filename) main();