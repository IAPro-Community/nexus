#Requires -Version 7.0
<#
.SYNOPSIS
    Proves that the native HUD can enable the independent popup from its own
    settings and that the mascot becomes usable, without restarting the app.

.DESCRIPTION
    The independent popup used to be an environment-only (NEXUS_HUD_DETACHED_POPOVER)
    startup switch, which made the mascot unreachable in the shipped app. This
    harness drives the real WebView through UI Automation in two sessions that
    share one isolated state directory:

      Session A (fresh install, popup disabled):
      1. opens the panel and asserts "Popup independente: Não" and that Mascote
         is disabled;
      2. invokes the toggle and asserts the preference persists, the hidden
         popup window is created and the capsule shrinks back to compact size;
      3. reopens the panel, asserts the popup window is visible, the capsule
         kept its compact size and Mascote is now enabled;
      4. selects Mascote and asserts the capsule becomes the mascot size, that
         the capsule itself renders the mascot and that the popup hides.

      Session B (restart with the persisted preference):
      5. asserts the HUD starts in MASCOT with the popup preference intact;
      6. opens the popup from the mascot and disables the popup inside it;
      7. asserts BAR is restored, the capsule returns to compact size, the
         popup hides and the legacy inline panel still opens.

    Evidence is RESULT.json plus screenshots. Any missing check fails closed.
#>
param(
    [Parameter(Mandatory)][string]$HudExe,
    [Parameter(Mandatory)][string]$EvidenceDirectory,
    [string]$ExpectedSha = '',
    [int]$TimeoutSeconds = 45
)

$ErrorActionPreference = 'Stop'
if (-not $IsWindows) { throw 'This harness only runs on Windows.' }

Add-Type -AssemblyName UIAutomationClient
Add-Type -AssemblyName UIAutomationTypes
Add-Type -AssemblyName System.Drawing
Add-Type -AssemblyName System.Windows.Forms

Add-Type -TypeDefinition @'
using System;
using System.Text;
using System.Collections.Generic;
using System.Runtime.InteropServices;
public class HudToggle {
    [StructLayout(LayoutKind.Sequential)] public struct RECT { public int Left, Top, Right, Bottom; }
    public class WinInfo {
        public IntPtr Handle; public string Title; public string Class;
        public int X, Y, W, H; public bool Visible;
    }
    private delegate bool Callback(IntPtr hwnd, IntPtr p);
    [DllImport("user32.dll")] private static extern bool EnumWindows(Callback cb, IntPtr p);
    [DllImport("user32.dll")] private static extern uint GetWindowThreadProcessId(IntPtr hwnd, out uint pid);
    [DllImport("user32.dll")] private static extern bool IsWindowVisible(IntPtr hwnd);
    [DllImport("user32.dll")] private static extern bool GetWindowRect(IntPtr hwnd, out RECT r);
    [DllImport("user32.dll")] private static extern int GetWindowText(IntPtr hwnd, StringBuilder t, int len);
    [DllImport("user32.dll")] private static extern int GetClassName(IntPtr hwnd, StringBuilder t, int len);
    [DllImport("user32.dll")] private static extern bool SetCursorPos(int x, int y);
    [DllImport("user32.dll")] private static extern void mouse_event(uint f, uint dx, uint dy, uint d, UIntPtr e);
    [DllImport("user32.dll")] private static extern uint GetDpiForWindow(IntPtr hwnd);
    [DllImport("user32.dll")] private static extern int GetSystemMetrics(int index);

    public static List<WinInfo> List(int pid) {
        var list = new List<WinInfo>();
        EnumWindows((h, _) => {
            uint p; GetWindowThreadProcessId(h, out p);
            if (p != pid) return true;
            var t = new StringBuilder(256); GetWindowText(h, t, t.Capacity);
            var c = new StringBuilder(256); GetClassName(h, c, c.Capacity);
            RECT r; GetWindowRect(h, out r);
            list.Add(new WinInfo { Handle = h, Title = t.ToString(), Class = c.ToString(),
                X = r.Left, Y = r.Top, W = r.Right - r.Left, H = r.Bottom - r.Top,
                Visible = IsWindowVisible(h) });
            return true;
        }, IntPtr.Zero);
        return list;
    }

    public static void Click(int x, int y) {
        SetCursorPos(x, y);
        System.Threading.Thread.Sleep(80);
        mouse_event(0x0002, 0, 0, 0, UIntPtr.Zero);
        System.Threading.Thread.Sleep(40);
        mouse_event(0x0004, 0, 0, 0, UIntPtr.Zero);
    }

    public static int Dpi(IntPtr hwnd) { return (int)GetDpiForWindow(hwnd); }
    public static int MinTrackWidth() { return GetSystemMetrics(28); }
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
    $result | ConvertTo-Json -Depth 8 | Set-Content -Path $resultPath -Encoding utf8
}

$hudPath = (Resolve-Path $HudExe).Path
$sha = [System.Security.Cryptography.SHA256]::Create()
$stream = [IO.File]::OpenRead($hudPath)
try {
    $result.hud_sha256 = ([BitConverter]::ToString($sha.ComputeHash($stream)) -replace '-', '').ToLower()
} finally {
    $stream.Close(); $sha.Dispose()
}
$repoRoot = (Resolve-Path (Join-Path $PSScriptRoot '..')).Path
$result.commit = (& git -C $repoRoot rev-parse HEAD 2>$null | Out-String).Trim()
if ($LASTEXITCODE -ne 0 -or $result.commit -notmatch '^[a-fA-F0-9]{40}

# ---- helpers -------------------------------------------------------------

function Get-HudWindows {
    param([int]$ProcessId)
    return @([HudToggle]::List($ProcessId) | Where-Object { $_.Class -eq 'WailsWebviewWindow' })
}

function Read-HudState {
    $path = Join-Path $env:NEXUS_DATA_DIR 'hud-window-state.json'
    if (-not (Test-Path $path)) { return $null }
    try { return (Get-Content $path -Raw | ConvertFrom-Json) } catch { return $null }
}

function Wait-HudState {
    param([scriptblock]$Predicate, [int]$Seconds = 15)
    $deadline = [DateTime]::UtcNow.AddSeconds($Seconds)
    while ([DateTime]::UtcNow -lt $deadline) {
        $state = Read-HudState
        if ($null -ne $state -and (& $Predicate $state)) { return $state }
        Start-Sleep -Milliseconds 150
    }
    return $null
}

function Get-UiaRoot {
    param([IntPtr]$Handle)
    return [System.Windows.Automation.AutomationElement]::FromHandle($Handle)
}

function Find-UiaElement {
    param(
        [IntPtr]$Handle,
        [string]$NamePrefix,
        [int]$Seconds = 10
    )
    $deadline = [DateTime]::UtcNow.AddSeconds($Seconds)
    while ([DateTime]::UtcNow -lt $deadline) {
        try {
            $root = Get-UiaRoot $Handle
            $all = $root.FindAll(
                [System.Windows.Automation.TreeScope]::Descendants,
                [System.Windows.Automation.Condition]::TrueCondition
            )
            foreach ($el in $all) {
                try {
                    $name = $el.Current.Name
                    if ($name -and $name.StartsWith($NamePrefix, [StringComparison]::Ordinal)) {
                        return $el
                    }
                } catch { }
            }
        } catch { }
        Start-Sleep -Milliseconds 250
    }
    return $null
}

function Get-UiaNames {
    param([IntPtr]$Handle, [int]$Limit = 40)
    $names = New-Object System.Collections.Generic.List[string]
    try {
        $root = Get-UiaRoot $Handle
        $all = $root.FindAll(
            [System.Windows.Automation.TreeScope]::Descendants,
            [System.Windows.Automation.Condition]::TrueCondition
        )
        foreach ($el in $all) {
            try {
                $name = $el.Current.Name
                if ($name) { $names.Add($name) }
                if ($names.Count -ge $Limit) { break }
            } catch { }
        }
    } catch { }
    return $names
}

function Invoke-UiaElement {
    param($Element)
    try {
        $pattern = $Element.GetCurrentPattern([System.Windows.Automation.InvokePattern]::Pattern)
        $pattern.Invoke()
        return 'invoke'
    } catch {
        $rect = $Element.Current.BoundingRectangle
        [HudToggle]::Click([int]($rect.X + $rect.Width / 2), [int]($rect.Y + $rect.Height / 2))
        return 'click'
    }
}

function Get-WindowByTitle {
    param([int]$ProcessId, [string]$Title)
    return @(Get-HudWindows -ProcessId $ProcessId | Where-Object { $_.Title -eq $Title }) | Select-Object -First 1
}

function Wait-WindowVisible {
    param([int]$ProcessId, [string]$Title, [int]$Seconds = 20)
    $deadline = [DateTime]::UtcNow.AddSeconds($Seconds)
    while ([DateTime]::UtcNow -lt $deadline) {
        $window = Get-WindowByTitle $ProcessId $Title
        if ($null -ne $window -and $window.Visible -and $window.W -gt 0) { return $window }
        Start-Sleep -Milliseconds 200
    }
    return $null
}

function Save-Screenshot {
    param([string]$Path, [int]$X, [int]$Y, [int]$Width, [int]$Height)
    $bmp = New-Object System.Drawing.Bitmap($Width, $Height)
    $g = [System.Drawing.Graphics]::FromImage($bmp)
    $g.CopyFromScreen($X, $Y, 0, 0, (New-Object System.Drawing.Size($Width, $Height)))
    $bmp.Save($Path, [System.Drawing.Imaging.ImageFormat]::Png)
    $g.Dispose(); $bmp.Dispose()
}

function Start-HudSession {
    param([string]$Tag)
    $process = Start-Process -FilePath $hudPath -PassThru `
        -RedirectStandardOutput (Join-Path $EvidenceDirectory "$Tag-stdout.log") `
        -RedirectStandardError (Join-Path $EvidenceDirectory "$Tag-stderr.log")
    $deadline = [DateTime]::UtcNow.AddSeconds($TimeoutSeconds)
    while ([DateTime]::UtcNow -lt $deadline) {
        $windows = @(Get-HudWindows -ProcessId $process.Id | Where-Object { $_.Visible -and $_.W -gt 0 })
        if ($windows.Count -gt 0) {
            return @{ Process = $process; Handle = $windows[0].Handle; Window = $windows[0] }
        }
        Start-Sleep -Milliseconds 250
    }
    Stop-Process -Id $process.Id -Force -ErrorAction SilentlyContinue
    throw "The HUD window did not appear in session $Tag."
}

function Stop-HudSession {
    param($Session)
    if ($null -ne $Session -and $null -ne $Session.Process -and -not $Session.Process.HasExited) {
        Stop-Process -Id $Session.Process.Id -Force -ErrorAction SilentlyContinue
        $Session.Process.WaitForExit(5000) | Out-Null
    }
}

# ---- isolated state shared by both sessions ------------------------------

$isolated = Join-Path $EvidenceDirectory 'isolated-state'
Remove-Item -Recurse -Force $isolated -ErrorAction SilentlyContinue
foreach ($key in @('NEXUS_DATA_DIR', 'NEXUS_CONFIG_DIR', 'NEXUS_STATE_DIR')) {
    $value = Join-Path $isolated $key
    New-Item -ItemType Directory -Path $value -Force | Out-Null
    [Environment]::SetEnvironmentVariable($key, $value, 'Process')
}
[Environment]::SetEnvironmentVariable('NEXUS_HUD_DETACHED_POPOVER', $null, 'Process')

$sessionA = $null
$sessionB = $null
try {
    # ===================== Session A: enable the popup =====================
    $sessionA = Start-HudSession -Tag 'session-a'
    $state = Wait-HudState { param($s) $null -ne $s.Appearance } 15
    if ($null -eq $state) { throw 'The HUD never wrote hud-window-state.json.' }
    $scale = [HudToggle]::Dpi($sessionA.Handle) / 96.0
    $compactW = [int]$state.CompactWidth
    $compactH = [int]$state.CompactHeight
    $minTrack = [HudToggle]::MinTrackWidth()
    $result.checks['scale'] = $scale
    $result.checks['windows_min_track_width'] = $minTrack
    $result.checks['a_state_detached_before'] = ($state.DetachedPopover -ne $true)
    $result.checks['a_appearance_before'] = $state.Appearance

    $main = Get-WindowByTitle $sessionA.Process.Id 'Nexus HUD'
    $result.checks['a_capsule_before'] = "$($main.X),$($main.Y),$($main.W),$($main.H)"

    # 1. legacy panel: toggle must say "Não" and Mascote must be disabled.
    [HudToggle]::Click([int]($main.X + $main.W / 2), [int]($main.Y + $main.H / 2))
    $toggle = Find-UiaElement -Handle $sessionA.Handle -NamePrefix 'Popup independente:' -Seconds 20
    if ($null -eq $toggle) {
        $result.checks['a_uia_sample'] = @(Get-UiaNames -Handle $sessionA.Handle -Limit 40)
        Write-Result
        throw 'The "Popup independente" toggle did not render.'
    }
    $result.checks['a_toggle_before'] = $toggle.Current.Name
    $result.checks['a_toggle_before_off'] = $toggle.Current.Name.EndsWith('N' + [char]0x00E3 + 'o')
    $mascotBefore = Find-UiaElement -Handle $sessionA.Handle -NamePrefix 'Mascote' -Seconds 5
    $result.checks['a_mascot_found_before'] = ($null -ne $mascotBefore)
    $result.checks['a_mascot_disabled_before'] = [bool](
        $null -ne $mascotBefore -and -not $mascotBefore.Current.IsEnabled
    )

    # 2. enable it: state persists, hidden popup exists, capsule stays compact.
    $result.checks['a_toggle_on_action'] = Invoke-UiaElement $toggle
    $enabledState = Wait-HudState { param($s) $s.DetachedPopover -eq $true } 20
    $result.checks['a_state_detached_on'] = ($null -ne $enabledState)
    if ($null -eq $enabledState) { throw 'The popup preference was not persisted.' }
    Start-Sleep -Milliseconds 900

    $popup = Get-WindowByTitle $sessionA.Process.Id 'Nexus HUD Popup'
    $result.checks['a_popup_window_created'] = ($null -ne $popup)
    $result.checks['a_popup_hidden_before_expand'] = [bool]($null -ne $popup -and -not $popup.Visible)

    $restoredW = [int][Math]::Round($compactW * $scale)
    $restoredH = [int][Math]::Round($compactH * $scale)
    $main = Get-WindowByTitle $sessionA.Process.Id 'Nexus HUD'
    $result.checks['a_capsule_after_enable'] = "$($main.X),$($main.Y),$($main.W),$($main.H)"
    $result.checks['a_capsule_restored'] = [bool](
        [Math]::Abs($main.W - $restoredW) -le 2 -and [Math]::Abs($main.H - $restoredH) -le 2
    )

    # 3. expand again: the popup shows and the capsule does not grow.
    $expand = Find-UiaElement -Handle $sessionA.Handle -NamePrefix 'Expandir Nexus HUD' -Seconds 10
    if ($null -eq $expand) { throw 'The capsule expand button did not render.' }
    $result.checks['a_expand_action'] = Invoke-UiaElement $expand
    $popup = Wait-WindowVisible -ProcessId $sessionA.Process.Id -Title 'Nexus HUD Popup' -Seconds 20
    $result.checks['a_popup_visible_on_expand'] = ($null -ne $popup)
    if ($null -eq $popup) { throw 'The popup window never became visible.' }
    Start-Sleep -Seconds 2

    $main = Get-WindowByTitle $sessionA.Process.Id 'Nexus HUD'
    $result.checks['a_capsule_kept_compact'] = [bool](
        [Math]::Abs($main.W - $restoredW) -le 2 -and [Math]::Abs($main.H - $restoredH) -le 2
    )
    $result.checks['a_popup_bounds'] = "$($popup.X),$($popup.Y),$($popup.W),$($popup.H)"

    # 4. mascot becomes available inside the popup.
    $mascot = Find-UiaElement -Handle $popup.Handle -NamePrefix 'Mascote' -Seconds 20
    $result.checks['a_popup_mascot_found'] = ($null -ne $mascot)
    $result.checks['a_mascot_enabled_after'] = [bool]($null -ne $mascot -and $mascot.Current.IsEnabled)
    if ($null -eq $mascot -or -not $mascot.Current.IsEnabled) {
        $result.checks['a_popup_uia_sample'] = @(Get-UiaNames -Handle $popup.Handle -Limit 40)
        Write-Result
        throw 'Mascote did not become available after enabling the popup.'
    }
    $result.checks['a_mascot_action'] = Invoke-UiaElement $mascot

    $mascotState = Wait-HudState { param($s) $s.Appearance -eq 'MASCOT' } 20
    $result.checks['a_state_appearance_mascot'] = ($null -ne $mascotState)
    if ($null -eq $mascotState) { throw 'MASCOT was not persisted.' }
    Start-Sleep -Seconds 2

    $main = Get-WindowByTitle $sessionA.Process.Id 'Nexus HUD'
    # Windows clamps a resizable window to SM_CXMINTRACK, so the 108 DIP mascot
    # window may render wider on the host while keeping its height exact.
    $expectedW = [Math]::Max([int][Math]::Round(108 * $scale), $minTrack)
    $expectedH = [int][Math]::Round(112 * $scale)
    $result.checks['a_capsule_mascot_size'] = "$($main.W)x$($main.H)"
    $result.checks['a_capsule_mascot_expected'] = "${expectedW}x${expectedH}"
    $result.checks['a_capsule_is_mascot'] = [bool](
        [Math]::Abs($main.W - $expectedW) -le 2 -and [Math]::Abs($main.H - $expectedH) -le 2
    )
    # The capsule itself must render the mascot, not a stale bar.
    $capsuleMascotButton = Find-UiaElement -Handle $sessionA.Handle -NamePrefix 'Expandir Nexus HUD' -Seconds 10
    $result.checks['a_capsule_renders_mascot'] = ($null -ne $capsuleMascotButton)
    $popupAfterMascot = Get-WindowByTitle $sessionA.Process.Id 'Nexus HUD Popup'
    $result.checks['a_popup_hidden_after_mascot'] = [bool](
        $null -ne $popupAfterMascot -and -not $popupAfterMascot.Visible
    )
    Save-Screenshot -Path (Join-Path $EvidenceDirectory 'mascot.png') `
        -X ([Math]::Max(0, $main.X - 16)) -Y ([Math]::Max(0, $main.Y - 16)) `
        -Width ($main.W + 32) -Height ($main.H + 32)

    # Kill the process so session B proves the opt-in survives a restart.
    Stop-HudSession $sessionA
    $sessionA = $null
    Start-Sleep -Milliseconds 700

    # ============ Session B: restart, then disable from the popup ============
    $sessionB = Start-HudSession -Tag 'session-b'
    $restartState = Wait-HudState {
        param($s) $s.Appearance -eq 'MASCOT' -and $s.DetachedPopover -eq $true
    } 20
    $result.checks['b_state_restored'] = ($null -ne $restartState)
    if ($null -eq $restartState) { throw 'The popup opt-in or MASCOT did not survive a restart.' }
    Start-Sleep -Seconds 3

    $main = Get-WindowByTitle $sessionB.Process.Id 'Nexus HUD'
    $result.checks['b_capsule_at_start'] = "$($main.X),$($main.Y),$($main.W),$($main.H)"
    $result.checks['b_capsule_is_mascot'] = [bool](
        [Math]::Abs($main.W - $expectedW) -le 2 -and [Math]::Abs($main.H - $expectedH) -le 2
    )
    $popup = Get-WindowByTitle $sessionB.Process.Id 'Nexus HUD Popup'
    $result.checks['b_popup_created_at_start'] = ($null -ne $popup)
    $result.checks['b_popup_hidden_at_start'] = [bool]($null -ne $popup -and -not $popup.Visible)

    # 5. open the panel from the mascot.
    $mascotButton = Find-UiaElement -Handle $sessionB.Handle -NamePrefix 'Expandir Nexus HUD' -Seconds 15
    if ($null -eq $mascotButton) {
        $result.checks['b_capsule_uia_sample'] = @(Get-UiaNames -Handle $sessionB.Handle -Limit 40)
        Write-Result
        throw 'The mascot button did not render after the restart.'
    }
    $result.checks['b_mascot_expand_action'] = Invoke-UiaElement $mascotButton
    $popup = Wait-WindowVisible -ProcessId $sessionB.Process.Id -Title 'Nexus HUD Popup' -Seconds 20
    $result.checks['b_popup_visible_after_expand'] = ($null -ne $popup)
    if ($null -eq $popup) { throw 'The popup did not reopen from the mascot.' }
    Start-Sleep -Seconds 2

    # 6. disable the popup from inside the popup.
    $toggleOff = Find-UiaElement -Handle $popup.Handle -NamePrefix 'Popup independente:' -Seconds 20
    if ($null -eq $toggleOff) {
        $result.checks['b_popup_uia_sample'] = @(Get-UiaNames -Handle $popup.Handle -Limit 40)
        Save-Screenshot -Path (Join-Path $EvidenceDirectory 'popup-reopen.png') `
            -X ([Math]::Max(0, $popup.X - 8)) -Y ([Math]::Max(0, $popup.Y - 8)) `
            -Width ($popup.W + 16) -Height ($popup.H + 16)
        Write-Result
        throw 'The popup toggle did not render in the popup.'
    }
    $result.checks['b_toggle_off_before'] = $toggleOff.Current.Name
    $result.checks['b_toggle_off_action'] = Invoke-UiaElement $toggleOff

    $offState = Wait-HudState {
        param($s) $s.DetachedPopover -ne $true -and $s.Appearance -eq 'BAR'
    } 20
    $result.checks['b_state_detached_off'] = ($null -ne $offState)
    if ($null -eq $offState) { throw 'Disabling the popup did not persist BAR.' }
    Start-Sleep -Seconds 2

    $main = Get-WindowByTitle $sessionB.Process.Id 'Nexus HUD'
    $result.checks['b_capsule_after_disable'] = "$($main.X),$($main.Y),$($main.W),$($main.H)"
    $result.checks['b_capsule_restored'] = [bool](
        [Math]::Abs($main.W - $restoredW) -le 2 -and [Math]::Abs($main.H - $restoredH) -le 2
    )
    $popup = Get-WindowByTitle $sessionB.Process.Id 'Nexus HUD Popup'
    $result.checks['b_popup_hidden_after_disable'] = [bool]($null -ne $popup -and -not $popup.Visible)

    # 7. the legacy inline panel must still open (no stale detached state).
    [HudToggle]::Click([int]($main.X + $main.W / 2), [int]($main.Y + $main.H / 2))
    $inlineToggle = Find-UiaElement -Handle $sessionB.Handle -NamePrefix 'Popup independente:' -Seconds 20
    $result.checks['b_inline_panel_after_disable'] = ($null -ne $inlineToggle)
    if ($null -ne $inlineToggle) {
        $result.checks['b_toggle_after_disable'] = $inlineToggle.Current.Name
    }
    $main = Get-WindowByTitle $sessionB.Process.Id 'Nexus HUD'
    $result.checks['b_capsule_after_inline_open'] = "$($main.X),$($main.Y),$($main.W),$($main.H)"
    Save-Screenshot -Path (Join-Path $EvidenceDirectory 'bar-restored.png') `
        -X ([Math]::Max(0, $main.X - 16)) -Y ([Math]::Max(0, $main.Y - 16)) `
        -Width ($main.W + 32) -Height ($main.H + 32)

    $required = @(
        'sha_match',
        'a_state_detached_before', 'a_toggle_before_off', 'a_mascot_found_before',
        'a_mascot_disabled_before', 'a_state_detached_on', 'a_popup_window_created',
        'a_popup_hidden_before_expand', 'a_capsule_restored', 'a_popup_visible_on_expand',
        'a_capsule_kept_compact', 'a_popup_mascot_found', 'a_mascot_enabled_after',
        'a_state_appearance_mascot', 'a_capsule_is_mascot', 'a_capsule_renders_mascot',
        'a_popup_hidden_after_mascot',
        'b_state_restored', 'b_capsule_is_mascot', 'b_popup_created_at_start',
        'b_popup_hidden_at_start', 'b_popup_visible_after_expand', 'b_state_detached_off',
        'b_capsule_restored', 'b_popup_hidden_after_disable', 'b_inline_panel_after_disable'
    )
    $failed = @($required | Where-Object { $result.checks[$_] -ne $true })
    $result.checks['failed_checks'] = $failed
    $result.verdict = if ($failed.Count -eq 0) { 'PASS' } else { 'FAIL' }
    Write-Result
    if ($failed.Count -gt 0) { throw "Mascot toggle checks failed: $($failed -join ', ')" }
    Write-Output 'PASS'
} catch {
    $result.checks['error'] = $_.Exception.Message
    $result.verdict = 'FAIL'
    Write-Result
    throw
} finally {
    Stop-HudSession $sessionA
    Stop-HudSession $sessionB
    Write-Result
}
) {
    throw 'Unable to determine the exact source commit for the HUD test.'
}
Write-Result

# ExpectedSha is the 40-digit *source commit*, not the 64-digit SHA256 of
# an executable. Mixing these two identifiers falsely blocked every
# commit-pinned interactive mascot test before it even launched.
if ($ExpectedSha) {
    if ($ExpectedSha -notmatch '^[a-fA-F0-9]{40}

# ---- helpers -------------------------------------------------------------

function Get-HudWindows {
    param([int]$ProcessId)
    return @([HudToggle]::List($ProcessId) | Where-Object { $_.Class -eq 'WailsWebviewWindow' })
}

function Read-HudState {
    $path = Join-Path $env:NEXUS_DATA_DIR 'hud-window-state.json'
    if (-not (Test-Path $path)) { return $null }
    try { return (Get-Content $path -Raw | ConvertFrom-Json) } catch { return $null }
}

function Wait-HudState {
    param([scriptblock]$Predicate, [int]$Seconds = 15)
    $deadline = [DateTime]::UtcNow.AddSeconds($Seconds)
    while ([DateTime]::UtcNow -lt $deadline) {
        $state = Read-HudState
        if ($null -ne $state -and (& $Predicate $state)) { return $state }
        Start-Sleep -Milliseconds 150
    }
    return $null
}

function Get-UiaRoot {
    param([IntPtr]$Handle)
    return [System.Windows.Automation.AutomationElement]::FromHandle($Handle)
}

function Find-UiaElement {
    param(
        [IntPtr]$Handle,
        [string]$NamePrefix,
        [int]$Seconds = 10
    )
    $deadline = [DateTime]::UtcNow.AddSeconds($Seconds)
    while ([DateTime]::UtcNow -lt $deadline) {
        try {
            $root = Get-UiaRoot $Handle
            $all = $root.FindAll(
                [System.Windows.Automation.TreeScope]::Descendants,
                [System.Windows.Automation.Condition]::TrueCondition
            )
            foreach ($el in $all) {
                try {
                    $name = $el.Current.Name
                    if ($name -and $name.StartsWith($NamePrefix, [StringComparison]::Ordinal)) {
                        return $el
                    }
                } catch { }
            }
        } catch { }
        Start-Sleep -Milliseconds 250
    }
    return $null
}

function Get-UiaNames {
    param([IntPtr]$Handle, [int]$Limit = 40)
    $names = New-Object System.Collections.Generic.List[string]
    try {
        $root = Get-UiaRoot $Handle
        $all = $root.FindAll(
            [System.Windows.Automation.TreeScope]::Descendants,
            [System.Windows.Automation.Condition]::TrueCondition
        )
        foreach ($el in $all) {
            try {
                $name = $el.Current.Name
                if ($name) { $names.Add($name) }
                if ($names.Count -ge $Limit) { break }
            } catch { }
        }
    } catch { }
    return $names
}

function Invoke-UiaElement {
    param($Element)
    try {
        $pattern = $Element.GetCurrentPattern([System.Windows.Automation.InvokePattern]::Pattern)
        $pattern.Invoke()
        return 'invoke'
    } catch {
        $rect = $Element.Current.BoundingRectangle
        [HudToggle]::Click([int]($rect.X + $rect.Width / 2), [int]($rect.Y + $rect.Height / 2))
        return 'click'
    }
}

function Get-WindowByTitle {
    param([int]$ProcessId, [string]$Title)
    return @(Get-HudWindows -ProcessId $ProcessId | Where-Object { $_.Title -eq $Title }) | Select-Object -First 1
}

function Wait-WindowVisible {
    param([int]$ProcessId, [string]$Title, [int]$Seconds = 20)
    $deadline = [DateTime]::UtcNow.AddSeconds($Seconds)
    while ([DateTime]::UtcNow -lt $deadline) {
        $window = Get-WindowByTitle $ProcessId $Title
        if ($null -ne $window -and $window.Visible -and $window.W -gt 0) { return $window }
        Start-Sleep -Milliseconds 200
    }
    return $null
}

function Save-Screenshot {
    param([string]$Path, [int]$X, [int]$Y, [int]$Width, [int]$Height)
    $bmp = New-Object System.Drawing.Bitmap($Width, $Height)
    $g = [System.Drawing.Graphics]::FromImage($bmp)
    $g.CopyFromScreen($X, $Y, 0, 0, (New-Object System.Drawing.Size($Width, $Height)))
    $bmp.Save($Path, [System.Drawing.Imaging.ImageFormat]::Png)
    $g.Dispose(); $bmp.Dispose()
}

function Start-HudSession {
    param([string]$Tag)
    $process = Start-Process -FilePath $hudPath -PassThru `
        -RedirectStandardOutput (Join-Path $EvidenceDirectory "$Tag-stdout.log") `
        -RedirectStandardError (Join-Path $EvidenceDirectory "$Tag-stderr.log")
    $deadline = [DateTime]::UtcNow.AddSeconds($TimeoutSeconds)
    while ([DateTime]::UtcNow -lt $deadline) {
        $windows = @(Get-HudWindows -ProcessId $process.Id | Where-Object { $_.Visible -and $_.W -gt 0 })
        if ($windows.Count -gt 0) {
            return @{ Process = $process; Handle = $windows[0].Handle; Window = $windows[0] }
        }
        Start-Sleep -Milliseconds 250
    }
    Stop-Process -Id $process.Id -Force -ErrorAction SilentlyContinue
    throw "The HUD window did not appear in session $Tag."
}

function Stop-HudSession {
    param($Session)
    if ($null -ne $Session -and $null -ne $Session.Process -and -not $Session.Process.HasExited) {
        Stop-Process -Id $Session.Process.Id -Force -ErrorAction SilentlyContinue
        $Session.Process.WaitForExit(5000) | Out-Null
    }
}

# ---- isolated state shared by both sessions ------------------------------

$isolated = Join-Path $EvidenceDirectory 'isolated-state'
Remove-Item -Recurse -Force $isolated -ErrorAction SilentlyContinue
foreach ($key in @('NEXUS_DATA_DIR', 'NEXUS_CONFIG_DIR', 'NEXUS_STATE_DIR')) {
    $value = Join-Path $isolated $key
    New-Item -ItemType Directory -Path $value -Force | Out-Null
    [Environment]::SetEnvironmentVariable($key, $value, 'Process')
}
[Environment]::SetEnvironmentVariable('NEXUS_HUD_DETACHED_POPOVER', $null, 'Process')

$sessionA = $null
$sessionB = $null
try {
    # ===================== Session A: enable the popup =====================
    $sessionA = Start-HudSession -Tag 'session-a'
    $state = Wait-HudState { param($s) $null -ne $s.Appearance } 15
    if ($null -eq $state) { throw 'The HUD never wrote hud-window-state.json.' }
    $scale = [HudToggle]::Dpi($sessionA.Handle) / 96.0
    $compactW = [int]$state.CompactWidth
    $compactH = [int]$state.CompactHeight
    $minTrack = [HudToggle]::MinTrackWidth()
    $result.checks['scale'] = $scale
    $result.checks['windows_min_track_width'] = $minTrack
    $result.checks['a_state_detached_before'] = ($state.DetachedPopover -ne $true)
    $result.checks['a_appearance_before'] = $state.Appearance

    $main = Get-WindowByTitle $sessionA.Process.Id 'Nexus HUD'
    $result.checks['a_capsule_before'] = "$($main.X),$($main.Y),$($main.W),$($main.H)"

    # 1. legacy panel: toggle must say "Não" and Mascote must be disabled.
    [HudToggle]::Click([int]($main.X + $main.W / 2), [int]($main.Y + $main.H / 2))
    $toggle = Find-UiaElement -Handle $sessionA.Handle -NamePrefix 'Popup independente:' -Seconds 20
    if ($null -eq $toggle) {
        $result.checks['a_uia_sample'] = @(Get-UiaNames -Handle $sessionA.Handle -Limit 40)
        Write-Result
        throw 'The "Popup independente" toggle did not render.'
    }
    $result.checks['a_toggle_before'] = $toggle.Current.Name
    $result.checks['a_toggle_before_off'] = $toggle.Current.Name.EndsWith('N' + [char]0x00E3 + 'o')
    $mascotBefore = Find-UiaElement -Handle $sessionA.Handle -NamePrefix 'Mascote' -Seconds 5
    $result.checks['a_mascot_found_before'] = ($null -ne $mascotBefore)
    $result.checks['a_mascot_disabled_before'] = [bool](
        $null -ne $mascotBefore -and -not $mascotBefore.Current.IsEnabled
    )

    # 2. enable it: state persists, hidden popup exists, capsule stays compact.
    $result.checks['a_toggle_on_action'] = Invoke-UiaElement $toggle
    $enabledState = Wait-HudState { param($s) $s.DetachedPopover -eq $true } 20
    $result.checks['a_state_detached_on'] = ($null -ne $enabledState)
    if ($null -eq $enabledState) { throw 'The popup preference was not persisted.' }
    Start-Sleep -Milliseconds 900

    $popup = Get-WindowByTitle $sessionA.Process.Id 'Nexus HUD Popup'
    $result.checks['a_popup_window_created'] = ($null -ne $popup)
    $result.checks['a_popup_hidden_before_expand'] = [bool]($null -ne $popup -and -not $popup.Visible)

    $restoredW = [int][Math]::Round($compactW * $scale)
    $restoredH = [int][Math]::Round($compactH * $scale)
    $main = Get-WindowByTitle $sessionA.Process.Id 'Nexus HUD'
    $result.checks['a_capsule_after_enable'] = "$($main.X),$($main.Y),$($main.W),$($main.H)"
    $result.checks['a_capsule_restored'] = [bool](
        [Math]::Abs($main.W - $restoredW) -le 2 -and [Math]::Abs($main.H - $restoredH) -le 2
    )

    # 3. expand again: the popup shows and the capsule does not grow.
    $expand = Find-UiaElement -Handle $sessionA.Handle -NamePrefix 'Expandir Nexus HUD' -Seconds 10
    if ($null -eq $expand) { throw 'The capsule expand button did not render.' }
    $result.checks['a_expand_action'] = Invoke-UiaElement $expand
    $popup = Wait-WindowVisible -ProcessId $sessionA.Process.Id -Title 'Nexus HUD Popup' -Seconds 20
    $result.checks['a_popup_visible_on_expand'] = ($null -ne $popup)
    if ($null -eq $popup) { throw 'The popup window never became visible.' }
    Start-Sleep -Seconds 2

    $main = Get-WindowByTitle $sessionA.Process.Id 'Nexus HUD'
    $result.checks['a_capsule_kept_compact'] = [bool](
        [Math]::Abs($main.W - $restoredW) -le 2 -and [Math]::Abs($main.H - $restoredH) -le 2
    )
    $result.checks['a_popup_bounds'] = "$($popup.X),$($popup.Y),$($popup.W),$($popup.H)"

    # 4. mascot becomes available inside the popup.
    $mascot = Find-UiaElement -Handle $popup.Handle -NamePrefix 'Mascote' -Seconds 20
    $result.checks['a_popup_mascot_found'] = ($null -ne $mascot)
    $result.checks['a_mascot_enabled_after'] = [bool]($null -ne $mascot -and $mascot.Current.IsEnabled)
    if ($null -eq $mascot -or -not $mascot.Current.IsEnabled) {
        $result.checks['a_popup_uia_sample'] = @(Get-UiaNames -Handle $popup.Handle -Limit 40)
        Write-Result
        throw 'Mascote did not become available after enabling the popup.'
    }
    $result.checks['a_mascot_action'] = Invoke-UiaElement $mascot

    $mascotState = Wait-HudState { param($s) $s.Appearance -eq 'MASCOT' } 20
    $result.checks['a_state_appearance_mascot'] = ($null -ne $mascotState)
    if ($null -eq $mascotState) { throw 'MASCOT was not persisted.' }
    Start-Sleep -Seconds 2

    $main = Get-WindowByTitle $sessionA.Process.Id 'Nexus HUD'
    # Windows clamps a resizable window to SM_CXMINTRACK, so the 108 DIP mascot
    # window may render wider on the host while keeping its height exact.
    $expectedW = [Math]::Max([int][Math]::Round(108 * $scale), $minTrack)
    $expectedH = [int][Math]::Round(112 * $scale)
    $result.checks['a_capsule_mascot_size'] = "$($main.W)x$($main.H)"
    $result.checks['a_capsule_mascot_expected'] = "${expectedW}x${expectedH}"
    $result.checks['a_capsule_is_mascot'] = [bool](
        [Math]::Abs($main.W - $expectedW) -le 2 -and [Math]::Abs($main.H - $expectedH) -le 2
    )
    # The capsule itself must render the mascot, not a stale bar.
    $capsuleMascotButton = Find-UiaElement -Handle $sessionA.Handle -NamePrefix 'Expandir Nexus HUD' -Seconds 10
    $result.checks['a_capsule_renders_mascot'] = ($null -ne $capsuleMascotButton)
    $popupAfterMascot = Get-WindowByTitle $sessionA.Process.Id 'Nexus HUD Popup'
    $result.checks['a_popup_hidden_after_mascot'] = [bool](
        $null -ne $popupAfterMascot -and -not $popupAfterMascot.Visible
    )
    Save-Screenshot -Path (Join-Path $EvidenceDirectory 'mascot.png') `
        -X ([Math]::Max(0, $main.X - 16)) -Y ([Math]::Max(0, $main.Y - 16)) `
        -Width ($main.W + 32) -Height ($main.H + 32)

    # Kill the process so session B proves the opt-in survives a restart.
    Stop-HudSession $sessionA
    $sessionA = $null
    Start-Sleep -Milliseconds 700

    # ============ Session B: restart, then disable from the popup ============
    $sessionB = Start-HudSession -Tag 'session-b'
    $restartState = Wait-HudState {
        param($s) $s.Appearance -eq 'MASCOT' -and $s.DetachedPopover -eq $true
    } 20
    $result.checks['b_state_restored'] = ($null -ne $restartState)
    if ($null -eq $restartState) { throw 'The popup opt-in or MASCOT did not survive a restart.' }
    Start-Sleep -Seconds 3

    $main = Get-WindowByTitle $sessionB.Process.Id 'Nexus HUD'
    $result.checks['b_capsule_at_start'] = "$($main.X),$($main.Y),$($main.W),$($main.H)"
    $result.checks['b_capsule_is_mascot'] = [bool](
        [Math]::Abs($main.W - $expectedW) -le 2 -and [Math]::Abs($main.H - $expectedH) -le 2
    )
    $popup = Get-WindowByTitle $sessionB.Process.Id 'Nexus HUD Popup'
    $result.checks['b_popup_created_at_start'] = ($null -ne $popup)
    $result.checks['b_popup_hidden_at_start'] = [bool]($null -ne $popup -and -not $popup.Visible)

    # 5. open the panel from the mascot.
    $mascotButton = Find-UiaElement -Handle $sessionB.Handle -NamePrefix 'Expandir Nexus HUD' -Seconds 15
    if ($null -eq $mascotButton) {
        $result.checks['b_capsule_uia_sample'] = @(Get-UiaNames -Handle $sessionB.Handle -Limit 40)
        Write-Result
        throw 'The mascot button did not render after the restart.'
    }
    $result.checks['b_mascot_expand_action'] = Invoke-UiaElement $mascotButton
    $popup = Wait-WindowVisible -ProcessId $sessionB.Process.Id -Title 'Nexus HUD Popup' -Seconds 20
    $result.checks['b_popup_visible_after_expand'] = ($null -ne $popup)
    if ($null -eq $popup) { throw 'The popup did not reopen from the mascot.' }
    Start-Sleep -Seconds 2

    # 6. disable the popup from inside the popup.
    $toggleOff = Find-UiaElement -Handle $popup.Handle -NamePrefix 'Popup independente:' -Seconds 20
    if ($null -eq $toggleOff) {
        $result.checks['b_popup_uia_sample'] = @(Get-UiaNames -Handle $popup.Handle -Limit 40)
        Save-Screenshot -Path (Join-Path $EvidenceDirectory 'popup-reopen.png') `
            -X ([Math]::Max(0, $popup.X - 8)) -Y ([Math]::Max(0, $popup.Y - 8)) `
            -Width ($popup.W + 16) -Height ($popup.H + 16)
        Write-Result
        throw 'The popup toggle did not render in the popup.'
    }
    $result.checks['b_toggle_off_before'] = $toggleOff.Current.Name
    $result.checks['b_toggle_off_action'] = Invoke-UiaElement $toggleOff

    $offState = Wait-HudState {
        param($s) $s.DetachedPopover -ne $true -and $s.Appearance -eq 'BAR'
    } 20
    $result.checks['b_state_detached_off'] = ($null -ne $offState)
    if ($null -eq $offState) { throw 'Disabling the popup did not persist BAR.' }
    Start-Sleep -Seconds 2

    $main = Get-WindowByTitle $sessionB.Process.Id 'Nexus HUD'
    $result.checks['b_capsule_after_disable'] = "$($main.X),$($main.Y),$($main.W),$($main.H)"
    $result.checks['b_capsule_restored'] = [bool](
        [Math]::Abs($main.W - $restoredW) -le 2 -and [Math]::Abs($main.H - $restoredH) -le 2
    )
    $popup = Get-WindowByTitle $sessionB.Process.Id 'Nexus HUD Popup'
    $result.checks['b_popup_hidden_after_disable'] = [bool]($null -ne $popup -and -not $popup.Visible)

    # 7. the legacy inline panel must still open (no stale detached state).
    [HudToggle]::Click([int]($main.X + $main.W / 2), [int]($main.Y + $main.H / 2))
    $inlineToggle = Find-UiaElement -Handle $sessionB.Handle -NamePrefix 'Popup independente:' -Seconds 20
    $result.checks['b_inline_panel_after_disable'] = ($null -ne $inlineToggle)
    if ($null -ne $inlineToggle) {
        $result.checks['b_toggle_after_disable'] = $inlineToggle.Current.Name
    }
    $main = Get-WindowByTitle $sessionB.Process.Id 'Nexus HUD'
    $result.checks['b_capsule_after_inline_open'] = "$($main.X),$($main.Y),$($main.W),$($main.H)"
    Save-Screenshot -Path (Join-Path $EvidenceDirectory 'bar-restored.png') `
        -X ([Math]::Max(0, $main.X - 16)) -Y ([Math]::Max(0, $main.Y - 16)) `
        -Width ($main.W + 32) -Height ($main.H + 32)

    $required = @(
        'sha_match',
        'a_state_detached_before', 'a_toggle_before_off', 'a_mascot_found_before',
        'a_mascot_disabled_before', 'a_state_detached_on', 'a_popup_window_created',
        'a_popup_hidden_before_expand', 'a_capsule_restored', 'a_popup_visible_on_expand',
        'a_capsule_kept_compact', 'a_popup_mascot_found', 'a_mascot_enabled_after',
        'a_state_appearance_mascot', 'a_capsule_is_mascot', 'a_capsule_renders_mascot',
        'a_popup_hidden_after_mascot',
        'b_state_restored', 'b_capsule_is_mascot', 'b_popup_created_at_start',
        'b_popup_hidden_at_start', 'b_popup_visible_after_expand', 'b_state_detached_off',
        'b_capsule_restored', 'b_popup_hidden_after_disable', 'b_inline_panel_after_disable'
    )
    $failed = @($required | Where-Object { $result.checks[$_] -ne $true })
    $result.checks['failed_checks'] = $failed
    $result.verdict = if ($failed.Count -eq 0) { 'PASS' } else { 'FAIL' }
    Write-Result
    if ($failed.Count -gt 0) { throw "Mascot toggle checks failed: $($failed -join ', ')" }
    Write-Output 'PASS'
} catch {
    $result.checks['error'] = $_.Exception.Message
    $result.verdict = 'FAIL'
    Write-Result
    throw
} finally {
    Stop-HudSession $sessionA
    Stop-HudSession $sessionB
    Write-Result
}
) {
        throw 'ExpectedSha must be the 40-character git source commit. The executable digest is recorded separately.'
    }
    if ($result.commit -ne $ExpectedSha.ToLower()) {
        $result.checks['sha_match'] = $false
        Write-Result
        throw "HUD source commit mismatch: expected $ExpectedSha, checkout=$($result.commit)"
    }
}
$result.checks['sha_match'] = $true
# A matching checkout alone is not proof the supplied binary came from it;
# independent build provenance must still verify hud_sha256.

# ---- helpers -------------------------------------------------------------

function Get-HudWindows {
    param([int]$ProcessId)
    return @([HudToggle]::List($ProcessId) | Where-Object { $_.Class -eq 'WailsWebviewWindow' })
}

function Read-HudState {
    $path = Join-Path $env:NEXUS_DATA_DIR 'hud-window-state.json'
    if (-not (Test-Path $path)) { return $null }
    try { return (Get-Content $path -Raw | ConvertFrom-Json) } catch { return $null }
}

function Wait-HudState {
    param([scriptblock]$Predicate, [int]$Seconds = 15)
    $deadline = [DateTime]::UtcNow.AddSeconds($Seconds)
    while ([DateTime]::UtcNow -lt $deadline) {
        $state = Read-HudState
        if ($null -ne $state -and (& $Predicate $state)) { return $state }
        Start-Sleep -Milliseconds 150
    }
    return $null
}

function Get-UiaRoot {
    param([IntPtr]$Handle)
    return [System.Windows.Automation.AutomationElement]::FromHandle($Handle)
}

function Find-UiaElement {
    param(
        [IntPtr]$Handle,
        [string]$NamePrefix,
        [int]$Seconds = 10
    )
    $deadline = [DateTime]::UtcNow.AddSeconds($Seconds)
    while ([DateTime]::UtcNow -lt $deadline) {
        try {
            $root = Get-UiaRoot $Handle
            $all = $root.FindAll(
                [System.Windows.Automation.TreeScope]::Descendants,
                [System.Windows.Automation.Condition]::TrueCondition
            )
            foreach ($el in $all) {
                try {
                    $name = $el.Current.Name
                    if ($name -and $name.StartsWith($NamePrefix, [StringComparison]::Ordinal)) {
                        return $el
                    }
                } catch { }
            }
        } catch { }
        Start-Sleep -Milliseconds 250
    }
    return $null
}

function Get-UiaNames {
    param([IntPtr]$Handle, [int]$Limit = 40)
    $names = New-Object System.Collections.Generic.List[string]
    try {
        $root = Get-UiaRoot $Handle
        $all = $root.FindAll(
            [System.Windows.Automation.TreeScope]::Descendants,
            [System.Windows.Automation.Condition]::TrueCondition
        )
        foreach ($el in $all) {
            try {
                $name = $el.Current.Name
                if ($name) { $names.Add($name) }
                if ($names.Count -ge $Limit) { break }
            } catch { }
        }
    } catch { }
    return $names
}

function Invoke-UiaElement {
    param($Element)
    try {
        $pattern = $Element.GetCurrentPattern([System.Windows.Automation.InvokePattern]::Pattern)
        $pattern.Invoke()
        return 'invoke'
    } catch {
        $rect = $Element.Current.BoundingRectangle
        [HudToggle]::Click([int]($rect.X + $rect.Width / 2), [int]($rect.Y + $rect.Height / 2))
        return 'click'
    }
}

function Get-WindowByTitle {
    param([int]$ProcessId, [string]$Title)
    return @(Get-HudWindows -ProcessId $ProcessId | Where-Object { $_.Title -eq $Title }) | Select-Object -First 1
}

function Wait-WindowVisible {
    param([int]$ProcessId, [string]$Title, [int]$Seconds = 20)
    $deadline = [DateTime]::UtcNow.AddSeconds($Seconds)
    while ([DateTime]::UtcNow -lt $deadline) {
        $window = Get-WindowByTitle $ProcessId $Title
        if ($null -ne $window -and $window.Visible -and $window.W -gt 0) { return $window }
        Start-Sleep -Milliseconds 200
    }
    return $null
}

function Save-Screenshot {
    param([string]$Path, [int]$X, [int]$Y, [int]$Width, [int]$Height)
    $bmp = New-Object System.Drawing.Bitmap($Width, $Height)
    $g = [System.Drawing.Graphics]::FromImage($bmp)
    $g.CopyFromScreen($X, $Y, 0, 0, (New-Object System.Drawing.Size($Width, $Height)))
    $bmp.Save($Path, [System.Drawing.Imaging.ImageFormat]::Png)
    $g.Dispose(); $bmp.Dispose()
}

function Start-HudSession {
    param([string]$Tag)
    $process = Start-Process -FilePath $hudPath -PassThru `
        -RedirectStandardOutput (Join-Path $EvidenceDirectory "$Tag-stdout.log") `
        -RedirectStandardError (Join-Path $EvidenceDirectory "$Tag-stderr.log")
    $deadline = [DateTime]::UtcNow.AddSeconds($TimeoutSeconds)
    while ([DateTime]::UtcNow -lt $deadline) {
        $windows = @(Get-HudWindows -ProcessId $process.Id | Where-Object { $_.Visible -and $_.W -gt 0 })
        if ($windows.Count -gt 0) {
            return @{ Process = $process; Handle = $windows[0].Handle; Window = $windows[0] }
        }
        Start-Sleep -Milliseconds 250
    }
    Stop-Process -Id $process.Id -Force -ErrorAction SilentlyContinue
    throw "The HUD window did not appear in session $Tag."
}

function Stop-HudSession {
    param($Session)
    if ($null -ne $Session -and $null -ne $Session.Process -and -not $Session.Process.HasExited) {
        Stop-Process -Id $Session.Process.Id -Force -ErrorAction SilentlyContinue
        $Session.Process.WaitForExit(5000) | Out-Null
    }
}

# ---- isolated state shared by both sessions ------------------------------

$isolated = Join-Path $EvidenceDirectory 'isolated-state'
Remove-Item -Recurse -Force $isolated -ErrorAction SilentlyContinue
foreach ($key in @('NEXUS_DATA_DIR', 'NEXUS_CONFIG_DIR', 'NEXUS_STATE_DIR')) {
    $value = Join-Path $isolated $key
    New-Item -ItemType Directory -Path $value -Force | Out-Null
    [Environment]::SetEnvironmentVariable($key, $value, 'Process')
}
[Environment]::SetEnvironmentVariable('NEXUS_HUD_DETACHED_POPOVER', $null, 'Process')

$sessionA = $null
$sessionB = $null
try {
    # ===================== Session A: enable the popup =====================
    $sessionA = Start-HudSession -Tag 'session-a'
    $state = Wait-HudState { param($s) $null -ne $s.Appearance } 15
    if ($null -eq $state) { throw 'The HUD never wrote hud-window-state.json.' }
    $scale = [HudToggle]::Dpi($sessionA.Handle) / 96.0
    $compactW = [int]$state.CompactWidth
    $compactH = [int]$state.CompactHeight
    $minTrack = [HudToggle]::MinTrackWidth()
    $result.checks['scale'] = $scale
    $result.checks['windows_min_track_width'] = $minTrack
    $result.checks['a_state_detached_before'] = ($state.DetachedPopover -ne $true)
    $result.checks['a_appearance_before'] = $state.Appearance

    $main = Get-WindowByTitle $sessionA.Process.Id 'Nexus HUD'
    $result.checks['a_capsule_before'] = "$($main.X),$($main.Y),$($main.W),$($main.H)"

    # 1. legacy panel: toggle must say "Não" and Mascote must be disabled.
    [HudToggle]::Click([int]($main.X + $main.W / 2), [int]($main.Y + $main.H / 2))
    $toggle = Find-UiaElement -Handle $sessionA.Handle -NamePrefix 'Popup independente:' -Seconds 20
    if ($null -eq $toggle) {
        $result.checks['a_uia_sample'] = @(Get-UiaNames -Handle $sessionA.Handle -Limit 40)
        Write-Result
        throw 'The "Popup independente" toggle did not render.'
    }
    $result.checks['a_toggle_before'] = $toggle.Current.Name
    $result.checks['a_toggle_before_off'] = $toggle.Current.Name.EndsWith('N' + [char]0x00E3 + 'o')
    $mascotBefore = Find-UiaElement -Handle $sessionA.Handle -NamePrefix 'Mascote' -Seconds 5
    $result.checks['a_mascot_found_before'] = ($null -ne $mascotBefore)
    $result.checks['a_mascot_disabled_before'] = [bool](
        $null -ne $mascotBefore -and -not $mascotBefore.Current.IsEnabled
    )

    # 2. enable it: state persists, hidden popup exists, capsule stays compact.
    $result.checks['a_toggle_on_action'] = Invoke-UiaElement $toggle
    $enabledState = Wait-HudState { param($s) $s.DetachedPopover -eq $true } 20
    $result.checks['a_state_detached_on'] = ($null -ne $enabledState)
    if ($null -eq $enabledState) { throw 'The popup preference was not persisted.' }
    Start-Sleep -Milliseconds 900

    $popup = Get-WindowByTitle $sessionA.Process.Id 'Nexus HUD Popup'
    $result.checks['a_popup_window_created'] = ($null -ne $popup)
    $result.checks['a_popup_hidden_before_expand'] = [bool]($null -ne $popup -and -not $popup.Visible)

    $restoredW = [int][Math]::Round($compactW * $scale)
    $restoredH = [int][Math]::Round($compactH * $scale)
    $main = Get-WindowByTitle $sessionA.Process.Id 'Nexus HUD'
    $result.checks['a_capsule_after_enable'] = "$($main.X),$($main.Y),$($main.W),$($main.H)"
    $result.checks['a_capsule_restored'] = [bool](
        [Math]::Abs($main.W - $restoredW) -le 2 -and [Math]::Abs($main.H - $restoredH) -le 2
    )

    # 3. expand again: the popup shows and the capsule does not grow.
    $expand = Find-UiaElement -Handle $sessionA.Handle -NamePrefix 'Expandir Nexus HUD' -Seconds 10
    if ($null -eq $expand) { throw 'The capsule expand button did not render.' }
    $result.checks['a_expand_action'] = Invoke-UiaElement $expand
    $popup = Wait-WindowVisible -ProcessId $sessionA.Process.Id -Title 'Nexus HUD Popup' -Seconds 20
    $result.checks['a_popup_visible_on_expand'] = ($null -ne $popup)
    if ($null -eq $popup) { throw 'The popup window never became visible.' }
    Start-Sleep -Seconds 2

    $main = Get-WindowByTitle $sessionA.Process.Id 'Nexus HUD'
    $result.checks['a_capsule_kept_compact'] = [bool](
        [Math]::Abs($main.W - $restoredW) -le 2 -and [Math]::Abs($main.H - $restoredH) -le 2
    )
    $result.checks['a_popup_bounds'] = "$($popup.X),$($popup.Y),$($popup.W),$($popup.H)"

    # 4. mascot becomes available inside the popup.
    $mascot = Find-UiaElement -Handle $popup.Handle -NamePrefix 'Mascote' -Seconds 20
    $result.checks['a_popup_mascot_found'] = ($null -ne $mascot)
    $result.checks['a_mascot_enabled_after'] = [bool]($null -ne $mascot -and $mascot.Current.IsEnabled)
    if ($null -eq $mascot -or -not $mascot.Current.IsEnabled) {
        $result.checks['a_popup_uia_sample'] = @(Get-UiaNames -Handle $popup.Handle -Limit 40)
        Write-Result
        throw 'Mascote did not become available after enabling the popup.'
    }
    $result.checks['a_mascot_action'] = Invoke-UiaElement $mascot

    $mascotState = Wait-HudState { param($s) $s.Appearance -eq 'MASCOT' } 20
    $result.checks['a_state_appearance_mascot'] = ($null -ne $mascotState)
    if ($null -eq $mascotState) { throw 'MASCOT was not persisted.' }
    Start-Sleep -Seconds 2

    $main = Get-WindowByTitle $sessionA.Process.Id 'Nexus HUD'
    # Windows clamps a resizable window to SM_CXMINTRACK, so the 108 DIP mascot
    # window may render wider on the host while keeping its height exact.
    $expectedW = [Math]::Max([int][Math]::Round(108 * $scale), $minTrack)
    $expectedH = [int][Math]::Round(112 * $scale)
    $result.checks['a_capsule_mascot_size'] = "$($main.W)x$($main.H)"
    $result.checks['a_capsule_mascot_expected'] = "${expectedW}x${expectedH}"
    $result.checks['a_capsule_is_mascot'] = [bool](
        [Math]::Abs($main.W - $expectedW) -le 2 -and [Math]::Abs($main.H - $expectedH) -le 2
    )
    # The capsule itself must render the mascot, not a stale bar.
    $capsuleMascotButton = Find-UiaElement -Handle $sessionA.Handle -NamePrefix 'Expandir Nexus HUD' -Seconds 10
    $result.checks['a_capsule_renders_mascot'] = ($null -ne $capsuleMascotButton)
    $popupAfterMascot = Get-WindowByTitle $sessionA.Process.Id 'Nexus HUD Popup'
    $result.checks['a_popup_hidden_after_mascot'] = [bool](
        $null -ne $popupAfterMascot -and -not $popupAfterMascot.Visible
    )
    Save-Screenshot -Path (Join-Path $EvidenceDirectory 'mascot.png') `
        -X ([Math]::Max(0, $main.X - 16)) -Y ([Math]::Max(0, $main.Y - 16)) `
        -Width ($main.W + 32) -Height ($main.H + 32)

    # Kill the process so session B proves the opt-in survives a restart.
    Stop-HudSession $sessionA
    $sessionA = $null
    Start-Sleep -Milliseconds 700

    # ============ Session B: restart, then disable from the popup ============
    $sessionB = Start-HudSession -Tag 'session-b'
    $restartState = Wait-HudState {
        param($s) $s.Appearance -eq 'MASCOT' -and $s.DetachedPopover -eq $true
    } 20
    $result.checks['b_state_restored'] = ($null -ne $restartState)
    if ($null -eq $restartState) { throw 'The popup opt-in or MASCOT did not survive a restart.' }
    Start-Sleep -Seconds 3

    $main = Get-WindowByTitle $sessionB.Process.Id 'Nexus HUD'
    $result.checks['b_capsule_at_start'] = "$($main.X),$($main.Y),$($main.W),$($main.H)"
    $result.checks['b_capsule_is_mascot'] = [bool](
        [Math]::Abs($main.W - $expectedW) -le 2 -and [Math]::Abs($main.H - $expectedH) -le 2
    )
    $popup = Get-WindowByTitle $sessionB.Process.Id 'Nexus HUD Popup'
    $result.checks['b_popup_created_at_start'] = ($null -ne $popup)
    $result.checks['b_popup_hidden_at_start'] = [bool]($null -ne $popup -and -not $popup.Visible)

    # 5. open the panel from the mascot.
    $mascotButton = Find-UiaElement -Handle $sessionB.Handle -NamePrefix 'Expandir Nexus HUD' -Seconds 15
    if ($null -eq $mascotButton) {
        $result.checks['b_capsule_uia_sample'] = @(Get-UiaNames -Handle $sessionB.Handle -Limit 40)
        Write-Result
        throw 'The mascot button did not render after the restart.'
    }
    $result.checks['b_mascot_expand_action'] = Invoke-UiaElement $mascotButton
    $popup = Wait-WindowVisible -ProcessId $sessionB.Process.Id -Title 'Nexus HUD Popup' -Seconds 20
    $result.checks['b_popup_visible_after_expand'] = ($null -ne $popup)
    if ($null -eq $popup) { throw 'The popup did not reopen from the mascot.' }
    Start-Sleep -Seconds 2

    # 6. disable the popup from inside the popup.
    $toggleOff = Find-UiaElement -Handle $popup.Handle -NamePrefix 'Popup independente:' -Seconds 20
    if ($null -eq $toggleOff) {
        $result.checks['b_popup_uia_sample'] = @(Get-UiaNames -Handle $popup.Handle -Limit 40)
        Save-Screenshot -Path (Join-Path $EvidenceDirectory 'popup-reopen.png') `
            -X ([Math]::Max(0, $popup.X - 8)) -Y ([Math]::Max(0, $popup.Y - 8)) `
            -Width ($popup.W + 16) -Height ($popup.H + 16)
        Write-Result
        throw 'The popup toggle did not render in the popup.'
    }
    $result.checks['b_toggle_off_before'] = $toggleOff.Current.Name
    $result.checks['b_toggle_off_action'] = Invoke-UiaElement $toggleOff

    $offState = Wait-HudState {
        param($s) $s.DetachedPopover -ne $true -and $s.Appearance -eq 'BAR'
    } 20
    $result.checks['b_state_detached_off'] = ($null -ne $offState)
    if ($null -eq $offState) { throw 'Disabling the popup did not persist BAR.' }
    Start-Sleep -Seconds 2

    $main = Get-WindowByTitle $sessionB.Process.Id 'Nexus HUD'
    $result.checks['b_capsule_after_disable'] = "$($main.X),$($main.Y),$($main.W),$($main.H)"
    $result.checks['b_capsule_restored'] = [bool](
        [Math]::Abs($main.W - $restoredW) -le 2 -and [Math]::Abs($main.H - $restoredH) -le 2
    )
    $popup = Get-WindowByTitle $sessionB.Process.Id 'Nexus HUD Popup'
    $result.checks['b_popup_hidden_after_disable'] = [bool]($null -ne $popup -and -not $popup.Visible)

    # 7. the legacy inline panel must still open (no stale detached state).
    [HudToggle]::Click([int]($main.X + $main.W / 2), [int]($main.Y + $main.H / 2))
    $inlineToggle = Find-UiaElement -Handle $sessionB.Handle -NamePrefix 'Popup independente:' -Seconds 20
    $result.checks['b_inline_panel_after_disable'] = ($null -ne $inlineToggle)
    if ($null -ne $inlineToggle) {
        $result.checks['b_toggle_after_disable'] = $inlineToggle.Current.Name
    }
    $main = Get-WindowByTitle $sessionB.Process.Id 'Nexus HUD'
    $result.checks['b_capsule_after_inline_open'] = "$($main.X),$($main.Y),$($main.W),$($main.H)"
    Save-Screenshot -Path (Join-Path $EvidenceDirectory 'bar-restored.png') `
        -X ([Math]::Max(0, $main.X - 16)) -Y ([Math]::Max(0, $main.Y - 16)) `
        -Width ($main.W + 32) -Height ($main.H + 32)

    $required = @(
        'sha_match',
        'a_state_detached_before', 'a_toggle_before_off', 'a_mascot_found_before',
        'a_mascot_disabled_before', 'a_state_detached_on', 'a_popup_window_created',
        'a_popup_hidden_before_expand', 'a_capsule_restored', 'a_popup_visible_on_expand',
        'a_capsule_kept_compact', 'a_popup_mascot_found', 'a_mascot_enabled_after',
        'a_state_appearance_mascot', 'a_capsule_is_mascot', 'a_capsule_renders_mascot',
        'a_popup_hidden_after_mascot',
        'b_state_restored', 'b_capsule_is_mascot', 'b_popup_created_at_start',
        'b_popup_hidden_at_start', 'b_popup_visible_after_expand', 'b_state_detached_off',
        'b_capsule_restored', 'b_popup_hidden_after_disable', 'b_inline_panel_after_disable'
    )
    $failed = @($required | Where-Object { $result.checks[$_] -ne $true })
    $result.checks['failed_checks'] = $failed
    $result.verdict = if ($failed.Count -eq 0) { 'PASS' } else { 'FAIL' }
    Write-Result
    if ($failed.Count -gt 0) { throw "Mascot toggle checks failed: $($failed -join ', ')" }
    Write-Output 'PASS'
} catch {
    $result.checks['error'] = $_.Exception.Message
    $result.verdict = 'FAIL'
    Write-Result
    throw
} finally {
    Stop-HudSession $sessionA
    Stop-HudSession $sessionB
    Write-Result
}
