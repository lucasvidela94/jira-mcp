#!/usr/bin/env pwsh
#Requires -Version 5.1
<#
.SYNOPSIS
    Install jira-mcp from GitHub releases.

.DESCRIPTION
    Downloads the latest (or specified) release of jira-mcp, verifies its
checksum, and installs it to %LOCALAPPDATA%\jira-mcp\bin by default.

.PARAMETER Version
    Version to install, e.g. "v1.2.3". Defaults to the latest release.

.PARAMETER InstallDir
    Directory to install the binary into. Defaults to
    "$env:LOCALAPPDATA\jira-mcp\bin".

.PARAMETER SkipChecksum
    Skip checksum verification.
#>

[CmdletBinding()]
param(
    [string]$Version = "",
    [string]$InstallDir = "$env:LOCALAPPDATA\jira-mcp\bin",
    [switch]$SkipChecksum
)

$Repo = "lucasvidela94/jira-mcp"
$ErrorActionPreference = "Stop"

function Write-Info {
    param([string]$Message)
    Write-Host "info: $Message" -ForegroundColor Cyan
}

function Write-Success {
    param([string]$Message)
    Write-Host "success: $Message" -ForegroundColor Green
}

function Write-Warning {
    param([string]$Message)
    Write-Host "warning: $Message" -ForegroundColor Yellow
}

function Write-ErrorAndExit {
    param([string]$Message)
    Write-Host "error: $Message" -ForegroundColor Red
    exit 1
}

function Get-Architecture {
    switch ($env:PROCESSOR_ARCHITECTURE) {
        "AMD64" { return "amd64" }
        "ARM64" { return "arm64" }
        default { Write-ErrorAndExit "unsupported architecture: $($env:PROCESSOR_ARCHITECTURE)" }
    }
}

function Resolve-Version {
    if ($Version) {
        return $Version
    }

    Write-Info "resolving latest version..."
    try {
        $release = Invoke-RestMethod -Uri "https://api.github.com/repos/$Repo/releases/latest" -UseBasicParsing
        return $release.tag_name
    }
    catch {
        Write-ErrorAndExit "could not determine latest version from GitHub: $_"
    }
}

function Add-ToPath {
    param([string]$Directory)

    $userPath = [Environment]::GetEnvironmentVariable("Path", "User")
    $parts = $userPath -split ";" | Where-Object { $_ -and ($_ -ne $Directory) }
    $newPath = ($parts + $Directory) -join ";"
    [Environment]::SetEnvironmentVariable("Path", $newPath, "User")

    $currentPath = $env:Path -split ";" | Where-Object { $_ -and ($_ -ne $Directory) }
    $env:Path = ($currentPath + $Directory) -join ";"
}

$os = "windows"
$arch = Get-Architecture
$version = Resolve-Version
$versionNoV = $version -replace "^v", ""
$archive = "jira-mcp_${versionNoV}_${os}_${arch}.zip"
$binary = "jira-mcp.exe"

$baseUrl = "https://github.com/$Repo/releases/download/$version"
$archiveUrl = "$baseUrl/$archive"
$checksumUrl = "$baseUrl/checksums.txt"

Write-Info "installing jira-mcp $version for $os/$arch..."

$tmpDir = Join-Path $env:TEMP ([System.Guid]::NewGuid().ToString())
New-Item -ItemType Directory -Path $tmpDir | Out-Null

try {
    $archivePath = Join-Path $tmpDir $archive
    try {
        Invoke-WebRequest -Uri $archiveUrl -OutFile $archivePath -UseBasicParsing
    }
    catch {
        Write-ErrorAndExit "could not download $archiveUrl : $_"
    }

    if (-not $SkipChecksum) {
        Write-Info "verifying checksum..."
        try {
            $checksums = Invoke-WebRequest -Uri $checksumUrl -UseBasicParsing | Select-Object -ExpandProperty Content
        }
        catch {
            Write-ErrorAndExit "could not download checksums.txt: $_"
        }

        $expected = ($checksums -split "`r?`n" | Where-Object { $_ -match $archive }) | ForEach-Object { ($_ -split "\s+")[0] } | Select-Object -First 1
        if (-not $expected) {
            Write-ErrorAndExit "checksum not found in checksums.txt"
        }

        $actual = (Get-FileHash -Path $archivePath -Algorithm SHA256).Hash.ToLower()
        if ($actual -ne $expected) {
            Write-ErrorAndExit "checksum mismatch for $archive`n  expected: $expected`n  actual:   $actual"
        }
        Write-Info "checksum verified"
    }
    else {
        Write-Warning "skipping checksum verification (-SkipChecksum)"
    }

    Expand-Archive -Path $archivePath -DestinationPath $tmpDir -Force

    $binarySource = Join-Path $tmpDir $binary
    if (-not (Test-Path $binarySource)) {
        Write-ErrorAndExit "extracted archive does not contain $binary"
    }

    if (-not (Test-Path $InstallDir)) {
        New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
    }

    $binaryDest = Join-Path $InstallDir $binary
    Copy-Item -Path $binarySource -Destination $binaryDest -Force

    Write-Success "installed $binary to $binaryDest"

    $pathDirs = $env:Path -split ";"
    if ($pathDirs -notcontains $InstallDir) {
        Write-Info "adding $InstallDir to user PATH..."
        Add-ToPath -Directory $InstallDir
        Write-Info "added $InstallDir to user PATH"
    }

    Write-Success "jira-mcp is installed. Run 'jira-mcp --version' to verify."
}
finally {
    Remove-Item -Path $tmpDir -Recurse -Force -ErrorAction SilentlyContinue
}
