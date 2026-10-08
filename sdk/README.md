# @vnytros/sdk

Official JavaScript/TypeScript SDK for the [Vnytros](https://github.com/vnytros/vnytros) API.
Vnytros is a free, open-source project: let your users point their own domain at
your service, check what their DNS looks like, and watch the records go live.

- Zero runtime dependencies. Node 18+.
- ESM, CJS and TypeScript types.
- MIT licensed. There is no paid tier and no API pricing.
- Source: the [`sdk/`](https://github.com/vnytros/vnytros/tree/main/sdk)
  folder of the [Vnytros monorepo](https://github.com/vnytros/vnytros).

## The one rule

**Your API key must stay on your server.** It decodes to your account identity,
so anyone who reads it from a browser bundle can act as you. Constructing the
client in a browser throws rather than shipping your key to every visitor; have
your page call a route in your own app that holds the key.

## Install

`@vnytros/sdk` is not published to the npm registry, so install it from the
monorepo (see [Using the SDK without npm](#using-the-sdk-without-npm) below for
the details of each method):

```bash
git clone https://github.com/vnytros/vnytros
cd vnytros/sdk && npm ci && npm pack            # -> vnytros-sdk-0.2.0.tgz
cd /path/to/your-app && npm install /path/to/vnytros/sdk/vnytros-sdk-0.2.0.tgz
```

Your code still imports it as `@vnytros/sdk`. Then configure:

```bash
VNYTROS_BASE_URL=https://api.your-domain.com   # your own Vnytros API deployment
VNYTROS_API_KEY=...            # from your deployment's dashboard
VNYTROS_WEBHOOK_SECRET=...     # WEBHOOK_SIGNING_SECRET on the API
```

## Quick start

Vnytros is self-hosted: there is no hosted Vnytros API, so `baseUrl` is
required and points at the API you run (see the
[`server/`](https://github.com/vnytros/vnytros/tree/main/server) folder of the
monorepo).

```ts
import { Vnytros } from "@vnytros/sdk";

const vny = new Vnytros({
  apiKey: process.env.VNYTROS_API_KEY!,
  baseUrl: process.env.VNYTROS_BASE_URL!, // "https://api.your-domain.com", or "http://localhost:8000" locally
});

const { provider } = await vny.detect("shop.acme.com");
```

## Using the SDK without npm

The package is not on npmjs.com. Every method below gives your app a normal
`node_modules/@vnytros/sdk`, so `import { Vnytros } from "@vnytros/sdk"` (or
`require`) works unchanged.

**(a) Tarball from `npm pack` (recommended for apps and CI).**

```bash
cd vnytros/sdk
npm ci
npm pack                       # runs the build first (prepack), writes vnytros-sdk-0.2.0.tgz
cd /path/to/your-app
npm install /path/to/vnytros-sdk-0.2.0.tgz
```

npm records the dependency as a `file:` path relative to your app (for a
tarball copied into the app folder: `"@vnytros/sdk": "file:vnytros-sdk-0.2.0.tgz"`),
so keep the `.tgz` with your app (e.g. commit it under `vendor/`) or regenerate
it in CI. The tarball contains only `dist/`, the README,
CHANGELOG and LICENSE, exactly what a registry install would.

**(b) Local folder link (for developing the SDK and an app together).**

```bash
cd vnytros/sdk && npm ci && npm run build    # dist/ must exist before installing
cd /path/to/your-app
npm install file:../path/to/vnytros/sdk
```

npm symlinks the folder, so rebuild (`npm run build`, or `npm run dev` to
watch) after changing SDK sources. Nothing is built for you on install.

**(c) GitHub Packages.** If you choose to publish the package to GitHub
Packages yourself, apps can install it from there with a scoped registry entry
in `.npmrc`. This repository does not set that up.

## How it works

```
1. detect     vny.detect(domain)                 which DNS host the domain uses (any provider)
2. records    either the user adds them by hand, or, for Route 53 / Hostinger,
              you write them with vny.domains.connectDirect()
3. verify     vny.domains.status() / waitUntilConnected(), or the domain.connected webhook
```

## Writing records directly

`connectDirect()` writes records straight to the DNS host using API
credentials for the zone. Supported write providers are **AWS Route 53 and
Hostinger**. Only record creation is implemented: the API cannot yet delete,
list or test records through a provider. For any other DNS host, show the user
the records to add manually and poll `status()`.

```ts
const { provider } = await vny.detect(domain);   // read-only, works for any host

await vny.domains.connectDirect({
  domain,
  provider: "aws",
  ip: MY_IP,
  providerConfig: { accessKey: process.env.AWS_ACCESS_KEY_ID!, secretKey: process.env.AWS_SECRET_ACCESS_KEY! },
});
```

Provider slugs and their credentials — the SDK types these as a discriminated
union, so mismatched credentials are a compile error:

| `provider` | `providerConfig` |
| --- | --- |
| `"aws"` | `{ accessKey, secretKey }` |
| `"hostinger"` | `{ apiToken }` |

Two traps the SDK guards for you, both verified against the Go factory:

- The Route 53 slug is **`"aws"`** — not `"route53"`, which the server rejects
  as unsupported.
- The credential keys must be **camelCase**. The Go structs carry no JSON tags,
  so Go matches keys case-insensitively but *not* across underscores:
  `access_key` silently unmarshals to empty and fails as "AWS Access Key is
  empty". `connectDirect()` rejects that before spending a request and tells you
  which key is wrong.

`connectDirect` writes the A record at `@`, the CNAME at `www`, and the TXT at
`@`. Those hosts are fixed by the API today, so it cannot point an arbitrary
subdomain.


## Playground

`playground/` is a runnable demo and the reference integration to copy: type
any domain, see its DNS host and current records, add the records shown, and
watch each one go live.

```bash
VNYTROS_API_KEY=... PLAYGROUND_IP=203.0.113.10 PLAYGROUND_TARGET=cname.yourapp.com \
  npm run playground          # http://localhost:5173
```

| Variable | Default | |
| --- | --- | --- |
| `VNYTROS_API_KEY` | required | create one on the API keys page of your deployment's dashboard; stays on the server |
| `VNYTROS_BASE_URL` | `http://localhost:8000` | your own Vnytros API (also `--base-url=URL`) |
| `PLAYGROUND_IP` / `PLAYGROUND_TARGET` | empty | prefilled A / `www` CNAME values |
| `PORT` | `5173` | |

It is a zero-dependency Node server, so it deploys anywhere Node runs.

## Express

```ts
import express from "express";
import { Vnytros } from "@vnytros/sdk";

const app = express();
const vny = new Vnytros({
  apiKey: process.env.VNYTROS_API_KEY!,
  baseUrl: process.env.VNYTROS_BASE_URL!,     // e.g. https://api.your-domain.com
  webhookSecret: process.env.VNYTROS_WEBHOOK_SECRET!,
});

app.post("/api/domain-status", express.json(), async (req, res) => {
  res.json(await vny.domains.status({ domain: req.body.domain, target: "cname.myplatform.com" }));
});

// Webhooks need the RAW body — the signature covers exact bytes, and
// re-serialising the parsed object can reorder keys and fail verification.
app.post("/webhooks/vnytros", express.raw({ type: "application/json" }), (req, res) => {
  let event;
  try {
    event = vny.webhooks.constructEvent(req.body, req.headers);
  } catch {
    return res.status(400).send("bad signature");
  }

  if (event.event === "domain.connected") markLive(event.data.domain);

  res.sendStatus(200);   // non-2xx is retried up to 5 times
});
```

Next.js equivalent — `await req.text()` gives you the raw body:

```ts
// app/api/webhooks/vnytros/route.ts
export async function POST(req: Request) {
  const event = vny.webhooks.constructEvent(await req.text(), req.headers);
  // ...
  return new Response(null, { status: 200 });
}
```

## Polling instead of webhooks

```ts
const status = await vny.domains.waitUntilConnected(
  { domain, ip: "203.0.113.10", txt: myTxtToken },
  { timeoutMs: 120_000, onPoll: (s, n) => console.log(`check ${n}:`, s.is_configured) },
);
```

A `VnytrosTimeoutError` here means *not propagated yet*, not *failed*. DNS can
take minutes.

## Telling the user what to expect first

```ts
const { provider, nameservers } = await vny.detect("shop.acme.com");

// If the zone is on Route 53 or Hostinger and you hold credentials
// for it, use connectDirect(). Otherwise show manual DNS instructions;
// provider?.name tells the user where to add them.
```

## Diagnostics

Every tool returns the same envelope (`verdict`, `score`, `grade`, `findings`),
so one UI component renders all of them:

```ts
const report = await vny.tools.dnsPropagation("shop.acme.com", "A");
console.log(report.verdict, report.grade);
for (const f of report.findings) console.log(f.status, f.title, f.remediation);
```

All 15 tools: `dnsPropagation`, `nameservers`, `cnameChain`, `caa`, `soa`,
`dnssec`, `emailAuth`, `spf`, `dkim`, `dmarc`, `domainInfo`, `tlsCertificate`,
`securityHeaders`, `redirectChain`, `ipInfo` (plus `catalogue()` to list them).
`emailAuth` and `dkim` accept optional DKIM selectors. These are rate limited
to 60 requests/minute per IP.

## Errors

```ts
import { VnytrosNotFoundError, VnytrosRateLimitError, VnytrosError } from "@vnytros/sdk";

try {
  await vny.domains.get("shop.acme.com");
} catch (err) {
  if (err instanceof VnytrosNotFoundError) return null;
  if (err instanceof VnytrosRateLimitError) return retryAfter(err.retryAfterMs);
  if (err instanceof VnytrosError) log({ code: err.code, requestId: err.requestId });
  throw err;
}
```

`VnytrosAuthError` · `VnytrosValidationError` · `VnytrosNotFoundError` ·
`VnytrosRateLimitError` · `VnytrosServerError` · `VnytrosConnectionError` ·
`VnytrosTimeoutError` · `VnytrosSignatureVerificationError`, all extending
`VnytrosError` with `code`, `statusCode`, and `requestId`.

### Retries

Idempotent requests (all GETs, plus `status()`) are retried twice by default
with jittered exponential backoff. `connectDirect()` and `verify()` are
**never** retried — each writes state, so a blind replay would do that twice. A `Retry-After` longer than 8s is returned to you as
a `VnytrosRateLimitError` rather than blocking your request.

## API

```ts
new Vnytros({ apiKey, baseUrl, timeoutMs?, maxRetries?, webhookSecret?, fetch? })

vny.detect(domain)                          // provider behind a domain
vny.dnsLookup(domain)                       // current records

vny.domains.connectDirect({ domain, provider: "aws" | "hostinger", providerConfig, ip?, target?, txt? })
vny.domains.status({ domain, ip?, target?, txt? })
vny.domains.verify(domain)
vny.domains.list()
vny.domains.get(domain)
vny.domains.waitUntilConnected(params, { timeoutMs?, intervalMs?, onPoll? })

vny.tools.*                                 // diagnostics, see above
vny.webhooks.constructEvent(rawBody, headers, { secret?, toleranceSec? })
```

Every method takes an optional trailing `AbortSignal`.

Response types mirror the API's JSON exactly, including its snake_case field
names (`is_configured`, `domain_name`), so there is no mapping
layer that can drift from the server.

## Webhook events

| Event | Fires when | `data` |
| --- | --- | --- |
| `domain.connected` | records verified | `{ domain, user_id }` |

Signed as `X-Vnytros-Signature: t=<unix>,v1=<hmac-sha256 of "t.body">`.
`constructEvent` checks the digest in constant time and rejects timestamps more
than 300s old, so a captured delivery cannot be replayed. Multiple `v1=` values
are accepted, which lets the API rotate secrets without downtime.

## Development

```bash
npm install
npm run typecheck
npm test          # builds, then runs node:test
```

The webhook suite verifies against `tests/fixtures/go-signature.json`, generated
by the Go worker's own test. If the signing scheme changes on either side, one
of the two suites fails. Regenerate with:

```bash
cd ../server && FIXTURE_OUT=$PWD/../sdk/tests/fixtures/go-signature.json \
  go test ./internal/worker/ -run TestWriteSignatureFixture
```

Run it from `sdk/` in a checkout of the monorepo; the server lives beside it in
`server/`.

The path is absolute because `go test` runs inside the package directory.

## Contributing and license

See the monorepo's
[CONTRIBUTING.md](https://github.com/vnytros/vnytros/blob/main/CONTRIBUTING.md).
Released under the [MIT License](./LICENSE); the rest of the monorepo is
AGPL-3.0-only, but this folder is MIT so you can use the SDK in any
application.
