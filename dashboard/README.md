# Vnytros Dashboard

The web dashboard for [Vnytros](https://github.com/Aadesh998/vnytros-domain-connect), a free, open-source
community project for connecting domains and sending email you can trust.

Vnytros is **self-hosted**: the project runs no public API, dashboard, or
accounts. You deploy this dashboard next to your own API
([`server/`](../server) in the same
[monorepo](https://github.com/Aadesh998/vnytros-domain-connect)) and point it at that API.
This folder is `dashboard/`; every command below runs from it.

Signed-in users can:

- see their connected domains and verification status
- create and manage API keys and browse webhook logs
- use Vnytros Mail: campaigns, templates, SMTP senders, branding and analytics
- manage their profile and workspace settings

Accounts live on your own API: users sign up, sign in, verify their
email and reset passwords on this dashboard's own pages (`/signup`, `/login`,
`/verify`, `/forgot-password`, `/reset-password`; Google/GitHub sign-in when
the API has them configured). No separate website is needed. Everything is available to every account. There are
no plans or paywalls.

The dashboard is a single-page app built with Vite, React 19, TypeScript,
Tailwind CSS v4 and React Router. It talks to the Vnytros API
([`server/`](../server)) and does not have a backend of its own.

## Self-hosting setup

Requirements: Node.js 20 or newer, npm, and a running Vnytros API (see the
[self-hosting guide](../docs/content/docs/self-hosting.mdx)).

```sh
git clone https://github.com/Aadesh998/vnytros-domain-connect
cd vnytros-domain-connect/dashboard
npm install
cp .env.example .env.local   # set VITE_API_BASE_URL to your API
npm run dev                  # http://localhost:5173
```

The API must accept the dashboard's origin:

- set the API's `FRONTEND_URL` to the dashboard URL (`http://localhost:5173`
  locally). The API allows that origin for CORS and builds its verification
  (`/verify?token=…`) and password-reset (`/reset-password?token=…`) email
  links from it, which this dashboard serves. Add any other dashboard origins
  to `ALLOWED_ORIGINS`;
- for Google/GitHub sign-in, add `<dashboard-origin>/auth/<provider>/callback`
  to the API's `OAUTH_ALLOWED_CALLBACK_URLS`.

Then open `/signup` on the dashboard to create the first account.

### Environment variables

Vite reads these at build time. They are public (they end up in the browser
bundle), so never put secrets in them. Put real values in `.env.local` (dev) or
`.env.production.local` / your CI environment (build); every `.env*` file
except `.env.example` is git-ignored.

| Variable | Required | Purpose | Default |
| --- | --- | --- | --- |
| `VITE_API_BASE_URL` | yes | Base URL of your Vnytros API (`/v1/*` routes), e.g. `https://api.your-domain.com` | none — `npm run build` fails without it, and `npm run dev` shows a configuration screen |
| `VITE_MAILFORGE_BASE_URL` | no | Base URL for mail routes (`/api/*`); mail is served by the same API | `VITE_API_BASE_URL` |

## Scripts

| Command | What it does |
| --- | --- |
| `npm run dev` | Start the dev server |
| `npm run lint` | Run ESLint |
| `npm run build` | Type-check (`tsc -b`) and build to `dist/` |
| `npm run preview` | Serve the production build locally |

## Deploying

The dashboard can be deployed to Cloudflare Workers as static assets with an
SPA fallback, built with the Cloudflare Vite plugin (`wrangler.jsonc`, Worker
`vnytros-dashboard`).

| Cloudflare Workers Builds setting | Value |
| --- | --- |
| Root directory | `dashboard` |
| Build variable | `VITE_API_BASE_URL` = your API URL |
| Build command | `npm run build` |
| Deploy command | `npx wrangler deploy` |

From a checkout with your own Cloudflare login: `npm run deploy`.

To use any other static host instead:

```sh
VITE_API_BASE_URL=https://api.your-domain.com npm run build
```

`npm run build` produces a static site in `dist/`. Serve it from any static
host, with every unknown path falling back to `index.html` (SPA fallback) so
client-side routes such as `/mail/campaigns` work on reload.

On a static host that builds from git, point it at this monorepo with:

| Setting | Value |
| --- | --- |
| Root / working directory | `dashboard` |
| Install command | `npm ci` |
| Build command | `npm run build` |
| Output directory | `dist` |
| Build-time environment | `VITE_API_BASE_URL` (and optionally `VITE_MAILFORGE_BASE_URL`) |
| Routing | rewrite every unknown path to `/index.html` |

## Contributing

See the monorepo's [CONTRIBUTING.md](../CONTRIBUTING.md).

## License

[AGPL-3.0-only](LICENSE), the same license as the rest of the monorepo except
`sdk/` (MIT).
