[CmdletBinding()]
param(
    [switch]$WithBrowser
)

$ErrorActionPreference = 'Stop'
$root = (Resolve-Path (Join-Path $PSScriptRoot '..')).Path
Set-Location $root

function Assert-Exit([string]$Name) {
    if ($LASTEXITCODE -ne 0) {
        throw "$Name failed with exit code $LASTEXITCODE"
    }
}

$branch = (git branch --show-current).Trim()
Assert-Exit 'Read Git branch'
if ($branch -ne 'release/v1.0.0') {
    throw "Certification must run on release/v1.0.0; current branch: $branch"
}

$sha = (git rev-parse HEAD).Trim()
Assert-Exit 'Read Git commit'
if ((git status --porcelain | Measure-Object).Count -gt 0) {
    throw 'Certification requires a clean worktree to bind evidence to an exact commit'
}
Write-Host "Certifying Nexus release/v1.0.0 at $sha"

$version = (& go version)
Assert-Exit 'Go version'
if ($version -notmatch 'go1\.(\d+)') {
    throw "Cannot identify Go version: $version"
}
$minor = [int]$Matches[1]
if ($minor -lt 25) {
    throw "Go 1.25+ required; detected: $version"
}

$changedGo = @(
    'internal/nexus/store/capacity_pace.go',
    'internal/nexus/store/capacity_pace_test.go',
    'internal/nexus/store/external_claude.go',
    'internal/nexus/store/external_claude_test.go',
    'internal/nexus/capacitymonitor/pace.go',
    'internal/nexus/capacitymonitor/pace_test.go',
    'internal/nexus/capacitymonitor/monitor.go',
    'internal/nexus/capacity_application.go',
    'internal/nexus/externalusage/claude.go',
    'internal/nexus/externalusage/claude_test.go',
    'internal/control/web/public_service_status.go',
    'internal/control/web/public_service_status_test.go',
    'internal/control/web/handlers_capacity.go',
    'internal/control/web/handlers_capacity_test.go',
    'internal/control/web/handlers_external_claude.go',
    'internal/control/web/handlers_external_claude_test.go',
    'internal/control/web/handlers_cost_csv.go',
    'internal/control/web/handlers_cost_csv_test.go',
    'internal/control/web/routes.go'
)
$unformatted = @(& gofmt -l $changedGo)
Assert-Exit 'gofmt check'
if ($unformatted.Count -gt 0) {
    throw "Go files require gofmt before certifying:`n$($unformatted -join [Environment]::NewLine)"
}

& go test ./internal/nexus/store ./internal/nexus/capacitymonitor ./internal/nexus/externalusage ./internal/control/web ./internal/nexus
Assert-Exit 'Go domain and HTTP regression tests'

Push-Location (Join-Path $root 'web')
try {
    & npm run quality:full
    Assert-Exit 'Frontend quality:full'
    if ($WithBrowser) {
        & npm run test:e2e:capacity
        Assert-Exit 'Capacity browser E2E'
    }
} finally {
    Pop-Location
}

Write-Host "PASS: automated local gates for commit $sha"
if (-not $WithBrowser) {
    Write-Warning 'Browser E2E, Windows ConPTY native smoke and release signing NOT certified by this run.'
} else {
    Write-Warning 'Windows ConPTY native smoke and release signing still require separate gates.'
}
