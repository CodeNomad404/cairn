#!/bin/bash
set -e

cd "$(dirname "$0")"

docker compose down
echo "Cairn 已停止，数据库数据已保留。"
