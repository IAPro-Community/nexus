param(
  [string]$OutDir = "C:\Dev\nexus-hud-validation\.tmp\native-hud",
  [int]$Cycles = 1,
  [int]$HudPid = 0
)
$ErrorActionPreference = "Stop"
Add-Type -AssemblyName System.Drawing
Add-Type @"
using System;
using System.Runtime.InteropServices;
public static class Win32Hud {
  [StructLayout(LayoutKind.Sequential)] public struct RECT { public int Left; public int Top; public int Right; public int Bottom; }
  [DllImport("user32.dll", SetLastError=true)] public static extern IntPtr FindWindow(string lpClassName, string lpWindowName);
  [DllImport("user32.dll")] public static extern bool GetWindowRect(IntPtr hWnd, out RECT lpRect);
  [DllImport("user32.dll")] public static extern bool SetForegroundWindow(IntPtr hWnd);
  [DllImport("user32.dll")] public static extern bool SetCursorPos(int X, int Y);
  [DllImport("user32.dll")] public static extern void mouse_event(uint dwFlags, uint dx, uint dy, uint dwData, UIntPtr dwExtraInfo);
  [DllImport("user32.dll")] public static extern bool ShowWindow(IntPtr hWnd, int nCmdShow);
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

function Get-HudWindow {
  if ($HudPid -gt 0) {
    $proc = Get-Process -Id $HudPid -ErrorAction SilentlyContinue
    if ($null -eq $proc -or $proc.MainWindowHandle -eq 0) { throw "Nexus HUD window not found for PID $HudPid" }
    return [IntPtr]$proc.MainWindowHandle
  }
  $hwnd = [Win32Hud]::FindWindow($null, "Nexus HUD")
  if ($hwnd -eq [IntPtr]::Zero) { throw "Nexus HUD window not found" }
  return $hwnd
}
function Initialize-HudForeground($hwnd) {
  $foreground = [Win32Hud]::GetForegroundWindow()
  $foregroundThread = [Win32Hud]::GetWindowThreadProcessId($foreground, [IntPtr]::Zero)
  $targetThread = [Win32Hud]::GetWindowThreadProcessId($hwnd, [IntPtr]::Zero)
  $attached = $false
  if ($foregroundThread -ne 0 -and $foregroundThread -ne $targetThread) {
    $attached = [Win32Hud]::AttachThreadInput($foregroundThread, $targetThread, $true)
  }
  [Win32Hud]::BringWindowToTop($hwnd) | Out-Null
  [Win32Hud]::SetForegroundWindow($hwnd) | Out-Null
  [Win32Hud]::SetActiveWindow($hwnd) | Out-Null
  if ($attached) {
    [Win32Hud]::AttachThreadInput($foregroundThread, $targetThread, $false) | Out-Null
  }
}
function Get-HudRect($hwnd) {
  $rect = New-Object Win32Hud+RECT
  if (-not [Win32Hud]::GetWindowRect($hwnd, [ref]$rect)) { throw "GetWindowRect failed" }
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
function Invoke-HudClick($hwnd, [int]$OffsetY = 0) {
  $r = Get-HudRect $hwnd
  # The capsule stays anchored to the window's bottom-right corner in both
  # compact and expanded states; never click the geometric center after growth.
  $x = $r.Left + $r.Width - 100
  $y = $r.Top + $r.Height - 36 + $OffsetY
  Initialize-HudForeground $hwnd
  Start-Sleep -Milliseconds 200
  [Win32Hud]::SetCursorPos($x, $y) | Out-Null
  Start-Sleep -Milliseconds 120
  [Win32Hud]::mouse_event([Win32Hud]::LEFTDOWN, 0, 0, 0, [UIntPtr]::Zero)
  Start-Sleep -Milliseconds 80
  [Win32Hud]::mouse_event([Win32Hud]::LEFTUP, 0, 0, 0, [UIntPtr]::Zero)
  Start-Sleep -Milliseconds 900
  return [pscustomobject]@{ X=$x; Y=$y }
}
function Invoke-HudDrag($hwnd, [int]$Dx, [int]$Dy) {
  $r = Get-HudRect $hwnd
  # Native drag grip sits at the left edge of the capsule row.
  $x = $r.Left + 20
  $y = $r.Top + $r.Height - 32
  Initialize-HudForeground $hwnd
  Start-Sleep -Milliseconds 200
  [Win32Hud]::SetCursorPos($x, $y) | Out-Null
  Start-Sleep -Milliseconds 120
  [Win32Hud]::mouse_event([Win32Hud]::LEFTDOWN, 0, 0, 0, [UIntPtr]::Zero)
  Start-Sleep -Milliseconds 150
  for ($i = 1; $i -le 10; $i++) {
    [Win32Hud]::SetCursorPos($x + [int]($Dx * $i / 10), $y + [int]($Dy * $i / 10)) | Out-Null
    Start-Sleep -Milliseconds 40
  }
  [Win32Hud]::mouse_event([Win32Hud]::LEFTUP, 0, 0, 0, [UIntPtr]::Zero)
  Start-Sleep -Milliseconds 700
}

$hwnd = Get-HudWindow
[Win32Hud]::ShowWindow($hwnd, 5) | Out-Null
Initialize-HudForeground $hwnd
Start-Sleep -Milliseconds 800
$before = Get-HudRect $hwnd
Save-HudShot $hwnd "hud-compact.png" | Out-Null

$results = @()
for ($i = 1; $i -le $Cycles; $i++) {
  $clickPoint = Invoke-HudClick $hwnd
  $expanded = Get-HudRect $hwnd
  $shot = Save-HudShot $hwnd "hud-expanded-$i.png"
  $results += [pscustomobject]@{ cycle=$i; phase="expanded"; screenX=$clickPoint.X; screenY=$clickPoint.Y; rect=$expanded; screenshot=$shot }

  $clickPoint2 = Invoke-HudClick $hwnd
  $collapsed = Get-HudRect $hwnd
  $shot2 = Save-HudShot $hwnd "hud-collapsed-$i.png"
  $results += [pscustomobject]@{ cycle=$i; phase="collapsed"; screenX=$clickPoint2.X; screenY=$clickPoint2.Y; rect=$collapsed; screenshot=$shot2 }
}

$dragBefore = Get-HudRect $hwnd
Invoke-HudDrag $hwnd -160 -120
$dragAfter = Get-HudRect $hwnd
Save-HudShot $hwnd "hud-after-drag.png" | Out-Null

[pscustomobject]@{
  before = $before
  cycles = $results
  drag = [pscustomobject]@{ before=$dragBefore; after=$dragAfter; movedX=($dragAfter.Left-$dragBefore.Left); movedY=($dragAfter.Top-$dragBefore.Top) }
} | ConvertTo-Json -Depth 8
