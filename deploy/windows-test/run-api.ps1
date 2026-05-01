param(
    [string]$Root = "C:\lms-arvand-backend",
    [string]$ExePath = "C:\lms-arvand-backend\bin\api.exe",
    [string]$EnvFile = ""
)

$ErrorActionPreference = "Stop"

if (-not (Test-Path $ExePath)) {
    throw "api binary not found: $ExePath"
}

$envCandidates = New-Object System.Collections.Generic.List[string]
if ([string]::IsNullOrWhiteSpace($EnvFile)) {
    $envCandidates.Add((Join-Path $Root ".env"))
    $envCandidates.Add((Join-Path (Split-Path $ExePath -Parent) ".env"))
    $envCandidates.Add((Join-Path (Split-Path (Split-Path $ExePath -Parent) -Parent) ".env"))
} else {
    $envCandidates.Add($EnvFile)
}
    
$resolvedEnv = $null
foreach ($candidate in $envCandidates) {
    if (Test-Path $candidate) {
        $resolvedEnv = (Resolve-Path $candidate).Path
        break
    }
}

if (-not $resolvedEnv) {
    throw "env file not found. Place .env in $Root, next to api.exe, or pass -EnvFile explicitly."
}

$env:ENV_FILE = $resolvedEnv

Write-Host "Using ENV_FILE=$resolvedEnv"
Write-Host "Starting $ExePath"

Set-Location $Root
& $ExePath
