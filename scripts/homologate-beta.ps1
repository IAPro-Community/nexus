param(
  [switch]$SkipDesktop,
  [switch]$AllowDirty,
  [string]$ExpectedSha = $env:NEXUS_EXPECTED_SHA
)

$ErrorActionPreference = 'Stop'
$env:GOTOOLCHAIN = 'local'
$env:GOFLAGS = '-mod=readonly'
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

if ($env:PROCESSOR_ARCHITECTURE -ne 'AMD64') {
  throw "Windows beta homologation requires native AMD64; got $env:PROCESSOR_ARCHITECTURE"
}

$GoVersionText = (go version).Trim()
if ($GoVersionText -notmatch '\bgo1\.25\.14\b') {
  throw "Windows beta homologation requires Go 1.25.14 to match CI; got: $GoVersionText"
}

$BunVersion = (bun --version).Trim()
if ($BunVersion -ne '1.3.9') {
  throw "Windows beta homologation requires Bun 1.3.9 to match CI; got: $BunVersion"
}

$CanonicalPublicKey = (Get-Content UPDATE_PUBLIC_KEY -Raw).Trim().ToLowerInvariant()
if ($CanonicalPublicKey -notmatch '^[0-9a-f]{64}$') {
  throw "UPDATE_PUBLIC_KEY must contain the canonical 32-byte Ed25519 public key as 64 hex characters"
}
if (-not [string]::IsNullOrWhiteSpace($env:NEXUS_UPDATE_PUBLIC_KEY) -and
    $env:NEXUS_UPDATE_PUBLIC_KEY.Trim().ToLowerInvariant() -ne $CanonicalPublicKey) {
  throw "NEXUS_UPDATE_PUBLIC_KEY does not match checked-in UPDATE_PUBLIC_KEY"
}
$BuildPublicKey = $CanonicalPublicKey
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
$Out = Join-Path $Root ("DEV/validation/local-beta/{0}/{1}/windows-{2}" -f $Version, $Sha, $Arch)
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

Write-Host "==> PowerShell syntax"
$PowerShellSyntaxLog = Join-Path $Out 'powershell-syntax.log'
foreach ($ScriptPath in @(
  'install.ps1',
  'uninstall.ps1',
  'scripts/homologate-beta.ps1',
  'scripts/refresh-beta-docs-windows.ps1'
)) {
  $Tokens = $null
  $ParseErrors = $null
  [System.Management.Automation.Language.Parser]::ParseFile(
    (Resolve-Path $ScriptPath).Path,
    [ref]$Tokens,
    [ref]$ParseErrors
  ) | Out-Null
  if ($ParseErrors.Count -gt 0) {
    $ParseErrors | ForEach-Object { $ScriptPath + ": " + $_.Message } |
      Tee-Object -FilePath $PowerShellSyntaxLog -Append
    throw "PowerShell syntax failed: $ScriptPath"
  }
  "PASS $ScriptPath" | Tee-Object -FilePath $PowerShellSyntaxLog -Append
}

Write-Host "==> documentation"
bun scripts/docs-verify.mjs 2>&1 | Tee-Object -FilePath (Join-Path $Out 'docs-verify.log')
if ($LASTEXITCODE -ne 0) { throw "documentation verification failed" }

Write-Host "==> frontend dependencies"
bun install --cwd web --frozen-lockfile 2>&1 | Tee-Object -FilePath (Join-Path $Out 'bun-install.log')
if ($LASTEXITCODE -ne 0) { throw "bun install failed" }

Write-Host "==> Playwright Chromium"
Push-Location web
try {
  bunx playwright install chromium 2>&1 | Tee-Object -FilePath (Join-Path $Out 'playwright-chromium.log')
  if ($LASTEXITCODE -ne 0) { throw "Playwright Chromium installation failed" }
} finally {
  Pop-Location
}

Write-Host "==> frontend quality"
$FrontendLog = Join-Path $Out 'frontend.log'
foreach ($Task in @('format:check','typecheck','lint','lint:styles','check:styles','test','build','verify')) {
  "--- bun run $Task" | Tee-Object -FilePath $FrontendLog -Append
  bun run --cwd web $Task 2>&1 | Tee-Object -FilePath $FrontendLog -Append
  if ($LASTEXITCODE -ne 0) { throw "frontend task failed: $Task" }
}

Write-Host "==> Go formatting"
# Check only source files tracked by Git. A release gate must not fail because
# an untracked SDK/toolchain/worktree happened to exist below the checkout.
$Unformatted = @()
$TrackedGoFiles = @(git ls-files -- '*.go')
foreach ($GoFile in $TrackedGoFiles) {
  $Result = @(gofmt -l $GoFile)
  if ($LASTEXITCODE -ne 0) { throw "gofmt failed while checking $GoFile" }
  if ($Result) { $Unformatted += $Result }
}
if ($Unformatted) {
  $Unformatted | Sort-Object -Unique | Set-Content (Join-Path $Out 'gofmt-fail.txt')
  throw "gofmt required"
}

Write-Host "==> Go vet"
go vet ./... 2>&1 | Tee-Object -FilePath (Join-Path $Out 'go-vet.log')
if ($LASTEXITCODE -ne 0) { throw "go vet failed" }

Write-Host "==> Go lint (pinned)"
go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.12.2 run --timeout=5m ./... 2>&1 | Tee-Object -FilePath (Join-Path $Out 'golangci-lint.log')
if ($LASTEXITCODE -ne 0) { throw "golangci-lint failed" }

Write-Host "==> Go security (pinned)"
$SavedGoToolchain = [Environment]::GetEnvironmentVariable('GOTOOLCHAIN', 'Process')
$env:GOTOOLCHAIN = 'local'
try {
  go run golang.org/x/vuln/cmd/govulncheck@v1.7.0 ./... 2>&1 | Tee-Object -FilePath (Join-Path $Out 'govulncheck.log')
  if ($LASTEXITCODE -ne 0) { throw "govulncheck failed" }
} finally {
  if ($null -eq $SavedGoToolchain) {
    Remove-Item Env:GOTOOLCHAIN -ErrorAction SilentlyContinue
  } else {
    [Environment]::SetEnvironmentVariable('GOTOOLCHAIN', $SavedGoToolchain, 'Process')
  }
}

Write-Host "==> Version contract"
go run ./scripts/verify-version-contract.go . 2>&1 | Tee-Object -FilePath (Join-Path $Out 'version-contract.log')
if ($LASTEXITCODE -ne 0) { throw "version contract failed" }

Write-Host "==> provider-native Mission/terminal regression preflight"
go test -count=1 ./internal/app ./internal/nexus/promotion ./internal/runtime 2>&1 | Tee-Object -FilePath (Join-Path $Out 'regression-interactive-terminal.log')
if ($LASTEXITCODE -ne 0) { throw "provider-native Mission/terminal regression tests failed" }

Write-Host "==> Go tests"
# Unit/integration tests must not inherit the developer's real Nexus profile,
# conversation store or user configuration. Keep the Go module/build caches so
# Windows does not redownload the toolchain, while isolating user-visible data.
$GoTestLog = Join-Path $Out 'go-test.log'
$TestHome = Join-Path $Out 'isolated-test-home'
New-Item -ItemType Directory -Force -Path $TestHome | Out-Null

$SavedEnv = @{}
foreach ($Name in @(
  'HOME','USERPROFILE','XDG_CONFIG_HOME','XDG_DATA_HOME','XDG_CACHE_HOME',
  'NEXUS_DATA_DIR','AI_MANAGER_DATA_DIR','AI_CLI_DATA_DIR',
  'GOPATH','GOMODCACHE','GOCACHE'
)) {
  $SavedEnv[$Name] = [Environment]::GetEnvironmentVariable($Name, 'Process')
}

$ExistingGoPath = (go env GOPATH).Trim()
$ExistingGoModCache = (go env GOMODCACHE).Trim()
$ExistingGoCache = (go env GOCACHE).Trim()

$env:HOME = $TestHome
$env:USERPROFILE = $TestHome
$env:XDG_CONFIG_HOME = Join-Path $TestHome '.config'
$env:XDG_DATA_HOME = Join-Path $TestHome '.local/share'
$env:XDG_CACHE_HOME = Join-Path $TestHome '.cache'
$env:NEXUS_DATA_DIR = Join-Path $TestHome 'nexus-data'
$env:AI_MANAGER_DATA_DIR = $env:NEXUS_DATA_DIR
$env:AI_CLI_DATA_DIR = $env:NEXUS_DATA_DIR
$env:GOPATH = $ExistingGoPath
$env:GOMODCACHE = $ExistingGoModCache
$env:GOCACHE = $ExistingGoCache

$GoTestExit = 1
try {
  go test -count=1 -v ./... 2>&1 | Tee-Object -FilePath $GoTestLog
  $GoTestExit = $LASTEXITCODE
} finally {
  foreach ($Name in $SavedEnv.Keys) {
    $Value = $SavedEnv[$Name]
    if ($null -eq $Value) {
      Remove-Item "Env:$Name" -ErrorAction SilentlyContinue
    } else {
      [Environment]::SetEnvironmentVariable($Name, [string]$Value, 'Process')
    }
  }
}

if ($GoTestExit -ne 0) {
  $SummaryPath = Join-Path $Out 'go-test-failures.txt'
  $PackageFailures = @(
    Select-String -Path $GoTestLog -Pattern '^FAIL\s+\S+' |
      ForEach-Object { $_.Line.Trim() }
  )
  $TestFailures = @(
    Select-String -Path $GoTestLog -Pattern '^--- FAIL:' |
      ForEach-Object { $_.Line.Trim() }
  )
  @(
    'Sanitized Go test failure summary'
    '================================='
    ''
    'Packages:'
    $PackageFailures
    ''
    'Tests:'
    $TestFailures
  ) | Set-Content $SummaryPath
  throw "go test failed; sanitized summary: $SummaryPath"
}



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
$Ldflags = "-s -w -X $Module/internal/buildinfo.Version=$Version -X $Module/internal/buildinfo.Commit=$Commit -X $Module/internal/buildinfo.BuildDate=$BuildDate -X $Module/internal/update.ProductionTrustRootHex=$BuildPublicKey"
go build -trimpath -buildvcs=false -ldflags $Ldflags -o nexus.exe ./cmd/nexus 2>&1 | Tee-Object -FilePath (Join-Path $Out 'build-cli.log')
if ($LASTEXITCODE -ne 0) { throw "CLI build failed" }

& .\nexus.exe version --json | Tee-Object -FilePath (Join-Path $Out 'version.json')
$VersionJson = Get-Content (Join-Path $Out 'version.json') -Raw | ConvertFrom-Json
if ($VersionJson.version -ne $Version) { throw "built version $($VersionJson.version) != VERSION $Version" }

Write-Host "==> Browser E2E on the current Windows binary"
$SavedSkipBuild = [Environment]::GetEnvironmentVariable('SKIP_BUILD', 'Process')
$env:SKIP_BUILD = '1'
try {
  foreach ($Task in @(
    'test:e2e',
    'test:e2e:composer',
    'test:e2e:surface',
    'test:e2e:capacity',
    'test:e2e:hud',
    'test:e2e:composer-spine',
    'test:e2e:composer-vague-spine',
    'test:e2e:composer-themes',
    'test:a11y',
    'test:visual'
  )) {
    $E2ELog = Join-Path $Out ("{0}.log" -f ($Task -replace ':','-'))
    bun run --cwd web $Task 2>&1 | Tee-Object -FilePath $E2ELog
    if ($LASTEXITCODE -ne 0) { throw "browser E2E failed: $Task" }
  }
} finally {
  if ($null -eq $SavedSkipBuild) {
    Remove-Item Env:SKIP_BUILD -ErrorAction SilentlyContinue
  } else {
    [Environment]::SetEnvironmentVariable('SKIP_BUILD', $SavedSkipBuild, 'Process')
  }
}

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
  bun run --cwd web build
  if ($LASTEXITCODE -ne 0) { throw "frontend build for Wails v3 failed" }

  New-Item -ItemType Directory -Path "cmd/nexus-desktop/build/bin" -Force | Out-Null
  New-Item -ItemType Directory -Path "cmd/nexus-hud/build/bin" -Force | Out-Null
  $DesktopPath = Join-Path $Root "cmd/nexus-desktop/build/bin/nexus-desktop.exe"
  $HudPath = Join-Path $Root "cmd/nexus-hud/build/bin/nexus-hud.exe"

  go build -trimpath -buildvcs=false -tags production -ldflags $Ldflags -o $DesktopPath ./cmd/nexus-desktop
  if ($LASTEXITCODE -ne 0) { throw "Nexus Desktop Wails v3 build failed" }
  go build -trimpath -buildvcs=false -tags production -ldflags $Ldflags -o $HudPath ./cmd/nexus-hud
  if ($LASTEXITCODE -ne 0) { throw "Nexus HUD Wails v3 build failed" }

  "wails-v3.0.0-beta.25" | Set-Content (Join-Path $Out 'native-framework.txt')
}

@'
# Manual beta smoke still required

These checks require a real interactive/native session and are intentionally not faked by the script.

- Start `nexus codex`; press Up/Down/Left/Right repeatedly. No literal `[A`, `[B`, `[C`, `[D` may appear.
- Start `nexus codex` and `nexus agy` with no Lead/delegation flags. Submit both a trivial prompt and a complex request such as "corrija a API". Neither may create, require, or ask for a Mission; provider-native Enter/commands must continue normally.
- Start `nexus codex --lead` with `NEXUS_DELEGATION_MODE` unset. The Lead may provide Nexus context, but ordinary prompts still must not be promoted to Mission because the persisted delegation mode starts OFF.
- Opt in deliberately with `NEXUS_DELEGATION_MODE=ASK` plus `--lead`. A Mission-worthy prompt may ask for promotion, while a simple prompt stays provider-owned. Repeat with `AUTO` only as an explicit orchestration smoke.
- In normal provider mode, submit prompts containing `permission`, `submission` and `transmission`. They must never be treated as explicit Mission requests.
- Explicitly request a Mission while strict admission is intentionally unavailable. The prompt must NOT execute as Direct; Nexus must show the exact blocker and retain the rejected plan id for remediation.
- With a hard/emergency cost budget requiring confirmation, an automatic heuristic promotion must not bypass the budget gate. It may return to Lead before a run exists; an explicit Mission request must stay blocked until the current cost state is explicitly confirmed.
- Keep the project default isolation as `project` and run an AUTO/CREATE Mission in a real Git repository. Nexus must use/create a worktree-safe Agent instead of blocking only because the project default is not `worktree`.
- Pin an EXISTING Agent with `Isolation=project` and attempt an autonomous Mission. Admission must fail before `run_id` with an actionable `worktree_isolation` blocker.
- Set project default to `worktree` but explicitly configure the pinned Agent with `Isolation=project`. The Agent override must win and admission must still fail; autonomous execution must never reach the canonical checkout.
- Open Agents and create/edit an Agent. OpenCode must be offered independently from AGY.
- Open Usage. If OpenCode is installed but has no profile yet, it must still be shown as an installed provider.
- Select an OpenCode Review or Investigate execution role. Nexus must materialize/discover the projected agent without invoking the model.
- Click Certify only after acknowledging quota/cost. Activation must select the projected agent; enforcement must prove effective write-tool denial/removal, keep the canary absent and only then show ENFORCED.
- Remove the active projection. The receipt link must clear and activation strategy must return to `NEXUS_OVERLAY`; an older/stale receipt must not remove or promote a newer projection.
- Repeat the custom-target smoke with the provider absent on the host when feasible. For VPN Dev Workspace on Windows, use target_workspace=/workspace; no host-visible profile home means overlay/BEST_EFFORT rather than a fabricated ENFORCED state.
- Launch Nexus HUD and verify show/hide, autostart toggle and basic projection.
- If voice is configured, verify microphone permission, capture, transcription, provider fallback, terminal insertion without automatic Enter, and one non-destructive global voice command.
'@ | Set-Content (Join-Path $Out 'MANUAL_CHECKS.md')

"completed_at=$([DateTime]::UtcNow.ToString('yyyy-MM-ddTHH:mm:ssZ'))" | Add-Content (Join-Path $Out 'environment.txt')
Write-Host "PASS automated Windows beta homologation: $Version @ $Sha"
Write-Host "Evidence: $Out"
