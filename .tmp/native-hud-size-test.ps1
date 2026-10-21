param(
  [string]$OutDir = "C:\Dev\nexus-hud-validation\.tmp\native-hud-size",
  [int]$HudPid = 0
)
$ErrorActionPreference = "Stop"
Add-Type -AssemblyName System.Drawing
Add-Type @"
using System;
using System.Runtime.InteropServices;
public static class Win32HudSize {
  [StructLayout(LayoutKind.Sequential)] public struct RECT { public int Left; public int Top; public int Right; public int Bottom; }
  [DllImport("user32.dll")] public static extern bool GetWindowRect(IntPtr hWnd, out RECT lpRect);
  [DllImport("user32.dll")] public static extern bool SetForegroundWindow(IntPtr hWnd);
  [DllImport("user32.dll")] public static extern bool SetCursorPos(int X, int Y);
  [DllImport("user32.dll")] public static extern void mouse_event(uint dwFlags, uint dx, uint dy, uint dwData, UIntPtr dwExtraInfo);
  [DllImport("user32.dll")] public static extern IntPtr GetForegroundWindow();
  [DllImport("user32.dll")] public static extern uint GetWindowThreadProcessId(IntPtr hWnd, IntPtr lpdwProcessId);
  [DllImport("user32.dll")] public static extern bool AttachThreadInput(uint idAttach, uint idAttachTo, bool fAttach);
  [DllImport("user32.dll")] public static extern IntPtr SetActiveWindow(IntPtr hWnd);
  [DllImport("user32.dll")] public static extern bool BringWindowToTop(IntPtr hWnd);
  public const uint LEFTDOWN = 0x0002;
  public const uint LEFTUP = 0x0004;
}
"@
New-Item -ItemType Directory -Force -Path $OutDir | Out-Null

function Initialize-HudForeground($hwnd) {
  $foreground = [Win32HudSize]::GetForegroundWindow()
  $foregroundThread = [Win32HudSize]::GetWindowThreadProcessId($foreground, [IntPtr]::Zero)
  $targetThread = [Win32HudSize]::GetWindowThreadProcessId($hwnd, [IntPtr]::Zero)
  $attached = $false
  if ($foregroundThread -ne 0 -and $foregroundThread -ne $targetThread) {
    $attached = [Win32HudSize]::AttachThreadInput($foregroundThread, $targetThread, $true)
  }
  [Win32HudSize]::BringWindowToTop($hwnd) | Out-Null
  [Win32HudSize]::SetForegroundWindow($hwnd) | Out-Null
  [Win32HudSize]::SetActiveWindow($hwnd) | Out-Null
  if ($attached) {
    [Win32HudSize]::AttachThreadInput($foregroundThread, $targetThread, $false) | Out-Null
  }
}

function Get-HudRect($hwnd) {
  $rect = New-Object Win32HudSize+RECT
  if (-not [Win32HudSize]::GetWindowRect($hwnd, [ref]$rect)) { throw "GetWindowRect failed" }
  return [pscustomobject]@{ Left=$rect.Left; Top=$rect.Top; Width=($rect.Right-$rect.Left); Height=($rect.Bottom-$rect.Top) }
}

function Save-HudShot($hwnd, [string]$Name) {
  $r = Get-HudRect $hwnd
  $bmp = New-Object System.Drawing.Bitmap $r.Width, $r.Height
  $g = [System.Drawing.Graphics]::FromImage($bmp)
  $g.CopyFromScreen($r.Left, $r.Top, 0, 0, $bmp.Size)
  $path = Join-Path $OutDir $Name
  $bmp.Save($path, [System.Drawing.Imaging.ImageFormat]::Png)
  $g.Dispose(); $bmp.Dispose()
  return $path
}

function Invoke-ClickAt($hwnd, [int]$X, [int]$Y, [int]$WaitMs = 1500) {
  Initialize-HudForeground $hwnd
  Start-Sleep -Milliseconds 200
  [Win32HudSize]::SetCursorPos($X, $Y) | Out-Null
  Start-Sleep -Milliseconds 150
  [Win32HudSize]::mouse_event([Win32HudSize]::LEFTDOWN, 0, 0, 0, [UIntPtr]::Zero)
  Start-Sleep -Milliseconds 80
  [Win32HudSize]::mouse_event([Win32HudSize]::LEFTUP, 0, 0, 0, [UIntPtr]::Zero)
  Start-Sleep -Milliseconds $WaitMs
}

$proc = Get-Process -Id $HudPid -ErrorAction Stop
$hwnd = [IntPtr]$proc.MainWindowHandle
if ($hwnd -eq [IntPtr]::Zero) { throw "HUD window not found for PID $HudPid" }
Initialize-HudForeground $hwnd
Start-Sleep -Milliseconds 800

$initial = Get-HudRect $hwnd
if ($initial.Height -lt 200) {
  $x = $initial.Left + $initial.Width - 100
  $y = $initial.Top + $initial.Height - 36
  Invoke-ClickAt $hwnd $x $y 1500
}
$expanded = Get-HudRect $hwnd
if ($expanded.Height -lt 200) { throw "HUD did not expand; got $($expanded.Width)x$($expanded.Height)" }
$before = Save-HudShot $hwnd "size-expanded-before.png"

function Click-Preset([int]$FractionNumerator) {
  $r = Get-HudRect $hwnd
  # A click on inert popover content activates the window without changing
  # state. Windows may otherwise swallow the first click after a resize.
  Invoke-ClickAt $hwnd ($r.Left + 20) ($r.Top + 100) 350
  $r = Get-HudRect $hwnd
  $x = $r.Left + [int]($r.Width * $FractionNumerator / 6)
  $y = $r.Top + $r.Height - 128
  Invoke-ClickAt $hwnd $x $y 2200
  return Get-HudRect $hwnd
}

$comfortable = Click-Preset 3
$comfortableShot = Save-HudShot $hwnd "size-comfortable.png"
$large = Click-Preset 5
$largeShot = Save-HudShot $hwnd "size-large.png"
$compact = Click-Preset 1
$compactShot = Save-HudShot $hwnd "size-compact-expanded.png"

# Collapse back to the compact capsule so the next launch starts clean.
$r = Get-HudRect $hwnd
Invoke-ClickAt $hwnd ($r.Left + $r.Width - 100) ($r.Top + $r.Height - 36) 1500
$collapsed = Get-HudRect $hwnd
$collapsedShot = Save-HudShot $hwnd "size-collapsed.png"

[pscustomobject]@{
  initial = $initial
  expanded = $expanded
  before = $before
  comfortable = $comfortable
  comfortableScreenshot = $comfortableShot
  large = $large
  largeScreenshot = $largeShot
  compact = $compact
  compactScreenshot = $compactShot
  collapsed = $collapsed
  collapsedScreenshot = $collapsedShot
} | ConvertTo-Json -Depth 6
