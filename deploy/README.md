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

The script does not inspect, commit, or push the local Git repository. It only
connects to the server. The server clones or updates `main` in its source cache
at `/opt/cairn-repo`, downloads dependencies, tests, builds, backs up PostgreSQL,
runs forward migrations, installs the new binary, restarts the service, and
checks `/healthz`. No source files are uploaded from the local machine. If the
health check fails, it restores the previous binary. Applied database migrations
are deliberately not rolled back.

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
