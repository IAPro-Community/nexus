#!/usr/bin/env node
// Nexus terminal split CSS regression: proves that the legacy absolute
// positioning from Tabs does not overlap two grid-managed PTY panels.
// Browser-only style smoke. Real PTY/ConPTY E2E remains a separate gate.
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { chromium } from 'playwright';

const webRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const shared = readFileSync(path.join(webRoot, 'src/app/workspace-styles/03-workspace-layout.css'), 'utf8');
const split = readFileSync(path.join(webRoot, 'src/workspace/TerminalSplit.module.scss'), 'utf8');

const browser = await chromium.launch({
  headless: true,
  executablePath: process.env.PLAYWRIGHT_EXECUTABLE_PATH || undefined,
  args: ['--no-sandbox', '--disable-dev-shm-usage'],
});

try {
  const page = await browser.newPage({ viewport: { width: 1440, height: 900 } });
  for (const { orientation, width, height, ratio } of [
    { orientation: 'vertical', width: 960, height: 480, ratio: 20 },
    { orientation: 'vertical', width: 960, height: 480, ratio: 50 },
    { orientation: 'vertical', width: 960, height: 480, ratio: 80 },
    { orientation: 'horizontal', width: 480, height: 720, ratio: 20 },
    { orientation: 'horizontal', width: 480, height: 720, ratio: 50 },
    { orientation: 'horizontal', width: 480, height: 720, ratio: 80 },
    { orientation: 'horizontal', width: 360, height: 640, ratio: 50 },
  ]) {
    await page.setContent(`
      <style>
        * { box-sizing: border-box }
        ${shared}
        ${split}
      </style>
      <div class="splitPanels" data-orientation="${orientation}"
           style="width:${width}px; height:${height}px; --nx-terminal-split-ratio: ${ratio}%">
        <div class="nx-workspace-panel splitPane" data-active="true"
             style="order:0">Primary PTY</div>
        <div class="nx-workspace-panel splitPane" data-active="true"
             style="order:2">Secondary PTY</div>
        <div class="nx-workspace-panel splitPane" data-active="false" hidden>Hidden PTY</div>
        <div class="divider" role="separator" style="order:1"></div>
      </div>
    `);
    const geometry = await page.evaluate(() => {
      const panes = [...document.querySelectorAll('.splitPane')];
      const separator = document.querySelector('.divider');
      const box = (element) => {
        const rect = element.getBoundingClientRect();
        return { x: rect.x, y: rect.y, width: rect.width, height: rect.height };
      };
      return {
        primary: box(panes[0]),
        secondary: box(panes[1]),
        divider: box(separator),
        primaryPosition: getComputedStyle(panes[0]).position,
        secondaryPosition: getComputedStyle(panes[1]).position,
        hiddenDisplay: getComputedStyle(panes[2]).display,
      };
    });
    const { primary, secondary, divider } = geometry;
    assert.equal(geometry.primaryPosition, 'relative', 'primary cannot inherit absolute Tabs positioning');
    assert.equal(geometry.secondaryPosition, 'relative', 'secondary cannot inherit absolute Tabs positioning');
    assert.equal(geometry.hiddenDisplay, 'none', 'third PTY cannot occupy a grid track');
    if (orientation === 'vertical') {
      assert.ok(primary.width > 50 && secondary.width > 50, 'both PTYs need usable widths');
      assert.ok(primary.x + primary.width <= divider.x + 1, 'primary cannot overlap the divider');
      assert.ok(divider.x + divider.width <= secondary.x + 1, 'secondary cannot overlap the divider');
      assert.ok(Math.abs(primary.y - secondary.y) <= 1, 'panes must sit in the same grid row');
    } else {
      assert.ok(primary.height > 50 && secondary.height > 50, 'both PTYs need usable heights');
      assert.ok(primary.y + primary.height <= divider.y + 1, 'primary cannot overlap the divider');
      assert.ok(divider.y + divider.height <= secondary.y + 1, 'secondary cannot overlap the divider');
      assert.ok(Math.abs(primary.x - secondary.x) <= 1, 'panes must sit in the same grid column');
    }
    console.log(`PASS terminal split ${orientation} ${ratio}% (${width}x${height})`);
  }
  await page.close();
} finally {
  await browser.close();
}
