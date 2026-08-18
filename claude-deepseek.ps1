# Claude Code - DeepSeek API 启动脚本
# 用法：. .\claude-deepseek.ps1   （注意前面有个点和空格，让环境变量导入当前会话）
# 或者直接执行启动 Claude Code：
#   .\claude-deepseek.ps1 -Launch

param(
    [switch]$Launch
)

if (-not $env:ANTHROPIC_AUTH_TOKEN) {
    throw "Set ANTHROPIC_AUTH_TOKEN to your DeepSeek API key before running this script."
}

$env:ANTHROPIC_BASE_URL            = "https://api.deepseek.com/anthropic"
$env:ANTHROPIC_MODEL               = "deepseek-v4-pro"
$env:ANTHROPIC_DEFAULT_OPUS_MODEL  = "deepseek-v4-pro"
$env:ANTHROPIC_DEFAULT_SONNET_MODEL = "deepseek-v4-pro"
$env:ANTHROPIC_DEFAULT_HAIKU_MODEL = "deepseek-v4-flash"
$env:CLAUDE_CODE_SUBAGENT_MODEL    = "deepseek-v4-flash"
$env:CLAUDE_CODE_EFFORT_LEVEL      = "max"

Write-Host "DeepSeek API environment configured:" -ForegroundColor Cyan
Write-Host "  Base URL  : $env:ANTHROPIC_BASE_URL"
Write-Host "  Model     : $env:ANTHROPIC_MODEL"
Write-Host "  Opus      : $env:ANTHROPIC_DEFAULT_OPUS_MODEL"
Write-Host "  Sonnet    : $env:ANTHROPIC_DEFAULT_SONNET_MODEL"
Write-Host "  Haiku     : $env:ANTHROPIC_DEFAULT_HAIKU_MODEL"
Write-Host "  Subagent  : $env:CLAUDE_CODE_SUBAGENT_MODEL"
Write-Host "  Effort    : $env:CLAUDE_CODE_EFFORT_LEVEL"

if ($Launch) {
    Write-Host ""
    Write-Host "Launching Claude Code..." -ForegroundColor Green
    claude
} else {
    Write-Host ""
    Write-Host "Tip: run with -Launch to start Claude Code directly" -ForegroundColor Yellow
    Write-Host "     .\claude-deepseek.ps1 -Launch" -ForegroundColor Yellow
}
