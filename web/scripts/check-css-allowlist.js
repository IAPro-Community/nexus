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

for (const file of stylesheets) {
  const relativePath = path.relative(path.resolve(__dirname, '..'), file).replace(/\\/g, '/');
  const isAllowed = allowedPatterns.some((pattern) => pattern.test(relativePath));
  if (!isAllowed) {
    violations.push(relativePath);
  }
}

if (violations.length > 0) {
  console.error('\x1b[31m[ERROR] Prohibited stylesheet architecture detected!\x1b[0m');
  console.error('All new and local component styles must use SCSS Modules (*.module.scss).');
  console.error('Global styles are restricted exclusively to explicit allowlisted files.');
  console.error('\nViolating files:');
  violations.forEach((v) => console.error(`  - ${v}`));
  process.exit(1);
}

console.log('\x1b[32m[OK] Stylesheet architecture check passed: all styles conform to SCSS Modules & global allowlist.\x1b[0m');
