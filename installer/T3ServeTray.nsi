; T3 Serve Tray installer (NSIS 3, Unicode, per-user)

Unicode true
RequestExecutionLevel user

Name "T3 Serve Tray"
Icon "..\internal\assets\tray.ico"
UninstallIcon "..\internal\assets\tray.ico"
OutFile "..\dist\T3ServeTray-setup.exe"
InstallDir "$LOCALAPPDATA\Programs\T3ServeTray"
InstallDirRegKey HKCU "Software\T3ServeTray" "InstallDir"

Function .onInit
  SetShellVarContext current
FunctionEnd

Function un.onInit
  SetShellVarContext current
FunctionEnd

Page directory
Page instfiles

UninstPage uninstConfirm
UninstPage instfiles

Section "Install"
  SetShellVarContext current
  SetOutPath "$INSTDIR"
  File "..\dist\t3tray.exe"

  WriteRegStr HKCU "Software\T3ServeTray" "InstallDir" "$INSTDIR"
  WriteUninstaller "$INSTDIR\Uninstall.exe"

  CreateDirectory "$SMPROGRAMS\T3 Serve Tray"
  CreateShortcut "$SMPROGRAMS\T3 Serve Tray\T3 Serve Tray.lnk" "$INSTDIR\t3tray.exe"
  CreateShortcut "$SMPROGRAMS\T3 Serve Tray\Uninstall T3 Serve Tray.lnk" "$INSTDIR\Uninstall.exe"
  CreateShortcut "$SMSTARTUP\T3 Serve Tray.lnk" "$INSTDIR\t3tray.exe"
SectionEnd

Section "Uninstall"
  SetShellVarContext current
  Delete "$SMSTARTUP\T3 Serve Tray.lnk"
  Delete "$SMPROGRAMS\T3 Serve Tray\T3 Serve Tray.lnk"
  Delete "$SMPROGRAMS\T3 Serve Tray\Uninstall T3 Serve Tray.lnk"
  RMDir "$SMPROGRAMS\T3 Serve Tray"

  Delete "$INSTDIR\t3tray.exe"
  Delete "$INSTDIR\Uninstall.exe"
  RMDir "$INSTDIR"

  DeleteRegKey HKCU "Software\T3ServeTray"
SectionEnd
