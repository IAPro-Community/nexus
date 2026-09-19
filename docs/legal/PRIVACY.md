# IAPro Nexus — Privacy Notice (beta)

Status: draft for owner review before first external distribution.

## 1. Data that stays on your device

Projects, prompts, code, files, terminal content, provider credentials,
cookies, and the local telemetry history never leave your machine through
Nexus itself. Provider CLIs you run may have their own network behavior;
that is governed by those providers' terms, not this notice.

## 2. Product analytics (opt-in)

- Disabled by default. Nothing is sent unless you opt in.
- When enabled, only allowlisted aggregate event names are recorded
  (e.g. `app_started`, `mission_completed`, provider id, OS, install
  method). No prompts, code, files, tokens, keys, paths, or terminal
  content are ever eligible fields — the client has no code path that
  sends them.
- You can opt out at any time (`NEXUS_ANALYTICS=0`); the app works
  identically offline.

## 3. Updates

The updater fetches a signed channel manifest and signed artifacts over
HTTPS and verifies the Ed25519 signature plus SHA-256 before installing.
Update checks transmit only your current version, channel, OS, and
architecture.

## 4. Installation identity

A random `installation_id` is minted on first run to correlate your own
installations (e.g. entitlements). It is not derived from hardware, MAC,
or serial numbers, and you can reset it at any time.

## 5. Diagnostics bundles

`nexus diagnostics` (or an expanded `doctor`) assembles a support bundle
only when you ask. Before creating it, Nexus shows exactly which data is
included. API keys, cookies, provider tokens, and project files are never
included automatically.

## 6. Contact and rights

For access, correction, or deletion requests regarding any beta data held
by IAPro, use the official support channel named in the beta invitation.
