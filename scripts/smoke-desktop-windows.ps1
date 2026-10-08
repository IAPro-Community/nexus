[CmdletBinding()]
param(
    [Parameter(Mandatory)][string]$DesktopExe,
    [Parameter(Mandatory)][string]$TargetSha,
    [Parameter(Mandatory)][string]$EvidencePath,
    [ValidateRange(1, 60)][int]$WaitSeconds = 8
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

if (-not (Test-Path -LiteralPath $DesktopExe -PathType Leaf)) {
    throw "Nexus Desktop executable not found: $DesktopExe"
}
foreach ($name in @('APPDATA', 'LOCALAPPDATA', 'NEXUS_DATA_DIR', 'AI_MANAGER_DATA_DIR')) {
    if ([string]::IsNullOrWhiteSpace([Environment]::GetEnvironmentVariable($name))) {
        throw "$name must point to an isolated Windows smoke directory"
    }
}
foreach ($directory in @($env:APPDATA, $env:LOCALAPPDATA, $env:NEXUS_DATA_DIR, $env:AI_MANAGER_DATA_DIR, (Split-Path -Parent $EvidencePath))) {
    New-Item -ItemType Directory -Path $directory -Force | Out-Null
}

if (-not ('NexusDesktopWindowProbe' -as [type])) {
    Add-Type -TypeDefinition @'
using System;
using System.Collections.Generic;
using System.Runtime.InteropServices;
using System.Text;

public static class NexusDesktopWindowProbe {
    private delegate bool EnumWindowsCallback(IntPtr hwnd, IntPtr lParam);
    [DllImport("user32.dll")] private static extern bool EnumWindows(EnumWindowsCallback callback, IntPtr lParam);
    [DllImport("user32.dll")] private static extern bool EnumChildWindows(IntPtr parent, EnumWindowsCallback callback, IntPtr lParam);
    [DllImport("user32.dll")] private static extern uint GetWindowThreadProcessId(IntPtr hwnd, out uint processId);
    [DllImport("user32.dll", CharSet = CharSet.Unicode)] private static extern int GetWindowText(IntPtr hwnd, StringBuilder text, int count);
    [DllImport("user32.dll", CharSet = CharSet.Unicode)] private static extern int GetClassName(IntPtr hwnd, StringBuilder text, int count);

    private static string Value(IntPtr hwnd, bool className) {
        var text = new StringBuilder(512);
        if (className) GetClassName(hwnd, text, text.Capacity);
        else GetWindowText(hwnd, text, text.Capacity);
        return text.ToString();
    }

    public static bool HasWailsWebView(int targetProcessId) {
        bool ready = false;
        EnumWindows((hwnd, _) => {
            GetWindowThreadProcessId(hwnd, out uint processId);
            if (processId == (uint)targetProcessId
                && Value(hwnd, true) == "WailsWebviewWindow"
                && Value(hwnd, false) == "IAPro Nexus") {
                EnumChildWindows(hwnd, (child, __) => {
                    GetWindowThreadProcessId(child, out uint childProcessId);
                    if (childProcessId == (uint)targetProcessId && Value(child, true) == "Chrome_WidgetWin_0") {
                        ready = true;
                        return false;
                    }
                    return true;
                }, IntPtr.Zero);
            }
            return !ready;
        }, IntPtr.Zero);
        return ready;
    }
}
'@
}

$stdout = "$EvidencePath.stdout.log"
$stderr = "$EvidencePath.stderr.log"
$webviewUserDataPath = Join-Path $env:LOCALAPPDATA 'IAPro\Nexus\desktop\WebView2'
$startedAt = [DateTime]::UtcNow
$process = $null

try {
    $process = Start-Process -FilePath $DesktopExe -PassThru -WindowStyle Hidden -RedirectStandardOutput $stdout -RedirectStandardError $stderr
    $deadline = [DateTime]::UtcNow.AddSeconds($WaitSeconds)
    $controllerReady = $false

    while ([DateTime]::UtcNow -lt $deadline) {
        $process.Refresh()
        if ($process.HasExited) { break }
        $controllerReady = [NexusDesktopWindowProbe]::HasWailsWebView($process.Id)
        Start-Sleep -Milliseconds 250
    }

    $process.Refresh()
    if ($process.HasExited) {
        if (Test-Path -LiteralPath $stdout) { Get-Content -LiteralPath $stdout }
        if (Test-Path -LiteralPath $stderr) { Get-Content -LiteralPath $stderr }
        throw "Nexus Desktop exited during the Windows launch smoke with code $($process.ExitCode)."
    }
    if (-not $controllerReady) {
        if (Test-Path -LiteralPath $stdout) { Get-Content -LiteralPath $stdout }
        if (Test-Path -LiteralPath $stderr) { Get-Content -LiteralPath $stderr }
        throw "Nexus Desktop did not create its Wails window and WebView2 controller within $WaitSeconds seconds."
    }

    if (-not (Test-Path -LiteralPath $webviewUserDataPath -PathType Container)) {
        throw "Nexus Desktop did not create its isolated WebView2 user data folder: $webviewUserDataPath"
    }

    @(
        "target_sha=$TargetSha"
        "platform=windows"
        "status=PASS"
        "wait_seconds=$([Math]::Ceiling(([DateTime]::UtcNow - $startedAt).TotalSeconds))"
        "wails_window=IAPro Nexus"
        "webview2_child=Chrome_WidgetWin_0"
        "webview2_userdata=$webviewUserDataPath"
    ) | Out-File -LiteralPath $EvidencePath -Encoding utf8
} finally {
    if ($null -ne $process) {
        $process.Refresh()
        if (-not $process.HasExited) {
            Stop-Process -Id $process.Id -Force -ErrorAction SilentlyContinue
        }
    }

    $ownedWebView2 = @()
    $cleanupDeadline = [DateTime]::UtcNow.AddSeconds(3)
    do {
        $ownedWebView2 = @(Get-CimInstance -ClassName Win32_Process -Filter "Name = 'msedgewebview2.exe'" -ErrorAction SilentlyContinue | Where-Object {
            -not [string]::IsNullOrWhiteSpace($_.CommandLine) -and
            $_.CommandLine.IndexOf($webviewUserDataPath, [StringComparison]::OrdinalIgnoreCase) -ge 0
        })
        if ($ownedWebView2.Count -eq 0) { break }
        Start-Sleep -Milliseconds 250
    } while ([DateTime]::UtcNow -lt $cleanupDeadline)

    foreach ($child in $ownedWebView2) {
        Stop-Process -Id $child.ProcessId -Force -ErrorAction SilentlyContinue
    }
}
