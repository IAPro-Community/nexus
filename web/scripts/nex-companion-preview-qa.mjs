/**
 * Browser smoke QA for the lightweight 2D companion preview.
 * No network, provider, Core events or window geometry are exercised.
 * Run: cd web && npm run art:nex:companion:qa
 */
import assert from 'node:assert/strict';
import { readFileSync, mkdirSync, writeFileSync } from 'node:fs';
import { resolve, dirname } from 'node:path';
import { fileURLToPath } from 'node:url';
import { chromium } from 'playwright';

const web = resolve(dirname(fileURLToPath(import.meta.url)), '..');
const html = readFileSync(resolve(web, 'dist/nex-companion-preview.html'), 'utf8');
const output = resolve(web, 'dist/nex-companion-qa');
const cases = [
  ['Repouso', 'ambient', 'idle'],
  ['Cumprimento', 'greet', 'idle'],
  ['Olhar', 'look', 'idle'],
  ['Trabalhando', 'thinking', 'working'],
  ['Precisa de você', 'attention', 'warning'],
  ['Falha crítica', 'error', 'critical'],
  ['Missão concluída', 'resolved', 'idle'],
  ['Sonolência', 'rest', 'idle'],
  ['Desconectado', 'still', 'offline'],
];
mkdirSync(output, { recursive: true });
const browser = await chromium.launch({
  headless: true,
  args: ['--no-sandbox', '--disable-dev-shm-usage'],
});
const result = { status: 'NOT_RUN', tested: [], browserErrors: [], nativeHudCertified: false };
try {
  const page = await browser.newPage({ viewport: { width: 390, height: 844 } });
  page.on('pageerror', (error) => result.browserErrors.push(String(error)));
  await page.setContent(html, { waitUntil: 'load' });
  for (const [label, behavior, mood] of cases) {
    await page.getByRole('button', { name: label, exact: true }).click();
    const actualBehavior = await page.locator('#companion').getAttribute('data-nex-behavior');
    const actualMood = await page.locator('#figure').getAttribute('data-mood');
    assert.equal(actualBehavior, behavior, `Behavior: ${label}`);
    assert.equal(actualMood, mood, `Operational display mood: ${label}`);
    await page.screenshot({ path: resolve(output, `${behavior}.png`) });
    result.tested.push(behavior);
  }
  await page.getByLabel('Olhar à esquerda').check();
  await page.getByRole('button', { name: 'Olhar', exact: true }).click();
  assert.equal(await page.locator('#companion').getAttribute('data-nex-look'), 'left');
  await page.getByLabel('Permitir movimento').uncheck();
  assert.equal(await page.locator('#companion').getAttribute('data-nex-behavior'), 'still');
  await page.getByLabel('Permitir movimento').check();
  await page.emulateMedia({ reducedMotion: 'reduce' });
  const frozenAnimation = await page.locator('#figure').evaluate(
    (element) => getComputedStyle(element).animationName,
  );
  assert.equal(frozenAnimation, 'none', 'OS reduced-motion freezes animation');
  assert.equal(result.browserErrors.length, 0, result.browserErrors.join('; '));
  result.status = 'PASS_BROWSER_FIXTURE_ONLY';
} catch (error) {
  result.status = 'FAIL';
  result.error = String(error);
  throw error;
} finally {
  writeFileSync(resolve(output, 'result.json'), JSON.stringify(result, null, 2) + '\n');
  await browser.close();
}
console.log(`NEX Companion browser QA: ${result.status}; ${result.tested.length} behaviors`);
