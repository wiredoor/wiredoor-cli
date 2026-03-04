#requires -Version 5.1
$ErrorActionPreference = 'Stop'

$ExeName = 'wiredoor.exe'
$ServiceName = 'wiredoorService'
$VERSION = '1.0.0'
$WgVERSION = '0.5.3'

$InstallDir = Join-Path $env:LOCALAPPDATA 'Wiredoor\bin'
$TempDir = Join-Path $env:TEMP 'Wiredoor-Install'
$Arch = if ([Environment]::Is64BitOperatingSystem) { 'amd64' } else { '' }

# Job artifacts base URL
$ReleaseBaseUrl = 'https://github.com/wiredoor/wiredoor-cli/releases/download/latest'

$FileName = "wiredoor_${VERSION}_windows_${Arch}.exe"
$DownloadUrl = "$ReleaseBaseUrl/$FileName"

# -----------------------------
# Helper functions
# -----------------------------
function Write-Info {
    param([Parameter(Mandatory)][string]$Msg)
    Write-Host "[wiredoor] $Msg"
}

function Fail {
    param([Parameter(Mandatory)][string]$Msg)
    throw "[wiredoor] ERROR: $Msg"
}

function New-DirectoryIfNotExists {
    param([Parameter(Mandatory)][string]$Path)
    if (-not (Test-Path -LiteralPath $Path)) {
        New-Item -ItemType Directory -Path $Path | Out-Null
    }
}

function Add-ToUserPath {
    param([Parameter(Mandatory)][string]$Dir)

    $userPath = [Environment]::GetEnvironmentVariable('Path', 'User')
    if ($userPath -and ($userPath.Split(';') -contains $Dir)) { return }

    $newPath = if ([string]::IsNullOrEmpty($userPath)) { $Dir } else { "$userPath;$Dir" }
    [Environment]::SetEnvironmentVariable('Path', $newPath, 'User')
    $env:Path = "$env:Path;$Dir"
}

function Test-Command {
    param([Parameter(Mandatory)][string]$Name)
    return $null -ne (Get-Command $Name -ErrorAction SilentlyContinue)
}

# -----------------------------
# Preflight
# -----------------------------
Write-Info "Installing Wiredoor CLI v$VERSION..."

if (-not (Test-Command 'Invoke-WebRequest')) {
    Fail 'Invoke-WebRequest is not available.'
}

# Recreate temp
if (Test-Path -LiteralPath $TempDir) {
    Remove-Item -Recurse -Force -LiteralPath $TempDir -ErrorAction SilentlyContinue
}

New-DirectoryIfNotExists $TempDir
New-DirectoryIfNotExists $InstallDir

# -----------------------------
# Download
# -----------------------------
$OutPath = Join-Path $TempDir ([IO.Path]::GetFileName($DownloadUrl))
Write-Info "Downloading: $DownloadUrl -> $OutPath"

Invoke-WebRequest -Uri $DownloadUrl -OutFile $OutPath -UseBasicParsing

if (-not (Test-Path -LiteralPath $OutPath)) {
    Fail "Download failed (file not found): $OutPath"
}

# -----------------------------
# If ZIP -> extract, else use file directly
# -----------------------------
$ext = [IO.Path]::GetExtension($OutPath).ToLowerInvariant()
$ExeCandidatePath = $null

if ($ext -eq '.zip') {
    Write-Info 'Extracting ZIP...'
    Expand-Archive -Path $OutPath -DestinationPath $TempDir -Force

    $candidate = Get-ChildItem -Path $TempDir -Recurse -File -Filter $ExeName | Select-Object -First 1
    if (-not $candidate) { Fail "Could not find $ExeName inside the ZIP." }

    $ExeCandidatePath = $candidate.FullName
}
elseif ($ext -eq '.exe') {
    Write-Info 'Downloaded EXE.'
    $ExeCandidatePath = $OutPath
}
else {
    Fail "Unsupported downloaded file type: $ext (expected .exe or .zip)"
}

# -----------------------------
# Check WireGuard
# -----------------------------
function Test-WireGuardInstalled {
    $keys = @(
        'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\*',
        'HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall\*'
    )

    foreach ($k in $keys) {
        $app = Get-ItemProperty -Path $k -ErrorAction SilentlyContinue |
            Where-Object { $_.DisplayName -eq 'WireGuard' } |
            Select-Object -First 1
        if ($app) { return $true }
    }
    return $false
}
if (-not (Test-WireGuardInstalled)) {
    $WgArch = if ($Arch -eq 'amd64') { 'amd64' } else { 'x86' }
    $WgDownloadUrl = "https://download.wireguard.com/windows-client/wireguard-$WgArch-$WgVERSION.msi"
    $WgInstallerPath = Join-Path $TempDir "wireguard-$WgArch-$WgVERSION.msi"

    Write-Info "Downloading: $WgDownloadUrl -> $WgInstallerPath"
    Invoke-WebRequest -Uri $WgDownloadUrl -OutFile $WgInstallerPath -UseBasicParsing

    if (-not (Test-Path -LiteralPath $WgInstallerPath)) {
        Fail "Download failed (file not found): $WgInstallerPath"
    }

    Write-Info 'Installing WireGuard...'
    $proc = Start-Process -FilePath msiexec.exe -ArgumentList @(
        '/i', $WgInstallerPath,
        '/qn',
        '/norestart'
    ) -Wait -PassThru
    if ($proc.ExitCode -ne 0) {
        Fail "WireGuard MSI failed with exit code $($proc.ExitCode)"
    }
    Get-Process -Name 'WireGuard' -ErrorAction SilentlyContinue | Stop-Process -Force
    # Start-Sleep -Seconds 1
    # Get-Process -Name 'WireGuard' -ErrorAction SilentlyContinue | Stop-Process -Force
}

# -----------------------------
# Install
# -----------------------------
$TargetExe = Join-Path $InstallDir $ExeName

# Stop and delete service (if it exists)
$svc = Get-Service -Name $ServiceName -ErrorAction SilentlyContinue
if ($svc) {
    Write-Info "Stopping service: $ServiceName"
    Stop-Service -Name $ServiceName -Force -ErrorAction SilentlyContinue
    Write-Info "Deleting service: $ServiceName"
    & sc.exe delete $ServiceName | Out-Null
}

if (Test-Path -LiteralPath $TargetExe) {
    $backup = "$TargetExe.bak"
    Copy-Item -LiteralPath $TargetExe -Destination $backup -Force
    Write-Info "Backed up existing binary: $backup"
}

Copy-Item -LiteralPath $ExeCandidatePath -Destination $TargetExe -Force
Write-Info "Installed: $TargetExe"

& $TargetExe install

# -----------------------------
# PATH
# -----------------------------
Add-ToUserPath $InstallDir
Write-Info "Added to PATH (User): $InstallDir"

# -----------------------------
# Cleanup
# -----------------------------
if (Test-Path -LiteralPath $TempDir) {
    Remove-Item -Recurse -Force -LiteralPath $TempDir -ErrorAction SilentlyContinue
}

# -----------------------------
# Verify
# -----------------------------
Write-Info 'Verifying install...'
& $TargetExe --version 2>$null | Out-Null

Write-Info 'Done!'
Write-Info 'Try: wiredoor --help'
