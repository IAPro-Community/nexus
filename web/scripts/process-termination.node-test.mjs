import assert from 'node:assert/strict';
import { spawn } from 'node:child_process';
import { test } from 'node:test';
import { terminateProcess } from './process-termination.mjs';

test('terminateProcess waits until the child has actually exited', async () => {
  const child = spawn(process.execPath, ['-e', 'setInterval(() => {}, 1000)'], {
    detached: process.platform !== 'win32',
    stdio: 'ignore',
  });

  await new Promise((resolve, reject) => {
    child.once('spawn', resolve);
    child.once('error', reject);
  });

  await terminateProcess(child);
  assert.ok(
    child.exitCode !== null || child.signalCode !== null,
    'child must have exited before cleanup continues',
  );
});
