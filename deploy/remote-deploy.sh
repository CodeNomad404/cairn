#!/usr/bin/env bash
set -Eeuo pipefail

if [[ $# -ne 7 ]]; then
  printf 'usage: remote-deploy.sh APP_DIR REPO_DIR GIT_KEY REPO_URL GOPROXY COMMIT RELEASE_ID\n' >&2
  exit 2
fi

readonly REMOTE_APP_DIR="$1"
readonly REMOTE_REPO_DIR="$2"
readonly REMOTE_GIT_SSH_KEY="$3"
readonly GITEE_REPO_URL="$4"
readonly GOPROXY="$5"
readonly DEPLOY_COMMIT="$6"
readonly RELEASE_ID="$7"
readonly ENV_FILE=/etc/cairn/cairn.env
readonly SERVICE=cairn

readonly CURRENT_BINARY="${REMOTE_APP_DIR}/bin/cairn"
readonly NEW_BINARY="/tmp/cairn-${RELEASE_ID}"
readonly OLD_BINARY="${REMOTE_APP_DIR}/bin/cairn.${RELEASE_ID}"
readonly CURRENT_FRONTEND="${REMOTE_APP_DIR}/web/dist"
readonly NEW_FRONTEND="${REMOTE_APP_DIR}/web/dist.new.${RELEASE_ID}"
readonly OLD_FRONTEND="${REMOTE_APP_DIR}/web/dist.${RELEASE_ID}"
readonly CURRENT_CONFIG="${REMOTE_APP_DIR}/config.yaml"
readonly OLD_CONFIG="${REMOTE_APP_DIR}/config.yaml.${RELEASE_ID}"

export PATH="/usr/local/go/bin:${HOME}/go/bin:${PATH}"
export GOPROXY

log() {
  printf '\n==> %s\n' "$*"
}

require_commands() {
  local command_name
  for command_name in "$@"; do
    command -v "${command_name}" >/dev/null 2>&1 || {
      printf 'missing required server command: %s\n' "${command_name}" >&2
      exit 1
    }
  done
}

healthcheck() {
  curl --fail --silent --show-error --retry 15 --retry-delay 1 \
    --retry-connrefused http://127.0.0.1:8080/healthz >/dev/null
}

binary_replaced=false
frontend_moved=false
frontend_installed=false
config_replaced=false

rollback() {
  local exit_code=$?
  trap - ERR

  rm -f "${NEW_BINARY}"
  sudo rm -rf "${NEW_FRONTEND}"

  if [[ "${frontend_installed}" == true ]]; then
    sudo rm -rf "${CURRENT_FRONTEND}"
  fi
  if [[ "${frontend_moved}" == true && -d "${OLD_FRONTEND}" ]]; then
    sudo mv "${OLD_FRONTEND}" "${CURRENT_FRONTEND}"
  fi

  if [[ "${config_replaced}" == true && -f "${OLD_CONFIG}" ]]; then
    sudo install -o cairn -g cairn -m 0644 "${OLD_CONFIG}" "${CURRENT_CONFIG}"
  fi

  if [[ "${binary_replaced}" == true && -f "${OLD_BINARY}" ]]; then
    log "Deployment failed; restoring the previous release"
    sudo install -o cairn -g cairn -m 0755 "${OLD_BINARY}" "${CURRENT_BINARY}"
    sudo systemctl restart "${SERVICE}"
    if healthcheck; then
      printf 'Rollback succeeded. Database migrations were not reverted.\n' >&2
    else
      printf 'Rollback health check failed; inspect: sudo journalctl -u cairn -n 100 --no-pager\n' >&2
    fi
  fi

  exit "${exit_code}"
}

require_commands git go node npm migrate rsync curl python3 flock

exec 9>/tmp/cairn-deploy.lock
if ! flock -n 9; then
  printf 'another Cairn deployment is already running\n' >&2
  exit 1
fi

if [[ -n "${DEPLOY_COMMIT}" && ! "${DEPLOY_COMMIT}" =~ ^[0-9a-fA-F]{40}$ ]]; then
  printf 'DEPLOY_COMMIT must be a full 40-character Git commit SHA\n' >&2
  exit 1
fi

sudo test -r "${ENV_FILE}" || {
  printf 'cannot read production environment file: %s\n' "${ENV_FILE}" >&2
  exit 1
}

if [[ -n "${REMOTE_GIT_SSH_KEY}" ]]; then
  printf -v GIT_SSH_COMMAND 'ssh -i %q -o IdentitiesOnly=yes -o StrictHostKeyChecking=accept-new' \
    "${REMOTE_GIT_SSH_KEY}"
  export GIT_SSH_COMMAND
fi

log "Updating the server repository"
if [[ -d "${REMOTE_REPO_DIR}/.git" ]]; then
  git -C "${REMOTE_REPO_DIR}" remote set-url origin "${GITEE_REPO_URL}"
  git -C "${REMOTE_REPO_DIR}" fetch --prune origin \
    +refs/heads/master:refs/remotes/origin/master
else
  if [[ -e "${REMOTE_REPO_DIR}" && -n "$(ls -A "${REMOTE_REPO_DIR}")" ]]; then
    printf 'server repository path exists but is not a Git repository: %s\n' \
      "${REMOTE_REPO_DIR}" >&2
    exit 1
  fi
  sudo mkdir -p "${REMOTE_REPO_DIR}"
  sudo chown "$(id -un):$(id -gn)" "${REMOTE_REPO_DIR}"
  git clone --branch master --single-branch "${GITEE_REPO_URL}" "${REMOTE_REPO_DIR}"
fi

if [[ -n "${DEPLOY_COMMIT}" ]]; then
  git -C "${REMOTE_REPO_DIR}" cat-file -e "${DEPLOY_COMMIT}^{commit}"
  git -C "${REMOTE_REPO_DIR}" merge-base --is-ancestor \
    "${DEPLOY_COMMIT}" origin/master || {
      printf 'requested commit is not contained in origin/master: %s\n' \
        "${DEPLOY_COMMIT}" >&2
      exit 1
    }
  git -C "${REMOTE_REPO_DIR}" checkout --detach "${DEPLOY_COMMIT}"
else
  git -C "${REMOTE_REPO_DIR}" checkout -B master origin/master
fi

cd "${REMOTE_REPO_DIR}"

log "Building the backend"
go mod download
go test ./...
go build -trimpath -ldflags='-s -w' -o "${NEW_BINARY}" ./cmd/server

if command -v pnpm >/dev/null 2>&1; then
  pnpm_command=(pnpm)
else
  readonly package_manager="$(node -p "require('./web/package.json').packageManager")"
  case "${package_manager}" in
    pnpm@*) ;;
    *)
      printf 'web/package.json must declare packageManager as pnpm@<version>\n' >&2
      exit 1
      ;;
  esac
  pnpm_command=(npm exec --yes --package="${package_manager}" -- pnpm)
fi

log "Building the frontend"
"${pnpm_command[@]}" --dir web install --frozen-lockfile
"${pnpm_command[@]}" --dir web run build

log "Backing up and migrating PostgreSQL"
sudo systemctl start cairn-backup.service
readonly MIGRATE_BIN="$(command -v migrate)"
sudo bash -c '
  set -Eeuo pipefail
  set -a
  source "$1"
  set +a
  : "${DB_PASSWORD:?DB_PASSWORD is required}"
  encoded_password="$(python3 -c '\''import sys, urllib.parse; print(urllib.parse.quote(sys.argv[1], safe=""))'\'' "${DB_PASSWORD}")"
  "$3" -path "$2" \
    -database "postgres://cairn:${encoded_password}@127.0.0.1:5432/cairn?sslmode=disable" up
' _ "${ENV_FILE}" "${REMOTE_REPO_DIR}/migrations" "${MIGRATE_BIN}"

# Keep only the operational files needed outside the source checkout.
sudo mkdir -p "${REMOTE_APP_DIR}/migrations" "${REMOTE_APP_DIR}/deploy"
sudo rsync -a --delete "${REMOTE_REPO_DIR}/migrations/" "${REMOTE_APP_DIR}/migrations/"
sudo install -o cairn -g cairn -m 0755 \
  "${REMOTE_REPO_DIR}/deploy/backup-postgres.sh" \
  "${REMOTE_APP_DIR}/deploy/backup-postgres.sh"
sudo install -o cairn -g cairn -m 0644 \
  "${REMOTE_REPO_DIR}/docker-compose.yml" "${REMOTE_APP_DIR}/docker-compose.yml"

log "Installing the new release"
sudo mkdir -p "${REMOTE_APP_DIR}/bin" "${REMOTE_APP_DIR}/web"
sudo mkdir -p "${NEW_FRONTEND}"
sudo rsync -a --delete "${REMOTE_REPO_DIR}/web/dist/" "${NEW_FRONTEND}/"
sudo chown -R cairn:cairn "${NEW_FRONTEND}"

if [[ -f "${CURRENT_BINARY}" ]]; then
  sudo cp --preserve=mode,ownership,timestamps "${CURRENT_BINARY}" "${OLD_BINARY}"
fi
if [[ -f "${CURRENT_CONFIG}" ]]; then
  sudo cp --preserve=mode,ownership,timestamps "${CURRENT_CONFIG}" "${OLD_CONFIG}"
fi

trap rollback ERR

sudo install -o cairn -g cairn -m 0644 config.yaml "${CURRENT_CONFIG}"
config_replaced=true
sudo install -o cairn -g cairn -m 0755 "${NEW_BINARY}" "${CURRENT_BINARY}"
rm -f "${NEW_BINARY}"
binary_replaced=true

if [[ -d "${CURRENT_FRONTEND}" ]]; then
  sudo mv "${CURRENT_FRONTEND}" "${OLD_FRONTEND}"
  frontend_moved=true
fi
sudo mv "${NEW_FRONTEND}" "${CURRENT_FRONTEND}"
frontend_installed=true

sudo systemctl restart "${SERVICE}"
healthcheck
sudo test -s "${CURRENT_FRONTEND}/index.html"
sudo systemctl is-active --quiet "${SERVICE}"

trap - ERR

# Keep the immediately previous release and remove older application copies.
if ! sudo find "${REMOTE_APP_DIR}/bin" -maxdepth 1 -type f -name 'cairn.20*T*' \
  ! -name "cairn.${RELEASE_ID}" -delete; then
  printf 'warning: could not prune old backend releases\n' >&2
fi
if ! sudo find "${REMOTE_APP_DIR}/web" -mindepth 1 -maxdepth 1 -type d \
  -name 'dist.20*T*' ! -name "dist.${RELEASE_ID}" -exec rm -rf -- {} +; then
  printf 'warning: could not prune old frontend releases\n' >&2
fi
if ! sudo find "${REMOTE_APP_DIR}" -maxdepth 1 -type f -name 'config.yaml.20*T*' \
  ! -name "config.yaml.${RELEASE_ID}" -delete; then
  printf 'warning: could not prune old configuration copies\n' >&2
fi

printf '\nDeployment succeeded: %s (%s)\n' \
  "${RELEASE_ID}" "$(git -C "${REMOTE_REPO_DIR}" rev-parse --short HEAD)"
