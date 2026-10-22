[CmdletBinding()]
param()

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

$repoRoot = Split-Path -Parent $PSScriptRoot
$uninstaller = Join-Path $repoRoot 'uninstall.ps1'
$tokens = $null
$parseErrors = $null
$ast = [System.Management.Automation.Language.Parser]::ParseFile($uninstaller, [ref]$tokens, [ref]$parseErrors)
if ($parseErrors.Count -gt 0) {
    throw "uninstall.ps1 parse failed: $($parseErrors -join '; ')"
}
$pathFunction = $ast.Find({
    param($node)
    $node -is [System.Management.Automation.Language.FunctionDefinitionAst] -and
        $node.Name -eq 'Get-NexusPathWithoutInstallDir'
}, $true)
if ($null -eq $pathFunction) {
    throw 'Get-NexusPathWithoutInstallDir was not found in uninstall.ps1.'
}
Invoke-Expression $pathFunction.Extent.Text

function Assert-Equal {
    param([string]$Expected, [string]$Actual, [string]$Name)
    if ($Expected -cne $Actual) {
        throw "$Name failed. Expected [$Expected], got [$Actual]."
    }
}

function Assert-True {
    param([bool]$Condition, [string]$Name)
    if (-not $Condition) { throw "$Name failed." }
}

function Get-InstallerFunctionDefinition {
    param([string]$Name)
    $source = Join-Path $repoRoot 'install.ps1'
    $tokens = $null
    $errors = $null
    $sourceAst = [System.Management.Automation.Language.Parser]::ParseFile($source, [ref]$tokens, [ref]$errors)
    if ($errors.Count -gt 0) { throw "install.ps1 parse failed: $($errors -join '; ')" }
    $definition = $sourceAst.Find({
        param($node)
        $node -is [System.Management.Automation.Language.FunctionDefinitionAst] -and $node.Name -eq $Name
    }, $true)
    if ($null -eq $definition) { throw "$Name was not found in install.ps1." }
    return $definition.Extent.Text
}

$path = 'D:\Programs\Nexus2;D:\Programs\Nexus;D:\Tools'
Assert-Equal 'D:\Programs\Nexus2;D:\Tools' `
    (Get-NexusPathWithoutInstallDir -UserPath $path -InstallDir 'D:\Programs\Nexus') `
    'PATH segment comparison and similar directory preservation'

$path = 'D:\Programs\Nexus;D:\Tools;D:\Programs\Nexus'
Assert-Equal 'D:\Tools' `
    (Get-NexusPathWithoutInstallDir -UserPath $path -InstallDir 'D:\Programs\Nexus') `
    'duplicate Nexus PATH segment removal'

$path = ';D:\Tools;;D:\Programs\Nexus;'
Assert-Equal ';D:\Tools;;' `
    (Get-NexusPathWithoutInstallDir -UserPath $path -InstallDir 'D:\Programs\Nexus') `
    'empty PATH segment preservation'

$env:NEXUS_UNINSTALL_TEST_ROOT = Join-Path $env:TEMP "nexus-uninstall-path-$PID"
$expandedEntry = '%NEXUS_UNINSTALL_TEST_ROOT%\Nexus'
Assert-Equal 'D:\Tools' `
    (Get-NexusPathWithoutInstallDir -UserPath "$expandedEntry;D:\Tools" -InstallDir (Join-Path $env:NEXUS_UNINSTALL_TEST_ROOT 'Nexus')) `
    'expanded variable PATH segment comparison'
Remove-Item Env:NEXUS_UNINSTALL_TEST_ROOT

$installer = Join-Path $repoRoot 'install.ps1'
$installTokens = $null
$installParseErrors = $null
$installAst = [System.Management.Automation.Language.Parser]::ParseFile(
    $installer, [ref]$installTokens, [ref]$installParseErrors
)
if ($installParseErrors.Count -gt 0) {
    throw "install.ps1 parse failed: $($installParseErrors -join '; ')"
}
$installPathFunction = $installAst.Find({
    param($node)
    $node -is [System.Management.Automation.Language.FunctionDefinitionAst] -and
        $node.Name -eq 'Assert-NexusTargetDirectoryIsNotVolumeRoot'
}, $true)
if ($null -eq $installPathFunction) {
    throw 'Assert-NexusTargetDirectoryIsNotVolumeRoot was not found in install.ps1.'
}
Invoke-Expression $installPathFunction.Extent.Text
Assert-NexusTargetDirectoryIsNotVolumeRoot -Path (Join-Path $env:TEMP 'nexus-safe-install')
$installerRefusedRoot = $false
try {
    Assert-NexusTargetDirectoryIsNotVolumeRoot -Path ([System.IO.Path]::GetPathRoot($env:TEMP))
} catch {
    $installerRefusedRoot = $true
}
Assert-True $installerRefusedRoot 'installer volume root refusal'

Invoke-Expression (Get-InstallerFunctionDefinition 'Resolve-LatestVersion')
Invoke-Expression (Get-InstallerFunctionDefinition 'Get-NexusReleaseCoordinate')
Invoke-Expression (Get-InstallerFunctionDefinition 'Assert-NexusManifestMatchesRequestedRelease')
Invoke-Expression (Get-InstallerFunctionDefinition 'Test-NexusArtifactSha256')
Invoke-Expression (Get-InstallerFunctionDefinition 'Extract-ShaFromManifest')
$GithubApi = 'https://fixture.invalid'
function Invoke-RestMethod {
    param([string]$Uri, [hashtable]$Headers)
    if ($Uri -ne 'https://fixture.invalid/releases/latest') { throw "Unexpected fixture URI: $Uri" }
    return [pscustomobject]@{ tag_name = 'v1.0.0-beta.10' }
}
Assert-Equal 'v1.0.0-beta.10' (Resolve-LatestVersion) 'offline latest tag resolution'

$betaWithV = Get-NexusReleaseCoordinate -RequestedVersion 'v1.0.0-beta.10'
Assert-Equal '1.0.0-beta.10' $betaWithV.Version 'v-prefixed beta normalization'
Assert-Equal 'channel-beta' $betaWithV.ReleaseTag 'beta rolling channel resolution'
Assert-Equal 'beta' $betaWithV.ManifestChannel 'beta manifest channel resolution'
$betaWithoutV = Get-NexusReleaseCoordinate -RequestedVersion '1.0.0-beta.10'
Assert-Equal $betaWithV.ReleaseTag $betaWithoutV.ReleaseTag 'unprefixed beta channel resolution'
$stable = Get-NexusReleaseCoordinate -RequestedVersion 'v1.0.0'
Assert-Equal 'v1.0.0' $stable.ReleaseTag 'stable immutable tag resolution'
Assert-Equal 'stable' $stable.ManifestChannel 'stable manifest channel resolution'
$invalidVersionRejected = $false
try { Get-NexusReleaseCoordinate -RequestedVersion 'latest' | Out-Null } catch { $invalidVersionRejected = $true }
Assert-True $invalidVersionRejected 'latest must be resolved before coordinate parsing'

$matchingManifest = [pscustomobject]@{ version = '1.0.0-beta.10'; channel = 'beta' }
Assert-NexusManifestMatchesRequestedRelease -Manifest $matchingManifest -Version '1.0.0-beta.10' -Channel 'beta'
$olderBetaRejected = $false
try {
    Assert-NexusManifestMatchesRequestedRelease `
        -Manifest ([pscustomobject]@{ version = '1.0.0-beta.9'; channel = 'beta' }) `
        -Version '1.0.0-beta.10' -Channel 'beta'
} catch { $olderBetaRejected = $true }
Assert-True $olderBetaRejected 'advanced beta pointer version mismatch rejection'
$wrongChannelRejected = $false
try {
    Assert-NexusManifestMatchesRequestedRelease `
        -Manifest ([pscustomobject]@{ version = '1.0.0-beta.10'; channel = 'stable' }) `
        -Version '1.0.0-beta.10' -Channel 'beta'
} catch { $wrongChannelRejected = $true }
Assert-True $wrongChannelRejected 'signed manifest channel mismatch rejection'
Assert-True (Test-NexusArtifactSha256 -Expected ('a' * 64) -Actual ('A' * 64)) 'case-insensitive SHA-256 match'
Assert-True (-not (Test-NexusArtifactSha256 -Expected ('b' * 64) -Actual ('a' * 64))) 'wrong SHA-256 rejection'

$pathInstallDir = 'C:\Fixture\Nexus'
$pathWithDuplicates = "C:\Tools;$pathInstallDir;C:\Fixture\Nexus2;$pathInstallDir\;C:\Other"
$pathWithoutInstallDir = Get-NexusPathWithoutInstallDir -UserPath $pathWithDuplicates -InstallDir $pathInstallDir
Assert-Equal 'C:\Tools;C:\Fixture\Nexus2;C:\Other' $pathWithoutInstallDir `
    'remove only complete case-insensitive path segments and preserve similar paths'
$previousPathFixture = [Environment]::GetEnvironmentVariable('NEXUS_UNINSTALL_PATH_FIXTURE', 'Process')
try {
    [Environment]::SetEnvironmentVariable('NEXUS_UNINSTALL_PATH_FIXTURE', 'C:\Fixture\Nexus', 'Process')
    $expandedPath = Get-NexusPathWithoutInstallDir `
        -UserPath '%NEXUS_UNINSTALL_PATH_FIXTURE%;C:\Keep' -InstallDir $pathInstallDir
    Assert-Equal 'C:\Keep' $expandedPath 'remove matching expanded environment path segment'
} finally {
    [Environment]::SetEnvironmentVariable('NEXUS_UNINSTALL_PATH_FIXTURE', $previousPathFixture, 'Process')
}

$manifestFixturePath = Join-Path $env:TEMP "nexus-installer-manifest-$PID.json"
$manifestFixture = [ordered]@{
    version = '1.0.0-beta.10'
    channel = 'beta'
    artifacts = [ordered]@{
        nexus_windows_x86_64_shadow = @{
            url = 'https://fixture.invalid/download/othernexus_Windows_x86_64.zip'
            sha256 = ('b' * 64)
        }
    }
}
$manifestFixture | ConvertTo-Json -Depth 5 | Set-Content -LiteralPath $manifestFixturePath -Encoding utf8
try {
    Assert-Equal '' (Extract-ShaFromManifest -ManifestPath $manifestFixturePath -ArtifactName 'nexus_Windows_x86_64.zip') `
        'similar but missing artifact rejection'
    $manifestFixture.artifacts['nexus_windows_x86_64'] = @{
        url = 'https://fixture.invalid/download/nexus_Windows_x86_64.zip'
        sha256 = ('a' * 64)
    }
    $manifestFixture | ConvertTo-Json -Depth 5 | Set-Content -LiteralPath $manifestFixturePath -Encoding utf8
    Assert-Equal ('a' * 64) `
        (Extract-ShaFromManifest -ManifestPath $manifestFixturePath -ArtifactName 'nexus_Windows_x86_64.zip') `
        'exact manifest artifact resolution'
} finally {
    Remove-Item -LiteralPath $manifestFixturePath -Force -ErrorAction SilentlyContinue
}

$testRoot = Join-Path $env:TEMP "nexus-uninstall-smoke-$([Guid]::NewGuid().ToString('N'))"
$unknownDir = Join-Path $testRoot 'unknown-install'
$managedDir = Join-Path $testRoot 'managed-install'
$emptyDir = Join-Path $testRoot 'empty-install'
New-Item -ItemType Directory -Path $unknownDir, $managedDir, $emptyDir -Force | Out-Null

try {
    $unknownFile = Join-Path $unknownDir 'valuable-data.txt'
    Set-Content -LiteralPath $unknownFile -Value 'preserve this file' -NoNewline
    $refusedUnknown = $false
    try {
        & $uninstaller -InstallDir $unknownDir -SkipPathCleanup -SkipDesktopShortcut
    } catch {
        $refusedUnknown = $true
    }
    Assert-True $refusedUnknown 'unknown directory refusal'
    Assert-True (Test-Path -LiteralPath $unknownFile -PathType Leaf) 'unknown directory data preservation'

    Set-Content -LiteralPath (Join-Path $managedDir '.nexus-managed') `
        -Value 'IAPro Nexus managed installation' -NoNewline
    Set-Content -LiteralPath (Join-Path $managedDir 'nexus.exe') -Value 'fixture binary' -NoNewline
    Set-Content -LiteralPath (Join-Path $managedDir 'project.db') -Value 'fixture user data' -NoNewline
    & $uninstaller -InstallDir $managedDir -SkipPathCleanup -SkipDesktopShortcut
    Assert-True (-not (Test-Path -LiteralPath (Join-Path $managedDir 'nexus.exe'))) 'known binary removal'
    Assert-True (Test-Path -LiteralPath (Join-Path $managedDir 'project.db') -PathType Leaf) 'project data preservation'
    Assert-True (Test-Path -LiteralPath $managedDir -PathType Container) 'unknown file keeps install directory'

    Set-Content -LiteralPath (Join-Path $emptyDir '.nexus-managed') `
        -Value 'IAPro Nexus managed installation' -NoNewline
    Set-Content -LiteralPath (Join-Path $emptyDir 'nexus.exe') -Value 'fixture binary' -NoNewline
    & $uninstaller -InstallDir $emptyDir -SkipPathCleanup -SkipDesktopShortcut
    Assert-True (-not (Test-Path -LiteralPath $emptyDir)) 'empty managed directory removal'
    & $uninstaller -InstallDir $emptyDir -SkipPathCleanup -SkipDesktopShortcut

    $volumeRoot = [System.IO.Path]::GetPathRoot($testRoot)
    $refusedRoot = $false
    try {
        & $uninstaller -InstallDir $volumeRoot -SkipPathCleanup -SkipDesktopShortcut
    } catch {
        $refusedRoot = $true
    }
    Assert-True $refusedRoot 'volume root refusal'

    Write-Output 'Windows uninstaller isolation smoke: PASS'
} finally {
    Remove-Item -LiteralPath $testRoot -Recurse -Force -ErrorAction SilentlyContinue
}
