import { describe, expect, it } from 'vitest';
import {
  missionAttentionQueryPath,
  missionAttentionSections,
  missionAttentionSessionKey,
} from './missionAttentionModel';
import type { AttentionGroup, AttentionItem } from '../types';

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

  it('binds a mobile job to its mission, task and intervention revision', () => {
    const item: AttentionItem = {
      mission_id: 'mission-1',
      project_id: 'project-1',
      state: 'BLOCKED_NEEDS_USER',
      level: 'REQUIRE_USER',
      reason_code: 'APPROVAL_REQUIRED',
      summary: 'Approve deploy',
      question: 'Approve?',
      impact: 'release',
      recommended_actions: [],
      age_seconds: 3,
      intervention: {
        id: 'intervention-1',
        reason_code: 'APPROVAL_REQUIRED',
        summary: 'Approve deploy',
        question: 'Approve?',
        context: '',
        recommended_actions: [],
        impact: 'release',
        mission_id: 'mission-1',
        task_id: 'package-2',
        source: 'NEXUS',
        timestamp: '2026-11-23T00:00:00Z',
        version: 3,
        scope: 'project-1',
        options: [],
        resolved: false,
      },
    };
    const reconnect = { ...item };
    expect(missionAttentionSessionKey(item)).toBe(missionAttentionSessionKey(reconnect));
    expect(missionAttentionSessionKey(item)).not.toBe(
      missionAttentionSessionKey({ ...item, mission_id: 'mission-2' }),
    );
    expect(missionAttentionSessionKey(item)).not.toBe(
      missionAttentionSessionKey({
        ...item,
        intervention: { ...item.intervention!, version: 4 },
      }),
    );
  });
});
