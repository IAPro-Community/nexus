# Session continuity

Same-runtime reattachment, native resume, new session, and cross-provider
context handoff are distinct states. `NATIVE_RESUME_UNVERIFIED` is not promoted
to verified solely because a process starts; the target runtime and resume
arguments are checked by the control boundary.

Runtime generations preserve Agent identity and revision lineage. Cross-provider
continuation uses a bounded context capsule rather than a foreign provider
session identifier.

Current evidence: focused unit/state tests. Real PTY follow-handoff and live
provider proof remain unverified.
