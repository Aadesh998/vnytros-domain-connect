# mailforge

Email campaign service for Vnytros, served by the api under
`<BASE_URL>/api/...` (e.g. `https://api.<your-domain>/api/...`). Tracking pixels
are minted on `BASE_URL`, the api's own origin.

Formerly the standalone `vnytros-mail` repo. It now lives here as
`internal/mailforge` and is **not a separate service or binary**: `cmd/server`
mounts its router at `/api` and serves it on `:8000` alongside the api.

That mount is possible without touching a single handler because the two halves
never shared a path — the api serves `/v1`, `/oauth` and `/.well-known`, and
every mailforge route is under `/api`. A gin engine is an `http.Handler`, so
chi hosts it directly. `internal/server/mailforge_mount_test.go` guards this.

There is no separate host any more: everything is the api's origin.

It is a **stateless HTTP API only**. It has no UI, no background senders and no
in-process queues. (It could once be scaled to zero independently; sharing the
api's process trades that away for one deployable, one config and one DB pool.)
Delivery and open tracking are performed by this repo's `cmd/worker`.

## How it fits together

```
dashboard (dashboard/)
         │
         │ JWT
         ▼
┌────────────────── cmd/server (:8000) ──────────────────┐
│  /v1, /oauth, /.well-known   the api                   │
│  /api                        mailforge (this package)  │
└────────────────────────────┬───────────────────────────┘
                             │ publishes campaign.send
                             │            campaign.track
                             ▼
                        cmd/worker ──> parallel SMTP
                             │
                ┌────────────┴─────────────┐
                │    one shared Postgres   │
                │ (owned by cmd/migration) │
                └──────────────────────────┘
```

Three facts follow from this and explain most of the code:

1. **No login here.** `cmd/server` (the api) issues access tokens; this service
   only *verifies* them with the same `JWT_SECRET`. See
   `internal/mailforge/middleware/auth.go`.
2. **One database, and the api owns the schema.** `internal/models/mail.go` is
   the canonical definition of the `mail_*` tables and `cmd/migration` is the
   **only** migrator — mailforge has no migration command. `model/model.go` used
   to hand-copy those structs, because Go could not share a type across the old
   module boundary; now that both halves are one module it is a set of **type
   aliases** onto `internal/models`, so the definitions cannot drift.
   Tables owned by the api (such as `users`) are not modelled here at all.
3. **Nothing is sent in-process.** `SendCampaign` persists the recipient ledger,
   publishes batches, and returns `202 Accepted`. Progress is observed by polling
   the campaign or its analytics.

The auth cookie is issued on `AUTH_COOKIE_DOMAIN` (e.g. `.<your-domain>`), so
the api's origin receives it automatically; a bearer token in the
`Authorization` header works too.

## Layout

| Path | Purpose |
|---|---|
| `server/server.go` | `Init` (config, shared DB, publisher, tracer) + `NewHandler` |
| `middleware/auth.go` | verifies the api's JWT |
| `authctx` | the authenticated caller on the request context |
| `model` | type aliases onto `internal/models` (api owns the schema) |
| `repositary` | every query is scoped by `user_id` |
| `services` | campaign, template, sender, branding, analytics logic |
| `queue` | RabbitMQ **publisher only** (never consumes) |
| `internal/jobs` | wire contract, now shared with the api rather than mirrored |

Paths above are relative to `internal/mailforge` except where shown from the
module root.

Tenancy lives in exactly one place: the auth middleware resolves the caller, and
every repository call folds that `user_id` into its `WHERE` clause. A zero user
id (unauthenticated) matches no rows.

## API

All routes need a token except the two public ones.

```
GET    /api/health                      public
GET    /api/track                        public — tracking pixel

GET    /api/campaign                     ?last_id&limit
GET    /api/campaign/draft
POST   /api/campaign
GET    /api/campaign/:id
PUT    /api/campaign/:id
DELETE /api/campaign/:id
POST   /api/campaign/:id/send            multipart "file" (CSV) [?smtp_config_id]  -> 202
GET    /api/campaign/:id/analytics       ?days&interval=hour|day
GET    /api/campaign/:id/recipients      ?status=queued|sent|failed&offset&limit

GET    /api/template                     ?last_id&limit
GET    /api/template/draft
POST   /api/template
GET    /api/template/:id
PUT    /api/template/:id
DELETE /api/template/:id

GET    /api/settings/smtp                list senders
POST   /api/settings/smtp                add sender
POST   /api/settings/smtp/seed           import the SMTP_* fallback as sender #1
GET    /api/settings/smtp/:id
PUT    /api/settings/smtp/:id            blank password keeps the stored one
POST   /api/settings/smtp/:id/default
DELETE /api/settings/smtp/:id

GET    /api/settings/branding            watermark settings
PUT    /api/settings/branding            any signed-in account

GET    /api/analytics/overview           ?days&interval
```

`/api/track` is unauthenticated because a recipient's mail client has no
session. It therefore takes ownership from the campaign row rather than trusting
a user id in the query string, and always returns the pixel so a failure never
shows as a broken image in someone's inbox.

## Environment

`ENV` selects the file (`.env.production` by default), and real deployments pass
these as process variables instead.

Since the merge there is **one** `.env.<env>` at the root of `server/` shared by every
binary, so group 1 below is no longer something to keep in sync by hand — it is
literally the same file the api reads. Everything else has a default in
`config/config.go` — don't restate defaults in `.env`.

**Group 1 — shared with the api, read from the same file.** The database is the
same one, the tokens were signed by `cmd/server`, and `RABITMQ` is where
`cmd/worker` listens. A mismatch means 401s on every route, or campaigns that
queue and never send.

| Variable | Why |
|---|---|
| `DB_*` | read by the api's config, not this one — mailforge is handed the api's `*gorm.DB` |
| `JWT_SECRET` | tokens are verified here, never issued |
| `RABITMQ` | `cmd/worker` consumes what we publish (e.g. the broker from `deploy/docker-compose.infra.yaml`) |

**Group 2 — this deployment's identity.** No sensible defaults.

| Variable | Notes |
|---|---|
| `BASE_URL` | public origin baked into tracking pixels — the api's own origin. Cannot be derived from a request (`cmd/worker` renders pixels in a background job). Empty or wrong = no open tracking, silently |
| `ALLOWED_ORIGINS` | CORS allowlist, comma separated, exact origins. No wildcard here: the dashboard sends credentials. (The api's `/v1` CORS reads the same list and additionally accepts `*.example.com` entries.) |

**Group 3 — defaulted, normally left unset.**

| Variable | Default |
|---|---|
| `SEND_BATCH_SIZE` | `50` recipients per `campaign.send` job |
| `OTEL_ENDPOINT` | `localhost:4318` |
| `MAIL_WATERMARK_IMAGE_URL` / `_LINK_URL` | empty — no image means no watermark footer |
| `MAIL_WATERMARK_LABEL` | `Sent with Vnytros` |
| `MAIL_WATERMARK_ALLOWED_IMAGE_URLS` | empty — extra images users may choose, comma separated |

**Optional.** The service-wide fallback sender, used only by
`POST /api/settings/smtp/seed` to import a user's first credential, now reads
the api's `SMTP_FROM_EMAIL` / `SMTP_PASSWORD` / `SMTP_HOST` / `SMTP_PORT` —
the two repos had been carrying the same SMTP credential under two names.
The old `EMAIL_*` keys still work as a fallback, so pre-merge env files keep
working. Users normally add their own sender in the dashboard.

`LoadConfig` warns at boot about anything missing from groups 1 and 2, so a
misconfigured deployment says so once rather than failing per request.

## Local development

```bash
docker compose up -d db rabbitmq      # shared Postgres + broker
make migration                        # cmd/migration owns the mail_* schema
make run                              # api + mailforge together on :8000
```

Its routes are then at `http://localhost:8000/api/...`.

There is no seed step. Templates are per-user and are created through the
dashboard's Templates page.

Because this service does not migrate, the `mail_*` tables must exist before it
starts. In deployment that means **`make migration` runs first**.

Campaigns only actually send when the worker is running:

```bash
make run-worker
```

```bash
make test        # unit + route tests (whole module)
make tidy
```

## Watermark rules

Every signed-in account may change or switch off the watermark. The default
image and link come from `MAIL_WATERMARK_IMAGE_URL` / `MAIL_WATERMARK_LINK_URL`;
with no image configured the footer is omitted. `watermark_image_url` is
validated against an allowlist built from `MAIL_WATERMARK_IMAGE_URL` plus
`MAIL_WATERMARK_ALLOWED_IMAGE_URLS` (`mailforge/services.AllowedWatermarkImageURLs`),
images the operator hosts themselves — custom uploads are not implemented. Every branding value is HTML-escaped before it reaches
outbound mail.
