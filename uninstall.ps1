# IAPro Nexus uninstaller for Windows / PowerShell Core.
# Removes Nexus-managed binaries and shortcut while preserving user data.
param(
    [string]$InstallDir = "",
    [switch]$SkipPathCleanup = $false,
    [switch]$SkipDesktopShortcut = $false
)

$ErrorActionPreference = 'Stop'
$IsWindowsOS = ($IsWindows -or ($env:OS -like "*Windows*"))

if ([string]::IsNullOrWhiteSpace($InstallDir)) {
    if (-not $IsWindowsOS) {
        throw "Default Nexus Windows install location is available only on Windows. Pass -InstallDir for an explicit location."
    }
    $InstallDir = Join-Path $env:LOCALAPPDATA "Programs\IAPro Nexus"
}

$InstallDir = [System.IO.Path]::GetFullPath($InstallDir)
$VolumeRoot = [System.IO.Path]::GetPathRoot($InstallDir)
if ([string]::Equals(
    $InstallDir.TrimEnd('\'), $VolumeRoot.TrimEnd('\'),
    [StringComparison]::OrdinalIgnoreCase
)) {
    throw "Refusing to uninstall from a volume root: $InstallDir"
}

# Never remove an arbitrary tree. The PowerShell installer writes a marker;
# NSIS installations already carry .nexus-nsis. Pre-marker installs are accepted
# only in the canonical per-user Nexus installation directory.
$CanonicalDir = if ($IsWindowsOS) {
    [System.IO.Path]::GetFullPath((Join-Path $env:LOCALAPPDATA "Programs\\IAPro Nexus"))
} else { "" }
$IsCanonicalDir = $IsWindowsOS -and [string]::Equals(
    $InstallDir.TrimEnd('\\'), $CanonicalDir.TrimEnd('\\'),
    [StringComparison]::OrdinalIgnoreCase
)
$NexusMarkerPath = Join-Path $InstallDir ".nexus-managed"
$NSISMarkerPath = Join-Path $InstallDir ".nexus-nsis"
$HasNexusMarker = (Test-Path -LiteralPath $NexusMarkerPath -PathType Leaf) -and
    ((Get-Content -LiteralPath $NexusMarkerPath -TotalCount 1) -eq "IAPro Nexus managed installation")
$HasNSISMarker = Test-Path -LiteralPath $NSISMarkerPath -PathType Leaf
$LegacyDefaultInstall = $IsCanonicalDir -and (Test-Path -LiteralPath (Join-Path $InstallDir "nexus.exe") -PathType Leaf)
$InstallRecognized = $HasNexusMarker -or $HasNSISMarker -or $LegacyDefaultInstall

if (Test-Path -LiteralPath $InstallDir -PathType Container) {
    $DirectoryInfo = Get-Item -LiteralPath $InstallDir -Force
    if (($DirectoryInfo.Attributes -band [IO.FileAttributes]::ReparsePoint) -ne 0) {
        throw "Refusing uninstall from a linked/reparse-point installation directory: $InstallDir"
    }
    if (-not $InstallRecognized) {
        throw "Refusing to delete an unrecognized directory: $InstallDir. Run the original uninstaller or reinstall Nexus into this location before uninstalling."
    }
    # Remove only known Nexus-owned files. Preserve unexpected files and user data.
    foreach ($name in @(
        "nexus.exe", "ai.exe", "nexus-desktop.exe", "nexus-hud.exe",
        "maestro.cmd", "Uninstall.exe", ".nexus-managed", ".nexus-nsis"
    )) {
        $ManagedFile = Join-Path $InstallDir $name
        if (Test-Path -LiteralPath $ManagedFile -PathType Leaf) {
            $Item = Get-Item -LiteralPath $ManagedFile -Force
            if (($Item.Attributes -band [IO.FileAttributes]::ReparsePoint) -ne 0) {
                throw "Refusing to delete a linked Nexus file: $ManagedFile"
            }
            Remove-Item -LiteralPath $ManagedFile -Force
        }
    }
    if (-not @(Get-ChildItem -LiteralPath $InstallDir -Force).Count) {
        Remove-Item -LiteralPath $InstallDir
        Write-Host "Removed empty Nexus installation directory $InstallDir" -ForegroundColor Green
    } else {
        Write-Host "Preserved remaining non-Nexus files in $InstallDir" -ForegroundColor Yellow
    }
} else {
    Write-Host "Nexus install directory already absent: $InstallDir" -ForegroundColor Gray
}

if (-not $SkipDesktopShortcut -and $IsWindowsOS) {
    $DesktopDir = [Environment]::GetFolderPath('Desktop')
    if (-not [string]::IsNullOrWhiteSpace($DesktopDir)) {
        $Shortcut = Join-Path $DesktopDir "IAPro Nexus.lnk"
        if ($InstallRecognized -and (Test-Path -LiteralPath $Shortcut -PathType Leaf)) {
            $Shell = New-Object -ComObject WScript.Shell
            $LinkTarget = $Shell.CreateShortcut($Shortcut).TargetPath
            $ExpectedTarget = Join-Path $InstallDir "nexus-desktop.exe"
            if (-not [string]::IsNullOrWhiteSpace($LinkTarget) -and
                [string]::Equals([System.IO.Path]::GetFullPath($LinkTarget), $ExpectedTarget, [StringComparison]::OrdinalIgnoreCase)) {
                Remove-Item -LiteralPath $Shortcut -Force
                Write-Host "Removed Nexus shortcut $Shortcut" -ForegroundColor Green
            }
        }
    }
}

function Get-NexusPathWithoutInstallDir {
    param(
        [Parameter(Mandatory)][AllowEmptyString()][string]$UserPath,
        [Parameter(Mandatory)][string]$InstallDir
    )

    $InstallPathKey = $InstallDir.TrimEnd('\')
    $Entries = @(
        $UserPath.Split([char[]]@(';'), [StringSplitOptions]::None) |
            Where-Object {
                -not [string]::Equals(
                    [Environment]::ExpandEnvironmentVariables($_).TrimEnd('\'),
                    $InstallPathKey,
                    [StringComparison]::OrdinalIgnoreCase
                )
            }
    )
    return ($Entries -join ';')
}

if (-not $SkipPathCleanup -and $IsWindowsOS -and $InstallRecognized) {
    $UserPath = [Environment]::GetEnvironmentVariable("Path", "User")
    if ($null -ne $UserPath) {
        $NewPath = Get-NexusPathWithoutInstallDir -UserPath $UserPath -InstallDir $InstallDir
        if ($NewPath -ne $UserPath) {
            [Environment]::SetEnvironmentVariable("Path", $NewPath, "User")
            Write-Host "Removed Nexus install directory from user PATH." -ForegroundColor Green
        }
    }
}

Write-Host "IAPro Nexus uninstall complete." -ForegroundColor Green
Write-Host "Profiles and project data were preserved." -ForegroundColor Gray
