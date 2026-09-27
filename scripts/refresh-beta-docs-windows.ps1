param(
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

$Sha = (git rev-parse HEAD).Trim()
if ($ExpectedSha) {
  $ResolvedExpectedSha = (git rev-parse $ExpectedSha 2>$null)
  if ($LASTEXITCODE -eq 0 -and $ResolvedExpectedSha) {
    $ExpectedSha = $ResolvedExpectedSha.Trim()
  }
  if ($Sha -ne $ExpectedSha.Trim()) {
    throw "documentation capture SHA mismatch: HEAD=$Sha expected=$ExpectedSha"
  }
}

$Dirty = git status --porcelain
if ($Dirty) {
  throw "working tree is dirty; documentation evidence must be captured from an exact clean SHA"
}

Write-Host "==> Install locked frontend dependencies"
bun install --cwd web --frozen-lockfile
if ($LASTEXITCODE -ne 0) {
  throw "bun install failed"
}

Write-Host "==> Ensure Playwright Chromium"
Push-Location web
try {
  bunx playwright install chromium
  if ($LASTEXITCODE -ne 0) {
    throw "Playwright Chromium installation failed"
  }

  Write-Host "==> Capture documentation evidence from $Sha"
  bun run docs:capture
  if ($LASTEXITCODE -ne 0) {
    throw "documentation capture failed"
  }
} finally {
  Pop-Location
}

Write-Host "==> Verify captured documentation"
bun scripts/docs-verify.mjs
if ($LASTEXITCODE -ne 0) {
  throw "documentation verification failed after recapture"
}

$ManifestPath = Join-Path $Root 'docs/assets/screenshots/manifest.json'
if (-not (Test-Path $ManifestPath)) {
  throw "screenshot manifest was not generated"
}

$Manifest = Get-Content $ManifestPath -Raw | ConvertFrom-Json
if ([string]$Manifest.source_sha -ne $Sha) {
  throw "screenshot manifest source_sha=$($Manifest.source_sha) does not match captured HEAD=$Sha"
}

$EvidenceChanges = @(
  git status --porcelain -- docs/assets/screenshots docs/assets/demos
)

if (-not $EvidenceChanges) {
  throw "capture completed but no documentation evidence changed; inspect docs-capture output before certifying"
}

Write-Host ""
Write-Host "Documentation evidence captured and verified for exact SHA $Sha."
Write-Host "Review and commit ONLY the generated documentation evidence before the final homologation rerun:"
$EvidenceChanges | ForEach-Object { Write-Host "  $_" }
Write-Host ""
Write-Host "Do not edit manifest.source_sha manually. After committing the evidence, rerun scripts/homologate-beta.ps1 from the new exact SHA."
