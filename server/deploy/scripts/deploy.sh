#!/usr/bin/env bash
#
# Build on the server, swap binaries, run under nohup, revert if unhealthy.
#
#   deploy.sh              pull, build, rotate current->previous, start, verify
#   deploy.sh --rollback   restore previous into current and restart
#   deploy.sh --status     pids + health
#   deploy.sh --start      start whatever is already in current
#   deploy.sh --stop       stop all three
#
# Layout:
#   ~/app/vnytros/            git clone of the monorepo; builds run in its server/
#   ~/release/current/        live binaries + .env.production + keys -> ~/keys
#   ~/release/previous/       the last set that ran, for reverting
#
# The processes run with their working directory set to ~/release/current,
# because the app reads BOTH .env.production and ./keys/*.pem relative to the
# working directory. That is why the deploy copies the env file in and links
# the keys dir on every rotation.

set -euo pipefail

# CLONE_DIR is the git clone of the monorepo; REPO_DIR is the Go module inside
# it (the server/ folder), where go.mod and ./cmd/* live.
CLONE_DIR="${CLONE_DIR:-$HOME/app/vnytros}"
REPO_DIR="${REPO_DIR:-$CLONE_DIR/server}"
RELEASE_ROOT="${RELEASE_ROOT:-$HOME/release}"
ENV_FILE="${ENV_FILE:-$HOME/.env.production}"
KEYS_DIR="${KEYS_DIR:-$HOME/keys}"
BRANCH="${BRANCH:-main}"

CURRENT="$RELEASE_ROOT/current"
PREVIOUS="$RELEASE_ROOT/previous"
STAGING="$RELEASE_ROOT/.staging"

BINARIES=(vnytros-server vnytros-mcp vnytros-worker)
HEALTH_ATTEMPTS="${HEALTH_ATTEMPTS:-10}"
HEALTH_DELAY="${HEALTH_DELAY:-3}"

export ENV=production

# go is expected on PATH. server-setup.sh guarantees that for every kind of
# shell by symlinking /usr/local/bin/go, which is on the default PATH even for
# a non-interactive `ssh host 'command'` session. No PATH fiddling here on
# purpose: if go is missing, the server is not set up, and papering over that
# in the deploy script would just hide it.
require_go() {
  command -v go >/dev/null 2>&1 || \
    die "go not on PATH. The server is not fully set up — run: bash ~/scripts/server-setup.sh"

  # go.mod pins a toolchain; an older Go fails with a confusing module error.
  local have want
  have="$(go env GOVERSION 2>/dev/null | sed 's/^go//')"
  want="$(awk '/^go /{print $2; exit}' "$REPO_DIR/go.mod" 2>/dev/null || echo 0)"
  if [ -n "$have" ] && [ -n "$want" ] && \
     [ "$(printf '%s\n%s\n' "$want" "$have" | sort -V | head -1)" != "$want" ]; then
    die "go $have is older than the $want required by go.mod — re-run ~/scripts/server-setup.sh"
  fi
  ok "using $(go version | awk '{print $3}') at $(command -v go)"
}

log()  { printf '\033[0;36m==>\033[0m %s\n' "$*"; }
ok()   { printf '\033[0;32m  ok\033[0m %s\n' "$*"; }
warn() { printf '\033[0;33m  !!\033[0m %s\n' "$*" >&2; }
die()  {
  printf '\033[0;31mFAILED\033[0m %s\n' "$*" >&2
  # An explicit exit does not fire the ERR trap, so route the recovery here too.
  if [ "${LIVE_WINDOW:-0}" = "1" ]; then
    LIVE_WINDOW=0
    trap - ERR
    recover_previous
  fi
  exit 1
}

# ---------------------------------------------------------------- process mgmt

start_all() {
  [ -d "$CURRENT" ] || die "no $CURRENT to start"
  cd "$CURRENT"

  for b in "${BINARIES[@]}"; do
    [ -x "./$b" ] || die "not executable: $CURRENT/$b"
    # nohup + setsid so the process survives this SSH session ending. Without
    # setsid, the shell's SIGHUP on disconnect can still reach it.
    setsid nohup "./$b" >> "$CURRENT/$b.log" 2>&1 < /dev/null &
    echo $! > "$CURRENT/$b.pid"
    ok "started $b (pid $(cat "$CURRENT/$b.pid"))"
  done
}

stop_all() {
  for dir in "$CURRENT" "$PREVIOUS"; do
    [ -d "$dir" ] || continue
    for b in "${BINARIES[@]}"; do
      local pf="$dir/$b.pid"
      [ -f "$pf" ] || continue
      local pid; pid="$(cat "$pf" 2>/dev/null || true)"
      if [ -n "$pid" ] && kill -0 "$pid" 2>/dev/null; then
        kill "$pid" 2>/dev/null || true
        for _ in $(seq 1 20); do kill -0 "$pid" 2>/dev/null || break; sleep 0.5; done
        kill -0 "$pid" 2>/dev/null && { warn "$b (pid $pid) ignored SIGTERM, killing"; kill -9 "$pid" 2>/dev/null || true; }
        ok "stopped $b (pid $pid)"
      fi
      rm -f "$pf"
    done
  done
  # Anything orphaned by a previous crash (stale or missing pidfile).
  for b in "${BINARIES[@]}"; do pkill -f "$RELEASE_ROOT/.*/$b" 2>/dev/null || true; done
}

pids_alive() {
  for b in "${BINARIES[@]}"; do
    local pf="$CURRENT/$b.pid" pid
    [ -f "$pf" ] || return 1
    pid="$(cat "$pf" 2>/dev/null || true)"
    [ -n "$pid" ] && kill -0 "$pid" 2>/dev/null || return 1
  done
  return 0
}

# Every process alive AND the two HTTP services answering. The worker has no
# port, so its pid is the only signal we get.
health_check() {
  for attempt in $(seq 1 "$HEALTH_ATTEMPTS"); do
    local rc=0
    pids_alive || rc=1
    if [ "$rc" -eq 0 ]; then
      curl -fs -o /dev/null --max-time 5 http://127.0.0.1:8000/           2>/dev/null || rc=1
      curl -fs -o /dev/null --max-time 5 http://127.0.0.1:8000/api/health 2>/dev/null || rc=1
      curl -fs -o /dev/null --max-time 5 http://127.0.0.1:5000/           2>/dev/null || rc=1
    fi
    if [ "$rc" -eq 0 ]; then
      ok "healthy on attempt $attempt (api :8000, mailforge /api/health, mcp :5000)"
      return 0
    fi
    [ "$attempt" -lt "$HEALTH_ATTEMPTS" ] && sleep "$HEALTH_DELAY"
  done

  warn "unhealthy after $HEALTH_ATTEMPTS attempts"
  for b in "${BINARIES[@]}"; do
    local pf="$CURRENT/$b.pid" pid=""
    [ -f "$pf" ] && pid="$(cat "$pf")"
    if [ -z "$pid" ] || ! kill -0 "$pid" 2>/dev/null; then
      warn "$b is NOT running — last lines of its log:"
      tail -n 15 "$CURRENT/$b.log" 2>/dev/null >&2 || true
    fi
  done
  return 1
}

# Same as furnish but never fatal — used on the recovery path, where dying
# again would abandon the box mid-restore.
furnish_quiet() {
  local dir="$1"
  [ -f "$ENV_FILE" ] && { cp "$ENV_FILE" "$dir/.env.production"; chmod 600 "$dir/.env.production"; }
  ln -sfn "$KEYS_DIR" "$dir/keys.tmp" 2>/dev/null && mv -Tf "$dir/keys.tmp" "$dir/keys" 2>/dev/null
  return 0
}

# Populate a release dir with the things the processes need beside them.
furnish() {
  local dir="$1"
  [ -f "$ENV_FILE" ] || die "missing $ENV_FILE — the app cannot start without it"
  cp "$ENV_FILE" "$dir/.env.production"
  chmod 600 "$dir/.env.production"
  # The app reads ./keys/*.pem from its working directory, so the keys have to
  # be reachable from inside the release dir. A symlink keeps one copy of the
  # private key on disk instead of one per release.
  ln -sfn "$KEYS_DIR" "$dir/keys.tmp" && mv -Tf "$dir/keys.tmp" "$dir/keys"
  [ -f "$dir/keys/private_key.pem" ] || warn "no private_key.pem under $KEYS_DIR — signing will fail"
}

LIVE_WINDOW=0

# Put the previous release back and start it. Must never call die(), or a
# failure here would recurse.
recover_previous() {
  warn "restoring the previous release"
  stop_all || true
  if [ ! -d "$PREVIOUS" ]; then
    warn "no previous release exists — nothing to restore, services are DOWN"
    return 1
  fi
  rm -rf "$CURRENT"
  cp -a "$PREVIOUS" "$CURRENT"
  furnish_quiet "$CURRENT"
  start_all || warn "could not start the restored release"
  if health_check; then
    warn "restored the previous release — the new one was NOT deployed"
  else
    warn "restored the previous release but it is still unhealthy"
  fi
}

# Fires on any unhandled non-zero command between rotation and a healthy check.
on_live_failure() {
  local rc=$?
  trap - ERR
  [ "$LIVE_WINDOW" = "1" ] || exit "$rc"
  LIVE_WINDOW=0
  warn "deploy failed mid-swap (exit $rc)"
  recover_previous || true
  exit "$rc"
}

# ---------------------------------------------------------------- commands

# Everything that can be checked without touching the running system, checked
# before we touch the running system. The first version validated the env file
# inside furnish(), i.e. AFTER stopping the services and rotating current into
# previous — so a missing file left the box down with no revert. Anything that
# can fail cheaply must fail here.
preflight() {
  [ -d "$CLONE_DIR/.git" ] || die "no git clone at $CLONE_DIR — see server/deploy/README.md Step 3"
  [ -f "$REPO_DIR/go.mod" ] || die "no go.mod at $REPO_DIR — expected the server/ folder of the monorepo clone"

  [ -f "$ENV_FILE" ] || die "missing $ENV_FILE
  The app reads .env.production from its working directory, and this file is
  the canonical copy that gets placed into each release. Put it there once:
      scp -i <your-key> .env.production <user>@<SERVER_IP>:~/
      ssh ... 'chmod 600 ~/.env.production'"
  [ -r "$ENV_FILE" ] || die "$ENV_FILE is not readable by $(id -un)"

  [ -f "$KEYS_DIR/private_key.pem" ] || die "missing $KEYS_DIR/private_key.pem
  Upload the signing keys from your laptop:  make upload-keys"
  [ -f "$KEYS_DIR/public_key.pem" ]  || die "missing $KEYS_DIR/public_key.pem"

  ok "preflight: repo, $ENV_FILE and keys all present"
}

cmd_deploy() {
  preflight

  log "fetching $BRANCH"
  cd "$CLONE_DIR"
  git fetch --quiet origin "$BRANCH"
  # reset rather than pull: a deploy target should match the remote exactly and
  # never stop on a local conflict.
  git reset --hard --quiet "origin/$BRANCH"
  ok "at $(git rev-parse --short HEAD) $(git log -1 --pretty=%s | cut -c1-60)"
  cd "$REPO_DIR"

  require_go
  log "building three binaries (ENV=production)"
  rm -rf "$STAGING"; mkdir -p "$STAGING"
  for t in server mcp worker; do
    # Build BEFORE touching anything live: a compile error must not take the
    # running services down.
    if ! CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" \
         -o "$STAGING/vnytros-$t" "./cmd/$t"; then
      rm -rf "$STAGING"
      die "build failed for cmd/$t — nothing was changed, services still running"
    fi
    ok "built vnytros-$t"
  done

  # From here the running system is being modified. Any unexpected failure —
  # not just an unhealthy health check — has to put the old release back.
  LIVE_WINDOW=1
  trap 'on_live_failure' ERR

  log "rotating current -> previous"
  stop_all
  if [ -d "$CURRENT" ]; then
    rm -rf "$PREVIOUS"
    mv "$CURRENT" "$PREVIOUS"
    ok "old binaries kept in $PREVIOUS"
  fi
  mv "$STAGING" "$CURRENT"
  furnish "$CURRENT"

  log "starting"
  start_all

  if health_check; then
    LIVE_WINDOW=0
    trap - ERR
    ok "deployed $(cd "$REPO_DIR" && git rev-parse --short HEAD)"
    return 0
  fi

  LIVE_WINDOW=0
  trap - ERR

  warn "new build is unhealthy — reverting to previous"
  stop_all
  [ -d "$PREVIOUS" ] || die "no previous release to revert to; services are DOWN"

  rm -rf "$CURRENT"
  # copy, not move, so previous survives for a second attempt
  cp -a "$PREVIOUS" "$CURRENT"
  furnish "$CURRENT"
  start_all

  if health_check; then
    warn "reverted — the new build was NOT deployed"
    exit 1
  fi
  die "revert is also unhealthy — check the database, broker and $ENV_FILE"
}

cmd_rollback() {
  [ -d "$PREVIOUS" ] || die "no previous release to roll back to"
  log "rolling back"
  stop_all
  rm -rf "$CURRENT"
  cp -a "$PREVIOUS" "$CURRENT"
  furnish "$CURRENT"
  start_all
  health_check && ok "rollback complete" || die "rolled back but still unhealthy"
}

cmd_status() {
  echo "repo     : $CLONE_DIR ($(cd "$CLONE_DIR" 2>/dev/null && git rev-parse --short HEAD 2>/dev/null || echo '?')), building $REPO_DIR"
  echo "current  : $([ -d "$CURRENT" ] && echo present || echo missing)"
  echo "previous : $([ -d "$PREVIOUS" ] && echo present || echo missing)"
  echo
  for b in "${BINARIES[@]}"; do
    local pid="-" state="stopped"
    if [ -f "$CURRENT/$b.pid" ]; then
      pid="$(cat "$CURRENT/$b.pid")"
      kill -0 "$pid" 2>/dev/null && state="running" || { state="dead"; pid="$pid (stale)"; }
    fi
    printf '%-16s %-9s pid %s\n' "$b" "$state" "$pid"
  done
  echo
  if health_check; then echo "overall  : HEALTHY"; else echo "overall  : UNHEALTHY"; fi
}

main() {
  mkdir -p "$RELEASE_ROOT"
  case "${1:-}" in
    "")         cmd_deploy ;;
    --rollback) cmd_rollback ;;
    --status)   cmd_status ;;
    --start)    furnish "$CURRENT"; start_all; health_check ;;
    --stop)     stop_all ;;
    -h|--help)  sed -n '2,20p' "$0" | sed 's/^# \{0,1\}//'; exit 0 ;;
    *)          die "unknown option: $1 (try --help)" ;;
  esac
}

main "$@"
