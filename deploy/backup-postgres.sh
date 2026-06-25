#!/usr/bin/env bash
set -euo pipefail

: "${BACKUP_DIR:?BACKUP_DIR is required}"
: "${DB_PASSWORD:?DB_PASSWORD is required}"

timestamp="$(date -u +%Y%m%dT%H%M%SZ)"
backup_path="${BACKUP_DIR}/cairn-${timestamp}.sql.gz"

mkdir -p "${BACKUP_DIR}"
PGPASSWORD="${DB_PASSWORD}" pg_dump \
  --host=127.0.0.1 \
  --port=5432 \
  --username=cairn \
  --dbname=cairn \
  --no-owner \
  --no-privileges |
  gzip >"${backup_path}"

find "${BACKUP_DIR}" -type f -name 'cairn-*.sql.gz' -mtime +14 -delete
printf 'backup created: %s\n' "${backup_path}"
