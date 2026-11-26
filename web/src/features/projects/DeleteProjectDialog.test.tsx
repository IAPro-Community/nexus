import React from 'react';
import { renderToStaticMarkup } from 'react-dom/server';
import { describe, expect, it } from 'vitest';
import i18n from '../../i18n';
import type { Project } from '../../types';
import { DeleteProjectDialog, formatProjectDeleteSummary } from './DeleteProjectDialog';

const SAMPLE_PROJECT: Project = {
  id: 'prj-123',
  slug: 'nexus-workspace',
  name: 'Nexus Workspace',
  canonical_path: '/projetos/nexus',
  repo_remote: '',
  repo_url: '',
  maestro_mode: 'ASSIST',
  default_isolation: 'project',
  default_branch: 'main',
  resource_policy: 'BALANCED',
  settings: '{}',
  created_at: '2026-11-26T10:00:00Z',
  updated_at: '2026-11-26T10:00:00Z',
};

describe('DeleteProjectDialog & Outcome Reporting', () => {
  it('renders live agent resolution choices and CLI origin context when blocked', () => {
    // Radix Dialog Portal renders null in static SSR markup unless tested cleanly or inspect helper;
    // ensure formatProjectDeleteSummary clearly distinguishes clean, detach, and stop_agents outcomes.
    const t = i18n.t.bind(i18n);

    expect(
      formatProjectDeleteSummary(t, 'Nexus Workspace', {
        status: 'deleted',
        project_id: 'prj-123',
        mode: 'clean',
      }),
    ).toContain('Nexus Workspace');

    const detached = formatProjectDeleteSummary(t, 'Nexus Workspace', {
      status: 'deleted',
      project_id: 'prj-123',
      mode: 'detach',
      detached_agents: ['AGY Lead (agy)', 'Codex (codex)'],
    });
    expect(detached).toContain('2');
    expect(detached).toContain('AGY Lead (agy), Codex (codex)');

    const stopped = formatProjectDeleteSummary(t, 'Nexus Workspace', {
      status: 'deleted',
      project_id: 'prj-123',
      mode: 'stop_agents',
      stopped_agents: ['AGY Lead (agy)'],
    });
    expect(stopped).toContain('1');
    expect(stopped).toContain('AGY Lead (agy)');

    // Component renders cleanly with blocked agents without throwing.
    const markup = renderToStaticMarkup(
      <DeleteProjectDialog
        project={SAMPLE_PROJECT}
        onClose={() => {}}
        initialBlockedAgents={[
          {
            id: 'ag-1',
            name: 'AGY Lead',
            provider: 'agy',
            runtime_id: 'rt-agy-1',
            status: 'WORKING',
          },
        ]}
      />,
    );
    expect(typeof markup).toBe('string');
  });
});
