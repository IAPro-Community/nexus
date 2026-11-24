import React from 'react';
import { renderToStaticMarkup } from 'react-dom/server';
import { describe, expect, it } from 'vitest';
import '../i18n';
import { WorkspacePresentationProvider } from './WorkspacePresentationProvider';
import { WorkspaceProvider } from './WorkspaceProvider';
import { WorkspaceSnapshotsControl } from './WorkspaceSnapshotsControl';

describe('WorkspaceSnapshotsControl', () => {
  it('exposes the project snapshot action as a localized, scoped control', () => {
    const html = renderToStaticMarkup(
      <WorkspaceProvider projectId="project-alpha">
        <WorkspacePresentationProvider projectId="project-alpha">
          <WorkspaceSnapshotsControl projectId="project-alpha" />
        </WorkspacePresentationProvider>
      </WorkspaceProvider>,
    );

    expect(html).toContain('data-tour="workspace-snapshots"');
    expect(html).toContain('aria-haspopup="dialog"');
    expect(html).toContain('aria-expanded="false"');
    expect(html).toContain('Snapshots');
  });
});
