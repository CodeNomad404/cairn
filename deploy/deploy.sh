#!/usr/bin/env bash
set -Eeuo pipefail

readonly CONFIG_FILE="${DEPLOY_CONFIG:-deploy/deploy.env}"
if [[ ! -r "${CONFIG_FILE}" ]]; then
  printf 'deployment config not found: %s\ncopy deploy/deploy.env.example to deploy/deploy.env\n' "${CONFIG_FILE}" >&2
  exit 1
fi

set -a
# shellcheck disable=SC1090
source "${CONFIG_FILE}"
set +a

: "${DEPLOY_HOST:?DEPLOY_HOST is required in ${CONFIG_FILE}}"
: "${GITEE_REPO_SSH:?GITEE_REPO_SSH is required in ${CONFIG_FILE}}"
: "${GITEE_REPO_HTTPS:?GITEE_REPO_HTTPS is required in ${CONFIG_FILE}}"

readonly DEPLOY_HOST
readonly DEPLOY_PORT="${DEPLOY_PORT:-22}"
readonly DEPLOY_USER="${DEPLOY_USER:-ubuntu}"
readonly REMOTE_APP_DIR="${REMOTE_APP_DIR:-/opt/cairn}"
readonly REMOTE_REPO_DIR="${REMOTE_REPO_DIR:-/opt/cairn-repo}"
readonly GITEE_REPO_SSH
readonly GITEE_REPO_HTTPS
readonly GITEE_SSH_KEY="${GITEE_SSH_KEY:-}"
readonly SSH_TARGET="${DEPLOY_USER}@${DEPLOY_HOST}"

log() {
  printf '\n==> %s\n' "$*"
}

require_command() {
  command -v "$1" >/dev/null 2>&1 || {
    printf 'missing required command: %s\n' "$1" >&2
    exit 1
  }
}

require_command git
require_command ssh

if [[ ! -f go.mod || ! -d cmd/server ]]; then
  printf 'run this script from the Cairn repository root\n' >&2
  exit 1
fi
if [[ -n "$(git status --short)" ]]; then
  printf 'working tree is not clean; commit or stash local changes before deploying\n' >&2
  exit 1
fi

log "Switching to local main branch"
git switch main

if git remote get-url gitee >/dev/null 2>&1; then
  git remote set-url gitee "${GITEE_REPO_SSH}"
else
  git remote add gitee "${GITEE_REPO_SSH}"
fi

log "Pushing main branch to Gitee"
if [[ -n "${GITEE_SSH_KEY}" ]]; then
  GIT_SSH_COMMAND="ssh -i ${GITEE_SSH_KEY} -o IdentitiesOnly=yes" git push gitee main
else
  git push gitee main
fi

release_id="$(date -u +%Y%m%dT%H%M%SZ)-$$"
ssh_control_path="/tmp/cairn-ssh-${USER:-user}-$$"
ssh_options=(-p "${DEPLOY_PORT}" -o ControlMaster=auto -o ControlPersist=60 -S "${ssh_control_path}")
ssh=(ssh "${ssh_options[@]}" "${SSH_TARGET}")
ssh_connected=false

cleanup() {
  if [[ "${ssh_connected}" == true ]]; then
    ssh "${ssh_options[@]}" -O exit "${SSH_TARGET}" >/dev/null 2>&1 || true
  fi
}
trap cleanup EXIT

log "Connecting to ${SSH_TARGET}:${DEPLOY_PORT}"
"${ssh[@]}" true
ssh_connected=true

log "Pulling from Gitee and deploying on the server"
"${ssh[@]}" "REMOTE_APP_DIR='${REMOTE_APP_DIR}' REMOTE_REPO_DIR='${REMOTE_REPO_DIR}' GITEE_REPO_HTTPS='${GITEE_REPO_HTTPS}' RELEASE_ID='${release_id}' bash -s" <<'REMOTE_SCRIPT'
set -Eeuo pipefail

export PATH="/usr/local/go/bin:${HOME}/go/bin:${PATH}"

: "${REMOTE_APP_DIR:?}"
: "${REMOTE_REPO_DIR:?}"
: "${GITEE_REPO_HTTPS:?}"
: "${RELEASE_ID:?}"

readonly env_file=/etc/cairn/cairn.env
readonly service=cairn
readonly current_binary="${REMOTE_APP_DIR}/bin/cairn"
readonly new_binary="/tmp/cairn-${RELEASE_ID}"
readonly install_binary="${REMOTE_APP_DIR}/bin/cairn.new.${RELEASE_ID}"
readonly old_binary="${REMOTE_APP_DIR}/bin/cairn.${RELEASE_ID}"
binary_replaced=false

rollback() {
  exit_code=$?
  trap - ERR
  rm -f "${new_binary}"
  sudo rm -f "${install_binary}"

  if [[ "${binary_replaced}" == true && -f "${old_binary}" ]]; then
    printf '\nDeployment failed; restoring the previous binary...\n' >&2
    sudo install -o cairn -g cairn -m 0755 "${old_binary}" "${current_binary}"
    sudo systemctl restart "${service}"
    if curl --fail --silent --show-error --retry 5 --retry-delay 2 \
      http://127.0.0.1:8080/healthz >/dev/null; then
      printf 'Rollback succeeded. Database migrations were not reverted.\n' >&2
    else
      printf 'Rollback health check failed; inspect: sudo journalctl -u cairn -n 100 --no-pager\n' >&2
    fi
  fi
  exit "${exit_code}"
}
trap rollback ERR

for command_name in git go migrate rsync curl python3; do
  command -v "${command_name}" >/dev/null 2>&1 || {
    printf 'missing required server command: %s\n' "${command_name}" >&2
    exit 1
  }
done
sudo test -r "${env_file}" || {
  printf 'cannot read production environment file: %s\n' "${env_file}" >&2
  exit 1
}

if [[ -d "${REMOTE_REPO_DIR}/.git" ]]; then
  sudo -u cairn git -C "${REMOTE_REPO_DIR}" remote set-url origin "${GITEE_REPO_HTTPS}"
  sudo -u cairn git -C "${REMOTE_REPO_DIR}" fetch --prune origin main
  sudo -u cairn git -C "${REMOTE_REPO_DIR}" checkout -B main origin/main
else
  if [[ -e "${REMOTE_REPO_DIR}" ]]; then
    printf 'server repository path exists but is not a Git repository: %s\n' "${REMOTE_REPO_DIR}" >&2
    exit 1
  fi
  sudo mkdir -p "${REMOTE_REPO_DIR}"
  sudo chown cairn:cairn "${REMOTE_REPO_DIR}"
  sudo -u cairn git clone --branch main --single-branch \
    "${GITEE_REPO_HTTPS}" "${REMOTE_REPO_DIR}"
fi

cd "${REMOTE_REPO_DIR}"
go mod download
go test ./...
go build -trimpath -ldflags='-s -w' -o "${new_binary}" ./cmd/server

sudo systemctl start cairn-backup.service

sudo bash -c '
  set -Eeuo pipefail
  set -a
  source "$1"
  set +a
  : "${DB_PASSWORD:?DB_PASSWORD is required}"
  encoded_password="$(python3 -c '\''import sys, urllib.parse; print(urllib.parse.quote(sys.argv[1], safe=""))'\'' "${DB_PASSWORD}")"
  migrate -path "$2" \
    -database "postgres://cairn:${encoded_password}@127.0.0.1:5432/cairn?sslmode=disable" up
' _ "${env_file}" "${REMOTE_REPO_DIR}/migrations"

sudo mkdir -p "${REMOTE_APP_DIR}/bin"
if [[ -f "${current_binary}" ]]; then
  sudo cp --preserve=mode,ownership,timestamps "${current_binary}" "${old_binary}"
fi

for directory in cmd internal web migrations deploy; do
  sudo mkdir -p "${REMOTE_APP_DIR}/${directory}"
  sudo rsync -a --delete "${REMOTE_REPO_DIR}/${directory}/" "${REMOTE_APP_DIR}/${directory}/"
done
for file in go.mod go.sum config.yaml docker-compose.yml; do
  sudo install -o cairn -g cairn -m 0644 "${REMOTE_REPO_DIR}/${file}" "${REMOTE_APP_DIR}/${file}"
done

sudo install -o cairn -g cairn -m 0755 "${new_binary}" "${install_binary}"
sudo mv "${install_binary}" "${current_binary}"
rm -f "${new_binary}"
binary_replaced=true
sudo systemctl restart "${service}"

curl --fail --silent --show-error --retry 10 --retry-delay 2 \
  http://127.0.0.1:8080/healthz >/dev/null
sudo systemctl is-active --quiet "${service}"

trap - ERR
printf '\nDeployment succeeded: %s (%s)\n' \
  "${RELEASE_ID}" "$(git -C "${REMOTE_REPO_DIR}" rev-parse --short HEAD)"
REMOTE_SCRIPT

log "Cairn is live at https://cairn.cn"
