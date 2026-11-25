import test from 'node:test';
import assert from 'node:assert/strict';
import { createNexusE2EBuildEnv, fixtureFeatureEnvironment } from './e2e-platform.mjs';

test('build is independent of inherited feature switches', () => {
  const source = {
    PATH: '/usr/bin',
    NEXUS_EXPERIMENTAL_COMPOSER_FLOW: '1',
    NEXUS_FEATURES_ENABLED: 'true',
    NEXUS_FEATURE_VOICE_ACTIONS: 'true',
  };
  assert.deepEqual(createNexusE2EBuildEnv(source), { PATH: '/usr/bin' });
  assert.equal(source.NEXUS_FEATURES_ENABLED, 'true');
});

test('fixture opts in only to its requested server capabilities', () => {
  const source = { PATH: '/usr/bin', NEXUS_FEATURE_MISSIONS: 'true' };
  assert.deepEqual(fixtureFeatureEnvironment(source), {
    PATH: '/usr/bin',
    NEXUS_FEATURES_ENABLED: 'false',
  });
  assert.deepEqual(
    fixtureFeatureEnvironment(source, { composerFlow: true, features: ['DEVICE_PREVIEW'] }),
    {
      PATH: '/usr/bin',
      NEXUS_FEATURES_ENABLED: 'true',
      NEXUS_FEATURE_DEVICE_PREVIEW: 'true',
      NEXUS_FEATURE_COMPOSER: 'true',
      NEXUS_FEATURE_FLOW: 'true',
      NEXUS_FEATURE_MISSIONS: 'true',
    },
  );
});
