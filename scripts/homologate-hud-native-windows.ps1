[CmdletBinding()]
param(
    [Parameter(Mandatory)][string]$HudExe,
    [Parameter(Mandatory)][string]$EvidenceDirectory,
    [string]$ExpectedSha = '',
    [ValidateRange(1, 50)][int]$Cycles = 20
)

# Real mouse/geometry acceptance. Run in an INTERACTIVE Windows desktop session.
# No signing, installer, registry edits, credential access or global process kills.
Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'
Add-Type -AssemblyName System.Drawing
Add-Type -AssemblyName System.Windows.Forms

if (-not $IsWindows) { throw 'This HUD acceptance probe requires Windows.' }
if (-not [Environment]::UserInteractive) { throw 'BLOCKED: an interactive desktop session is required.' }
$exe = (Resolve-Path -LiteralPath $HudExe -ErrorAction Stop).Path
$root = Split-Path -Parent $PSScriptRoot
$sha = (& git -C $root rev-parse HEAD).Trim()
if ($LASTEXITCODE -ne 0 -or $sha -notmatch '^[a-f0-9]{40}$') { throw 'Cannot identify the source SHA.' }
if ($ExpectedSha -and $sha -ne $ExpectedSha) { throw "SHA mismatch: $sha != $ExpectedSha" }

# Never interact with an existing user HUD or accidentally terminate it.
if (@(Get-Process -Name 'nexus-hud*' -ErrorAction SilentlyContinue).Count -gt 0) {
    throw 'BLOCKED: another Nexus HUD is already running; use an isolated machine/session.'
}

$dest = [IO.Path]::GetFullPath($EvidenceDirectory)
New-Item -ItemType Directory -Path $dest -Force | Out-Null
$original = @{}
foreach ($key in @('NEXUS_DATA_DIR', 'NEXUS_CONFIG_DIR', 'NEXUS_STATE_DIR', 'NEXUS_ENABLE_HUD', 'NEXUS_DISABLE_HUD')) {
    $original[$key] = [Environment]::GetEnvironmentVariable($key, 'Process')
}
$isolated = Join-Path $dest 'isolated-state'
foreach ($key in @('NEXUS_DATA_DIR', 'NEXUS_CONFIG_DIR', 'NEXUS_STATE_DIR')) {
    $value = Join-Path $isolated $key
    New-Item -ItemType Directory -Path $value -Force | Out-Null
    [Environment]::SetEnvironmentVariable($key, $value, 'Process')
}
[Environment]::SetEnvironmentVariable('NEXUS_ENABLE_HUD', $null, 'Process')
[Environment]::SetEnvironmentVariable('NEXUS_DISABLE_HUD', $null, 'Process')

if (-not ('NexusHUDNativeMouseProbe' -as [type])) {
    Add-Type -TypeDefinition @'
using System;
using System.Text;
using System.Runtime.InteropServices;
public static class NexusHUDNativeMouseProbe {
    [StructLayout(LayoutKind.Sequential)]
    public struct Rect { public int Left, Top, Right, Bottom; }
    private delegate bool Callback(IntPtr hwnd, IntPtr param);
    [DllImport("user32.dll")] private static extern bool EnumWindows(Callback callback, IntPtr param);
    [DllImport("user32.dll")] private static extern uint GetWindowThreadProcessId(IntPtr hwnd, out uint pid);
    [DllImport("user32.dll")] private static extern int GetWindowText(IntPtr hwnd, StringBuilder text, int length);
    [DllImport("user32.dll")] private static extern bool IsWindowVisible(IntPtr hwnd);
    [DllImport("user32.dll")] private static extern bool GetWindowRect(IntPtr hwnd, out Rect rect);
    [DllImport("user32.dll")] private static extern bool SetForegroundWindow(IntPtr hwnd);
    [DllImport("user32.dll")] private static extern bool SetCursorPos(int x, int y);
    [DllImport("user32.dll")] private static extern void mouse_event(uint flags, uint dx, uint dy, uint data, UIntPtr extra);
    public static IntPtr Find(int processId) {
        IntPtr found = IntPtr.Zero;
        EnumWindows((hwnd, _) => {
            uint pid;
            GetWindowThreadProcessId(hwnd, out pid);
            if (pid != processId || !IsWindowVisible(hwnd)) return true;
            var title = new StringBuilder(512);
            GetWindowText(hwnd, title, title.Capacity);
            if (title.ToString().Contains("Nexus HUD")) { found = hwnd; return false; }
            return true;
        }, IntPtr.Zero);
        return found;
    }
    public static int[] Bounds(IntPtr hwnd) {
        Rect r;
        if (hwnd == IntPtr.Zero || !GetWindowRect(hwnd, out r)) return new int[0];
        return new int[] { r.Left, r.Top, r.Right-r.Left, r.Bottom-r.Top };
    }
    public static void Click(IntPtr hwnd, int x, int y) {
        SetForegroundWindow(hwnd);
        SetCursorPos(x, y);
        System.Threading.Thread.Sleep(100);
        mouse_event(0x0002, 0, 0, 0, UIntPtr.Zero);
        mouse_event(0x0004, 0, 0, 0, UIntPtr.Zero);
    }
}
'@
}

function Capture-Hud([int[]]$Bounds, [string]$Filename) {
    # Save a small region around the window to include OS shadows.
    $screen = [System.Windows.Forms.SystemInformation]::VirtualScreen
    $pad = 28
    $x = [Math]::Max($screen.Left, $Bounds[0] - $pad)
    $y = [Math]::Max($screen.Top, $Bounds[1] - $pad)
    $w = [Math]::Min($Bounds[0] + $Bounds[2] + $pad, $screen.Right) - $x
    $h = [Math]::Min($Bounds[1] + $Bounds[3] + $pad, $screen.Bottom) - $y
    if ($w -le 0 -or $h -le 0) { throw 'HUD lies outside virtual screen.' }
    $bitmap = New-Object System.Drawing.Bitmap($w, $h)
    $graphics = [System.Drawing.Graphics]::FromImage($bitmap)
    try {
        $graphics.CopyFromScreen($x, $y, 0, 0, (New-Object System.Drawing.Size($w, $h)))
        $bitmap.Save((Join-Path $dest $Filename), [System.Drawing.Imaging.ImageFormat]::Png)
    } finally { $graphics.Dispose(); $bitmap.Dispose() }
}

function Wait-Geometry([IntPtr]$Handle, [bool]$Expanded) {
    $deadline = [DateTime]::UtcNow.AddSeconds(4)
    while ([DateTime]::UtcNow -lt $deadline) {
        $b = [NexusHUDNativeMouseProbe]::Bounds($Handle)
        if ($b.Count -eq 4) {
            if ($Expanded -and $b[2] -ge 300 -and $b[3] -ge 300) { return ,$b }
            if (-not $Expanded -and $b[2] -le 330 -and $b[3] -le 100) { return ,$b }
        }
        Start-Sleep -Milliseconds 120
    }
    throw "HUD did not transition to Expanded=$Expanded"
}

$proc = $null
$result = [ordered]@{
    sha = $sha
    platform = 'windows'
    type = 'native_interactive_mouse'
    cycles_requested = $Cycles
    cycles_completed = 0
    screenshot_compact = $null
    screenshot_expanded = $null
    verdict = 'INCONCLUSIVE'
    failure = $null
    visual_shadow_review = 'NOT_TESTED'
    transparent_click_through = 'NOT_TESTED'
    note = 'Geometry and real mouse were tested; visual shadow and click-through require screenshot/manual review.'
}
try {
    $proc = Start-Process -FilePath $exe -PassThru -RedirectStandardOutput (Join-Path $dest 'private-stdout.log') -RedirectStandardError (Join-Path $dest 'private-stderr.log')
    $handle = [IntPtr]::Zero
    $deadline = [DateTime]::UtcNow.AddSeconds(25)
    while ([DateTime]::UtcNow -lt $deadline) {
        $proc.Refresh()
        if ($proc.HasExited) { throw 'Native HUD exited before showing a window.' }
        $handle = [NexusHUDNativeMouseProbe]::Find($proc.Id)
        if ($handle -ne [IntPtr]::Zero) { break }
        Start-Sleep -Milliseconds 200
    }
    if ($handle -eq [IntPtr]::Zero) { throw 'No visible Nexus HUD window was found for the isolated process.' }
    $collapsed = Wait-Geometry $handle $false
    Capture-Hud $collapsed 'HUD_BEFORE.png'
    $result.screenshot_compact = 'HUD_BEFORE.png'

    for ($i=0; $i -lt $Cycles; $i++) {
        $b = [NexusHUDNativeMouseProbe]::Bounds($handle)
        # Capsule is bottom-aligned; drag grip occupies the leftmost 24px.
        $x = $b[0] + [int]($b[2]*0.68)
        $y = $b[1] + $b[3] - 27
        [NexusHUDNativeMouseProbe]::Click($handle, $x, $y)
        $expanded = Wait-Geometry $handle $true
        if ($i -eq 0) {
            Capture-Hud $expanded 'HUD_EXPANDED.png'
            $result.screenshot_expanded = 'HUD_EXPANDED.png'
        }
        $x = $expanded[0] + [int]($expanded[2]*0.68)
        $y = $expanded[1] + $expanded[3] - 27
        [NexusHUDNativeMouseProbe]::Click($handle, $x, $y)
        $collapsed = Wait-Geometry $handle $false
        $result.cycles_completed = $i+1
    }
    $result.verdict = 'PASS_FUNCTIONAL_PENDING_VISUAL'
} catch {
    $result.verdict = 'NO_GO'
    $result.failure = $_.Exception.Message
} finally {
    if ($null -ne $proc) {
        $proc.Refresh()
        if (-not $proc.HasExited) { Stop-Process -Id $proc.Id -Force -ErrorAction SilentlyContinue }
    }
    foreach ($key in $original.Keys) {
        [Environment]::SetEnvironmentVariable($key, $original[$key], 'Process')
    }
    $result | ConvertTo-Json -Depth 5 | Set-Content -LiteralPath (Join-Path $dest 'HUD_NATIVE_RESULTS.json') -Encoding utf8
}
Write-Host "HUD native evidence: $dest"
Write-Host "Result: $($result.verdict); completed $($result.cycles_completed)/$Cycles"
if ($result.verdict -eq 'NO_GO') { throw "Native HUD interaction failed: $($result.failure)" }
