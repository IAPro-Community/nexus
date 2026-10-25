import { readFileSync } from 'node:fs';
import { describe, expect, it } from 'vitest';

const source = readFileSync(new URL('./FlowRunSurface.tsx', import.meta.url), 'utf8');
const styles = readFileSync(new URL('./FlowRunSurface.module.scss', import.meta.url), 'utf8');

describe('Flow Run status layout', () => {
  it('keeps labels and values separated in both status summaries', () => {
    expect(source).toContain("import styles from './FlowRunSurface.module.scss'");
    expect(source.match(/className=\{styles\.status\}/g)).toHaveLength(2);
    expect(source.match(/className=\{styles\.statusRow\}/g)?.length).toBeGreaterThanOrEqual(8);
    expect(styles).toMatch(/\.status\s*\{[^}]*display:\s*grid/s);
    expect(styles).toMatch(/\.statusRow\s*\{[^}]*gap:/s);
  });
});
