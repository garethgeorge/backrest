!define BUILD_DIR "."
!define OUT_DIR "."
!define APP_NAME "Backrest"
!define COMP_NAME "garethgeorge"
!define WEB_SITE "https://github.com/garethgeorge/backrest"
!define COPYRIGHT "garethgeorge   2024"
!define DESCRIPTION "${APP_NAME} installer"
!define LICENSE_TXT "${BUILD_DIR}\LICENSE"
!define INSTALLER_NAME "${OUT_DIR}\Backrest-setup.exe"
!define MAIN_APP_EXE "backrest-windows-tray.exe"
!define INSTALL_TYPE "SetShellVarContext current"
!define REG_ROOT "HKCU"
!define REG_UNINSTALL_PATH "Software\Microsoft\Windows\CurrentVersion\Uninstall\${APP_NAME}"
!define REG_START_MENU "Start Menu Folder"
# Extract version from the changelog.
!searchparse /file "${BUILD_DIR}\CHANGELOG.md" `## [` VERSION_LOG `]`
# NSIS requires X.X.X.X format in VIProductVersion. Use it everywhere for consistency.
!define VERSION "${VERSION_LOG}.0"
# User variables.
Var SM_folder
Var UIPort
Var WelcomePortNote
Var Prev_version

######################################################################
# Installer file properties
VIProductVersion "${VERSION}"
VIAddVersionKey "ProductName"  "${APP_NAME}"
VIAddVersionKey "CompanyName"  "${COMP_NAME}"
VIAddVersionKey "LegalCopyright"  "${COPYRIGHT}"
VIAddVersionKey "FileDescription"  "${DESCRIPTION}"
VIAddVersionKey "FileVersion"  "${VERSION}"
VIAddVersionKey "ProductVersion" "${VERSION}"

######################################################################
# Installer settings
RequestExecutionLevel user
SetCompressor LZMA
Name "${APP_NAME}"
OutFile "${INSTALLER_NAME}"
XPStyle on
InstallDirRegKey "${REG_ROOT}" "${REG_UNINSTALL_PATH}" ""
InstallDir "$LOCALAPPDATA\Programs\Backrest"
ManifestDPIAware true
ShowInstDetails show
ShowUninstDetails show
# Include NSIS scripts used by this script.
!include "MUI2.nsh"
!include "LogicLib.nsh"
!include "StrFunc.nsh"
# Declare used built-in functions.
${StrStr}

######################################################################
# GUI pages
# Interface configuration, applies to all pages.
!define MUI_ABORTWARNING
!define MUI_UNABORTWARNING
!define MUI_COMPONENTSPAGE_NODESC

!define MUI_TEXT_WELCOME_INFO_TEXT "Setup will guide you through the installation of Backrest.$\r$\n$\r$\nBackrest binds to 127.0.0.1:9898 for web UI. You may change the port using BACKREST_PORT environment variable (see installation documentation). If setting the variable, exit and re-run this installer to have it pick up the new value.$\r$\n$\r$\nYou must use a custom port if there are other Backrest instances running concurrently on this system.$WelcomePortNote$\r$\n$\r$\n$_CLICK"
!insertmacro MUI_PAGE_WELCOME
!insertmacro MUI_PAGE_LICENSE "${LICENSE_TXT}"
!insertmacro MUI_PAGE_COMPONENTS
!insertmacro MUI_PAGE_DIRECTORY

!define MUI_STARTMENUPAGE_REGISTRY_ROOT "${REG_ROOT}"
!define MUI_STARTMENUPAGE_REGISTRY_KEY "${REG_UNINSTALL_PATH}"
!define MUI_STARTMENUPAGE_REGISTRY_VALUENAME "${REG_START_MENU}"
# Get Start Menu folder name from the dialog screen and store it in a variable.
!insertmacro MUI_PAGE_STARTMENU "Application" $SM_folder
!insertmacro MUI_PAGE_INSTFILES

!define MUI_FINISHPAGE_RUN "$INSTDIR\${MAIN_APP_EXE}"
!define MUI_FINISHPAGE_RUN_TEXT "Start ${APP_NAME} (runs in the system tray)"
# Use the built-in readme option to open the app URL.
!define MUI_FINISHPAGE_SHOWREADME http://localhost:$UIPort/
!define MUI_FINISHPAGE_SHOWREADME_TEXT "Open Backrest user interface"
!insertmacro MUI_PAGE_FINISH

# Uninstall pages.
!insertmacro MUI_UNPAGE_CONFIRM
!insertmacro MUI_UNPAGE_INSTFILES
!insertmacro MUI_UNPAGE_FINISH

!insertmacro MUI_LANGUAGE "English"

######################################################################
# Functions
# Have to define the function this way to allow re-using it in the uninstall section.
!macro KillProc UN
Function ${UN}KillProc
DetailPrint "Stopping Backrest if it is running..."
# Gracefully attempt to stop Backrest process in the current session.
# Do it 5 times 1 second apart, then kill forcefully.
nsExec::ExecToLog 'powershell.exe -Command "while ((Get-Process -Name backrest-windows-tray -ea SilentlyContinue | where { $$_.SessionId -eq (Get-Process -Id $$PID).SessionId }) -and ($$count -ne 5)) { $$count++ ; taskkill /FI """USERNAME eq $$([Environment]::UserName)""" /IM backrest-windows-tray.exe; sleep 1 }; if ($$count -eq 5) { Stop-Process -Name backrest-windows-tray }" '
FunctionEnd
!macroend
!insertmacro KillProc ""
!insertmacro KillProc "un."

Function .onInit
# Old pre-1.6.2 installer installed into C:\Program Files; override the default path when upgrading.
# $R0, $R1 etc are registers; used here as local temp variables.
ReadRegStr $Prev_version ${REG_ROOT} "${REG_UNINSTALL_PATH}" "DisplayVersion"
${If} "$Prev_version" == "00.00.00.00"
StrCpy $INSTDIR "$LOCALAPPDATA\Programs\Backrest"
${EndIf}
# Read BACKREST_PORT environment variable.
ReadEnvStr $R1 BACKREST_PORT
${If} "$R1" == ""
  # Use the default port and welcome text if the var is empty.
  StrCpy $UIPort "9898"
  StrCpy $WelcomePortNote ""
${Else}
  # Extract substring starting with a colon, assign to $R2.
  ${StrStr} $R2 $R1 ":"
  # Assign the value to $UIPort, omitting the first character (:).
  StrCpy $UIPort $R2 "" 1
  StrCpy $WelcomePortNote "$\r$\n$\r$\nNOTE: detected BACKREST_PORT variable present. Will use port $UIPort for shortcuts."
${EndIf}
FunctionEnd

######################################################################
# Sections
Section "Application files"
SectionIn RO
${INSTALL_TYPE}
Call KillProc
# Clean up remnants from the old installer (except for items in "Program Files" which would require elevation).
${If} "$Prev_version" == "00.00.00.00"
  Delete "$DESKTOP\${APP_NAME} Console.lnk"
  Delete "$SMPROGRAMS\$SM_Folder\${APP_NAME} Website.lnk"
  Delete "$SMPROGRAMS\${APP_NAME}\Uninstall ${APP_NAME}.lnk"
  DeleteRegKey ${REG_ROOT} "Software\Microsoft\Windows\CurrentVersion\App Paths\${MAIN_APP_EXE}"
${EndIf}

SetOverwrite ifnewer
SetOutPath "$INSTDIR"
File "${BUILD_DIR}\backrest.exe"
File "${BUILD_DIR}\backrest-windows-tray.exe"
File "${BUILD_DIR}\LICENSE"
File "${BUILD_DIR}\icon.ico"
WriteUninstaller "$INSTDIR\uninstall.exe"

# Start Menu shortcuts
!insertmacro MUI_STARTMENU_WRITE_BEGIN "Application"
CreateDirectory "$SMPROGRAMS\$SM_folder"
CreateShortCut "$SMPROGRAMS\$SM_folder\${APP_NAME}.lnk" "$INSTDIR\${MAIN_APP_EXE}" "" "$INSTDIR\icon.ico" 0
CreateShortCut "$SMPROGRAMS\$SM_folder\${APP_NAME} UI.lnk" "http://localhost:$UIPort/" "" "$INSTDIR\icon.ico" 0
WriteIniStr "$SMPROGRAMS\$SM_folder\${APP_NAME} website.url" "InternetShortcut" "URL" "${WEB_SITE}"
!insertmacro MUI_STARTMENU_WRITE_END

# Registry entries
WriteRegStr ${REG_ROOT} "${REG_UNINSTALL_PATH}" "DisplayName" "${APP_NAME}"
WriteRegStr ${REG_ROOT} "${REG_UNINSTALL_PATH}" "UninstallString" "$INSTDIR\uninstall.exe"
WriteRegStr ${REG_ROOT} "${REG_UNINSTALL_PATH}" "DisplayIcon" "$INSTDIR\icon.ico"
WriteRegStr ${REG_ROOT} "${REG_UNINSTALL_PATH}" "DisplayVersion" "${VERSION}"
WriteRegStr ${REG_ROOT} "${REG_UNINSTALL_PATH}" "Publisher" "${COMP_NAME}"
WriteRegStr ${REG_ROOT} "${REG_UNINSTALL_PATH}" "URLInfoAbout" "${WEB_SITE}"
SectionEnd

Section "Run application at startup (recommended)"
CreateDirectory $SMSTARTUP
CreateShortcut "$SMSTARTUP\${APP_NAME}.lnk" "$INSTDIR\${MAIN_APP_EXE}" "" "$INSTDIR\icon.ico" 0
SectionEnd

Section "Desktop shortcut"
CreateShortCut "$DESKTOP\${APP_NAME} UI.lnk" "http://localhost:$UIPort/" "" "$INSTDIR\icon.ico" 0
SectionEnd

Section "Uninstall"
${INSTALL_TYPE}
Call un.KillProc
Delete "$INSTDIR\LICENSE"
Delete "$INSTDIR\icon.ico"
Delete "$INSTDIR\install.lock"
Delete "$INSTDIR\restic*.exe"
Delete "$INSTDIR\backrest.exe"
Delete "$INSTDIR\backrest-windows-tray.exe"
Delete "$INSTDIR\uninstall.exe"
RmDir "$INSTDIR"
# Get Start Menu folder name from the registry and delete shortcuts.
!insertmacro MUI_STARTMENU_GETFOLDER "Application" $SM_folder
Delete "$SMPROGRAMS\$SM_folder\${APP_NAME}.lnk"
Delete "$SMPROGRAMS\$SM_folder\${APP_NAME} UI.lnk"
Delete "$SMPROGRAMS\$SM_folder\${APP_NAME} website.url"
RmDir "$SMPROGRAMS\$SM_folder"
# Startup and desktop shortcuts
Delete "$SMSTARTUP\${APP_NAME}.lnk"
Delete "$DESKTOP\${APP_NAME} UI.lnk"
# Registry
DeleteRegKey ${REG_ROOT} "${REG_UNINSTALL_PATH}"
SectionEnd
