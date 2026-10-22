#Requires -Version 7.0
<#
.SYNOPSIS
    Proves that the native HUD window is click-through outside its interactive
    surfaces and still receives clicks on the capsule.

.DESCRIPTION
    The HUD window is a rectangle, but only the capsule/panel must swallow
    clicks. This harness:

      1. starts an isolated HUD with a target window behind it;
      2. waits until the native watcher marks a transparent corner as
         click-through (WS_EX_TRANSPARENT);
      3. asserts the capsule itself stays interactive;
      4. clicks the transparent corner and requires the target window to become
         foreground.

    Evidence is written as RESULT.json plus a screenshot. The script fails
    closed: any missing piece of evidence produces FAIL and a non-zero exit.

.PARAMETER HudExe
    Path to the freshly built nexus-hud.exe.

.PARAMETER EvidenceDirectory
    Directory that receives RESULT.json, the screenshot and the private logs.

.PARAMETER ExpectedSha
    Optional expected SHA-256 of the HUD binary. Mismatch fails the run.

.PARAMETER TimeoutSeconds
    How long to wait for the click-through watcher before failing.
#>
param(
    [Parameter(Mandatory)][string]$HudExe,
    [Parameter(Mandatory)][string]$EvidenceDirectory,
    [string]$ExpectedSha = '',
    [int]$TimeoutSeconds = 25
)

$ErrorActionPreference = 'Stop'
if (-not $IsWindows) { throw 'This harness only runs on Windows.' }

Add-Type -AssemblyName System.Drawing
Add-Type -AssemblyName System.Windows.Forms

Add-Type -TypeDefinition @'
using System;
using System.Text;
using System.Threading;
using System.Runtime.InteropServices;

public static class HudClickThrough {
    [StructLayout(LayoutKind.Sequential)] public struct Rect { public int Left, Top, Right, Bottom; }
    private delegate bool Callback(IntPtr hwnd, IntPtr param);

    [DllImport("user32.dll")] private static extern bool EnumWindows(Callback cb, IntPtr p);
    [DllImport("user32.dll")] private static extern uint GetWindowThreadProcessId(IntPtr hwnd, out uint pid);
    [DllImport("user32.dll")] private static extern bool IsWindowVisible(IntPtr hwnd);
    [DllImport("user32.dll")] private static extern bool GetWindowRect(IntPtr hwnd, out Rect rect);
    [DllImport("user32.dll")] private static extern int GetWindowText(IntPtr hwnd, StringBuilder text, int len);
    [DllImport("user32.dll")] private static extern bool SetForegroundWindow(IntPtr hwnd);
    [DllImport("user32.dll")] private static extern bool SetCursorPos(int x, int y);
    [DllImport("user32.dll")] private static extern void mouse_event(uint f, uint dx, uint dy, uint d, UIntPtr e);
    [DllImport("user32.dll")] private static extern bool MoveWindow(IntPtr h, int x, int y, int w, int hh, bool repaint);
    [DllImport("user32.dll")] private static extern IntPtr GetForegroundWindow();
    [DllImport("user32.dll")] private static extern int GetClassName(IntPtr h, StringBuilder text, int len);
    [DllImport("user32.dll")] private static extern int GetWindowLong(IntPtr h, int index);
    [DllImport("user32.dll")] private static extern bool BringWindowToTop(IntPtr h);

    public static IntPtr FindMain(int pid) {
        IntPtr best = IntPtr.Zero; int bestArea = 0;
        EnumWindows((h, _) => {
            uint p; GetWindowThreadProcessId(h, out p);
            if (p != pid || !IsWindowVisible(h)) return true;
            var c = new StringBuilder(256); GetClassName(h, c, c.Capacity);
            if (c.ToString() != "WailsWebviewWindow") return true;
            Rect r; GetWindowRect(h, out r);
            int area = (r.Right - r.Left) * (r.Bottom - r.Top);
            if (area > bestArea) { bestArea = area; best = h; }
            return true;
        }, IntPtr.Zero);
        return best;
    }

    public static IntPtr FindByTitle(string title) {
        IntPtr found = IntPtr.Zero;
        EnumWindows((h, _) => {
            if (!IsWindowVisible(h)) return true;
            var sb = new StringBuilder(512); GetWindowText(h, sb, sb.Capacity);
            if (sb.ToString() == title) { found = h; return false; }
            return true;
        }, IntPtr.Zero);
        return found;
    }

    public static int[] Bounds(IntPtr h) {
        Rect r; GetWindowRect(h, out r);
        return new int[] { r.Left, r.Top, r.Right - r.Left, r.Bottom - r.Top };
    }

    public static void Place(IntPtr h, int x, int y, int w, int hh) { MoveWindow(h, x, y, w, hh, true); }
    public static void Move(int x, int y) { SetCursorPos(x, y); }
    public static bool IsClickThrough(IntPtr h) { return (GetWindowLong(h, -20) & 0x20) != 0; }

    public static void Click(int x, int y) {
        SetCursorPos(x, y);
        Thread.Sleep(150);
        mouse_event(0x0002, 0, 0, 0, UIntPtr.Zero);
        mouse_event(0x0004, 0, 0, 0, UIntPtr.Zero);
    }

    public static string Title(IntPtr h) {
        var sb = new StringBuilder(512); GetWindowText(h, sb, sb.Capacity); return sb.ToString();
    }
    public static bool Focus(IntPtr h) { return SetForegroundWindow(h); }
    public static bool Raise(IntPtr h) { return BringWindowToTop(h); }
    public static IntPtr Foreground() { return GetForegroundWindow(); }
}
'@

New-Item -ItemType Directory -Path $EvidenceDirectory -Force | Out-Null
$resultPath = Join-Path $EvidenceDirectory 'RESULT.json'
$result = [ordered]@{
    platform = 'windows'
    timestamp_utc = [DateTime]::UtcNow.ToString('yyyy-MM-ddTHH:mm:ssZ')
    hud_sha256 = ''
    commit = ''
    verdict = 'FAIL'
    checks = [ordered]@{}
}

function Write-Result {
    $result | ConvertTo-Json -Depth 6 | Set-Content -Path $resultPath -Encoding utf8
}

$hudPath = (Resolve-Path $HudExe).Path
$sha = [System.Security.Cryptography.SHA256]::Create()
$stream = [System.IO.File]::OpenRead($hudPath)
try {
    $result.hud_sha256 = ([BitConverter]::ToString($sha.ComputeHash($stream)) -replace '-', '').ToLower()
} finally {
    $stream.Close(); $sha.Dispose()
}
$result.commit = (& git -C $PSScriptRoot rev-parse HEAD 2>$null | Out-String).Trim()
Write-Result

if ($ExpectedSha -and $result.hud_sha256 -ne $ExpectedSha.ToLower()) {
    $result.checks['sha_match'] = $false
    Write-Result
    throw "HUD SHA mismatch: expected $ExpectedSha, got $($result.hud_sha256)"
}
$result.checks['sha_match'] = $true

$isolated = Join-Path $EvidenceDirectory 'isolated-state'
foreach ($k in @('NEXUS_DATA_DIR', 'NEXUS_CONFIG_DIR', 'NEXUS_STATE_DIR')) {
    $v = Join-Path $isolated $k
    New-Item -ItemType Directory -Path $v -Force | Out-Null
    [Environment]::SetEnvironmentVariable($k, $v, 'Process')
}

$hud = $null
$targetPowerShell = $null
$targetRunspace = $null
$targetState = [hashtable]::Synchronized(@{})
$exitCode = 1
try {
    # WinForms needs an STA thread with its own message loop; a PowerShell
    # runspace keeps the harness self-contained without compiling WinForms.
    $targetRunspace = [runspacefactory]::CreateRunspace()
    $targetRunspace.ApartmentState = 'STA'
    $targetRunspace.ThreadOptions = 'ReuseThread'
    $targetRunspace.Open()
    $targetPowerShell = [powershell]::Create()
    $targetPowerShell.Runspace = $targetRunspace
    $null = $targetPowerShell.AddScript({
        param($state)
        Add-Type -AssemblyName System.Windows.Forms
        Add-Type -AssemblyName System.Drawing
        $form = New-Object System.Windows.Forms.Form
        $form.Text = 'Nexus ClickTarget'
        $form.FormBorderStyle = 'None'
        $form.StartPosition = 'Manual'
        $form.Bounds = New-Object System.Drawing.Rectangle(40, 40, 900, 400)
        $form.BackColor = [System.Drawing.Color]::FromArgb(18, 22, 30)
        $label = New-Object System.Windows.Forms.Label
        $label.Text = 'CLICK TARGET'
        $label.ForeColor = [System.Drawing.Color]::White
        $label.Dock = 'Fill'
        $label.TextAlign = 'MiddleCenter'
        $label.Font = New-Object System.Drawing.Font('Segoe UI', 28, [System.Drawing.FontStyle]::Bold)
        $form.Controls.Add($label)
        $state.Form = $form
        [System.Windows.Forms.Application]::Run($form)
    }).AddArgument($targetState)
    $null = $targetPowerShell.BeginInvoke()

    $targetDeadline = [DateTime]::UtcNow.AddSeconds(10)
    while ($null -eq $targetState.Form -and [DateTime]::UtcNow -lt $targetDeadline) {
        Start-Sleep -Milliseconds 100
    }
    if ($null -eq $targetState.Form) { throw 'The click target window did not start.' }
    Start-Sleep -Milliseconds 400

    $hud = Start-Process -FilePath $hudPath -PassThru `
        -RedirectStandardOutput (Join-Path $EvidenceDirectory 'private-stdout.log') `
        -RedirectStandardError (Join-Path $EvidenceDirectory 'private-stderr.log')

    Start-Sleep -Seconds 7
    $hHud = [HudClickThrough]::FindMain($hud.Id)
    if ($hHud -eq [IntPtr]::Zero) { throw 'No HUD window appeared.' }
    Start-Sleep -Seconds 3

    $bounds = [HudClickThrough]::Bounds($hHud)
    $result.checks['hud_bounds'] = $bounds -join ','
    if ($bounds[2] -le 0 -or $bounds[3] -le 0) { throw 'HUD window has no area.' }

    $targetState.Bounds = @(
        ($bounds[0] - 120),
        ($bounds[1] - 120),
        ($bounds[2] + 240),
        ($bounds[3] + 240)
    )
    $null = $targetState.Form.BeginInvoke([Action] {
        $b = $targetState.Bounds
        $targetState.Form.Bounds = New-Object System.Drawing.Rectangle($b[0], $b[1], $b[2], $b[3])
    })
    Start-Sleep -Milliseconds 600
    $hTarget = [HudClickThrough]::FindByTitle('Nexus ClickTarget')
    if ($hTarget -eq [IntPtr]::Zero) { throw 'No click target window.' }
    Start-Sleep -Milliseconds 400

    # The target must sit directly behind the HUD, otherwise a passing click
    # lands on whatever window happens to be in between.
    [HudClickThrough]::Focus($hTarget) | Out-Null
    [HudClickThrough]::Raise($hTarget) | Out-Null
    Start-Sleep -Milliseconds 500
    [HudClickThrough]::Focus($hHud) | Out-Null
    [HudClickThrough]::Raise($hHud) | Out-Null
    Start-Sleep -Milliseconds 500
    $result.checks['foreground_before'] = [HudClickThrough]::Title([HudClickThrough]::Foreground())
    if ($result.checks['foreground_before'] -ne 'Nexus HUD') {
        throw "HUD did not take focus (foreground: $($result.checks['foreground_before']))"
    }

    $tx = $bounds[0] + 8
    $ty = $bounds[1] + 8
    $result.checks['click_point'] = "$tx,$ty"

    [HudClickThrough]::Move($tx, $ty)
    $sw = [System.Diagnostics.Stopwatch]::StartNew()
    $stable = 0
    while ($sw.ElapsedMilliseconds -lt ($TimeoutSeconds * 1000) -and $stable -lt 3) {
        if ([HudClickThrough]::IsClickThrough($hHud)) { $stable++ } else { $stable = 0 }
        Start-Sleep -Milliseconds 40
    }
    $result.checks['click_through_wait_ms'] = $sw.ElapsedMilliseconds
    $result.checks['transparent_corner_click_through'] = ($stable -ge 3)

    $cx = $bounds[0] + [int]($bounds[2] * 0.6)
    $cy = $bounds[1] + [int]($bounds[3] / 2)
    [HudClickThrough]::Move($cx, $cy)
    Start-Sleep -Milliseconds 300
    $result.checks['capsule_click_through'] = [HudClickThrough]::IsClickThrough($hHud)
    if ($result.checks['capsule_click_through']) { throw 'The capsule became click-through; the HUD would be unusable.' }

    [HudClickThrough]::Move($tx, $ty)
    $sw.Reset()
    $stable = 0
    while ($sw.ElapsedMilliseconds -lt 5000 -and $stable -lt 3) {
        if ([HudClickThrough]::IsClickThrough($hHud)) { $stable++ } else { $stable = 0 }
        Start-Sleep -Milliseconds 40
    }
    $result.checks['click_through_restored'] = ($stable -ge 3)
    if (-not $result.checks['click_through_restored']) { throw 'Click-through was not restored after leaving the capsule.' }

    [HudClickThrough]::Click($tx, $ty)
    Start-Sleep -Milliseconds 1000
    $result.checks['foreground_after'] = [HudClickThrough]::Title([HudClickThrough]::Foreground())
    $result.checks['click_reached_target'] = ($result.checks['foreground_after'] -eq 'Nexus ClickTarget')

    $bmp = New-Object System.Drawing.Bitmap(($bounds[2] + 64), ($bounds[3] + 64))
    $graphics = [System.Drawing.Graphics]::FromImage($bmp)
    $graphics.CopyFromScreen(($bounds[0] - 32), ($bounds[1] - 32), 0, 0,
        (New-Object System.Drawing.Size(($bounds[2] + 64), ($bounds[3] + 64))))
    $graphics.Dispose()
    $bmp.Save((Join-Path $EvidenceDirectory 'HUD_CLICK_THROUGH.png'), [System.Drawing.Imaging.ImageFormat]::Png)
    $bmp.Dispose()

    $passed = $result.checks['transparent_corner_click_through'] -and
        (-not $result.checks['capsule_click_through']) -and
        $result.checks['click_through_restored'] -and
        $result.checks['click_reached_target']
    $result.verdict = if ($passed) { 'PASS' } else { 'FAIL' }
    Write-Result
    if ($passed) { $exitCode = 0 }
} catch {
    $result.checks['error'] = $_.Exception.Message
    Write-Result
} finally {
    if ($null -ne $targetState.Form) {
        try {
            $null = $targetState.Form.BeginInvoke([Action] { $targetState.Form.Close() })
            Start-Sleep -Milliseconds 300
        } catch { }
    }
    if ($null -ne $targetPowerShell) { $targetPowerShell.Dispose() }
    if ($null -ne $targetRunspace) { $targetRunspace.Close(); $targetRunspace.Dispose() }
    foreach ($p in @($hud)) {
        if ($null -ne $p) {
            $p.Refresh()
            if (-not $p.HasExited) { Stop-Process -Id $p.Id -Force -ErrorAction SilentlyContinue }
        }
    }
}

"VERDICT=$($result.verdict)"
"RESULT=$resultPath"
exit $exitCode
