import { describe, expect, it } from 'vitest';
import type { AttentionItem, ProjectAgentSummary } from '../../types';
import { groupAttentionByProject, summariesByProjectId } from './useProjectRailOutline';
import { formatOpenedAgo } from './projectRailRelativeTime';

function makeItem(overrides: Partial<AttentionItem> = {}): AttentionItem {
  return {
    mission_id: 'run_1',
    project_id: 'proj_a',
    state: 'BLOCKED_NEEDS_USER',
    level: 'REQUIRE_USER',
    reason_code: 'TEST',
    summary: 'Needs decision',
    question: 'Approve?',
    impact: 'Paused',
    recommended_actions: [],
    age_seconds: 10,
    ...overrides,
  };
}

describe('groupAttentionByProject', () => {
  it('groups needs_you and failed by project_id and tolerates null arrays', () => {
    const map = groupAttentionByProject({
      needs_you: [
        makeItem({ mission_id: 'n1', project_id: 'proj_a' }),
        makeItem({ mission_id: 'n2', project_id: 'proj_b' }),
      ],
      failed: [
        makeItem({
          mission_id: 'f1',
          project_id: 'proj_a',
          state: 'FAILED',
          level: 'IN_APP',
        }),
      ],
    });
    expect(map.get('proj_a')?.needsYou).toHaveLength(1);
    expect(map.get('proj_a')?.failed).toHaveLength(1);
    expect(map.get('proj_b')?.needsYou).toHaveLength(1);
    expect(map.get('proj_b')?.failed).toHaveLength(0);

    const empty = groupAttentionByProject({ needs_you: null, failed: null });
    expect(empty.size).toBe(0);
  });
});

describe('summariesByProjectId', () => {
  it('indexes summaries and ignores null payloads', () => {
    const list: ProjectAgentSummary[] = [
      { project_id: 'proj_a', agent_count: 2, working_count: 1 },
      { project_id: 'proj_b', agent_count: 0, working_count: 0 },
    ];
    const map = summariesByProjectId(list);
    expect(map.get('proj_a')?.working_count).toBe(1);
    expect(summariesByProjectId(null).size).toBe(0);
  });
});

describe('formatOpenedAgo', () => {
  it('returns never label for missing timestamps', () => {
    expect(formatOpenedAgo(undefined, 'en', 'Never')).toBe('Never');
  });

  it('formats recent times relatively', () => {
    const recent = new Date(Date.now() - 5 * 60 * 1000).toISOString();
    const label = formatOpenedAgo(recent, 'en', 'Never');
    expect(label.toLowerCase()).toMatch(/minute|min/);
  });
});
