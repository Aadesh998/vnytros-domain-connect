# Vnytros

Vnytros is a free, open-source DNS and email-authentication toolkit with a
built-in mail platform. It detects which DNS provider hosts a domain, writes
records through the AWS Route 53 and Hostinger APIs, verifies that domains are
connected (with signed webhooks), runs 15 DNS / email / TLS / HTTP diagnostic
checks, exposes all of it over a REST API and an MCP server for AI clients, and
ships Vnytros Mail (templates, campaigns, your own SMTP senders, open
tracking).

**Vnytros is self-hosted.** You run the stack on your own machine or domain,
with your own database, SMTP account and credentials. There are no plans,
tiers or paywalls.

- Source: <https://github.com/vnytros/vnytros>
- Documentation: [`docs/content/docs/`](docs/content/docs) (the source of the
  docs site; run it locally with `cd docs && npm run dev`)

**Contents:**
[How it works](#how-it-works) ·
[What's in this repository](#whats-in-this-repository) ·
[Run it locally](#run-it-locally) ·
[Test it](#test-it) ·
[Troubleshooting](#troubleshooting) ·
[Deploying](#deploying-each-part) ·
[SDK without npm](#using-the-sdk-without-npm) ·
[Domain Connect](#domain-connect) ·
[Licensing](#licensing)

---

## How it works

```
                 ┌──────────────────────┐        ┌──────────────────────┐
  Browser ──────►│ dashboard/  (:5173)  │        │ AI client (Claude…)  │
                 │ accounts, API keys,  │        └──────────┬───────────┘
                 │ domains, Mail        │                   │ OAuth + MCP
                 └──────────┬───────────┘                   ▼
                            │ REST               ┌───────────────────────┐
  Your app ── @vnytros/sdk ─┼───────────────────►│ server/cmd/mcp (:5000)│
            (or curl)       ▼                    └───────────┬───────────┘
                 ┌───────────────────────────────────────────┴──────────┐
                 │ server/cmd/server  (:8000)                            │
                 │  /v1/*      detect, DNS writes, status, verify,       │
                 │             15 diagnostic tools, auth, API keys       │
                 │  /oauth/*   OAuth server (used by MCP clients)        │
                 │  /api/*     Vnytros Mail (templates, campaigns)       │
                 │  /.well-known/domainconnect   Domain Connect          │
                 └───────┬──────────────────────┬────────────────────────┘
                         │                      │ jobs
                         ▼                      ▼
                   ┌───────────┐        ┌──────────────┐     ┌──────────────────────┐
                   │ PostgreSQL│        │  RabbitMQ    │────►│ server/cmd/worker    │
                   └───────────┘        └──────────────┘     │ emails, verification,│
                                                             │ webhooks             │
                                                             └──────────────────────┘
    DNS writes go out to the AWS Route 53 / Hostinger APIs with credentials you supply.
```

**A typical flow:**

1. **Detect.** `GET /v1/detect?domain=shop.acme.com` reports which DNS provider
   hosts the domain (Cloudflare, Route 53, Hostinger, …) from its nameservers.
2. **Write records.** `POST /v1/connect/direct` creates A / CNAME / TXT records
   at Route 53 or Hostinger using API credentials the domain owner provides.
3. **Check and verify.** `POST /v1/status` shows whether the expected records
   are live; `POST /v1/connect/verify` marks the domain verified and the worker
   sends a signed `domain.connected` webhook to your app.
4. **Diagnose.** `/v1/tools/*` runs SPF, DKIM, DMARC, propagation (25 public
   resolvers), DNSSEC, CAA, TLS, security headers and more, each with a grade
   and plain-language findings.

## What's in this repository

One repository, organised by folder. Each folder is an independent project
with its own toolchain, lockfile, README, LICENSE and CI workflow; there is
no root `package.json` or shared workspace.

| Folder | What it is | Stack | Deploys to | License |
| --- | --- | --- | --- | --- |
| [`server/`](server) | REST API (`/v1`), OAuth server, MCP server, mail platform (mailforge) and background worker | Go, PostgreSQL, RabbitMQ | your own Linux server ([`server/deploy/`](server/deploy/README.md)) | AGPL-3.0-only |
| [`dashboard/`](dashboard) | Web dashboard: accounts, API keys, domains, Vnytros Mail | Vite, React, TypeScript | Cloudflare Workers (or any static host) | AGPL-3.0-only |
| [`website/`](website) | Single-page project landing page (self-hostable) | Next.js, OpenNext | Cloudflare Workers | AGPL-3.0-only |
| [`docs/`](docs) | Documentation site ([content](docs/content/docs)) | Fumadocs, Next.js | Cloudflare Workers via OpenNext (or any Next.js host) | AGPL-3.0-only |
| [`sdk/`](sdk) | `@vnytros/sdk` 0.2.0, the JavaScript/TypeScript SDK | TypeScript, zero runtime deps | installed into your app ([without npm](#using-the-sdk-without-npm)) | MIT |

The server has five commands under `server/cmd/`:

| Command | Run with | What it does |
| --- | --- | --- |
| `cmd/server` | `make run` | REST API, OAuth server and Vnytros Mail on port 8000 |
| `cmd/worker` | `make run-worker` | Background jobs: emails, verification, webhooks |
| `cmd/mcp` | `make run-mcp` | MCP server for AI clients on port 5000 (`/mcp`) |
| `cmd/migration` | `make migration` | Creates the schema and loads DNS-provider data (safe to re-run) |
| `cmd/apikey` | `make apikey EMAIL=…` | Creates an API key without the dashboard |

Repository-wide files: [`CONTRIBUTING.md`](CONTRIBUTING.md),
[`SECURITY.md`](SECURITY.md), [`LICENSE`](LICENSE), the root
[`Makefile`](Makefile) (`make help` lists every shortcut) and the CI workflows
in [`.github/workflows/`](.github/workflows) (one per folder, each filtered to
changes in that folder).

---

## Run it locally

### Prerequisites

| Tool | Version |
| --- | --- |
| Go | the version in [`server/go.mod`](server/go.mod) (1.26+) |
| Node.js + npm | 22 or newer |
| Docker with Compose | any recent version |
| `make`, `openssl`, `curl` | usually already installed |

```bash
git clone https://github.com/vnytros/vnytros
cd vnytros
```

### 1. Start PostgreSQL and RabbitMQ

```bash
cd server
docker compose up -d db rabbitmq
```

Postgres listens on `127.0.0.1:5432` and RabbitMQ on `127.0.0.1:5672`
(management UI: <http://localhost:15672>, user `admin`, password `secret`).
These are local development containers; your data survives restarts in
Docker volumes.

### 2. Configure the server

```bash
cp .env.example .env.production
```

Open `server/.env.production` and set three secrets to long random strings
(`openssl rand -hex 32` prints one):

```bash
JWT_SECRET=...
APIKEY_SIGNING_SECRET=...
WEBHOOK_SIGNING_SECRET=...
```

Everything else already matches the Docker containers and local URLs. Optional:

- `SMTP_HOST`, `SMTP_PORT`, `SMTP_FROM_EMAIL`, `SMTP_PASSWORD` to send
  verification and password-reset emails (a Gmail app password or a free
  Mailtrap inbox works). Without SMTP, use `make apikey … ARGS=-create` in
  step 5, which creates an already-verified account.
- `GOOGLE_*` / `GITHUB_*` for social login.

The server reads `.env.production` from the `server/` folder; set `ENV_FILE`
to load a different file. Every variable is documented in
[`server/.env.example`](server/.env.example) and
[`server/README.md`](server/README.md).

### 3. Generate keys and create the database schema

```bash
make keys        # Domain Connect keypair into server/keys/ (gitignored)
make migration   # tables + DNS provider data
```

### 4. Start the server (three terminals, each in `server/`)

```bash
make run          # API on http://localhost:8000
make run-worker   # background jobs
make run-mcp      # MCP server on http://localhost:5000/mcp
```

### 5. Create an account and an API key

**Without email set up** (fastest):

```bash
make apikey EMAIL=you@example.com ARGS=-create
```

This creates a verified account and prints its password once (use it to log
in to the dashboard). It then creates an API key, stores it in the database
and saves it to `server/keys/vnytros-api-key.env`. Load it into your shell
for the tests below:

```bash
set -a; source keys/vnytros-api-key.env; set +a     # sets VNYTROS_API_KEY and VNYTROS_BASE_URL
```

**With SMTP set up:** sign up in the dashboard (step 6), click the link in the
verification email, then create a key under **Developer**, or run
`make apikey EMAIL=you@example.com` for the existing account.

### 6. Start the dashboard (new terminal)

```bash
cd dashboard
npm ci
cp .env.example .env.local      # VITE_API_BASE_URL=http://localhost:8000
npm run dev                     # http://localhost:5173
```

### 7. Optional: website and docs

```bash
cd website && npm ci && npm run dev              # http://localhost:3000
cd docs    && npm ci && npm run dev -- -p 3001   # http://localhost:3001
```

### Local URLs

| Part | URL |
| --- | --- |
| API | <http://localhost:8000> |
| MCP server | <http://localhost:5000/mcp> |
| Dashboard | <http://localhost:5173> |
| SDK playground | <http://localhost:5174> (see below) |
| Website / Docs | <http://localhost:3000> / <http://localhost:3001> |
| RabbitMQ UI | <http://localhost:15672> |

To stop everything, press `Ctrl+C` in each terminal, then run
`docker compose down` in `server/` (`docker compose down -v` also deletes the
data).

---

## Test it

The examples use the local API. To test a deployed server instead, set
`VNYTROS_BASE_URL` to its URL (for example `https://api.your-domain.com`) and
use an API key created on that deployment.

```bash
export VNYTROS_BASE_URL=${VNYTROS_BASE_URL:-http://localhost:8000}
```

### 1. Is the API up?

```bash
curl $VNYTROS_BASE_URL/
# Domain Connect API is running
```

### 2. Public endpoints (no API key)

**Which DNS provider hosts a domain:**

```bash
curl "$VNYTROS_BASE_URL/v1/detect?domain=example.com"
# {"domain":"example.com","nameservers":[...],"provider":{"name":"...",...}}
```

**Are the expected records live?** (`ip`, `target` and `txt` are all optional)

```bash
curl -X POST $VNYTROS_BASE_URL/v1/status \
  -H "Content-Type: application/json" \
  -d '{"domain":"example.com","ip":"1.2.3.4"}'
# {"domain":"example.com","is_configured":false,"records":[{"type":"A","expected":"1.2.3.4","current":[...],"status":false}]}
```

**Diagnostic tools.** Each returns a verdict, a grade and findings:

```bash
curl "$VNYTROS_BASE_URL/v1/tools/email/auth?domain=your-domain.com"       # SPF + DKIM + DMARC together
curl "$VNYTROS_BASE_URL/v1/tools/dns/propagation?domain=your-domain.com"  # 25 public resolvers
curl "$VNYTROS_BASE_URL/v1/tools/tls/certificate?domain=your-domain.com"
curl "$VNYTROS_BASE_URL/v1/tools"                                         # list of every tool
```

| Path | Checks |
| --- | --- |
| `/v1/tools/dns/propagation` | the record across 25 public resolvers |
| `/v1/tools/dns/nameservers` | delegation and nameserver health |
| `/v1/tools/dns/cname` | the CNAME chain |
| `/v1/tools/dns/caa` | which certificate authorities may issue certificates |
| `/v1/tools/dns/soa` | the SOA record |
| `/v1/tools/dns/dnssec` | DNSSEC signing and validation |
| `/v1/tools/email/auth` | SPF, DKIM and DMARC together |
| `/v1/tools/email/spf`, `/dkim`, `/dmarc` | each one on its own (`&selector=` for DKIM) |
| `/v1/tools/domain/info` | registration (RDAP/WHOIS) |
| `/v1/tools/tls/certificate` | the TLS certificate and protocols |
| `/v1/tools/http/headers` | HTTP security headers (`?url=`) |
| `/v1/tools/http/redirects` | the redirect chain (`?url=`) |
| `/v1/tools/ip/info` | an IP address or host |

Tip: test the email tools with your own domain. DKIM keys live under selector
names the checker has to guess, so a large provider's domain can grade poorly
when its current selector isn't one of the common names. Pass
`&selector=<name>` if you know it.

### 3. Authenticated endpoints (API key)

```bash
AUTH="Authorization: Bearer $VNYTROS_API_KEY"

curl -H "$AUTH" $VNYTROS_BASE_URL/v1/connect/domains                       # your connected domains
curl -H "$AUTH" "$VNYTROS_BASE_URL/v1/connect/domain?domain=your-domain.com"
```

A `200` means the key works, `401` means no key was sent, and `400` means the
key is malformed or was signed with a different `APIKEY_SIGNING_SECRET`.

**Write DNS records** (Route 53 or Hostinger). This changes real DNS, so use
a domain or subdomain you control and can clean up:

```bash
# Hostinger: an API token from hPanel
curl -X POST $VNYTROS_BASE_URL/v1/connect/direct -H "$AUTH" \
  -H "Content-Type: application/json" \
  -d '{"domain":"test.your-domain.com","provider":"hostinger",
       "ip":"203.0.113.10","txt":"vnytros-verify-123",
       "provider_config":{"APIToken":"YOUR_HOSTINGER_TOKEN"}}'

# AWS Route 53: an IAM user allowed to change the hosted zone
#   "provider":"aws", "provider_config":{"AccessKey":"...","SecretKey":"..."}
```

Then check and verify:

```bash
curl -X POST $VNYTROS_BASE_URL/v1/status -H "Content-Type: application/json" \
  -d '{"domain":"test.your-domain.com","ip":"203.0.113.10","txt":"vnytros-verify-123"}'

curl -X POST $VNYTROS_BASE_URL/v1/connect/verify -H "$AUTH" \
  -H "Content-Type: application/json" -d '{"domain":"test.your-domain.com"}'
```

Only record **creation** is implemented for both providers; deleting,
listing and connection tests are not yet.

### 4. The dashboard

Open <http://localhost:5173> and log in:

- **Overview:** account summary.
- **Domains:** domains connected with your API keys and their verification
  status.
- **Developer:** create, disable and delete API keys, and see webhook logs.
- **Mail:** add an SMTP sender under **Settings**, build a template, send a
  campaign and open its report (opens are tracked).
- **Profile / Settings:** account details.

### 5. The SDK playground (a browser demo of the SDK)

```bash
cd sdk
npm ci
set -a; source ../server/keys/vnytros-api-key.env; set +a   # or export VNYTROS_API_KEY=... yourself
PORT=5174 npm run playground                                 # http://localhost:5174
```

It detects a domain's provider, writes records and verifies them, all from a
web page. It calls `VNYTROS_BASE_URL` (default `http://localhost:8000`). The
port is set to 5174 because the dashboard already uses 5173.

### 6. The SDK in your own code

```ts
import { Vnytros } from "@vnytros/sdk";   // to install it, see "Using the SDK without npm"

const vny = new Vnytros({
  apiKey: process.env.VNYTROS_API_KEY!,
  baseUrl: process.env.VNYTROS_BASE_URL!,   // http://localhost:8000 locally
});

const { provider } = await vny.detect("your-domain.com");
const status = await vny.domains.status({ domain: "your-domain.com", ip: "203.0.113.10" });
const report = await vny.tools.emailAuth("your-domain.com");
console.log(provider?.name, status.is_configured, report);
```

### 7. The MCP server (AI assistants)

Add `http://localhost:5000/mcp` as a remote MCP server (Streamable HTTP) in an
MCP client that supports OAuth. For a quick local test, the MCP Inspector
works:

```bash
npx @modelcontextprotocol/inspector     # then connect to http://localhost:5000/mcp
```

The client is sent to the API's OAuth login (`/oauth/authorize`); sign in
with your Vnytros account. You then get 9 domain tools (`domain_provider`,
`domain_status`, `list_user_domains`, …) and 15 diagnostics (`email_spf`,
`dns_propagation`, `tls_certificate`, …). Hosted clients such as Claude can
only reach a server on a public HTTPS URL, so use a deployed server, or a
tunnel, for those.

### 8. Automated checks

From the repository root:

```bash
make test        # every folder: Go build/vet/test, type checks, lint, builds, SDK tests
```

Or one part at a time: `make server-check`, `make dashboard-check`,
`make website-check`, `make docs-check`, `make sdk-check`. CI runs the same
checks on every push and pull request.

---

## Troubleshooting

| Problem | Fix |
| --- | --- |
| `port 5432 already in use` | Another Postgres is running. Stop it, or change the host port in `server/docker-compose.yaml` and `DB_PORT` in `.env.production` to match. |
| Server exits at startup saying a variable is not set | You have `ENV=production`, which requires `JWT_SECRET`, `APIKEY_SIGNING_SECRET`, `BASE_URL`, `FRONTEND_URL` and `ALLOWED_ORIGINS`. Locally, keep `ENV=development`. |
| Dashboard shows a configuration error | `VITE_API_BASE_URL` is missing from `dashboard/.env.local`. Restart `npm run dev` after setting it. |
| Browser console shows CORS errors | Add the dashboard's exact origin to `ALLOWED_ORIGINS` (and `FRONTEND_URL`) in `server/.env.production`, then restart `make run`. |
| Can't log in after signing up | The account isn't verified. Configure SMTP, or create the account with `make apikey EMAIL=… ARGS=-create`. |
| API returns `400` for a key that looks valid | The key was signed with another deployment's `APIKEY_SIGNING_SECRET`. Create the key on the server you're calling. |
| Playground: `VNYTROS_API_KEY is not set` | Load the key first: `set -a; source ../server/keys/vnytros-api-key.env; set +a`. |
| Playground or dashboard won't start on 5173 | Both default to 5173; run the playground with `PORT=5174`. |
| Emails never arrive | Check that `make run-worker` is running and the `SMTP_*` values are right. RabbitMQ's UI (port 15672) shows stuck queues. |

---

## Deploying each part

Every part deploys on its own, from its own folder. Nothing here assumes a
particular hosting account.

| Part | How it deploys |
| --- | --- |
| `server/` | Follow [`server/deploy/README.md`](server/deploy/README.md), running its `make` targets from `server/`. Infrastructure (RabbitMQ, OpenTelemetry collector, Tempo, Prometheus, Grafana) runs under Docker Compose (`server/deploy/docker-compose.infra.yaml`); the three Go binaries are built **on the server** from a clone of this repository (`~/app/vnytros/server`) and run behind nginx with Let's Encrypt. PostgreSQL is yours to provide. The optional GitHub Actions deploy is [`.github/workflows/deploy.yml`](.github/workflows/deploy.yml): manual (`workflow_dispatch`) and driven entirely by the `DEPLOY_SSH_*` secrets you add to your fork. |
| `dashboard/` | Cloudflare Workers (static assets with SPA fallback) via the Cloudflare Vite plugin (`wrangler.jsonc`, Worker `vnytros-dashboard`). In Cloudflare Workers Builds set the **root directory to `dashboard`**, the build variable `VITE_API_BASE_URL` to your API, build `npm run build`, deploy `npx wrangler deploy`; from a checkout, `npm run deploy`. Any static host also works: output directory `dist`, rewrite unknown paths to `/index.html`. |
| `website/` | Cloudflare Workers via [OpenNext](https://opennext.js.org/cloudflare), from `website/` (`wrangler.jsonc`, `open-next.config.ts`). In Cloudflare Workers Builds set the **root directory to `website`**; the build is `npm run build` (OpenNext) and the deploy `npx wrangler deploy`. From a checkout, `npm run deploy` in `website/` does both with your own Cloudflare login. Any Next.js host also works with `npm run build:next`. |
| `docs/` | Cloudflare Workers via OpenNext (`wrangler.jsonc`, `open-next.config.ts`, Worker `vnytros-docs`). In Cloudflare Workers Builds set the **root directory to `docs`**, build `npx opennextjs-cloudflare build`, deploy `npx wrangler deploy`; from a checkout, `npm run deploy`. Any Node.js host also works (`npm run build`, then `npm start`). No environment variables. |
| `sdk/` | Not deployed: apps install it. See below. |

**Before the first production deploy of the server**, read
[`server/deploy/README.md`](server/deploy/README.md). Set `ENV=production`
and the production URLs (`BASE_URL`, `FRONTEND_URL`, `ALLOWED_ORIGINS`,
`MCP_SERVER_BASE_URL`, `OAUTH_ISSUER_URL`, `DC_PROVIDER_DOMAIN`), and back up
the database before running `make migration` on an existing deployment: it
drops tables left over from older versions.

Each folder's README has the details.

## Using the SDK without npm

`@vnytros/sdk` is not published to the npm registry. Install it into your app
from this repository; your code still imports `@vnytros/sdk` as usual. Both
methods below were tested by installing into a fresh project and importing
`Vnytros` (ESM and CJS).

**(a) Tarball from `npm pack`** (recommended for apps and CI):

```bash
cd vnytros/sdk
npm ci
npm pack                         # prepack runs the build; writes vnytros-sdk-0.2.0.tgz
cd /path/to/your-app
npm install /path/to/vnytros-sdk-0.2.0.tgz
```

npm records the tarball as a `file:` dependency relative to your app, so keep
the `.tgz` with the app (for example under `vendor/`) or regenerate it in CI.
`make sdk-pack` from the repository root does the same as the first three
lines.

**(b) Local folder** (for developing the SDK and an app side by side):

```bash
cd vnytros/sdk && npm ci && npm run build      # dist/ must exist first
cd /path/to/your-app
npm install file:../path/to/vnytros/sdk
```

npm symlinks the folder; rebuild the SDK (`npm run build`, or `npm run dev` to
watch) after changing its sources.

**(c) GitHub Packages** is an option only if you choose to publish the package
there yourself; this repository does not set it up.

## Domain Connect

The Vnytros `custom-domain` service template (providerId `vnytros.dev`) is
accepted upstream in the official
[Domain-Connect/Templates](https://github.com/Domain-Connect/Templates)
repository:

- [PR #1064](https://github.com/Domain-Connect/Templates/pull/1064):
  `vnytros.dev.custom-domain.json`, merged 2026-05-01
- [PR #1166](https://github.com/Domain-Connect/Templates/pull/1166): more
  records, merged 2026-05-31

The accepted file is
[`vnytros.dev.custom-domain.json`](https://github.com/Domain-Connect/Templates/blob/master/vnytros.dev.custom-domain.json) in that repository.
That template belongs to the `vnytros.dev` providerId; if you self-host on
your own domain, write and submit your own template (see
[`server/README.md`](server/README.md#domain-connect)).

DNS writes through provider APIs are supported for **AWS Route 53** and
**Hostinger**, and only record creation (`AddRecord`) is implemented.

## Licensing

Everything in this repository is licensed under the
[GNU Affero General Public License v3.0 only](LICENSE) (`AGPL-3.0-only`),
**except `sdk/`, which is MIT**.

Each folder carries its own `LICENSE` file so that it stays correctly licensed
when built, copied or deployed on its own:

| Folder | License file |
| --- | --- |
| `server/`, `dashboard/`, `website/`, `docs/` | `LICENSE` in each folder: AGPL-3.0-only, identical to the root [`LICENSE`](LICENSE) |
| `sdk/` | [`sdk/LICENSE`](sdk/LICENSE): MIT, so you can use the SDK in any application, open source or not |

Under the AGPL, if you run a modified version of an AGPL part as a network
service, you must offer its source to the users of that service.

## Contributing and security

See [CONTRIBUTING.md](CONTRIBUTING.md) for setup and the checks for each
folder. Report vulnerabilities privately through
[GitHub security advisories](https://github.com/vnytros/vnytros/security/advisories/new),
as described in [SECURITY.md](SECURITY.md), not in public issues.
