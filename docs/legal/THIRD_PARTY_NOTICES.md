# IAPro Nexus — Third-Party Notices (v1.0.0-beta.1)

Status: generated inventory for owner review. Per-component license texts
ship with the release artifact; this file records the method plus the
pinned inventories.

## Method

- Go: `go list -m all` pinned in `docs/legal/GO_MODULES.txt` (regenerate on
  every release: `go list -m all > docs/legal/GO_MODULES.txt`).
- Web: direct runtime dependencies pinned below from `web/package.json`;
  transitive closure is locked in `web/bun.lock`. Regenerate with
  `bun install --frozen-lockfile` + `bun pm ls` equivalent at release time.
- SBOM (CycloneDX) is produced by the release pipeline (see
  `docs/release/BETA_PLAYBOOK.md`); this file is the human-readable index.

## Go modules

See `docs/legal/GO_MODULES.txt`. The exact module inventory is regenerated again at the certified Wails v3 release cut.

Notable copyleft-adjacent review points for the owner (no conclusion
stated here — confirm with counsel):

- `modernc.org/sqlite` and its `modernc.org/*` tree (BSD-style; pure-Go
  SQLite driver, statically linked into the binary).
- `github.com/go-git/go-git/v5` (Apache-2.0) and transitive git plumbing.
- `github.com/wailsapp/wails/v3` v3.0.0-beta.25 (MIT) desktop runtime plus
  `@wailsio/runtime` 3.0.0-beta.25 and the platform-native WebView stack.
- `golang.org/x/*` (BSD-3-Clause), `github.com/charmbracelet/*` (MIT),
  `github.com/gorilla/websocket` (BSD-2-Clause).

## Web runtime dependencies (direct, from web/package.json)

- `@assistant-ui/react` — MIT
- `@radix-ui/react-alert-dialog`, `@radix-ui/react-dialog`,
  `@radix-ui/react-dropdown-menu`, `@radix-ui/react-tooltip` — MIT
- `@xyflow/react` — MIT
- `i18next`, `i18next-browser-languagedetector` — MIT
- `lucide-react` — ISC
- `react`, `react-dom` — MIT
- `react-i18next` — MIT
- `react-qr-code` — MIT
- `react-router-dom` — MIT
- `sonner` — MIT
- `xterm`, `xterm-addon-fit` — MIT

(License labels above are the packages' declared licenses; verify against
the SBOM at release time. `tailwindcss`/`sass`/`vitest`/`playwright` and
other devDependencies do not ship in the product bundle but are listed in
the SBOM build section.)
