import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import { fileURLToPath } from 'node:url';
import { chromium } from 'playwright';

const html = await readFile(fileURLToPath(new URL('./index.html', import.meta.url)), 'utf8');
const browser = await chromium.launch({ headless: true });
const pageErrors = [];
try {
  const page = await browser.newPage({ viewport: { width: 1365, height: 900 } });
  page.on('pageerror', (error) => pageErrors.push(error.message));
  await page.setContent(html);
  assert.equal(await page.locator('#state-list button[data-state]').count(), 9);
  assert.equal(await page.locator('#stage').getAttribute('data-state'), 'idle');

  await page.locator('button[data-state="pointing"]').click();
  assert.equal(await page.locator('#stage').getAttribute('data-state'), 'pointing');
  await page.locator('#canvas').focus();
  await page.keyboard.press('Enter');
  assert.equal(await page.locator('#stage').getAttribute('data-state'), 'curious');

  await page.locator('button[data-state="idle"]').click();
  const bounds = await page.locator('#canvas').boundingBox();
  assert.ok(bounds, 'the Nex canvas should be visible');
  const x = bounds.x + bounds.width * 0.41;
  const y = bounds.y + bounds.height * 0.46;
  await page.mouse.move(x + 50, y - 30);
  await page.mouse.down();
  await page.mouse.move(x + 130, y - 100, { steps: 8 });
  assert.equal(await page.locator('#canvas').getAttribute('data-dragging'), 'true');
  await page.mouse.up();
  assert.equal(await page.locator('#canvas').getAttribute('data-dragging'), 'false');

  await page.locator('#reduce').check();
  await page.locator('button[data-state="sleeping"]').click();
  assert.equal(await page.locator('#stage').getAttribute('data-state'), 'sleeping');
  assert.equal(await page.locator('#fps').innerText(), 'estático');
  await page.locator('#autoplay').click();
  assert.equal(await page.locator('#autoplay').isChecked(), false);
  assert.deepEqual(pageErrors, []);

  const mobile = await browser.newPage({ viewport: { width: 390, height: 844 } });
  await mobile.setContent(html);
  assert.equal(await mobile.evaluate(() => document.documentElement.scrollWidth > window.innerWidth), false);
  console.log('Nex Motion Lab: desktop, interaction, reduced motion and mobile checks passed');
} finally {
  await browser.close();
}
