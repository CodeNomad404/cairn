$ErrorActionPreference = "Stop"
$projectRoot = Split-Path -Parent $MyInvocation.MyCommand.Path
Set-Location $projectRoot

docker compose down
Write-Host "Cairn stopped. Database data was preserved." -ForegroundColor Green
