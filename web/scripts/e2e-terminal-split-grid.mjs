#!/usr/bin/env node
// Nexus terminal split CSS regression: proves that the legacy absolute
// positioning from Tabs does not overlap two grid-managed PTY panels.
// Browser-only style smoke. Real PTY/ConPTY E2E remains a separate gate.
import assert from 'node:assert/strict';
import { existsSync, readFileSync } from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { chromium } from 'playwright';
import { compile, compileString } from 'sass';

const webRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const shared = readFileSync(
  path.join(webRoot, 'src/app/workspace-styles/03-workspace-layout.css'),
  'utf8',
);
const ptyStyles = readFileSync(
  path.join(webRoot, 'src/app/workspace-styles/11-notifications-and-composer.css'),
  'utf8',
);
const typography = readFileSync(
  path.join(webRoot, 'src/app/workspace-styles/13-communication-and-task-ui.css'),
  'utf8',
);
const split = readFileSync(path.join(webRoot, 'src/workspace/TerminalSplit.module.scss'), 'utf8');
const ptyTabModulePath = path.join(webRoot, 'src/workspace/PtyTabs.module.scss');
const ptyTabModule = existsSync(ptyTabModulePath)
  ? compileString(readFileSync(ptyTabModulePath, 'utf8')).css
  : '';
const terminalPaneModule = compile(
  path.join(webRoot, 'src/components/TerminalPane.module.scss'),
).css;

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
    // setContent replaces documentElement; platform selection belongs after it.
    await page.evaluate(() => {
      document.documentElement.dataset.nexusPlatform = 'web';
    });
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
        primaryOverflow: getComputedStyle(panes[0]).overflow,
        secondaryOverflow: getComputedStyle(panes[1]).overflow,
        primaryGutter: getComputedStyle(panes[0]).scrollbarGutter,
        secondaryGutter: getComputedStyle(panes[1]).scrollbarGutter,
      };
    });
    const { primary, secondary, divider } = geometry;
    assert.equal(
      geometry.primaryPosition,
      'relative',
      'primary cannot inherit absolute Tabs positioning',
    );
    assert.equal(
      geometry.secondaryPosition,
      'relative',
      'secondary cannot inherit absolute Tabs positioning',
    );
    assert.equal(geometry.hiddenDisplay, 'none', 'third PTY cannot occupy a grid track');
    assert.equal(
      geometry.primaryOverflow,
      'hidden',
      'web Split primary must not inherit page scrollbars',
    );
    assert.equal(
      geometry.secondaryOverflow,
      'hidden',
      'web Split secondary must not inherit page scrollbars',
    );
    assert.equal(
      geometry.primaryGutter,
      'auto',
      'web Split primary cannot reserve scrollbar gutter',
    );
    assert.equal(
      geometry.secondaryGutter,
      'auto',
      'web Split secondary cannot reserve scrollbar gutter',
    );
    if (orientation === 'vertical') {
      assert.ok(primary.width > 50 && secondary.width > 50, 'both PTYs need usable widths');
      assert.ok(primary.x + primary.width <= divider.x + 1, 'primary cannot overlap the divider');
      assert.ok(
        divider.x + divider.width <= secondary.x + 1,
        'secondary cannot overlap the divider',
      );
      assert.ok(Math.abs(primary.y - secondary.y) <= 1, 'panes must sit in the same grid row');
    } else {
      assert.ok(primary.height > 50 && secondary.height > 50, 'both PTYs need usable heights');
      assert.ok(primary.y + primary.height <= divider.y + 1, 'primary cannot overlap the divider');
      assert.ok(
        divider.y + divider.height <= secondary.y + 1,
        'secondary cannot overlap the divider',
      );
      assert.ok(Math.abs(primary.x - secondary.x) <= 1, 'panes must sit in the same grid column');
    }
    console.log(`PASS terminal split ${orientation} ${ratio}% (${width}x${height})`);
  }

  for (const splitWidth of [1040, 720]) {
    await page.setContent(`
    <style>* { box-sizing: border-box } ${shared} ${ptyStyles} ${typography} ${ptyTabModule}</style>
    <div class="nx-workspace-tabs nx-pty-tabs ptyTabs" role="tablist"
         style="width:560px;height:34px">
      <div class="nx-pty-tab">
        <button class="nx-workspace-tab ptyTabButton" role="tab">
          <span class="nx-workspace-tab__label ptyTabLabel">developer@workstation:/tmp/nexus-linux-100-2026-10-25/split-native/repository/first-long-session</span>
        </button>
      </div>
      <div class="nx-pty-tab">
        <button class="nx-workspace-tab ptyTabButton" role="tab">
          <span class="nx-workspace-tab__label ptyTabLabel">developer@workstation:/tmp/nexus-linux-100-2026-10-25/split-native/repository/second-long-session</span>
        </button>
      </div>
    </div>
  `);
    const tabLayout = await page.evaluate(() =>
      [...document.querySelectorAll('.ptyTabLabel')].map((label) => {
        const tab = label.closest('.nx-pty-tab');
        const labelRect = label.getBoundingClientRect();
        const tabRect = tab.getBoundingClientRect();
        const style = getComputedStyle(label);
        return {
          overflow: style.overflowX,
          textOverflow: style.textOverflow,
          minWidth: style.minWidth,
          clientWidth: label.clientWidth,
          scrollWidth: label.scrollWidth,
          labelRight: labelRect.right,
          tabRight: tabRect.right,
        };
      }),
    );
    assert.equal(tabLayout.length, 2, 'both PTY tabs should remain present');
    for (const tab of tabLayout) {
      assert.equal(
        tab.overflow,
        'hidden',
        'long PTY tab identity must not paint over its neighbor',
      );
      assert.equal(
        tab.textOverflow,
        'ellipsis',
        'long PTY tab identity should be visibly truncated',
      );
      assert.equal(
        tab.minWidth,
        '0px',
        'PTY tab text must be allowed to shrink inside its flex item',
      );
      assert.ok(
        tab.scrollWidth > tab.clientWidth,
        'fixture should exercise a genuinely long identity',
      );
      assert.ok(tab.labelRight <= tab.tabRight + 1, 'PTY tab text must stay inside its own tab');
    }
    console.log('PASS long PTY tab identities remain clipped inside each tab');

    await page.setContent(`
    <style>
      * { box-sizing: border-box }
      ${shared}
      ${split}
      ${terminalPaneModule}
    </style>
    <div class="splitPanels" data-orientation="vertical"
         style="width:${splitWidth}px;height:600px;--nx-terminal-split-ratio:50%">
      ${['left', 'right']
        .map(
          (side) => `
        <div class="splitPane" data-active="true" style="order:${side === 'left' ? 0 : 2}">
          <section class="terminalPaneRoot" data-chrome="full">
            <div class="terminalHeader" data-terminal-header>
              <div class="terminalIdentity" data-terminal-identity>
                <div class="terminalSessionIdentity" data-terminal-session-identity>
                  <span aria-hidden="true">●</span>
                  <button type="button" class="terminalTitleButton" data-terminal-title>
                    <span class="terminalTitleText">Shell</span><span aria-hidden="true">✎</span>
                  </button>
                  <span class="terminalPtyTitle" data-terminal-pty-title>Native Split Fixture</span>
                </div>
                <div class="terminalRuntimeMetadata" data-terminal-runtime-metadata>
                  <span aria-hidden="true">|</span><span>shell</span><span>(local)</span>
                  <span aria-hidden="true">|</span>
                  <span class="terminalRuntimeId" data-terminal-runtime-id
                        title="ID: shell-06GQ3W8JJ05ZQRSSCAKSG9YSH0">
                    ID: shell-06GQ3W8JJ05ZQRSSCAKSG9YSH0
                  </span>
                </div>
              </div>
              <div class="terminalActions" data-terminal-actions>
                <button type="button" aria-label="Voice input">Voice</button>
                <div class="zoomGroup" role="group" aria-label="Terminal font size">
                  <button type="button">−</button><button type="button">13px</button><button type="button">+</button>
                </div>
                <span data-role="CONTROL">CONTROL</span>
                <button type="button">Release</button>
              </div>
            </div>
            <div class="terminalContainer"></div>
          </section>
        </div>
      `,
        )
        .join('')}
      <div class="divider" role="separator" style="order:1"></div>
    </div>
  `);
    const compactHeaders = await page.evaluate(() =>
      [...document.querySelectorAll('[data-terminal-header]')].map((header) => {
        const identity = header.querySelector('[data-terminal-identity]');
        const actions = header.querySelector('[data-terminal-actions]');
        const runtimeId = header.querySelector('[data-terminal-runtime-id]');
        const headerRect = header.getBoundingClientRect();
        const rect = (element) => {
          const value = element.getBoundingClientRect();
          return { left: value.left, right: value.right, top: value.top, bottom: value.bottom };
        };
        return {
          headerLeft: headerRect.left,
          headerWidth: header.clientWidth,
          headerScrollWidth: header.scrollWidth,
          identity: rect(identity),
          actions: rect(actions),
          runtimeIdWidth: runtimeId.clientWidth,
          runtimeIdScrollWidth: runtimeId.scrollWidth,
          runtimeIdOverflow: getComputedStyle(runtimeId).overflowX,
          runtimeIdEllipsis: getComputedStyle(runtimeId).textOverflow,
          runtimeIdWhiteSpace: getComputedStyle(runtimeId).whiteSpace,
        };
      }),
    );
    assert.equal(compactHeaders.length, 2, 'both split PTYs must render a toolbar');
    for (const header of compactHeaders) {
      assert.ok(
        header.headerScrollWidth <= header.headerWidth + 1,
        `split toolbar content must stay within the pane (${header.headerScrollWidth}px > ${header.headerWidth}px)`,
      );
      const headerRight = header.headerLeft + header.headerWidth;
      assert.ok(header.identity.right <= headerRight + 1, 'identity must not overpaint the pane');
      assert.ok(header.actions.right <= headerRight + 1, 'controls must not overpaint the pane');
      assert.ok(
        header.identity.bottom <= header.actions.top + 1 ||
          header.actions.bottom <= header.identity.top + 1,
        'identity and controls must use separate rows when their combined width cannot fit',
      );
      assert.ok(
        header.runtimeIdScrollWidth > header.runtimeIdWidth,
        'fixture must test a long runtime id',
      );
      assert.equal(
        header.runtimeIdOverflow,
        'hidden',
        'runtime id must be clipped in a narrow pane',
      );
      assert.equal(header.runtimeIdEllipsis, 'ellipsis', 'runtime id must end in an ellipsis');
      assert.equal(header.runtimeIdWhiteSpace, 'nowrap', 'runtime id must remain on one line');
    }
    console.log(
      `PASS compact split terminal toolbars at ${splitWidth / 2}px per PTY keep identity and controls inside each pane`,
    );
  }
  await page.close();
} finally {
  await browser.close();
}
