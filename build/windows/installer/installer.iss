; To compile use: `& "C:\Users\dmesa\AppData\Local\Programs\Inno Setup 6\ISCC.exe" .\installer.iss`

#define AppName "Wiredoor CLI"
#define AppPublisher "Wiredoor"
#define AppURL "https://github.com/wiredoor/wiredoor-cli"

#ifndef AppVersion
  #define AppVersion "1.2.2"
#endif

#ifndef SourceExe
  #define SourceExe "..\..\..\dist\wiredoor_1.2.2_windows_amd64.exe"
#endif

[Setup]
AppId=Wiredoor.WiredoorCLI
AppName={#AppName}
AppVersion={#AppVersion}
AppPublisher={#AppPublisher}
AppPublisherURL={#AppURL}

DefaultDirName={autopf}\Wiredoor
UsePreviousAppDir=no
DisableDirPage=yes
DefaultGroupName=Wiredoor
DisableProgramGroupPage=yes

PrivilegesRequired=admin
ArchitecturesAllowed=x64compatible
ArchitecturesInstallIn64BitMode=x64compatible

OutputDir=..\..\..\dist
OutputBaseFilename=wiredoor_{#AppVersion}_windows_amd64_setup
Compression=lzma2
SolidCompression=yes

ChangesEnvironment=yes
CloseApplications=yes
RestartApplications=yes
WizardStyle=modern

UninstallDisplayName={#AppName}
UninstallDisplayIcon={app}\wiredoor.exe
SetupIconFile=..\icons\wiredoor.ico

[Files]
Source: "{#SourceExe}"; DestDir: "{app}"; DestName: "wiredoor.exe"; Flags: ignoreversion
Source: "{#SourceExe}"; DestDir: "{tmp}"; DestName: "wiredoor-service.exe"; Flags: dontcopy

[Registry]
Root: HKLM; \
  Subkey: "SYSTEM\CurrentControlSet\Control\Session Manager\Environment"; \
  ValueType: expandsz; \
  ValueName: "Path"; \
  ValueData: "{olddata};{app}"; \
  Check: NeedsAddPath

[UninstallRun]
Filename: "{app}\wiredoor.exe"; \
  Parameters: "service uninstall"; \
  Flags: runhidden waituntilterminated skipifdoesntexist; \
  RunOnceId: "UninstallWiredoorService"

[Code]
const
  WireGuardUrl = 'https://download.wireguard.com/windows-client/wireguard-installer.exe';
  WireGuardFile = 'wireguard.msi';

function IsWireGuardInstalled(): Boolean;
begin
  Result := FileExists(
    ExpandConstant('{autopf}\WireGuard\wireguard.exe')
  );
end;

function NeedsAddPath(): Boolean;
var
  Path: String;
  AppPath: String;
begin
  AppPath := ExpandConstant('{app}');

  if not RegQueryStringValue(
    HKLM,
    'SYSTEM\CurrentControlSet\Control\Session Manager\Environment',
    'Path',
    Path
  ) then
  begin
    Result := True;
    Exit;
  end;

  Result := Pos(
    ';' + Uppercase(AppPath) + ';',
    ';' + Uppercase(Path) + ';'
  ) = 0;
end;

procedure RemoveFromPath(RootKey: Integer; Subkey, Directory: String);
var
  Path, Entry: String;
  Position: Integer;
begin
  if not RegQueryStringValue(RootKey, Subkey, 'Path', Path) then
    Exit;

  Path := ';' + Path + ';';
  Entry := ';' + Uppercase(Directory) + ';';
  Position := Pos(Entry, Uppercase(Path));
  if Position = 0 then
    Exit;

  while Position > 0 do
  begin
    Delete(Path, Position, Length(Entry) - 1);
    Position := Pos(Entry, Uppercase(Path));
  end;

  Path := Copy(Path, 2, Length(Path) - 2);
  if not RegWriteExpandStringValue(RootKey, Subkey, 'Path', Path) then
    RaiseException('Could not update PATH.');
end;

procedure RemoveLegacyUserInstall();
var
  OldDirectory, OldExe: String;
begin
  OldDirectory := ExpandConstant('{localappdata}\Wiredoor\bin');
  OldExe := OldDirectory + '\wiredoor.exe';

  if FileExists(OldExe) then
    if not DeleteFile(OldExe) then
      RaiseException('Could not remove ' + OldExe);

  RemoveFromPath(HKCU, 'Environment', OldDirectory);
end;

procedure CurStepChanged(CurStep: TSetupStep);
var
  ResultCode: Integer;
  ExePath: String;
begin
  if CurStep = ssInstall then
  begin
    ExtractTemporaryFile('wiredoor-service.exe');
    if not Exec(
      ExpandConstant('{tmp}\wiredoor-service.exe'),
      'service stop',
      '', SW_HIDE, ewWaitUntilTerminated, ResultCode
    ) then
      RaiseException('Could not stop the existing Wiredoor service: ' +
        SysErrorMessage(ResultCode));

    if ResultCode <> 0 then
      RaiseException('Could not stop the existing Wiredoor service. Exit code: ' +
        IntToStr(ResultCode));
  end;

  if CurStep = ssPostInstall then
  begin
    ExePath := ExpandConstant('{app}\wiredoor.exe');

    if not Exec(
      ExePath,
      'service install',
      ExpandConstant('{app}'),
      SW_HIDE,
      ewWaitUntilTerminated,
      ResultCode
    ) then
      RaiseException(
        'Failed to execute wiredoor service install: ' +
        SysErrorMessage(ResultCode)
      );

    Log('wiredoor service install exit code: ' +
      IntToStr(ResultCode));

    if ResultCode <> 0 then
      RaiseException(
        'wiredoor service install failed with exit code ' +
        IntToStr(ResultCode)
      );

    RemoveLegacyUserInstall();
  end;
end;

procedure CurUninstallStepChanged(CurUninstallStep: TUninstallStep);
begin
  if CurUninstallStep = usPostUninstall then
    RemoveFromPath(HKLM,
      'SYSTEM\CurrentControlSet\Control\Session Manager\Environment',
      ExpandConstant('{app}'));
end;

function ServiceExists: Boolean;
var
  ResultCode: Integer;
begin
  Result :=
    Exec('sc.exe', 'query wiredoorService', '', SW_HIDE, ewWaitUntilTerminated, ResultCode)
    and (ResultCode = 0);
end;

function WireGuardInstalled: Boolean;
var
  ResultCode: Integer;
begin
  Result :=
    Exec('sc.exe', 'query WireGuardManager', '', SW_HIDE, ewWaitUntilTerminated, ResultCode)
    and (ResultCode = 0);
end;

function PrepareToInstall(var NeedsRestart: Boolean): String;
var
  DownloadPage: TDownloadWizardPage;
  ResultCode: Integer;
  MsiPath: String;
begin
  Result := '';
  NeedsRestart := False;

  if IsWireGuardInstalled() then
  begin
    Log('WireGuard is already installed');
    Exit;
  end;

  Log('WireGuard is missing. Downloading dependency');

  DownloadPage := CreateDownloadPage(
    'Installing dependencies',
    'Downloading WireGuard...',
    nil
  );

  try
    DownloadPage.Clear;
    DownloadPage.Add(
      WireGuardUrl,
      WireGuardFile,
      ''
    );

    DownloadPage.Show;
    try
      DownloadPage.Download;
    finally
      DownloadPage.Hide;
    end;

    MsiPath := ExpandConstant('{tmp}\') + WireGuardFile;

    Log('Installing WireGuard');

    if not Exec(
      ExpandConstant('{sys}\msiexec.exe'),
      '/i "' + MsiPath + '" /qn /norestart',
      '',
      SW_HIDE,
      ewWaitUntilTerminated,
      ResultCode
    ) then
    begin
      Result := 'Could not start WireGuard installer';
      Exit;
    end;

    if ResultCode = 3010 then
    begin
      NeedsRestart := True;
      Result := 'WireGuard requires a system restart. ' +
        'Restart Windows and run Wiredoor Setup again.';
      Exit;
    end;

    if ResultCode <> 0 then
    begin
      Result := 'WireGuard installation failed: ' +
        IntToStr(ResultCode);
      Exit;
    end;

    if not IsWireGuardInstalled() then
    begin
      Result := 'WireGuard installation could not be verified';
      Exit;
    end;

    Log('WireGuard installed successfully');

  except
    Result := 'Failed to download or install WireGuard: ' +
      GetExceptionMessage;
  end;
end;
