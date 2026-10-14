[CmdletBinding()]
param(
    [Parameter(Mandatory)][string]$DesktopExe,
    [Parameter(Mandatory)][string]$TargetSha,
    [Parameter(Mandatory)][string]$EvidencePath,
    [ValidateNotNullOrEmpty()][string]$SingleInstanceID = 'com.iapro.nexus.desktop',
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
using System.ComponentModel;
using System.Runtime.InteropServices;
using System.Text;

public static class NexusDesktopWindowProbe {
    [StructLayout(LayoutKind.Sequential, CharSet = CharSet.Unicode)]
    private struct ProcessEntry32 {
        public uint dwSize;
        public uint cntUsage;
        public uint th32ProcessID;
        public UIntPtr th32DefaultHeapID;
        public uint th32ModuleID;
        public uint cntThreads;
        public uint th32ParentProcessID;
        public int pcPriClassBase;
        public uint dwFlags;
        [MarshalAs(UnmanagedType.ByValTStr, SizeConst = 260)] public string szExeFile;
    }

    private delegate bool EnumWindowsCallback(IntPtr hwnd, IntPtr lParam);
    [DllImport("user32.dll")] private static extern bool EnumWindows(EnumWindowsCallback callback, IntPtr lParam);
    [DllImport("user32.dll")] private static extern bool EnumChildWindows(IntPtr parent, EnumWindowsCallback callback, IntPtr lParam);
    [DllImport("user32.dll")] private static extern uint GetWindowThreadProcessId(IntPtr hwnd, out uint processId);
    [DllImport("user32.dll", CharSet = CharSet.Unicode)] private static extern IntPtr FindWindow(string className, string windowName);
    [DllImport("user32.dll", CharSet = CharSet.Unicode)] private static extern int GetWindowText(IntPtr hwnd, StringBuilder text, int count);
    [DllImport("user32.dll", CharSet = CharSet.Unicode)] private static extern int GetClassName(IntPtr hwnd, StringBuilder text, int count);
    [DllImport("kernel32.dll", SetLastError = true)] private static extern IntPtr CreateToolhelp32Snapshot(uint flags, uint processId);
    [DllImport("kernel32.dll", CharSet = CharSet.Unicode, SetLastError = true)] private static extern bool Process32FirstW(IntPtr snapshot, ref ProcessEntry32 entry);
    [DllImport("kernel32.dll", CharSet = CharSet.Unicode, SetLastError = true)] private static extern bool Process32NextW(IntPtr snapshot, ref ProcessEntry32 entry);
    [DllImport("kernel32.dll", SetLastError = true)] private static extern bool CloseHandle(IntPtr handle);

    private static string Value(IntPtr hwnd, bool className) {
        var text = new StringBuilder(512);
        if (className) GetClassName(hwnd, text, text.Capacity);
        else GetWindowText(hwnd, text, text.Capacity);
        return text.ToString();
    }

    public static bool HasWailsWebView(int targetProcessId) {
        bool ready = false;
        EnumWindows((hwnd, _) => {
            uint processId = 0;
            GetWindowThreadProcessId(hwnd, out processId);
            if (processId == (uint)targetProcessId
                && Value(hwnd, true) == "WailsWebviewWindow"
                && Value(hwnd, false) == "IAPro Nexus") {
                EnumChildWindows(hwnd, (child, __) => {
                    uint childProcessId = 0;
                    GetWindowThreadProcessId(child, out childProcessId);
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

    public static int GetSingleInstanceOwner(string uniqueId) {
        string id = "wails-app-" + uniqueId;
        IntPtr hwnd = FindWindow(id + "-sic", id + "-siw");
        if (hwnd == IntPtr.Zero) return 0;

        uint processId = 0;
        GetWindowThreadProcessId(hwnd, out processId);
        return (int)processId;
    }

    public static int[] GetOwnedWebView2ProcessIds(int rootProcessId) {
        const uint TH32CS_SNAPPROCESS = 0x00000002;
        IntPtr snapshot = CreateToolhelp32Snapshot(TH32CS_SNAPPROCESS, 0);
        if (snapshot == new IntPtr(-1)) {
            throw new Win32Exception(Marshal.GetLastWin32Error(), "Could not enumerate Windows processes for the isolated Desktop smoke.");
        }

        try {
            var parents = new Dictionary<uint, uint>();
            var names = new Dictionary<uint, string>();
            var entry = new ProcessEntry32();
            entry.dwSize = (uint)Marshal.SizeOf(typeof(ProcessEntry32));
            if (!Process32FirstW(snapshot, ref entry)) {
                throw new Win32Exception(Marshal.GetLastWin32Error(), "Could not read the first Windows process entry.");
            }
            do {
                parents[entry.th32ProcessID] = entry.th32ParentProcessID;
                names[entry.th32ProcessID] = entry.szExeFile ?? "";
                entry.dwSize = (uint)Marshal.SizeOf(typeof(ProcessEntry32));
            } while (Process32NextW(snapshot, ref entry));

            var result = new List<int>();
            var pending = new Queue<uint>();
            pending.Enqueue((uint)rootProcessId);
            while (pending.Count > 0) {
                uint parentId = pending.Dequeue();
                foreach (KeyValuePair<uint, uint> process in parents) {
                    if (process.Value != parentId) continue;
                    pending.Enqueue(process.Key);
                    if (String.Equals(names[process.Key], "msedgewebview2.exe", StringComparison.OrdinalIgnoreCase)) {
                        result.Add((int)process.Key);
                    }
                }
            }
            return result.ToArray();
        } finally {
            CloseHandle(snapshot);
        }
    }
}
'@
}

$stdout = "$EvidencePath.stdout.log"
$stderr = "$EvidencePath.stderr.log"
$webviewUserDataPath = Join-Path $env:LOCALAPPDATA 'IAPro\Nexus\desktop\WebView2'
if ($webviewUserDataPath.Length -gt 240) {
    throw "WebView2 user-data path is too long for a reliable Windows smoke ($($webviewUserDataPath.Length) characters). Set LOCALAPPDATA to a short isolated directory so the full path stays at or below 240 characters."
}
$startedAt = [DateTime]::UtcNow
$process = $null
$smokeSucceeded = $false
$ownedWebView2Ids = @()
$singleInstanceOwnerId = [NexusDesktopWindowProbe]::GetSingleInstanceOwner($SingleInstanceID)
if ($singleInstanceOwnerId -gt 0) {
    $ownerDescription = "PID $singleInstanceOwnerId"
    $ownerProcess = Get-Process -Id $singleInstanceOwnerId -ErrorAction SilentlyContinue
    if ($null -ne $ownerProcess) {
        $ownerDescription = "$($ownerProcess.ProcessName) (PID $singleInstanceOwnerId)"
    }
    throw "Wails single-instance ID '$SingleInstanceID' is already held by $ownerDescription. Build the smoke executable with a dedicated ID and pass it through -SingleInstanceID; do not terminate the active application."
}

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

    $smokeSucceeded = $true
} finally {
    $cleanupEnumerationError = $null
    if ($null -ne $process) {
        $process.Refresh()
        try {
            $ownedWebView2Ids = @([NexusDesktopWindowProbe]::GetOwnedWebView2ProcessIds($process.Id))
        } catch {
            $cleanupEnumerationError = $_.Exception.Message
        }
        if (-not $process.HasExited) {
            Stop-Process -Id $process.Id -Force -ErrorAction SilentlyContinue
        }
    }

    $cleanupDeadline = [DateTime]::UtcNow.AddSeconds(3)
    do {
        $remainingWebView2Ids = @($ownedWebView2Ids | Where-Object { Get-Process -Id $_ -ErrorAction SilentlyContinue })
        if ($remainingWebView2Ids.Count -eq 0) { break }
        Start-Sleep -Milliseconds 250
    } while ([DateTime]::UtcNow -lt $cleanupDeadline)

    foreach ($processId in $remainingWebView2Ids) {
        $child = Get-Process -Id $processId -ErrorAction SilentlyContinue
        if ($null -ne $child -and $child.ProcessName -ieq 'msedgewebview2') {
            Stop-Process -Id $processId -Force -ErrorAction SilentlyContinue
        }
    }

    $leftoverWebView2Ids = @($ownedWebView2Ids | Where-Object { Get-Process -Id $_ -ErrorAction SilentlyContinue })
    if ($cleanupEnumerationError) {
        throw "Could not verify isolated WebView2 cleanup: $cleanupEnumerationError"
    }
    if ($leftoverWebView2Ids.Count -gt 0) {
        throw "Isolated WebView2 processes remain after cleanup: $($leftoverWebView2Ids -join ', ')"
    }
}

if (-not $smokeSucceeded) {
    throw 'Desktop smoke did not complete its launch checks.'
}

@(
    "target_sha=$TargetSha"
    "platform=windows"
    "status=PASS"
    "single_instance_id=$SingleInstanceID"
    "wait_seconds=$([Math]::Ceiling(([DateTime]::UtcNow - $startedAt).TotalSeconds))"
    "wails_window=IAPro Nexus"
    "webview2_child=Chrome_WidgetWin_0"
    "webview2_userdata=$webviewUserDataPath"
    'webview2_process_cleanup=PASS'
) | Out-File -LiteralPath $EvidencePath -Encoding utf8
