import test from 'node:test';
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { verifyAgentWorkspaceAcceptanceDoc } from '../agent-workspace-doc-sync.mjs';

const root = resolve(import.meta.dirname, '../..');
const manifest = JSON.parse(readFileSync(resolve(root, 'docs/reference/agent-workspace-v1.json'), 'utf8'));
const markdown = readFileSync(resolve(root, 'docs/product/agent-workspace-v1-acceptance.md'), 'utf8');
const fixtures = [
  { id: 'alpha-one', sourceStatus: 'partial', issue: 12 },
  { id: 'beta-two', sourceStatus: 'missing', issue: 27 },
];
const sample = [
  '| `alpha-one` | Alpha | partial | [#12](https://github.com/IAPro-Community/nexus/issues/12) | Acceptance |',
  '| `beta-two` | Beta | missing | [#27](https://github.com/IAPro-Community/nexus/issues/27) | Acceptance |',
].join('\n');

test('canonical acceptance Markdown matches all 48 manifest entries', () => {
  assert.equal(manifest.features.length, 48);
  assert.deepEqual(verifyAgentWorkspaceAcceptanceDoc(manifest.features, markdown), []);
});
test('valid fixture is accepted', () => {
  assert.deepEqual(verifyAgentWorkspaceAcceptanceDoc(fixtures, sample), []);
});
test('stale source status is rejected', () => {
  assert.match(verifyAgentWorkspaceAcceptanceDoc(fixtures, sample.replace('partial', 'missing')).join('\n'), /sourceStatus drift/);
});
test('wrong issue URL is rejected', () => {
  assert.match(verifyAgentWorkspaceAcceptanceDoc(fixtures, sample.replace('/12)', '/99)')).join('\n'), /issue link differs/);
});
test('missing, extra, and repeated feature rows are rejected', () => {
  assert.match(verifyAgentWorkspaceAcceptanceDoc(fixtures, sample.split('\n')[0]).join('\n'), /missing from acceptance table/);
  assert.match(verifyAgentWorkspaceAcceptanceDoc(fixtures, sample + '\n' + sample.split('\n')[0]).join('\n'), /duplicate/);
  assert.match(verifyAgentWorkspaceAcceptanceDoc(fixtures, sample + '\n| `gamma` | Gamma | partial | [#5](https://github.com/IAPro-Community/nexus/issues/5) | Acceptance |').join('\n'), /undocumented feature id/);
});
