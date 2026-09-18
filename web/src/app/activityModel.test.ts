import { describe, expect, it } from 'vitest';
import { mapDurableActivity } from './activityModel';

describe('mapDurableActivity', () => {
  it('maps durable project events to the activity view contract', () => {
    expect(
      mapDurableActivity({
        id: 'evt-1',
        correlation_id: 'run-1',
        agent_id: 'agent-1',
        project_id: 'project-1',
        kind: 'FLOW_BLOCKED',
        timestamp: '2026-09-05T12:00:00Z',
        summary: 'Flow blocked on verification',
      }),
    ).toMatchObject({
      id: 'evt-1',
      correlation_id: 'run-1',
      type: 'FLOW_BLOCKED',
      runtime_id: 'agent-1',
      summary: 'Flow blocked on verification',
    });
  });
});
