[CmdletBinding()]
param(
    [string]$ExpectedSha = $env:NEXUS_EXPECTED_SHA,
    [string]$OutputDirectory = '',
    [switch]$AllowDirty
)

# Native Windows first-run preflight. No login, no global PATH writes, no
# installation or removal of provider accounts, no user-data mutation.
Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'
$root = Split-Path -Parent $PSScriptRoot
Push-Location $root
try {
    if ($env:PROCESSOR_ARCHITECTURE -ne 'AMD64') {
        throw "Windows AMD64 is required for this certification"
    }
    $sha = (& git rev-parse HEAD).Trim()
    if ($LASTEXITCODE -ne 0 -or $sha -notmatch '^[0-9a-f]{40}$') {
        throw 'Cannot establish exact source SHA'
    }
    if (-not [string]::IsNullOrWhiteSpace($ExpectedSha) -and $sha -ne $ExpectedSha.Trim()) {
        throw "Source SHA mismatch: expected $ExpectedSha, got $sha"
    }
    $rawBranch = (& git branch --show-current 2>$null)
    $branchName = if ($null -ne $rawBranch) { $rawBranch.ToString().Trim() } else { '' }
    if ([string]::IsNullOrWhiteSpace($branchName)) {
        if ($env:GITHUB_REF_NAME -eq 'release/v1.0.0' -or $env:GITHUB_REF -eq 'refs/heads/release/v1.0.0') {
            $branchName = 'release/v1.0.0'
        } else {
            $branches = @(& git branch -a --contains HEAD 2>$null)
            if ($branches -match 'release/v1\.0\.0') {
                $branchName = 'release/v1.0.0'
            }
        }
    }
    if ($branchName -ne 'release/v1.0.0') {
        throw "Certification requires release/v1.0.0; got $branchName"
    }
    if (-not $AllowDirty) {
        $trackedChanges = @(& git status --porcelain --untracked-files=no)
        if ($LASTEXITCODE -ne 0 -or $trackedChanges.Count -gt 0) {
            throw 'Tracked checkout differs from tested SHA; use a clean checkout'
        }
    }
    if ([string]::IsNullOrWhiteSpace($OutputDirectory)) {
        $OutputDirectory = Join-Path ([System.IO.Path]::GetTempPath()) ("nexus-first-run-" + $sha.Substring(0, 12))
    }
    New-Item -ItemType Directory -Force -Path $OutputDirectory | Out-Null
    $outputRoot = (Resolve-Path -LiteralPath $OutputDirectory).Path
    $logRoot = Join-Path $outputRoot 'private-logs'
    New-Item -ItemType Directory -Force -Path $logRoot | Out-Null
    $version = (Get-Content (Join-Path $root 'VERSION') -Raw).Trim()
    $script:results = New-Object 'System.Collections.Generic.List[object]'
    $script:activeLog = $null
    $fixture = Join-Path $outputRoot ('sandbox-' + [guid]::NewGuid().ToString('N'))
    $binary = Join-Path $fixture 'bin\nexus.exe'
    $originalEnvironment = @{}
    foreach ($key in @('NEXUS_DATA_DIR', 'NEXUS_CONFIG_DIR', 'NEXUS_STATE_DIR', 'CODEX_HOME', 'PATH')) {
        $originalEnvironment[$key] = [Environment]::GetEnvironmentVariable($key, 'Process')
    }
    function Add-Gate {
        param([string]$Id, [string]$Status, [string]$Detail)
        $script:results.Add([pscustomobject]@{ id=$Id; status=$Status; detail=$Detail })
        Write-Host "[$Status] $Id - $Detail"
    }
    function Run-Gate {
        param([string]$Id, [scriptblock]$Action)
        $script:activeLog = Join-Path $logRoot ($Id + '.log')
        $timer = [System.Diagnostics.Stopwatch]::StartNew()
        try {
            & $Action
            Add-Gate $Id 'PASS' ("completed in " + $timer.ElapsedMilliseconds + ' ms')
        } catch {
            # Do not put raw stderr, account data or paths in the public JSON.
            Add-Gate $Id 'FAIL' ("see private-logs/" + $Id + '.log')
            $_.Exception.ToString() | Out-File -LiteralPath $script:activeLog -Append -Encoding utf8
        } finally {
            $timer.Stop()
        }
    }
    function Run-Native {
        param([string]$Executable, [string[]]$CommandArguments)
        $savedPreference = $ErrorActionPreference
        $nativeExit = 0
        try {
            $ErrorActionPreference = 'Continue'
            $global:LASTEXITCODE = 0
            & $Executable @CommandArguments 2>&1 |
                ForEach-Object { $_.ToString() } |
                Out-File -LiteralPath $script:activeLog -Append -Encoding utf8
            $nativeExit = $LASTEXITCODE
        } finally {
            $ErrorActionPreference = $savedPreference
        }
        if ($nativeExit -ne 0) {
            throw "$Executable exited with code $nativeExit"
        }
    }
    function Probe-Nexus {
        param([string[]]$CommandArguments)
        $savedPreference = $ErrorActionPreference
        try {
            $ErrorActionPreference = 'Continue'
            $global:LASTEXITCODE = 0
            $outputLines = @(& $binary @CommandArguments 2>&1 | ForEach-Object { $_.ToString() })
            $nativeExit = $LASTEXITCODE
        } finally {
            $ErrorActionPreference = $savedPreference
        }
        return [pscustomobject]@{ code=$nativeExit; output=($outputLines -join [Environment]::NewLine) }
    }
    function Require-ExitZero {
        param($Execution, [string]$Operation)
        if ($Execution.code -ne 0) {
            throw "$Operation failed with exit code $($Execution.code)"
        }
    }
    try {
        Run-Gate 'toolchain-consistency' {
            $go = (& go version).Trim()
            if ($LASTEXITCODE -ne 0 -or $go -notmatch '\bgo1\.26\.10\b') {
                throw "Canonical patched Go 1.26.10 required"
            }
            $bun = (& bun --version).Trim()
            if ($LASTEXITCODE -ne 0 -or $bun -ne '1.3.9') {
                throw "Canonical Bun 1.3.9 required"
            }
        }
        Run-Gate 'powershell-installer-uninstaller-fixtures' {
            Run-Native 'powershell.exe' @('-NoProfile', '-ExecutionPolicy', 'Bypass', '-File', (Join-Path $root 'scripts\test-uninstall-windows.ps1'))
        }
        Run-Gate 'project-directory-regression' {
            Run-Native 'go' @('test', '-count=1', './internal/nexus', '-run', 'TestProjectApplicationServiceRejectsInvalidDirectories|TestProjectApplicationServiceCRUDPreservesCoreRules')
        }
        Run-Gate 'conpty-terminal-regression' {
            Run-Native 'go' @('test', '-count=1', './internal/control/terminal', './internal/control/host', './internal/runtime')
        }
        Run-Gate 'web-first-run-typecheck' {
            Run-Native 'bun' @('run', '--cwd', 'web', 'typecheck')
        }
        Run-Gate 'web-first-run-unit-tests' {
            Run-Native 'bun' @('run', '--cwd', 'web', 'test')
        }
        Run-Gate 'build-cli-exact-source' {
            $binDir = Split-Path -Parent $binary
            New-Item -ItemType Directory -Force -Path $binDir | Out-Null
            Run-Native 'go' @('build', '-trimpath', '-o', $binary, './cmd/nexus')
            if (-not (Test-Path -LiteralPath $binary -PathType Leaf)) {
                throw 'Built Nexus executable does not exist'
            }
        }
        if (Test-Path -LiteralPath $binary -PathType Leaf) {
            # Probe in a disposable environment with an empty Codex home.
            # Do not reuse, inspect or alter the user's configured accounts.
            New-Item -ItemType Directory -Force -Path $fixture | Out-Null
            $env:NEXUS_DATA_DIR = Join-Path $fixture 'data'
            $env:NEXUS_CONFIG_DIR = Join-Path $fixture 'config'
            $env:NEXUS_STATE_DIR = Join-Path $fixture 'state'
            $env:CODEX_HOME = Join-Path $fixture 'empty-codex-home'
            New-Item -ItemType Directory -Force -Path $env:CODEX_HOME | Out-Null
            Run-Gate 'read-only-provider-discovery' {
                $probe = Probe-Nexus @('providers', 'status', '--json')
                Require-ExitZero $probe 'Provider discovery'
                $parsed = $probe.output | ConvertFrom-Json
                if ($null -eq $parsed) { throw 'Provider JSON was empty or invalid' }
            }
            Run-Gate 'windows-project-registration-and-persistence' {
                $directory = Join-Path $fixture ('Project With Spaces ' + [char]0x00E7 + 'ao')
                New-Item -ItemType Directory -Force -Path $directory | Out-Null
                $created = Probe-Nexus @('projects', 'add', $directory, '--name', 'Windows First Run Fixture')
                Require-ExitZero $created 'Valid project creation'
                $listed = Probe-Nexus @('projects', 'list')
                Require-ExitZero $listed 'Project listing'
                if ($listed.output -notmatch 'Windows First Run Fixture') {
                    throw 'Persisted project was missing from a new CLI process'
                }
                $missing = Join-Path $fixture 'does-not-exist'
                $rejected = Probe-Nexus @('projects', 'add', $missing, '--name', 'Invalid Project')
                if ($rejected.code -eq 0) { throw 'CLI accepted a nonexistent directory' }
                $file = Join-Path $fixture 'not-a-folder.txt'
                [System.IO.File]::WriteAllText($file, 'fixture')
                $rejectedFile = Probe-Nexus @('projects', 'add', $file, '--name', 'Invalid File')
                if ($rejectedFile.code -eq 0) { throw 'CLI accepted a regular file as a directory' }
                $after = Probe-Nexus @('projects', 'list')
                Require-ExitZero $after 'Final project listing'
                if ($after.output -match 'Invalid Project|Invalid File') {
                    throw 'A rejected project was persisted'
                }
            }
        } else {
            Add-Gate 'read-only-provider-discovery' 'BLOCKED' 'CLI build did not complete'
            Add-Gate 'windows-project-registration-and-persistence' 'BLOCKED' 'CLI build did not complete'
        }
        # Explicitly untested native acceptance criteria. Never infer these from
        # unit tests, synthetic screenshots or a successful CLI compilation.
        Add-Gate 'desktop-wails-first-run' 'NOT_RUN' 'Requires native desktop interaction and screenshots'
        Add-Gate 'codex-conpty-mouse-scroll-live' 'NOT_RUN' 'Requires prolonged real Codex interactive session'
        Add-Gate 'multiple-authenticated-codex-accounts' 'NOT_RUN' 'Requires approved live multi-account test'
        Add-Gate 'signed-installer-install-update-rollback' 'NOT_RUN' 'Requires signed distributable and isolated Windows test'
    } finally {
        foreach ($key in $originalEnvironment.Keys) {
            [Environment]::SetEnvironmentVariable($key, $originalEnvironment[$key], 'Process')
        }
        if (Test-Path -LiteralPath $fixture) {
            Remove-Item -LiteralPath $fixture -Recurse -Force -ErrorAction SilentlyContinue
        }
    }
    $failures = @($script:results | Where-Object { $_.status -eq 'FAIL' -or $_.status -eq 'BLOCKED' })
    $automated = if ($failures.Count -eq 0) { 'PASS' } else { 'FAIL' }
    $summary = [ordered]@{
        schema_version = 1
        source_sha = $sha
        branch = $branchName
        version = $version
        generated_at_utc = [DateTime]::UtcNow.ToString('o')
        automated_result = $automated
        release_verdict = 'NO_GO_WINDOWS'
        release_note = 'Native UI, signed distribution and authenticated Codex interaction require separate exact-SHA evidence'
        gates = @($script:results.ToArray())
    }
    $summaryFile = Join-Path $outputRoot 'WINDOWS_FIRST_RUN_SUMMARY.json'
    $summary | ConvertTo-Json -Depth 8 | Set-Content -LiteralPath $summaryFile -Encoding utf8
    Write-Host "Sanitized summary: $summaryFile"
    Write-Host "Raw local logs in private-logs must be reviewed before sharing."
    if ($failures.Count -gt 0) { exit 1 }
} finally {
    Pop-Location
}