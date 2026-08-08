#!/bin/bash
set -e

cd "$(dirname "$0")"

if ! docker info >/dev/null 2>&1; then
  echo "Docker is not running. Start Docker Desktop and try again." >&2
  exit 1
fi

ensure_image() {
  mirror_image="$1"
  official_image="$2"

  if docker image inspect "${official_image}" >/dev/null 2>&1; then
    return
  fi

  echo "Pulling ${official_image} from the DaoCloud mirror..."
  docker pull "${mirror_image}"
  docker tag "${mirror_image}" "${official_image}"
}

ensure_image "docker.m.daocloud.io/library/golang:1.26-alpine" "golang:1.26-alpine"
ensure_image "docker.m.daocloud.io/library/node:24-alpine" "node:24-alpine"
ensure_image "docker.m.daocloud.io/library/postgres:16" "postgres:16"
ensure_image "docker.m.daocloud.io/library/alpine:3.24.1" "alpine:3.24.1"

echo "启动 Cairn 本地开发环境..."
echo "前端：http://localhost:5173"
echo "后端：http://localhost:8080"
echo ""

exec docker compose up --build
