# IAPro Nexus Installer for PowerShell (Windows & PowerShell Core)
# Prebuilt signed-release path. Usage:
#   .\install.ps1 -Version v0.5.0-beta.23
#   .\install.ps1 -Version latest
#   .\install.ps1 -BuildFromSource

param(
    [switch]$WithMaestro = $false,
    [switch]$NoDesktop = $false,
    [string]$InstallDir = $env:NEXUS_INSTALL_DIR,
    [string]$Version = $env:NEXUS_VERSION,
    [switch]$BuildFromSource = $false,
    [string]$SourceRef = $env:NEXUS_SOURCE_REF,
    [switch]$Yes = $false,
    [switch]$InstallPrerequisites = $false
)

$ErrorActionPreference = 'Stop'
$InstallDesktop = -not $NoDesktop
$MaestroExit = 0

$Repo = if ($env:NEXUS_RELEASE_REPO) { $env:NEXUS_RELEASE_REPO } else { "IAPro-Community/nexus-releases" }
$GithubUrl = "https://github.com/$Repo"
$GithubApi = "https://api.github.com/repos/$Repo"
$ReleaseDownloadBase = if ($env:NEXUS_RELEASE_BASE_URL) { $env:NEXUS_RELEASE_BASE_URL.TrimEnd('/') } else { "$GithubUrl/releases/download" }
# Source checkouts (-SourceRef builds) always come from the private source
# repository, never from the binaries mirror.
$SourceRepo = if ($env:NEXUS_SOURCE_REPO) { $env:NEXUS_SOURCE_REPO } else { "IAPro-Community/nexus" }
$SourceUrl = "https://github.com/$SourceRepo"

$IsWindowsOS = if (Test-Path Variable:IsWindows) { [bool]$IsWindows } else { ($env:OS -like "*Windows*") }

function Ensure-GoCompiler {
    if (Get-Command go -ErrorAction SilentlyContinue) {
        return
    }

    if (-not $IsWindowsOS) {
        throw "Go >=1.25 is required for source builds. Install Go manually and run the installer again."
    }

    if (-not $InstallPrerequisites) {
        throw "Go is missing. Install it from https://go.dev/dl/ or explicitly allow optional dependency installation with -InstallPrerequisites. A prebuilt Nexus release does not need Go."
    }
    $Winget = Get-Command winget -ErrorAction SilentlyContinue
    if (-not $Winget) {
        throw "Go >=1.25 is required for source builds, but winget is unavailable. Install Go from https://go.dev/dl/ and run the installer again."
    }

    Write-Host "Go was not found. Installing the official Go package with WinGet..." -ForegroundColor Yellow
    & $Winget.Source install --id GoLang.Go --exact --accept-source-agreements --accept-package-agreements
    if ($LASTEXITCODE -ne 0) {
        throw "WinGet could not install Go. Install Go from https://go.dev/dl/ and run the installer again."
    }

    $MachinePath = [Environment]::GetEnvironmentVariable("Path", "Machine")
    $UserPath = [Environment]::GetEnvironmentVariable("Path", "User")
    $env:Path = "$UserPath;$MachinePath"
    if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
        throw "Go was installed but is not available in PATH yet. Restart PowerShell and run the installer again."
    }
    Write-Host "✓ Go compiler is available." -ForegroundColor Green
}

function Ensure-Bun {
    if (Get-Command bun -ErrorAction SilentlyContinue) {
        return
    }
    if (-not $InstallPrerequisites) {
        throw "Bun is missing. Install it from https://bun.sh or explicitly allow optional dependency installation with -InstallPrerequisites. A prebuilt Nexus release does not need Bun."
    }
    Write-Host "Bun was not found. Installing via official Bun script (explicitly authorized)..." -ForegroundColor Yellow
    try {
        Invoke-RestMethod -Uri "https://bun.sh/install.ps1" | Invoke-Expression
    } catch {
        throw "Bun >=1.3.9 is required for source builds. Install from https://bun.sh and re-run. $_"
    }
    $BunHome = Join-Path $HOME ".bun\bin"
    if (Test-Path $BunHome) {
        $env:Path = "$BunHome;$env:Path"
    }
    if (-not (Get-Command bun -ErrorAction SilentlyContinue)) {
        throw "Bun was installed but is not on PATH yet. Restart PowerShell and run the installer again."
    }
    Write-Host "✓ Bun is available." -ForegroundColor Green
}

function Resolve-LatestVersion {
    $url = "$GithubApi/releases/latest"
    $attempt = 0
    while ($attempt -lt 3) {
        $attempt++
        try {
            $resp = Invoke-RestMethod -Uri $url -Headers @{ "User-Agent" = "iapro-nexus-installer" }
            if (-not $resp.tag_name) {
                throw "GitHub API response did not contain tag_name."
            }
            Write-Host "Resolved -Version latest to $($resp.tag_name); installation still requires the signed update manifest." -ForegroundColor Yellow
            return [string]$resp.tag_name
        } catch {
            if ($attempt -ge 3) { throw }
            Write-Host "GitHub API resolve failed (attempt $attempt/3): $_" -ForegroundColor Yellow
            Start-Sleep -Seconds ($attempt * 2)
        }
    }
}

function Get-NexusReleaseCoordinate {
    param([Parameter(Mandatory)][string]$RequestedVersion)

    if ($RequestedVersion -notmatch '^v?[0-9]+\.[0-9]+\.[0-9]+([.-][A-Za-z0-9.-]+)?$') {
        throw "Invalid Nexus version: $RequestedVersion"
    }
    $Version = $RequestedVersion.TrimStart('v')
    $IsPrerelease = $Version -match '-(?:alpha|beta|rc)\.'
    [pscustomobject]@{
        Version = $Version
        ReleaseTag = if ($IsPrerelease) { 'channel-beta' } else { "v$Version" }
        ManifestChannel = if ($IsPrerelease) { 'beta' } else { 'stable' }
    }
}

function Assert-NexusManifestMatchesRequestedRelease {
    param(
        [Parameter(Mandatory)]$Manifest,
        [Parameter(Mandatory)][string]$Version,
        [Parameter(Mandatory)][string]$Channel
    )

    if ([string]$Manifest.version -ne $Version -or [string]$Manifest.channel -ne $Channel) {
        throw "Signed manifest version/channel mismatch: requested $Version ($Channel), received $($Manifest.version) ($($Manifest.channel))."
    }
}

function Test-NexusArtifactSha256 {
    param([Parameter(Mandatory)][string]$Expected, [Parameter(Mandatory)][string]$Actual)

    return $Expected -match '^[0-9a-fA-F]{64}$' -and
        [string]::Equals($Expected, $Actual, [StringComparison]::OrdinalIgnoreCase)
}

function Ensure-WebView2 {
    if (-not $IsWindowsOS) { return }
    $marker = @(
        "${env:ProgramFiles(x86)}\Microsoft\EdgeWebView\Application",
        "$env:ProgramFiles\Microsoft\EdgeWebView\Application"
    ) | Where-Object { Test-Path $_ }
    if ($marker) { return }

    Write-Host "⚠️  WebView2 runtime not detected. Desktop may fail; CLI remains usable." -ForegroundColor Yellow
    $Winget = Get-Command winget -ErrorAction SilentlyContinue
    if (-not $InstallPrerequisites) {
        Write-Host "WebView2 was not installed automatically. Install it from https://developer.microsoft.com/microsoft-edge/webview2/ or rerun with -InstallPrerequisites to authorize WinGet." -ForegroundColor Yellow
        return
    }
    if ($Winget) {
        Write-Host "Installing WebView2 via WinGet (explicitly authorized)..." -ForegroundColor Yellow
        & $Winget.Source install --id Microsoft.EdgeWebView2Runtime --exact --accept-source-agreements --accept-package-agreements
        if ($LASTEXITCODE -ne 0) { Write-Host "WebView2 installation did not succeed; use the official installer." -ForegroundColor Yellow }
    } else {
        Write-Host "Install WebView2 from https://developer.microsoft.com/microsoft-edge/webview2/" -ForegroundColor Yellow
    }
}

Write-Host "=== IAPro Nexus Installer (Signed Prebuilt Release) ===" -ForegroundColor Cyan
Write-Host "Release repo: $Repo" -ForegroundColor Gray

if ([string]::IsNullOrWhiteSpace($Version) -and -not $BuildFromSource) {
    throw "A version is required for verified installation. Use -Version vX.Y.Z, -Version latest, or -BuildFromSource from a checkout."
}

if ($Version -eq "latest") {
    $Version = Resolve-LatestVersion
}

if (-not [string]::IsNullOrWhiteSpace($Version)) {
    $ReleaseCoordinate = Get-NexusReleaseCoordinate -RequestedVersion $Version
    $Version = $ReleaseCoordinate.Version
} else {
    $ReleaseCoordinate = $null
}

$BinaryName = if ($IsWindowsOS) { "nexus.exe" } else { "nexus" }
$AiAliasName = if ($IsWindowsOS) { "ai.exe" } else { "ai" }

function Assert-NexusTargetDirectoryIsNotVolumeRoot {
    param([Parameter(Mandatory)][string]$Path)

    $FullPath = [System.IO.Path]::GetFullPath($Path)
    $VolumeRoot = [System.IO.Path]::GetPathRoot($FullPath)
    if ([string]::Equals(
        $FullPath.TrimEnd('\'), $VolumeRoot.TrimEnd('\'),
        [StringComparison]::OrdinalIgnoreCase
    )) {
        throw "Refusing to install Nexus to a volume root: $FullPath"
    }
}

if (-not [string]::IsNullOrWhiteSpace($InstallDir)) {
    $TargetDir = [System.IO.Path]::GetFullPath($InstallDir)
    $LegacyTargetDir = ""
} elseif ($IsWindowsOS) {
    $TargetDir = Join-Path $env:LOCALAPPDATA "Programs\IAPro Nexus"
    $LegacyTargetDir = Join-Path $env:LOCALAPPDATA "Programs\ai-cli"
} else {
    $TargetDir = Join-Path $HOME ".local/bin"
    $LegacyTargetDir = ""
}

Assert-NexusTargetDirectoryIsNotVolumeRoot -Path $TargetDir

if (-not (Test-Path $TargetDir)) {
    New-Item -ItemType Directory -Path $TargetDir -Force | Out-Null
}

if ($IsWindowsOS -and (-not [string]::IsNullOrWhiteSpace($LegacyTargetDir)) -and (Test-Path $LegacyTargetDir)) {
    Write-Host "Legacy ai-cli installation found at $LegacyTargetDir; leaving it untouched while installing IAPro Nexus to $TargetDir." -ForegroundColor Gray
}

$TargetPath = Join-Path $TargetDir $BinaryName
$Installed = $false

# Ed25519 public key for manifest signature verification (matches internal/update ProductionTrustRoot).
$NexusPubKey = "ee2e268394ce6a0634a9ea2f414ee831dd536d12f7e2b80107f94566e58aa334"

# PowerShell 5.1 converts a native child's stderr into a terminating
# NativeCommandError while ErrorActionPreference is Stop — for example the
# banner `bun run` prints for its scripts. Run build tools through this helper
# so their output streams as text and $LASTEXITCODE stays the single source of
# truth for the caller's error checks.
function Invoke-NativeBuild([scriptblock]$Run) {
    $Saved = $ErrorActionPreference
    $ErrorActionPreference = 'Continue'
    try {
        & $Run 2>&1 | ForEach-Object { $_.ToString() }
    } finally {
        $ErrorActionPreference = $Saved
    }
}

function Build-NexusFromSource {
    param([string]$RootPath)

    Push-Location $RootPath
    try {
        if (-not (Test-Path "./go.mod") -or -not (Test-Path "./cmd/nexus") -or -not (Test-Path "./web")) {
            throw "Source checkout is missing go.mod, cmd/nexus, or web."
        }

        $ModuleLine = Get-Content -LiteralPath "./go.mod" | Where-Object { $_ -match '^module\s+' } | Select-Object -First 1
        if ($ModuleLine -ne "module github.com/IAPro-Community/nexus") {
            throw "Refusing source build from an unrelated Go module: $ModuleLine"
        }

        Write-Host "Installing locked frontend dependencies..." -ForegroundColor Yellow
        Invoke-NativeBuild { & bun install --cwd web --frozen-lockfile }
        if ($LASTEXITCODE -ne 0) { throw "bun install failed during source build." }

        Write-Host "Building frontend bundle..." -ForegroundColor Yellow
        Invoke-NativeBuild { & bun run --cwd web build }
        if ($LASTEXITCODE -ne 0) { throw "frontend build failed during source build." }

        $SourceVersion = (Get-Content VERSION -Raw).Trim()
        if ([string]::IsNullOrWhiteSpace($SourceVersion)) {
            throw "VERSION is empty in source checkout."
        }

        $Commit = "unknown"
        if (Get-Command git -ErrorAction SilentlyContinue) {
            $ResolvedCommit = (& git rev-parse --short HEAD 2>$null)
            if ($LASTEXITCODE -eq 0 -and $ResolvedCommit) {
                $Commit = ([string]$ResolvedCommit).Trim()
            }
        }
        $BuildDate = [DateTime]::UtcNow.ToString("yyyy-MM-ddTHH:mm:ssZ")
        $Module = "github.com/IAPro-Community/nexus"
        $Ldflags = "-s -w -X $Module/internal/buildinfo.Version=$SourceVersion -X $Module/internal/buildinfo.Commit=$Commit -X $Module/internal/buildinfo.BuildDate=$BuildDate -X $Module/internal/update.ProductionTrustRootHex=$NexusPubKey"

        Write-Host "Building Nexus $SourceVersion (commit: $Commit) with Go..." -ForegroundColor Yellow
        Invoke-NativeBuild { & go build -trimpath -buildvcs=false -ldflags $Ldflags -o $BinaryName ./cmd/nexus }
        if ($LASTEXITCODE -ne 0) { throw "go build failed during source build." }

        $Built = Join-Path (Get-Location) $BinaryName
        if (-not (Test-Path $Built)) {
            throw "source build did not produce $BinaryName."
        }

        $VersionJson = (& $Built version --json 2>$null) -join "`n"
        try {
            $EmbeddedVersion = ($VersionJson | ConvertFrom-Json).version
        } catch {
            throw "source-built binary could not report valid version metadata."
        }
        if ($EmbeddedVersion -ne $SourceVersion) {
            throw "source-built binary version mismatch: expected $SourceVersion, got $EmbeddedVersion"
        }

        Copy-Item -Path $Built -Destination $TargetPath -Force
    } finally {
        Pop-Location
    }
}

function Verify-ManifestSignature {
    param([string]$ManifestPath, [string]$SigPath)
    if (-not (Test-Path $SigPath)) {
        Write-Host "ERROR: update-manifest.sig missing — refusing unsigned install" -ForegroundColor Red
        return $false
    }
    $SigHex = (Get-Content $SigPath -Raw).Trim().Replace("`n","").Replace("`r","")
    if ([string]::IsNullOrWhiteSpace($SigHex)) {
        Write-Host "ERROR: update-manifest.sig is empty — refusing unsigned install" -ForegroundColor Red
        return $false
    }
    if (Get-Command python3 -ErrorAction SilentlyContinue) {
        python3 -c @"
from cryptography.hazmat.primitives.asymmetric.ed25519 import Ed25519PublicKey
pub = Ed25519PublicKey.from_public_bytes(bytes.fromhex('$NexusPubKey'))
sig = bytes.fromhex('$SigHex')
data = open(r'$ManifestPath', 'rb').read()
pub.verify(sig, data)
"@ 2>&1 | Out-Null
        if ($LASTEXITCODE -eq 0) {
            Write-Host "Manifest signature VERIFIED (Ed25519)" -ForegroundColor Green
            return $true
        }
        Write-Host "ERROR: manifest signature verification failed" -ForegroundColor Red
        return $false
    }
    Write-Host "ERROR: python3+cryptography required to verify Ed25519 release manifests" -ForegroundColor Red
    return $false
}

function Extract-ShaFromManifest {
    param([string]$ManifestPath, [string]$ArtifactName)
    $Manifest = Get-Content -LiteralPath $ManifestPath -Raw | ConvertFrom-Json
    if ($null -eq $Manifest.artifacts) { return "" }

    foreach ($Artifact in $Manifest.artifacts.PSObject.Properties) {
        $Url = [string]$Artifact.Value.url
        $UrlPath = $Url
        $ParsedUri = $null
        if ([Uri]::TryCreate($Url, [UriKind]::Absolute, [ref]$ParsedUri)) {
            $UrlPath = [Uri]::UnescapeDataString($ParsedUri.AbsolutePath)
        }
        if ($UrlPath.EndsWith("/$ArtifactName", [StringComparison]::OrdinalIgnoreCase)) {
            return [string]$Artifact.Value.sha256
        }
    }

    $Key = $ArtifactName.ToLowerInvariant().Replace('-', '_')
    if ($Key.EndsWith('.tar.gz', [StringComparison]::Ordinal)) {
        $Key = $Key.Substring(0, $Key.Length - 7)
    } else {
        $Extension = [System.IO.Path]::GetExtension($Key)
        if ($Extension) { $Key = $Key.Substring(0, $Key.Length - $Extension.Length) }
    }
    foreach ($Artifact in $Manifest.artifacts.PSObject.Properties) {
        if ([string]::Equals($Artifact.Name, $Key, [StringComparison]::OrdinalIgnoreCase)) {
            return [string]$Artifact.Value.sha256
        }
    }
    return ""
}


$Arch = if ([System.Environment]::Is64BitOperatingSystem) {
    if ([System.Runtime.InteropServices.RuntimeInformation]::OSArchitecture -eq [System.Runtime.InteropServices.Architecture]::Arm64) {
        "arm64"
    } else {
        "x86_64"
    }
} else {
    "i386"
}

$OsName = if ($IsWindowsOS) { "Windows" } elseif ($IsMacOS) { "Darwin" } else { "Linux" }
$ArchiveExt = if ($IsWindowsOS) { "zip" } else { "tar.gz" }
$ArchiveName = "nexus_${OsName}_${Arch}.${ArchiveExt}"
# Beta/RC assets use the rolling channel-beta release; stable assets use
# immutable version releases. Both paths require the signed manifest.
$ReleaseTag = if ($ReleaseCoordinate) { $ReleaseCoordinate.ReleaseTag } else { "" }
$ExpectedManifestChannel = if ($ReleaseCoordinate) { $ReleaseCoordinate.ManifestChannel } else { "" }
$ReleaseAssetBase = "$ReleaseDownloadBase/$ReleaseTag"
$DownloadUrl = "$ReleaseAssetBase/$ArchiveName"

$TempDir = Join-Path ([System.IO.Path]::GetTempPath()) ([System.Guid]::NewGuid().ToString())
New-Item -ItemType Directory -Path $TempDir -Force | Out-Null

try {
    if (-not $BuildFromSource -and -not [string]::IsNullOrWhiteSpace($Version)) {
        Write-Host "Attempting to download Nexus v${Version}: $ArchiveName..." -ForegroundColor Yellow
        # 1) Fetch and verify signed manifest before downloading the artifact.
        $ManifestPath = Join-Path $TempDir "update-manifest.json"
        $SigPath = Join-Path $TempDir "update-manifest.sig"
        $ManifestUrl = "$ReleaseAssetBase/update-manifest.json"
        $SigUrl = "$ReleaseAssetBase/update-manifest.sig"
        Invoke-WebRequest -Uri $ManifestUrl -OutFile $ManifestPath -UseBasicParsing
        Write-Host "Signed manifest downloaded" -ForegroundColor Green
        Invoke-WebRequest -Uri $SigUrl -OutFile $SigPath -UseBasicParsing
        if (-not (Verify-ManifestSignature -ManifestPath $ManifestPath -SigPath $SigPath)) {
            throw "Release manifest signature verification failed."
        }
        # The signed rolling pointer may have moved to a different beta.
        # A pinned version MUST match exactly, even if its signature is valid.
        $VerifiedManifest = Get-Content -LiteralPath $ManifestPath -Raw | ConvertFrom-Json
        Assert-NexusManifestMatchesRequestedRelease `
            -Manifest $VerifiedManifest -Version $Version -Channel $ExpectedManifestChannel
        $Expected = Extract-ShaFromManifest -ManifestPath $ManifestPath -ArtifactName $ArchiveName
        if ([string]::IsNullOrWhiteSpace($Expected)) {
            throw "Artifact $ArchiveName missing from signed manifest"
        }
        Write-Host "Using SHA-256 from signed manifest" -ForegroundColor Green

        # 2) Download artifact
        $ZipPath = Join-Path $TempDir $ArchiveName
        $attempt = 0
        $downloaded = $false
        while ($attempt -lt 3 -and -not $downloaded) {
            $attempt++
            try {
                Invoke-WebRequest -Uri $DownloadUrl -OutFile $ZipPath -UseBasicParsing
                $downloaded = $true
            } catch {
                if ($attempt -ge 3) { throw }
                Start-Sleep -Seconds ($attempt * 2)
            }
        }

        # 3) Verify SHA-256 from signed manifest only
        $Actual = (Get-FileHash -Path $ZipPath -Algorithm SHA256).Hash.ToLowerInvariant()
        if (-not (Test-NexusArtifactSha256 -Expected $Expected -Actual $Actual)) {
            throw "Release checksum verification failed for $ArchiveName."
        }
        Write-Host "SHA-256 verified" -ForegroundColor Green

        if ($IsWindowsOS) {
            Expand-Archive -Path $ZipPath -DestinationPath $TempDir -Force
        } else {
            tar -xzf $ZipPath -C $TempDir
        }
        $ExtractedBin = Join-Path $TempDir $BinaryName
        if (-not (Test-Path $ExtractedBin)) {
            $ExtractedBin = Join-Path $TempDir $AiAliasName
        }
        if (Test-Path $ExtractedBin) {
            $VersionJson = (& $ExtractedBin version --json 2>$null) -join "`n"
            try {
                $EmbeddedVersion = ($VersionJson | ConvertFrom-Json).version
            } catch {
                throw "Verified artifact could not report valid version metadata."
            }
            if ([string]::IsNullOrWhiteSpace($EmbeddedVersion) -or $EmbeddedVersion -ne $Version) {
                throw "Signed artifact binary version mismatch: expected $Version, got $EmbeddedVersion"
            }
            $Staging = Join-Path $TempDir "nexus.install"
            Copy-Item -Path $ExtractedBin -Destination $Staging -Force
            Move-Item -Path $Staging -Destination $TargetPath -Force
            $Installed = $true
        }
    }
} catch {
    if (-not $BuildFromSource) {
        throw
    }
    Write-Host "Verified release unavailable; explicit source build requested." -ForegroundColor Yellow
}

if (-not $Installed -and $BuildFromSource) {
    Ensure-GoCompiler
    Ensure-Bun
    Write-Host "Building from source with Bun + Go (GNU Make not required)..." -ForegroundColor Yellow
    if ((Test-Path "./go.mod") -and (Test-Path "./cmd/nexus")) {
        Build-NexusFromSource "."
        $Installed = $true
    } elseif (-not [string]::IsNullOrWhiteSpace($SourceRef)) {
        if ($SourceRef -notmatch '^[A-Za-z0-9._/-]+$') { throw "Invalid source ref." }
        if (-not (Get-Command git -ErrorAction SilentlyContinue)) {
            throw "git is required for -SourceRef builds."
        }
        Write-Host "Fetching explicitly requested source ref $SourceRef..." -ForegroundColor Yellow
        $CloneDir = Join-Path $TempDir "repo"
        New-Item -ItemType Directory -Path $CloneDir -Force | Out-Null
        & git -C $CloneDir init -q
        if ($LASTEXITCODE -ne 0) { throw "git init failed for source build." }
        & git -C $CloneDir remote add origin "$SourceUrl.git"
        if ($LASTEXITCODE -ne 0) { throw "git remote setup failed for source build." }
        Invoke-NativeBuild { & git -C $CloneDir fetch --depth 1 origin $SourceRef }
        if ($LASTEXITCODE -ne 0) { throw "git fetch failed for source ref $SourceRef." }
        Invoke-NativeBuild { & git -C $CloneDir checkout --detach FETCH_HEAD }
        if ($LASTEXITCODE -ne 0) { throw "git checkout failed for source ref $SourceRef." }
        Build-NexusFromSource $CloneDir
        $Installed = $true
    } else {
        throw "-BuildFromSource requires a Nexus checkout or -SourceRef=<tag-or-commit>."
    }
}

if (-not $Installed) {
    throw "Installation failed: no verified release artifact was installed."
}

$AiAliasPath = Join-Path $TargetDir $AiAliasName
if (Test-Path $TargetPath) {
    Copy-Item -Path $TargetPath -Destination $AiAliasPath -Force
}

if ($WithMaestro) {
    Write-Host "`nChecking Orquestrador Maestro dependency (-WithMaestro requested)..." -ForegroundColor Yellow
    if (-not (Get-Command orquestrador-maestro -ErrorAction SilentlyContinue) -and -not (Get-Command maestro -ErrorAction SilentlyContinue)) {
        if (Get-Command npm -ErrorAction SilentlyContinue) {
            Write-Host "Installing Orquestrador Maestro CLI (@iapro/orquestrador-maestro-cli)..." -ForegroundColor Yellow
            npm install -g @iapro/orquestrador-maestro-cli
            if ($LASTEXITCODE -ne 0) {
                Write-Host "⚠️  Maestro npm install failed." -ForegroundColor Yellow
                $MaestroExit = 2
            }
        } else {
            Write-Host "⚠️  Node.js / npm not detected. Maestro unavailable." -ForegroundColor Yellow
            $MaestroExit = 2
        }
    }
} else {
    Write-Host "`nMaestro auto-install skipped (Nexus does not silently install third-party packages)." -ForegroundColor Gray
    Write-Host "To install Maestro, run with '-WithMaestro' or: npm install -g @iapro/orquestrador-maestro-cli" -ForegroundColor Gray
}

$MaestroCmd = Get-Command orquestrador-maestro -ErrorAction SilentlyContinue
if (-not $MaestroCmd) {
    $MaestroCmd = Get-Command maestro -ErrorAction SilentlyContinue
}
if ($MaestroCmd) {
    $MaestroAliasPath = Join-Path $TargetDir "maestro.cmd"
    $MaestroCommandName = $MaestroCmd.Name
    [System.IO.File]::WriteAllText($MaestroAliasPath, "@echo off`r`n$MaestroCommandName %*`r`n", [System.Text.UTF8Encoding]::new($false))
    Write-Host "✓ Linked Maestro alias ($MaestroAliasPath)" -ForegroundColor Green
}

# Uninstall only recognizes directories created by Nexus or the native NSIS
# installer. Write an ownership marker after the verified CLI has been installed.
$ManagedInstallMarker = Join-Path $TargetDir ".nexus-managed"
[System.IO.File]::WriteAllText(
    $ManagedInstallMarker,
    "IAPro Nexus managed installation`n",
    [System.Text.UTF8Encoding]::new($false)
)

Write-Host "✓ Successfully installed IAPro Nexus to $TargetPath" -ForegroundColor Green

if ($InstallDesktop -and $IsWindowsOS -and $Arch -ne "x86_64" -and -not [string]::IsNullOrWhiteSpace($Version)) {
    Write-Host "⚠️  Native Desktop is not published/certified for Windows/$Arch; CLI/Core installation remains valid." -ForegroundColor Yellow
}

if ($InstallDesktop -and $IsWindowsOS -and $Arch -eq "x86_64" -and -not [string]::IsNullOrWhiteSpace($Version)) {
    $DesktopArchiveName = "nexus-desktop_Windows_${Arch}.zip"
    $DesktopDownloadUrl = "$ReleaseAssetBase/$DesktopArchiveName"
    $DesktopZipPath = Join-Path $TempDir $DesktopArchiveName
    $ManifestPath = Join-Path $TempDir "update-manifest.json"
    $SigPath = Join-Path $TempDir "update-manifest.sig"
    $NsisName = "nexus-setup-Windows_x86_64.exe"
    try {
        Write-Host "Attempting to install the native IAPro Nexus Desktop shell..." -ForegroundColor Yellow
        if (-not (Test-Path $ManifestPath) -or -not (Test-Path $SigPath)) {
            $ManifestUrl = "$ReleaseAssetBase/update-manifest.json"
            $SigUrl = "$ReleaseAssetBase/update-manifest.sig"
            Invoke-WebRequest -Uri $ManifestUrl -OutFile $ManifestPath -UseBasicParsing
            Invoke-WebRequest -Uri $SigUrl -OutFile $SigPath -UseBasicParsing
        }
        if (-not (Verify-ManifestSignature -ManifestPath $ManifestPath -SigPath $SigPath)) {
            throw "Release manifest signature verification failed for desktop install."
        }
        $ExpectedDesktop = Extract-ShaFromManifest -ManifestPath $ManifestPath -ArtifactName $DesktopArchiveName
        if ([string]::IsNullOrWhiteSpace($ExpectedDesktop)) {
            throw "Native Desktop artifact $DesktopArchiveName missing from signed update-manifest (unsigned desktop-checksums.txt is refused)"
        }
        Invoke-WebRequest -Uri $DesktopDownloadUrl -OutFile $DesktopZipPath -UseBasicParsing
        $ActualDesktop = (Get-FileHash -Path $DesktopZipPath -Algorithm SHA256).Hash.ToLowerInvariant()
        if ($ExpectedDesktop.ToLowerInvariant() -ne $ActualDesktop) {
            throw "Native Desktop checksum verification failed for $DesktopArchiveName."
        }
        $DesktopExtractDir = Join-Path $TempDir "desktop"
        Expand-Archive -Path $DesktopZipPath -DestinationPath $DesktopExtractDir -Force
        $DesktopBinary = Join-Path $DesktopExtractDir "nexus-desktop.exe"
        if (-not (Test-Path $DesktopBinary)) { throw "Native Desktop executable missing from $DesktopArchiveName." }
        $DesktopTargetPath = Join-Path $TargetDir "nexus-desktop.exe"
        Copy-Item -Path $DesktopBinary -Destination $DesktopTargetPath -Force

        Ensure-WebView2

        $DesktopShortcutPath = Join-Path ([Environment]::GetFolderPath('Desktop')) "IAPro Nexus.lnk"
        $Shell = New-Object -ComObject WScript.Shell
        $Shortcut = $Shell.CreateShortcut($DesktopShortcutPath)
        $Shortcut.TargetPath = $DesktopTargetPath
        $Shortcut.WorkingDirectory = $TargetDir
        $Shortcut.Description = "IAPro Nexus Workspace OS"
        $Shortcut.Save()
        Write-Host "✓ Native Desktop installed to $DesktopTargetPath" -ForegroundColor Green
        Write-Host "✓ Shortcut created at $DesktopShortcutPath" -ForegroundColor Green

        # NSIS installer (optional asset): must also be listed in the signed manifest when present.
        $NsisSha = Extract-ShaFromManifest -ManifestPath $ManifestPath -ArtifactName $NsisName
        if (-not [string]::IsNullOrWhiteSpace($NsisSha)) {
            Write-Host "NSIS installer is covered by signed update-manifest ($NsisName)." -ForegroundColor Green
        }
    } catch {
        Write-Host "ERROR: Native Desktop install refused: $_" -ForegroundColor Red
        throw
    }
}

Remove-Item -Path $TempDir -Recurse -Force -ErrorAction SilentlyContinue

if ($IsWindowsOS) {
    $UserPath = [Environment]::GetEnvironmentVariable("Path", "User")
    $PathEntries = $UserPath -split ';' | Where-Object { $_ -ne "" }
    $NormalizedTargetDir = $TargetDir.TrimEnd('\')

    if (-not ($PathEntries | Where-Object { $_.TrimEnd('\') -ieq $NormalizedTargetDir })) {
        Write-Host "Adding $TargetDir to user PATH environment variable..." -ForegroundColor Yellow
        $NewPath = if ([string]::IsNullOrEmpty($UserPath)) { $TargetDir } else { "$UserPath;$TargetDir" }
        [Environment]::SetEnvironmentVariable("Path", $NewPath, "User")
        Write-Host "✓ Added $TargetDir to PATH." -ForegroundColor Green
    }
    if (-not (($env:PATH -split ';') | Where-Object { $_.TrimEnd('\') -ieq $NormalizedTargetDir })) {
        $env:PATH = "$TargetDir;$env:PATH"
    }
}

Write-Host "`nRunning nexus doctor (warnings do not fail install)..." -ForegroundColor Yellow
try {
    & $TargetPath doctor
} catch {
    Write-Host "nexus doctor reported issues (non-fatal for install)." -ForegroundColor Yellow
}

Write-Host "`nSetup complete!" -ForegroundColor Green
Write-Host "Run 'nexus doctor' to verify provider and Maestro dependencies." -ForegroundColor White
Write-Host "To start the Workspace OS:" -ForegroundColor Cyan
Write-Host "  nexus web" -ForegroundColor Cyan
Write-Host "Note: -Version latest resolves a GitHub tag, then still requires a signed update-manifest.json + Ed25519 signature." -ForegroundColor Gray

if ($MaestroExit -ne 0) {
    exit $MaestroExit
}
