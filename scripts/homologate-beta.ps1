param(
  [switch]$SkipDesktop,
  [switch]$AllowDirty
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
$Arch = $env:PROCESSOR_ARCHITECTURE
$Out = Join-Path $Root ("DEV/validation/local-beta3/{0}/windows-{1}" -f $Sha, $Arch)
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
  Write-Host "==> Windows Wails Desktop + HUD build"
  bun --cwd web run build
  Push-Location cmd/nexus-desktop
  try {
    go run github.com/wailsapp/wails/v2/cmd/wails@v2.15.0 build -clean -s -m -tags production -ldflags $Ldflags
  } finally { Pop-Location }
  Push-Location cmd/nexus-hud
  try {
    go run github.com/wailsapp/wails/v2/cmd/wails@v2.15.0 build -clean -s -m -tags production -ldflags $Ldflags
  } finally { Pop-Location }
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
- Select an OpenCode Review or Investigate execution role. Nexus must materialize/discover the projected agent without invoking the model.
- Click Certify only after acknowledging quota/cost. Activation must select the projected agent; enforcement must attempt a controlled write/edit/patch, observe a permission/policy deny, keep the canary absent and only then show ENFORCED.
- Repeat the custom-target smoke with the provider absent on the host when feasible. For VPN Dev Workspace on Windows, use target_workspace=/workspace; no host-visible profile home means overlay/BEST_EFFORT rather than a fabricated ENFORCED state.
- Launch Nexus HUD and verify show/hide, autostart toggle and basic projection.
- If voice is configured, verify microphone permission, capture, transcription, provider fallback, terminal insertion without automatic Enter, and one non-destructive global voice command.
'@ | Set-Content (Join-Path $Out 'MANUAL_CHECKS.md')

"completed_at=$([DateTime]::UtcNow.ToString('yyyy-MM-ddTHH:mm:ssZ'))" | Add-Content (Join-Path $Out 'environment.txt')
Write-Host "PASS automated local beta homologation: $Version @ $Sha"
Write-Host "Evidence: $Out"
