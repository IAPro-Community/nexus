; IAPro Nexus Windows installer (NSIS)
; Built by CI job "Windows NSIS installer". Output: nexus-setup-Windows_x86_64.exe

!include "MUI2.nsh"
!include "WinMessages.nsh"

Name "IAPro Nexus"
OutFile "nexus-setup-Windows_x86_64.exe"
Unicode True
InstallDir "$LOCALAPPDATA\Programs\IAPro Nexus"
RequestExecutionLevel user
SetCompressor /SOLID lzma

!define MUI_ABORTWARNING
!insertmacro MUI_PAGE_WELCOME
!insertmacro MUI_PAGE_DIRECTORY
!insertmacro MUI_PAGE_INSTFILES
!insertmacro MUI_PAGE_FINISH
!insertmacro MUI_LANGUAGE "English"

Function AddToUserPath
  ReadRegStr $0 HKCU "Environment" "Path"
  StrCmp $0 "" emptyPath
  ; Skip if already present
  Push $0
  Push "$INSTDIR"
  Call StrStr
  Pop $1
  StrCmp $1 "" appendPath donePath
emptyPath:
  WriteRegExpandStr HKCU "Environment" "Path" "$INSTDIR"
  Goto broadcast
appendPath:
  WriteRegExpandStr HKCU "Environment" "Path" "$0;$INSTDIR"
broadcast:
  SendMessage ${HWND_BROADCAST} ${WM_SETTINGCHANGE} 0 "STR:Environment" /TIMEOUT=5000
donePath:
FunctionEnd

Function CaptureOriginalUserPath
  ClearErrors
  ReadRegDWORD $0 HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\IAPro Nexus" "UserPathMetadataVersion"
  IfErrors captureOriginalPath
  StrCmp $0 1 doneCaptureUserPath

captureOriginalPath:
  ClearErrors
  ReadRegStr $0 HKCU "Environment" "Path"
  IfErrors originalPathMissing

  ; REG_EXPAND_SZ is the safe fallback if the native type query fails.
  StrCpy $3 2
  System::Call 'advapi32::RegOpenKeyExW(p 0x80000001, w "Environment", i 0, i 0x20019, *p .r1) i .r2'
  StrCmp $2 0 0 storeOriginalPath
  StrCpy $4 0
  System::Call 'advapi32::RegQueryValueExW(p r1, w "Path", p 0, *i .r3, p 0, *i .r4) i .r5'
  System::Call 'advapi32::RegCloseKey(p r1)'
  StrCmp $5 0 0 storeOriginalPath

storeOriginalPath:
  WriteRegDWORD HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\IAPro Nexus" "UserPathOriginalExists" 1
  WriteRegDWORD HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\IAPro Nexus" "UserPathOriginalType" $3
  WriteRegDWORD HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\IAPro Nexus" "UserPathMetadataVersion" 1
  Goto doneCaptureUserPath

originalPathMissing:
  WriteRegDWORD HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\IAPro Nexus" "UserPathOriginalExists" 0
  WriteRegDWORD HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\IAPro Nexus" "UserPathOriginalType" 0
  WriteRegDWORD HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\IAPro Nexus" "UserPathMetadataVersion" 1

doneCaptureUserPath:
FunctionEnd

Function un.RemoveFromUserPath
  ClearErrors
  ReadRegStr $0 HKCU "Environment" "Path"
  IfErrors doneRemove
  StrCmp $0 "" emptyCurrentPath
  Push $0
  Push "$INSTDIR"
  Call un.StrStr
  Pop $1
  StrCmp $1 "" doneRemove
  StrCpy $2 "$INSTDIR;"
  Push $0
  Push $2
  Call un.StrStr
  Pop $1
  StrCmp $1 "" tryTrailing
  StrLen $3 $2
  StrLen $4 $0
  StrLen $5 $1
  IntOp $6 $4 - $5
  StrCpy $7 $0 $6
  StrCpy $8 $1 "" $3
  StrCpy $0 "$7$8"
  Goto writePath
tryTrailing:
  StrCpy $2 ";$INSTDIR"
  Push $0
  Push $2
  Call un.StrStr
  Pop $1
  StrCmp $1 "" exactPath
  StrLen $3 $2
  StrLen $4 $0
  StrLen $5 $1
  IntOp $6 $4 - $5
  StrCpy $7 $0 $6
  StrCpy $8 $1 "" $3
  StrCpy $0 "$7$8"
  Goto writePath
exactPath:
  StrCmp $0 "$INSTDIR" 0 doneRemove
  StrCpy $0 ""
writePath:
  ClearErrors
  ReadRegDWORD $9 HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\IAPro Nexus" "UserPathOriginalExists"
  IfErrors writePathExpand
  StrCmp $9 0 deleteEmptyPath

  ClearErrors
  ReadRegDWORD $9 HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\IAPro Nexus" "UserPathOriginalType"
  IfErrors writePathExpand
  StrCmp $9 1 writePathString
  Goto writePathExpand

emptyCurrentPath:
  ClearErrors
  ReadRegDWORD $9 HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\IAPro Nexus" "UserPathOriginalExists"
  IfErrors doneRemove
  StrCmp $9 0 deleteEmptyPath
  Goto writePath

deleteEmptyPath:
  StrCmp $0 "" 0 writePathExpand
  DeleteRegValue HKCU "Environment" "Path"
  Goto broadcastPathChange

writePathString:
  WriteRegStr HKCU "Environment" "Path" "$0"
  Goto broadcastPathChange

writePathExpand:
  WriteRegExpandStr HKCU "Environment" "Path" "$0"
broadcastPathChange:
  SendMessage ${HWND_BROADCAST} ${WM_SETTINGCHANGE} 0 "STR:Environment" /TIMEOUT=5000
  Goto doneRemove

doneRemove:
FunctionEnd

; Simple substring search (Haystack on stack, Needle on stack)
Function StrStr
  Exch $R1 ; needle
  Exch
  Exch $R2 ; haystack
  Push $R3
  Push $R4
  Push $R5
  StrLen $R3 $R1
  StrCpy $R4 0
loop:
  StrCpy $R5 $R2 $R3 $R4
  StrCmp $R5 $R1 found
  StrCmp $R5 "" notfound
  IntOp $R4 $R4 + 1
  Goto loop
found:
  StrCpy $R1 $R2 "" $R4
  Goto done
notfound:
  StrCpy $R1 ""
done:
  Pop $R5
  Pop $R4
  Pop $R3
  Pop $R2
  Exch $R1
FunctionEnd

Function un.StrStr
  Exch $R1 ; needle
  Exch
  Exch $R2 ; haystack
  Push $R3
  Push $R4
  Push $R5
  StrLen $R3 $R1
  StrCpy $R4 0
loop:
  StrCpy $R5 $R2 $R3 $R4
  StrCmp $R5 $R1 found
  StrCmp $R5 "" notfound
  IntOp $R4 $R4 + 1
  Goto loop
found:
  StrCpy $R1 $R2 "" $R4
  Goto done
notfound:
  StrCpy $R1 ""
done:
  Pop $R5
  Pop $R4
  Pop $R3
  Pop $R2
  Exch $R1
FunctionEnd

Section "IAPro Nexus" SecMain
  SetOutPath "$INSTDIR"
  File "payload\nexus.exe"
  FileOpen $0 "$INSTDIR\.nexus-nsis" w
  FileClose $0
  File /nonfatal "payload\ai.exe"
  File /nonfatal "payload\nexus-desktop.exe"
  File /nonfatal "payload\nexus-hud.exe"

  Call CaptureOriginalUserPath
  Call AddToUserPath

  IfFileExists "$INSTDIR\nexus-desktop.exe" 0 skipDesktop
    CreateShortcut "$DESKTOP\IAPro Nexus.lnk" "$INSTDIR\nexus-desktop.exe"
  skipDesktop:

  WriteUninstaller "$INSTDIR\Uninstall.exe"

  WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\IAPro Nexus" "DisplayName" "IAPro Nexus"
  WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\IAPro Nexus" "InstallLocation" "$INSTDIR"
  WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\IAPro Nexus" "UninstallString" '"$INSTDIR\Uninstall.exe"'
  WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\IAPro Nexus" "QuietUninstallString" '"$INSTDIR\Uninstall.exe" /S'
  WriteRegDWORD HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\IAPro Nexus" "NoModify" 1
  WriteRegDWORD HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\IAPro Nexus" "NoRepair" 1
SectionEnd

Section "Uninstall"
  Delete "$INSTDIR\nexus.exe"
  Delete "$INSTDIR\.nexus-nsis"
  Delete "$INSTDIR\ai.exe"
  Delete "$INSTDIR\nexus-desktop.exe"
  Delete "$INSTDIR\nexus-hud.exe"
  Delete "$INSTDIR\Uninstall.exe"
  Delete "$DESKTOP\IAPro Nexus.lnk"
  Call un.RemoveFromUserPath
  DeleteRegKey HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\IAPro Nexus"
  RMDir "$INSTDIR"
SectionEnd
