#!/usr/bin/env node
import { spawn, spawnSync } from 'node:child_process';
import assert from 'node:assert/strict';
import { existsSync, mkdirSync, rmSync, writeFileSync } from 'node:fs';
import path from 'node:path';
import { tmpdir } from 'node:os';
import { fileURLToPath } from 'node:url';
import { chromium } from 'playwright';
import AxeBuilder from '@axe-core/playwright';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);
const webDir = path.resolve(__dirname, '..');
const repoRoot = path.resolve(webDir, '..');
const binPath = path.resolve(repoRoot, 'nexus');
const screenshotDir = path.resolve(repoRoot, '.tempmediaStorage');

if (!existsSync(screenshotDir)) {
  mkdirSync(screenshotDir, { recursive: true });
}

console.log('=== Nexus E2E Hardening & Responsiveness Verification ===');
console.log('1. Rebuilding nexus binary from the current checkout...');
const buildRes = spawnSync('make', ['build'], { cwd: repoRoot, stdio: 'inherit' });
assert.equal(buildRes.status, 0, 'make build must succeed before browser assertions');
assert.ok(existsSync(binPath), 'current-checkout nexus binary must exist after build');

async function terminateProcess(proc) {
  if (!proc || !proc.pid) return;
  return new Promise((resolve) => {
    try {
      if (process.platform === 'win32') {
        spawn('taskkill', ['/pid', String(proc.pid), '/T', '/F']);
        resolve();
      } else {
        process.kill(-proc.pid, 'SIGTERM');
        const timeout = setTimeout(() => {
          try {
            process.kill(-proc.pid, 'SIGKILL');
          } catch (_) {}
          resolve();
        }, 1200);
        proc.on('exit', () => {
          clearTimeout(timeout);
          resolve();
        });
      }
    } catch (_) {
      resolve();
    }
  });
}

async function main() {
  console.log('2. Starting Nexus Web Server on ephemeral port (--port 0)...');
  const isolatedDataDir =
    process.env.NEXUS_E2E_DATA_DIR || path.join(tmpdir(), `nexus-browser-e2e-${process.pid}`);
  const isolatedProjectPath =
    process.env.NEXUS_E2E_PROJECT_PATH || path.join(isolatedDataDir, 'project');
  if (!process.env.NEXUS_E2E_DATA_DIR) rmSync(isolatedDataDir, { recursive: true, force: true });
  mkdirSync(isolatedProjectPath, { recursive: true });
  if (!existsSync(path.join(isolatedProjectPath, '.git'))) {
    const gitInit = spawnSync('git', ['init', '-q', isolatedProjectPath], { encoding: 'utf8' });
    assert.equal(gitInit.status, 0, `browser E2E fixture git init failed: ${gitInit.stderr || ''}`);
  }
  const serverEnv = {
    ...process.env,
    NEXUS_DATA_DIR: isolatedDataDir,
    AI_MANAGER_DATA_DIR: isolatedDataDir,
    NEXUS_E2E_PROJECT_PATH: isolatedProjectPath,
  };
  let serverOutput = '';
  const server = spawn(binPath, ['web', '--port', '0', '--listen', '127.0.0.1', '--no-open'], {
    cwd: repoRoot,
    detached: true,
    stdio: ['ignore', 'pipe', 'pipe'],
    env: serverEnv,
  });

  let bootstrapUrl = '';
  let urlResolved = false;
  const portPromise = new Promise((resolve, reject) => {
    const timer = setTimeout(() => {
      clearInterval(poll);
      reject(new Error('Timed out waiting for server bootstrap URL (12s)'));
    }, 12000);

    // The CLI intentionally prints only the public URL. The one-time bootstrap
    // token is persisted in the local listen state and exposed through the
    // authenticated `nexus web url` command, so tests must resolve it there
    // instead of requiring the server to echo secrets to stdout.
    const resolveFromListenState = () => {
      const result = spawnSync(binPath, ['web', 'url'], {
        cwd: repoRoot,
        encoding: 'utf8',
        timeout: 1000,
        env: serverEnv,
      });
      const candidate = String(result.stdout || '').trim();
      if (!/#nexus_bootstrap=[a-f0-9]+$/i.test(candidate)) return false;
      clearTimeout(timer);
      clearInterval(poll);
      bootstrapUrl = candidate;
      urlResolved = true;
      resolve(candidate);
      return true;
    };

    const poll = setInterval(() => {
      if (!urlResolved) resolveFromListenState();
    }, 100);

    let output = '';
    server.stdout.on('data', (chunk) => {
      output += chunk.toString();
      serverOutput += chunk.toString();
      const match = output.match(/Bootstrap:\s*(http:\/\/127\.0\.0\.1:\d+\/\?token=[a-f0-9]+)/i);
      if (match && !urlResolved) {
        // Do NOT resolve from stdout alone — it lacks the #nexus_bootstrap
        // hash fragment that initSession() needs. Wait for `nexus web url`
        // to provide the full URL with the bootstrap token.
        console.log(`  (stdout bootstrap URL detected, waiting for nexus web url for hash token...)`);
      }
    });

    server.stderr.on('data', (chunk) => {
      output += chunk.toString();
      serverOutput += chunk.toString();
    });

    server.on('exit', (code) => {
      clearTimeout(timer);
      clearInterval(poll);
      reject(new Error(`Server exited prematurely with code ${code}. Output:\n${output}`));
    });
  });

  let browser;
  let context;
  let page;
  const consoleErrors = [];
  const requestFailures = [];
  const responses = [];
  const pageErrors = [];

  try {
    await portPromise;
    console.log(`Server bound successfully! Bootstrap URL: ${bootstrapUrl}`);

    console.log('3. Launching headless browser...');
    browser = await chromium.launch({
      executablePath: existsSync('/usr/bin/google-chrome') ? '/usr/bin/google-chrome' : undefined,
      headless: true,
      args: ['--no-sandbox', '--disable-setuid-sandbox', '--disable-dev-shm-usage'],
    });

    const viewports = [
      { width: 320, height: 568, name: 'mobile-se' },
      { width: 390, height: 844, name: 'mobile-iphone' },
      { width: 768, height: 1024, name: 'tablet' },
      { width: 1024, height: 768, name: 'laptop-compact' },
      { width: 1280, height: 800, name: 'laptop-critical' },
      { width: 1440, height: 900, name: 'desktop' },
    ];

    context = await browser.newContext();
    page = await context.newPage();
    // Deterministic clean onboarding/layout state (no timeout-driven residue).
    await context.addInitScript(() => {
      try {
        window.localStorage.clear();
        window.sessionStorage.clear();
      } catch (_) {
        /* ignore quota/security errors in headless */
      }
    });
    page.on('pageerror', (error) => pageErrors.push(String(error?.message || error)));
    page.on('console', (message) => {
      if (message.type() === 'error') consoleErrors.push(message.text());
    });
    page.on('requestfailed', (request) => {
      requestFailures.push(
        `${request.method()} ${request.url()} → ${request.failure()?.errorText || 'unknown'}`,
      );
    });
    page.on('response', (response) => {
      if (response.url().includes('/api/'))
        responses.push(`${response.status()} ${response.url()}`);
    });

    console.log(`Navigating to ${bootstrapUrl}...`);
    const mainResponse = await page.goto(bootstrapUrl, {
      waitUntil: 'domcontentloaded',
      timeout: 15000,
    });
    assert.ok(mainResponse, 'bootstrap navigation must return a main response');
    assert.ok(mainResponse.ok(), `bootstrap navigation returned HTTP ${mainResponse.status()}`);
    await page.waitForFunction(
      () => document.readyState === 'interactive' || document.readyState === 'complete',
    );
    // A clean data directory legitimately opens the global Project Hub. Seed
    // one project through the authenticated API, then exercise the real shell.
    const apiOrigin = new URL(bootstrapUrl).origin;
    const projectsResponse = await context.request.get(`${apiOrigin}/api/v1/projects`);
    assert.equal(projectsResponse.status(), 200, 'project list must be available after bootstrap');
    let projects = await projectsResponse.json();
    if (!Array.isArray(projects) || projects.length === 0) {
      const sessionResponse = await context.request.get(`${apiOrigin}/api/v1/session`);
      assert.equal(sessionResponse.status(), 200, 'session must be available after bootstrap');
      const session = await sessionResponse.json();
      assert.ok(session.csrf_token, 'browser E2E fixture needs a CSRF token');
      const createResponse = await context.request.post(`${apiOrigin}/api/v1/projects`, {
        headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': session.csrf_token },
        data: { name: 'Browser E2E Fixture', path: isolatedProjectPath },
      });
      assert.equal(createResponse.status(), 201, 'browser E2E fixture project must be created');
      projects = [await createResponse.json()];
    }
    const project = projects[0];
    assert.ok(project?.id, 'browser E2E fixture project must have an id');
    const shellUrl = new URL(`/p/${encodeURIComponent(project.id)}/overview`, apiOrigin);
    await page.goto(shellUrl.toString(), { waitUntil: 'domcontentloaded', timeout: 15000 });
    await page.waitForSelector('.nx-os-shell', { timeout: 10000 });

    console.log('3.1. Testing Overview → Terminal tab persistence...');
    const overviewProductTab = page.locator('.nx-workspace-tab[data-kind="overview"]').first();
    const terminalsProductTab = page.locator('.nx-workspace-tab[data-kind="terminals"]').first();
    await overviewProductTab.waitFor({ state: 'visible', timeout: 10000 });
    await terminalsProductTab.waitFor({ state: 'visible', timeout: 10000 });
    await overviewProductTab.click();
    await page.waitForFunction(
      () =>
        document
          .querySelector('.nx-workspace-tab[data-kind="overview"]')
          ?.getAttribute('aria-selected') === 'true',
    );
    await page.waitForSelector('[aria-labelledby="overview-resume-title"]', { timeout: 10000 });
    assert.ok(
      await page.locator('#overview-resume-title').isVisible(),
      'Overview must expose the Resume-first panel',
    );
    assert.match(
      new URL(page.url()).pathname,
      /\/overview$/,
      'Overview tab must own the overview route',
    );
    await terminalsProductTab.click();
    await page.waitForFunction(
      () =>
        document
          .querySelector('.nx-workspace-tab[data-kind="terminals"]')
          ?.getAttribute('aria-selected') === 'true',
    );
    assert.match(
      new URL(page.url()).pathname,
      /\/terminals$/,
      'Terminal tab must own the terminals route',
    );
    await page.waitForFunction(
      () =>
        document
          .querySelector('.nx-workspace-tab[data-kind="terminals"]')
          ?.getAttribute('aria-selected') === 'true',
      undefined,
      { timeout: 5000 },
    );
    assert.equal(
      await terminalsProductTab.getAttribute('aria-selected'),
      'true',
      'Terminal must remain active after persisted layout synchronization',
    );
    assert.equal(
      pageErrors.some((message) => /dimensions|Viewport\._innerRefresh/i.test(message)),
      false,
      `Terminal tab must not raise xterm viewport errors: ${pageErrors.join(' | ')}`,
    );

    // ANSI foreground/background pairs are emitted by the user's shell inside
    // xterm and are not Nexus-owned UI styling. Keep the terminal in the
    // interaction assertions below, but exclude its arbitrary output from the
    // product chrome contrast audit.
    const axeResults = await new AxeBuilder({ page }).exclude('.xterm').analyze();
    const seriousViolations = axeResults.violations.filter((violation) =>
      ['serious', 'critical'].includes(violation.impact),
    );
    if (seriousViolations.length > 0) console.error(JSON.stringify(seriousViolations, null, 2));
    assert.equal(
      seriousViolations.length,
      0,
      `critical/serious accessibility violations: ${seriousViolations.map((item) => item.id).join(', ')}`,
    );
    const minorViolations = axeResults.violations.filter(
      (violation) => !['serious', 'critical'].includes(violation.impact),
    );
    if (minorViolations.length > 0) {
      console.warn(
        `  ⚠ Axe non-blocking findings: ${minorViolations
          .map(
            (item) =>
              `${item.id} (${item.impact || 'unknown'}) ${item.nodes
                .map((node) => node.target.join(' '))
                .join(', ')}`,
          )
          .join(', ')}`,
      );
    }
    console.log(
      `  ✓ Axe accessibility scan passed (${axeResults.violations.length} minor violations)`,
    );
    console.log('4. Testing semantic global deep-links...');
    const routeUrl = (pathname) => {
      const url = new URL(pathname, bootstrapUrl);
      // Bootstrap authentication is carried by the one-time URL token in this
      // headless smoke test. Preserve it while exercising deep links.
      url.search = new URL(bootstrapUrl).search;
      return url;
    };
    const updatesUrl = routeUrl('/updates');
    await page.goto(updatesUrl.toString(), { waitUntil: 'domcontentloaded', timeout: 15000 });
    await page.locator('.nx-settings-tabs').waitFor({ state: 'visible', timeout: 10000 });
    assert.equal(
      new URL(page.url()).pathname,
      '/updates',
      'updates deep-link must remain canonical',
    );
    const welcomeUrl = routeUrl('/welcome');
    await page.goto(welcomeUrl.toString(), { waitUntil: 'domcontentloaded', timeout: 15000 });
    await page
      .locator('.nx-tour-layer, .nx-welcome-modal')
      .first()
      .waitFor({ state: 'visible', timeout: 10000 });
    assert.equal(
      new URL(page.url()).pathname,
      '/welcome',
      'welcome deep-link must remain canonical',
    );
    await page.goBack({ waitUntil: 'domcontentloaded' });
    assert.equal(
      new URL(page.url()).pathname,
      '/updates',
      'browser back must restore previous semantic route',
    );
    await page.goForward({ waitUntil: 'domcontentloaded' });
    assert.equal(
      new URL(page.url()).pathname,
      '/welcome',
      'browser forward must restore deep-link',
    );
    await page.goto(bootstrapUrl, { waitUntil: 'domcontentloaded', timeout: 15000 });
    await page.waitForSelector('.nx-os-shell', { timeout: 10000 });

    console.log('5. Testing Breakpoints and Create Menu Button...');
    for (const vp of viewports) {
      await page.setViewportSize({ width: vp.width, height: vp.height });
      // Wait for shell to remain mounted after viewport change (deterministic).
      await page.waitForSelector('.nx-os-shell', { state: 'visible', timeout: 5000 });

      const createBtn = page.locator('[data-testid="topbar-create-menu-btn"]').first();
      await createBtn.waitFor({ state: 'visible', timeout: 5000 });
      assert.equal(
        await createBtn.isVisible(),
        true,
        `Create menu button must be visible at ${vp.width}x${vp.height}`,
      );

      const box = await createBtn.boundingBox();
      assert.ok(box, `Create menu button must have bounding box at ${vp.width}px`);
      assert.ok(
        box.width >= 30,
        `Create menu button width (${box.width}) must be >= 30 at ${vp.width}px`,
      );
      assert.ok(
        box.height >= 26,
        `Create menu button height (${box.height}) must be >= 26 at ${vp.width}px`,
      );

      // Verifica ausência física de sobreposição (elementFromPoint)
      const elementAtPoint = await page.evaluate(
        ({ x, y }) => {
          const el = document.elementFromPoint(x, y);
          const describe = (node) =>
            node
              ? {
                  tag: node.tagName,
                  className: typeof node.className === 'string' ? node.className : null,
                  testId: node.getAttribute('data-testid'),
                  id: node.id || null,
                  rect: (() => {
                    const rect = node.getBoundingClientRect();
                    return { x: rect.x, y: rect.y, width: rect.width, height: rect.height };
                  })(),
                  zIndex: getComputedStyle(node).zIndex,
                  pointerEvents: getComputedStyle(node).pointerEvents,
                  position: getComputedStyle(node).position,
                }
              : null;
          const button = document.querySelector('[data-testid="topbar-create-menu-btn"]');
          const layers = document.elementsFromPoint(x, y);
          return {
            // Chromium may report a positioned ancestor as the hit-test target
            // for a static descendant. Reject only unrelated overlay siblings.
            matches: Boolean(
              button &&
                layers.every(
                  (layer) => layer === button || layer.contains(button) || button.contains(layer),
                ),
            ),
            button: describe(document.querySelector('[data-testid="topbar-create-menu-btn"]')),
            ancestors: (() => {
              const out = [];
              let node = document.querySelector('[data-testid="topbar-create-menu-btn"]');
              while (node && out.length < 6) {
                const style = getComputedStyle(node);
                const rect = node.getBoundingClientRect();
                out.push({
                  tag: node.tagName,
                  className: typeof node.className === 'string' ? node.className : null,
                  rect: { x: rect.x, y: rect.y, width: rect.width, height: rect.height },
                  pointerEvents: style.pointerEvents,
                  transform: style.transform,
                  opacity: style.opacity,
                  visibility: style.visibility,
                });
                node = node.parentElement;
              }
              return out;
            })(),
            topmost: describe(el),
            layers: layers.slice(0, 8).map(describe),
            point: { x, y },
            scroll: { x: window.scrollX, y: window.scrollY },
          };
        },
        { x: box.x + box.width / 2, y: box.y + box.height / 2 },
      );
      assert.equal(
        elementAtPoint.matches,
        true,
        `Create menu button must be topmost element at ${vp.width}px: ${JSON.stringify(elementAtPoint)}`,
      );

      const shotPath = path.join(screenshotDir, `e2e_bp_${vp.width}.png`);
      await page.screenshot({ path: shotPath });
      console.log(
        `  ✓ Breakpoint ${vp.width}x${vp.height} verified (Zero obstruction, shot: ${shotPath})`,
      );
    }

    console.log('6. Testing Settings Surface, Accordion WAI-ARIA and Density Delta...');
    await page.setViewportSize({ width: 1280, height: 800 });
    const settingsBtn = page
      .getByRole('button', { name: /apar[eê]ncia|appearance|settings/i })
      .first();
    await settingsBtn.waitFor({ state: 'visible', timeout: 5000 });
    await settingsBtn.click();

    const settingsTab = page.locator('.nx-workspace-tab[data-kind="settings"]').first();
    await settingsTab.waitFor({ state: 'visible', timeout: 5000 });
    await settingsTab.click();
    await page.waitForFunction(
      () =>
        document
          .querySelector('.nx-workspace-tab[data-kind="settings"]')
          ?.getAttribute('aria-selected') === 'true',
    );

    const settingsTabs = page.locator('.nx-settings-tabs [role="tab"]');
    assert.ok((await settingsTabs.count()) >= 5, 'Settings must expose all five semantic tabs');
    for (let i = 0; i < (await settingsTabs.count()); i += 1) {
      const tab = settingsTabs.nth(i);
      assert.ok(await tab.getAttribute('aria-controls'), `settings tab ${i} must control a panel`);
      assert.ok(await tab.getAttribute('id'), `settings tab ${i} must have a stable id`);
    }
    await settingsTabs.first().focus();
    assert.equal(
      await page.evaluate(() => document.activeElement?.getAttribute('role')),
      'tab',
      'keyboard focus must land on a settings tab',
    );
    await page.keyboard.press('ArrowRight');
    assert.equal(
      await page.evaluate(() => document.activeElement?.getAttribute('role')),
      'tab',
      'arrow navigation must remain within settings tabs',
    );
    await settingsTabs.first().click();
    await page.waitForFunction(
      () =>
        document.querySelector('.nx-settings-tabs [role="tab"]')?.getAttribute('aria-selected') ===
        'true',
    );

    // Validação do Accordion
    const accordionHeader = page.locator('button[id^="theme-cat-hdr-"]').first();
    await accordionHeader.waitFor({ state: 'visible', timeout: 15000 });
    const ariaExpanded = await accordionHeader.getAttribute('aria-expanded');
    assert.ok(
      ariaExpanded === 'true' || ariaExpanded === 'false',
      'aria-expanded must be boolean string',
    );

    // Validação dos Radios com Swatches
    const themeRadio = page.locator('div[role="radio"]').first();
    await themeRadio.waitFor({ state: 'visible', timeout: 5000 });
    assert.equal(await themeRadio.isVisible(), true, 'Theme radio must be visible');
    const hasSwatches = await themeRadio
      .locator('.nx-theme-palette, span[title^="Fundo:"]')
      .count();
    assert.ok(hasSwatches > 0, 'Theme radio must display color palette swatches');

    // Validação Quantitativa de Densidade (Compact vs Comfortable)
    const card = page.locator('.nx-settings-card').first();
    const densityCompBtn = page
      .locator('button:has-text("Compacta"), button:has-text("Compact")')
      .first();
    const densityComfBtn = page
      .locator('button:has-text("Confortável"), button:has-text("Comfortable")')
      .first();

    await settingsTabs.nth(1).click();
    await densityComfBtn.waitFor({ state: 'visible', timeout: 5000 });
    await densityCompBtn.waitFor({ state: 'visible', timeout: 5000 });
    await densityComfBtn.click();
    await page.waitForFunction(() => document.documentElement.dataset.density === 'comfortable');
    const boxComf = await card.boundingBox();
    await densityCompBtn.click();
    await page.waitForFunction(() => document.documentElement.dataset.density === 'compact');
    const boxComp = await card.boundingBox();
    assert.ok(boxComf && boxComp, 'density cards must have measurable bounds');
    console.log(
      `  Measured Card Height - Comfortable: ${boxComf.height.toFixed(1)}px, Compact: ${boxComp.height.toFixed(1)}px`,
    );
    assert.ok(
      boxComf.height > boxComp.height,
      'Comfortable card height must be strictly greater than Compact',
    );

    const settingsShot = path.join(screenshotDir, 'e2e_settings_accordion.png');
    await page.screenshot({ path: settingsShot });
    console.log(`  ✓ Settings & Accordion verified (shot: ${settingsShot})`);

    console.log('✓ All E2E Hardening assertions PASSED successfully!');
  } catch (error) {
    const diagnostics = {
      url: page?.url() || bootstrapUrl,
      title: page ? await page.title().catch(() => '') : '',
      pageErrors,
      consoleErrors,
      requestFailures,
      responses,
      serverOutput: serverOutput.replace(/token=[a-f0-9]+/gi, 'token=[REDACTED]'),
      storageKeys: page
        ? await page
            .evaluate(() => ({
              local: Object.keys(localStorage),
              session: Object.keys(sessionStorage),
            }))
            .catch(() => ({ local: [], session: [] }))
        : { local: [], session: [] },
      cookies: page
        ? await page
            .context()
            .cookies()
            .then((items) =>
              items.map(({ name, domain, path, secure, httpOnly, sameSite }) => ({
                name,
                domain,
                path,
                secure,
                httpOnly,
                sameSite,
              })),
            )
            .catch(() => [])
        : [],
      bodyExcerpt: page
        ? await page
            .locator('body')
            .innerText({ timeout: 1000 })
            .then((value) => value.slice(0, 2000))
            .catch(() => '')
        : '',
      error: String(error?.stack || error),
    };
    writeFileSync(
      path.join(screenshotDir, 'e2e-failure-diagnostics.json'),
      `${JSON.stringify(diagnostics, null, 2)}\n`,
    );
    if (page)
      await page
        .screenshot({ path: path.join(screenshotDir, 'e2e-failure.png'), fullPage: true })
        .catch(() => {});
    throw error;
  } finally {
    if (context) await context.close().catch(() => {});
    if (browser) await browser.close().catch(() => {});
    console.log('6. Terminating test server...');
    await terminateProcess(server);
  }
}

main().catch((err) => {
  console.error('E2E Verification FAILED:', err);
  process.exit(1);
});
