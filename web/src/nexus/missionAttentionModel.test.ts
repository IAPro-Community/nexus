import { describe, expect, it } from 'vitest';
import { missionAttentionQueryPath, missionAttentionSections } from './missionAttentionModel';
import type { AttentionGroup } from '../types';

describe('mission attention query', () => {
  it('keeps workspace queries global when no filter is selected', () => {
    expect(missionAttentionQueryPath()).toBe('/api/v1/attention');
  });

  it('encodes project and mission filters as an intersection', () => {
    expect(
      missionAttentionQueryPath({
        projectId: 'project / one',
        missionId: 'mission?two',
      }),
    ).toBe('/api/v1/attention?project_id=project+%2F+one&mission_id=mission%3Ftwo');
  });

  it('omits blank filter values', () => {
    expect(missionAttentionQueryPath({ projectId: ' ', missionId: 'mission-1' })).toBe(
      '/api/v1/attention?mission_id=mission-1',
    );
  });

  it('keeps actionable, completed, and failed Mission items in distinct sections', () => {
    const group: AttentionGroup = {
      needs_you: [
        {
          mission_id: 'needs',
          project_id: 'p1',
          state: 'BLOCKED_NEEDS_USER',
          level: 'REQUIRE_USER',
          reason_code: '',
          summary: '',
          question: '',
          impact: '',
          recommended_actions: [],
          age_seconds: 0,
        },
      ],
      completed: [
        {
          mission_id: 'done',
          project_id: 'p1',
          state: 'COMPLETED_VERIFIED',
          level: 'IN_APP',
          reason_code: '',
          summary: '',
          question: '',
          impact: '',
          recommended_actions: [],
          age_seconds: 0,
        },
      ],
      failed: [
        {
          mission_id: 'failed',
          project_id: 'p2',
          state: 'FAILED_NO_PROGRESS',
          level: 'NOTIFY',
          reason_code: '',
          summary: '',
          question: '',
          impact: '',
          recommended_actions: [],
          age_seconds: 0,
        },
      ],
      all_items: [],
      total_needs: 1,
    };

    expect(
      missionAttentionSections(group).map(({ id, items }) => [id, items[0]?.mission_id]),
    ).toEqual([
      ['needs_you', 'needs'],
      ['completed', 'done'],
      ['failed', 'failed'],
    ]);
    expect(
      missionAttentionSections({ needs_you: null, completed: null, failed: null }).map(
        ({ items }) => items,
      ),
    ).toEqual([[], [], []]);
  });
});
