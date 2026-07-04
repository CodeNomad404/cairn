#!/bin/bash
# Cairn 本地开发启动脚本
set -e

cd "$(dirname "$0")"

export DB_PASSWORD=cairn_dev
export DEV=true

# 检查端口是否被占用
if lsof -i :8080 -sTCP:LISTEN > /dev/null 2>&1; then
  echo "⚠️  端口 8080 已被占用，正在释放..."
  ./stop.sh
  sleep 1
fi

# 检查并启动数据库
if ! docker compose ps --status running 2>/dev/null | grep -q .; then
  echo "🐘 启动 PostgreSQL..."
  docker compose up -d
fi

echo "🐘 等待 PostgreSQL 就绪..."
for _ in {1..30}; do
  if docker compose exec -T db pg_isready -U cairn -d cairn >/dev/null 2>&1; then
    break
  fi
  sleep 1
done
if ! docker compose exec -T db pg_isready -U cairn -d cairn >/dev/null 2>&1; then
  echo "❌ PostgreSQL 在 30 秒内未就绪"
  exit 1
fi

# 确保 air 可执行
if ! command -v air &> /dev/null; then
  if [ -f "$HOME/go/bin/air" ]; then
    export PATH="$HOME/go/bin:$PATH"
  else
    echo "❌ 未找到 air，请先安装：go install github.com/air-verse/air@latest"
    exit 1
  fi
fi

echo "🚀 Cairn 开发模式启动..."
echo "   地址：http://localhost:8080"
echo "   按 Ctrl+C 停止"
echo ""

exec air
