param(
  [string]$OutDir = "C:\Dev\nexus-hud-validation\.tmp\native-hud-uia",
  [int]$HudPid = 0
)
$ErrorActionPreference = "Stop"
Add-Type -AssemblyName System.Drawing
Add-Type -AssemblyName UIAutomationClient
Add-Type -AssemblyName UIAutomationTypes
Add-Type @"
using System;
using System.Runtime.InteropServices;
public static class Win32HudUia {
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

function Get-HudRect($hwnd) {
  $rect = New-Object Win32HudUia+RECT
  if (-not [Win32HudUia]::GetWindowRect($hwnd, [ref]$rect)) { throw "GetWindowRect failed" }
  return [pscustomobject]@{ Left=$rect.Left; Top=$rect.Top; Width=($rect.Right-$rect.Left); Height=($rect.Bottom-$rect.Top) }
}

function Initialize-HudForeground($hwnd) {
  $foreground = [Win32HudUia]::GetForegroundWindow()
  $foregroundThread = [Win32HudUia]::GetWindowThreadProcessId($foreground, [IntPtr]::Zero)
  $targetThread = [Win32HudUia]::GetWindowThreadProcessId($hwnd, [IntPtr]::Zero)
  $attached = $false
  if ($foregroundThread -ne 0 -and $foregroundThread -ne $targetThread) {
    $attached = [Win32HudUia]::AttachThreadInput($foregroundThread, $targetThread, $true)
  }
  [Win32HudUia]::BringWindowToTop($hwnd) | Out-Null
  [Win32HudUia]::SetForegroundWindow($hwnd) | Out-Null
  [Win32HudUia]::SetActiveWindow($hwnd) | Out-Null
  if ($attached) {
    [Win32HudUia]::AttachThreadInput($foregroundThread, $targetThread, $false) | Out-Null
  }
}

function Get-HudRoot($hwnd) {
  return [System.Windows.Automation.AutomationElement]::FromHandle($hwnd)
}

function Get-HudButtons($root) {
  $cond = New-Object System.Windows.Automation.PropertyCondition(
    [System.Windows.Automation.AutomationElement]::ControlTypeProperty,
    [System.Windows.Automation.ControlType]::Button
  )
  return $root.FindAll([System.Windows.Automation.TreeScope]::Descendants, $cond)
}

function Find-HudButton($root, [string[]]$NamePatterns) {
  $buttons = Get-HudButtons $root
  foreach ($pattern in $NamePatterns) {
    foreach ($button in $buttons) {
      if ($button.Current.Name -like $pattern) { return $button }
    }
  }
  return $null
}

function Invoke-HudButton($root, [string[]]$NamePatterns, [int]$WaitMs = 1800, [string]$ExpandAction = "") {
  $button = Find-HudButton $root $NamePatterns
  $NamePattern = $NamePatterns -join " | "
  if ($null -eq $button) { throw "HUD button not found: $NamePattern" }
  $patterns = @($button.GetSupportedPatterns() | ForEach-Object { $_.ProgrammaticName })
  if ($patterns -contains "TogglePatternIdentifiers.Pattern") {
    $pattern = $button.GetCurrentPattern([System.Windows.Automation.TogglePattern]::Pattern)
    $pattern.Toggle()
  } elseif ($patterns -contains "ExpandCollapsePatternIdentifiers.Pattern") {
    $pattern = $button.GetCurrentPattern([System.Windows.Automation.ExpandCollapsePattern]::Pattern)
    if ($ExpandAction -eq "collapse") { $pattern.Collapse() }
    elseif ($ExpandAction -eq "expand") { $pattern.Expand() }
    else { $pattern.Expand() }
  } elseif ($patterns -contains "InvokePatternIdentifiers.Pattern") {
    $pattern = $button.GetCurrentPattern([System.Windows.Automation.InvokePattern]::Pattern)
    $pattern.Invoke()
  } else {
    throw "HUD button has no supported action: $NamePattern ($($patterns -join ', '))"
  }
  Start-Sleep -Milliseconds $WaitMs
  return Get-HudRect $hwnd
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

function Invoke-HudDrag([int]$Dx, [int]$Dy) {
  $root = Get-HudRoot $hwnd
  $handle = Find-HudButton $root "Arrastar HUD*"
  if ($null -eq $handle) { throw "Native HUD drag handle not found" }
  $r = $handle.Current.BoundingRectangle
  $x = [int]($r.Left + $r.Width / 2)
  $y = [int]($r.Top + $r.Height / 2)
  $before = Get-HudRect $hwnd
  Initialize-HudForeground $hwnd
  Start-Sleep -Milliseconds 250
  [Win32HudUia]::SetCursorPos($x, $y) | Out-Null
  Start-Sleep -Milliseconds 150
  [Win32HudUia]::mouse_event([Win32HudUia]::LEFTDOWN, 0, 0, 0, [UIntPtr]::Zero)
  Start-Sleep -Milliseconds 180
  for ($i = 1; $i -le 12; $i++) {
    [Win32HudUia]::SetCursorPos($x + [int]($Dx * $i / 12), $y + [int]($Dy * $i / 12)) | Out-Null
    Start-Sleep -Milliseconds 45
  }
  [Win32HudUia]::mouse_event([Win32HudUia]::LEFTUP, 0, 0, 0, [UIntPtr]::Zero)
  Start-Sleep -Milliseconds 900
  return [pscustomobject]@{ before=$before; after=(Get-HudRect $hwnd); handle=[pscustomobject]@{ Left=$r.Left; Top=$r.Top; Width=$r.Width; Height=$r.Height } }
}

$proc = Get-Process -Id $HudPid -ErrorAction Stop
$hwnd = [IntPtr]$proc.MainWindowHandle
if ($hwnd -eq [IntPtr]::Zero) { throw "HUD window not found for PID $HudPid" }
Initialize-HudForeground $hwnd
Start-Sleep -Milliseconds 600

$root = Get-HudRoot $hwnd
$initial = Get-HudRect $hwnd
$initialNames = @(Get-HudButtons $root | ForEach-Object { $_.Current.Name })

# Ensure the popover is open. The capsule is the only button whose name starts
# with "Recolher Nexus HUD." when collapsed, and the header close button starts
# with "Recolher Nexus HUD" when expanded.
if ($null -eq (Find-HudButton $root "Compact*")) {
  $null = Invoke-HudButton $root @("Expandir Nexus HUD*", "Recolher Nexus HUD. Nexus*") 1600 "expand"
  $root = Get-HudRoot $hwnd
}
if ($null -eq (Find-HudButton $root "Compact*")) { throw "HUD size controls did not appear" }

$expanded = Get-HudRect $hwnd
$expandedShot = Save-HudShot $hwnd "expanded.png"
$expandedNames = @(Get-HudButtons $root | ForEach-Object { $_.Current.Name })

$compactPreset = Invoke-HudButton $root "Compact*" 2200
$compactShot = Save-HudShot $hwnd "preset-compact.png"
$root = Get-HudRoot $hwnd

$comfortablePreset = Invoke-HudButton $root "*fort*" 2200
$comfortableShot = Save-HudShot $hwnd "preset-comfortable.png"
$root = Get-HudRoot $hwnd

$largePreset = Invoke-HudButton $root @("Grande", "Large") 2200
$largeShot = Save-HudShot $hwnd "preset-large.png"
$root = Get-HudRoot $hwnd

$backToCompact = Invoke-HudButton $root "Compact*" 2200
$backToCompactShot = Save-HudShot $hwnd "preset-back-to-compact.png"
$root = Get-HudRoot $hwnd

# Collapse through the capsule, then drag the dedicated grip.
$collapsed = Invoke-HudButton $root "Recolher Nexus HUD. Nexus*" 1600 "collapse"
$collapsedShot = Save-HudShot $hwnd "collapsed.png"
$drag = Invoke-HudDrag -144 -96
$dragShot = Save-HudShot $hwnd "after-drag.png"

[pscustomobject]@{
  initial = $initial
  initialButtons = $initialNames
  expanded = $expanded
  expandedScreenshot = $expandedShot
  expandedButtons = $expandedNames
  compactPreset = $compactPreset
  compactScreenshot = $compactShot
  comfortablePreset = $comfortablePreset
  comfortableScreenshot = $comfortableShot
  largePreset = $largePreset
  largeScreenshot = $largeShot
  backToCompact = $backToCompact
  backToCompactScreenshot = $backToCompactShot
  collapsed = $collapsed
  collapsedScreenshot = $collapsedShot
  drag = $drag
  dragScreenshot = $dragShot
} | ConvertTo-Json -Depth 8
