# IAPro Nexus Installer for PowerShell (Windows & PowerShell Core)
# Zero-toolchain release path. Usage:
#   .\install.ps1 -Version v0.5.0-beta.23
#   .\install.ps1 -Version latest
#   .\install.ps1 -BuildFromSource

param(
    [switch]$WithMaestro = $false,
    [switch]$NoDesktop = $false,
    [string]$Version = $env:NEXUS_VERSION,
    [switch]$BuildFromSource = $false,
    [string]$SourceRef = $env:NEXUS_SOURCE_REF,
    [switch]$Yes = $false
)

$ErrorActionPreference = 'Stop'
$InstallDesktop = -not $NoDesktop
$MaestroExit = 0

$Repo = if ($env:NEXUS_RELEASE_REPO) { $env:NEXUS_RELEASE_REPO } else { "IAPro-Community/nexus-releases" }
$GithubUrl = "https://github.com/$Repo"
$GithubApi = "https://api.github.com/repos/$Repo"
# Source checkouts (-SourceRef builds) always come from the private source
# repository, never from the binaries mirror.
$SourceRepo = if ($env:NEXUS_SOURCE_REPO) { $env:NEXUS_SOURCE_REPO } else { "IAPro-Community/nexus" }
$SourceUrl = "https://github.com/$SourceRepo"

$IsWindowsOS = ($IsWindows -or ($env:OS -like "*Windows*"))

function Ensure-GoCompiler {
    if (Get-Command go -ErrorAction SilentlyContinue) {
        return
    }

    if (-not $IsWindowsOS) {
        throw "Go >=1.25 is required for source builds. Install Go manually and run the installer again."
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
    Write-Host "Bun was not found. Installing via official Bun script..." -ForegroundColor Yellow
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
            Write-Host "Resolved -Version latest to $($resp.tag_name) (checksum integrity; not a signed pin)." -ForegroundColor Yellow
            return [string]$resp.tag_name
        } catch {
            if ($attempt -ge 3) { throw }
            Write-Host "GitHub API resolve failed (attempt $attempt/3): $_" -ForegroundColor Yellow
            Start-Sleep -Seconds ($attempt * 2)
        }
    }
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
    if ($Winget) {
        Write-Host "Attempting WebView2 install via WinGet..." -ForegroundColor Yellow
        & $Winget.Source install --id Microsoft.EdgeWebView2Runtime --exact --accept-source-agreements --accept-package-agreements
    } else {
        Write-Host "Install WebView2 from https://developer.microsoft.com/microsoft-edge/webview2/" -ForegroundColor Yellow
    }
}

Write-Host "=== IAPro Nexus Installer (Zero-Toolchain Release) ===" -ForegroundColor Cyan
Write-Host "Release repo: $Repo" -ForegroundColor Gray

if ([string]::IsNullOrWhiteSpace($Version) -and -not $BuildFromSource) {
    throw "A version is required for verified installation. Use -Version vX.Y.Z, -Version latest, or -BuildFromSource from a checkout."
}

if ($Version -eq "latest") {
    $Version = Resolve-LatestVersion
}

if (-not [string]::IsNullOrWhiteSpace($Version)) {
    if ($Version -notmatch '^v?[0-9]+\.[0-9]+\.[0-9]+([.-][A-Za-z0-9.-]+)?$') {
        throw "Invalid Nexus version: $Version"
    }
    $Version = $Version.TrimStart('v')
}

$BinaryName = if ($IsWindowsOS) { "nexus.exe" } else { "nexus" }
$AiAliasName = if ($IsWindowsOS) { "ai.exe" } else { "ai" }

if ($IsWindowsOS) {
    $TargetDir = Join-Path $env:LOCALAPPDATA "Programs\IAPro Nexus"
    $LegacyTargetDir = Join-Path $env:LOCALAPPDATA "Programs\ai-cli"
} else {
    $TargetDir = Join-Path $HOME ".local/bin"
    $LegacyTargetDir = ""
}

if (-not (Test-Path $TargetDir)) {
    New-Item -ItemType Directory -Path $TargetDir -Force | Out-Null
}

if ($IsWindowsOS -and (Test-Path $LegacyTargetDir)) {
    Write-Host "Legacy ai-cli installation found at $LegacyTargetDir; leaving it untouched while installing IAPro Nexus to $TargetDir." -ForegroundColor Gray
}

$TargetPath = Join-Path $TargetDir $BinaryName
$Installed = $false

# Ed25519 public key for manifest signature verification (matches internal/update ProductionTrustRoot).
$NexusPubKey = "ee2e268394ce6a0634a9ea2f414ee831dd536d12f7e2b80107f94566e58aa334"

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
    $env:NEXUS_MANIFEST_PATH = $ManifestPath
    $env:NEXUS_ARTIFACT_NAME = $ArtifactName
    $result = python3 -c @"
import json, os, sys
from urllib.parse import unquote, urlparse
m = json.load(open(os.environ['NEXUS_MANIFEST_PATH']))
want = os.environ['NEXUS_ARTIFACT_NAME']
arts = m.get('artifacts', {})
for k, v in arts.items():
    url = unquote(urlparse(v.get('url', '')).path)
    if url.endswith('/' + want) or url.endswith(want):
        print(v.get('sha256', ''), end='')
        sys.exit(0)
key = want.lower().replace('-', '_')
if key.endswith('.tar.gz'):
    key = key[:-7]
elif '.' in key:
    key = key.rsplit('.', 1)[0]
for k, v in arts.items():
    if k == key or key in k or k in key:
        print(v.get('sha256', ''), end='')
        sys.exit(0)
print('', end='')
"@ 2>&1
    return $result.Trim()
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
$DownloadUrl = "$GithubUrl/releases/download/v$Version/$ArchiveName"
$ChecksumsUrl = "$GithubUrl/releases/download/v$Version/checksums.txt"

$TempDir = Join-Path ([System.IO.Path]::GetTempPath()) ([System.Guid]::NewGuid().ToString())
New-Item -ItemType Directory -Path $TempDir -Force | Out-Null

try {
    if (-not $BuildFromSource -and -not [string]::IsNullOrWhiteSpace($Version)) {
        Write-Host "Attempting to download Nexus v${Version}: $ArchiveName..." -ForegroundColor Yellow
        # 1) Fetch and verify signed manifest before downloading the artifact.
        $ManifestPath = Join-Path $TempDir "update-manifest.json"
        $SigPath = Join-Path $TempDir "update-manifest.sig"
        $ManifestUrl = "$GithubUrl/releases/download/v$Version/update-manifest.json"
        $SigUrl = "$GithubUrl/releases/download/v$Version/update-manifest.sig"
        Invoke-WebRequest -Uri $ManifestUrl -OutFile $ManifestPath -UseBasicParsing
        Write-Host "Signed manifest downloaded" -ForegroundColor Green
        Invoke-WebRequest -Uri $SigUrl -OutFile $SigPath -UseBasicParsing
        if (-not (Verify-ManifestSignature -ManifestPath $ManifestPath -SigPath $SigPath)) {
            throw "Release manifest signature verification failed."
        }
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
        if ($Expected.ToLowerInvariant() -ne $Actual) {
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
    if (-not (Get-Command make -ErrorAction SilentlyContinue)) {
        throw "make is required for -BuildFromSource. Install make (e.g. chocolatey/scoop/MSYS2) and re-run."
    }
    Write-Host "Building from source via make build (Go + Bun)..." -ForegroundColor Yellow
    if ((Test-Path "./go.mod") -and (Test-Path "./cmd/nexus")) {
        bun --cwd web install --frozen-lockfile
        make build
        $Built = if (Test-Path "./nexus.exe") { "./nexus.exe" } elseif (Test-Path "./nexus") { "./nexus" } else { $null }
        if (-not $Built) { throw "make build did not produce nexus binary." }
        Copy-Item -Path $Built -Destination $TargetPath -Force
        $Installed = $true
    } elseif (-not [string]::IsNullOrWhiteSpace($SourceRef)) {
        if ($SourceRef -notmatch '^[A-Za-z0-9._/-]+$') { throw "Invalid source ref." }
        if (-not (Get-Command git -ErrorAction SilentlyContinue)) {
            throw "git is required for -SourceRef builds."
        }
        Write-Host "Cloning explicitly requested source ref $SourceRef..." -ForegroundColor Yellow
        $CloneDir = Join-Path $TempDir "repo"
        git clone --depth 1 --branch $SourceRef "$SourceUrl.git" $CloneDir
        Push-Location $CloneDir
        try {
            bun --cwd web install --frozen-lockfile
            make build
            $Built = if (Test-Path "./nexus.exe") { "./nexus.exe" } elseif (Test-Path "./nexus") { "./nexus" } else { $null }
            if (-not $Built) { throw "make build did not produce nexus binary." }
            Copy-Item -Path $Built -Destination $TargetPath -Force
            $Installed = $true
        } finally {
            Pop-Location
        }
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
if ($MaestroCmd) {
    $MaestroAliasPath = Join-Path $TargetDir "maestro.cmd"
    Set-Content -Path $MaestroAliasPath -Value "@echo off`r`norquestrador-maestro %*" -Force
    Write-Host "✓ Linked Maestro alias ($MaestroAliasPath)" -ForegroundColor Green
}

Write-Host "✓ Successfully installed IAPro Nexus to $TargetPath" -ForegroundColor Green

if ($InstallDesktop -and $IsWindowsOS -and -not [string]::IsNullOrWhiteSpace($Version)) {
    $DesktopArchiveName = "nexus-desktop_Windows_${Arch}.zip"
    $DesktopDownloadUrl = "$GithubUrl/releases/download/v$Version/$DesktopArchiveName"
    $DesktopZipPath = Join-Path $TempDir $DesktopArchiveName
    $ManifestPath = Join-Path $TempDir "update-manifest.json"
    $SigPath = Join-Path $TempDir "update-manifest.sig"
    $NsisName = "nexus_${Version}_windows_amd64_setup.exe"
    try {
        Write-Host "Attempting to install the native IAPro Nexus Desktop shell..." -ForegroundColor Yellow
        if (-not (Test-Path $ManifestPath) -or -not (Test-Path $SigPath)) {
            $ManifestUrl = "$GithubUrl/releases/download/v$Version/update-manifest.json"
            $SigUrl = "$GithubUrl/releases/download/v$Version/update-manifest.sig"
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

    if ($PathEntries -notcontains $TargetDir) {
        Write-Host "Adding $TargetDir to user PATH environment variable..." -ForegroundColor Yellow
        $NewPath = if ([string]::IsNullOrEmpty($UserPath)) { $TargetDir } else { "$UserPath;$TargetDir" }
        [Environment]::SetEnvironmentVariable("Path", $NewPath, "User")
        $env:PATH = "$TargetDir;$env:PATH"
        Write-Host "✓ Added $TargetDir to PATH." -ForegroundColor Green
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
