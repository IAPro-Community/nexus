#!/usr/bin/env bash
set -euo pipefail

# Evidence-oriented inventory for the Nexus consolidation campaign. The
# script is intentionally read-only: it reports implementation/documentation
# presence and does not infer that a dispatcher is the whole feature.
root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$root"

status_for() {
  local label="$1"; shift
  if "$@" >/dev/null 2>&1; then
    printf '%-28s IMPLEMENTED\n' "$label"
  else
    printf '%-28s UNKNOWN\n' "$label"
  fi
}

printf 'Nexus characterization\n'
printf 'branch=%s\n' "$(git branch --show-current)"
printf 'head=%s\n' "$(git rev-parse HEAD)"
printf 'working_tree=%s\n\n' "$(test -z "$(git status --porcelain)" && echo clean || echo dirty)"

printf '%-28s STATUS\n' CAPABILITY
printf '%-28s ------\n' ----------------------------
status_for 'project store' rg -q 'func \(s \*Store\) (CreateProject|ListProjects)' internal/nexus/store
status_for 'agent store' rg -q 'func \(s \*Store\) (CreateAgent|ListAgents)' internal/nexus/store
status_for 'agent generations' rg -q 'func \(s \*Store\) (AddGeneration|ListGenerations)' internal/nexus/store
status_for 'work plans' rg -q 'type WorkPlan struct' internal/nexus/store
status_for 'mission runner' rg -q 'type MissionRunner struct' internal/nexus/runner
status_for 'prompt compiler' rg -q 'func \(e \*NexusEngine\) CompilePrompt' internal/nexus/intelligence
status_for 'Maestro boundary' rg -q 'type MaestroClient struct' internal/nexus
status_for 'quota evidence' rg -q 'UsageUnknown|UNKNOWN' internal/core/quota
status_for 'CLI plan compile case' rg -q 'case "compile"' internal/app/nexus_cli_cmds.go
status_for 'CLI plan run case' rg -q 'case "run"' internal/app/nexus_cli_cmds.go

printf '\nSurface references\n'
rg -n 'plan \[list\|create\|show\|compile\|run\]|case "(compile|run)"|func \(e \*NexusEngine\) CompilePrompt|func \(s \*Store\) AddGeneration' \
  internal/app internal/nexus docs/product docs/refactoring 2>/dev/null || true
