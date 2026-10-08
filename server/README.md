# Vnytros server

Vnytros is a free, open-source DNS and email-authentication toolkit with a
built-in mail platform. This folder, `server/` in the
[Vnytros monorepo](https://github.com/vnytros/vnytros), is the Go backend: the
REST API, the OAuth server for MCP clients, the MCP server, the mail platform
(mailforge) and the background worker. Every command below runs from
`server/`.

**This is self-hosted software. No hosted service is provided** — there is no
public Vnytros API, MCP server, mail platform or account to sign up for. You run
it yourself, on your own domain, with your own database, SMTP account and
credentials. Nothing in the code points at any project-operated server; every
URL comes from your configuration and defaults to `localhost`.

There are no plans, tiers or paywalls. Project home:
<https://github.com/vnytros/vnytros>; documentation:
[`docs/content/docs/`](../docs/content/docs).

## What it does

- **Domain tools** — detect which DNS provider hosts a domain (any provider;
  read-only), look up records, check whether expected A / CNAME / TXT records
  are in place, and verify domains you have linked.
- **Direct DNS writes** — apply records through a provider's API for the two
  providers that are supported: **AWS Route 53** (`aws`) and **Hostinger**
  (`hostinger`). Only `AddRecord` is implemented; `DeleteRecord`,
  `ListRecords` and `TestConnection` are not implemented for any adapter yet.
- **Network and email diagnostics** — 15 checks under `/v1/tools/*`: DNS
  propagation, nameservers, CNAME chains, CAA, SOA, DNSSEC, SPF, DKIM, DMARC,
  combined email authentication, RDAP registration, TLS certificates, HTTP
  security headers, redirect chains and IP info.
- **Vnytros Mail** — templates, campaigns, your own SMTP senders, open
  tracking and analytics.
- **MCP server** — the domain tools and diagnostics as Model Context Protocol
  tools for AI clients.

## Architecture

One module (`domain-connect-backend`), three binaries:

| Binary | Entry point | What it serves |
|---|---|---|
| `vnytros-server` | `cmd/server` | `:8000` — the REST API at `/v1`, the OAuth server at `/oauth` and `/.well-known/oauth-authorization-server`, Domain Connect discovery at `/.well-known/domainconnect*` and `/_domainconnect/_dckeypubv1.public`, and mailforge mounted at `/api` |
| `vnytros-mcp` | `cmd/mcp` | `:5000` — the MCP server (streamable HTTP) plus a landing page listing its tools |
| `vnytros-worker` | `cmd/worker` | RabbitMQ consumer: email sending, campaign batches, open tracking, webhook delivery, domain verification |

Supporting command: `cmd/migration` (`make migration`) — the only schema
migrator (GORM AutoMigrate). It also loads the DNS provider reference data used
by provider detection (`internal/db/migrations/dns_provider.sql`, embedded in
the binary and upserted by id), so there is no separate seed step and it is
safe to re-run.

PostgreSQL holds all state; RabbitMQ carries jobs from the server to the
worker. Without RabbitMQ the server still runs and sends system email
synchronously. Each binary exposes Prometheus metrics on loopback
(9101 / 9102 / 9103) and can export traces over OTLP.

```
cmd/            entry points (server, mcp, worker, migration, apikey)
internal/
  server/       router for /v1, /oauth, /.well-known; mounts mailforge at /api
  handler/ service/ repository/ models/ views/   the /v1 API
  dns/          provider detection data + write providers (aws, hostinger)
  netcheck/     the 15 diagnostic tools
  mailforge/    the mail platform (gin), mounted at /api
  mcp/          MCP tools and landing page
  worker/       queue consumers
  telemetry/    tracing and metrics
deploy/         self-hosting kit (scripts, nginx templates, observability) — see deploy/README.md
```

## Domain Connect

When `DC_PROVIDER_DOMAIN` is set (e.g. `example.com`), the Domain Connect
template is served at `/.well-known/domainconnect/<DC_PROVIDER_DOMAIN>/custom-domain`
from `./templates/<DC_PROVIDER_DOMAIN>.custom-domain.json`; with it unset the
route is not registered.

The project's own template, [`vnytros.dev.custom-domain.json`](https://github.com/Domain-Connect/Templates/blob/master/vnytros.dev.custom-domain.json)
(providerId `vnytros.dev`, serviceId `custom-domain`), is accepted upstream in
the official [Domain-Connect/Templates](https://github.com/Domain-Connect/Templates)
repository: [PR #1064](https://github.com/Domain-Connect/Templates/pull/1064)
(merged 2026-05-01) and [PR #1166](https://github.com/Domain-Connect/Templates/pull/1166)
(merged 2026-05-31).

That template belongs to the `vnytros.dev` providerId. If you self-host on your
own domain, write your own `templates/<your-domain>.custom-domain.json` with
your own providerId (and `syncPubKeyDomain` pointing at your domain's key) and
submit it to [Domain-Connect/Templates](https://github.com/Domain-Connect/Templates)
yourself; DNS providers only apply templates they have onboarded.

## Quickstart (local development)

Requirements: Go (see the version in `go.mod`), Docker with Compose, `make`
and `openssl`.

```bash
git clone https://github.com/vnytros/vnytros
cd vnytros/server

# 1. Postgres and RabbitMQ
docker compose up -d db rabbitmq

# 2. Config. The server loads .env.production from the working directory by
#    default; set ENV_FILE to load a different file. Process environment
#    variables take precedence over the file.
cp .env.example .env.production
#    then edit it: at least JWT_SECRET, APIKEY_SIGNING_SECRET and
#    WEBHOOK_SIGNING_SECRET should be long random strings.

# 3. Domain Connect keypair (written to keys/, which is gitignored)
make keys

# 4. Schema and DNS provider data (idempotent; re-run after upgrades)
make migration

# 5. Run (each in its own terminal)
make run          # API + OAuth + mailforge on :8000
make run-worker   # background jobs
make run-mcp      # MCP server on :5000
```

The values in `.env.example` match the credentials in `docker-compose.yaml`,
so the database and broker work without further changes. No accounts are
created for you: sign up through `POST /v1/auth/signup` or your dashboard.

To get an API key without the dashboard (`cmd/apikey`):

```bash
make apikey EMAIL=you@example.com               # existing account
make apikey EMAIL=you@example.com ARGS=-create  # creates a verified account first
```

The key is stored in the database like a dashboard-created key and also
written to `keys/vnytros-api-key.env` (mode 600, gitignored). Load it with
`set -a; source keys/vnytros-api-key.env; set +a`. Other flags: `-webhook`
(URL that receives the key's domain events), `-name`, `-out` (`""` to skip
the file).

Build and test:

```bash
make build-all    # binaries in bin/
make test         # go test ./...
go vet ./...
```

## Deploying

To run it on your own server (nginx + Let's Encrypt, Docker for the broker and
Grafana / Prometheus / Tempo), follow [deploy/README.md](deploy/README.md):
infrastructure runs under Docker Compose (`deploy/docker-compose.infra.yaml`)
and the three binaries are built on the server from a clone of the monorepo
(`~/app/vnytros/server`). Run the guide's `make` targets from `server/`. The
optional GitHub Actions deploy is the manual `deploy` workflow at the monorepo
root (`.github/workflows/deploy.yml`).

## Configuration

Every environment variable the code reads is listed, with its default, in
[`.env.example`](.env.example). Defaults are local (`localhost`) or empty —
never a hosted instance. With `ENV=production` the server refuses to start if
`JWT_SECRET`, `APIKEY_SIGNING_SECRET`, `BASE_URL`, `FRONTEND_URL` or
`ALLOWED_ORIGINS` is unset. The groups:

| Group | Variables |
|---|---|
| Runtime | `ENV`, `ENV_FILE` (the API always listens on `:8000`, MCP on `:5000`) |
| Database | `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME` |
| Queue | `RABITMQ` (AMQP URL; empty = send system mail synchronously) |
| Auth and secrets | `JWT_SECRET`, `JWT_ACCESS_TTL_MIN`, `JWT_REFRESH_TTL_HOURS`, `APIKEY_SIGNING_SECRET`, `WEBHOOK_SIGNING_SECRET`, `AUTH_COOKIE_DOMAIN`, `AUTH_COOKIE_SECURE` |
| Public URLs and CORS | `BASE_URL`, `FRONTEND_URL`, `DASHBOARD_URL`, `ALLOWED_ORIGINS`, `MCP_SERVER_BASE_URL`, `OAUTH_ISSUER_URL`, `OAUTH_ALLOWED_CALLBACK_URLS` |
| Branding and identity | `PRODUCT_NAME`, `SUPPORT_EMAIL`, `EMAIL_LOGO_URL`, `DC_PROVIDER_DOMAIN` |
| Social login (optional) | `GOOGLE_CLIENT_ID`, `GOOGLE_CLIENT_SECRET`, `GOOGLE_CLIENT_CALLBACK_URL`, `GOOGLE_MCP_CALLBACK_URL`, `GITHUB_CLIENT_ID`, `GITHUB_CLIENT_SECRET`, `GITHUB_CLIENT_CALLBACK_URL` |
| System mail (SMTP) | `SMTP_FROM_EMAIL`, `SMTP_PASSWORD`, `SMTP_HOST`, `SMTP_PORT` (`EMAIL_FROM`, `EMAIL_PASSWORD`, `EMAIL_HOST`, `EMAIL_PORT` as fallbacks), `MAIL_SEND_CONCURRENCY` |
| Mail platform | `SEND_BATCH_SIZE`, `MAIL_WATERMARK_IMAGE_URL`, `MAIL_WATERMARK_LINK_URL`, `MAIL_WATERMARK_LABEL`, `MAIL_WATERMARK_ALLOWED_IMAGE_URLS` |
| Observability | `OTEL_ENDPOINT`, `OTEL_TRACES_SAMPLE_RATIO`, `METRICS_HOST` |
| Server infra (`deploy/.env`, not the app) | `RABBITMQ_DEFAULT_USER`, `RABBITMQ_DEFAULT_PASS`, `GRAFANA_ADMIN_PASSWORD`, `GRAFANA_ROOT_URL` — see `deploy/infra.env.example` |

Never commit a real env file or anything under `keys/`; both are gitignored.

## Contributing and security

See the monorepo's [CONTRIBUTING.md](../CONTRIBUTING.md). Please report
vulnerabilities privately as described in [SECURITY.md](../SECURITY.md), not in
public issues.

## License

[GNU Affero General Public License v3.0 only](LICENSE) (AGPL-3.0-only). If you
run a modified version as a network service, you must offer its source to the
users of that service.
