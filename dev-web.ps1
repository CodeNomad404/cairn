# Cairn Windows 本地开发 - 启动前端
# 用法：在项目根目录执行 .\dev-web.ps1

$ErrorActionPreference = "Stop"
$projectRoot = Split-Path -Parent $MyInvocation.MyCommand.Path

Write-Host "Starting Vue frontend..." -ForegroundColor Cyan

Set-Location "$projectRoot\web"

# 检查 node_modules
if (-not (Test-Path "node_modules")) {
    Write-Host "First run, installing dependencies..." -ForegroundColor Yellow
    pnpm install
    if ($LASTEXITCODE -ne 0) {
        Write-Host "[WARN] pnpm install failed, trying npm install..." -ForegroundColor Yellow
        npm install
    }
}

Write-Host "  Frontend: http://localhost:5173"
Write-Host "  API proxy /api/* -> http://localhost:8080"
Write-Host ""
Write-Host "========================================"
Write-Host "  Vue frontend running, Ctrl+C to stop"
Write-Host "========================================"
Write-Host ""

pnpm dev
