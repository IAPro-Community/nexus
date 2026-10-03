$ErrorActionPreference = 'Stop'
$ProgressPreference = 'SilentlyContinue'

function Write-NexusEnvelope {
    param([hashtable]$Envelope)
    [Console]::Out.Write(($Envelope | ConvertTo-Json -Depth 12 -Compress))
}

function Fail-Nexus {
    param([string]$Message)
    Write-NexusEnvelope @{ ok = $false; error = $Message }
    exit 0
}

$raw = [Console]::In.ReadToEnd()
if ([string]::IsNullOrWhiteSpace($raw)) {
    Fail-Nexus 'computer-use helper request is required'
}

try {
    $request = $raw | ConvertFrom-Json
} catch {
    Fail-Nexus 'invalid computer-use helper JSON'
}

if ([int]$request.protocol_version -ne 1) {
    Fail-Nexus 'unsupported computer-use helper protocol'
}

$initError = ''
try {
    Add-Type -AssemblyName UIAutomationClient
    Add-Type -AssemblyName UIAutomationTypes
    Add-Type -AssemblyName System.Drawing
    Add-Type -AssemblyName System.Windows.Forms
    Add-Type -TypeDefinition @'
using System;
using System.Runtime.InteropServices;

public struct RECT {
    public int Left;
    public int Top;
    public int Right;
    public int Bottom;
}

public static class NexusWin32 {
    [DllImport("user32.dll")]
    public static extern bool GetWindowRect(IntPtr hWnd, out RECT rect);

    [DllImport("user32.dll")]
    public static extern bool SetForegroundWindow(IntPtr hWnd);

    [DllImport("user32.dll")]
    public static extern bool ShowWindow(IntPtr hWnd, int nCmdShow);

    [DllImport("user32.dll")]
    public static extern bool MoveWindow(IntPtr hWnd, int X, int Y, int nWidth, int nHeight, bool repaint);

    [DllImport("user32.dll")]
    public static extern bool SetCursorPos(int X, int Y);

    [DllImport("user32.dll")]
    public static extern void mouse_event(uint flags, uint dx, uint dy, uint data, UIntPtr extraInfo);

    [StructLayout(LayoutKind.Sequential)]
    public struct INPUT {
        public uint type;
        public INPUTUNION U;
    }

    [StructLayout(LayoutKind.Explicit)]
    public struct INPUTUNION {
        [FieldOffset(0)]
        public MOUSEINPUT mi;
        [FieldOffset(0)]
        public KEYBDINPUT ki;
    }

    [StructLayout(LayoutKind.Sequential)]
    public struct MOUSEINPUT {
        public int dx;
        public int dy;
        public uint mouseData;
        public uint dwFlags;
        public uint time;
        public UIntPtr dwExtraInfo;
    }

    [StructLayout(LayoutKind.Sequential)]
    public struct KEYBDINPUT {
        public ushort wVk;
        public ushort wScan;
        public uint dwFlags;
        public uint time;
        public UIntPtr dwExtraInfo;
    }

    [DllImport("user32.dll", SetLastError = true)]
    private static extern uint SendInput(uint nInputs, INPUT[] pInputs, int cbSize);

    public static void SendUnicodeText(string text) {
        if (String.IsNullOrEmpty(text)) {
            return;
        }
        var inputs = new INPUT[text.Length * 2];
        int index = 0;
        foreach (char ch in text) {
            inputs[index++] = new INPUT {
                type = 1,
                U = new INPUTUNION {
                    ki = new KEYBDINPUT { wVk = 0, wScan = (ushort)ch, dwFlags = 0x0004 }
                }
            };
            inputs[index++] = new INPUT {
                type = 1,
                U = new INPUTUNION {
                    ki = new KEYBDINPUT { wVk = 0, wScan = (ushort)ch, dwFlags = 0x0004 | 0x0002 }
                }
            };
        }
        uint sent = SendInput((uint)inputs.Length, inputs, Marshal.SizeOf(typeof(INPUT)));
        if (sent != (uint)inputs.Length) {
            throw new System.ComponentModel.Win32Exception(Marshal.GetLastWin32Error(), "SendInput did not deliver all unicode keyboard events");
        }
    }

    public static void ScrollVertical(int delta) {
        mouse_event(0x0800, 0, 0, unchecked((uint)delta), UIntPtr.Zero);
    }

    public static void ScrollHorizontal(int delta) {
        mouse_event(0x1000, 0, 0, unchecked((uint)delta), UIntPtr.Zero);
    }
}
'@
} catch {
    $initError = $_.Exception.Message
}

function Require-Automation {
    if (-not [string]::IsNullOrWhiteSpace($initError)) {
        throw "Windows UI Automation initialization failed: $initError"
    }
}

function Get-NexusAppProcesses {
    param([string]$App)

    $needle = ([string]$App).Trim()
    if ([string]::IsNullOrWhiteSpace($needle)) {
        throw 'app is required'
    }

    $processes = @()
    $pidValue = 0
    if ([int]::TryParse($needle, [ref]$pidValue)) {
        $candidate = Get-Process -Id $pidValue -ErrorAction SilentlyContinue
        if ($null -ne $candidate -and $candidate.MainWindowHandle -ne 0) {
            $processes += $candidate
        }
    } else {
        $processes = @(
            Get-Process -ErrorAction SilentlyContinue |
                Where-Object {
                    $_.MainWindowHandle -ne 0 -and (
                        $_.ProcessName -ieq $needle -or
                        $_.MainWindowTitle -ieq $needle
                    )
                } |
                Sort-Object Id
        )
    }

    if ($processes.Count -eq 0) {
        throw "application '$needle' has no accessible top-level window"
    }
    return $processes
}

function Get-NexusWindowRect {
    param([IntPtr]$Handle)

    $rect = New-Object RECT
    if (-not [NexusWin32]::GetWindowRect($Handle, [ref]$rect)) {
        throw 'could not read target window bounds'
    }
    return @{
        x = [int]$rect.Left
        y = [int]$rect.Top
        width = [Math]::Max(0, [int]($rect.Right - $rect.Left))
        height = [Math]::Max(0, [int]($rect.Bottom - $rect.Top))
    }
}

function Get-NexusWindow {
    param($Payload)

    $windowId = ([string]$Payload.window_id).Trim()
    $processes = @(Get-NexusAppProcesses ([string]$Payload.app))

    if (-not [string]::IsNullOrWhiteSpace($windowId)) {
        foreach ($process in $processes) {
            if ($process.MainWindowHandle.ToInt64().ToString() -eq $windowId) {
                return $process
            }
        }
        throw "window '$windowId' was not found for the requested application"
    }

    $index = 0
    if ($null -ne $Payload.window_index) {
        $index = [int]$Payload.window_index
    }
    if ($index -lt 0 -or $index -ge $processes.Count) {
        throw "window index $index is outside the available range"
    }
    return $processes[$index]
}

function Get-NexusRuntimeId {
    param(
        [System.Windows.Automation.AutomationElement]$Element,
        [int]$Index
    )
    try {
        $runtimeId = $Element.GetRuntimeId()
        if ($null -ne $runtimeId -and $runtimeId.Count -gt 0) {
            return ($runtimeId -join '.')
        }
    } catch {}
    return "idx:$Index"
}

function Get-NexusElementActions {
    param([System.Windows.Automation.AutomationElement]$Element)

    $actions = New-Object System.Collections.Generic.List[string]
    $pattern = $null
    try {
        if ($Element.TryGetCurrentPattern([System.Windows.Automation.InvokePattern]::Pattern, [ref]$pattern)) {
            $actions.Add('INVOKE')
        }
    } catch {}
    $pattern = $null
    try {
        if ($Element.TryGetCurrentPattern([System.Windows.Automation.ValuePattern]::Pattern, [ref]$pattern)) {
            $actions.Add('SET_VALUE')
        }
    } catch {}
    $pattern = $null
    try {
        if ($Element.TryGetCurrentPattern([System.Windows.Automation.SelectionItemPattern]::Pattern, [ref]$pattern)) {
            $actions.Add('SELECT')
        }
    } catch {}
    $pattern = $null
    try {
        if ($Element.TryGetCurrentPattern([System.Windows.Automation.TogglePattern]::Pattern, [ref]$pattern)) {
            $actions.Add('TOGGLE')
        }
    } catch {}
    return @($actions)
}

function Get-NexusElements {
    param([IntPtr]$Handle)

    $root = [System.Windows.Automation.AutomationElement]::FromHandle($Handle)
    if ($null -eq $root) {
        throw 'UI Automation could not attach to target window'
    }

    $items = New-Object System.Collections.Generic.List[object]
    $items.Add($root)
    $descendants = $root.FindAll(
        [System.Windows.Automation.TreeScope]::Descendants,
        [System.Windows.Automation.Condition]::TrueCondition
    )
    $limit = [Math]::Min($descendants.Count, 399)
    for ($i = 0; $i -lt $limit; $i++) {
        $items.Add($descendants.Item($i))
    }
    return @($items)
}

function Convert-NexusElement {
    param(
        [System.Windows.Automation.AutomationElement]$Element,
        [int]$Index
    )

    $name = ''
    $role = ''
    $sensitive = $false
    $value = ''
    $x = 0
    $y = 0
    $width = 0
    $height = 0

    try { $name = [string]$Element.Current.Name } catch {}
    try {
        $role = [string]$Element.Current.ControlType.ProgrammaticName
        $role = $role -replace '^ControlType\.', ''
    } catch {}
    try { $sensitive = [bool]$Element.Current.IsPassword } catch {}
    try {
        $bounds = $Element.Current.BoundingRectangle
        if (-not [Double]::IsNaN($bounds.X) -and -not [Double]::IsInfinity($bounds.X)) {
            $x = [int][Math]::Round($bounds.X)
            $y = [int][Math]::Round($bounds.Y)
            $width = [Math]::Max(0, [int][Math]::Round($bounds.Width))
            $height = [Math]::Max(0, [int][Math]::Round($bounds.Height))
        }
    } catch {}

    if (-not $sensitive) {
        try {
            $pattern = $null
            if ($Element.TryGetCurrentPattern([System.Windows.Automation.ValuePattern]::Pattern, [ref]$pattern)) {
                $value = [string]$pattern.Current.Value
                if ($value.Length -gt 2048) {
                    $value = $value.Substring(0, 2048)
                }
            }
        } catch {}
    }

    return [ordered]@{
        id = Get-NexusRuntimeId $Element $Index
        index = $Index
        role = $role
        name = $name
        value = $value
        actions = @(Get-NexusElementActions $Element)
        sensitive = $sensitive
        x = $x
        y = $y
        width = $width
        height = $height
    }
}

function Get-NexusElement {
    param(
        [IntPtr]$Handle,
        $Payload
    )

    $elements = @(Get-NexusElements $Handle)
    if ($null -ne $Payload.element_index) {
        $index = [int]$Payload.element_index
        if ($index -lt 0 -or $index -ge $elements.Count) {
            throw "element index $index is outside the available range"
        }
        return $elements[$index]
    }

    $wantedId = ([string]$Payload.element_id).Trim()
    if (-not [string]::IsNullOrWhiteSpace($wantedId)) {
        for ($i = 0; $i -lt $elements.Count; $i++) {
            if ((Get-NexusRuntimeId $elements[$i] $i) -eq $wantedId) {
                return $elements[$i]
            }
        }
        throw "element '$wantedId' is no longer present"
    }
    throw 'element_id or element_index is required'
}

function New-NexusScreenshot {
    param(
        [hashtable]$Rect,
        [string]$SnapshotId
    )

    if ($Rect.width -le 0 -or $Rect.height -le 0) {
        throw 'target window has no visible screenshot bounds'
    }
    if ($Rect.width -gt 4096 -or $Rect.height -gt 4096) {
        throw 'target window exceeds the safe screenshot size'
    }

    $root = $env:AI_CLI_DATA_DIR
    if ([string]::IsNullOrWhiteSpace($root)) {
        $root = Join-Path ([IO.Path]::GetTempPath()) 'iapro-nexus'
    }
    $dir = Join-Path $root 'computer-use'
    [IO.Directory]::CreateDirectory($dir) | Out-Null
    $path = Join-Path $dir ("snapshot-" + $SnapshotId + ".png")

    $bitmap = New-Object System.Drawing.Bitmap($Rect.width, $Rect.height)
    try {
        $graphics = [System.Drawing.Graphics]::FromImage($bitmap)
        try {
            $graphics.CopyFromScreen(
                $Rect.x,
                $Rect.y,
                0,
                0,
                (New-Object System.Drawing.Size($Rect.width, $Rect.height)),
                [System.Drawing.CopyPixelOperation]::SourceCopy
            )
        } finally {
            $graphics.Dispose()
        }
        $bitmap.Save($path, [System.Drawing.Imaging.ImageFormat]::Png)
    } finally {
        $bitmap.Dispose()
    }
    return $path
}

function New-NexusSnapshot {
    param(
        $Payload,
        [bool]$IncludeScreenshot
    )

    Require-Automation
    $process = Get-NexusWindow $Payload
    $handle = $process.MainWindowHandle
    if ($Payload.restore_window) {
        [NexusWin32]::ShowWindow($handle, 9) | Out-Null
    }

    $rect = Get-NexusWindowRect $handle
    $rawElements = @(Get-NexusElements $handle)
    $elements = New-Object System.Collections.Generic.List[object]
    $treeLines = New-Object System.Collections.Generic.List[string]
    for ($i = 0; $i -lt $rawElements.Count; $i++) {
        $descriptor = Convert-NexusElement $rawElements[$i] $i
        $elements.Add($descriptor)
        if ($treeLines.Count -lt 300) {
            $line = ("{0} {1} {2}" -f $i, $descriptor.role, $descriptor.name).Trim()
            if ($line.Length -gt 300) {
                $line = $line.Substring(0, 300)
            }
            $treeLines.Add($line)
        }
    }

    $snapshotId = [Guid]::NewGuid().ToString('N')
    $screenshotPath = ''
    if ($IncludeScreenshot) {
        $screenshotPath = New-NexusScreenshot $rect $snapshotId
    }

    return [ordered]@{
        id = $snapshotId
        app = [ordered]@{
            name = [string]$process.ProcessName
            pid = [int]$process.Id
        }
        window = [ordered]@{
            id = $handle.ToInt64().ToString()
            index = 0
            title = [string]$process.MainWindowTitle
            x = $rect.x
            y = $rect.y
            width = $rect.width
            height = $rect.height
            minimized = $false
        }
        elements = @($elements)
        tree_text = ($treeLines -join [Environment]::NewLine)
        screenshot_path = $screenshotPath
        captured_at = [DateTime]::UtcNow.ToString('o')
    }
}

function Invoke-NexusClick {
    param([System.Windows.Automation.AutomationElement]$Element)

    $pattern = $null
    if ($Element.TryGetCurrentPattern([System.Windows.Automation.InvokePattern]::Pattern, [ref]$pattern)) {
        $pattern.Invoke()
        return
    }
    $pattern = $null
    if ($Element.TryGetCurrentPattern([System.Windows.Automation.SelectionItemPattern]::Pattern, [ref]$pattern)) {
        $pattern.Select()
        return
    }

    $bounds = $Element.Current.BoundingRectangle
    if ($bounds.Width -le 0 -or $bounds.Height -le 0) {
        throw 'target element has no clickable bounds'
    }
    $cx = [int][Math]::Round($bounds.X + ($bounds.Width / 2))
    $cy = [int][Math]::Round($bounds.Y + ($bounds.Height / 2))
    if (-not [NexusWin32]::SetCursorPos($cx, $cy)) {
        throw 'could not position pointer on target element'
    }
    [NexusWin32]::mouse_event(0x0002, 0, 0, 0, [UIntPtr]::Zero)
    [NexusWin32]::mouse_event(0x0004, 0, 0, 0, [UIntPtr]::Zero)
}

function Set-NexusValue {
    param(
        [System.Windows.Automation.AutomationElement]$Element,
        [string]$Text
    )

    if ([bool]$Element.Current.IsPassword) {
        throw 'sensitive password fields are not writable by computer-use'
    }
    $pattern = $null
    if (-not $Element.TryGetCurrentPattern([System.Windows.Automation.ValuePattern]::Pattern, [ref]$pattern)) {
        throw 'target element does not expose ValuePattern'
    }
    if ($pattern.Current.IsReadOnly) {
        throw 'target element is read-only'
    }
    $pattern.SetValue($Text)
}

function Test-NexusPointInRect {
    param(
        [int]$X,
        [int]$Y,
        [hashtable]$Rect
    )
    return (
        $X -ge $Rect.x -and
        $Y -ge $Rect.y -and
        $X -lt ($Rect.x + $Rect.width) -and
        $Y -lt ($Rect.y + $Rect.height)
    )
}

function Invoke-NexusCoordinateClick {
    param(
        [int]$X,
        [int]$Y,
        [hashtable]$Rect
    )
    if (-not (Test-NexusPointInRect $X $Y $Rect)) {
        throw 'click coordinates are outside the target window'
    }
    if (-not [NexusWin32]::SetCursorPos($X, $Y)) {
        throw 'could not position pointer in target window'
    }
    [NexusWin32]::mouse_event(0x0002, 0, 0, 0, [UIntPtr]::Zero)
    [NexusWin32]::mouse_event(0x0004, 0, 0, 0, [UIntPtr]::Zero)
}

function Send-NexusText {
    param(
        [System.Windows.Automation.AutomationElement]$Element,
        [string]$Text
    )
    if ([bool]$Element.Current.IsPassword) {
        throw 'sensitive password fields are not writable by computer-use'
    }
    if ($Text.Length -gt 8192) {
        throw 'TYPE_TEXT exceeds the 8192 character limit'
    }
    $Element.SetFocus()
    Start-Sleep -Milliseconds 40
    [NexusWin32]::SendUnicodeText($Text)
}

function Invoke-NexusScroll {
    param(
        [string]$Direction,
        $Payload,
        [hashtable]$Rect
    )
    $directionValue = ([string]$Direction).Trim().ToUpperInvariant()
    $x = [int]($Rect.x + [Math]::Floor($Rect.width / 2))
    $y = [int]($Rect.y + [Math]::Floor($Rect.height / 2))
    if ($null -ne $Payload.x) { $x = [int]$Payload.x }
    if ($null -ne $Payload.y) { $y = [int]$Payload.y }
    if (-not (Test-NexusPointInRect $x $y $Rect)) {
        throw 'scroll coordinates are outside the target window'
    }
    if (-not [NexusWin32]::SetCursorPos($x, $y)) {
        throw 'could not position pointer in target window'
    }
    switch ($directionValue) {
        'UP' { [NexusWin32]::ScrollVertical(120) }
        'DOWN' { [NexusWin32]::ScrollVertical(-120) }
        'LEFT' { [NexusWin32]::ScrollHorizontal(-120) }
        'RIGHT' { [NexusWin32]::ScrollHorizontal(120) }
        default { throw "unsupported scroll direction '$directionValue'" }
    }
}

function Invoke-NexusDrag {
    param(
        $Payload,
        [hashtable]$Rect
    )
    if ($null -eq $Payload.x -or $null -eq $Payload.y -or $null -eq $Payload.to_x -or $null -eq $Payload.to_y) {
        throw 'DRAG requires x/y/to_x/to_y'
    }
    $x = [int]$Payload.x
    $y = [int]$Payload.y
    $toX = [int]$Payload.to_x
    $toY = [int]$Payload.to_y
    if (-not (Test-NexusPointInRect $x $y $Rect) -or -not (Test-NexusPointInRect $toX $toY $Rect)) {
        throw 'drag coordinates must stay inside the target window'
    }
    if (-not [NexusWin32]::SetCursorPos($x, $y)) {
        throw 'could not position pointer at drag origin'
    }
    [NexusWin32]::mouse_event(0x0002, 0, 0, 0, [UIntPtr]::Zero)
    try {
        Start-Sleep -Milliseconds 40
        if (-not [NexusWin32]::SetCursorPos($toX, $toY)) {
            throw 'could not position pointer at drag destination'
        }
        Start-Sleep -Milliseconds 40
    } finally {
        [NexusWin32]::mouse_event(0x0004, 0, 0, 0, [UIntPtr]::Zero)
    }
}

try {
    $operation = ([string]$request.operation).Trim().ToLowerInvariant()

    switch ($operation) {
        'capabilities' {
            $available = [string]::IsNullOrWhiteSpace($initError)
            Write-NexusEnvelope @{
                ok = $true
                capabilities = [ordered]@{
                    provider = 'windows-uia-powershell'
                    provider_version = '1'
                    protocol_version = 1
                    platform = 'windows'
                    list_apps = $available
                    list_windows = $available
                    accessibility_tree = $available
                    screenshot = $available
                    click = $available
                    set_value = $available
                    type_text = $available
                    press_key = $false
                    hotkey = $false
                    paste_text = $false
                    scroll = $available
                    drag = $available
                    window_management = $available
                }
            }
        }
        'permissions' {
            $available = [string]::IsNullOrWhiteSpace($initError)
            Write-NexusEnvelope @{
                ok = $true
                permissions = [ordered]@{
                    available = $available
                    accessibility = $(if ($available) { 'AVAILABLE' } else { 'UNAVAILABLE' })
                    screen_capture = $(if ($available) { 'AVAILABLE' } else { 'UNAVAILABLE' })
                    input_monitoring = 'NOT_REQUIRED'
                    reason = $(if ($available) {
                        'Windows UI Automation is available in the current user session; elevated applications may remain inaccessible across integrity levels.'
                    } else {
                        "Windows UI Automation initialization failed: $initError"
                    })
                }
            }
        }
        'list_apps' {
            Require-Automation
            $apps = @(
                Get-Process -ErrorAction SilentlyContinue |
                    Where-Object { $_.MainWindowHandle -ne 0 } |
                    Sort-Object ProcessName, Id |
                    ForEach-Object {
                        [ordered]@{
                            name = [string]$_.ProcessName
                            pid = [int]$_.Id
                        }
                    }
            )
            Write-NexusEnvelope @{ ok = $true; apps = $apps }
        }
        'list_windows' {
            Require-Automation
            $app = [string]$request.payload.app
            $processes = @(Get-NexusAppProcesses $app)
            $windows = New-Object System.Collections.Generic.List[object]
            for ($i = 0; $i -lt $processes.Count; $i++) {
                $process = $processes[$i]
                $rect = Get-NexusWindowRect $process.MainWindowHandle
                $windows.Add([ordered]@{
                    id = $process.MainWindowHandle.ToInt64().ToString()
                    index = $i
                    title = [string]$process.MainWindowTitle
                    x = $rect.x
                    y = $rect.y
                    width = $rect.width
                    height = $rect.height
                    minimized = $false
                })
            }
            Write-NexusEnvelope @{ ok = $true; windows = @($windows) }
        }
        'snapshot' {
            $snapshot = New-NexusSnapshot $request.payload ([bool]$request.payload.screenshot)
            Write-NexusEnvelope @{ ok = $true; snapshot = $snapshot }
        }
        'act' {
            Require-Automation
            $before = New-NexusSnapshot $request.payload $false
            $process = Get-NexusWindow $request.payload
            $handle = $process.MainWindowHandle
            $kind = ([string]$request.payload.kind).Trim().ToUpperInvariant()

            switch ($kind) {
                'CLICK' {
                    if (-not [string]::IsNullOrWhiteSpace(([string]$request.payload.element_id)) -or $null -ne $request.payload.element_index) {
                        $element = Get-NexusElement $handle $request.payload
                        Invoke-NexusClick $element
                    } elseif ($null -ne $request.payload.x -and $null -ne $request.payload.y) {
                        $rect = Get-NexusWindowRect $handle
                        Invoke-NexusCoordinateClick ([int]$request.payload.x) ([int]$request.payload.y) $rect
                    } else {
                        throw 'CLICK requires an element or x/y coordinates'
                    }
                }
                'SET_VALUE' {
                    $element = Get-NexusElement $handle $request.payload
                    Set-NexusValue $element ([string]$request.payload.text)
                }
                'TYPE_TEXT' {
                    $element = Get-NexusElement $handle $request.payload
                    Send-NexusText $element ([string]$request.payload.text)
                }
                'SCROLL' {
                    $rect = Get-NexusWindowRect $handle
                    Invoke-NexusScroll ([string]$request.payload.direction) $request.payload $rect
                }
                'DRAG' {
                    $rect = Get-NexusWindowRect $handle
                    Invoke-NexusDrag $request.payload $rect
                }
                'FOCUS_WINDOW' {
                    if ($request.payload.restore_window) {
                        [NexusWin32]::ShowWindow($handle, 9) | Out-Null
                    }
                    if (-not [NexusWin32]::SetForegroundWindow($handle)) {
                        throw 'Windows rejected foreground activation for this window'
                    }
                }
                'MOVE_WINDOW' {
                    if ($null -eq $request.payload.x -or $null -eq $request.payload.y) {
                        throw 'MOVE_WINDOW requires x and y'
                    }
                    $rect = Get-NexusWindowRect $handle
                    if (-not [NexusWin32]::MoveWindow(
                        $handle,
                        [int]$request.payload.x,
                        [int]$request.payload.y,
                        $rect.width,
                        $rect.height,
                        $true
                    )) {
                        throw 'could not move target window'
                    }
                }
                'RESIZE_WINDOW' {
                    if ($null -eq $request.payload.width -or $null -eq $request.payload.height) {
                        throw 'RESIZE_WINDOW requires width and height'
                    }
                    $width = [int]$request.payload.width
                    $height = [int]$request.payload.height
                    if ($width -le 0 -or $height -le 0 -or $width -gt 8192 -or $height -gt 8192) {
                        throw 'requested window size is outside the safe range'
                    }
                    $rect = Get-NexusWindowRect $handle
                    if (-not [NexusWin32]::MoveWindow(
                        $handle,
                        $rect.x,
                        $rect.y,
                        $width,
                        $height,
                        $true
                    )) {
                        throw 'could not resize target window'
                    }
                }
                default {
                    throw "computer action '$kind' is not supported by the Windows helper"
                }
            }

            Start-Sleep -Milliseconds 120
            $after = New-NexusSnapshot $request.payload $false
            Write-NexusEnvelope @{
                ok = $true
                action = [ordered]@{
                    snapshot = $after
                    verification = [ordered]@{
                        state = 'OBSERVED'
                        reason = 'post-action accessibility snapshot captured'
                        before_snapshot_id = $before.id
                        after_snapshot_id = $after.id
                    }
                }
            }
        }
        default {
            Fail-Nexus "unsupported computer-use operation '$operation'"
        }
    }
} catch {
    Fail-Nexus $_.Exception.Message
}
