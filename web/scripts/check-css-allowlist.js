import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);
const srcDir = path.resolve(__dirname, '../src');

// Explicit global allowlist
const allowedPatterns = [
  /^src\/index\.css$/,
  /^src\/app\/workspace-os\.css$/,
  /^src\/app\/workspace-styles\/[0-9]{2}-[a-z0-9-]+\.css$/,
  /^src\/styles\/[a-zA-Z0-9_.-]+\.scss$/,
  /^src\/[a-zA-Z0-9_/-]+\.module\.scss$/,
  /^src\/[a-zA-Z0-9_/-]+\.module\.css$/,
];

function scanDirectory(dir, fileList = []) {
  const files = fs.readdirSync(dir);
  for (const file of files) {
    const fullPath = path.join(dir, file);
    const stat = fs.statSync(fullPath);
    if (stat.isDirectory()) {
      scanDirectory(fullPath, fileList);
    } else if (/\.(css|scss|sass)$/i.test(file)) {
      fileList.push(fullPath);
    }
  }
  return fileList;
}

const stylesheets = scanDirectory(srcDir);
const violations = [];
const stylesheetSet = new Set(stylesheets);
const partialImporters = new Map();

// Sass partials imported by a CSS Module stay scoped to that module.
// Do not grant them a global allowlist exemption when they are imported
// from a global stylesheet, or when they are orphaned.
for (const importer of stylesheets) {
  if (!/\.s[ac]ss$/i.test(importer)) continue;
  const source = fs.readFileSync(importer, 'utf8');
  const imports = source.matchAll(/@(?:use|forward|import)\s+['"]([^'"]+)['"]/g);
  for (const match of imports) {
    const specifier = match[1];
    if (!specifier.startsWith('.')) continue;
    const target = path.resolve(path.dirname(importer), specifier);
    const partial = path.join(
      path.dirname(target),
      '_' + path.basename(target).replace(/\.scss$/, '') + '.scss',
    );
    if (!stylesheetSet.has(partial)) continue;
    if (!partialImporters.has(partial)) partialImporters.set(partial, []);
    partialImporters.get(partial).push(importer);
  }
}

for (const file of stylesheets) {
  const relativePath = path.relative(path.resolve(__dirname, '..'), file).replace(/\\/g, '/');
  const isAllowlisted = allowedPatterns.some((pattern) => pattern.test(relativePath));
  const importers = partialImporters.get(file) || [];
  const isScopedPartial =
    /^_[a-zA-Z0-9-]+\.scss$/.test(path.basename(file)) &&
    importers.length > 0 &&
    importers.every((importer) => /\.module\.scss$/.test(importer));
  if (!isAllowlisted && !isScopedPartial) {
    violations.push(relativePath);
  }
}
if (violations.length > 0) {
  console.error('\x1b[31m[ERROR] Prohibited stylesheet architecture detected!\x1b[0m');
  console.error('Local styles must be CSS Modules or Sass partials used exclusively by CSS Modules.');
  console.error('Global styles are restricted exclusively to explicit allowlisted files.');
  console.error('\nViolating files:');
  violations.forEach((v) => console.error(`  - ${v}`));
  process.exit(1);
}

console.log('\x1b[32m[OK] Stylesheet architecture check passed: all styles conform to SCSS Modules & global allowlist.\x1b[0m');
