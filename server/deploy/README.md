# Self-hosting on a single server

A reusable guide for running Vnytros on one Linux server you control (any
cloud VM or bare-metal box running Ubuntu 22.04+/Debian works; the examples
assume Ubuntu). There is no hosted Vnytros service — you bring your own
domain, server, database, SMTP account and credentials.

This guide lives in the `server/` folder of the
[Vnytros monorepo](https://github.com/Aadesh998/vnytros-domain-connect). Every `make` command
below is run from `server/` on your laptop (`cd vnytros-domain-connect/server`), and paths
such as `deploy/` and `.env.example` are relative to it.

Throughout, replace the placeholders:

| Placeholder | Meaning |
|---|---|
| `<your-domain>` / `DOMAIN` | the domain you own, e.g. `example.com` |
| `<SERVER_IP>` | your server's public IP address |
| `<user>` | the non-root login user on the server (e.g. `ubuntu`) |
| `<path/to/key>` | the SSH private key for that user |

The three Go binaries are built **on the server** from a git clone of the
monorepo (inside its `server/` folder) and run as
plain processes (`setsid nohup`, restarted at boot by an `@reboot` cron entry);
everything else runs in Docker; nginx terminates TLS in front.

```
                                ┌───────────────────────────────────────────┐
  api.<your-domain>  ─────────► │ nginx :443  (TLS via certbot)             │
  mcp.<your-domain>  ─────────► │   ├─► 127.0.0.1:8000  vnytros-server      │
  grafana.<your-domain> (opt.)► │   │        (api /v1 + oauth + mail /api)  │
  rabbitmq.<your-domain> (opt.)►│   └─► 127.0.0.1:5000  vnytros-mcp         │
                                │        vnytros-worker (no port)           │
                                │                                           │
                                │   docker compose (deploy/…infra.yaml)     │
                                │     rabbitmq · otel-collector · tempo     │
                                │     prometheus · grafana · node-exporter  │
                                └───────────────────────────────────────────┘
                                                 │
                                   PostgreSQL (managed or another host)
```

Filesystem layout on the server (in the service user's home):

```
~/
├── app/vnytros/            git clone of the monorepo; deploy.sh builds in app/vnytros/server
├── release/
│   ├── current/            live binaries + .env.production + keys -> ~/keys
│   └── previous/           the last set that ran, for rollback
├── scripts/                deploy.sh, server-setup.sh, nginx-install.sh
├── keys/                   Domain Connect keypair (make keys)
├── deploy/                 compose, nginx templates, observability (rsynced)
│   └── .env                infra credentials (from infra.env.example)
└── .env.production         app config (chmod 600), copied into each release
```

The processes run with their working directory set to `~/release/current`
because the app reads both `.env.production` and `./keys/*.pem` relative to
the working directory.

## Step 1 — Server

Any VM with **2 vCPU / 4 GB RAM** and ~30 GB disk is comfortable: Prometheus,
Tempo, Grafana and RabbitMQ together want ~1.5 GB before the binaries load.
Give it a static public IP.

Firewall / security group inbound rules:

| Port | Source | Why |
|---|---|---|
| 443 | anywhere | HTTPS |
| 80 | anywhere | ACME HTTP-01 challenge + the HTTPS redirect |
| 22 | **your IP only** (see "Deploy access") | admin access |

Nothing else. Do **not** open 5672, 15672, 3000, 9090, 9101-9103 or 8000 —
every one of those is bound to `127.0.0.1` by design.

## Step 2 — DNS

Create A (and AAAA, if you have IPv6) records pointing at `<SERVER_IP>`:

| Name | Serves |
|---|---|
| `api.<your-domain>` | API `/v1`, OAuth `/oauth`, Domain Connect discovery, mail `/api` → :8000 |
| `mcp.<your-domain>` | MCP server → :5000 |
| `grafana.<your-domain>` | optional: dashboards → :3000 |
| `rabbitmq.<your-domain>` | optional: broker console → :15672 |

The hostnames are only a convention of the nginx templates; any names work if
you edit the templates. The dashboard (`dashboard/`, any static host), the
project website (`website/`) and the docs (`docs/`) are hosted separately and
are not covered here; see the deployment section of the root
[README](../../README.md#deploying-each-part).

**Prometheus** (:9090) and **Tempo** (:3200) have no authentication and get no
hostname. Grafana is the front door for both; use an SSH tunnel otherwise.

### If your DNS is on Cloudflare

Keep the records **DNS only** (grey cloud) unless you know you want the proxy.
Proxying breaks certbot's HTTP-01 challenge (use DNS-01 or a Cloudflare Origin
Certificate instead), and makes every client IP Cloudflare's — which is why
`nginx/conf.d/00-shared.conf` ships Cloudflare's published IP ranges with
`real_ip_header CF-Connecting-IP`. Those lines only matter behind the
Cloudflare proxy; elsewhere they are inert and can be deleted.

## Step 3 — Prepare the server

From your laptop, push the deploy files and scripts (every remote `make`
target needs `EC2_HOST` and `EC2_KEY`; they have no defaults):

```bash
export EC2_HOST=<SERVER_IP> EC2_KEY=<path/to/key> EC2_USER=<user>
make sync-deploy      # deploy/          -> ~/deploy
make sync-scripts     # deploy/scripts/  -> ~/scripts
```

Then on the server, once:

```bash
DOMAIN=<your-domain> bash ~/scripts/server-setup.sh
```

It installs Go (the version `go.mod` requires), Docker + Compose, nginx,
certbot and `envsubst`; creates the directory layout; registers the `@reboot`
restart; clones the monorepo to `~/app/vnytros` (`REPO_URL=` to build your
fork instead, `CLONE_DIR=` for another location); and,
when `DOMAIN` is set, renders and installs the nginx config. Safe to re-run.

Go is symlinked into `/usr/local/bin`, because a non-interactive
`ssh host 'command'` reads no dotfiles; the setup script verifies `go` resolves
on a bare PATH before finishing.

## Step 4 — App config

Start from the template and fill in values for **your** deployment:

```bash
cp .env.example .env.production   # on your laptop
$EDITOR .env.production
make sync-env                     # -> ~/.env.production, chmod 600
```

For production, at minimum:

```bash
ENV=production
BASE_URL=https://api.<your-domain>
OAUTH_ISSUER_URL=https://api.<your-domain>
MCP_SERVER_BASE_URL=https://mcp.<your-domain>
FRONTEND_URL=https://<your-domain>
DASHBOARD_URL=https://app.<your-domain>          # wherever you host the dashboard
ALLOWED_ORIGINS=https://<your-domain>,https://app.<your-domain>
AUTH_COOKIE_DOMAIN=.<your-domain>
AUTH_COOKIE_SECURE=true
RABITMQ=amqp://<rabbit-user>:<rabbit-pass>@127.0.0.1:5672/
OTEL_ENDPOINT=localhost:4318
# plus DB_*, JWT_SECRET, APIKEY_SIGNING_SECRET, WEBHOOK_SIGNING_SECRET, SMTP_*
```

With `ENV=production` the server refuses to start if `JWT_SECRET`,
`APIKEY_SIGNING_SECRET`, `BASE_URL`, `FRONTEND_URL` or `ALLOWED_ORIGINS` is
empty. `.env.example` documents every variable.

## Step 5 — Signing keys

Generate a fresh Domain Connect keypair locally and ship it:

```bash
make keys          # writes keys/private_key.pem + keys/public_key.pem (gitignored)
make upload-keys   # -> ~/keys, 700/600 permissions
```

Never reuse a key from another deployment. The public key is served at
`/_domainconnect/_dckeypubv1.public`.

## Step 6 — Infrastructure (broker + observability)

On the server:

```bash
cd ~/deploy
cp infra.env.example .env && chmod 600 .env
$EDITOR .env      # RABBITMQ_DEFAULT_USER/PASS (must match RABITMQ above),
                  # GRAFANA_ADMIN_PASSWORD, GRAFANA_ROOT_URL
docker compose -f docker-compose.infra.yaml up -d
docker compose -f docker-compose.infra.yaml ps
```

Compose refuses to start without `RABBITMQ_DEFAULT_PASS` and
`GRAFANA_ADMIN_PASSWORD`; no credential is stored in the repository. Check:

```bash
curl -fsS -o /dev/null -w '%{http_code}\n' localhost:15672        # 200, broker UI
curl -fsS -o /dev/null -w '%{http_code}\n' localhost:3000/login   # 200, grafana
curl -fsS -o /dev/null -w '%{http_code}\n' localhost:9090/-/ready # 200, prometheus
```

## Step 7 — nginx and TLS

The nginx config is a set of **envsubst templates** in `nginx/conf.d/`:

```
nginx/conf.d/00-shared.conf            upstreams, websocket map, Cloudflare real-IP, catch-all
nginx/conf.d/api.conf.template         server_name api.${DOMAIN}
nginx/conf.d/mcp.conf.template         server_name mcp.${DOMAIN}
nginx/conf.d/grafana.conf.template     server_name grafana.${DOMAIN}   (optional)
nginx/conf.d/rabbitmq.conf.template    server_name rabbitmq.${DOMAIN}  (optional)
```

`deploy/scripts/nginx-install.sh` renders them with
`envsubst '${DOMAIN}'` — only `${DOMAIN}` is replaced, so nginx's own
`$host`, `$remote_addr` etc. survive — into `/etc/nginx/conf.d/<name>.conf`,
then tests and reloads nginx:

```bash
make nginx-install DOMAIN=<your-domain>
# or on the server; HOSTS limits which hosts are installed:
DOMAIN=<your-domain> HOSTS="api mcp" bash ~/scripts/nginx-install.sh
```

Then issue certificates (the hostnames must already resolve to the server and
port 80 must be open):

```bash
make certs DOMAIN=<your-domain>
# equivalently, on the server:
sudo certbot --nginx -d api.<your-domain> -d mcp.<your-domain> \
  -d grafana.<your-domain> -d rabbitmq.<your-domain>
```

certbot edits the installed files in place to add the 443 blocks and
redirects. Re-running `nginx-install` overwrites those edits, so run certbot
again afterwards (it reuses valid certificates).

Things in these files that exist for a reason:

- **`client_max_body_size 25m`** on api — campaign sends upload a recipient
  CSV; nginx's 1 MB default would reject real lists with a 413.
- **`proxy_buffering off`** on mcp — the handler streams; buffering during a
  long tool call is indistinguishable from a hang.
- **Commented `allow`/`deny`** on grafana and rabbitmq — as shipped only their
  logins protect them. Uncomment and set your admin IP, put other auth in
  front, or skip those hosts (`HOSTS="api mcp"`) and use an SSH tunnel. The
  RabbitMQ console is broker administration: whoever opens it can purge queues
  and read message payloads.
- **The `return 444` catch-all** — without it nginx answers an unknown `Host`
  from whichever server block comes first.
- **`X-Forwarded-For $remote_addr`** — overwritten, not appended, so a client
  cannot forge its IP for rate limiting.

## Step 8 — Migrate, then deploy

Schema changes are a deliberate manual step. Run the migrator against your
production database (from a machine that can reach it, with the production
env loaded):

```bash
ENV_FILE=.env.production make migration
```

It creates or updates the schema and upserts the DNS provider reference data
used by provider detection. Every step is idempotent, so run it again after
each upgrade. There is no separate seed step and no default accounts.

Then deploy, on the server:

```bash
~/scripts/deploy.sh
```

It fetches the configured branch (`BRANCH`, default `main`) in
`~/app/vnytros`, builds the three
static binaries, rotates `current` → `previous`, starts them, and health-checks
`:8000/`, `:8000/api/health` and `:5000/`. **If they never go healthy it puts
the previous release back** and exits non-zero.

From your laptop:

```bash
make status      # what is live, and is it healthy
make rollback    # go back to the previous release
make logs        # follow vnytros-server; SVC=vnytros-worker for another
```

Verify from outside:

```bash
curl -fsS https://api.<your-domain>/ ; echo
curl -fsS https://api.<your-domain>/api/health ; echo
curl -fsSI https://mcp.<your-domain>/ | head -1
```

### Optional: deploy from GitHub Actions

`.github/workflows/deploy.yml` (at the monorepo root) is **manual only** (`workflow_dispatch`) and
fully secret-driven. In your fork, create a dedicated deploy key on your
laptop, authorise it on the server, and add four repository secrets:

```bash
ssh-keygen -t ed25519 -f ~/.ssh/vnytros_deploy -C "github-actions-deploy" -N ""
ssh -i <path/to/key> <user>@<SERVER_IP> \
  "echo '$(cat ~/.ssh/vnytros_deploy.pub)' >> ~/.ssh/authorized_keys"
make known-hosts EC2_HOST=<SERVER_IP>     # output -> DEPLOY_SSH_KNOWN_HOSTS
```

| Secret | Value |
|---|---|
| `DEPLOY_SSH_HOST` | `<SERVER_IP>` |
| `DEPLOY_SSH_USER` | `<user>` |
| `DEPLOY_SSH_KEY` | contents of `~/.ssh/vnytros_deploy` (private half) |
| `DEPLOY_SSH_KNOWN_HOSTS` | the `ssh-keyscan` output above |

Optionally add a `production` environment with yourself as required reviewer.
Then **Actions → deploy → Run workflow**. CI (`.github/workflows/server.yml`:
build, vet, test) runs on every push and pull request that touches `server/`
and needs no secrets.

---

## Observability

Instrumented at the code level, not just deployed. `internal/telemetry` is the
single place tracing and metrics are configured; each binary calls `Init` and
`Serve` once at startup.

**Traces** — OTLP/HTTP to the collector, then Tempo, viewable in Grafana Explore.
Verified end to end: a request produced a span in Tempo reading
`service.name=vnytros-server`, named `GET <chi route pattern>`, kind SERVER, with
`http.route`, `http.request.method` and `http.response.status_code` set.

| Service | `service.name` | What is traced |
|---|---|---|
| `cmd/server` | `vnytros-server` | every `/v1` and `/oauth` route (otelhttp, span named by chi route pattern) plus mailforge's `/api` routes (otelgin) |
| `cmd/mcp` | `vnytros-mcp` | every MCP request, including long tool calls |
| `cmd/worker` | `vnytros-worker` | one consumer span per job, with queue name, body size and outcome; errors recorded on the span |

The service name used to be hardcoded `go-backend` for everything; it is now per
binary, which is what makes the three separable in Tempo. Spans also carry
`deployment.environment` from `ENV`.

**Metrics** — each binary serves `/metrics` on its own loopback port, scraped by
Prometheus:

| Port | Service |
|---|---|
| 9101 | `vnytros-server` |
| 9102 | `vnytros-mcp` |
| 9103 | `vnytros-worker` — no HTTP server of its own, so this is the only outside view of it |

Application metrics:

- `vnytros_http_requests_total{route,method,status}` and
  `vnytros_http_request_duration_seconds{route,method}` — labelled by **chi route
  pattern**, so a route with an `{id}` segment is one series rather than one per id.
- `vnytros_jobs_total{queue,outcome}` where outcome is `ack`, `dlq`, `requeue`
  or `panic` — the four branches of the worker's dispatch. A rise in `requeue`
  is retry churn you can actually alert on.
- `vnytros_job_duration_seconds{queue}`.
- `go_*` and `process_*` (goroutines, heap, GC, fds) come free with the default
  registry.

Span-derived RED metrics: the collector's `spanmetrics` connector turns spans
into `traces_span_metrics_calls_total` and
`traces_span_metrics_duration_milliseconds`, dimensioned by `http_route`,
`http_request_method` and `http_response_status_code`. This is not a duplicate
of the counters above — it is the **only per-endpoint breakdown for mailforge's
`/api` routes**, because the chi middleware sees them all as the single mounted
pattern `/api/*` while otelgin names each one. Four panels on the API dashboard
read it.

`http.route` is set by `telemetry.routeAttribute`, the innermost of the three
HTTP middlewares. chi only fills in `RoutePattern()` during routing, which
happens after the middleware chain, so the attribute can only be read on the way
back out — and only a middleware *inside* the otelhttp one still has an open
span to write it to. Get that order wrong and the connector's `http_route`
dimension is silently empty. `internal/server/apitrace_test.go` guards it.

Infrastructure metrics: RabbitMQ queue depth and consumer counts
(`rabbitmq_prometheus` on 15692 — the ones that tell you whether campaigns are
draining), host CPU/memory/disk via node-exporter, and RED metrics derived from
spans by the collector's `spanmetrics` connector as a cross-check.

### Two deliberate networking choices

**Prometheus and Grafana run on the host network.** The Go services run
directly on the host and bind loopback; a bridged container cannot reach that.
The usual `host.docker.internal` workaround depends on the host firewall
permitting bridge→host traffic, which it frequently does not — verified failing
during setup. On the host network every target is simply `127.0.0.1`, and the
app's `/metrics` can stay bound to loopback rather than `0.0.0.0`.

**Targets are written `127.0.0.1`, never `localhost`.** `localhost` resolves to
`::1` first on many hosts, and the Go listeners bind IPv4, so every scrape
failed with `dial tcp [::1]: connection refused`. Same reason the Grafana
datasource URLs are IPv4.

Neither Prometheus (`127.0.0.1:9090`) nor Grafana (`GF_SERVER_HTTP_ADDR`) is
exposed: host networking offers no port mapping, so both bind loopback
explicitly. Reach Grafana by tunnelling:

```bash
ssh -i <path/to/key> -L 3000:localhost:3000 <user>@<SERVER_IP>
```

Then open `http://localhost:3000` (user `admin`, password `GRAFANA_ADMIN_PASSWORD`
from `~/deploy/.env`). It can also have its own hostname — see
`nginx/conf.d/grafana.conf.template` — but the tunnel needs no exposure at all.

### Dashboards

Provisioned from `observability/grafana/dashboards/*.json` into a **Vnytros**
folder, so they exist the moment Grafana boots — nothing to import by hand.
`allowUiUpdates: false` keeps the files authoritative: edit the JSON, re-run
`make sync-deploy`, then `docker compose -f docker-compose.infra.yaml restart grafana` in `~/deploy`.

| Dashboard | Answers |
|---|---|
| **Overview** | Is anything broken right now? Liveness of all three services + broker, 5xx rate, queue backlog, host memory — then drill down |
| **API & MCP** | Request rate and latency percentiles by chi route, status breakdown, 4xx/5xx per route, slowest routes table, and Go runtime (goroutines, heap, GC, fds) for all three binaries |
| **Worker & Queues** | Jobs/sec by outcome (ack/dlq/requeue/panic), duration p95 per queue, failure ratio, and per-queue depth, unacked and consumer counts from the broker |
| **Host** | CPU by mode, memory, disk space and I/O, network — the server's headroom |

Reach them at `https://grafana.<your-domain>` (if you installed that host) or over the tunnel.

### Alerts

`observability/grafana/provisioning/alerting/rules.yaml` provisions 12 rules:
each service down, broker down, 5xx over 5%, p95 over 2s, queue backlog over
5000, any DLQ or panic, requeue churn, memory over 92%, disk over 90%. Each
carries a summary naming the command to run next.

**They will not reach you until you add a contact point.** Grafana's default
notification policy only records firing alerts in the UI (Alerting → Alert
rules). Add an email/Slack destination under Alerting → Contact points and point
the default policy at it. Left unset deliberately — a wrong destination fails
silently, which is worse than an obvious gap.

Thresholds are starting points. Tune them after a week of real traffic; an alert
that cries wolf gets muted, and a muted alert is worse than none.

### A note on per-queue metrics

RabbitMQ's default `/metrics` is **aggregated only** — verified against the
running broker, it emits zero series carrying a `queue` label. Per-queue depth
therefore comes from a second scrape job (`rabbitmq-queues`) hitting
`/metrics/detailed` with the `queue_coarse_metrics` and `queue_consumer_count`
families, and those series are named `rabbitmq_detailed_queue_*`. Without that
job every per-queue panel would be silently empty.

### Sanity checks after deploying

```bash
curl -s 127.0.0.1:9101/metrics | grep -c vnytros_http     # api metrics exist
curl -s 127.0.0.1:9103/metrics | grep vnytros_jobs_total   # worker metrics
curl -s 127.0.0.1:9090/api/v1/targets | jq '.data.activeTargets[]|{job:.labels.job,health}'
```

All eight Prometheus targets should report `up`. Traces appear in Grafana →
Explore → Tempo once traffic flows; `service.name` distinguishes the binaries.

### Database queries

`internal/db` installs the GORM OpenTelemetry plugin, so every query on the
shared pool is traced — one pool serves `cmd/server` (api and mailforge),
`cmd/mcp` and `cmd/worker`, so that is the only place it needs installing.

`WithoutQueryVariables()` is set deliberately: by default the plugin inlines
bound parameters into `db.statement`, which would ship SMTP passwords
(`mail_smtp_configs.Password`), API keys and password hashes to the trace
backend in plain text. The statement shape is what helps debugging; the values
are not worth the exposure. `internal/db/tracing_test.go` guards this.

A query span only attaches to the request that caused it if the caller passes a
context (`db.WithContext(ctx)`). mailforge's repositories do this throughout, so
its queries appear as children of the HTTP span. The api's repositories do not
take a `context.Context` at all, so their queries currently emit **orphan root
spans** — the latency is recorded, but not linked to the request. Closing that
means threading a context through the api's repository and service layers; see
"What is still not instrumented".

### What is still not instrumented

The api's repository layer takes no `context.Context` (55 methods across 9
files), so its database spans are orphans as described above. Threading context
through the repository and service layers would fix it — and needs care, because
some work is deliberately launched on `context.Background()` so it outlives the
request (webhook dispatch, domain verification enqueue); handing those a request
context would cancel them when the response is written.

Ports 9101-9103 must stay **closed in the security group** — `/metrics` exposes
route names, queue names and latency profiles. They are loopback-bound, so this
is defence in depth rather than the only control.

Database query spans exist only for mailforge (its GORM plugin). The api's and
worker's own GORM calls are untraced, so a slow query on `/v1` shows as time
inside the request span without a child span naming the statement. Adding the
same plugin to `internal/db` would close that.

## Day-to-day operations

```bash
# what is running, and is it healthy
~/scripts/deploy.sh --status

# logs, live (each binary appends to its own file in the release dir)
tail -f ~/release/current/vnytros-server.log
tail -f ~/release/current/vnytros-worker.log

# restart / stop / roll back
~/scripts/deploy.sh --start
~/scripts/deploy.sh --stop
~/scripts/deploy.sh --rollback

# proxy
sudo nginx -t && sudo systemctl reload nginx
tail -f /var/log/nginx/api.error.log

# infra
cd ~/deploy && docker compose -f docker-compose.infra.yaml ps
docker compose -f ~/deploy/docker-compose.infra.yaml logs -f rabbitmq
```

To apply changes under `deploy/` later: `make sync-deploy`, then on the server
`docker compose -f docker-compose.infra.yaml up -d` (compose/observability) or
`DOMAIN=<your-domain> bash ~/scripts/nginx-install.sh` followed by certbot
(nginx).

## Secrets

- The root `.gitignore` covers `.env`, `.env.*` (except `.env.example`), `keys/`,
  `*.pem`, `*.log` and `bin/`. Never commit real values.
- Generate every secret yourself (`openssl rand -hex 32`), and a fresh keypair
  with `make keys`. Never reuse values from another deployment, from examples,
  or from git history.
- If a secret is ever committed, rotate it — removing it from the tree does not
  remove it from history.

## Deploy access: the honest caveat

The optional GitHub workflow SSHes in, so port 22 must accept connections from
GitHub's runners, which use large, changing IP ranges. Options, least to most
work:

1. **Open 22 to the world** with key-only auth (`PasswordAuthentication no`)
   plus `fail2ban`. Common; the key is the only thing standing in the way.
2. **Restrict to GitHub's ranges**, refreshed from `https://api.github.com/meta`
   (`.actions[]`). Narrower, but it is hundreds of CIDRs and they change.
3. **No inbound SSH at all**: deploy by hand over a VPN/allowlisted IP, or use
   your cloud's agent-based remote command service (e.g. AWS SSM with GitHub
   OIDC) to run `~/scripts/deploy.sh`.

Or skip the workflow entirely and run `~/scripts/deploy.sh` yourself.
