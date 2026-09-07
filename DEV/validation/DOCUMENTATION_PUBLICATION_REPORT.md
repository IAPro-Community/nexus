# Documentation Publication Report

## Baseline

- Branch base: `feat/nexus-maximum-delivery`
- Base SHA: `5b37572dbed6e551217cdf3d1af16fbe93e2e092`
- Final documentation SHA: `1293dff` (local worktree branch)
- Reference main: `f71eb515278168e33d626fd631cd89dfb5e58faf`
- Worktree: `docs/nexus-publication-overhaul`

## Inventory and decisions

The repository contained a large README acting as a manual, public pages mixed
with historical `community-preview` material, and engineering/agent evidence in
`DEV/` and `.omx/`. The public entry point is now the README plus `docs/README.md`;
the existing historical material was preserved rather than deleted. New
canonical pages cover getting started, product surfaces, Composer, Flow,
Mission, Maestro, architecture, Desktop, operations and terminology.

`DEV/` and `.omx/` remain internal evidence and execution areas. Public pages do
not use them as onboarding navigation. The docs verifier scans the curated
public tree and rejects missing local links, missing required assets and
internal-path references.

## Visual evidence

The capture command builds the current checkout, boots the real Nexus Web
server, authenticates through its bootstrap URL, waits for `.nx-os-shell`, and
captures the mounted UI. Paths are sanitized only in the screenshot DOM; no
status or product state is fabricated.

| Surface | Real capture | Source SHA | Notes |
| --- | --- | --- | --- |
| Workspace | PASS | `5b37572...` | `workspace-overview.png` |
| Composer | PASS | `5b37572...` | `composer.png` |
| Flow | PASS | `5b37572...` | `flow.png` |
| Agents | PASS | `5b37572...` | `agents.png` |
| Projects | NOT_CAPTURED | — | needs a deterministic scenario |
| Direct Session | NOT_CAPTURED | — | needs a deterministic scenario |
| Provider | NOT_CAPTURED | — | no safe dedicated state captured |
| Terminal | NOT_CAPTURED | — | requires a controlled runtime fixture |
| Mission | NOT_CAPTURED | — | requires a real run fixture |
| Usage/Quota | NOT_CAPTURED | — | must preserve truthful provider state |
| Desktop | NOT_CAPTURED | — | native capture not available in this Linux session |

The required visual coverage is therefore incomplete. This report intentionally
does not turn missing evidence into a publication claim.

## Verification

- `bun install --frozen-lockfile`: PASS
- `bun run docs:capture`: PASS; 4 real Web captures at the baseline SHA
- `make docs-verify`: PASS; 55 curated Markdown files and required assets
- `git diff --check`: PASS
- Frontend build executed by capture: PASS
- Full cross-platform runtime, native Desktop and external CI: UNKNOWN in this
  documentation worktree and not claimed here.

## Truthfulness and language

README PT-BR, EN and ES now share the same positioning, Direct workflow,
progressive Composer/Flow/Mission explanation, Web/Desktop/CLI distinction and
Maestro-optional statement. Unsupported native-platform claims are directed to
the platform matrix instead of being presented as universal support.

## Final verdict

`NO-GO` for publication: the public information architecture and four real Web
captures are in place, but the mandatory visual evidence matrix is not complete,
especially for Terminal, Mission, Usage/Quota and native Desktop. Capture those
surfaces from deterministic, sanitized scenarios and rerun `make docs-verify`
before changing this verdict.
