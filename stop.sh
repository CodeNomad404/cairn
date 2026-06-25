#!/bin/bash
# Cairn 本地开发停止脚本

echo "🛑 停止 Cairn..."

# 停止 air 和 cairn 相关进程
pkill -f "air" 2>/dev/null && echo "   已停止 air"
pkill -f "cairn" 2>/dev/null && echo "   已停止 cairn"
pkill -f "tmp/main" 2>/dev/null && echo "   已停止 tmp/main"

sleep 1

# 确认端口已释放
if lsof -i :8080 -sTCP:LISTEN > /dev/null 2>&1; then
  echo "⚠️  端口 8080 仍被占用，强制释放..."
  lsof -ti :8080 -sTCP:LISTEN | xargs kill -9 2>/dev/null
fi

echo "✅ 已停止"
