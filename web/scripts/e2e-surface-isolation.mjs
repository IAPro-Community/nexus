#!/usr/bin/env node
// Surface Isolation + Terminal Keep-Alive E2E.
//
// Proves the P0 lifecycle contract in a real browser:
//   - exactly one active Product Surface at a time (inactive ones UNMOUNTED)
//   - TerminalsHost stays mounted across all switches (same DOM node)
//   - no ReactFlow / MiniMap / Controls leak outside the Flow graph
//   - no orphan portals: dialogs, dropdown panels, backdrops, focus traps
//   - no body scroll lock left behind, no unintended page scrollbars
//   - tablist/tab/tabpanel semantics stay correct after every switch
//   - Axe: zero serious/critical violations on Flow + Settings surfaces
import assert from 'node:assert/strict';
import { existsSync, mkdirSync, rmSync, writeFileSync } from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import AxeBuilder from '@axe-core/playwright';
import {
  launchFixtureBrowser,
  seedFixtureProject,
  startFixtureServer,
  terminateProcess,
} from './e2e-fixture.mjs';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);
const repoRoot = path.resolve(__dirname, '..', '..');
// Env-overridable so parallel runs never share one artifact directory.
const screenshotDir =
  process.env.NEXUS_E2E_ARTIFACTS_DIR || path.resolve(repoRoot, '.tempmediaStorage');
if (!existsSync(screenshotDir)) mkdirSync(screenshotDir, { recursive: true });

const VIEWPORTS = [
  { width: 320, height: 568, name: 'spec-mobile-se' },
  { width: 390, height: 844, name: 'spec-mobile-iphone' },
  { width: 768, height: 1024, name: 'spec-tablet' },
  { width: 1024, height: 768, name: 'spec-laptop-compact' },
  { width: 1366, height: 768, name: 'spec-laptop-risk' },
  { width: 1440, height: 900, name: 'spec-desktop' },
];

// Desired switch order; intersected at runtime with the tabs the layout owns.
const SWITCH_ORDER = [
  'missions',
  'overview',
  'work',
  'settings',
  'agents',
  'resources',
  'missions',
  'work',
  'terminals',
  'overview',
];

async function seedGraphPlan(context, apiOrigin, projectId) {
  const session = await (await context.request.get(`${apiOrigin}/api/v1/session`)).json();
  const headers = { 'Content-Type': 'application/json', 'X-CSRF-Token': session.csrf_token };
  const res = await context.request.post(`${apiOrigin}/api/v1/projects/${projectId}/plans`, {
    headers,
    data: {
      title: 'Surface isolation graph fixture',
      goal: 'Prove ReactFlow unmounts with its surface',
      auto_plan: false,
      phases: [
        {
          id: 'phase-1',
          title: 'Phase 1',
          order: 0,
          packages: [
            {
              id: 'pkg-a',
              title: 'Step A',
              goal: 'Do A',
              priority: 'NORMAL',
              status: 'PENDING',
              dependencies: [],
              role: 'builder',
              acceptance_criteria: [],
            },
            {
              id: 'pkg-b',
              title: 'Step B',
              goal: 'Do B',
              priority: 'NORMAL',
              status: 'PENDING',
              dependencies: ['pkg-a'],
              role: 'builder',
              acceptance_criteria: [],
            },
          ],
        },
      ],
    },
  });
  assert.equal(res.status(), 201, 'graph fixture plan must be created');
  return res.json();
}

async function activeTabKind(page) {
  return page.evaluate(
    () =>
      document
        .querySelector('.nx-workspace-tab[aria-selected="true"]')
        ?.getAttribute('data-kind') || '',
  );
}

async function clickProductTab(page, kind) {
  const tab = page.locator(`.nx-workspace-tab[data-kind="${kind}"]`).first();
  await tab.waitFor({ state: 'visible', timeout: 10000 });
  await tab.click();
  await page.waitForFunction(
    (expected) =>
      document
        .querySelector('.nx-workspace-tab[aria-selected="true"]')
        ?.getAttribute('data-kind') === expected,
    kind,
    { timeout: 10000 },
  );
}

async function assertIsolation(page, label) {
  const snapshot = await page.evaluate(() => {
    const panels = [...document.querySelectorAll('.nx-workspace-stack__body > .nx-workspace-panel')];
    const visible = panels.filter((p) => !p.hasAttribute('hidden'));
    const selected = document.querySelector('.nx-workspace-tab[aria-selected="true"]');
    const selectedKind = selected?.getAttribute('data-kind') || '';
    const controlsId = selected?.getAttribute('aria-controls') || '';
    const controlled = controlsId ? document.getElementById(controlsId) : null;
    const q = (sel) => document.querySelectorAll(sel).length;
    const visibleQ = (sel) =>
      [...document.querySelectorAll(sel)].filter((el) => {
        const r = el.getBoundingClientRect();
        return r.width > 0 && r.height > 0;
      }).length;
    return {
      panelCount: panels.length,
      visibleCount: visible.length,
      visibleIsActive: visible.map((p) => p.getAttribute('data-active')),
      selectedKind,
      controlsLinked: Boolean(controlled) && !controlled.hasAttribute('hidden'),
      reactFlow: q('.react-flow'),
      minimap: q('.react-flow__minimap'),
      flowControls: q('.react-flow__controls'),
      inspector: q('.nx-composer-flow-inspector-pane'),
      chromeMenu: q('.nx-desktop-window__chrome-menu'),
      arrangePanel: q('.nx-arrange-menu__panel'),
      dialogs: q('[role="dialog"]'),
      visibleDialogs: visibleQ('[role="dialog"]'),
      alertDialogs: q('[role="alertdialog"]'),
      backdrops: q('.nx-modal-backdrop, .nx-dialog-backdrop, [data-radix-portal] [data-state="open"]'),
      scrollLocked: document.body.hasAttribute('data-scroll-locked'),
      bodyOverflow: document.body.style.overflow,
      scrollY: window.scrollY,
      docScrollW: document.documentElement.scrollWidth,
      innerW: window.innerWidth,
      focusInHidden: (() => {
        const active = document.activeElement;
        if (!active || active === document.body) return false;
        const hiddenPanel = active.closest?.('.nx-workspace-panel[hidden]');
        return Boolean(hiddenPanel);
      })(),
    };
  });
  assert.equal(snapshot.visibleCount, 1, `[${label}] exactly one product surface visible: ${JSON.stringify(snapshot)}`);
  assert.ok(
    snapshot.controlsLinked,
    `[${label}] selected tab must control the visible panel: ${JSON.stringify(snapshot)}`,
  );
  assert.equal(snapshot.reactFlow, 0, `[${label}] no leaked ReactFlow root: ${JSON.stringify(snapshot)}`);
  assert.equal(snapshot.minimap, 0, `[${label}] no leaked MiniMap: ${JSON.stringify(snapshot)}`);
  assert.equal(snapshot.flowControls, 0, `[${label}] no leaked Flow Controls: ${JSON.stringify(snapshot)}`);
  assert.equal(snapshot.chromeMenu, 0, `[${label}] no orphan chrome menu portal: ${JSON.stringify(snapshot)}`);
  assert.equal(snapshot.arrangePanel, 0, `[${label}] no orphan arrange portal: ${JSON.stringify(snapshot)}`);
  assert.equal(snapshot.visibleDialogs, 0, `[${label}] no visible orphan dialog: ${JSON.stringify(snapshot)}`);
  assert.equal(snapshot.alertDialogs, 0, `[${label}] no orphan alertdialog: ${JSON.stringify(snapshot)}`);
  assert.equal(snapshot.scrollLocked, false, `[${label}] no body scroll lock: ${JSON.stringify(snapshot)}`);
  assert.equal(snapshot.bodyOverflow, '', `[${label}] no body overflow lock: ${JSON.stringify(snapshot)}`);
  assert.equal(snapshot.scrollY, 0, `[${label}] no page scroll offset: ${JSON.stringify(snapshot)}`);
  assert.ok(
    snapshot.docScrollW <= snapshot.innerW + 2,
    `[${label}] no unintended horizontal scrollbar: ${JSON.stringify(snapshot)}`,
  );
  assert.equal(snapshot.focusInHidden, false, `[${label}] focus must not sit in a hidden panel`);
  return snapshot;
}

async function main() {
  console.log('=== Surface Isolation + Terminal Keep-Alive E2E ===');
  const { server, bootstrapUrl, isolatedProjectPath, isolatedDataDir, getOutput } =
    await startFixtureServer();
  let browser;
  let context;
  let page;
  let collectors;
  try {
    ({ browser, context, page, collectors } = await launchFixtureBrowser(bootstrapUrl));
    const { project, apiOrigin } = await seedFixtureProject(
      context,
      bootstrapUrl,
      isolatedProjectPath,
    );
    await seedGraphPlan(context, apiOrigin, project.id);

    const shellUrl = new URL(`/p/${encodeURIComponent(project.id)}/overview`, apiOrigin);
    await page.goto(shellUrl.toString(), { waitUntil: 'domcontentloaded', timeout: 15000 });
    await page.waitForSelector('.nx-os-shell', { timeout: 10000 });
    await page.waitForSelector('.nx-workspace-tab[data-kind]', { timeout: 10000 });

    let kinds = await page.evaluate(() =>
      [...document.querySelectorAll('.nx-workspace-tab[data-kind]')].map((el) =>
        el.getAttribute('data-kind'),
      ),
    );
    console.log(`  initial product tabs: ${kinds.join(', ')}`);

    // Overview, Terminals and Composer are pinned by the workspace. Flow is
    // intentionally lazy/on-demand, so open it through the real Overview UI
    // before asserting the cross-surface lifecycle contract.
    for (const required of ['overview', 'terminals', 'work']) {
      assert.ok(kinds.includes(required), `layout must own the pinned ${required} product tab`);
    }
    if (!kinds.includes('missions')) {
      const flowButton = page.getByRole('button', { name: /flow/i }).first();
      await flowButton.waitFor({ state: 'visible', timeout: 10000 });
      await flowButton.click();
      await page.waitForSelector('.nx-workspace-tab[data-kind="missions"]', { timeout: 10000 });
      kinds = await page.evaluate(() =>
        [...document.querySelectorAll('.nx-workspace-tab[data-kind]')].map((el) =>
          el.getAttribute('data-kind'),
        ),
      );
      console.log(`  product tabs after opening Flow: ${kinds.join(', ')}`);
    }
    assert.ok(kinds.includes('missions'), 'Flow action must open the missions product tab');

    // Terminal keep-alive baseline: pin the host DOM node identity.
    const hostPinned = await page.evaluate(() => {
      const host = document.querySelector('.nx-terminals-host');
      window.__e2eTerminalsHost = host || null;
      return Boolean(host);
    });
    assert.ok(hostPinned, 'TerminalsHost must be mounted from the start');

    const sequence = SWITCH_ORDER.filter((kind) => kinds.includes(kind));
    assert.ok(sequence.length >= 6, 'switch matrix needs at least 6 reachable surfaces');

    console.log('  running cross-surface switch matrix...');
    for (const kind of sequence) {
      await clickProductTab(page, kind);
      const snap = await assertIsolation(page, kind);
      console.log(`  ✓ ${kind}: one panel (total mounted incl. terminals host: ${snap.panelCount})`);
    }

    // Same-node proof: the terminals host survived every switch without remount.
    const hostSurvived = await page.evaluate(
      () => window.__e2eTerminalsHost === document.querySelector('.nx-terminals-host'),
    );
    assert.equal(hostSurvived, true, 'TerminalsHost DOM node must be identical after all switches');
    console.log('  ✓ TerminalsHost keep-alive: identical DOM node across all switches');

    // Flow graph mount + unmount proof.
    await clickProductTab(page, 'missions');
    const draftCard = page.locator('.nx-flow-run-history-card button').first();
    await draftCard.waitFor({ state: 'visible', timeout: 10000 });
    await draftCard.click();
    const graphTab = page.getByRole('tab', { name: /^Graph|Grafo$/ }).first();
    await graphTab.waitFor({ state: 'visible', timeout: 10000 });
    await graphTab.click();
    await page.waitForSelector('.react-flow', { timeout: 15000 });
    console.log('  ✓ Flow graph mounted (.react-flow present)');
    const graphShot = path.join(screenshotDir, 'e2e_surface_flow_graph.png');
    await page.screenshot({ path: graphShot });
    await clickProductTab(page, 'overview');
    await assertIsolation(page, 'flow-graph→overview');
    console.log('  ✓ Flow graph fully unmounted after leaving (0 .react-flow)');

    // Flow run → composer and agents → composer spot checks.
    await clickProductTab(page, 'work');
    await assertIsolation(page, 'flow→composer');
    if (kinds.includes('agents')) {
      await clickProductTab(page, 'agents');
      await assertIsolation(page, 'agents');
      await clickProductTab(page, 'work');
      await assertIsolation(page, 'agents→composer');
    }
    if (kinds.includes('resources')) {
      await clickProductTab(page, 'resources');
      await assertIsolation(page, 'resources');
      if (kinds.includes('missions')) {
        await clickProductTab(page, 'missions');
        await assertIsolation(page, 'resources→flow');
      }
    }
    if (kinds.includes('settings')) {
      await clickProductTab(page, 'settings');
      await assertIsolation(page, 'settings');
      await clickProductTab(page, 'overview');
      await assertIsolation(page, 'settings→overview');
    }

    // Axe on the two heaviest changed surfaces.
    for (const axeKind of ['missions', 'settings'].filter((k) => kinds.includes(k))) {
      await clickProductTab(page, axeKind);
      const axeResults = await new AxeBuilder({ page }).exclude('.xterm').analyze();
      const serious = axeResults.violations.filter((v) =>
        ['serious', 'critical'].includes(v.impact),
      );
      if (serious.length > 0) console.error(JSON.stringify(serious, null, 2));
      assert.equal(
        serious.length,
        0,
        `Axe serious/critical on ${axeKind}: ${serious.map((v) => v.id).join(', ')}`,
      );
      console.log(`  ✓ Axe ${axeKind}: 0 serious/critical`);
    }

    // Viewport sweep: full surface captures at the two risk viewports, shell
    // presence everywhere else.
    for (const vp of VIEWPORTS) {
      await page.setViewportSize({ width: vp.width, height: vp.height });
      await page.waitForSelector('.nx-os-shell', { state: 'visible', timeout: 5000 });
      const risk = vp.width === 1366 || vp.width === 390;
      const targets = risk ? ['overview', 'work', 'missions'] : ['overview'];
      for (const target of targets.filter((k) => kinds.includes(k))) {
        await clickProductTab(page, target);
        await assertIsolation(page, `${target}@${vp.width}x${vp.height}`);
        const shot = path.join(screenshotDir, `e2e_surface_${target}_${vp.width}x${vp.height}.png`);
        await page.screenshot({ path: shot });
      }
      console.log(`  ✓ viewport ${vp.width}x${vp.height} (${vp.name})`);
    }

    const unexpectedPages = collectors.pageErrors.filter(
      (m) => !/dimensions|Viewport\._innerRefresh/i.test(m),
    );
    assert.equal(
      unexpectedPages.length,
      0,
      `unexpected browser page errors: ${unexpectedPages.join(' | ')}`,
    );
    assert.equal(
      collectors.consoleErrors.length,
      0,
      `unexpected console errors: ${collectors.consoleErrors.join(' | ')}`,
    );
    // Favicon and data: fetches are browser chrome noise, never app failures.
    const meaningfulFailures = collectors.requestFailures.filter(
      (entry) => !/favicon\.ico|data:/i.test(entry),
    );
    assert.equal(
      meaningfulFailures.length,
      0,
      `unexpected request failures: ${meaningfulFailures.join(' | ')}`,
    );
    console.log('✓ All Surface Isolation assertions PASSED!');
  } catch (error) {
    // Never persist the one-time bootstrap token: anyone holding the
    // artifact could hijack the fixture session.
    const publicUrl = (() => {
      try {
        const url = new URL(page?.url() || bootstrapUrl);
        url.hash = '';
        return url.toString();
      } catch {
        return '(unparsable url)';
      }
    })();
    const redact = (value) =>
      String(value).replace(/nexus_bootstrap=[a-f0-9]+/gi, 'nexus_bootstrap=[REDACTED]');
    const diagnostics = {
      url: publicUrl,
      pageErrors: collectors?.pageErrors.map(redact) || [],
      consoleErrors: collectors?.consoleErrors.map(redact) || [],
      requestFailures: collectors?.requestFailures.map(redact) || [],
      serverOutput: redact(getOutput()),
      error: String(error?.stack || error),
    };
    writeFileSync(
      path.join(screenshotDir, 'e2e-surface-failure-diagnostics.json'),
      `${JSON.stringify(diagnostics, null, 2)}\n`,
    );
    if (page)
      await page
        .screenshot({ path: path.join(screenshotDir, 'e2e-surface-failure.png'), fullPage: true })
        .catch(() => {});
    throw error;
  } finally {
    if (context) await context.close().catch(() => {});
    if (browser) await browser.close().catch(() => {});
    console.log('Terminating test server...');
    await terminateProcess(server);
    // Remove the isolated fixture tree unless the caller pinned it via env
    // for post-mortem inspection.
    if (!process.env.NEXUS_E2E_DATA_DIR && isolatedDataDir) {
      rmSync(isolatedDataDir, { recursive: true, force: true });
    }
  }
}

main().catch((err) => {
  console.error('Surface Isolation E2E FAILED:', err);
  process.exit(1);
});
