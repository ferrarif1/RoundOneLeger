[CmdletBinding()]
param(
    [string]$Output = 'dist/ledger.exe',
    [string]$Arch = 'amd64',
    [switch]$SkipNpmInstall,
    [switch]$SkipFrontendBuild
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

$scriptDir = Split-Path -Path $MyInvocation.MyCommand.Path -Parent
$repoRoot = Resolve-Path (Join-Path $scriptDir '..')
Set-Location $repoRoot

function Invoke-Step {
    param(
        [string]$Label,
        [scriptblock]$Action
    )
    Write-Host "==> $Label" -ForegroundColor Cyan
    & $Action
}

if (-not $SkipNpmInstall) {
    Invoke-Step -Label 'Installing frontend dependencies' -Action {
        npm --prefix web install --no-fund --no-audit
    }
}

if (-not $SkipFrontendBuild) {
    Invoke-Step -Label 'Building frontend bundle' -Action {
        if (Test-Path 'web/dist') {
            Remove-Item -Path 'web/dist' -Recurse -Force
        }
        npm --prefix web run build
    }
}

$resolvedOutput = $Output
if (-not [System.IO.Path]::IsPathRooted($resolvedOutput)) {
    $resolvedOutput = Join-Path $repoRoot $resolvedOutput
}

$distDir = Split-Path -Path $resolvedOutput -Parent
if (-not (Test-Path $distDir)) {
    New-Item -Path $distDir -ItemType Directory | Out-Null
}

$env:CGO_ENABLED = '0'
$env:GOOS = 'windows'
$env:GOARCH = $Arch

Invoke-Step -Label "Compiling Go backend (embedded SPA)" -Action {
    go build -o $resolvedOutput ./cmd/server
}

Write-Host "All-in-one executable created at $resolvedOutput" -ForegroundColor Green
Write-Host 'Double-click the file or run it from PowerShell to launch the backend + embedded frontend (http://localhost:8080).' -ForegroundColor Green
