#requires -Version 5.1
$ErrorActionPreference = "Stop"

$AppName    = "wiredoor-cli"
$ExeName    = "wiredoor.exe"
$VERSION    = "1.0.0"
$WgVERSION  = "0.5.3"

$InstallDir = Join-Path $env:LOCALAPPDATA "Wiredoor\bin"
$TempDir    = Join-Path $env:TEMP "Wiredoor-Install"
$Arch       = if ([Environment]::Is64BitOperatingSystem) { "amd64" } else { "386" }

# Job artifacts base URL
# $ReleaseBaseUrl = "https://github.com/wiredoor/wiredoor-cli/releases/download/latest"
$ReleaseBaseUrl = "https://gitlab.infladoor.com/api/v4/projects/40/jobs/1438/artifacts/dist"

$FileName    = "wiredoor_${VERSION}_windows_${Arch}.exe"
$DownloadUrl = "$ReleaseBaseUrl/$FileName"
$GitLabToken = $env:GITLAB_TOKEN

# -----------------------------
# Helper functions
# -----------------------------
function Write-Info($msg) { Write-Host "[wiredoor] $msg" }
function Fail($msg) { throw "[wiredoor] ERROR: $msg" }

function Ensure-Dir($path) {
  if (!(Test-Path $path)) { New-Item -ItemType Directory -Path $path | Out-Null }
}

function Add-ToUserPath($dir) {
  $userPath = [Environment]::GetEnvironmentVariable("Path", "User")
  if ($userPath -and ($userPath.Split(";") -contains $dir)) { return }
  $newPath = if ([string]::IsNullOrEmpty($userPath)) { $dir } else { "$userPath;$dir" }
  [Environment]::SetEnvironmentVariable("Path", $newPath, "User")
  $env:Path = "$env:Path;$dir"
}

function Test-Command($name) {
  return $null -ne (Get-Command $name -ErrorAction SilentlyContinue)
}

# -----------------------------
# Preflight
# -----------------------------
Write-Info "📦 Installing Wiredoor CLI v$VERSION..."

if (!(Test-Command "Invoke-WebRequest")) {
  Fail "Invoke-WebRequest is not available."
}

# Recreate temp
if (Test-Path $TempDir) { Remove-Item -Recurse -Force $TempDir -ErrorAction SilentlyContinue }
Ensure-Dir $TempDir
Ensure-Dir $InstallDir

# -----------------------------
# Download
# -----------------------------
$OutPath = Join-Path $TempDir ([IO.Path]::GetFileName($DownloadUrl))
Write-Info "Downloading: $DownloadUrl"

$headers = @{}
$headers["PRIVATE-TOKEN"] = "glpat-7k6iWjehLe8pY_5MiyUU"
Write-Info "Using headers: $($headers)"
Write-Info "Downloading: $DownloadUrl to $OutPath"
Invoke-WebRequest -Uri $DownloadUrl -OutFile $OutPath -UseBasicParsing -Headers $headers

if (!(Test-Path $OutPath)) {
  Fail "Download failed (file not found): $OutPath"
}

# -----------------------------
# If ZIP -> extract, else use file directly
# -----------------------------
$ext = [IO.Path]::GetExtension($OutPath).ToLowerInvariant()
$ExeCandidatePath = $null

if ($ext -eq ".zip") {
  Write-Info "Extracting ZIP..."
  Expand-Archive -Path $OutPath -DestinationPath $TempDir -Force

  $candidate = Get-ChildItem -Path $TempDir -Recurse -File -Filter $ExeName | Select-Object -First 1
  if (!$candidate) { Fail "Could not find $ExeName inside the ZIP." }
  $ExeCandidatePath = $candidate.FullName
}
elseif ($ext -eq ".exe") {
  Write-Info "Downloaded EXE."
  $ExeCandidatePath = $OutPath
}
else {
  Fail "Unsupported downloaded file type: $ext (expected .exe or .zip)"
}

# -----------------------------
# Check WireGuard
# -----------------------------
if (!(Get-Command "wireguard")) {
  $WgDownloadUrl = "https://download.wireguard.com/windows-client/wireguard-$Arch-$WgVERSION.msi"
  $WgInstallerPath = Join-Path $TempDir "wireguard-$Arch-$WgVERSION.msi"
  Write-Info "Downloading: $WgDownloadUrl to $WgInstallerPath"
  Invoke-WebRequest -Uri `
    -Uri $WgDownloadUrl `
    -OutFile $WgInstallerPath
  if (!(Test-Path $WgInstallerPath)) {
    Fail "Download failed (file not found): $WgInstallerPath"
  }

  msiexec /i $WgInstallerPath /qn

  if (!(Get-Command "wireguard")) {
    Fail "WireGuard installation failed."
  } else {
    Write-Info "WireGuard installed successfully."
  }
}

# -----------------------------
# Install
# -----------------------------
$TargetExe = Join-Path $InstallDir $ExeName

if (Test-Path $TargetExe) {
  $backup = "$TargetExe.bak"
  Copy-Item $TargetExe $backup -Force
  Write-Info "Backed up existing binary to: $backup"
}

Copy-Item $ExeCandidatePath $TargetExe -Force
Write-Info "Installed: $TargetExe"

# -----------------------------
# PATH
# -----------------------------
Add-ToUserPath $InstallDir
Write-Info "Added to PATH (User): $InstallDir"

# -----------------------------
# Cleanup
# -----------------------------
if (Test-Path $TempDir) {
  Remove-Item -Recurse -Force $TempDir -ErrorAction SilentlyContinue
}

# -----------------------------
# Verify
# -----------------------------
Write-Info "Verifying install..."
& $TargetExe --version 2>$null | Out-Null

Write-Info "Done!"
Write-Info "Try: wiredoor --help"

# -----------------------------
# Install Service
# -----------------------------
#stop previous version
sc stop wiredoorService

#delete previous
# Stop-Service -Name $ServiceName -Force
sc delete wiredoorService

$TargetExe --install

#sc create wiredoorService binPath= "\"$TargetExe\" service --serviceInterval 10" start= auto obj= LocalSystem
#
sc start wiredoorService

# ================================
# Variables de configuración
# ================================
# $ServiceName   = "wiredoorService"
# $ExePath       = "$TargetExe"

# # ================================
# # remove previous service
# # ================================
# $svc = Get-Service -Name $ServiceName -ErrorAction SilentlyContinue
# if ($svc) {
    # if ($svc.Status -eq [System.ServiceProcess.ServiceControllerStatus]::Running) {
        # Stop-Service -Name $ServiceName -Force
    # }
    # sc.exe delete $ServiceName | Out-Null
    # Start-Sleep -Seconds 2
# }
# # ================================
# # install service as custom user
# # ================================

# $ExePath install

# # ================================
# # 7. Verify completion
# # ================================
# $svcFinal = Get-Service -Name $ServiceName -ErrorAction SilentlyContinue
# if ($svcFinal -and $svcFinal.Status -eq [System.ServiceProcess.ServiceControllerStatus]::Running) {
    # Write-Host "`n✅ Servicio $ServiceName instalado e iniciado correctamente bajo la cuenta $UserName."
    # Write-Host "🔐 Contraseña generada: $Password"
# } else {
    # Write-Host "`n❌ Error: el servicio $ServiceName no se inició correctamente."
# }

