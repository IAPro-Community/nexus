#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"
[[ "$(git branch --show-current)" == release/v1.0.0 ]] || { echo 'Expected release/v1.0.0' >&2; exit 1; }
[[ -z "$(git status --porcelain)" ]] || { echo 'Worktree must be clean' >&2; exit 1; }
SHA="$(git rev-parse HEAD)"
echo "Certifying Nexus release/v1.0.0 at $SHA"
go version | grep -Eq 'go1\.(2[5-9]|[3-9][0-9])' || { echo 'Go >=1.25 required' >&2; exit 1; }
GO_FILES=(
internal/nexus/store/capacity_pace.go
internal/nexus/store/capacity_pace_test.go
internal/nexus/store/external_claude.go
internal/nexus/store/external_claude_test.go
internal/nexus/capacitymonitor/pace.go
internal/nexus/capacitymonitor/pace_test.go
internal/nexus/capacitymonitor/monitor.go
internal/nexus/capacity_application.go
internal/nexus/externalusage/claude.go
internal/nexus/externalusage/claude_test.go
internal/control/web/public_service_status.go
internal/control/web/public_service_status_test.go
internal/control/web/handlers_capacity.go
internal/control/web/handlers_capacity_test.go
internal/control/web/handlers_external_claude.go
internal/control/web/handlers_external_claude_test.go
internal/control/web/handlers_cost_csv.go
internal/control/web/handlers_cost_csv_test.go
internal/control/web/routes.go
)
UNFORMATTED="$(gofmt -l "${GO_FILES[@]}")"
if [[ -n "$UNFORMATTED" ]]; then
  printf 'Go files require gofmt:\n%s\n' "$UNFORMATTED" >&2
  exit 1
fi
go test ./internal/nexus/store ./internal/nexus/capacitymonitor ./internal/nexus/externalusage ./internal/control/web ./internal/nexus
(cd web && npm run quality:full)
if [[ "${1:-}" == '--browser' ]]; then
  (cd web && npm run test:e2e:capacity)
fi
printf 'PASS: automated gates at %s\n' "$SHA"
printf 'Windows ConPTY, native desktop, packaging and signing still require independent gates.\n'
