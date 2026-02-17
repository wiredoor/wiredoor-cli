; To compile use: `& "C:\Users\dmesa\AppData\Local\Programs\Inno Setup 6\ISCC.exe" .\installer.iss`

[Setup]
AppName=Wiredoor CLI
AppVersion=1.0.0
DefaultDirName={localappdata}\Wiredoor
DefaultGroupName=Wiredoor
OutputBaseFilename=WiredoorSetup
Compression=lzma
SolidCompression=yes
PrivilegesRequired=admin
ChangesEnvironment=yes

[Dirs]
Name: "{localappdata}\Wiredoor\bin"

[Files]
; Empaqueta wiredoor.exe dentro del instalador
Source: "dist\wiredoor_1.0.0_windows_amd64.exe"; DestDir: "{localappdata}\Wiredoor\bin"; Flags: ignoreversion
; Opcional: empaqueta el MSI de WireGuard para instalar offline
Source: "dist\wireguard-amd64-0.5.3.msi"; DestDir: "{tmp}"; Flags: deleteafterinstall

[Registry]
; Añadir al PATH del usuario (HKCU). Si prefieres PATH del sistema, usa HKLM y requiere admin (ya lo tienes).
Root: HKCU; Subkey: "Environment"; ValueType: expandsz; ValueName: "Path"; \
  ValueData: "{olddata};{localappdata}\Wiredoor\bin"; Flags: preservestringtype

[Run]
; 1 Instalar WireGuard (si lo empaquetas). Si no lo empaquetas, puedes descargarlo antes o fallar si no está.
Filename: "msiexec.exe"; Parameters: "/i ""{tmp}\wireguard-amd64-0.5.3.msi"" /qn"; \
  StatusMsg: "Installing WireGuard..."; Flags: runhidden waituntilterminated; Check: not WireGuardInstalled

; 2 Parar y borrar servicio previo (ignorar errores si no existe)
Filename: "sc.exe"; Parameters: "stop wiredoorService"; Flags: runhidden; StatusMsg: "Stopping service..."; \
  Check: ServiceExists
Filename: "sc.exe"; Parameters: "delete wiredoorService"; Flags: runhidden; StatusMsg: "Removing old service..."; \
  Check: ServiceExists

; 3 Crear servicio nuevo
Filename: "sc.exe"; Parameters: "create wiredoorService binPath= """"{localappdata}\Wiredoor\bin\wiredoor.exe"""" service --serviceInterval 10 start= auto obj= LocalSystem"; \
  Flags: runhidden waituntilterminated; StatusMsg: "Creating service..."

; 4 Arrancar
Filename: "sc.exe"; Parameters: "start wiredoorService"; Flags: runhidden waituntilterminated; StatusMsg: "Starting service..."

[UninstallRun]
; Stop and delete service on uninstall
Filename: "sc.exe"; Parameters: "stop wiredoorService"; Flags: runhidden; RunOnceId: "StopWiredoorService"
Filename: "sc.exe"; Parameters: "delete wiredoorService"; Flags: runhidden; RunOnceId: "DeleteWiredoorService"

[Code]
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
  // Devuelve True si existe el servicio WireGuardManager (ajusta si en tu instalación cambia el nombre)
  Result :=
    Exec('sc.exe', 'query WireGuardManager', '', SW_HIDE, ewWaitUntilTerminated, ResultCode)
    and (ResultCode = 0);
end;