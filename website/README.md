# Vnytros website

The project landing page for **Vnytros**, an
open-source DNS and email-authentication toolkit with a built-in mail
platform. You self-host it. There is no hosted service, and the project runs
no public copy of this page: its home is the
[GitHub repository](https://github.com/Aadesh998/vnytros-domain-connect).

This site is deliberately small: a static Next.js app with a single page, no
accounts, no forms and no API calls. It is the `website/` folder of the
[Vnytros monorepo](https://github.com/Aadesh998/vnytros-domain-connect); every command below
runs from it.

| Route      | What it is                                                                 |
| ---------- | -------------------------------------------------------------------------- |
| `/`        | What Vnytros is and does, the accepted Domain Connect template, self-hosting steps, repository links |

Every path the site used to serve (`/tools`, `/guides`, `/blog`, `/mcp`,
`/support`, `/login` and the rest) redirects permanently to `/`. The list is
`RETIRED_PATHS` in `next.config.ts`. The site also serves `/sitemap.xml`,
`/robots.txt` and a generated Open Graph image.

## Development

Requires Node.js 20+.

```bash
git clone https://github.com/Aadesh998/vnytros-domain-connect
cd vnytros-domain-connect/website
npm install
cp .env.example .env.local   # optional
npm run dev                  # http://localhost:3000
```

| Command              | What it does                                        |
| -------------------- | --------------------------------------------------- |
| `npm run dev`        | Next.js dev server                                  |
| `npm run lint`       | ESLint                                              |
| `npm run build:next` | Plain `next build`                                  |
| `npm run build`      | Cloudflare Workers build via OpenNext               |
| `npm run preview`    | Build and preview the Worker locally                |
| `npm run deploy`     | Build and deploy to Cloudflare (needs your account) |

## Deployment

The site runs on Cloudflare Workers through
[OpenNext](https://opennext.js.org/cloudflare). The configuration is in
`wrangler.jsonc` (the Worker name, assets and the R2 incremental-cache bucket)
and in `open-next.config.ts`.

With Cloudflare Workers Builds connected to the monorepo, set:

| Setting | Value |
| --- | --- |
| Root directory | `website` |
| Build command | `npm run build` (runs `opennextjs-cloudflare build`) |
| Deploy command | `npx wrangler deploy` (or `npm run deploy` for build + deploy) |

From a local checkout, `npm run deploy` in `website/` builds and deploys with
your own Cloudflare login. Change the Worker `name` (and the matching
`services` entry) and the R2 bucket in `wrangler.jsonc` if you deploy your own
copy. Any Node.js Next.js host also works with `npm run build:next` and
`npm start`, again with `website` as the root directory.

## Environment variables

Optional variables, inlined at build time (see `.env.example`):

| Variable | Default | Purpose |
| --- | --- | --- |
| `NEXT_PUBLIC_SITE_URL` | `http://localhost:3000` | Origin you serve the site from: `metadataBase`, canonical URL, Open Graph URL, sitemap, robots and JSON-LD. Set it to your own origin before building. |
| `NEXT_PUBLIC_GA_MEASUREMENT_ID` | (empty) | Google Analytics 4 ID. Empty loads no analytics script. |

## Other parts of the monorepo

- [`server/`](../server): Go REST API, MCP server, worker and Vnytros Mail
- [`dashboard/`](../dashboard): web dashboard
- [`sdk/`](../sdk): `@vnytros/sdk` (MIT)
- [`docs/`](../docs): documentation site (pages in [`docs/content/docs/`](../docs/content/docs))

## Contributing & licence

See the monorepo's [CONTRIBUTING.md](../CONTRIBUTING.md). This folder is
licensed under the [GNU Affero General Public License v3.0](./LICENSE)
(AGPL-3.0-only).
