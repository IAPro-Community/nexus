# Security and isolation — IAPro Nexus

**English** | [Português](security.pt-BR.md) | [Español](security.es.md)

> Local-first does **not** mean sandboxed or incapable of network access. The source release is not automatically certified for every supported platform. See [Security policy](../SECURITY.md).

## Trust boundaries

- Nexus operates coding agents and provider CLIs. Those processes typically run with **the host user's permissions**; Git worktrees and isolated profile directories are not OS sandboxes.
- Each provider/profile has its own credential configuration where supported. A provider can transmit code and prompts under **its own privacy terms**.
- Session bootstrap tokens and cookies are sensitive. Web uses loopback by default; remote access must use an explicit secure deployment and authentication.
- UNKNOWN health/quota is not evidence of safety or availability.
- No default authorization for push, deploy, destructive Git operations or repeating an external action with unknown outcome.

## Runtime and network

Use `nexus security --json` and `nexus doctor --json` for diagnosis. Before changing settings that expose an API or terminal beyond loopback, review [remote access](product/remote-access.md) and [guardrails](architecture/guardrails.md). A tunnel invitation can provide **broad host workspace access**, not a read-only spectator role.

## Logs, credentials and diagnostics

Do not post OAuth secrets, API keys, cookies, session bootstrap links, terminal captures, environment files or personal paths in public issues. Inspect diagnostics locally first, and apply redaction. Provider auth remains controlled by the provider's supported flow; Nexus must not copy arbitrary credentials across account scopes.

## Release security

Source tests, security declarations, dependency licenses, signed update manifests and native signing are **separate** evidence gates. A CI job skipped or cancelled is not PASS. See [release certification](operations/release.md), [platform support](operations/platform-support.md), and the [publication audit](PUBLICATION_CHECKLIST.md).

Report exploitable vulnerabilities privately as described in [SECURITY.md](../SECURITY.md).
