#!/usr/bin/env node
// Real Core/HTTP/DOM/PTY E2E for Agent grouping. OpenCode is a local fixture
// binary: it reports a version and execs a shell, so no provider quota is used.
import assert from 'node:assert/strict';
import { existsSync, mkdirSync, writeFileSync } from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import {
  launchFixtureBrowser,
  meaningfulFixtureRequestFailures,
  seedFixtureProject,
  startFixtureServer,
  terminateProcess,
} from './e2e-fixture.mjs';

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const repoRoot = path.resolve(__dirname, '..', '..');
const artifacts =
  process.env.NEXUS_E2E_ARTIFACTS_DIR || path.join(repoRoot, '.tempmediaStorage/agent-groups');
mkdirSync(artifacts, { recursive: true });

async function jsonResponse(response, expectedStatus, label) {
  assert.equal(response.status(), expectedStatus, `${label} returned HTTP ${response.status()}`);
  return response.json();
}

async function apiPost(context, url, csrf, data, label, expectedStatus = 200) {
  return jsonResponse(
    await context.request.post(url, {
      headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': csrf },
      data,
    }),
    expectedStatus,
    label,
  );
}

async function readAgent(context, apiOrigin, agentId) {
  return jsonResponse(
    await context.request.get(`${apiOrigin}/api/v1/agents/${encodeURIComponent(agentId)}`),
    200,
    `read Agent ${agentId}`,
  );
}

async function captureAtViewport(page, viewport, fileName) {
  await page.setViewportSize(viewport);
  await page.waitForTimeout(150);
  const dimensions = await page.evaluate(() => ({
    innerWidth: window.innerWidth,
    documentWidth: document.documentElement.scrollWidth,
    bodyWidth: document.body.scrollWidth,
  }));
  assert.ok(
    dimensions.documentWidth <= dimensions.innerWidth,
    `Agent groups overflow at ${viewport.width}px: ${JSON.stringify(dimensions)}`,
  );
  const screenshot = path.join(artifacts, fileName);
  await page.screenshot({ path: screenshot, fullPage: true });
  return { viewport, screenshot, dimensions };
}

async function main() {
  console.log('=== Agent groups with a live supervised terminal E2E ===');
  const fixtureRoot = path.join(os.tmpdir(), `nexus-agent-groups-${process.pid}`);
  const fixtureBin = path.join(fixtureRoot, 'bin');
  mkdirSync(fixtureBin, { recursive: true });
  const fixtureCommand = process.platform === 'win32' ? 'opencode.cmd' : 'opencode';
  const fixturePath = path.join(fixtureBin, fixtureCommand);
  const fixtureContents =
    process.platform === 'win32'
      ? '@echo off\r\nif "%1"=="--version" (echo opencode-fixture/1 & exit /b 0)\r\n%COMSPEC% /Q\r\n'
      : '#!/bin/sh\nif [ "$1" = "--version" ]; then echo opencode-fixture/1; exit 0; fi\nexec /bin/sh\n';
  writeFileSync(fixturePath, fixtureContents, { mode: 0o700 });

  const previousPath = process.env.PATH || '';
  const previousRealHome = process.env.NEXUS_REAL_HOME;
  process.env.PATH = `${fixtureBin}${path.delimiter}${previousPath}`;
  process.env.NEXUS_REAL_HOME = path.join(fixtureRoot, 'host-home');

  let fixture;
  let browser;
  let context;
  let page;
  let collectors;
  const createdAgents = [];
  const pageStartRequests = [];
  let explicitStartCount = 0;

  try {
    fixture = await startFixtureServer();
    ({ browser, context, page, collectors } = await launchFixtureBrowser(fixture.bootstrapUrl));
    page.setDefaultTimeout(15000);
    page.on('request', (request) => {
      if (request.method() === 'POST' && /\/api\/v1\/agents\/[^/]+\/start$/.test(request.url())) {
        pageStartRequests.push(request.url());
      }
    });

    const { project, apiOrigin } = await seedFixtureProject(
      context,
      fixture.bootstrapUrl,
      fixture.isolatedProjectPath,
    );
    const session = await jsonResponse(
      await context.request.get(`${apiOrigin}/api/v1/session`),
      200,
      'read authenticated session',
    );
    const csrf = session.csrf_token;
    assert.ok(csrf, 'fixture session must expose CSRF token');

    const provider = await apiPost(
      context,
      `${apiOrigin}/api/v1/providers`,
      csrf,
      { provider: 'opencode', profile: 'agent-groups-fixture' },
      'create local OpenCode fixture profile',
      201,
    );
    assert.equal(provider.profile.provider, 'opencode');

    const resources = await jsonResponse(
      await context.request.get(`${apiOrigin}/api/v1/resources`),
      200,
      'list local fixture resources',
    );
    const fixtureResource = resources.accounts.find(
      (account) => account.provider === 'opencode' && account.profile === 'agent-groups-fixture',
    );
    assert.ok(fixtureResource, 'local OpenCode fixture profile must be discoverable');
    assert.equal(fixtureResource.quota_view?.status, 'UNSUPPORTED');
    assert.equal(fixtureResource.eligibility?.eligible, true);

    const agentNames = ['Groups E2E Alpha', 'Groups E2E Beta'];
    for (const name of agentNames) {
      const agent = await apiPost(
        context,
        `${apiOrigin}/api/v1/projects/${encodeURIComponent(project.id)}/agents`,
        csrf,
        { name, role: 'developer' },
        `create ${name}`,
        201,
      );
      createdAgents.push(agent);
      await apiPost(
        context,
        `${apiOrigin}/api/v1/resources/select`,
        csrf,
        {
          agent_id: agent.id,
          provider: 'opencode',
          profile: 'agent-groups-fixture',
          policy: 'MANUAL',
        },
        `allocate local fixture to ${name}`,
      );
      const started = await apiPost(
        context,
        `${apiOrigin}/api/v1/agents/${encodeURIComponent(agent.id)}/start`,
        csrf,
        {},
        `start supervised ${name}`,
      );
      explicitStartCount += 1;
      assert.equal(started.runtime.agent_id, agent.id);
      assert.equal(started.runtime.state, 'RUNNING');
    }
    assert.equal(explicitStartCount, 2, 'exactly one explicit initial start per fixture Agent');

    const beforeById = new Map();
    for (const agent of createdAgents) {
      const detail = await readAgent(context, apiOrigin, agent.id);
      assert.equal(detail.generations.length, 1, `${agent.name} must have exactly one generation`);
      beforeById.set(agent.id, detail.generations[0]);
    }

    await page.goto(`${apiOrigin}/p/${encodeURIComponent(project.id)}/agents`, {
      waitUntil: 'domcontentloaded',
      timeout: 20000,
    });
    await page.getByRole('heading', { name: /Agent fleet|Frota de Agentes|Flota de Agentes/i }).waitFor();

    const createGroupButton = page.getByRole('button', { name: /Create group|Criar grupo/i });
    await createGroupButton.click();
    await page.getByLabel(/Group name|Nome do grupo|Nombre del grupo/i).fill('Release E2E');
    await page.getByLabel(/^Task$|^Tarefa$/i).fill('Preserve live session while organizing');
    await page.getByLabel(/^Worktree$/i).fill('../worktrees/release-e2e');
    await page.getByLabel(/^Feature$/i).fill('agent-groups');
    const createResponse = page.waitForResponse(
      (response) =>
        response.request().method() === 'POST' &&
        response.url().endsWith(`/api/v1/projects/${project.id}/agent-groups`),
    );
    await page.getByRole('button', { name: /Create group|Criar grupo|Crear grupo/i }).last().click();
    const createdGroup = await jsonResponse(await createResponse, 201, 'create group through Agents UI');

    const groupFilter = page.getByLabel(/Filter by group|Filtrar por grupo/i);
    await groupFilter.selectOption('all');
    for (const agent of createdAgents) {
      const card = page.locator(`article[data-agent-id="${agent.id}"]`);
      await card.waitFor({ state: 'visible' });
      await card
        .getByRole('combobox', { name: new RegExp(agent.name, 'i') })
        .selectOption(createdGroup.id);
    }

    await groupFilter.selectOption('');
    await page.waitForFunction(
      () => document.querySelectorAll('article[data-agent-id]').length === 0,
      null,
      { timeout: 10000 },
    );
    await groupFilter.selectOption(createdGroup.id);
    await page.waitForFunction(
      () => document.querySelectorAll('article[data-agent-id]').length === 2,
      null,
      { timeout: 10000 },
    );
    const beforeOrder = await page.locator('article[data-agent-id]').evaluateAll((cards) =>
      cards.map((card) => card.getAttribute('data-agent-id')),
    );
    assert.equal(beforeOrder.length, 2, 'group filter must show both persistent Agents');
    assert.deepEqual(beforeOrder, createdAgents.map((agent) => agent.id));

    const betaCard = page.locator(`article[data-agent-id="${createdAgents[1].id}"]`);
    await betaCard.getByRole('button', { name: /Move agent up|Mover agente para cima/i }).click();
    await page.waitForFunction(
      (expected) =>
        [...document.querySelectorAll('article[data-agent-id]')]
          .map((element) => element.getAttribute('data-agent-id'))
          .join('|') === expected.join('|'),
      [createdAgents[1].id, createdAgents[0].id],
      { timeout: 10000 },
    );

    const orderedGroupsResponse = await context.request.get(
      `${apiOrigin}/api/v1/projects/${encodeURIComponent(project.id)}/agent-groups`,
    );
    const groups = await jsonResponse(orderedGroupsResponse, 200, 'read group after reorder');
    const persistedGroup = groups.find((group) => group.id === createdGroup.id);
    assert.deepEqual(
      persistedGroup.agent_ids,
      [createdAgents[1].id, createdAgents[0].id],
      'manual reorder must persist the stable Agent IDs in the new order',
    );
    const afterOrder = await page.locator('article[data-agent-id]').evaluateAll((cards) =>
      cards.map((card) => card.getAttribute('data-agent-id')),
    );
    assert.deepEqual(afterOrder, [createdAgents[1].id, createdAgents[0].id], 'visible list follows persisted order');

    const groupDetail = await jsonResponse(
      await context.request.get(
        `${apiOrigin}/api/v1/projects/${encodeURIComponent(project.id)}/agent-groups`,
      ),
      200,
      'read group metadata',
    );
    const confirmedGroup = groupDetail.find((group) => group.id === createdGroup.id);
    assert.equal(confirmedGroup.task, 'Preserve live session while organizing');
    assert.equal(confirmedGroup.worktree, '../worktrees/release-e2e');
    assert.equal(confirmedGroup.feature, 'agent-groups');

    const viewportEvidence = [];
    viewportEvidence.push(
      await captureAtViewport(page, { width: 1280, height: 900 }, 'agent-groups-1280x900.png'),
      await captureAtViewport(page, { width: 768, height: 1024 }, 'agent-groups-768x1024.png'),
      await captureAtViewport(page, { width: 320, height: 568 }, 'agent-groups-320x568.png'),
      await captureAtViewport(page, { width: 390, height: 844 }, 'agent-groups-390x844.png'),
    );
    const mobileDimensions = viewportEvidence.at(-1).dimensions;

    await page.setViewportSize({ width: 1280, height: 900 });
    const terminalButton = betaCard.getByRole('button', { name: /^Terminal$/i });
    const terminalSocketPromise = page.waitForEvent('websocket', {
      predicate: (socket) => socket.url().includes(`/agents/${createdAgents[1].id}/terminal`),
      timeout: 15000,
    });
    await terminalButton.click();
    const terminalSocket = await terminalSocketPromise;
    await page.locator('.nx-agent-terminal').waitFor({ state: 'visible', timeout: 15000 });
    const terminalNotice = page.locator('[data-feature-discovery="agent-terminal"] [role="status"]');
    if (await terminalNotice.count()) {
      await terminalNotice
        .getByRole('button', { name: /Dismiss|Dispensar|Descartar/i })
        .click();
    }
    const terminalLayer = page.locator('.nx-tour-layer');
    if (await terminalLayer.count()) {
      await terminalLayer.getByRole('button', { name: /Close tour|Fechar tour|Cerrar recorrido/i }).click();
    }
    assert.ok(
      (await page.locator('.nx-agent-terminal__xterm .xterm').count()) > 0,
      'the real terminal surface must render its xterm instance',
    );

    const betaGeneration = beforeById.get(createdAgents[1].id);
    assert.ok(
      new URL(terminalSocket.url()).searchParams.get('runtime_id') === betaGeneration.runtime_id,
      `terminal must attach the same live runtime ${betaGeneration.runtime_id}`,
    );

    for (const agent of createdAgents) {
      const detail = await readAgent(context, apiOrigin, agent.id);
      assert.equal(detail.generations.length, 1, `${agent.name} must not receive a second generation`);
      const generation = detail.generations[0];
      assert.equal(generation.id, beforeById.get(agent.id).id);
      assert.equal(generation.runtime_id, beforeById.get(agent.id).runtime_id);
    }
    assert.equal(
      pageStartRequests.length,
      0,
      `Agents UI/filter/reorder/terminal callback must not call start: ${pageStartRequests.join(', ')}`,
    );

    await page.screenshot({ path: path.join(artifacts, 'agent-groups-live-terminal.png'), fullPage: true });
    writeFileSync(
      path.join(artifacts, 'agent-groups-live-session-pass.json'),
      `${JSON.stringify(
        {
          result: 'PASS_LOCAL_E2E_NOT_RELEASE_CERTIFICATION',
          providerFixture: 'OpenCode fixture binary execing local shell; no provider quota/network calls',
          projectId: project.id,
          group: {
            id: confirmedGroup.id,
            task: confirmedGroup.task,
            worktree: confirmedGroup.worktree,
            feature: confirmedGroup.feature,
            orderedAgentIds: confirmedGroup.agent_ids,
          },
          generations: createdAgents.map((agent) => ({
            agentId: agent.id,
            generationId: beforeById.get(agent.id).id,
            runtimeId: beforeById.get(agent.id).runtime_id,
            countAfterReorder: 1,
          })),
          terminalWebSocket: terminalSocket.url(),
          additionalStartCallsFromUI: pageStartRequests.length,
          viewports: viewportEvidence,
          terminalScreenshot: path.join(artifacts, 'agent-groups-live-terminal.png'),
        },
        null,
        2,
      )}\n`,
    );
    console.log('PASS Core created two real supervised OpenCode fixture runtimes; no provider quota was called');
    console.log('PASS UI created a task/worktree/feature group and persisted its ordered Agent IDs');
    console.log('PASS live terminal attached to the same Agent ID and runtime generation after reorder');
    console.log(`PASS browser issued ${pageStartRequests.length} additional start calls after initial setup`);
    console.log(`CAPTURE ${path.join(artifacts, 'agent-groups-1280x900.png')}`);
    console.log(`CAPTURE ${path.join(artifacts, 'agent-groups-390x844.png')} ${JSON.stringify(mobileDimensions)}`);
    assert.equal(collectors.pageErrors.length, 0, collectors.pageErrors.join(' | '));
    assert.deepEqual(meaningfulFixtureRequestFailures(collectors.requestFailures), []);
  } catch (error) {
    const redact = (value) => String(value).replace(/nexus_bootstrap=[a-f0-9]+/gi, 'nexus_bootstrap=[REDACTED]');
    writeFileSync(
      path.join(artifacts, 'agent-groups-live-session-failure.json'),
      `${JSON.stringify({ url: page?.url(), error: redact(error?.stack || error), server: redact(fixture?.getOutput() || '') }, null, 2)}\n`,
    );
    if (page) {
      await page.screenshot({ path: path.join(artifacts, 'agent-groups-live-session-failure.png'), fullPage: true }).catch(() => {});
    }
    throw error;
  } finally {
    if (context && fixture) {
      const apiOrigin = new URL(fixture.bootstrapUrl).origin;
      const sessionResponse = await context.request.get(`${apiOrigin}/api/v1/session`).catch(() => null);
      if (sessionResponse?.ok()) {
        const session = await sessionResponse.json().catch(() => null);
        for (const agent of createdAgents) {
          await context.request
            .post(`${apiOrigin}/api/v1/agents/${encodeURIComponent(agent.id)}/stop`, {
              headers: { 'X-CSRF-Token': session?.csrf_token || '' },
              data: {},
            })
            .catch(() => {});
        }
      }
      await context.close().catch(() => {});
    }
    if (browser) await browser.close().catch(() => {});
    if (fixture) await terminateProcess(fixture.server);
    process.env.PATH = previousPath;
    if (previousRealHome === undefined) delete process.env.NEXUS_REAL_HOME;
    else process.env.NEXUS_REAL_HOME = previousRealHome;
  }
}

main().catch((error) => {
  console.error(error?.stack || error);
  process.exitCode = 1;
});
