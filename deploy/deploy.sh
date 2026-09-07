#!/usr/bin/env bash
set -Eeuo pipefail

readonly PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
readonly CONFIG_FILE="${DEPLOY_CONFIG:-${PROJECT_ROOT}/deploy/deploy.env}"
readonly REMOTE_SCRIPT="${PROJECT_ROOT}/deploy/remote-deploy.sh"

if [[ -r "${CONFIG_FILE}" ]]; then
  set -a
  # shellcheck disable=SC1090
  source "${CONFIG_FILE}"
  set +a
elif [[ -z "${DEPLOY_HOST:-}" || -z "${GITEE_REPO_URL:-}" ]]; then
  printf 'deployment config not found: %s\ncopy deploy/deploy.env.example to deploy/deploy.env\n' "${CONFIG_FILE}" >&2
  exit 1
fi

: "${DEPLOY_HOST:?DEPLOY_HOST is required}"
: "${GITEE_REPO_URL:?GITEE_REPO_URL is required}"

readonly DEPLOY_PORT="${DEPLOY_PORT:-22}"
readonly DEPLOY_USER="${DEPLOY_USER:-ubuntu}"
readonly DEPLOY_SSH_KEY="${DEPLOY_SSH_KEY:-}"
readonly REMOTE_APP_DIR="${REMOTE_APP_DIR:-/opt/cairn}"
readonly REMOTE_REPO_DIR="${REMOTE_REPO_DIR:-/opt/cairn-repo}"
readonly REMOTE_GIT_SSH_KEY="${REMOTE_GIT_SSH_KEY:-}"
readonly GO_PROXY="${GO_PROXY:-https://goproxy.cn,direct}"
readonly DEPLOY_COMMIT="${DEPLOY_COMMIT:-}"
readonly RELEASE_ID="$(date -u +%Y%m%dT%H%M%SZ)-$$"
readonly SSH_TARGET="${DEPLOY_USER}@${DEPLOY_HOST}"

if [[ ! -r "${REMOTE_SCRIPT}" ]]; then
  printf 'remote deployment script not found: %s\n' "${REMOTE_SCRIPT}" >&2
  exit 1
fi

if [[ -n "${DEPLOY_COMMIT}" && ! "${DEPLOY_COMMIT}" =~ ^[0-9a-fA-F]{40}$ ]]; then
  printf 'DEPLOY_COMMIT must be a full 40-character Git commit SHA\n' >&2
  exit 1
fi

ssh_options=(-p "${DEPLOY_PORT}")
if [[ -n "${DEPLOY_SSH_KEY}" ]]; then
  ssh_options+=(-i "${DEPLOY_SSH_KEY}" -o IdentitiesOnly=yes)
fi

remote_args=(
  "${REMOTE_APP_DIR}"
  "${REMOTE_REPO_DIR}"
  "${REMOTE_GIT_SSH_KEY}"
  "${GITEE_REPO_URL}"
  "${GO_PROXY}"
  "${DEPLOY_COMMIT}"
  "${RELEASE_ID}"
)
remote_command='bash -s --'
for argument in "${remote_args[@]}"; do
  printf -v quoted_argument '%q' "${argument}"
  remote_command+=" ${quoted_argument}"
done

printf '\n==> Deploying to %s:%s\n' "${SSH_TARGET}" "${DEPLOY_PORT}"
ssh "${ssh_options[@]}" "${SSH_TARGET}" "${remote_command}" <"${REMOTE_SCRIPT}"

printf '\n==> Cairn is live at https://cairn.cn\n'
