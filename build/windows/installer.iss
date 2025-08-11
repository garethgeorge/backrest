#define B "Backrest"
#define TrayExe "backrest-windows-tray.exe"
#define Website "https://github.com/garethgeorge/backrest/"
; The following is needed to extract the version from the change log.
; If the application executable had the version info, then could use built-in GetVersion* functions.
#define fHandle FileOpen("CHANGELOG.md")
#expr FileRead(fHandle)
#expr FileRead(fHandle)
#define Line FileRead(fHandle)
#expr FileClose(fHandle)
#define VStart Pos("[", Line) + 1
#define VEnd Pos("]", Line)
#define VLen (VEnd - VStart)
#define BackrestVersion Copy(Line, VStart, VLen)

[Setup]
AppName={#B}
AppVersion={#BackrestVersion}
AppVerName={#B} {#BackrestVersion}
AppPublisher=garethgeorge
AppPublisherURL={#Website}
VersionInfoVersion={#BackrestVersion}
DefaultDirName={autopf}\{#B}
DefaultGroupName={#B}
DisableProgramGroupPage=yes
AlwaysShowDirOnReadyPage=yes
UninstallDisplayIcon={app}\icon.ico
#ifndef Arch
  #define Arch "x86_64"
#endif
OutputBaseFilename={#B}Setup-{#Arch}
PrivilegesRequired=lowest
PrivilegesRequiredOverridesAllowed=dialog
UsePreviousPrivileges=no
SetupMutex={#B}Setup
; Disable built-in RestartManager functionality, see comments under [Files].
CloseApplications=no
RestartApplications=no
#if SameText(Arch, "arm64")
  #define ArchAllowed "arm64"
#else
  #define ArchAllowed "x64os"
#endif 
ArchitecturesAllowed={#ArchAllowed}
ArchitecturesInstallIn64BitMode={#ArchAllowed}
SetupLogging=yes
UninstallLogging=yes

[Tasks]
Name: "adminstartcurrent"; Description: "Run {#B} as the current user. Automatically start when the current user logs in. Inherits user access to network resources. Configuration is stored in the current user profile."; GroupDescription: "Execution context"; Check: IsAdminInstallMode; Flags: exclusive
Name: "adminstartsystem"; Description: "Run {#B} as the system user. Automatically start before any user logs in. Configuration is stored in the system user profile."; GroupDescription: "Execution context"; Check: IsAdminInstallMode; Flags: exclusive unchecked
Name: "autostart"; Description: "Automatically start {#B} at log on"; Check: IsUserInstallMode
Name: "desktopicon"; Description: "Create a desktop icon"; Flags: unchecked
Name: "addtopath"; Description: "Add {#B} directory to PATH ({code:GetEnvTarget})"; Flags: unchecked
#define PortDesc "Select a network port for the web interface"
Name: "port9898"; Description: "Default (9898)"; GroupDescription: "{#PortDesc}"; Flags: exclusive; Check: IsUserInstallMode
Name: "port9899"; Description: "9899"; GroupDescription: "{#PortDesc}"; Flags: exclusive unchecked; Check: IsUserInstallMode
Name: "port9900"; Description: "9900"; GroupDescription: "{#PortDesc}"; Flags: exclusive unchecked; Check: IsUserInstallMode
Name: "port9901"; Description: "9901"; GroupDescription: "{#PortDesc}"; Flags: exclusive unchecked; Check: IsUserInstallMode
Name: "port9902"; Description: "9902"; GroupDescription: "{#PortDesc}"; Flags: exclusive unchecked; Check: IsUserInstallMode

[Files]
; Need to stop Backrest not only when uninstalling but also before upgrades or reinstalls
; This is only an issue when Backrest runs as SYSTEM or non-current user. Inno can natively close applications in other cases,
; but doing it the same way in all cases for consistency.
Source: "LICENSE"; DestDir: "{app}"; Flags: ignoreversion; BeforeInstall: StopBackrest
Source: "icon.ico"; DestDir: "{app}"; Flags: ignoreversion
Source: "{#TrayExe}"; DestDir: "{app}"; Flags: ignoreversion
Source: "backrest.exe"; DestDir: "{app}"; Flags: ignoreversion

[Icons]
; For user install mode only.
Name: "{autostartup}\{#B} systray"; Filename: "{app}\{#TrayExe}"; Parameters: "{code:GetPortParam}"; IconFilename: "{app}\icon.ico"; Check: IsUserInstallMode
Name: "{group}\{#B} systray"; Filename: "{app}\{#TrayExe}"; Parameters: "{code:GetPortParam}"; IconFilename: "{app}\icon.ico"; Check: IsUserInstallMode
; For both modes.
Name: "{group}\{#B}{code:GetIconSuffix}"; Filename: "http://localhost:{code:GetPort}/"; IconFilename: "{app}\icon.ico"
Name: "{group}\{#B} website"; Filename: "{#Website}"
Name: "{autodesktop}\{#B}{code:GetIconSuffix}"; Filename: "http://localhost:{code:GetPort}/"; IconFilename: "{app}\icon.ico"; Tasks: desktopicon

[Messages]
PrivilegesRequiredOverrideText2=%1 can be installed to run with standard or administrative privileges.%n%nIf you need to use Windows VSS feature with "--use-fs-snapshot" restic option, select the administrative option.%nProtecting Backrest web UI with a password is highly recommended, especially with the administrative install.
PrivilegesRequiredOverrideAllUsers=Install &system-wide with administrative privileges

[Run]
; Use Task Scheduler to run Backrest elevated. The 30s delay is needed to avoid an issue with tray icon being broken.
; The double-quotes escape double-quotes inside the parameter. The backslash escapes double-quotes inside the -Command block.
Filename: "powershell.exe"; Parameters: "-ExecutionPolicy Bypass -Command ""$t = New-ScheduledTaskTrigger -AtLogOn -User $env:USERNAME ; $t.Delay = 'PT30S'; Register-ScheduledTask -Force -TaskName '{#B}' -RunLevel Highest -Trigger $t -Action $(New-ScheduledTaskAction -Execute \""{app}\{#TrayExe}\"" -Argument '--bind-address 127.0.0.1:9897' -WorkingDirectory '{app}') -Settings $(New-ScheduledTaskSettingsSet -AllowStartIfOnBatteries -ExecutionTimeLimit 0); Start-ScheduledTask -TaskName '{#B}'"" "; Flags: runascurrentuser logoutput runhidden; Tasks: adminstartcurrent; Check: IsAdminInstallMode
; System user task. No need for systray here, and running it without returning control is the only way to stop it gracefully later.
Filename: "powershell.exe"; Parameters: "-ExecutionPolicy Bypass -Command ""Register-ScheduledTask -Force -TaskName '{#B}' -RunLevel Highest -User System -Trigger $(New-ScheduledTaskTrigger -AtStartup) -Action $(New-ScheduledTaskAction -Execute \""{app}\backrest.exe\"" -Argument '--bind-address 127.0.0.1:9897' -WorkingDirectory '{app}') -Settings $(New-ScheduledTaskSettingsSet -AllowStartIfOnBatteries -ExecutionTimeLimit 0); Start-ScheduledTask -TaskName '{#B}'"" "; Flags: runascurrentuser logoutput runhidden; Tasks: adminstartsystem; Check: IsAdminInstallMode
; PATH
Filename: "powershell.exe"; Parameters: "-ExecutionPolicy Bypass -Command ""$newp = '{app}'; $a = [Environment]::GetEnvironmentVariable('PATH', '{code:GetEnvTarget}') -split ';' ; if ($a -notcontains $newp) {{ echo 'Adding to PATH'; $a += $newp; $path = $a -join ';' ; [Environment]::SetEnvironmentVariable('PATH', $path, '{code:GetEnvTarget}') }"" "; Flags: logoutput runhidden; Tasks: addtopath
; Remove from PATH for existing installation when unchecked.
; Reuse the same command in multiple places. Two quotes to escape in preprocessor, two more for the Parameters directive.
#define PathDelCmd "-ExecutionPolicy Bypass -Command """"$newp = '{app}'; $a = [Environment]::GetEnvironmentVariable('PATH', '{code:GetEnvTarget}') -split ';' ; if ($a -contains $newp) {{ echo 'Removing from PATH'; $path = ($a | Where-Object {{ $_ -ne $newp }) -join ';' ; [Environment]::SetEnvironmentVariable('PATH', $path, '{code:GetEnvTarget}') }"""" "
Filename: "powershell.exe"; Parameters: "{#PathDelCmd}"; Flags: logoutput runhidden; Tasks: not addtopath; Check: IsExistingInstallation

Filename: "{app}\{#TrayExe}"; Parameters: "{code:GetPortParam}"; Description: "Start {#B} (runs in the system tray)"; Flags: postinstall waituntilidle; Check: IsUserInstallMode
Filename: "http://localhost:{code:GetPort}/"; Description: "Open {#B} user interface"; Flags: postinstall shellexec

[UninstallRun]
Filename: "powershell.exe"; Parameters: "{#PathDelCmd}"; Flags: logoutput runhidden; RunOnceId: "RemoveFromPath"

[UninstallDelete]
Type: files; Name: "{app}\restic*.exe"
Type: files; Name: "{app}\install.lock"
; Built-in deletion runs before this section and fails to remove the directory due to the files above.
Type: dirifempty; Name: "{app}"

[Code]
var
  UserInstallationExists, AdminInstallationExists: Boolean;
  PreviousAdminUser, PreviousAdminTasks, PreviousVersionUser, PreviousVersionAdmin: String;
  AppName, AppDirAdmin, AppDirUser, RegKey, Cmd: String;
  
procedure AssignGlobals();
begin
  AppName := ExpandConstant('{#SetupSetting("AppName")}');
  Cmd := ExpandConstant('{cmd}');
  RegKey := 'SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\' + AppName + '_is1';
  if RegQueryStringValue(HKEY_CURRENT_USER, RegKey, 'Inno Setup: App Path', AppDirUser)
    then UserInstallationExists := True else UserInstallationExists := False;
  if RegQueryStringValue(HKEY_LOCAL_MACHINE, RegKey, 'Inno Setup: App Path', AppDirAdmin)
    then AdminInstallationExists := True else AdminInstallationExists := False;
  RegQueryStringValue(HKEY_CURRENT_USER, RegKey, 'DisplayVersion', PreviousVersionUser)
  RegQueryStringValue(HKEY_LOCAL_MACHINE, RegKey, 'DisplayVersion', PreviousVersionAdmin)
  RegQueryStringValue(HKEY_LOCAL_MACHINE, RegKey, 'Inno Setup: Selected Tasks', PreviousAdminTasks);
  RegQueryStringValue(HKEY_LOCAL_MACHINE, RegKey, 'Inno Setup: User', PreviousAdminUser);
end;

function IsUserInstallMode(): Boolean;
begin
  if IsAdminInstallMode then Result := False else Result := True;
end;

function GetPort(Param: String): String;
var
  S: String;
  A: array of String;
  i: Integer;
begin
  if IsAdminInstallMode then Result := '9897'
  else
  begin
    A := StringSplit(WizardSelectedTasks(False), [','], stAll);
    for i := 0 to GetArrayLength(A) - 1 do
    begin
      S := A[i];
      if Pos('port', S) > 0 then StringChangeEx(S, 'port', '', True);
    end;
    Result := S;
  end;
end;

function GetPortParam(Param: String): String;
var
  S: String;
begin
  S := GetPort('');
  // Don't add any shortcut parameters for default port selection.
  if S = '9898' then Result := '' else Result := '--bind-address 127.0.0.1:' + S;
end;

function GetIconSuffix(Param: String): String;
begin
  if IsAdminInstallMode and UserInstallationExists then Result := ' (system-wide)'
  else if IsUserInstallMode and AdminInstallationExists then Result := ' (current user)'
  else Result := '';
end;

function GetEnvTarget(Param: String): String;
begin
  if IsAdminInstallMode then Result := 'Machine' else Result := 'User';
end;

function IsExistingInstallation(): Boolean;
begin
  if UserInstallationExists or AdminInstallationExists then Result := True else Result := False;
end;

procedure StopBackrest();
var
  ResultCode: Integer;
begin
  // Attempt to terminate Backrest gracefully for the current user, wait for a second (since taskkill returns immediately),
  // then check and kill forcefully if it's still running.
  if IsUserInstallMode then
    ExecAndLogOutput(Cmd, '/C "taskkill /FI "USERNAME eq %USERNAME%" /IM "backrest-windows-tray.exe" & ping -n 2 127.0.0.1 >nul & tasklist /FI "USERNAME eq %USERNAME%" | findstr /I /V "setup" | findstr "backrest" && (echo Forcing & taskkill /FI "USERNAME eq %USERNAME%" /IM "backrest-windows-tray.exe" /F & taskkill /FI "USERNAME eq %USERNAME%" /IM "backrest.exe" /F)" ',
    '', SW_HIDE, ewWaitUntilTerminated, ResultCode, nil)
  // For admin installs stop through the task scheduler. For the SYSTEM or non-current user this is the only way to stop gracefully.
  // Ending the scheduled task makes a gracefull attempt, then force kills.
  else if IsAdminInstallMode then
  begin
    ExecAndLogOutput(Cmd, '/C schtasks /End /TN ' + AppName + ' || (taskkill /FI "USERNAME eq %USERNAME%" /IM "backrest-windows-tray.exe" /F & taskkill /FI "USERNAME eq %USERNAME%" /IM "backrest.exe" /F)',
    '', SW_HIDE, ewWaitUntilTerminated, ResultCode, nil);
    // Remove the task when uninstalling.
    if IsUninstaller then
      ExecAndLogOutput(Cmd, '/C schtasks /Delete /TN ' + AppName + ' /F', '', SW_HIDE, ewWaitUntilTerminated, ResultCode, nil);
  end;
end;

procedure ShowUpgradeMsg(AppDir: String; InstalledVersion: String);
var
  AppVersion, Msg: String;
  InstalledVersionFull, NewVersionFull: Int64;
  CompResult: Integer;
begin
  AppVersion := ExpandConstant('{#SetupSetting("AppVersion")}');
  // Convert both old and new versions to the correct type. It also automatically adds '.0' if '0.0.0' format is used.
  if StrToVersion(InstalledVersion, InstalledVersionFull) and StrToVersion(AppVersion, NewVersionFull) then
  begin
    CompResult := ComparePackedVersion(InstalledVersionFull, NewVersionFull);
    if CompResult < 0 then Msg := 'upgrade'
    else if CompResult = 0 then Msg := 'reinstall'
    else if CompResult > 0 then Msg := 'downgrade'
    else Msg := 'upgrade/reinstall/downgrade';
  end;
  MsgBox('Detected existing installation of Backrest ' + InstalledVersion + 
    ' in ' + Chr(13) + Chr(10) + AppDir + Chr(13) + Chr(10) + Chr(13) + Chr(10) +
    'Setup will ' + Msg + ' to version ' + AppVersion + '.', mbInformation, MB_OK);
end;

function NextButtonClick(CurPageID: Integer): Boolean;
var
  ResultPortCheck: Integer;
begin
  Result := True
  if CurPageID = wpSelectTasks then
  begin
    // Prevent installing an admin instance when a user instance exists under the same user.
    // It wouldn't work anyway due to the same configuration path.
    if IsAdminInstallMode and WizardIsTaskSelected('adminstartcurrent') and UserInstallationExists then
    begin
      MsgBox('Detected an existing non-administrative installation under the same user ' + GetUserNameString +
      '. Cannot proceed with this selection.', mbError, MB_OK)
      Result := False;
    end
    else if IsUserInstallMode and not UserInstallationExists then
    begin
      if ExecAndLogOutput(Cmd, '/C netstat -na | findstr LISTENING | findstr /C:":' + GetPort('') + ' "', '', SW_HIDE,
          ewWaitUntilTerminated, ResultPortCheck, nil) and (ResultPortCheck = 0) then
      begin
        MsgBox('Selected port is in use, choose another one and try again.', mbError, MB_OK)
        Result := False;
      end;
    end;
  end;
end;

procedure RunListClickCheck(Sender: TObject);
begin
  if not WizardForm.RunList.Checked[0] then
  begin
    WizardForm.RunList.Checked[1] := False;
    WizardForm.RunList.ItemEnabled[1] := False;
  end
  else WizardForm.RunList.ItemEnabled[1] := True;
end;

procedure CurPageChanged(CurPageID: Integer);
var
  i: Integer;
begin
  if CurPageID = wpSelectTasks then
    // Prevent the user from switching installation type back and forth in admin mode.
    // It would cause issues with terminating processes and also with Backrest data being in different user profiles.
    if IsAdminInstallMode and AdminInstallationExists then
      for i := 1 to 2 do WizardForm.TasksList.ItemEnabled[i] := False
end;

procedure InitializeWizard();
begin
  WizardForm.ReadyMemo.ScrollBars := ssVertical;
  WizardForm.ReadyMemo.WordWrap := True;
  // Uncheck and disable the "open" checkbox when the "start" is unchecked on the Finish page.
  if IsUserInstallMode then WizardForm.RunList.OnClickCheck := @RunListClickCheck;
end;

function InitializeSetup(): Boolean;
var
  OldUninstaller: String;
  I: Integer;
begin
  Result := True;
  AssignGlobals;
  // Check for presence of the old Nullsoft installation.
  if RegQueryStringValue(HKEY_CURRENT_USER, 'Software\Microsoft\Windows\CurrentVersion\Uninstall\Backrest',
    'UninstallString', OldUninstaller) then
  begin
    MsgBox('Detected an existing installation done by the old installer that needs to be uninstalled before proceeding. Your configuration will not be impacted.' + Chr(13) + Chr(10) + Chr(13) + Chr(10) +
    'Re-run this setup after uninstallation is complete.', mbInformation, MB_OK);
    ExecAsOriginalUser(OldUninstaller, '', '', SW_SHOWNORMAL, ewNoWait, I);
    Abort;
  end;
  // Upgrade/reinstall scenarios.
  if IsUserInstallMode and UserInstallationExists then ShowUpgradeMsg(AppDirUser, PreviousVersionUser)
  else if IsUserInstallMode and AdminInstallationExists then
  begin
    if GetUserNameString = PreviousAdminUser then
    begin
      // Prevent installing a user instance when an admin instance already exists under the same user.
      MsgBox('Detected an existing administrative installation under the same user ' + PreviousAdminUser + 
      '. Cannot proceed. Uninstall it and try again. Setup will exit now.', mbError, MB_OK);
      Result := False;
    end
    else begin
    // But allow and notify about an existing admin instance under a different user, if present.
      MsgBox('Detected an existing administrative installation of Backrest ' + PreviousVersionAdmin + ' under user ' +
      PreviousAdminUser + '.' + Chr(13) + Chr(10) + Chr(13) + Chr(10) +
      'Setup will install a non-administrative instance for the current user.', mbInformation, MB_OK);
    end;
  end
  // Notify the user about the existing installation of the same type, if present.
  else if IsAdminInstallMode and AdminInstallationExists then 
  begin
    ShowUpgradeMsg(AppDirAdmin, PreviousVersionAdmin);
    // Warn if attempting to upgrade/reinstall under a different user.
    if (Pos('adminstartcurrent', PreviousAdminTasks) > 0) and (GetUserNameString <> PreviousAdminUser) then
      MsgBox('Warning! Previous installation of this type was done by user ' + PreviousAdminUser +
      '. If you proceed, Backrest will be reinstalled to run under the current user. Configuration will be lost.', mbError, MB_OK);
  end;
end;

function InitializeUninstall(): Boolean;
begin
  Result := True;
  AssignGlobals;
  if AdminInstallationExists and UserInstallationExists then
    MsgBox('Detected two existing installations of Backrest. Ensure you are running the correct uninstaller. Windows Settings - Apps panel shows only one entry at a time.' + Chr(13) + Chr(10) +
    'Use Control Panel - Programs and Features instead to identify the correct uninstaller. Or run unins000.exe directly from the appropriate Backrest directory.', mbInformation, MB_OK);
end;

procedure CurUninstallStepChanged(CurUninstallStep: TUninstallStep);
var
  BackrestConfig: String;
begin
  // Under the system user installation a special profile is used.
  if IsAdminInstallMode and AdminInstallationExists and (Pos('adminstartsystem', PreviousAdminTasks) > 0) then
    BackrestConfig := 'C:\Windows\system32\config\systemprofile\AppData\Roaming\backrest'
  else
    BackrestConfig := GetEnv('APPDATA') + '\backrest';
  case CurUninstallStep of
    usUninstall:
      StopBackrest;
    usDone:
    begin
      if MsgBox('Do you want to delete Backrest configuration in this location?' + Chr(13) + Chr(10) + BackrestConfig,
        mbConfirmation, MB_YESNO or MB_DEFBUTTON2) = IDYES then
        if DelTree(BackrestConfig, True, True, True) then MsgBox('Done!', mbInformation, MB_OK)
        else MsgBox('Failed to remove the path', mbInformation, MB_OK);
    end;
  end;
end;
