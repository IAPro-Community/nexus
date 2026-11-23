#!/usr/bin/env node
// Golden PTY replay for issue #196: replays the sanitized capture of a real
// 1000-event interactive PTY session through xterm 5.3 in Chromium and proves
// the rendered buffer reconstructs the exact final prompt, with zero console
// errors. Mirror of internal/runtime/terminal_output_golden_test.go.
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { chromium } from 'playwright';

const scriptDirectory = path.dirname(fileURLToPath(import.meta.url));
const webRoot = path.resolve(scriptDirectory, '..');
const repoRoot = path.resolve(webRoot, '..');
const fixturePath = path.join(repoRoot, 'internal/runtime/testdata/pty196_golden.scrubbed.bin');
const xtermPath = path.join(webRoot, 'node_modules/xterm/lib/xterm.js');

const expectedPrompt =
  '4pfrg23w0l50h8wjdh08tjthqmfe696umhun6jdtgtjw1v3k28ozz9ie9on0pmg45c9nhc4w7xolxgvj8l6v8u5xjx7zmcu0g16hlvou8r6vyib0mps07e19cga0nkwfpu4oe0wx2ulf4nn156era4wtrnbkbma';

const raw = readFileSync(fixturePath);
assert.ok(raw.length > 0, `golden fixture must not be empty: ${fixturePath}`);

const browser = await chromium.launch();
const page = await browser.newPage();
const errors = [];
page.on('console', (message) => {
  if (message.type() === 'error') errors.push(message.text());
});
page.on('pageerror', (error) => errors.push(String(error)));

try {
  await page.setContent('<div id="term"></div>');
  await page.addScriptTag({ path: xtermPath });
  await page.evaluate(() => {
    const term = new Terminal({ convertEol: false, cursorBlink: false, rows: 24, cols: 80 });
    term.open(document.getElementById('term'));
    window.__term = term;
    window.__dump = () => {
      const buffer = term.buffer.active;
      const lines = [];
      for (let i = 0; i < buffer.length; i += 1) {
        const line = buffer.getLine(i);
        if (line) lines.push(line.translateToString(true));
      }
      return lines;
    };
  });
  await page.evaluate(async (bytes) => {
    await new Promise((resolve) => window.__term.write(new Uint8Array(bytes), resolve));
  }, Array.from(raw));
  await page.waitForTimeout(500);

  const lines = await page.evaluate(() => window.__dump());
  const report = reconstructPrompt(lines);

  assert.equal(
    report.text,
    expectedPrompt,
    `rendered prompt mismatch: ${JSON.stringify(report)}`,
  );
  assert.equal(report.promptLines, 3, `prompt must render on 3 lines: ${JSON.stringify(report)}`);
  assert.deepEqual(errors, [], `terminal produced console errors: ${JSON.stringify(errors)}`);

  console.log(
    JSON.stringify({
      fixture: 'internal/runtime/testdata/pty196_golden.scrubbed.bin',
      bytes: raw.length,
      prompt_lines: report.promptLines,
      rendered_length: report.text.length,
      exact_match: true,
      console_errors: errors.length,
    }),
  );
} finally {
  await browser.close();
}

function reconstructPrompt(lines) {
  const promptIndexes = [];
  lines.forEach((line, index) => {
    if (line.startsWith('> ')) promptIndexes.push(index);
  });
  if (promptIndexes.length === 0) return { text: '', promptLines: 0 };
  const start = promptIndexes[promptIndexes.length - 1];
  let text = lines[start].slice(2).trim();
  let promptLines = 1;
  for (let i = start + 1; i < lines.length; i += 1) {
    if (!/^\s+\S/.test(lines[i])) break;
    text += lines[i].trim();
    promptLines += 1;
  }
  return { text, promptLines };
}
