import assert from 'node:assert/strict';
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import test from 'node:test';
import {
  collectViolations,
  diffAgainstBaseline,
  parseTokens,
  tally,
} from './check-token-parity.mjs';

test('parseTokens reads spacing, radius and font families from token declarations', () => {
  const spacing = parseTokens('--nx-spacing-2: 8px;\n--nx-spacing-3: 12px;', 'spacing');
  assert.deepEqual([...spacing.get('px')].sort(), ['12px', '8px']);

  const radius = parseTokens('--nx-radius-sm: 4px;\n--nx-radius-full: 999px;', 'radius');
  assert.deepEqual([...radius.get('px')].sort(), ['4px', '999px']);

  const font = parseTokens('--nx-font-size-sm: 0.929rem;', 'font');
  assert.deepEqual([...font.get('rem')], ['0.929rem']);
});

test('parseTokens ignores alias declarations that do not carry a literal length', () => {
  const spacing = parseTokens('--nx-space-2: var(--nx-spacing-2);', 'spacing');
  assert.equal(spacing.size, 0);
});

test('collectViolations flags single literals that duplicate a token value', () => {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'token-parity-'));
  const file = path.join(dir, 'sample.module.scss');
  fs.writeFileSync(
    file,
    [
      '.a {',
      '  padding: 8px;',
      '  gap: var(--nx-space-2);',
      '  margin: 0 8px;',
      '  border-radius: 4px;',
      '  font-size: 11px;',
      '  --nx-spacing-2: 8px;',
      '}',
    ].join('\n'),
  );

  const tables = {
    spacing: new Map([['px', new Set(['8px'])]]),
    radius: new Map([['px', new Set(['4px'])]]),
    font: new Map([['rem', new Set(['0.786rem'])]]),
  };
  const violations = collectViolations(tables, [file]).map((entry) => `${entry.prop}:${entry.value}`);
  assert.deepEqual(violations, ['padding:8px', 'border-radius:4px']);
});

test('diffAgainstBaseline is a ratchet: only regressions fail', () => {
  const baseline = { 'a.scss|gap|8px': 2 };
  const grown = diffAgainstBaseline({ 'a.scss|gap|8px': 3 }, baseline);
  assert.equal(grown.added.length, 1);

  const shrunk = diffAgainstBaseline({ 'a.scss|gap|8px': 1 }, baseline);
  assert.equal(shrunk.added.length, 0);
  assert.equal(shrunk.reduced.length, 1);

  const stable = diffAgainstBaseline({ 'a.scss|gap|8px': 2 }, baseline);
  assert.equal(stable.added.length, 0);
  assert.equal(stable.reduced.length, 0);
});

test('tally groups violations by file, property and value', () => {
  const counts = tally([
    { file: 'a.scss', line: 1, prop: 'gap', value: '8px' },
    { file: 'a.scss', line: 4, prop: 'gap', value: '8px' },
    { file: 'b.scss', line: 2, prop: 'padding', value: '4px' },
  ]);
  assert.deepEqual(counts, { 'a.scss|gap|8px': 2, 'b.scss|padding|4px': 1 });
});