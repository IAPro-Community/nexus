import { asArray } from '../lib/safeArray';
import type { AttentionItem } from '../types';

export interface MissionAttentionFilter {
  projectId?: string;
  missionId?: string;
}

export interface MissionAttentionSection {
  id: 'needs_you' | 'completed' | 'failed';
  items: AttentionItem[];
}

type AttentionGroupLists = {
  needs_you?: AttentionItem[] | null;
  completed?: AttentionItem[] | null;
  failed?: AttentionItem[] | null;
};

export function missionAttentionQueryPath(filter?: MissionAttentionFilter): string {
  const params = new URLSearchParams();
  const projectId = filter?.projectId?.trim();
  const missionId = filter?.missionId?.trim();
  if (projectId) params.set('project_id', projectId);
  if (missionId) params.set('mission_id', missionId);
  const query = params.toString();
  return query ? `/api/v1/attention?${query}` : '/api/v1/attention';
}

export function missionAttentionSections(
  group?: AttentionGroupLists | null,
): MissionAttentionSection[] {
  return [
    { id: 'needs_you', items: asArray<AttentionItem>(group?.needs_you) },
    { id: 'completed', items: asArray<AttentionItem>(group?.completed) },
    { id: 'failed', items: asArray<AttentionItem>(group?.failed) },
  ];
}
