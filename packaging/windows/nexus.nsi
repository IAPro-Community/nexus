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

Section "IAPro Nexus" SecMain
  SetOutPath "$INSTDIR"
  File "payload\nexus.exe"
  FileOpen $0 "$INSTDIR\.nexus-nsis" w
  FileClose $0
  File /nonfatal "payload\ai.exe"
  File /nonfatal "payload\nexus-desktop.exe"

  Call AddToUserPath

  IfFileExists "$INSTDIR\nexus-desktop.exe" 0 skipDesktop
    CreateShortcut "$DESKTOP\IAPro Nexus.lnk" "$INSTDIR\nexus-desktop.exe"
  skipDesktop:

  WriteUninstaller "$INSTDIR\Uninstall.exe"
SectionEnd

Section "Uninstall"
  Delete "$INSTDIR\nexus.exe"
  Delete "$INSTDIR\.nexus-nsis"
  Delete "$INSTDIR\ai.exe"
  Delete "$INSTDIR\nexus-desktop.exe"
  Delete "$INSTDIR\Uninstall.exe"
  Delete "$DESKTOP\IAPro Nexus.lnk"
  RMDir "$INSTDIR"
SectionEnd
