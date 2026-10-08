# Vnytros

Vnytros is a free, open-source DNS and email-authentication toolkit with a
built-in mail platform. It detects which DNS provider hosts a domain, writes
records through the AWS Route 53 and Hostinger APIs, verifies that domains are
connected (with signed webhooks), runs 15 DNS / email / TLS / HTTP diagnostic
checks, exposes all of it over a REST API and an MCP server for AI clients, and
ships Vnytros Mail (templates, campaigns, your own SMTP senders, open
tracking).

**Vnytros is self-hosted only.** The project runs no hosted API, dashboard,
mail service or accounts, and there are no plans, tiers or paywalls. You run
the stack on your own domain with your own database, SMTP account and
credentials.

- Website: <https://vnytros.dev>
- Documentation: <https://docs.vnytros.dev>

## What's in this repository

One repository, organised by folder. Each folder is an independent project
with its own toolchain, lockfile, README, LICENSE and CI workflow; there is
no root `package.json` or shared workspace.

| Folder | What it is | Stack | Deploys to | License |
| --- | --- | --- | --- | --- |
| [`server/`](server) | REST API (`/v1`), OAuth server, MCP server, mail platform (mailforge) and background worker | Go, PostgreSQL, RabbitMQ | your own Linux server ([`server/deploy/`](server/deploy/README.md)) | AGPL-3.0-only |
| [`dashboard/`](dashboard) | Web dashboard: accounts, API keys, domains, Vnytros Mail | Vite, React, TypeScript | Cloudflare Workers (or any static host) | AGPL-3.0-only |
| [`website/`](website) | Single-page project site, [vnytros.dev](https://vnytros.dev) | Next.js, OpenNext | Cloudflare Workers | AGPL-3.0-only |
| [`docs/`](docs) | Documentation site, [docs.vnytros.dev](https://docs.vnytros.dev) | Fumadocs, Next.js | Cloudflare Workers via OpenNext (or any Next.js host) | AGPL-3.0-only |
| [`sdk/`](sdk) | `@vnytros/sdk` 0.2.0, the JavaScript/TypeScript SDK | TypeScript, zero runtime deps | installed into your app ([without npm](#using-the-sdk-without-npm)) | MIT |

Repository-wide files: [`CONTRIBUTING.md`](CONTRIBUTING.md),
[`SECURITY.md`](SECURITY.md), [`LICENSE`](LICENSE), the root
[`Makefile`](Makefile) and the CI workflows in
[`.github/workflows/`](.github/workflows) (one per folder, each filtered to
changes in that folder).

## Quick start: self-host the whole stack locally

Requirements: Go (the version in `server/go.mod`), Docker with Compose, `make`,
`openssl`, and Node.js 20+ with npm.

```bash
git clone https://github.com/vnytros/vnytros
cd vnytros
```

**1. API, worker and MCP server** (`server/`):

```bash
cd server
docker compose up -d db rabbitmq      # throwaway Postgres + RabbitMQ on 127.0.0.1
cp .env.example .env.production       # set JWT_SECRET, APIKEY_SIGNING_SECRET, WEBHOOK_SIGNING_SECRET
make keys                             # Domain Connect keypair into keys/ (gitignored)
make migration                        # schema + DNS provider data (idempotent)
make run                              # API + OAuth + mail on :8000
make run-worker                       # background jobs (second terminal)
make run-mcp                          # MCP server on :5000 (third terminal)
```

`.env.example` already sets `FRONTEND_URL` and `ALLOWED_ORIGINS` to the
dashboard's local origin (`http://localhost:5173`); see
[`server/README.md`](server/README.md) for every variable.

**2. Dashboard** (`dashboard/`, from the repository root):

```bash
cd dashboard
npm ci
cp .env.example .env.local            # VITE_API_BASE_URL defaults to http://localhost:8000
npm run dev                           # http://localhost:5173, then open /signup
```

**3. Optional: website and docs** (`website/`, `docs/`): `npm ci && npm run dev`
in either folder (both serve on port 3000, so run one at a time).

The root `Makefile` wraps the common commands, for example `make server-build`,
`make dashboard-dev`, `make website-build`, `make docs-build`, `make sdk-test`,
and `make test` to run every folder's checks. `make help` lists them all.

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

A copy lives at
[`server/templates/vnytros.dev.custom-domain.json`](server/templates/vnytros.dev.custom-domain.json).
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
