/**
 * Compatibility facade kept at its historical path.
 *
 * The declarations moved to `src/typeDefs/*` (one module per domain) and the
 * barrel re-exports them, so every `from '../types'` import keeps resolving to
 * the same public names. New code may import from a domain module directly.
 */
export * from './typeDefs';
