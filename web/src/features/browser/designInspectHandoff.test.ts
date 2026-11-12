import { describe, expect, it, vi } from 'vitest';
import type { Agent } from '../../types';
import { handoffDesignInspectionToSelectedAgent } from './designInspectHandoff';
import type { NexusBrowserDesignInspection } from './BrowserSurface';

const agents: Agent[] = [
  {
    id: 'agent-alpha',
    project_id: 'project-a',
    name: 'Alpha',
    role: 'Lead',
    status: 'RUNNING',
    continuity_status: '',
    created_at: '',
    updated_at: '',
  },
  {
    id: 'agent-beta',
    project_id: 'project-a',
    name: 'Beta',
    role: 'Builder',
    status: 'RUNNING',
    continuity_status: '',
    created_at: '',
    updated_at: '',
  },
  {
    id: 'agent-other-project',
    project_id: 'project-b',
    name: 'Other',
    role: 'Lead',
    status: 'RUNNING',
    continuity_status: '',
    created_at: '',
    updated_at: '',
  },
];

const inspection: NexusBrowserDesignInspection = {
  selector: '#hero-title',
  tag: 'h1',
  role: 'heading',
  name: 'Hero title',
  text: 'Welcome',
  attributes: { id: 'hero-title' },
  computed_styles: { color: 'rgb(1, 2, 3)' },
  masking_applied: true,
};

describe('design inspection Agent handoff', () => {
  it('sends only to the explicitly selected Agent and never starts a runtime', async () => {
    const askAgent = vi.fn().mockResolvedValue({ accepted: true });
    await handoffDesignInspectionToSelectedAgent({
      agents,
      projectId: 'project-a',
      selectedAgentId: 'agent-beta',
      inspection,
      askAgent,
    });
    expect(askAgent).toHaveBeenCalledTimes(1);
    expect(askAgent).toHaveBeenCalledWith(
      'agent-beta',
      expect.stringContaining('Selector: #hero-title'),
      false,
    );
    expect(askAgent).not.toHaveBeenCalledWith('agent-alpha', expect.anything(), expect.anything());
  });

  it('requires explicit selection and rejects an Agent outside the selected project', async () => {
    const askAgent = vi.fn().mockResolvedValue({ accepted: true });
    await expect(
      handoffDesignInspectionToSelectedAgent({
        agents,
        projectId: 'project-a',
        selectedAgentId: '',
        inspection,
        askAgent,
      }),
    ).rejects.toThrow(/select an agent/i);
    await expect(
      handoffDesignInspectionToSelectedAgent({
        agents,
        projectId: 'project-a',
        selectedAgentId: 'agent-other-project',
        inspection,
        askAgent,
      }),
    ).rejects.toThrow(/does not belong/i);
    expect(askAgent).not.toHaveBeenCalled();
  });

  it('fails closed when crop masking was not applied', async () => {
    const askAgent = vi.fn().mockResolvedValue({ accepted: true });
    await expect(
      handoffDesignInspectionToSelectedAgent({
        agents,
        projectId: 'project-a',
        selectedAgentId: 'agent-alpha',
        inspection: { ...inspection, masking_applied: false },
        askAgent,
      }),
    ).rejects.toThrow(/did not apply content masks/i);
    expect(askAgent).not.toHaveBeenCalled();
  });
});
