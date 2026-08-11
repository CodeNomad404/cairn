$ErrorActionPreference = "Stop"
$scriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$projectRoot = Split-Path -Parent $scriptDir
Set-Location $projectRoot

docker compose down
Write-Host "Cairn stopped. Database data was preserved." -ForegroundColor Green
