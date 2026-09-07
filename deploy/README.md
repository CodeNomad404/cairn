# Cairn Deployment

Production layout (the full source checkout used for builds lives separately at
`/opt/cairn-repo`):

```text
/opt/cairn                 runtime files and operational database files
/opt/cairn-repo            server-side Git checkout used for builds
/etc/cairn/cairn.env       production secrets
/var/backups/cairn         database backups
```

## First deployment

The server needs Git, Go 1.26.3, Node.js 24 LTS with npm, golang-migrate,
rsync, Caddy, Docker, PostgreSQL client tools, curl, and Python 3. Use an LTS
Node.js release in production; Vite's preview server is not a production server.
The deployment script uses an installed pnpm when available, otherwise npm runs
the exact pnpm version declared in `web/package.json`.

```bash
sudo useradd --system --home /opt/cairn --shell /usr/sbin/nologin cairn
sudo mkdir -p /opt/cairn /etc/cairn /var/backups/cairn
sudo chown -R cairn:cairn /opt/cairn /var/backups/cairn

sudo cp deploy/cairn.env.example /etc/cairn/cairn.env
sudo chmod 600 /etc/cairn/cairn.env
# Edit the production password before continuing.

docker compose --env-file /etc/cairn/cairn.env up -d db
set -a
source /etc/cairn/cairn.env
set +a
migrate -path migrations \
  -database "postgres://cairn:${DB_PASSWORD}@127.0.0.1:5432/cairn?sslmode=disable" up

go build -trimpath -ldflags="-s -w" -o bin/cairn ./cmd/server
```

Install services:

```bash
sudo cp deploy/cairn.service /etc/systemd/system/
sudo cp deploy/cairn-backup.service /etc/systemd/system/
sudo cp deploy/cairn-backup.timer /etc/systemd/system/
sudo cp deploy/Caddyfile /etc/caddy/Caddyfile
sudo systemctl daemon-reload
sudo systemctl enable --now cairn cairn-backup.timer caddy
```

Verify:

```bash
curl --fail http://127.0.0.1:8080/healthz
systemctl status cairn
systemctl list-timers cairn-backup.timer
```

Before each deployment, run the backup service and migrations:

```bash
sudo systemctl start cairn-backup.service
set -a
source /etc/cairn/cairn.env
set +a
migrate -path migrations \
  -database "postgres://cairn:${DB_PASSWORD}@127.0.0.1:5432/cairn?sslmode=disable" up
sudo systemctl restart cairn
```

## Daily deployment

Create the local deployment configuration once. The real configuration is
ignored by Git:

```bash
cp deploy/deploy.env.example deploy/deploy.env
```

Edit `DEPLOY_HOST` and `GITEE_REPO_URL` in the copied configuration. Set
`DEPLOY_SSH_KEY` for passwordless server login and `REMOTE_GIT_SSH_KEY` to the
Gitee deployment key on the server. `GO_PROXY` controls the Go module proxy used
during the remote build and defaults to `https://goproxy.cn,direct`. The example
intentionally contains no real server address, repository, private key, or
password. The shared deployment script normally needs no local edits.

Install the deployment public key on the server once (this command asks for the
server password):

```bash
ssh-copy-id -i ~/.ssh/id_ed25519_cairn.pub ubuntu@YOUR_SERVER_IP
```

After that, daily deployments do not need the server password.

For a private Gitee repository, create a separate key on the server and add its
public key to Gitee:

```bash
ssh-keygen -t ed25519 -C "cairn-server-deploy" \
  -f ~/.ssh/id_ed25519_gitee
cat ~/.ssh/id_ed25519_gitee.pub
```

Then run it from the repository root:

```bash
./deploy/deploy.sh
# or: task deploy
```

The script also accepts deployment settings directly from environment variables,
which allows a CI/CD runner to use protected variables instead of creating
`deploy/deploy.env`. Set `DEPLOY_COMMIT` to the runner's full 40-character commit
SHA so the server deploys exactly the revision that passed CI:

```bash
DEPLOY_HOST=YOUR_SERVER_IP \
GITEE_REPO_URL=git@gitee.com:YOUR_ACCOUNT/cairn.git \
DEPLOY_COMMIT="$(git rev-parse HEAD)" \
./deploy/deploy.sh
```

Only one remote deployment may run at a time. A second deployment exits without
making changes while the first one holds the deployment lock.

## Continuous integration

Run all CI checks locally with:

```bash
./deploy/ci.sh
# or: task ci
```

CI systems with separate Go and Node.js build environments can run the two
halves independently:

```bash
./deploy/ci.sh backend
./deploy/ci.sh frontend
```

The backend job downloads modules, runs `go vet`, runs all Go tests, and builds
the server. The frontend job installs the exact locked dependencies, performs
Vue/TypeScript checks, and creates the production frontend build.

The local `deploy.sh` only loads configuration and sends `remote-deploy.sh` over
one SSH connection. The server clones or updates `master` in its source cache at
`/opt/cairn-repo`, downloads dependencies, runs Go tests, builds the Go binary
and Vue production assets, backs up PostgreSQL, runs forward migrations,
installs the new backend and frontend, restarts the service, and checks
`/healthz`. No source files or `node_modules` are copied into the runtime
directory. If the health check fails, it restores the previous binary, frontend,
and configuration. Applied database migrations are deliberately not rolled back.
After a successful deployment, only the immediately previous application release
is retained for rollback.

Use another configuration file when needed:

```bash
DEPLOY_CONFIG=/path/to/deploy.env ./deploy/deploy.sh
```

The script never stores an SSH password. SSH will prompt for one unless key-based
authentication is configured. The SSH user must be allowed to run the deployment
`sudo` commands.

When `deploy/Caddyfile` changes, validate and install it separately before
reloading Caddy:

```bash
sudo caddy validate --config /opt/cairn/deploy/Caddyfile
sudo install -o root -g root -m 0644 /opt/cairn/deploy/Caddyfile /etc/caddy/Caddyfile
sudo systemctl reload caddy
```
