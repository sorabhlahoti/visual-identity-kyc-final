$ErrorActionPreference = "Stop"
Set-Location "$PSScriptRoot\..\inference"

if (-not (Get-Command uv -ErrorAction SilentlyContinue)) {
  Write-Host "uv not found. Installing uv for the current Windows user..."
  powershell -ExecutionPolicy ByPass -c "irm https://astral.sh/uv/install.ps1 | iex"
  $env:Path = "$env:USERPROFILE\.local\bin;$env:USERPROFILE\.cargo\bin;$env:Path"
}

uv venv .venv --python 3.11
.\.venv\Scripts\python.exe -m ensurepip --upgrade
uv sync
Write-Host "Inference venv is ready at inference\.venv"
Write-Host "Next: .\.venv\Scripts\python.exe scripts\download_models.py"
