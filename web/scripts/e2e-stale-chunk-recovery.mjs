#!/usr/bin/env node
import assert from 'node:assert/strict';
import { mkdirSync } from 'node:fs';
import path from 'node:path';
import {
  launchFixtureBrowser,
  seedFixtureProject,
  startFixtureServer,
  terminateProcess,
} from './e2e-fixture.mjs';

async function main() {
  const fixture = await startFixtureServer();
  let browser;
  let context;
  let page;
  const currentChunkResponses = [];

  try {
    ({ browser, context, page } = await launchFixtureBrowser(fixture.bootstrapUrl, {
      clearStorage: false,
      blockServiceWorkers: true,
    }));
    const { project, apiOrigin } = await seedFixtureProject(
      context,
      fixture.bootstrapUrl,
      fixture.isolatedProjectPath,
    );
    const session = await (await context.request.get(`${apiOrigin}/api/v1/session`)).json();
    const disposablePath = path.join(fixture.isolatedDataDir, 'disposable-project');
    mkdirSync(disposablePath, { recursive: true });
    const createResponse = await context.request.post(`${apiOrigin}/api/v1/projects`, {
      headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': session.csrf_token },
      data: { name: 'Stale Chunk Disposable', path: disposablePath },
    });
    assert.equal(createResponse.status(), 201, 'disposable project should be created');
    const disposableProject = await createResponse.json();

    page.on('response', (response) => {
      if (/\/chunks\/[^/]+\.js$/.test(new URL(response.url()).pathname)) {
        currentChunkResponses.push(response.status());
      }
    });
    const workspaceUrl = `${apiOrigin}/p/${encodeURIComponent(project.id)}/overview`;
    await page.goto(workspaceUrl, {
      waitUntil: 'domcontentloaded',
    });
    await page.locator('#overview-runtimes-title').waitFor({ timeout: 20000 });
    const responseCountBeforeReload = currentChunkResponses.length;
    await page.reload({ waitUntil: 'domcontentloaded' });
    await page.locator('#overview-runtimes-title').waitFor({ timeout: 20000 });
    assert.ok(
      currentChunkResponses.slice(responseCountBeforeReload).includes(200),
      `reloaded workspace must fetch current chunks: ${currentChunkResponses}`,
    );
    const oldChunk = await context.request.get(`${apiOrigin}/chunks/ProjectOverviewSurface-old.js`);
    assert.equal(oldChunk.status(), 404, 'an old generated chunk must return 404');
    assert.doesNotMatch(oldChunk.headers()['content-type'] || '', /text\/html/i);

    const deleteResponse = await context.request.delete(
      `${apiOrigin}/api/v1/projects/${disposableProject.id}`,
      { headers: { 'X-CSRF-Token': session.csrf_token } },
    );
    assert.equal(deleteResponse.status(), 200, 'deleting an idle project should succeed');
    const remainingProjects = await (
      await context.request.get(`${apiOrigin}/api/v1/projects`)
    ).json();
    assert.ok(
      remainingProjects.every((candidate) => candidate.id !== disposableProject.id),
      'the disposable project should be removed',
    );
    console.log('Stale chunk recovery and idle project deletion E2E passed.');
  } finally {
    await browser?.close();
    await terminateProcess(fixture.server);
  }
}

main().catch((error) => {
  console.error(error);
  process.exitCode = 1;
});
