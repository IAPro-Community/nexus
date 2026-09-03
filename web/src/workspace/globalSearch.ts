/**
 * Global search across projects, agents, provider sessions, and flow runs.
 * Uses fuzzy matching for local search.
 */

export interface SearchResult {
  id: string;
  type: 'project' | 'agent' | 'provider-session' | 'flow-run';
  title: string;
  subtitle?: string;
  projectId?: string;
  action: 'open' | 'focus' | 'adopt';
}

/**
 * Simple fuzzy match that checks if all characters in the query appear in order.
 */
function fuzzyMatch(query: string, text: string): boolean {
  const lowerQuery = query.toLowerCase();
  const lowerText = text.toLowerCase();
  let queryIndex = 0;

  for (let i = 0; i < lowerText.length && queryIndex < lowerQuery.length; i++) {
    if (lowerText[i] === lowerQuery[queryIndex]) {
      queryIndex++;
    }
  }

  return queryIndex === lowerQuery.length;
}

/**
 * Score a fuzzy match (higher is better).
 */
function fuzzyScore(query: string, text: string): number {
  const lowerQuery = query.toLowerCase();
  const lowerText = text.toLowerCase();

  // Exact match
  if (lowerText === lowerQuery) return 100;

  // Starts with query
  if (lowerText.startsWith(lowerQuery)) return 90;

  // Contains query as substring
  if (lowerText.includes(lowerQuery)) return 80;

  // Fuzzy match
  if (!fuzzyMatch(query, text)) return -1;

  // Score based on how early matches occur
  let score = 0;
  let queryIndex = 0;
  for (let i = 0; i < lowerText.length && queryIndex < lowerQuery.length; i++) {
    if (lowerText[i] === lowerQuery[queryIndex]) {
      score += 10 - Math.min(i, 10); // Higher score for earlier matches
      queryIndex++;
    }
  }

  return score;
}

/**
 * Search across all available entities.
 */
export function globalSearch(
  query: string,
  projects: Array<{ id: string; name: string }>,
  agents: Array<{ id: string; title: string; projectId: string }>,
  providerSessions: Array<{ id: string; title: string; projectId: string; provider: string }>,
  flowRuns: Array<{ id: string; title: string; projectId: string }>,
): SearchResult[] {
  if (!query.trim()) return [];

  const results: Array<SearchResult & { score: number }> = [];

  // Search projects
  for (const project of projects) {
    const score = fuzzyScore(query, project.name);
    if (score > 0) {
      results.push({
        id: project.id,
        type: 'project',
        title: project.name,
        action: 'open',
        score,
      });
    }
  }

  // Search agents
  for (const agent of agents) {
    const score = fuzzyScore(query, agent.title);
    if (score > 0) {
      results.push({
        id: agent.id,
        type: 'agent',
        title: agent.title,
        projectId: agent.projectId,
        action: 'focus',
        score,
      });
    }
  }

  // Search provider sessions
  for (const session of providerSessions) {
    const score = fuzzyScore(query, session.title);
    if (score > 0) {
      results.push({
        id: session.id,
        type: 'provider-session',
        title: session.title,
        subtitle: session.provider,
        projectId: session.projectId,
        action: 'adopt',
        score,
      });
    }
  }

  // Search flow runs
  for (const run of flowRuns) {
    const score = fuzzyScore(query, run.title);
    if (score > 0) {
      results.push({
        id: run.id,
        type: 'flow-run',
        title: run.title,
        projectId: run.projectId,
        action: 'focus',
        score,
      });
    }
  }

  // Sort by score descending
  return results
    .sort((a, b) => b.score - a.score)
    .map(({ score: _, ...result }) => result);
}
