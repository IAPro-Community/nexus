import assert from 'node:assert/strict';
import { test } from 'node:test';
import { createNexusE2EBuildEnv } from './e2e-platform.mjs';

test('regular E2E builds keep experimental Composer and Flow disabled', () => {
  const source = { PATH: '/usr/bin', NEXUS_EXPERIMENTAL_COMPOSER_FLOW: '1' };
  const result = createNexusE2EBuildEnv(source);

  assert.equal(result.NEXUS_EXPERIMENTAL_COMPOSER_FLOW, '0');
  assert.equal(source.NEXUS_EXPERIMENTAL_COMPOSER_FLOW, '1');
});

test('Composer and Flow E2E builds opt in explicitly', () => {
  const result = createNexusE2EBuildEnv({ PATH: '/usr/bin' }, { composerFlow: true });

  assert.equal(result.NEXUS_EXPERIMENTAL_COMPOSER_FLOW, '1');
  assert.equal(result.PATH, '/usr/bin');
});
