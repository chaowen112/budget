$ErrorActionPreference = "Stop"

$RootDir = Split-Path -Parent $PSScriptRoot
Set-Location $RootDir

if (-not (Get-Command docker -ErrorAction SilentlyContinue)) {
  throw "docker is not installed or not in PATH."
}

if ((-not (Test-Path ".env")) -and (Test-Path ".env.example")) {
  Copy-Item ".env.example" ".env"
  Write-Host "Created .env from .env.example"
}

Write-Host "Stopping containers and deleting volumes..."
docker compose down -v --remove-orphans

Write-Host "Done. Database volume has been reset."
Write-Host "Services are currently stopped. Start manually when ready:"
Write-Host "  docker compose up -d --build"
