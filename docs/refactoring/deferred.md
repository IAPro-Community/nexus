# Deferred / intentionally unchanged

The consolidation deliberately does not include:

- a major Web/Desktop visual redesign;
- new provider integrations or quota scraping;
- a new orchestration engine or a Nexus takeover of Maestro policy;
- a full rewrite of `handlers_nexus.go` or `app.go` without domain-specific
  seams and regression tests;
- a breaking migration from legacy `error` strings to a new error envelope;
- native Windows/macOS execution claims from a Linux workstation;
- automatic commits, pushes, releases, or changes to the pre-existing dirty
  `.omx`, `.superpowers`, `loadtest`, and `DEV/` worktree entries.

These are follow-up work, not hidden acceptance gaps in the route/API
consolidation milestone.
