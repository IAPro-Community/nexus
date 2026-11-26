import React from 'react';
import { renderToStaticMarkup } from 'react-dom/server';
import { describe, expect, it } from 'vitest';
import i18n from '../../i18n';
import type { Project } from '../../types';
import {
  DeleteProjectDialog,
  formatLiveAgentDisplay,
  formatProjectDeleteSummary,
} from './DeleteProjectDialog';

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
  it('formats CLI profile labels, pluralized outcomes, and renders blocked agents cleanly', () => {
    const t = i18n.t.bind(i18n);

    const cliDisplay = formatLiveAgentDisplay(t, {
      id: 'ag-1',
      name: 'Nexus Interactive Lead',
      provider: 'agy',
      profile: 'omega-edu',
      runtime_id: 'agy-06H1JNW2YJ7BMCP4A2J9HTWC04',
      status: 'WORKING',
    });
    expect(cliDisplay.title).toContain('agy:omega-edu');
    expect(cliDisplay.meta).toContain('…J9HTWC04');
    expect(cliDisplay.meta).toContain('WORKING');
    expect(cliDisplay.fullRuntimeId).toBe('agy-06H1JNW2YJ7BMCP4A2J9HTWC04');

    const customDisplay = formatLiveAgentDisplay(t, {
      id: 'ag-2',
      name: 'QA Reviewer',
      provider: 'codex',
      profile: 'work',
      runtime_id: 'rt-123',
    });
    expect(customDisplay.title).toBe('QA Reviewer');
    expect(customDisplay.meta).toContain('codex:work');

    expect(
      formatProjectDeleteSummary(t, 'Nexus Workspace', {
        status: 'deleted',
        project_id: 'prj-123',
        mode: 'clean',
      }),
    ).toContain('Nexus Workspace');

    const detachedMany = formatProjectDeleteSummary(t, 'Nexus Workspace', {
      status: 'deleted',
      project_id: 'prj-123',
      mode: 'detach',
      detached_agents: ['AGY Lead (agy:omega-edu)', 'Codex (codex:work)'],
    });
    expect(detachedMany).toContain('2');
    expect(detachedMany).toContain('sessões ativas preservadas');

    const stoppedOne = formatProjectDeleteSummary(t, 'Nexus Workspace', {
      status: 'deleted',
      project_id: 'prj-123',
      mode: 'stop_agents',
      stopped_agents: ['AGY Lead (agy:omega-edu)'],
    });
    expect(stoppedOne).toContain('1');
    expect(stoppedOne).toContain('agente encerrado');

    const markup = renderToStaticMarkup(
      <DeleteProjectDialog
        project={SAMPLE_PROJECT}
        onClose={() => {}}
        initialBlockedAgents={[
          {
            id: 'ag-1',
            name: 'Nexus Interactive Lead',
            provider: 'agy',
            profile: 'omega-edu',
            runtime_id: 'rt-agy-1',
            status: 'WORKING',
          },
        ]}
      />,
    );
    expect(typeof markup).toBe('string');
  });
});
