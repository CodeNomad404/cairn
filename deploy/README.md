# Cairn Deployment

Production layout:

```text
/opt/cairn                 application source and binary
/etc/cairn/cairn.env       production secrets
/var/backups/cairn         database backups
```

## First deployment

```bash
sudo useradd --system --home /opt/cairn --shell /usr/sbin/nologin cairn
sudo mkdir -p /opt/cairn /etc/cairn /var/backups/cairn
sudo chown -R cairn:cairn /opt/cairn /var/backups/cairn

sudo cp deploy/cairn.env.example /etc/cairn/cairn.env
sudo chmod 600 /etc/cairn/cairn.env
# Edit the production password before continuing.

docker compose --env-file /etc/cairn/cairn.env up -d
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

Edit `DEPLOY_HOST`, `GITEE_REPO_SSH`, and `GITEE_REPO_HTTPS` in the copied
configuration. Set `GITEE_SSH_KEY` when the local Gitee push uses a non-default
SSH key. The example intentionally contains no real server address, repository,
or password. The shared deployment script normally needs no local edits.

Then run it from the repository root:

```bash
./deploy/deploy.sh
# or: task deploy
```

It refuses to run with uncommitted changes, switches to `main`, and pushes
`main` to Gitee. The server then clones or updates its source cache at
`/opt/cairn-repo` from Gitee, downloads
dependencies, tests, builds, backs up PostgreSQL, runs forward migrations,
installs the new binary, restarts the service, and checks `/healthz`. No source
files are uploaded from the local machine. If the health check fails, it restores
the previous binary. Applied database migrations are deliberately not rolled
back.

Use another configuration file when needed:

```bash
DEPLOY_CONFIG=/path/to/deploy.env ./deploy/deploy.sh
```

The script never stores an SSH password. SSH will prompt for one unless key-based
authentication is configured. The SSH user must be allowed to run the deployment
`sudo` commands.
