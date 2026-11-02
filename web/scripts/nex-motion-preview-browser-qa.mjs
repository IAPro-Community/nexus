import assert from 'node:assert/strict';
import { existsSync, mkdirSync, writeFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { pathToFileURL } from 'node:url';
import { chromium } from 'playwright';

const web = resolve(import.meta.dirname, '..');
const preview = resolve(web, 'dist/nex-motion-preview.html');
const output = resolve(web, 'dist/nex-motion-preview-qa');
if (!existsSync(preview)) throw new Error('Build the preview with npm run art:nex:preview first.');
mkdirSync(output, { recursive: true });
const ids = [
  'idle_hover',
  'idle_blink',
  'idle_glance',
  'idle_orbit_breathe',
  'idle_head_tilt',
  'idle_fin_twitch',
  'idle_module_inspect',
  'idle_orbit_shuffle',
  'idle_curiosity',
  'idle_micro_stretch',
  'idle_drowse',
  'idle_wake',
];
const targets = {
  idle_hover: '.guardianSprite',
  idle_blink: '.guardianEyes',
  idle_glance: '.guardianEyes',
  idle_orbit_breathe: '.nexModuleNear',
  idle_head_tilt: '[data-nex-part="head-assembly"]',
  idle_fin_twitch: '.nexFinLeft',
  idle_module_inspect: '[data-nex-module="1"] .nexModuleNear',
  idle_orbit_shuffle: '[data-nex-module="1"] .nexModuleNear',
  idle_curiosity: '.guardianEyes',
  idle_micro_stretch: '.guardianBody',
  idle_drowse: '.guardianEyes',
  idle_wake: '.guardianEyes',
};
const sampleMs = {
  idle_hover: 600,
  idle_blink: 200,
  idle_glance: 500,
  idle_orbit_breathe: 850,
  idle_head_tilt: 720,
  idle_fin_twitch: 380,
  idle_module_inspect: 700,
  idle_orbit_shuffle: 700,
  idle_curiosity: 700,
  idle_micro_stretch: 700,
  idle_drowse: 1500,
  idle_wake: 400,
};
const browser = await chromium.launch({ headless: true, args: ['--no-sandbox'] });
const evidence = {
  status: 'NOT_RUN',
  platform: process.platform,
  node: process.version,
  chromium: browser.version(),
  viewports: [390, 768, 1280],
  actions: [],
  moods: [],
  reduced_motion: 'NOT_RUN',
  javascript_errors: [],
};
try {
  const page = await browser.newPage({ viewport: { width: 1280, height: 900 } });
  const errors = [];
  page.on('pageerror', (error) => errors.push(String(error)));
  await page.goto(pathToFileURL(preview).href);
  assert.equal(await page.locator('[data-idle]').count(), 12, 'all 12 action controls exist');
  assert.equal(await page.locator('[data-select]').count(), 5, 'all five moods exist');
  for (const id of ids) {
    await page.locator(`[data-idle="${id}"]`).click();
    const sprite = page.locator('[data-panel="idle"] .guardian svg');
    assert.equal(
      await sprite.getAttribute('data-idle-action'),
      id,
      `${id} is selected on the canonical sprite`,
    );
    const animation = await page
      .locator(`[data-panel="idle"] ${targets[id]}`)
      .first()
      .evaluate((node) => getComputedStyle(node).animationName);
    assert.notEqual(animation, 'none', `${id} runs a real CSS keyframe`);
    const target = page.locator(`[data-panel="idle"] ${targets[id]}`).first();
    const initialTransform = await target.evaluate((node) => getComputedStyle(node).transform);
    await page.waitForTimeout(sampleMs[id]);
    const activeTransform = await target.evaluate((node) => getComputedStyle(node).transform);
    assert.notEqual(activeTransform, initialTransform, `${id} changes rendered transform over time`);
    const screenshot = `${id}.png`;
    await page.screenshot({ path: resolve(output, screenshot) });
    evidence.actions.push({ id, keyframes: animation, screenshot });
  }
  for (const mood of ['working', 'warning', 'critical', 'offline']) {
    await page.locator(`[data-select="${mood}"]`).click();
    assert.equal(
      await page.locator(`[data-panel="${mood}"] .guardian`).getAttribute('data-mood'),
      mood,
    );
    assert.equal(
      await page.locator(`[data-panel="${mood}"] svg`).getAttribute('data-idle-action'),
      null,
    );
    if (mood === 'warning' || mood === 'critical') {
      const reactions = await page.locator(`[data-panel="${mood}"] .guardianSprite`).evaluate((node) =>
        getComputedStyle(node).animationIterationCount,
      );
      assert.equal(reactions, '1', `${mood} gives one short reaction, then rests`);
    }
    evidence.moods.push({ id: mood, preempts_idle: true });
  }
  await page.locator('[data-select="idle"]').click();
  evidence.moods.push({ id: 'idle', preempts_idle: false });
  await page.locator('[data-idle="idle_blink"]').click();
  await page.locator('#qa-reduced').check();
  assert.equal(await page.locator('[data-idle="idle_blink"]').isDisabled(), true);
  assert.equal(
    await page.locator('[data-panel="idle"] svg').getAttribute('data-idle-action'),
    null,
  );
  const reducedAnimation = await page
    .locator('[data-panel="idle"] .guardianEyes')
    .evaluate((node) => getComputedStyle(node).animationName);
  assert.equal(reducedAnimation, 'none', 'motion off freezes the idle renderer');
  evidence.reduced_motion = reducedAnimation === 'none' ? 'PASS' : 'FAIL';
  for (const width of [390, 768, 1280]) {
    await page.setViewportSize({ width, height: 900 });
    await page.screenshot({ path: resolve(output, `reduced-${width}.png`) });
  }
  assert.deepEqual(errors, [], 'browser has no JavaScript errors');
  evidence.javascript_errors = errors;
  evidence.status = 'PASS_BROWSER_MECHANICS_ONLY';
  writeFileSync(resolve(output, 'result.json'), `${JSON.stringify(evidence, null, 2)}\n`);
  console.log(
    `NEX motion preview browser QA PASS: ${ids.length} gestures, 5 moods, reduced motion, 3 widths.`,
  );
} finally {
  await browser.close();
}
