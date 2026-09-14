import type { ProjectCodeIdentity, ProjectFact } from '../../types';
import { asArray } from '../../lib/safeArray';

export const MAX_VISIBLE_INTELLIGENCE_FACTS = 8;

export type PresentedIntelligenceFact = {
  id: string;
  labelKind: 'raw' | 'modules' | 'categoryGroup';
  label: string;
  category?: string;
  count?: number;
  fact: ProjectFact;
};

function isModuleFact(fact: ProjectFact): boolean {
  return fact.category === 'architecture' && (fact.key || '').startsWith('module.');
}

function isPathLikeKey(key: string): boolean {
  return key.includes('/') || /\.(md|ya?ml|json|go|ts|tsx|html)$/i.test(key);
}

function summaryFact(category: string, key: string, facts: ProjectFact[]): ProjectFact {
  return {
    category,
    key,
    value_type: 'LIST',
    value: facts.map((item) => item.key.replace(/^module\./, '')),
    basis: 'DERIVED',
    confidence: 'MEDIUM',
    observed_at: facts[0]?.observed_at || '',
    provenance: [],
  };
}

export function formatProjectIdentity(identity?: ProjectCodeIdentity | null): string {
  if (!identity) return '—';
  const branch = (identity.branch || '').trim();
  const sha = (identity.head_sha || '').trim();
  const shortSha = sha.length > 12 ? sha.slice(0, 12) : sha;
  if (branch && shortSha) return `${branch} · ${shortSha}`;
  if (shortSha) return shortSha;
  const digest = (identity.identity_digest || '').trim();
  if (!digest) return '—';
  return digest.length > 16 ? `${digest.slice(0, 16)}…` : digest;
}

export function presentProjectIntelligenceFacts(facts: ProjectFact[] | null | undefined): {
  rows: PresentedIntelligenceFact[];
  hiddenCount: number;
} {
  const list = asArray<ProjectFact>(facts);
  const buckets = new Map<string, ProjectFact[]>();
  for (const fact of list) {
    const bucket = isModuleFact(fact)
      ? 'architecture:modules'
      : isPathLikeKey(fact.key || '')
        ? `${fact.category}:paths`
        : `${fact.category}:${fact.key}`;
    const current = buckets.get(bucket) || [];
    current.push(fact);
    buckets.set(bucket, current);
  }
  const semantic: PresentedIntelligenceFact[] = [];
  const grouped: PresentedIntelligenceFact[] = [];
  for (const [id, items] of buckets) {
    if (id === 'architecture:modules') {
      grouped.push({
        id,
        labelKind: 'modules',
        label: `modules × ${items.length}`,
        count: items.length,
        fact:
          items.length === 1
            ? { ...items[0], key: items[0].key.replace(/^module\./, '') }
            : summaryFact('architecture', 'modules', items),
      });
      continue;
    }
    if (id.endsWith(':paths')) {
      grouped.push({
        id,
        labelKind: 'categoryGroup',
        label: `${items[0].category} × ${items.length}`,
        category: items[0].category,
        count: items.length,
        fact: summaryFact(items[0].category, 'sources', items),
      });
      continue;
    }
    semantic.push({
      id,
      labelKind: 'raw',
      label: items[0].key,
      fact: items[0],
    });
  }
  const rows = [...semantic, ...grouped];
  if (rows.length <= MAX_VISIBLE_INTELLIGENCE_FACTS) {
    return { rows, hiddenCount: 0 };
  }
  return {
    rows: rows.slice(0, MAX_VISIBLE_INTELLIGENCE_FACTS),
    hiddenCount: rows.length - MAX_VISIBLE_INTELLIGENCE_FACTS,
  };
}
