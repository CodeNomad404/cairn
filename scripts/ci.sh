#!/usr/bin/env bash
set -Eeuo pipefail

readonly PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
readonly TARGET="${1:-all}"

log() {
  printf '\n==> %s\n' "$*"
}

require_command() {
  command -v "$1" >/dev/null 2>&1 || {
    printf 'missing required CI command: %s\n' "$1" >&2
    exit 1
  }
}

run_backend() {
  require_command go

  local build_output
  build_output="$(mktemp "${TMPDIR:-/tmp}/cairn-ci-server.XXXXXX")"
  trap 'rm -f "${build_output:-}"' RETURN

  log "Downloading Go dependencies"
  go mod download

  log "Running Go static checks"
  go vet ./...

  log "Running Go tests"
  go test ./...

  log "Building the Go server"
  go build -trimpath -o "${build_output}" ./cmd/server
}

run_frontend() {
  require_command node
  require_command npm

  local package_manager
  local -a pnpm_command

  package_manager="$(node -p "require('./web/package.json').packageManager")"
  case "${package_manager}" in
    pnpm@*) ;;
    *)
      printf 'web/package.json must declare packageManager as pnpm@<version>\n' >&2
      exit 1
      ;;
  esac

  if command -v pnpm >/dev/null 2>&1; then
    pnpm_command=(pnpm)
  else
    log "pnpm is not installed; using npm exec with ${package_manager}"
    pnpm_command=(npm exec --yes --package="${package_manager}" -- pnpm)
  fi

  log "Installing locked frontend dependencies"
  "${pnpm_command[@]}" --dir web install --frozen-lockfile

  log "Type-checking and building the frontend"
  "${pnpm_command[@]}" --dir web run build
}

cd "${PROJECT_ROOT}"

case "${TARGET}" in
  backend)
    run_backend
    ;;
  frontend)
    run_frontend
    ;;
  all)
    run_backend
    run_frontend
    ;;
  *)
    printf 'usage: %s [backend|frontend|all]\n' "$0" >&2
    exit 2
    ;;
esac

log "CI checks passed (${TARGET})"
