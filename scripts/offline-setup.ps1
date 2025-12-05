<#
.SYNOPSIS
    Prepare an offline-ready environment for the RoundOneledger application on Windows.
.DESCRIPTION
    Detects Go, Node.js, npm, PostgreSQL, and Docker Desktop, installs them from the
    offline artifact directory when missing, restores the vendor and node_modules caches,
    optionally initializes the database, and starts both backend and frontend services.

    The script expects the offline artifacts collected on an internet-connected
    workstation (see README for the gathering procedure) to be copied into the
    repository under the default .\offline-artifacts directory. Installers can
    live anywhere under the offline root (for example .\offline-artifacts\installers).
.PARAMETER OfflineRoot
    Overrides the default offline artifact directory (..\offline-artifacts relative
    to the script location).
.PARAMETER UseDocker
    Load pre-saved Docker images and bootstrap services via docker compose instead of
    using a local PostgreSQL instance.
.PARAMETER InitializeDatabase
    Perform database initialization after tooling has been installed.
.PARAMETER FrontendOnly
    Skip backend execution and only serve the frontend build output.
.PARAMETER BackendOnly
    Skip frontend serving and only run the backend executable.
#>
[CmdletBinding(SupportsShouldProcess=$true)]
param(
    [string]$OfflineRoot,
    [switch]$UseDocker,
    [switch]$InitializeDatabase,
    [switch]$FrontendOnly,
    [switch]$BackendOnly
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

function Write-Info([string]$Message) {
    Write-Host "[INFO] $Message" -ForegroundColor Cyan
}

function Write-Warn([string]$Message) {
    Write-Warning "[WARN] $Message"
}

$scriptDir = Split-Path -Path $MyInvocation.MyCommand.Path -Parent
$repoRoot = Resolve-Path (Join-Path $scriptDir '..')
if (-not $OfflineRoot) {
    $OfflineRoot = Join-Path $repoRoot 'offline-artifacts'
}

function Test-CommandAvailable([string]$CommandName) {
    return [bool](Get-Command $CommandName -ErrorAction SilentlyContinue)
}

function Invoke-Installer {
    param(
        [string]$Name,
        [string]$Path,
        [string]$Arguments
    )

    if (-not (Test-Path $Path)) {
        Write-Warn "Installer for $Name not found at $Path"
        return $false
    }

    Write-Info "Installing $Name from $Path"
    $psi = New-Object System.Diagnostics.ProcessStartInfo
    $psi.FileName = $Path
    $psi.Arguments = $Arguments
    $psi.UseShellExecute = $false
    $psi.RedirectStandardError = $true
    $psi.RedirectStandardOutput = $true
    $process = [System.Diagnostics.Process]::Start($psi)
    $process.WaitForExit()

    if ($process.ExitCode -ne 0) {
        Write-Warn "$Name installer exited with code $($process.ExitCode)"
        return $false
    }

    return $true
}

function Ensure-Go {
    if (Test-CommandAvailable 'go') {
        Write-Info 'Go is already installed.'
        return
    }

    $installer = Get-ChildItem -Path $OfflineRoot -Filter 'go1.22*.msi' -File -Recurse -ErrorAction SilentlyContinue | Select-Object -First 1
    if (-not $installer) {
        Write-Warn 'Go installer not found. Please install Go 1.22 manually.'
        return
    }

    Invoke-Installer -Name 'Go' -Path $installer.FullName -Arguments '/quiet /qn'
}

function Ensure-Node {
    if (Test-CommandAvailable 'node') {
        Write-Info 'Node.js is already installed.'
        return
    }

    $installer = Get-ChildItem -Path $OfflineRoot -Filter 'node-v18*.msi' -File -Recurse -ErrorAction SilentlyContinue | Select-Object -First 1
    if (-not $installer) {
        Write-Warn 'Node.js installer not found. Please install Node.js 18 manually.'
        return
    }

    Invoke-Installer -Name 'Node.js' -Path $installer.FullName -Arguments '/quiet /qn'
}

function Ensure-PostgreSQL {
    if (Test-CommandAvailable 'psql') {
        Write-Info 'PostgreSQL client/server already available.'
        return
    }

    $installer = Get-ChildItem -Path $OfflineRoot -Filter 'postgresql-*.exe' -File -Recurse -ErrorAction SilentlyContinue | Select-Object -First 1
    if (-not $installer) {
        Write-Warn 'PostgreSQL installer not found. Please install PostgreSQL 15 manually.'
        return
    }

    $arguments = '--mode unattended --unattendedmodeui minimal --superpassword postgres --servicename postgresql-15'
    Invoke-Installer -Name 'PostgreSQL' -Path $installer.FullName -Arguments $arguments
}

function Ensure-Docker {
    if (Test-CommandAvailable 'docker') {
        Write-Info 'Docker Desktop is already installed.'
        return
    }

    $installer = Get-ChildItem -Path $OfflineRoot -Filter 'DockerDesktopInstaller.exe' -File -Recurse -ErrorAction SilentlyContinue | Select-Object -First 1
    if (-not $installer) {
        Write-Warn 'Docker Desktop installer not found. Please install Docker Desktop manually.'
        return
    }

    Invoke-Installer -Name 'Docker Desktop' -Path $installer.FullName -Arguments 'install --quiet'
}

function Configure-GoEnvironment {
    $goPath = Join-Path $repoRoot '.gopath'
    if (-not (Test-Path $goPath)) {
        New-Item -Path $goPath -ItemType Directory | Out-Null
    }

    [Environment]::SetEnvironmentVariable('GOPATH', $goPath, 'Process')
    [Environment]::SetEnvironmentVariable('GOPATH', $goPath, 'User')
    Write-Info "GOPATH set to $goPath"
}

function Configure-NpmCache {
    $npmCacheSource = Join-Path $OfflineRoot 'npm-cache'
    if (-not (Test-Path $npmCacheSource)) {
        Write-Warn "npm cache not found at $npmCacheSource"
        return
    }

    $npmCacheTarget = Join-Path $env:LOCALAPPDATA 'npm-cache'
    if (-not (Test-Path $npmCacheTarget)) {
        New-Item -Path $npmCacheTarget -ItemType Directory -Force | Out-Null
    }

    Write-Info "Restoring npm cache to $npmCacheTarget"
    robocopy $npmCacheSource $npmCacheTarget /MIR | Out-Null
    npm config set cache $npmCacheTarget --location user | Out-Null
}

function Restore-Directory {
    param(
        [string]$Source,
        [string]$Target
    )

    if (-not (Test-Path $Source)) {
        Write-Warn "Source $Source not found; skipping restore."
        return
    }

    if (Test-Path $Target) {
        Write-Info "Clearing existing $Target"
        Remove-Item -Path $Target -Recurse -Force
    }

    $extension = [IO.Path]::GetExtension($Source)
    if ($extension -eq '.zip') {
        Write-Info "Expanding archive $Source to $Target"
        Expand-Archive -Path $Source -DestinationPath $Target
    } else {
        Write-Info "Copying $Source to $Target"
        robocopy $Source $Target /MIR | Out-Null
    }
}

function Restore-Artifacts {
    Restore-Directory -Source (Join-Path $OfflineRoot 'vendor') -Target (Join-Path $repoRoot 'vendor')
    Restore-Directory -Source (Join-Path $OfflineRoot 'node_modules') -Target (Join-Path $repoRoot 'web\node_modules')
    Restore-Directory -Source (Join-Path $OfflineRoot 'build') -Target (Join-Path $repoRoot 'web\build')
    $packageLockSource = Join-Path $OfflineRoot 'web-package-lock.json'
    if (Test-Path $packageLockSource) {
        Copy-Item -Path $packageLockSource -Destination (Join-Path $repoRoot 'web\package-lock.json') -Force
        Write-Info 'Restored web/package-lock.json from offline artifacts.'
    }
    $backendBinSource = Join-Path $OfflineRoot 'server.exe'
    if (Test-Path $backendBinSource) {
        $distDir = Join-Path $repoRoot 'dist'
        if (-not (Test-Path $distDir)) {
            New-Item -ItemType Directory -Path $distDir | Out-Null
        }
        Copy-Item -Path $backendBinSource -Destination (Join-Path $distDir 'server.exe') -Force
        Write-Info 'Restored backend executable to dist/server.exe'
    } else {
        Write-Warn 'Backend executable not found in offline artifacts.'
    }
}

function Load-DockerImages {
    $imagesDir = Join-Path $OfflineRoot 'docker-images'
    if (-not (Test-Path $imagesDir)) {
        Write-Warn 'Docker image archives directory not found; skipping load.'
        return
    }

    Get-ChildItem -Path $imagesDir -Filter '*.tar' | ForEach-Object {
        Write-Info "Loading Docker image $($_.Name)"
        docker load -i $_.FullName | Write-Host
    }
}

function Initialize-DatabaseWithPsql {
    $psql = Get-Command 'psql.exe' -ErrorAction SilentlyContinue
    if (-not $psql) {
        Write-Warn 'psql executable not found; cannot initialize database.'
        return
    }

    $envFile = Join-Path $repoRoot '.env'
    if (Test-Path $envFile) {
        Get-Content $envFile | ForEach-Object {
            if ($_ -match '^(?<key>[^#=]+)=(?<value>.+)$') {
                $key = $Matches['key']
                $value = $Matches['value']
                [Environment]::SetEnvironmentVariable($key, $value, 'Process')
            }
        }
    }

    $migrationDir = Join-Path $repoRoot 'migrations'
    if (-not (Test-Path $migrationDir)) {
        Write-Warn 'Migrations directory not found; skipping psql initialization.'
        return
    }

    Write-Info 'Applying SQL migrations using psql.'
    Get-ChildItem -Path $migrationDir -Filter '*.sql' | Sort-Object Name | ForEach-Object {
        Write-Info "Running migration $($_.Name)"
        & $psql.Source -f $_.FullName
    }
}

function Initialize-DatabaseWithDocker {
    Load-DockerImages
    Write-Info 'Starting services via docker compose.'
    docker compose -f (Join-Path $repoRoot 'docker-compose.yml') up -d
}

function Start-Backend {
    $backendPath = Join-Path $repoRoot 'dist/server.exe'
    if (-not (Test-Path $backendPath)) {
        Write-Warn 'Backend executable not found at dist/server.exe'
        return
    }

    Write-Info 'Starting backend server.'
    Start-Process -FilePath $backendPath -WorkingDirectory $repoRoot
}

function Start-Frontend {
    $buildDir = Join-Path $repoRoot 'web/build'
    if (-not (Test-Path $buildDir)) {
        Write-Warn "Frontend build not found at $buildDir"
        return
    }

    Write-Info 'Serving frontend build via npx serve.'
    $serveCmd = Get-Command 'npx.cmd' -ErrorAction SilentlyContinue
    if (-not $serveCmd) {
        Write-Warn 'npx command not found; ensure Node.js/npm are installed.'
        return
    }

    Start-Process -FilePath $serveCmd.Source -ArgumentList @('serve', 'build', '--listen', '0.0.0.0:3000') -WorkingDirectory (Join-Path $repoRoot 'web')
}

Write-Info "Using offline artifacts at $OfflineRoot"
Ensure-Go
Ensure-Node
Ensure-PostgreSQL
Ensure-Docker
Configure-GoEnvironment
Configure-NpmCache
Restore-Artifacts

if ($InitializeDatabase) {
    if ($UseDocker) {
        Initialize-DatabaseWithDocker
    } else {
        Initialize-DatabaseWithPsql
    }
}

if (-not $FrontendOnly) {
    Start-Backend
}

if (-not $BackendOnly) {
    Start-Frontend
}

Write-Info 'Offline setup script complete.'
