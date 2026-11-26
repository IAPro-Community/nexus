#!/usr/bin/env node
import assert from 'node:assert/strict';
import { existsSync, mkdirSync, writeFileSync } from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { request as pwRequest } from 'playwright';
import {
  launchFixtureBrowser,
  meaningfulFixtureRequestFailures,
  seedFixtureProject,
  startFixtureServer,
  terminateProcess,
} from './e2e-fixture.mjs';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);
const repoRoot = path.resolve(__dirname, '..', '..');
const outDir =
  process.env.NEXUS_E2E_ARTIFACTS_DIR ||
  path.resolve(repoRoot, 'DEV', 'validation', 'nex-2', '8d1f1592', 'p4-p5-assistant');
if (!existsSync(outDir)) mkdirSync(outDir, { recursive: true });

async function main() {
  console.log('=== Nex 2.0 P4/P5 Assistant, Memory & Audited Actions E2E ===');
  const { server, bootstrapUrl, isolatedProjectPath, getOutput } = await startFixtureServer();

  let browser;
  let context;
  let page;
  let collectors;
  let unauthApi;
  try {
    ({ browser, context, page, collectors } = await launchFixtureBrowser(bootstrapUrl));
    const { apiOrigin } = await seedFixtureProject(context, bootstrapUrl, isolatedProjectPath);

    // 1. Real server auth, CSRF, forbidden action policy, and idempotent confirm
    unauthApi = await pwRequest.newContext();
    const unauthActions = await unauthApi.get(`${apiOrigin}/api/v1/nex/actions`);
    assert.equal(unauthActions.status(), 401, 'Unauthenticated /api/v1/nex/actions must return 401');
    const unauthMemory = await unauthApi.get(`${apiOrigin}/api/v1/nex/memory`);
    assert.equal(unauthMemory.status(), 401, 'Unauthenticated /api/v1/nex/memory must return 401');

    const sessionRes = await context.request.get(`${apiOrigin}/api/v1/session`);
    assert.equal(sessionRes.status(), 200);
    const sessionBody = await sessionRes.json();
    const csrfToken = sessionBody.csrf_token;
    assert.ok(csrfToken, 'Authenticated session must expose csrf_token');

    const noCsrfRes = await context.request.post(`${apiOrigin}/api/v1/nex/actions/propose`, {
      headers: { 'Content-Type': 'application/json' },
      data: { action_id: 'read_hud_summary' },
    });
    assert.equal(noCsrfRes.status(), 403, 'Missing CSRF token must return 403');

    for (const forbidden of ['shell', 'push', 'paid_fallback']) {
      const forbiddenRes = await context.request.post(`${apiOrigin}/api/v1/nex/actions/propose`, {
        headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': csrfToken },
        data: { action_id: forbidden },
      });
      assert.equal(forbiddenRes.status(), 403, `Forbidden action ${forbidden} must return 403`);
    }

    const delegatePropose = await context.request.post(`${apiOrigin}/api/v1/nex/actions/propose`, {
      headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': csrfToken },
      data: { action_id: 'delegate_agent_task', params: { target: 'agents' } },
    });
    assert.equal(delegatePropose.status(), 201);
    const { receipt: delegateProposed } = await delegatePropose.json();
    assert.equal(delegateProposed.status, 'PROPOSED');

    const delegateConfirm1 = await context.request.post(`${apiOrigin}/api/v1/nex/actions/confirm`, {
      headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': csrfToken },
      data: { receipt_id: delegateProposed.receipt_id, approved: true },
    });
    assert.equal(delegateConfirm1.status(), 200);
    const { receipt: delegateDone1 } = await delegateConfirm1.json();
    assert.equal(delegateDone1.status, 'UNAVAILABLE');

    const delegateConfirm2 = await context.request.post(`${apiOrigin}/api/v1/nex/actions/confirm`, {
      headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': csrfToken },
      data: { receipt_id: delegateProposed.receipt_id, approved: true },
    });
    assert.equal(delegateConfirm2.status(), 200);
    const { receipt: delegateDone2 } = await delegateConfirm2.json();
    assert.equal(delegateDone2.receipt_id, delegateDone1.receipt_id);
    assert.equal(delegateDone2.status, 'UNAVAILABLE');

    // 2. Durable chat & proposal confirm/dismiss in /nex-v2.html
    await page.setViewportSize({ width: 520, height: 680 });
    await page.goto(`${apiOrigin}/nex-v2.html`, { waitUntil: 'networkidle', timeout: 15000 });
    await page.waitForFunction(
      () => Boolean(window.__nexV2PresenceMetrics) && window.__nexV2Metrics?.rigReady === true,
      null,
      { timeout: 10000 },
    );

    const openPanel = async () => {
      await page.locator('[data-nex-ready] button:not([data-wails-draggable])').click();
      await page.waitForSelector('[data-nex-chat]', { timeout: 5000 });
    };

    await openPanel();
    await page.locator('[data-nex-chat-input]').fill('status');
    await page.locator('[data-nex-chat-send]').click();
    await page.waitForFunction(
      () =>
        Array.from(document.querySelectorAll('[data-role="assistant"]')).some((el) =>
          (el.textContent || '').includes('snapshot'),
        ),
      null,
      { timeout: 5000 },
    );

    await page.reload({ waitUntil: 'networkidle', timeout: 15000 });
    await page.waitForFunction(
      () => Boolean(window.__nexV2PresenceMetrics) && window.__nexV2Metrics?.rigReady === true,
      null,
      { timeout: 10000 },
    );
    await openPanel();
    await page.waitForFunction(
      () =>
        Array.from(document.querySelectorAll('[data-role="assistant"]')).some((el) =>
          (el.textContent || '').includes('snapshot'),
        ),
      null,
      { timeout: 5000 },
    );

    await page.locator('[data-nex-chat-input]').fill('abrir agentes');
    await page.locator('[data-nex-chat-send]').click();
    await page.waitForSelector('[data-nex-proposal="nexus.open.agents"]', { timeout: 5000 });
    await page.locator('[data-nex-proposal="nexus.open.agents"] [data-nex-proposal-confirm]').click();
    await page.waitForSelector('[data-nex-proposal="nexus.open.agents"]', {
      state: 'detached',
      timeout: 5000,
    });

    await page.locator('[data-nex-chat-input]').fill('abrir alertas');
    await page.locator('[data-nex-chat-send]').click();
    await page.waitForSelector('[data-nex-proposal="nexus.open.usage"]', { timeout: 5000 });
    await page.locator('[data-nex-proposal="nexus.open.usage"] [data-nex-proposal-dismiss]').click();
    await page.waitForSelector('[data-nex-proposal="nexus.open.usage"]', {
      state: 'detached',
      timeout: 5000,
    });

    // 3. Personal memory (@personal) CRUD & reload persistence
    await page.locator('[data-nex-memory-toggle]').click();
    await page.waitForSelector('[data-nex-memory-panel]', { timeout: 5000 });
    await page.locator('[data-nex-memory-key]').fill('idioma');
    await page.locator('[data-nex-memory-value]').fill('pt-BR');
    await page.locator('[data-nex-memory-save]').click();
    await page.waitForFunction(
      () => {
        const text = document.querySelector('[data-nex-memory-panel]')?.textContent || '';
        return text.includes('idioma:') && text.includes('pt-BR');
      },
      null,
      { timeout: 5000 },
    );

    await page.reload({ waitUntil: 'networkidle', timeout: 15000 });
    await page.waitForFunction(
      () => Boolean(window.__nexV2PresenceMetrics) && window.__nexV2Metrics?.rigReady === true,
      null,
      { timeout: 10000 },
    );
    await openPanel();
    await page.locator('[data-nex-memory-toggle]').click();
    await page.waitForFunction(
      () => (document.querySelector('[data-nex-memory-panel]')?.textContent || '').includes('pt-BR'),
      null,
      { timeout: 5000 },
    );
    await page.locator('[data-nex-memory-delete]').first().click();
    await page.waitForFunction(
      () => document.querySelectorAll('[data-nex-memory-delete]').length === 0,
      null,
      { timeout: 5000 },
    );
    await page.locator('[data-nex-memory-toggle]').click();

    // 4. Audited receipts UI, read_hud_summary execution, and Privacy Mode masking
    await page.locator('[data-nex-receipts-toggle]').click();
    await page.waitForSelector('[data-nex-receipts-panel]', { timeout: 5000 });
    await page.waitForFunction(
      () => {
        const list = document.querySelector('[data-nex-receipts-list]');
        if (!list) return false;
        const hasExecuted = Boolean(list.querySelector('[data-receipt-status="EXECUTED"]'));
        const hasDenied = Boolean(list.querySelector('[data-receipt-status="DENIED"]'));
        const hasUnavailable = Boolean(list.querySelector('[data-receipt-status="UNAVAILABLE"]'));
        return hasExecuted && hasDenied && hasUnavailable;
      },
      null,
      { timeout: 5000 },
    );

    await page.locator('[data-nex-action-read-hud]').click();
    await page.waitForFunction(
      () =>
        Array.from(
          document.querySelectorAll('[data-nex-receipts-list] [data-receipt-status="EXECUTED"]'),
        ).some((el) => (el.textContent || '').includes('read_hud_summary')),
      null,
      { timeout: 5000 },
    );
    await page.screenshot({ path: path.join(outDir, 'p4-p5-assistant-receipts.png') });

    await page.locator('[data-nex-panel-settings]').click();
    await page.waitForSelector('[data-nex-settings]', { timeout: 3000 });
    await page.locator('[data-nex-toggle="privacy-mode"]').click();
    await page.locator('[data-nex-settings-close]').click();

    const maskedReceiptsText = await page.locator('[data-nex-receipts-list]').innerText();
    assert.ok(
      !maskedReceiptsText.includes('navigation approved'),
      'Receipt summary must be masked under Privacy Mode',
    );
    assert.ok(
      !maskedReceiptsText.includes('Agents surface'),
      'Delegation summary must be masked under Privacy Mode',
    );
    assert.ok(maskedReceiptsText.includes('•••'), 'Privacy Mode must display masked placeholder');
    await page.screenshot({ path: path.join(outDir, 'p4-p5-assistant-privacy.png') });

    const receiptCount = await page.locator('[data-nex-receipts-list] [data-nex-receipt]').count();
    assert.ok(receiptCount >= 4, `Expected at least 4 receipts, got ${receiptCount}`);

    const requestFailures = meaningfulFixtureRequestFailures(collectors.requestFailures);
    assert.equal(
      collectors.pageErrors.length,
      0,
      `Page errors: ${collectors.pageErrors.join('; ')}`,
    );
    assert.equal(
      collectors.consoleErrors.length,
      0,
      `Console errors: ${collectors.consoleErrors.join('; ')}`,
    );
    assert.equal(requestFailures.length, 0, `Request failures: ${requestFailures.join('; ')}`);

    const summary = {
      authVerified: true,
      csrfEnforced: true,
      forbiddenBlocked: 3,
      idempotentConfirmVerified: true,
      durableChatVerified: true,
      personalMemoryCrudVerified: true,
      receiptsVerified: receiptCount,
      privacyMaskingVerified: true,
      pageErrors: collectors.pageErrors.length,
      consoleErrors: collectors.consoleErrors.length,
      requestFailures: requestFailures.length,
    };
    writeFileSync(path.join(outDir, 'summary.json'), `${JSON.stringify(summary, null, 2)}\n`);
    console.log('Nex 2.0 P4/P5 Assistant & Actions E2E: PASS', JSON.stringify(summary));
  } catch (error) {
    console.error('Nex 2.0 P4/P5 Assistant & Actions E2E: FAIL:', error);
    console.error('Server output:\n' + getOutput());
    process.exitCode = 1;
  } finally {
    if (unauthApi) await unauthApi.dispose().catch(() => {});
    if (context) await context.close().catch(() => {});
    if (browser) await browser.close().catch(() => {});
    await terminateProcess(server);
  }
}

await main();
