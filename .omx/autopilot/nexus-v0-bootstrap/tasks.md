# Nexus V0 task ledger

## Plan 01 — Core Integrity

- [ ] 01.1 terminal protocol exhaustive and lossless
- [ ] 01.2 Windows terminal/host lifecycle
- [ ] 01.3 filesystem mkdir and database cleanup
- [ ] 01.4 recommendation-backed Direct resource selection
- [ ] 01.5 provider config allowlisting

## Plan 02 — Workspace OS + Direct Work

- [x] 02.1 presentation state model and persistence (initial implementation)
- [x] 02.2 desktop window renderer (initial implementation)
- [x] 02.3 workspace launcher (valid surface routing)
- [x] 02.4 Ask existing Agent (API + UI contract)
- [ ] 02.5 normal Project Shell

Policy ruling: task commits are required by the bootstrap prompt, but automatic commits are forbidden by the global repository contract; keep task boundaries and record intended commit messages without creating commits.
