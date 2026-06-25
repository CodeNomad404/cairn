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
