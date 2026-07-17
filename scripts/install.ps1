#!/usr/bin/env pwsh
#Requires -Version 5.1
<#
.SYNOPSIS
    Install jira-mcp from GitHub releases.

.DESCRIPTION
    Downloads the latest (or specified) release of jira-mcp, verifies its
    checksum, installs it to %LOCALAPPDATA%\jira-mcp\bin by default, and
    optionally writes an MCP client configuration file.

.PARAMETER Version
    Version to install, e.g. "v1.2.3". Defaults to the latest release.

.PARAMETER InstallDir
    Directory to install the binary into. Defaults to
    "$env:LOCALAPPDATA\jira-mcp\bin".

.PARAMETER SkipChecksum
    Skip checksum verification.

.PARAMETER Configure
    Force the MCP client configuration wizard.

.PARAMETER NoConfigure
    Skip the MCP client configuration wizard.

.PARAMETER Client
    Pre-select the MCP client: opencode, claude, cursor, or windsurf.

.PARAMETER JiraUrl
    Jira Cloud base URL.

.PARAMETER JiraUsername
    Jira username/email.

.PARAMETER JiraApiToken
    Jira API token.

.PARAMETER Yes
    Skip confirmations.
#>

[CmdletBinding()]
param(
    [string]$Version = "",
    [string]$InstallDir = "$env:LOCALAPPDATA\jira-mcp\bin",
    [switch]$SkipChecksum,
    [switch]$Configure,
    [switch]$NoConfigure,
    [string]$Client = "",
    [string]$JiraUrl = "",
    [string]$JiraUsername = "",
    [string]$JiraApiToken = "",
    [switch]$Yes
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

function Test-ValidClient {
    param([string]$Value)
    return $Value -in @('opencode', 'claude', 'cursor', 'windsurf')
}

function Get-ConfigPath {
    param([string]$Client)
    $configHome = if ($env:XDG_CONFIG_HOME) { $env:XDG_CONFIG_HOME } else { Join-Path $env:USERPROFILE '.config' }
    switch ($Client) {
        'opencode' { return Join-Path $configHome 'opencode\opencode.json' }
        'claude' { return Join-Path $configHome 'claude\claude_desktop_config.json' }
        'cursor' { return Join-Path $env:USERPROFILE '.cursor\mcp.json' }
        'windsurf' { return Join-Path $configHome 'windsurf\mcp_config.json' }
    }
}

function Read-SecureInput {
    param([string]$Prompt)
    $secure = Read-Host -Prompt $Prompt -AsSecureString
    return [System.Runtime.InteropServices.Marshal]::PtrToStringAuto(
        [System.Runtime.InteropServices.Marshal]::SecureStringToBSTR($secure)
    )
}

function Prompt-Client {
    if ($Client) { return }
    $valid = $false
    while (-not $valid) {
        $input = Read-Host "Which MCP client do you want to configure? [opencode/claude/cursor/windsurf/none] (default: opencode)"
        if (-not $input) { $input = 'opencode' }
        if ($input -eq 'none' -or $input -eq 'skip') {
            $script:Client = 'none'
            return
        }
        if (Test-ValidClient -Value $input) {
            $script:Client = $input
            $valid = $true
        }
        else {
            Write-Host "error: invalid client: $input" -ForegroundColor Red
        }
    }
}

function Prompt-JiraUrl {
    if ($JiraUrl) { return }
    $valid = $false
    while (-not $valid) {
        $input = Read-Host "Jira URL (e.g. https://yourcompany.atlassian.net)"
        if ($input -match '^https?://') {
            $script:JiraUrl = $input
            $valid = $true
        }
        else {
            Write-Host "error: Jira URL must start with http:// or https://" -ForegroundColor Red
        }
    }
}

function Prompt-JiraUsername {
    if ($JiraUsername) { return }
    $valid = $false
    while (-not $valid) {
        $input = Read-Host "Jira username/email"
        if ($input -match '@') {
            $script:JiraUsername = $input
            $valid = $true
        }
        else {
            Write-Host "error: Username must contain '@'" -ForegroundColor Red
        }
    }
}

function Prompt-JiraApiToken {
    if ($JiraApiToken) { return }
    $valid = $false
    while (-not $valid) {
        $input = Read-SecureInput -Prompt "Jira API token"
        if ($input) {
            $script:JiraApiToken = $input
            $valid = $true
        }
        else {
            Write-Host "error: API token is required" -ForegroundColor Red
        }
    }
}

function Confirm-WriteConfig {
    param([string]$ConfigPath)
    if ($Yes) { return $true }
    Write-Host ""
    Write-Host "Will write jira-mcp configuration for $Client at $ConfigPath"
    Write-Host "Jira URL: $JiraUrl"
    Write-Host "Username: $JiraUsername"
    Write-Host "API token: <hidden>"
    $input = Read-Host "Proceed? [Y/n]"
    return $input -notmatch '^(n|no)$'
}

function Write-ClientConfig {
    param(
        [string]$Client,
        [string]$Path,
        [string]$Url,
        [string]$Username,
        [string]$Token
    )

    $dir = Split-Path -Path $Path -Parent
    if (-not (Test-Path $dir)) {
        New-Item -ItemType Directory -Path $dir -Force | Out-Null
    }

    if ($Client -eq 'opencode') {
        $newEntry = [PSCustomObject]@{
            mcp = [PSCustomObject]@{
                jira = [PSCustomObject]@{
                    type = 'local'
                    command = @('jira-mcp')
                    environment = [PSCustomObject]@{
                        JIRA_URL = $Url
                        JIRA_USERNAME = $Username
                        JIRA_API_TOKEN = $Token
                    }
                }
            }
        }
    }
    else {
        $newEntry = [PSCustomObject]@{
            mcpServers = [PSCustomObject]@{
                jira = [PSCustomObject]@{
                    command = 'jira-mcp'
                    env = [PSCustomObject]@{
                        JIRA_URL = $Url
                        JIRA_USERNAME = $Username
                        JIRA_API_TOKEN = $Token
                    }
                }
            }
        }
    }

    $fileExisted = Test-Path $Path
    $data = $null
    if ($fileExisted) {
        try {
            $content = Get-Content -Path $Path -Raw -ErrorAction Stop
            $data = $content | ConvertFrom-Json -ErrorAction Stop
        }
        catch {
            Write-Host "error: could not parse existing config at ${Path}: $_" -ForegroundColor Red
            Write-Host "Please fix the existing config manually. The new jira entry would be:" -ForegroundColor Red
            $newEntry | ConvertTo-Json -Depth 10
            exit 1
        }
    }

    if (-not $data) {
        $data = [PSCustomObject]@{}
    }

    foreach ($key in $newEntry.PSObject.Properties.Name) {
        $value = $newEntry.$key
        $existing = $data.PSObject.Properties[$key]
        if ($existing -and ($existing.Value -is [PSCustomObject]) -and ($value -is [PSCustomObject])) {
            foreach ($subKey in $value.PSObject.Properties.Name) {
                $existing.Value | Add-Member -NotePropertyName $subKey -NotePropertyValue $value.$subKey -Force
            }
        }
        else {
            $data | Add-Member -NotePropertyName $key -NotePropertyValue $value -Force
        }
    }

    if ($fileExisted) {
        Copy-Item -Path $Path -Destination "${Path}.bak" -Force
    }

    $data | ConvertTo-Json -Depth 10 | Set-Content -Path $Path
}

function Test-Interactive {
    try {
        return -not [System.Console]::IsInputRedirected
    }
    catch {
        return $false
    }
}

function Configure-Client {
    if ($NoConfigure) { return }

    if ($Client -and ($Client -eq 'none' -or $Client -eq 'skip')) { return }

    $allProvided = [bool]($Client -and $JiraUrl -and $JiraUsername -and $JiraApiToken)

    # Without interactive input we cannot prompt; proceed only if every required
    # value was supplied up front or if the user explicitly forced configuration.
    if (-not $Configure -and -not (Test-Interactive) -and -not $allProvided) { return }

    if (-not $Client) {
        if (-not (Test-Interactive)) { return }
        Prompt-Client
    }

    if ($Client -eq 'none' -or $Client -eq 'skip') { return }

    if (-not (Test-ValidClient -Value $Client)) {
        Write-Host "error: invalid client: $Client" -ForegroundColor Red
        return
    }

    if (-not $JiraUrl -and -not (Test-Interactive)) { return }
    Prompt-JiraUrl

    if (-not $JiraUsername -and -not (Test-Interactive)) { return }
    Prompt-JiraUsername

    if (-not $JiraApiToken -and -not (Test-Interactive)) { return }
    Prompt-JiraApiToken

    $configPath = Get-ConfigPath -Client $Client

    # If every value was provided non-interactively, treat that as confirmation
    # because we cannot prompt; otherwise respect -Yes or ask.
    if (-not $allProvided -and -not (Confirm-WriteConfig -ConfigPath $configPath)) {
        Write-Info "configuration skipped"
        return
    }

    Write-ClientConfig -Client $Client -Path $configPath -Url $JiraUrl -Username $JiraUsername -Token $JiraApiToken

    Write-Success "Configured jira-mcp for $Client at $configPath"
    if (Test-Path "${configPath}.bak") {
        Write-Info "Backed up previous config to ${configPath}.bak"
    }
    Write-Info "You can verify with: jira-mcp --version"
    Write-Info "Create or verify API tokens: https://id.atlassian.com/manage-profile/security/api-tokens"
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

    Configure-Client

    Write-Success "jira-mcp is installed. Run 'jira-mcp --version' to verify."
}
finally {
    Remove-Item -Path $tmpDir -Recurse -Force -ErrorAction SilentlyContinue
}
