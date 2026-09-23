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

if (Test-Path $InstallDir) {
    Remove-Item -Path $InstallDir -Recurse -Force
    Write-Host "Removed $InstallDir" -ForegroundColor Green
} else {
    Write-Host "Nexus install directory already absent: $InstallDir" -ForegroundColor Gray
}

if (-not $SkipDesktopShortcut -and $IsWindowsOS) {
    $DesktopDir = [Environment]::GetFolderPath('Desktop')
    if (-not [string]::IsNullOrWhiteSpace($DesktopDir)) {
        $Shortcut = Join-Path $DesktopDir "IAPro Nexus.lnk"
        if (Test-Path $Shortcut) {
            Remove-Item -Path $Shortcut -Force
            Write-Host "Removed $Shortcut" -ForegroundColor Green
        }
    }
}

if (-not $SkipPathCleanup -and $IsWindowsOS) {
    $UserPath = [Environment]::GetEnvironmentVariable("Path", "User")
    if ($null -ne $UserPath) {
        $Entries = @(
            $UserPath -split ';' |
                Where-Object {
                    -not [string]::IsNullOrWhiteSpace($_) -and
                    -not [string]::Equals(
                        [Environment]::ExpandEnvironmentVariables($_).TrimEnd('\'),
                        $InstallDir.TrimEnd('\'),
                        [StringComparison]::OrdinalIgnoreCase
                    )
                }
        )
        $NewPath = ($Entries -join ';')
        if ($NewPath -ne $UserPath) {
            [Environment]::SetEnvironmentVariable("Path", $NewPath, "User")
            Write-Host "Removed Nexus install directory from user PATH." -ForegroundColor Green
        }
    }
}

Write-Host "IAPro Nexus uninstall complete." -ForegroundColor Green
Write-Host "Profiles and project data were preserved." -ForegroundColor Gray
