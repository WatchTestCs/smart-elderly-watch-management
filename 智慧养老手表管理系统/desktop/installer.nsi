Unicode true
!include "MUI2.nsh"
!include "LogicLib.nsh"
!include "x64.nsh"
!include "FileFunc.nsh"
!include "WinVer.nsh"

!ifndef APP_VERSION
  !define APP_VERSION "2.0.1"
!endif

!ifdef SIGN_SCRIPT
  !uninstfinalize '"${SIGN_SHELL}" -NoProfile -ExecutionPolicy Bypass -File "${SIGN_SCRIPT}" -FilePath "%1"' = 0
!endif

Name "岁宁"
OutFile "${OUTPUT}"
InstallDir "$LOCALAPPDATA\Programs\Suining"
InstallDirRegKey HKCU "Software\WatchTestCs\Suining" "InstallDir"
RequestExecutionLevel user
SetCompressor /SOLID lzma
SetCompressorDictSize 32
ManifestDPIAware true
VIProductVersion "${APP_VERSION}.0"
VIAddVersionKey /LANG=2052 "ProductName" "岁宁"
VIAddVersionKey /LANG=2052 "FileDescription" "岁宁安装程序"
VIAddVersionKey /LANG=2052 "FileVersion" "${APP_VERSION}"
VIAddVersionKey /LANG=2052 "LegalCopyright" "WatchTestCs"
Icon "suining.ico"
UninstallIcon "suining.ico"
!define MUI_ABORTWARNING
!define MUI_FINISHPAGE_RUN "$INSTDIR\Suining.exe"
!define MUI_FINISHPAGE_RUN_NOTCHECKED
!define MUI_WELCOMEPAGE_TITLE "欢迎使用岁宁"
!define MUI_WELCOMEPAGE_TEXT "岁月有伴，日常安宁。$\r$\n$\r$\n软件已包含所需运行环境，无需另装 Java、Node.js 或数据库。$\r$\n$\r$\n首次打开时创建管理员账号。升级和卸载均保留您在本机保存的记录。安装前请先退出正在运行的岁宁。"
!insertmacro MUI_PAGE_WELCOME
!insertmacro MUI_PAGE_DIRECTORY
!insertmacro MUI_PAGE_INSTFILES
!insertmacro MUI_PAGE_FINISH
!insertmacro MUI_UNPAGE_CONFIRM
!insertmacro MUI_UNPAGE_INSTFILES
!insertmacro MUI_LANGUAGE "SimpChinese"

Function .onInit
  ${IfNot} ${RunningX64}
    MessageBox MB_ICONSTOP "岁宁需要 64 位 Windows 10 或更新版本。"
    Abort
  ${EndIf}
  ${IfNot} ${AtLeastWin10}
    MessageBox MB_ICONSTOP "岁宁需要 Windows 10 或更新版本。"
    Abort
  ${EndIf}
FunctionEnd

Section "岁宁" SEC_MAIN
  SetShellVarContext current
  SetOutPath "$INSTDIR"
  ${If} ${FileExists} "$INSTDIR\Suining.exe"
    System::Call 'kernel32::CreateFileW(w "$INSTDIR\Suining.exe", i 0x40000000, i 0, p 0, i 3, i 0x80, p 0) p .r0'
    ${If} $0 == -1
      MessageBox MB_ICONSTOP "请先退出正在运行的岁宁，再继续安装。"
      SetErrorLevel 1
      Abort
    ${EndIf}
    System::Call 'kernel32::CloseHandle(p r0)'
  ${EndIf}
  SetOverwrite on
  ClearErrors
  File /r "${APPDIR}\*"
  ${If} ${Errors}
    MessageBox MB_ICONSTOP "文件未能写入，请退出正在运行的岁宁后重新安装。"
    SetErrorLevel 1
    Abort
  ${EndIf}
  WriteUninstaller "$INSTDIR\Uninstall.exe"
  CreateDirectory "$SMPROGRAMS\岁宁"
  CreateShortcut "$SMPROGRAMS\岁宁\岁宁.lnk" "$INSTDIR\Suining.exe"
  CreateShortcut "$DESKTOP\岁宁.lnk" "$INSTDIR\Suining.exe"
  WriteRegStr HKCU "Software\WatchTestCs\Suining" "InstallDir" "$INSTDIR"
  WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\Suining" "DisplayName" "岁宁"
  WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\Suining" "DisplayVersion" "${APP_VERSION}"
  WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\Suining" "Publisher" "WatchTestCs"
  WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\Suining" "DisplayIcon" "$INSTDIR\Suining.exe"
  WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\Suining" "UninstallString" `"$INSTDIR\Uninstall.exe"`
  WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\Suining" "QuietUninstallString" `"$INSTDIR\Uninstall.exe" /S`
  WriteRegDWORD HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\Suining" "NoModify" 1
  WriteRegDWORD HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\Suining" "NoRepair" 1
SectionEnd

Section "Uninstall"
  SetShellVarContext current
  Delete "$DESKTOP\岁宁.lnk"
  Delete "$SMPROGRAMS\岁宁\岁宁.lnk"
  RMDir "$SMPROGRAMS\岁宁"
  DeleteRegKey HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\Suining"
  DeleteRegKey HKCU "Software\WatchTestCs\Suining"
  ; Delete only files shipped by this installer; preserve unrelated files and user data.
  !include "${UNINSTALL_FILES}"
  Delete "$INSTDIR\Uninstall.exe"
  RMDir "$INSTDIR"
SectionEnd
