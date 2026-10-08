#!/usr/bin/env bash
#
# One-time server preparation. Run ON the server (Ubuntu/Debian) as the
# non-root user that will run the services:
#
#   DOMAIN=example.com bash ~/scripts/server-setup.sh
#
# DOMAIN is optional; when set, the nginx templates are rendered for it.
# REPO_URL overrides the git repository to build from (your fork, say). It is
# the whole monorepo; the Go module that gets built lives in its server/ folder.
#
# Installs Go, docker, nginx and certbot, creates the directory layout, and
# registers a @reboot job so the services come back after a restart. Does not
# issue certificates or start anything — those need DNS and ~/.env.production.
# Safe to re-run.

set -euo pipefail

GO_VERSION="1.26.2"
REPO_URL="${REPO_URL:-https://github.com/vnytros/vnytros.git}"
# Where the monorepo is cloned. deploy.sh builds from $CLONE_DIR/server.
CLONE_DIR="${CLONE_DIR:-$HOME/app/vnytros}"

log() { printf '\033[0;36m==>\033[0m %s\n' "$*"; }
ok()  { printf '\033[0;32m  ok\033[0m %s\n' "$*"; }

[ "$(id -u)" != "0" ] || { echo "run as the (non-root) user that will run the services, not root" >&2; exit 1; }

log "directories"
mkdir -p ~/app ~/release ~/scripts ~/keys ~/deploy
sudo mkdir -p /var/www/certbot
ok "~/app ~/release ~/scripts ~/keys ~/deploy"

log "packages"
sudo apt-get update -qq
sudo apt-get install -y -qq ca-certificates curl gnupg git jq gettext-base

# Go, from the official tarball. Ubuntu's apt package is far behind and go.mod
# requires 1.26.2 — the build simply refuses to run on an older toolchain.
if ! /usr/local/go/bin/go version 2>/dev/null | grep -q "go$GO_VERSION"; then
  log "installing Go $GO_VERSION"
  arch=$(dpkg --print-architecture)   # amd64 or arm64
  curl -fsSL "https://go.dev/dl/go${GO_VERSION}.linux-${arch}.tar.gz" -o /tmp/go.tgz
  sudo rm -rf /usr/local/go
  sudo tar -C /usr/local -xzf /tmp/go.tgz
  rm -f /tmp/go.tgz
fi
# Make go reachable from EVERY kind of shell, which is the whole point.
#
# `ssh host 'command'` runs a non-interactive, non-login shell: it reads neither
# ~/.profile nor ~/.bashrc, so anything that only edits those is invisible to
# the deploy. /usr/local/bin IS on the default PATH that PAM hands such a
# session, so a symlink there is what actually fixes it everywhere.
#
# Go resolves GOROOT from the real path of the binary, so symlinking rather
# than copying is correct and keeps upgrades to one tarball extraction.
sudo ln -sfn /usr/local/go/bin/go    /usr/local/bin/go
sudo ln -sfn /usr/local/go/bin/gofmt /usr/local/bin/gofmt

# Interactive/login shells additionally get GOPATH's bin, for convenience only.
sudo tee /etc/profile.d/go.sh > /dev/null <<'PROFILE'
export PATH=$PATH:/usr/local/go/bin:$HOME/go/bin
PROFILE
sudo chmod 644 /etc/profile.d/go.sh

export PATH=$PATH:/usr/local/go/bin
ok "$(go version)"
# Prove it: run with the bare PATH a non-interactive ssh session gets, with no
# profile or rc file in play. If this cannot find go, neither can the deploy.
if env -i PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin \
     sh -c 'command -v go >/dev/null'; then
  ok "go resolves on a bare non-interactive PATH — the deploy will find it"
else
  echo "  !! go still not on the non-interactive PATH; check /usr/local/bin/go" >&2
fi

if ! command -v docker >/dev/null; then
  log "installing docker"
  sudo install -m 0755 -d /etc/apt/keyrings
  curl -fsSL https://download.docker.com/linux/ubuntu/gpg \
    | sudo gpg --dearmor -o /etc/apt/keyrings/docker.gpg
  echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] \
https://download.docker.com/linux/ubuntu $(lsb_release -cs) stable" \
    | sudo tee /etc/apt/sources.list.d/docker.list > /dev/null
  sudo apt-get update -qq
  sudo apt-get install -y -qq docker-ce docker-ce-cli containerd.io docker-compose-plugin
fi
sudo usermod -aG docker "$(id -un)"
ok "docker $(docker --version | awk '{print $3}' | tr -d ,)"

sudo apt-get install -y -qq nginx certbot python3-certbot-nginx
ok "nginx + certbot"

# nohup processes do NOT survive a reboot, and there is no systemd unit in this
# setup. This @reboot entry restarts whatever is in ~/release/current.
log "@reboot job"
if ! crontab -l 2>/dev/null | grep -q 'deploy.sh --start'; then
  ( crontab -l 2>/dev/null; echo "@reboot sleep 30 && $HOME/scripts/deploy.sh --start >> $HOME/release/reboot.log 2>&1" ) | crontab -
fi
ok "services will restart after a reboot"

if [ ! -d "$CLONE_DIR/.git" ]; then
  log "cloning $REPO_URL"
  git clone --quiet "$REPO_URL" "$CLONE_DIR"
fi
[ -f "$CLONE_DIR/server/go.mod" ] || { echo "no server/go.mod in $CLONE_DIR — is REPO_URL the Vnytros monorepo?" >&2; exit 1; }
ok "repo at $(git -C "$CLONE_DIR" rev-parse --short HEAD 2>/dev/null || echo 'unknown') (builds from $CLONE_DIR/server)"

if [ -n "${DOMAIN:-}" ] && [ -d ~/deploy/nginx ]; then
  log "nginx config for $DOMAIN"
  DOMAIN="$DOMAIN" bash ~/scripts/nginx-install.sh
fi

cat <<'NEXT'

Next, in order:
  (Run the make commands below from the server/ folder of your local clone.)
  1. Put the app config on the box (chmod 600), from your laptop:
       make sync-env EC2_HOST=<SERVER_IP> EC2_KEY=<path/to/key>
     or copy .env.production to ~/.env.production by any means you like.
  2. make upload-keys EC2_HOST=<SERVER_IP> EC2_KEY=<path/to/key>
  3. cp ~/deploy/infra.env.example ~/deploy/.env   # then set real passwords
     cd ~/deploy && docker compose -f docker-compose.infra.yaml up -d
  4. nginx (if you did not pass DOMAIN above) and certificates:
       DOMAIN=<your-domain> bash ~/scripts/nginx-install.sh
       sudo certbot --nginx -d api.<your-domain> -d mcp.<your-domain> \
         -d grafana.<your-domain> -d rabbitmq.<your-domain>
  5. ~/scripts/deploy.sh   (or run the manual GitHub workflow)

NEXT
