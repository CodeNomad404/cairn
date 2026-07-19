# Cairn Windows 本地开发一键启动脚本
# 用法：在项目根目录执行 .\dev.ps1
# 可选参数：.\dev.ps1 -SkipDb   跳过数据库启动（已手动启动时使用）

param(
    [switch]$SkipDb
)

$ErrorActionPreference = "Stop"
$projectRoot = Split-Path -Parent $MyInvocation.MyCommand.Path
Set-Location $projectRoot

# 环境变量
$env:DB_PASSWORD = "cairn_dev"
$env:DEV = "true"

# ----------------------------------------------------
# 1. 检查必要软件
# ----------------------------------------------------
Write-Host "Checking environment..." -ForegroundColor Cyan

$checks = @{
    "go"     = "Go not found, install from https://go.dev/dl/"
    "docker" = "Docker not found, install Docker Desktop first"
    "node"   = "Node.js not found, install from https://nodejs.org/"
}

foreach ($cmd in $checks.Keys) {
    if (-not (Get-Command $cmd -ErrorAction SilentlyContinue)) {
        Write-Host "[FAIL] $($checks[$cmd])" -ForegroundColor Red
        exit 1
    }
}
Write-Host "  Go:      $(go version)"
Write-Host "  Docker:  $(docker --version)"
Write-Host "  Node:    $(node --version)"

# ----------------------------------------------------
# 2. 启动 PostgreSQL
# ----------------------------------------------------
if (-not $SkipDb) {
    Write-Host ""
    Write-Host "Starting PostgreSQL..." -ForegroundColor Cyan

    # 检查 Docker Desktop 是否运行
    docker info 2>&1 | Out-Null
    if ($LASTEXITCODE -ne 0) {
        Write-Host "[FAIL] Docker Desktop is not running, start it first" -ForegroundColor Red
        exit 1
    }

    docker compose up -d
    if ($LASTEXITCODE -ne 0) {
        Write-Host "[FAIL] PostgreSQL failed to start" -ForegroundColor Red
        exit 1
    }

    # 等待 PostgreSQL 就绪
    Write-Host "Waiting for PostgreSQL to be ready..." -ForegroundColor Yellow
    $ready = $false
    for ($i = 0; $i -lt 30; $i++) {
        docker compose exec -T db pg_isready -U cairn -d cairn 2>&1 | Out-Null
        if ($LASTEXITCODE -eq 0) {
            $ready = $true
            break
        }
        Start-Sleep -Seconds 1
    }

    if (-not $ready) {
        Write-Host "[FAIL] PostgreSQL not ready in 30s, check: docker compose ps" -ForegroundColor Red
        exit 1
    }
    Write-Host "  PostgreSQL ready (127.0.0.1:5432)" -ForegroundColor Green
} else {
    Write-Host "  Skipping PostgreSQL (-SkipDb)" -ForegroundColor Yellow
}

# ----------------------------------------------------
# 3. 启动 Go 后端
# ----------------------------------------------------
Write-Host ""
Write-Host "Starting Go backend..." -ForegroundColor Cyan
Write-Host "  Backend : http://localhost:8080" -ForegroundColor White
Write-Host "  Health  : http://localhost:8080/healthz" -ForegroundColor White
Write-Host ""

# 检查端口占用
$port8080 = netstat -ano 2>$null | Select-String ":8080.*LISTENING"
if ($port8080) {
    Write-Host "[WARN] Port 8080 is in use:" -ForegroundColor Yellow
    Write-Host "  $port8080" -ForegroundColor Yellow
    Write-Host "  Close the process or run: .\stop.ps1" -ForegroundColor Yellow
    exit 1
}

Write-Host "========================================"
Write-Host "  Go backend running, Ctrl+C to stop"
Write-Host "========================================"
Write-Host ""

go run ./cmd/server
