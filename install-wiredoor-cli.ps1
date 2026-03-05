#requires -Version 5.1
$ErrorActionPreference = 'Stop'

$ExeName = 'wiredoor.exe'
$ServiceName = 'wiredoorService'
$WgVERSION = '0.5.3'

$ApiUrl = "https://api.github.com/repos/wiredoor/wiredoor-cli/releases/latest"
$InstallDir = Join-Path $env:LOCALAPPDATA 'Wiredoor\bin'
$TempDir = Join-Path $env:TEMP 'Wiredoor-Install'

if (-not [Environment]::Is64BitOperatingSystem) {
    throw "Unsupported architecture: x86 (32-bit OS)."
}

$Arch = 'amd64'

$ReleaseInfo = Invoke-RestMethod -Uri $ApiUrl -UseBasicParsing
$VERSION = ($ReleaseInfo.tag_name -replace '^v','')

$ReleaseBaseUrl = 'https://github.com/wiredoor/wiredoor-cli/releases/download/v' + $VERSION

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

function Test-IsAdministrator {
    $id = [Security.Principal.WindowsIdentity]::GetCurrent()
    $p  = New-Object Security.Principal.WindowsPrincipal($id)
    return $p.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
}

function Invoke-AdminElevation {
    param([string[]]$PassThruArgs = @())

    if (Test-IsAdministrator) { return }

    Write-Host "[wiredoor] Elevation required. Requesting admin privileges..."

    $scriptPath = $PSCommandPath
    if ([string]::IsNullOrWhiteSpace($scriptPath)) {
        $scriptPath = Join-Path $env:TEMP 'wiredoor-install.ps1'

        try {
            [Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12
            Invoke-WebRequest -Uri "https://raw.githubusercontent.com/wiredoor/wiredoor-cli/main/install-wiredoor-cli.ps1" -UseBasicParsing -OutFile $scriptPath
        } catch {
            throw "[wiredoor] ERROR: Cannot self-elevate when run via iwr|iex because the script cannot be persisted to disk: $($_.Exception.Message)"
        }
    }

    $joinedArgs = if ($PassThruArgs -and $PassThruArgs.Count -gt 0) { ($PassThruArgs -join ' ') } else { '' }

    $cmd = @"
& '$scriptPath' $joinedArgs
`$ec = `$LASTEXITCODE
if (`$ec -ne 0) {
    Write-Host '[wiredoor] Install failed (exit code:' `$ec ')' -ForegroundColor Red
} else {
    Write-Host '[wiredoor] Done!' -ForegroundColor Green
}
rm -LiteralPath '$scriptPath' -ErrorAction SilentlyContinue
Write-Host 'Press Enter to close...'
[void][Console]::ReadLine()
exit `$ec
"@

    $argList = @(
        '-NoProfile',
        '-ExecutionPolicy', 'Bypass',
        '-Command', $cmd
    )

    try {
        Start-Process -FilePath 'powershell.exe' -Verb RunAs -ArgumentList $argList | Out-Null
    } catch {
        throw "[wiredoor] ERROR: Elevation was cancelled or failed: $($_.Exception.Message)"
    }

    exit 0
}

Invoke-AdminElevation

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
