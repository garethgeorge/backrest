!define BUILD_DIR "."
!define OUT_DIR "."
!define APP_NAME "Backrest"
!define COMP_NAME "garethgeorge"
!define WEB_SITE "https://github.com/garethgeorge/backrest"
!define COPYRIGHT "garethgeorge   2024"
!define DESCRIPTION "${APP_NAME} installer"
!define LICENSE_TXT "${BUILD_DIR}\LICENSE"
!define MAIN_APP_EXE "backrest-windows-tray.exe"
!define INSTALL_TYPE "SetShellVarContext all"
!define REG_ROOT "HKLM"
!define REG_UNINSTALL_PATH "Software\Microsoft\Windows\CurrentVersion\Uninstall\${APP_NAME}"
# Extract version from the changelog.
!searchparse /file "${BUILD_DIR}\CHANGELOG.md" `## [` VERSION `]`
# User variables.
Var UIPort
Var WelcomeTitle
Var WelcomeText
Var WelcomePortNote
Var OldVersion
Var Cmd
Var InstallMode
Var InstallModeLower

######################################################################
# Installer file properties
# NSIS requires X.X.X.X format in VIProductVersion.
VIProductVersion "${VERSION}.0"
VIAddVersionKey "ProductName" "${APP_NAME}"
VIAddVersionKey "CompanyName" "${COMP_NAME}"
VIAddVersionKey "LegalCopyright" "${COPYRIGHT}"
VIAddVersionKey "FileDescription" "${DESCRIPTION}"
VIAddVersionKey "FileVersion" "${VERSION}"
VIAddVersionKey "ProductVersion" "${VERSION}"

######################################################################
# Installer settings
Unicode True
RequestExecutionLevel admin
SetCompressor LZMA
Name "${APP_NAME}"
Caption "$(^Name) ${VERSION} Setup"
!ifdef ARCH
OutFile "${OUT_DIR}\Backrest-${ARCH}-setup.exe"
!else
OutFile "${OUT_DIR}\Backrest-setup.exe"
!endif
XPStyle on
# Default installation directory.
InstallDir "$PROGRAMFILES\Backrest"
# If existing installation is detected, use that directory instead.
InstallDirRegKey "${REG_ROOT}" "${REG_UNINSTALL_PATH}" "UninstallString"
ManifestDPIAware true
ShowInstDetails show
ShowUninstDetails show
# Include NSIS headers used by this script.
!include "MUI2.nsh"
!include "LogicLib.nsh"
!include "Memento.nsh"
!include "WordFunc.nsh"
# Defines for the Memento macro. 
!define MEMENTO_REGISTRY_ROOT "${REG_ROOT}"
!define MEMENTO_REGISTRY_KEY "${REG_UNINSTALL_PATH}"

######################################################################
# GUI pages
# Prompt to confirm exiting the installer.
!define MUI_ABORTWARNING
!define MUI_UNABORTWARNING

!define MUI_WELCOMEPAGE_TITLE "$WelcomeTitle"
!define MUI_TEXT_WELCOME_INFO_TEXT "$WelcomeText"
!define MUI_PAGE_CUSTOMFUNCTION_PRE onPreWelcome
!define MUI_PAGE_CUSTOMFUNCTION_LEAVE onLeaveWelcome
!insertmacro MUI_PAGE_WELCOME

!insertmacro MUI_PAGE_LICENSE "${LICENSE_TXT}"

!define MUI_COMPONENTSPAGE_NODESC
!define MUI_COMPONENTSPAGE_TEXT_COMPLIST "Select components to install:$\r$\n$\r$\nSelections will be remembered for future upgrades"
!define MUI_PAGE_CUSTOMFUNCTION_PRE onPreComponents
!insertmacro MUI_PAGE_COMPONENTS

!define MUI_PAGE_CUSTOMFUNCTION_PRE onPreDirectory
!insertmacro MUI_PAGE_DIRECTORY

!insertmacro MUI_PAGE_INSTFILES

!define MUI_FINISHPAGE_RUN "$INSTDIR\${MAIN_APP_EXE}"
!define MUI_FINISHPAGE_RUN_TEXT "&Start ${APP_NAME} (runs in the system tray)"
# Use the built-in readme option to open the app URL.
!define MUI_FINISHPAGE_SHOWREADME http://localhost:$UIPort/
!define MUI_FINISHPAGE_SHOWREADME_TEXT "&Open Backrest user interface"
!define MUI_PAGE_CUSTOMFUNCTION_SHOW onShowFinish
!insertmacro MUI_PAGE_FINISH

# Uninstall pages.
!define MUI_UNFINISHPAGE_NOAUTOCLOSE
!insertmacro MUI_UNPAGE_CONFIRM
!insertmacro MUI_UNPAGE_INSTFILES
!insertmacro MUI_UNPAGE_FINISH

!insertmacro MUI_LANGUAGE "English"

######################################################################
# Functions
# Have to define the function this way to allow re-using it in the uninstall section.
!macro KillProcess UN
Function ${UN}KillProcess
ReadEnvStr $Cmd COMSPEC
DetailPrint "Stopping Backrest if it is running..."
# Gracefully attempt to stop Backrest processes for the current user.
# Do it 5 times, then kill forcefully.
nsExec::ExecToLog '$Cmd /C echo off & (for /L %i in (1,1,5) do tasklist /FI "USERNAME eq %USERNAME%" | findstr /I /V "setup" | findstr "backrest" && taskkill /FI "USERNAME eq %USERNAME%" /IM "backrest-windows-tray.exe" || exit) & taskkill /FI "USERNAME eq %USERNAME%" /IM "backrest-windows-tray.exe" /F & taskkill /FI "USERNAME eq %USERNAME%" /IM "backrest.exe" /F '
FunctionEnd
!macroend
!insertmacro KillProcess ""
!insertmacro KillProcess "un."

Function .onInit
# $R0, $R1 etc are registers; used here as local variables.
# Read some environment variables.
ReadEnvStr $Cmd COMSPEC
ReadEnvStr $R1 BACKREST_PORT
${If} "$R1" == ""
  # Use the default port and welcome text if the var is empty.
  StrCpy $UIPort "9898"
  StrCpy $WelcomePortNote ""
${Else}
  # Extract port number.
  ${WordFind} "$R1" ":" "+2" $UIPort
  StrCpy $WelcomePortNote "$\r$\n$\r$\nNOTE: detected BACKREST_PORT environment variable. Will use port $UIPort for shortcuts."
${EndIf}

# Read the previous Backrest version, if any.
ReadRegStr $OldVersion ${REG_ROOT} "${REG_UNINSTALL_PATH}" "DisplayVersion"
${If} "$OldVersion" == "00.00.00.00"
  # Old pre-1.6.2 installer installed into C:\Program Files; override the default path when upgrading.
  StrCpy $INSTDIR "$LOCALAPPDATA\Programs\Backrest"
${EndIf}

${If} "$OldVersion" != ""
  # Detected existing installation.
  ${MementoSectionRestore}
  ${VersionCompare} "$OldVersion" "${VERSION}" $R3
  ${Select} $R3
    ${Case} "0"
      StrCpy $InstallMode "Reinstall"
    ${Case} "1"
      StrCpy $InstallMode "Downgrade"
    ${CaseElse}
      StrCpy $InstallMode "Upgrade"
  ${EndSelect}
  StrCpy $WelcomeTitle "Welcome to ${APP_NAME} $InstallMode"
  # Convert to lowercase for Welcome text.
  ${StrFilter} "$InstallMode" "-" "" "" $InstallModeLower
  StrCpy $WelcomeText "Setup will guide you through the $InstallModeLower of ${APP_NAME} from version $OldVersion to ${VERSION}.$\r$\n$\r$\nInstallation directory is $INSTDIR $WelcomePortNote$\r$\n$\r$\nClick Next to continue."
${Else}
  # New installation.
  # Check if port is already in use and go into the abort mode.
  nsExec::ExecToStack '$Cmd /C netstat.exe -na | findstr LISTENING | findstr ":$UIPort " '
  Pop $R4
  ${If} "$R4" == "0"
    StrCpy "$InstallMode" "Abort"
    StrCpy $WelcomeTitle "Error"
    StrCpy $WelcomeText "*** WARNING ***$\r$\nBackrest binds to port $UIPort for web UI. This port is currently in use by another Backrest instance or another application.$\r$\n$\r$\nPerform the following:$\r$\nClick Start - type $\"environment$\", Enter to open System Properties.\r\nClick Environment Variables. Click New in the top section. Enter BACKREST_PORT as the name and 127.0.0.1:port as the value, where $\"port$\" is a number between 1024 and 65535 (avoid known ports; try 9900), then OK 3 times.\r\nExit and re-run this installer to have it pick up the new value.\r\nSee installation documentation for more details.\r\n$\r$\nClick Exit to exit."
  ${Else}
    StrCpy $WelcomeTitle "Welcome to ${APP_NAME} Setup"
    StrCpy $WelcomeText "Setup will guide you through the installation of ${APP_NAME}.$WelcomePortNote$\r$\n$\r$\nClick Next to continue."
  ${EndIf}
${EndIf}
FunctionEnd

Function onPreWelcome
  ${If} "$InstallMode" == "Abort"
    # Change text on the button.
    GetDlgItem $R5 $HWNDPARENT 1
    ${NSD_SetText} $R5 "&Exit"
  ${EndIf}
FunctionEnd

Function onLeaveWelcome
  ${If} "$InstallMode" == "Abort"
    Quit
  ${EndIf}
FunctionEnd

Function onPreComponents
  ${If} "$InstallMode" != ""
    GetDlgItem $R6 $HWNDPARENT 1
    ${NSD_SetText} $R6 "$(^InstallBtn)"
  ${EndIf}
FunctionEnd

Function onPreDirectory
  # Skip directory page.
  ${If} "$InstallMode" != ""
    Abort
  ${EndIf}
FunctionEnd

Function onShowFinish
  # Run custom functions when the checkboxes are clicked.
  ${NSD_OnClick} $mui.FinishPage.Run onChkRun
FunctionEnd

Function onChkRun
  Pop $R7
  ${NSD_GetState} $mui.FinishPage.Run $7
  ${If} $7 == ${BST_UNCHECKED}
    ${NSD_Uncheck} $mui.FinishPage.ShowReadme
    EnableWindow $mui.FinishPage.ShowReadme 0
  ${Else}
    EnableWindow $mui.FinishPage.ShowReadme 1
  ${EndIf}
FunctionEnd

Function .onInstSuccess
  ${MementoSectionSave}
FunctionEnd

######################################################################
# Sections
Section "Application files"
SectionIn RO
${INSTALL_TYPE}
Call KillProcess
# Clean up remnants from the old installer (except for items in "Program Files" which would require elevation).
${If} "$OldVersion" == "00.00.00.00"
  Delete "$DESKTOP\${APP_NAME} Console.lnk"
  Delete "$SMPROGRAMS\${APP_NAME}\${APP_NAME} Website.lnk"
  Delete "$SMPROGRAMS\${APP_NAME}\Uninstall ${APP_NAME}.lnk"
  DeleteRegKey ${REG_ROOT} "Software\Microsoft\Windows\CurrentVersion\App Paths\${MAIN_APP_EXE}"
${EndIf}

# Allow reinstall and downgrade by overwriting the files.
SetOverwrite on
SetOutPath "$INSTDIR"
File "${BUILD_DIR}\backrest.exe"
File "${BUILD_DIR}\backrest-windows-tray.exe"
File "${BUILD_DIR}\LICENSE"
File "${BUILD_DIR}\icon.ico"
WriteUninstaller "$INSTDIR\uninstall.exe"

# Start Menu shortcuts.
CreateDirectory "$SMPROGRAMS\${APP_NAME}"
CreateShortCut "$SMPROGRAMS\${APP_NAME}\${APP_NAME}.lnk" "$INSTDIR\${MAIN_APP_EXE}" "" "$INSTDIR\icon.ico" 0
CreateShortCut "$SMPROGRAMS\${APP_NAME}\${APP_NAME} UI.lnk" "http://localhost:$UIPort/" "" "$INSTDIR\icon.ico" 0
WriteIniStr "$SMPROGRAMS\${APP_NAME}\${APP_NAME} website.url" "InternetShortcut" "URL" "${WEB_SITE}"

# Registry entries.
WriteRegStr ${REG_ROOT} "${REG_UNINSTALL_PATH}" "DisplayName" "${APP_NAME}"
WriteRegStr ${REG_ROOT} "${REG_UNINSTALL_PATH}" "UninstallString" "$INSTDIR\uninstall.exe"
WriteRegStr ${REG_ROOT} "${REG_UNINSTALL_PATH}" "DisplayIcon" "$INSTDIR\icon.ico"
WriteRegStr ${REG_ROOT} "${REG_UNINSTALL_PATH}" "DisplayVersion" "${VERSION}"
WriteRegStr ${REG_ROOT} "${REG_UNINSTALL_PATH}" "Publisher" "${COMP_NAME}"
WriteRegStr ${REG_ROOT} "${REG_UNINSTALL_PATH}" "URLInfoAbout" "${WEB_SITE}"
WriteRegStr ${REG_ROOT} "${REG_UNINSTALL_PATH}" "InstallLocation" "$INSTDIR"
SectionEnd

${MementoSection} "Run application at startup (recommended)" sect_startup
CreateDirectory $SMSTARTUP
CreateShortcut "$SMSTARTUP\${APP_NAME}.lnk" "$INSTDIR\${MAIN_APP_EXE}" "" "$INSTDIR\icon.ico" 0
${MementoSectionEnd}

${MementoSection} "Run application as Administrator at startup (advanced)" sect_startup_admin
ExecWait 'schtasks /Create /TN "Backrest Startup" /TR "$\"$INSTDIR\${MAIN_APP_EXE}$\"" /SC ONLOGON /RL HIGHEST /F'
${MementoSectionEnd}

${MementoSection} "Desktop shortcut" sect_desktop
CreateShortCut "$DESKTOP\${APP_NAME} UI.lnk" "http://localhost:$UIPort/" "" "$INSTDIR\icon.ico" 0
${MementoSectionEnd}
${MementoSectionDone}

# If a previous installation created the shortcuts, remove them when user deselects
# upon upgrade/reinstall to honour the new choice.
Section "-Remove deselected shortcuts"
${IfNot} ${SectionIsSelected} ${sect_startup}
  Delete "$SMSTARTUP\${APP_NAME}.lnk"
${EndIf}
${IfNot} ${SectionIsSelected} ${sect_startup_admin}
  ExecWait 'schtasks /Delete /TN "Backrest Startup" /F'
${EndIf}
${IfNot} ${SectionIsSelected} ${sect_desktop}
  Delete "$DESKTOP\${APP_NAME} UI.lnk"
${EndIf}
SectionEnd

Section "Uninstall"
${INSTALL_TYPE}
Call un.KillProcess
Delete "$INSTDIR\LICENSE"
Delete "$INSTDIR\icon.ico"
Delete "$INSTDIR\install.lock"
Delete "$INSTDIR\restic*.exe"
Delete "$INSTDIR\backrest.exe"
Delete "$INSTDIR\backrest-windows-tray.exe"
Delete "$INSTDIR\uninstall.exe"
RmDir "$INSTDIR"
# Delete Start Menu shortcuts.
Delete "$SMPROGRAMS\${APP_NAME}\${APP_NAME}.lnk"
Delete "$SMPROGRAMS\${APP_NAME}\${APP_NAME} UI.lnk"
Delete "$SMPROGRAMS\${APP_NAME}\${APP_NAME} website.url"
RmDir "$SMPROGRAMS\${APP_NAME}"
# Startup and desktop shortcuts.
Delete "$SMSTARTUP\${APP_NAME}.lnk"
Delete "$DESKTOP\${APP_NAME} UI.lnk"
ExecWait 'schtasks /Delete /TN "Backrest Startup" /F'
# Registry key.
DeleteRegKey ${REG_ROOT} "${REG_UNINSTALL_PATH}"
SectionEnd
