import { readFileSync, readdirSync, statSync, existsSync } from 'node:fs';
import path from 'node:path';

const root = process.cwd();
const skip = new Set(['node_modules', '.git', '.tmp', 'dist', 'build', 'embedded', '.tempmediaStorage', '.worktrees']);
const mdFiles = [];
function walk(dir) {
  for (const entry of readdirSync(dir)) {
    if (skip.has(entry)) continue;
    const full = path.join(dir, entry);
    const st = statSync(full);
    if (st.isDirectory()) walk(full);
    else if (/\.mdx?$/i.test(entry)) mdFiles.push(full);
  }
}
walk(root);

const broken = [];
const linkCount = new Map();
for (const file of mdFiles) {
  const text = readFileSync(file, 'utf8');
  const rel = path.relative(root, file).replaceAll('\\', '/');
  const re = /!?\[[^\]]*\]\(([^)\s]+)(?:\s+"[^"]*")?\)/g;
  let m;
  while ((m = re.exec(text))) {
    let target = m[1];
    if (/^(https?:|mailto:|tel:|data:|#)/i.test(target)) continue;
    if (target.startsWith('<') && target.endsWith('>')) target = target.slice(1, -1);
    const clean = decodeURIComponent(target.split('#')[0].split('?')[0]);
    if (!clean) continue;
    linkCount.set(rel, (linkCount.get(rel) || 0) + 1);
    const abs = path.resolve(path.dirname(file), clean);
    if (!existsSync(abs)) broken.push({ file: rel, target, resolved: path.relative(root, abs).replaceAll('\\', '/') });
  }
}

const versions = new Map();
const versionRe = /\b\d+\.\d+\.\d+(?:-beta\.\d+)?\b/g;
for (const file of mdFiles) {
  const rel = path.relative(root, file).replaceAll('\\', '/');
  if (rel.startsWith('DEV/')) continue;
  const text = readFileSync(file, 'utf8');
  for (const m of text.matchAll(versionRe)) {
    const v = m[0];
    if (!/^\d+\.\d+\.\d+/.test(v)) continue;
    if (!versions.has(v)) versions.set(v, []);
    versions.get(v).push(rel);
  }
}

const screenshots = [];
for (const file of mdFiles) {
  const rel = path.relative(root, file).replaceAll('\\', '/');
  const text = readFileSync(file, 'utf8');
  for (const m of text.matchAll(/!\[[^\]]*\]\(([^)]+)\)/g)) {
    const target = m[1];
    if (/^(https?:|data:)/i.test(target)) continue;
    screenshots.push({ file: rel, target });
  }
}

console.log(JSON.stringify({
  markdownFiles: mdFiles.length,
  docsMarkdown: mdFiles.filter((f) => path.relative(root, f).startsWith('docs')).length,
  devMarkdown: mdFiles.filter((f) => path.relative(root, f).startsWith('DEV')).length,
  brokenLinks: broken.length,
  broken: broken.slice(0, 200),
  versions: Object.fromEntries([...versions.entries()].sort((a, b) => b[1].length - a[1].length).slice(0, 20)),
  imageRefs: screenshots.length,
}, null, 2));
