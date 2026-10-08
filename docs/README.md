# Vnytros Docs

Source for [docs.vnytros.dev](https://docs.vnytros.dev) — the reference for the
Vnytros API (provider detection, direct DNS connect, verification, network
tools, webhooks, OAuth/MCP). Built with [Fumadocs](https://fumadocs.dev) on
Next.js. This is the `docs/` folder of the
[Vnytros monorepo](https://github.com/vnytros/vnytros); every command below
runs from it.

Vnytros is a free, open-source, **self-hosted** community project: there is
no hosted Vnytros API, so examples use the placeholder `https://api.example.com`
and readers substitute their own deployment's URL. There are no plans or
paywalls.

## Develop

Requires Node.js 20+.

```bash
npm install      # also generates .source/ via fumadocs-mdx
npm run dev      # http://localhost:3000/docs
npm run build    # production build
```

No environment variables are needed.

## Layout

- `content/docs/` — the MDX pages. Sidebar order lives in each folder's `meta.json`.
- `app/` — Next.js routes, metadata, sitemap, robots.
- `lib/` — Fumadocs source loader, shared layout (nav links), JSON-LD.
- `CONTENT-SCOPE.md` — what belongs in the public docs and what doesn't (not published).
- `scripts/audit-api-docs.mjs` — `npm run audit:docs` diffs the API router
  (`../server/internal/server/server.go`, in the same monorepo) against the
  docs. It fails if the router file is missing.

## Deploying

docs.vnytros.dev runs on Cloudflare Workers through
[OpenNext](https://opennext.js.org/cloudflare) (`wrangler.jsonc`,
`open-next.config.ts`, Worker `vnytros-docs`, R2 cache bucket
`vnytros-docs-opennext-cache`).

| Cloudflare Workers Builds setting | Value |
| --- | --- |
| Root directory | `docs` |
| Build command | `npx opennextjs-cloudflare build` |
| Deploy command | `npx wrangler deploy` |

From a checkout with your own Cloudflare login, `npm run deploy` does both;
`npm run preview` runs the Worker locally. No environment variables are needed.

Any Node.js host works too: `npm run build`, then `npm start`.

## Other parts of the monorepo

- [`server/`](../server) — API, worker, MCP server
- [`dashboard/`](../dashboard) — dashboard
- [`website/`](../website) — project website
- [`sdk/`](../sdk) — JavaScript SDK (MIT)

Self-hosting instructions for the whole stack are in
[`content/docs/self-hosting.mdx`](content/docs/self-hosting.mdx)
(published at docs.vnytros.dev/docs/self-hosting).

## Contributing

See the monorepo's [CONTRIBUTING.md](../CONTRIBUTING.md).

## License

[AGPL-3.0-only](LICENSE).
