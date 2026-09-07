$ErrorActionPreference = "Stop"
$scriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$projectRoot = Split-Path -Parent $scriptDir
Set-Location $projectRoot

Write-Host "Starting Cairn local development environment..." -ForegroundColor Cyan

docker info *> $null
if ($LASTEXITCODE -ne 0) {
    throw "Docker Desktop is not running. Start it and try again."
}

$images = @(
    @{ Mirror = "docker.m.daocloud.io/library/golang:1.26-alpine"; Official = "golang:1.26-alpine" },
    @{ Mirror = "docker.m.daocloud.io/library/node:24-alpine"; Official = "node:24-alpine" },
    @{ Mirror = "docker.m.daocloud.io/library/postgres:16"; Official = "postgres:16" },
    @{ Mirror = "docker.m.daocloud.io/library/alpine:3.24.1"; Official = "alpine:3.24.1" }
)

foreach ($image in $images) {
    docker image inspect $image.Official *> $null
    if ($LASTEXITCODE -eq 0) {
        continue
    }

    Write-Host "Pulling $($image.Official) from the DaoCloud mirror..." -ForegroundColor Yellow
    docker pull $image.Mirror
    if ($LASTEXITCODE -ne 0) {
        throw "Failed to pull $($image.Official) from $($image.Mirror)"
    }

    docker tag $image.Mirror $image.Official
    if ($LASTEXITCODE -ne 0) {
        throw "Failed to tag $($image.Official)"
    }
}

Write-Host "Frontend: http://localhost:5173"
Write-Host "Backend:  http://localhost:8080"
Write-Host ""

docker compose up --build
