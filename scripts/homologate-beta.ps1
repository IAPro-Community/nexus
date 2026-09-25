param(
  [switch]$SkipDesktop,
  [switch]$AllowDirty,
  [string]$ExpectedSha = $env:NEXUS_EXPECTED_SHA
)

$ErrorActionPreference = 'Stop'
$Root = Split-Path -Parent $PSScriptRoot
Set-Location $Root

function Require-Command([string]$Name) {
  if (-not (Get-Command $Name -ErrorAction SilentlyContinue)) {
    throw "missing required command: $Name"
  }
}

Require-Command git
Require-Command go
Require-Command bun

$Version = (Get-Content VERSION -Raw).Trim()
$Sha = (git rev-parse HEAD).Trim()
$ShortSha = (git rev-parse --short HEAD).Trim()

if ($ExpectedSha) {
  $ResolvedExpectedSha = (git rev-parse $ExpectedSha 2>$null)
  if ($LASTEXITCODE -eq 0 -and $ResolvedExpectedSha) {
    $ExpectedSha = $ResolvedExpectedSha.Trim()
  }
  if ($Sha -ne $ExpectedSha.Trim()) {
    throw "homologation SHA mismatch: HEAD=$Sha expected=$ExpectedSha"
  }
}
$Arch = $env:PROCESSOR_ARCHITECTURE
$Out = Join-Path $Root ("DEV/validation/local-beta2/{0}/windows-{1}" -f $Sha, $Arch)
New-Item -ItemType Directory -Force -Path $Out | Out-Null

if (-not $AllowDirty) {
  $Dirty = git status --porcelain
  if ($Dirty) {
    throw "working tree is dirty; homologation must run on an exact SHA (use -AllowDirty only for exploratory runs)"
  }
}

@(
  "sha=$Sha"
  "short_sha=$ShortSha"
  "version=$Version"
  "os=windows"
  "arch=$Arch"
  "go=$(go version)"
  "bun=$(bun --version)"
  "started_at=$([DateTime]::UtcNow.ToString('yyyy-MM-ddTHH:mm:ssZ'))"
) | Tee-Object -FilePath (Join-Path $Out 'environment.txt')

Write-Host "==> documentation"
bun scripts/docs-verify.mjs 2>&1 | Tee-Object -FilePath (Join-Path $Out 'docs-verify.log')

Write-Host "==> frontend dependencies"
bun --cwd web install --frozen-lockfile 2>&1 | Tee-Object -FilePath (Join-Path $Out 'bun-install.log')

Write-Host "==> frontend quality"
$FrontendLog = Join-Path $Out 'frontend.log'
foreach ($Task in @('format:check','typecheck','lint','lint:styles','check:styles','test','build','verify')) {
  "--- bun run $Task" | Tee-Object -FilePath $FrontendLog -Append
  bun --cwd web run $Task 2>&1 | Tee-Object -FilePath $FrontendLog -Append
  if ($LASTEXITCODE -ne 0) { throw "frontend task failed: $Task" }
}

Write-Host "==> Go formatting"
$Unformatted = gofmt -l .
$Unformatted = $Unformatted | Where-Object { $_ -notmatch '^[.]?[\\/]?\.worktrees[\\/]' }
if ($Unformatted) {
  $Unformatted | Set-Content (Join-Path $Out 'gofmt-fail.txt')
  throw "gofmt required"
}

Write-Host "==> Go vet"
go vet ./... 2>&1 | Tee-Object -FilePath (Join-Path $Out 'go-vet.log')
if ($LASTEXITCODE -ne 0) { throw "go vet failed" }

Write-Host "==> Go tests"
go test -v ./... 2>&1 | Tee-Object -FilePath (Join-Path $Out 'go-test.log')
if ($LASTEXITCODE -ne 0) { throw "go test failed" }

Write-Host "==> regression suites: interactive routing and terminal input"
go test -count=1 ./internal/nexus/promotion ./internal/runtime 2>&1 | Tee-Object -FilePath (Join-Path $Out 'regression-interactive-terminal.log')
if ($LASTEXITCODE -ne 0) { throw "interactive/terminal regression tests failed" }

Write-Host "==> control/runtime E2E"
go test -count=1 ./internal/control/terminal/... ./internal/control/protocol/... ./internal/control/host/... ./internal/control/web/... 2>&1 | Tee-Object -FilePath (Join-Path $Out 'control-e2e.log')
if ($LASTEXITCODE -ne 0) { throw "control/runtime E2E failed" }

Write-Host "==> Agent Interop / OpenCode Projection regressions"
go test -count=1 ./internal/core/provider/agentprojection/opencode ./internal/control/launcher ./internal/control/web ./internal/nexus ./internal/nexus/store 2>&1 | Tee-Object -FilePath (Join-Path $Out 'agent-interop.log')
if ($LASTEXITCODE -ne 0) { throw "Agent Interop/OpenCode Projection regressions failed" }

Write-Host "==> build CLI"
$Commit = $ShortSha
$BuildDate = [DateTime]::UtcNow.ToString('yyyy-MM-ddTHH:mm:ssZ')
$Module = 'github.com/IAPro-Community/nexus'
$Ldflags = "-s -w -X $Module/internal/buildinfo.Version=$Version -X $Module/internal/buildinfo.Commit=$Commit -X $Module/internal/buildinfo.BuildDate=$BuildDate"
go build -trimpath -buildvcs=false -ldflags $Ldflags -o nexus.exe ./cmd/nexus 2>&1 | Tee-Object -FilePath (Join-Path $Out 'build-cli.log')
if ($LASTEXITCODE -ne 0) { throw "CLI build failed" }

& .\nexus.exe version --json | Tee-Object -FilePath (Join-Path $Out 'version.json')
$VersionJson = Get-Content (Join-Path $Out 'version.json') -Raw | ConvertFrom-Json
if ($VersionJson.version -ne $Version) { throw "built version $($VersionJson.version) != VERSION $Version" }

& .\nexus.exe doctor --json | Tee-Object -FilePath (Join-Path $Out 'doctor.json')
& .\nexus.exe providers status --json | Tee-Object -FilePath (Join-Path $Out 'providers.json')
try {
  & .\nexus.exe usage --json | Tee-Object -FilePath (Join-Path $Out 'usage.json')
} catch {
  $_ | Out-File (Join-Path $Out 'usage-error.txt')
}

if (Get-Command opencode -ErrorAction SilentlyContinue) {
  Write-Host "==> OpenCode visibility regression"
  $Providers = Get-Content (Join-Path $Out 'providers.json') -Raw
  if ($Providers -notmatch '(?i)opencode') { throw "OpenCode is installed but absent from nexus providers status --json" }
  $UsagePath = Join-Path $Out 'usage.json'
  if (-not (Test-Path $UsagePath) -or ((Get-Content $UsagePath -Raw) -notmatch '(?i)opencode')) {
    throw "OpenCode is installed but absent from nexus usage --json"
  }
}

if (-not $SkipDesktop) {
  Write-Host "==> Windows Wails v3 Desktop + HUD build"
  bun --cwd web run build
  New-Item -ItemType Directory -Path "cmd/nexus-desktop/build/bin" -Force | Out-Null
  New-Item -ItemType Directory -Path "cmd/nexus-hud/build/bin" -Force | Out-Null
  go build -trimpath -buildvcs=false -tags production -ldflags $Ldflags -o cmd/nexus-desktop/build/bin/nexus-desktop.exe ./cmd/nexus-desktop
  if ($LASTEXITCODE -ne 0) { throw "Wails v3 Desktop build failed" }
  go build -trimpath -buildvcs=false -tags production -ldflags $Ldflags -o cmd/nexus-hud/build/bin/nexus-hud.exe ./cmd/nexus-hud
  if ($LASTEXITCODE -ne 0) { throw "Wails v3 HUD build failed" }
}

@'
# Manual beta smoke still required

These checks require a real interactive/native session and are intentionally not faked by the script.

- Start `nexus codex`; press Up/Down/Left/Right repeatedly. No literal `[A`, `[B`, `[C`, `[D` may appear.
- In Direct/Lead mode, submit a simple request. It must remain with the Lead and must not be promoted to Mission unless decomposition/delegation/explicit Mission intent is present.
- In Direct/Lead mode, submit prompts containing `permission`, `submission` and `transmission`. They must not be treated as explicit Mission requests.
- Force a heuristic Mission-worthy request while strict admission is intentionally unavailable. The original prompt must execute once in the Lead; no dead-end and no orphan heuristic WorkPlan should remain.
- Explicitly request a Mission while strict admission is intentionally unavailable. The prompt must NOT execute as Direct; Nexus must show the exact blocker and retain the rejected plan id for remediation.
- With a hard/emergency cost budget requiring confirmation, an automatic heuristic promotion must not bypass the budget gate. It may return to Lead before a run exists; an explicit Mission request must stay blocked until the current cost state is explicitly confirmed.
- Keep the project default isolation as `project` and run an AUTO/CREATE Mission in a real Git repository. Nexus must use/create a worktree-safe Agent instead of blocking only because the project default is not `worktree`.
- Pin an EXISTING Agent with `Isolation=project` and attempt an autonomous Mission. Admission must fail before `run_id` with an actionable `worktree_isolation` blocker.
- Set project default to `worktree` but explicitly configure the pinned Agent with `Isolation=project`. The Agent override must win and admission must still fail; autonomous execution must never reach the canonical checkout.
- Open Agents and create/edit an Agent. OpenCode must be offered independently from AGY.
- Open Usage. If OpenCode is installed but has no profile yet, it must still be shown as an installed provider.
- With an OpenCode runtime and isolated/shared Nexus profile, apply a REVIEW projection and verify that discovery lists `nexus-review` without invoking an LLM.
- Run the explicit Agent Projection activation probe only after acknowledging possible quota/cost. It must select `nexus-review`, record `NATIVE_ON_LAUNCH`, and must not promote read-only enforcement to `ENFORCED`.
- Remove the active projection. The receipt link must clear and activation strategy must return to `NEXUS_OVERLAY`; an older/stale receipt must not remove or promote a newer projection.
- Exercise one custom target (Docker/VPN/WSL/SSH wrapper): provider may be absent on the host, `target_cwd` must resolve to the target-visible workspace, and no shell metacharacter template may be accepted.
- Launch Nexus HUD and verify show/hide, autostart toggle and basic projection.
- If voice is configured, verify microphone permission, transcription, provider fallback and terminal insertion without automatic Enter.
'@ | Set-Content (Join-Path $Out 'MANUAL_CHECKS.md')

"completed_at=$([DateTime]::UtcNow.ToString('yyyy-MM-ddTHH:mm:ssZ'))" | Add-Content (Join-Path $Out 'environment.txt')
Write-Host "PASS automated local beta homologation: $Version @ $Sha"
Write-Host "Evidence: $Out"
