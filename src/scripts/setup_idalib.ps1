#!/usr/bin/env pwsh
# Setup idalib for ida-headless-mcp on Windows.
# Auto-detects the latest IDA Pro / Essential installation under
# C:\Program Files. Override with -IdaPath or env IDA_PATH.
#
# Installs idapro into the plugin-local venv (created by scripts/launch.py)
# unless -SystemPython is passed, in which case the global `python` is used.

[CmdletBinding()]
param(
    [string]$IdaPath = $env:IDA_PATH,
    [switch]$SystemPython
)

$ErrorActionPreference = "Stop"

function Resolve-PythonExe {
    if ($SystemPython) { return "python" }

    $repoRoot = Resolve-Path (Join-Path $PSScriptRoot "..\..")
    $venvCandidates = @(
        Join-Path $repoRoot ".venv\Scripts\python.exe"
    )
    foreach ($candidate in $venvCandidates) {
        if (Test-Path $candidate) { return $candidate }
    }

    Write-Host "Plugin venv not found at $($venvCandidates[0])." -ForegroundColor Red
    Write-Host "Run 'python scripts/launch.py version' once to bootstrap it,"
    Write-Host "or pass -SystemPython to install idalib globally."
    exit 1
}

$Py = Resolve-PythonExe
Write-Host "Using Python: $Py" -ForegroundColor Cyan

function Find-IdaPath {
    if ($IdaPath -and (Test-Path $IdaPath)) {
        return (Resolve-Path $IdaPath).Path
    }

    $patterns = @(
        "C:\Program Files\IDA Pro*",
        "C:\Program Files\IDA Essential*",
        "C:\Program Files\IDA*"
    )
    $candidates = @()
    foreach ($p in $patterns) {
        $candidates += Get-ChildItem -Path $p -Directory -ErrorAction SilentlyContinue
    }
    if ($candidates.Count -eq 0) { return $null }

    return ($candidates | Sort-Object Name -Descending | Select-Object -First 1).FullName
}

$IdaPath = Find-IdaPath
if (-not $IdaPath) {
    Write-Host "No IDA installation found under C:\Program Files." -ForegroundColor Red
    Write-Host "Set IDA_PATH or pass -IdaPath 'C:\Program Files\IDA Pro 9.X'."
    exit 1
}

Write-Host "Found $IdaPath" -ForegroundColor Green

$IdalibDir = Join-Path $IdaPath "idalib"
if (-not (Test-Path $IdalibDir)) {
    Write-Host "idalib directory not found in $IdaPath" -ForegroundColor Red
    Write-Host "idalib requires IDA Pro 9.0+ or IDA Essential 9.2+."
    exit 1
}
Write-Host "Found idalib" -ForegroundColor Green

$PythonDir = Join-Path $IdalibDir "python"
$Wheel = Get-ChildItem -Path (Join-Path $PythonDir "*.whl") -ErrorAction SilentlyContinue | Select-Object -First 1
$SetupPy = Join-Path $PythonDir "setup.py"

Write-Host ""
Write-Host "Installing idapro Python package..."
if ($Wheel) {
    & $Py -m pip install --force-reinstall $Wheel.FullName
    if ($LASTEXITCODE -ne 0) {
        Write-Host "pip install failed for $($Wheel.Name)" -ForegroundColor Yellow
        exit 1
    }
    Write-Host "Installed $($Wheel.Name)" -ForegroundColor Green
} elseif (Test-Path $SetupPy) {
    & $Py -m pip install $PythonDir
    if ($LASTEXITCODE -ne 0) {
        Write-Host "pip install failed" -ForegroundColor Yellow
        exit 1
    }
    Write-Host "Installed idapro via setup.py" -ForegroundColor Green
} else {
    Write-Host "No wheel or setup.py found in $PythonDir" -ForegroundColor Red
    exit 1
}

Write-Host ""
Write-Host "Activating idalib..."
$ActivateScript = Join-Path $PythonDir "py-activate-idalib.py"
if (-not (Test-Path $ActivateScript)) {
    Write-Host "Activation script not found: $ActivateScript" -ForegroundColor Red
    exit 1
}
& $Py $ActivateScript -d $IdaPath
if ($LASTEXITCODE -ne 0) {
    Write-Host "Failed to activate idalib" -ForegroundColor Red
    exit 1
}
Write-Host "idalib activated" -ForegroundColor Green

Write-Host ""
Write-Host "Testing idalib import..."
& $Py -c "import idapro; v=idapro.get_library_version(); print(f'idalib {v[0]}.{v[1]} ready')"
if ($LASTEXITCODE -ne 0) {
    Write-Host "Failed to import idapro" -ForegroundColor Red
    exit 1
}
Write-Host "Setup complete" -ForegroundColor Green
