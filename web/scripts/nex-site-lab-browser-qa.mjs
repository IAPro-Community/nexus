import assert from 'node:assert/strict';
import { mkdirSync, writeFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { chromium } from 'playwright';

const urlArgument = process.argv.find((arg) => arg.startsWith('--url='));
const url = urlArgument?.slice('--url='.length)
  ?? process.env.NEX_SITE_LAB_URL
  ?? 'http://127.0.0.1:4330/nex-lab/';
const output = resolve(import.meta.dirname, '../dist/nex-site-lab-qa');
mkdirSync(output, { recursive: true });

const browser = await chromium.launch({ headless: true, args: ['--no-sandbox'] });
const result = {
  status: 'NOT_RUN',
  url,
  chromium: browser.version(),
  viewports: [390, 768, 1280],
  states: [],
  actions: [],
  screenshots: [],
  page_errors: [],
};
try {
  const page = await browser.newPage({ viewport: { width: 390, height: 900 } });
  page.on('pageerror', (error) => result.page_errors.push(String(error)));
  await page.goto(url, { waitUntil: 'networkidle', timeout: 30_000 });
  assert.equal(await page.locator('meta[name="robots"]').getAttribute('content'), 'noindex,nofollow');
  assert.equal(await page.locator('iframe').count(), 1, 'verified candidate iframe is visible');
  const frame = page.frameLocator('iframe');
  const states = frame.locator('#states button');
  const actions = frame.locator('#idles button');
  assert.equal(await states.count(), 5, 'five operational mood controls exist');
  assert.equal(await actions.count(), 12, 'twelve candidate idle controls exist');

  for (const mood of ['idle', 'working', 'warning', 'critical', 'offline']) {
    await frame.locator(`#states button[data-action-value="${mood}"]`).click();
    assert.equal(await frame.locator('#nex').getAttribute('data-state'), mood);
    if (mood !== 'idle') assert.equal(await frame.locator('#nex').getAttribute('data-action'), '');
    result.states.push(mood);
  }

  await frame.locator('#states button[data-action-value="idle"]').click();
  await frame.locator('#idles button[data-action-value="idle_curiosity"]').click();
  const animation = await frame.locator('.eyes').first().evaluate(
    (node) => getComputedStyle(node).animationName,
  );
  assert.notEqual(animation, 'none', 'selected gesture runs in the embedded preview');
  result.actions.push({ id: 'idle_curiosity', keyframe: animation });

  for (const width of result.viewports) {
    await page.setViewportSize({ width, height: 900 });
    const dimensions = await page.evaluate(() => ({
      viewport: document.documentElement.clientWidth,
      content: document.documentElement.scrollWidth,
    }));
    assert.ok(dimensions.content <= dimensions.viewport, `no outer horizontal overflow at ${width}px`);
    const screenshot = `nex-lab-${width}.png`;
    await page.screenshot({ path: resolve(output, screenshot), fullPage: true });
    result.screenshots.push(screenshot);
  }
  assert.deepEqual(result.page_errors, []);
  result.status = 'PASS_SITE_LAB_BROWSER_MECHANICS_ONLY';
  writeFileSync(resolve(output, 'result.json'), `${JSON.stringify(result, null, 2)}\n`);
  console.log(`NEX site lab browser QA PASS: 5 moods, 12 controls, ${result.viewports.join('/')}px.`);
} finally {
  await browser.close();
}
