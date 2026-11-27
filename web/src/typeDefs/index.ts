/**
 * Public type barrel for the whole application.
 *
 * Domain modules live in one file each so a change touches a small surface;
 * every existing `import type { … } from '../types'` keeps working unchanged
 * because `web/src/types.ts` re-exports this barrel.
 *
 * - core:      runtime/project/agent identities shared by every domain
 * - agents:    agent detail, configuration and execution roles
 * - missions:  missions, runs, evidence, coordination and attention
 * - usage:     provider accounts, capacity, cost and capabilities
 * - composer:  Composer sessions, intelligence, routing and Flow planning
 * - maestro:   Maestro catalog and submission contracts
 * - workspace: filesystem, git, release desk, context and Agent projection
 */
export * from './core';
export * from './agents';
export * from './missions';
export * from './usage';
export * from './composer';
export * from './maestro';
export * from './workspace';
