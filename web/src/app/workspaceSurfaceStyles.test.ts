import { describe, expect, it } from 'vitest';
import { readWorkspaceStyles } from './workspaceStylesTestUtils';

const styles = readWorkspaceStyles();

describe('workspace surface layering and responsive resource layout', () => {
  it('keeps attention surfaces above desktop windows and below modal layers', () => {
    expect(styles).toMatch(/--nx-z-attention:\s*11000;/);
    expect(styles).toMatch(/--nx-z-drawer:\s*12000;/);
    expect(styles).toMatch(
      /\.nx-attention-radar__panel\s*\{[\s\S]*?z-index:\s*var\(--nx-z-attention\);/,
    );
    expect(styles).toMatch(
      /\.nx-context-drawer\s*\{[\s\S]*?z-index:\s*calc\(var\(--nx-z-drawer\) \+ 1\);/,
    );
    expect(styles).toMatch(/--nx-z-modal-backdrop:\s*13000;/);
    expect(styles).toMatch(/--nx-z-toast:\s*12500;/);
    expect(styles).toMatch(
      /\.nx-attention-toaster-root \[data-sonner-toaster\]\s*\{[\s\S]*?z-index:\s*var\(--nx-z-toast\) !important;/,
    );
    expect(styles).not.toMatch(/z-index:\s*20000/);
  });

  it('renders provider resources in two desktop columns and one mobile column', () => {
    expect(styles).toMatch(
      /\.nx-resource-account-list\s*\{[\s\S]*?grid-template-columns:\s*repeat\(2,\s*minmax\(0,\s*1fr\)\);/,
    );
    expect(styles).toMatch(
      /@media \(max-width:\s*680px\)\s*\{[\s\S]*?\.nx-resource-account-list\s*\{[\s\S]*?grid-template-columns:\s*1fr;/,
    );
  });
});
