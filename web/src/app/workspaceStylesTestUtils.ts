import { readFileSync } from 'node:fs';

/**
 * Read the legacy workspace stylesheet exactly as the import facade orders it.
 * Tests using textual CSS assertions must not inspect workspace-os.css alone:
 * it is now a list of imports rather than the stylesheet rules themselves.
 */
export function readWorkspaceStyles(): string {
  const facade = readFileSync(new URL('./workspace-os.css', import.meta.url), 'utf8');
  const sections = Array.from(
    facade.matchAll(/^@import "\.\/workspace-styles\/([a-z0-9-]+\.css)";$/gm),
    (match) => match[1],
  );
  if (sections.length === 0) {
    throw new Error('Workspace CSS import facade has no sections');
  }

  return sections
    .map((section) =>
      readFileSync(new URL(`./workspace-styles/${section}`, import.meta.url), 'utf8'),
    )
    .join('');
}
