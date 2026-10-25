import assert from 'node:assert/strict';
import { mkdir } from 'node:fs/promises';
import path from 'node:path';
import { chromium } from 'playwright';

const siteUrl = process.env.DOCS_SITE_URL || 'http://127.0.0.1:3029/nexus/';
const browser = await chromium.launch({
  headless: true,
  executablePath: process.env.PLAYWRIGHT_EXECUTABLE_PATH || '/usr/bin/google-chrome',
});

try {
  const page = await browser.newPage();
  const pageErrors = [];
  page.on('pageerror', (error) => pageErrors.push(error.message));

  await page.goto(siteUrl, { waitUntil: 'networkidle' });
  const searchInput = page.locator('input[aria-label="Search"]');
  await searchInput.waitFor({ state: 'visible' });

  const searchIndexResponse = page.waitForResponse(
    (response) => response.url().includes('search-index') && response.url().includes('.json'),
    { timeout: 8000 },
  );
  await searchInput.click();
  const indexResponse = await searchIndexResponse;
  assert.equal(indexResponse.status(), 200, 'search index must be served successfully');
  assert.match(
    indexResponse.headers()['content-type'] || '',
    /application\/json/i,
    'search index must have a JSON content type',
  );

  await searchInput.fill('terminal');
  const terminalResult = page.locator('a[href*="/product/terminals"]').first();
  await terminalResult.waitFor({ state: 'visible', timeout: 8000 });
  assert.match(await terminalResult.innerText(), /terminais/i);
  assert.deepEqual(pageErrors, [], 'search must not produce browser page errors');

  if (process.env.DOCS_SEARCH_SCREENSHOT) {
    await mkdir(path.dirname(process.env.DOCS_SEARCH_SCREENSHOT), { recursive: true });
    await page.screenshot({ path: process.env.DOCS_SEARCH_SCREENSHOT, fullPage: true });
  }

  console.log(`PASS docs search index and terminal result: ${siteUrl}`);
} finally {
  await browser.close();
}
