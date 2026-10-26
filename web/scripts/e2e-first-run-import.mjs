#!/usr/bin/env node
import assert from 'node:assert/strict';
import { existsSync, mkdirSync, rmSync } from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import {
  launchFixtureBrowser,
  seedFixtureProject,
  startFixtureServer,
  terminateProcess,
} from './e2e-fixture.mjs';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);
const repoRoot = path.resolve(__dirname, '..', '..');
const screenshotDir =
  process.env.NEXUS_E2E_ARTIFACTS_DIR || path.resolve(repoRoot, '.tempmediaStorage', 'first-run');
if (!existsSync(screenshotDir)) mkdirSync(screenshotDir, { recursive: true });

async function main() {
  console.log('=== Nexus first-run host-session import E2E ===');
  const { server, bootstrapUrl, isolatedDataDir, isolatedProjectPath, getOutput } =
    await startFixtureServer();
  let browser;
  let context;
  let page;
  let firstRunImportRequests = 0;
  let usageImportRequests = 0;
  let usageResourceReads = 0;

  try {
    ({ browser, context, page } = await launchFixtureBrowser(bootstrapUrl));
    await page.setViewportSize({ width: 1280, height: 800 });
    await context.route('**/api/v1/providers**', (route) =>
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify([
          {
            id: 'codex',
            installed: true,
            version: 'fixture',
            registration_state: 'PENDING_AUTH',
            control_level: 'FULL',
            capabilities: {},
          },
        ]),
      }),
    );
    await context.route('**/api/v1/system/doctor**', (route) =>
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          schema: 'fixture',
          generated_at: new Date().toISOString(),
          version: 'fixture',
          os: 'linux',
          arch: 'amd64',
          checks: [],
          providers: {},
          credentials: { status: 'UNKNOWN', mechanism: 'fixture' },
        }),
      }),
    );
    await context.route('**/api/v1/profiles/import-host-sessions**', (route) => {
      if (new URL(page.url()).pathname.endsWith('/resources')) usageImportRequests += 1;
      else firstRunImportRequests += 1;
      return route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          results: [
            { provider: 'codex', profile: 'default', status: 'IMPORTED' },
            { provider: 'claude', status: 'NO_SESSION' },
            { provider: 'gemini', status: 'FAILED' },
          ],
        }),
      });
    });
    await context.route('**/api/v1/resources', (route) => {
      usageResourceReads += 1;
      const accounts = usageImportRequests
        ? [
            {
              id: 'codex:default',
              provider: 'codex',
              profile: 'default',
              display_name: 'Codex fixture',
              authenticated: true,
              is_default: true,
              available: true,
              quota_remaining: 0,
              quota_total: 0,
              rate_limited: false,
              health: 'healthy',
              last_checked: new Date().toISOString(),
              quota_view: { status: 'UNKNOWN' },
            },
          ]
        : [];
      return route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ accounts, policy: 'BALANCED' }),
      });
    });

    await page.goto(bootstrapUrl, { waitUntil: 'domcontentloaded', timeout: 15000 });
    await page.locator('[data-testid="project-hub-root"]').waitFor({
      state: 'visible',
      timeout: 10000,
    });
    const brandMark = await page
      .locator('[data-testid="project-hub-root"] .nx-brand-mark--hero')
      .boundingBox();
    assert.ok(
      brandMark && brandMark.y >= 0 && brandMark.y + brandMark.height <= 800,
      'the first-run brand mark must stay within the viewport',
    );
    const importButton = page.getByRole('button', { name: 'Importar sessões existentes' });
    await importButton.waitFor({ state: 'visible', timeout: 10000 });
    await importButton.click();

    const result = page.getByRole('status').filter({ hasText: 'Importação concluída:' });
    await result.waitFor({ state: 'visible', timeout: 5000 });
    const resultText = await result.innerText();
    assert.match(resultText, /1 importadas/);
    assert.match(resultText, /1 sem sessão importável/);
    assert.match(resultText, /1 falharam/);
    assert.equal(firstRunImportRequests, 1, 'first-run import must call the endpoint once');
    await page.screenshot({ path: path.join(screenshotDir, 'first-run-layout.png'), fullPage: true });
    await result.screenshot({ path: path.join(screenshotDir, 'first-run-import-result.png') });

    const { project, apiOrigin } = await seedFixtureProject(
      context,
      bootstrapUrl,
      isolatedProjectPath,
    );
    await page.goto(`${apiOrigin}/p/${encodeURIComponent(project.id)}/resources`, {
      waitUntil: 'domcontentloaded',
      timeout: 15000,
    });
    const usageImportButton = page.getByTestId('usage-import-host-sessions');
    await usageImportButton.waitFor({ state: 'visible', timeout: 15000 });
    await usageImportButton.click();

    const usageResult = page.getByRole('status').filter({ hasText: 'Importação concluída:' });
    await usageResult.waitFor({ state: 'visible', timeout: 5000 });
    await page.getByText('Codex fixture', { exact: true }).waitFor({
      state: 'visible',
      timeout: 5000,
    });
    assert.equal(usageImportRequests, 1, 'Usage / Accounts must call the import endpoint once');
    assert.ok(usageResourceReads >= 2, 'Usage / Accounts must refresh profiles after import');
    await page.screenshot({
      path: path.join(screenshotDir, 'usage-accounts-host-session-import.png'),
      fullPage: true,
    });
    console.log('First-run host-session import feedback E2E: PASS');
    console.log('Usage / Accounts host-session import and profile refresh E2E: PASS');
  } catch (error) {
    console.error(`First-run import E2E server output:\n${getOutput()}`);
    throw error;
  } finally {
    if (context) await context.close().catch(() => {});
    if (browser) await browser.close().catch(() => {});
    await terminateProcess(server);
    if (!process.env.NEXUS_E2E_DATA_DIR && isolatedDataDir) {
      rmSync(isolatedDataDir, { recursive: true, force: true });
    }
  }
}

main().catch((error) => {
  console.error(`First-run host-session import E2E: FAIL: ${error?.stack || error}`);
  process.exit(1);
});
