[CmdletBinding()]
param()

$root = (Resolve-Path .).Path
$evidenceDir = Join-Path $root 'DEV\validation\windows-first-run\560a83b20c62611734e934d980b5725c3ef24cf1'
$logsDir = Join-Path $evidenceDir 'logs'
$logFile = Join-Path $logsDir '16-clean-install-smoke.log'

Write-Host "=== Phase 3: Clean Installation Smoke in Disposable Sandbox ==="

$sandboxDir = Join-Path $env:TEMP ("nexus-sandbox-install-" + [System.Guid]::NewGuid().ToString('N'))
$origUserPath = [Environment]::GetEnvironmentVariable("Path", "User")

try {
    Write-Host "Sandbox directory: $sandboxDir"
    
    # Run install.ps1 with -BuildFromSource and isolated -InstallDir
    & powershell -NoProfile -ExecutionPolicy Bypass -File (Join-Path $root 'install.ps1') `
        -BuildFromSource -InstallDir $sandboxDir -Yes *>&1 | Tee-Object -FilePath $logFile
    
    $installSuccess = ($LASTEXITCODE -eq 0)
    Write-Host "`nInstaller exit code: $LASTEXITCODE" | Tee-Object -FilePath $logFile -Append

    # Verify directory structure and files
    $nexusExe = Join-Path $sandboxDir 'nexus.exe'
    $aiExe = Join-Path $sandboxDir 'ai.exe'
    $managedMarker = Join-Path $sandboxDir '.nexus-managed'

    $hasNexus = Test-Path $nexusExe
    $hasAi = Test-Path $aiExe
    $hasManaged = Test-Path $managedMarker

    Write-Host "nexus.exe present: $hasNexus" | Tee-Object -FilePath $logFile -Append
    Write-Host "ai.exe present: $hasAi" | Tee-Object -FilePath $logFile -Append
    Write-Host ".nexus-managed present: $hasManaged" | Tee-Object -FilePath $logFile -Append

    # Test running nexus doctor in sandbox
    $doctorLog = & $nexusExe doctor 2>&1
    Write-Host "`nSandbox nexus doctor output:`n$($doctorLog -join "`n")" | Tee-Object -FilePath $logFile -Append

    # Test uninstaller isolation
    Write-Host "`nTesting uninstaller on sandbox..." | Tee-Object -FilePath $logFile -Append
    & powershell -NoProfile -ExecutionPolicy Bypass -File (Join-Path $root 'uninstall.ps1') `
        -InstallDir $sandboxDir -SkipPathCleanup -SkipDesktopShortcut *>&1 | Tee-Object -FilePath $logFile -Append

    $postUninstallNexus = Test-Path $nexusExe
    $postUninstallDir = Test-Path $sandboxDir
    Write-Host "Post-uninstall nexus.exe present: $postUninstallNexus" | Tee-Object -FilePath $logFile -Append
    Write-Host "Post-uninstall sandbox dir present: $postUninstallDir" | Tee-Object -FilePath $logFile -Append

    if ($installSuccess -and $hasNexus -and $hasAi -and $hasManaged -and (-not $postUninstallNexus)) {
        Write-Host "`nPHASE 3 CLEAN INSTALL TEST: PASS" | Tee-Object -FilePath $logFile -Append
    } else {
        Write-Host "`nPHASE 3 CLEAN INSTALL TEST: FAIL" | Tee-Object -FilePath $logFile -Append
    }
} finally {
    # Restore user PATH to original value
    [Environment]::SetEnvironmentVariable("Path", $origUserPath, "User")
    Write-Host "Restored original user PATH."
    if (Test-Path $sandboxDir) {
        Remove-Item -Path $sandboxDir -Recurse -Force -ErrorAction SilentlyContinue
    }
}
