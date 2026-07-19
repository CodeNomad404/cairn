# Cairn Windows 本地开发 - 停止所有服务
Write-Host "Stopping PostgreSQL..." -ForegroundColor Cyan
docker compose down
Write-Host "  PostgreSQL stopped (data kept)"
Write-Host ""
Write-Host "Tip: to also delete data, run: docker compose down -v" -ForegroundColor Yellow
Write-Host "Tip: press Ctrl+C in the backend terminal to stop Go server" -ForegroundColor Yellow
