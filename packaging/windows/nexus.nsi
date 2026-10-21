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
  StrCpy $1 0
  StrCpy $2 ""
  StrLen $3 $0
checkPathSegment:
  StrCmp $1 $3 checkLastPathSegment
  StrCpy $4 $0 1 $1
  StrCmp $4 ";" completePathSegment
  StrCpy $2 "$2$4"
  IntOp $1 $1 + 1
  Goto checkPathSegment
completePathSegment:
  StrCmp $2 "$INSTDIR" donePath
  StrCpy $2 ""
  IntOp $1 $1 + 1
  Goto checkPathSegment
checkLastPathSegment:
  StrCmp $2 "$INSTDIR" donePath
  StrCmp $0 "" setPathOnly
  StrLen $3 $0
  IntOp $3 $3 - 1
  StrCpy $4 $0 1 $3
  StrCmp $4 ";" appendPath
  StrCpy $0 "$0;"
appendPath:
  StrCpy $0 "$0$INSTDIR"
  Goto writeUserPath
setPathOnly:
  StrCpy $0 "$INSTDIR"
writeUserPath:
  ; Preserve REG_SZ versus REG_EXPAND_SZ when the value exists.
  StrCpy $3 2
  System::Call 'advapi32::RegOpenKeyExW(p 0x80000001, w "Environment", i 0, i 0x20019, *p .r5) i .r6'
  StrCmp $6 0 0 writeExpandedUserPath
  System::Call 'advapi32::RegQueryValueExW(p r5, w "Path", p 0, *i .r3, p 0, p 0) i .r6'
  System::Call 'advapi32::RegCloseKey(p r5)'
  StrCmp $6 0 0 writeExpandedUserPath
  StrCmp $3 1 writeStringUserPath
  Goto writeExpandedUserPath
writeStringUserPath:
  WriteRegStr HKCU "Environment" "Path" "$0"
  Goto broadcast
writeExpandedUserPath:
  WriteRegExpandStr HKCU "Environment" "Path" "$0"
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
  StrCpy $1 0
  StrCpy $2 ""
  StrCpy $3 ""
  StrCpy $4 1
  StrLen $5 $0
removePathSegment:
  StrCmp $1 $5 removeLastPathSegment
  StrCpy $6 $0 1 $1
  StrCmp $6 ";" keepOrDropPathSegment
  StrCpy $2 "$2$6"
  IntOp $1 $1 + 1
  Goto removePathSegment
keepOrDropPathSegment:
  StrCmp $2 "$INSTDIR" dropPathSegment
  StrCmp $4 1 0 appendKeptPathSegment
  StrCpy $3 "$2"
  StrCpy $4 0
  Goto dropPathSegment
appendKeptPathSegment:
  StrCpy $3 "$3;$2"
dropPathSegment:
  StrCpy $2 ""
  IntOp $1 $1 + 1
  Goto removePathSegment
removeLastPathSegment:
  StrCmp $2 "$INSTDIR" pathSegmentsDone
  StrCmp $4 1 0 appendLastPathSegment
  StrCpy $3 "$2"
  StrCpy $4 0
  Goto pathSegmentsDone
appendLastPathSegment:
  StrCpy $3 "$3;$2"
pathSegmentsDone:
  StrCpy $0 "$3"
  StrCmp $4 1 emptyCurrentPath
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
    IfFileExists "$DESKTOP\IAPro Nexus.lnk" skipDesktop
    CreateShortcut "$DESKTOP\IAPro Nexus.lnk" "$INSTDIR\nexus-desktop.exe"
    WriteRegDWORD HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\IAPro Nexus" "CreatedDesktopShortcut" 1
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
  ClearErrors
  ReadRegDWORD $0 HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\IAPro Nexus" "CreatedDesktopShortcut"
  IfErrors skipDeleteDesktopShortcut
  IntCmp $0 1 0 skipDeleteDesktopShortcut skipDeleteDesktopShortcut
  Delete "$DESKTOP\IAPro Nexus.lnk"
skipDeleteDesktopShortcut:
  Call un.RemoveFromUserPath
  DeleteRegKey HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\IAPro Nexus"
  RMDir "$INSTDIR"
SectionEnd
