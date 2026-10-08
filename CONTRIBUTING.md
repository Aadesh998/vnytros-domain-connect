# Contributing

Thanks for helping improve Vnytros. Issues and pull requests are welcome at
<https://github.com/Aadesh998/vnytros-domain-connect>.

This is one repository with five independent parts. Each folder builds and
tests on its own; there is no shared workspace or root `package.json`.

| Folder | What it is | Toolchain | License |
| --- | --- | --- | --- |
| [`server/`](server) | Go API, OAuth, MCP server, mail platform, worker | Go (version in `server/go.mod`) | AGPL-3.0-only |
| [`dashboard/`](dashboard) | Vite + React dashboard | Node.js 20+ | AGPL-3.0-only |
| [`website/`](website) | Next.js project landing page | Node.js 20+ | AGPL-3.0-only |
| [`docs/`](docs) | Fumadocs docs site | Node.js 20+ | AGPL-3.0-only |
| [`sdk/`](sdk) | `@vnytros/sdk` JavaScript SDK | Node.js 18+ | MIT |

## Workflow

1. Open an issue first for anything larger than a small fix, so the approach
   can be agreed before you write code. For larger docs changes, too.
2. Fork the repository and create a branch from `main`.
3. Set up the folder you are changing as described in its README.
4. Keep changes focused (one concern per pull request) and match the existing
   code style.
5. Run that folder's checks (below, or `make <folder>-check` from the root;
   `make test` runs every folder's checks). CI runs the same checks, only for
   the folders a pull request touches.
6. Describe what changed and why in the pull request, with screenshots for UI
   changes.

## Checks per folder

### `server/`

```bash
cd server
gofmt -l .        # should print nothing
go build ./...
go vet ./...
go test ./...
```

- Schema changes go through `cmd/migration` (`make migration`), which also
  upserts the DNS provider data in `internal/db/migrations/dns_provider.sql`.
  Keep migrations and that data idempotent so they are safe on both fresh
  installs and existing databases.
- If you add an environment variable, add it to `server/.env.example` with a
  placeholder value.

### `dashboard/`

```bash
cd dashboard
npm ci
npm run lint
VITE_API_BASE_URL=http://localhost:8000 npm run build   # tsc -b + vite build
```

Feature folders live under `src/features`, API clients under `src/lib`, data
hooks under `src/hooks`.

### `website/`

```bash
cd website
npm ci
npm run lint
npx tsc --noEmit
npm run build:next   # or `npm run build` for the Cloudflare (OpenNext) build
```

- Use the existing `vn-*` design tokens (`app/theme.css`) rather than raw
  colours.
- Keep the site simple: no accounts, forms or API calls.
- Only describe what the open-source server actually does. DNS write support
  is claimed only for AWS Route 53 and Hostinger, the providers the server
  implements.

### `docs/`

```bash
cd docs
npm ci
npm run build
npm run audit:docs   # diffs server/internal/server/server.go against the docs
```

- Pages live in `content/docs/`. New pages must be listed in the folder's
  `meta.json` to appear in the sidebar.
- Document the API contract only; read `docs/CONTENT-SCOPE.md` first.
- If you add or change a `/v1` route in `server/`, update the docs in the same
  pull request: `audit:docs` fails on a route that is neither documented nor
  deliberately excluded.

### `sdk/`

```bash
cd sdk
npm ci
npm run typecheck
npm test          # builds, then runs node:test against dist/
```

- The SDK has no runtime dependencies; please keep it that way.
- Keep wire types in `src/types.ts` identical to the API's JSON, snake_case
  included. If the API changes shape, update the types and a test together.
- New endpoints follow the existing resource style (`src/resources/*.ts`),
  take an optional trailing `AbortSignal`, and get a test in `tests/`.
- Only idempotent requests may be retried. Do not mark a request that writes
  state as `idempotent`.
- Never add code that would put an API key in a browser.
- Add a line to `sdk/CHANGELOG.md` for anything user-visible.

## Secrets

Never commit secrets: no `.env` or `.env.*` files (other than
`.env.example`), no `keys/` directory and no `*.pem` files. The root
`.gitignore` covers these for every folder.

## Security issues

Do not open a public issue for a vulnerability; see [SECURITY.md](SECURITY.md).

## Licensing of contributions

By contributing, you agree that your contribution is licensed under the
license of the folder it lands in: [AGPL-3.0-only](LICENSE) for `server/`,
`dashboard/`, `website/` and `docs/`, and the [MIT License](sdk/LICENSE) for
`sdk/`.
