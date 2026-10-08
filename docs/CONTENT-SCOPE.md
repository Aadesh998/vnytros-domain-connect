# What belongs in the public docs

This file is **not published**. Fumadocs only builds `content/docs/`
(see `source.config.ts`), so this stays internal.

It exists because "document the API" and "document the system" are two
different jobs, and only the first one belongs in the public docs.

## The rule

Document the **contract**, not the **implementation**.

A reader of these docs is someone integrating against **their own
self-hosted Vnytros API**. The project runs no public instance (no hosted API,
MCP server, mail, or accounts), so:

- Every example uses the placeholder base URL `https://api.example.com`
  (MCP: `https://mcp.example.com/mcp`). Never use a real project-owned host
  in an example. The intro and quickstart say once to substitute your own URL.
- Accounts and API keys come from "your deployment's dashboard" or the API —
  never a "sign up" link or a link to a hosted dashboard.
- No "free hosted tools", "public API", or "use our API" claims. The project
  runs no website or hosted service; its home is the GitHub repository
  (https://github.com/vnytros/vnytros). Point to a docs page or GitHub instead.
- The quickstart starts with [Self-hosting](content/docs/self-hosting.mdx).

They need to know what to send, what comes back, what can go wrong, and
which behaviours are asynchronous. They do not need to know how any of it
is built, and telling them creates two problems: it invites calls to
endpoints that are not for them, and it turns internal refactors into
documentation bugs.

Include a fact only if a caller's own code would be wrong without it.

**In scope**
- Path, method, auth requirement, parameters, response shape, error codes.
- Observable async behaviour — e.g. `/v1/connect/direct` queues verification,
  so the caller must poll rather than assume completion. That changes how
  they write their code, so it is contract.
- Concepts a caller cannot use the API without — Domain Connect, the report
  envelope, the scoring model, pagination, roles insofar as they explain a
  `403`.

**Out of scope**
- Storage, queues, workers, schedulers, deploy topology, provider SDKs.
- Wording that describes internal shape rather than the visible payload
  ("each log row" → "each entry").
- Endpoints the reader cannot legitimately call (see below).

## Deliberately undocumented endpoints

These exist in `server/internal/server/server.go` (in the monorepo) and are intentionally absent
from the docs. This is not a backlog — it is the boundary. If you add one
here, add the reason too.

| Endpoint | Why not |
| --- | --- |
| `POST /v1/vnytros/webhook` | Test receiver; logs headers. No contract to document. |
| `GET /v1/dashboard/domains` | Session-cookie auth for our own UI. Duplicates the API-key route `GET /v1/connect/domains`, which *is* documented. |
| `GET /v1/dashboard/domain` | As above; duplicate of `GET /v1/connect/domain`. |
| `POST /v1/dashboard/domain/verify` | As above; duplicate of `POST /v1/connect/verify`. |

Vnytros is a free, open-source project: there are no plans, billing, or
top-up endpoints, and no donation or support page; the docs must not
describe or link to any. Likewise the Domain Connect *apply* flow (`POST /v1/connect/auto`,
`GET /v1/callback`) was removed from the server; do not document it. Direct
Connect is documented only for the providers the server actually implements
(`aws`, `hostinger`). Name.com was removed (its adapter was a stub that wrote
nothing). Only record creation is implemented; the docs must say that
delete, list, and test are not.

`self-hosting.mdx` is the one page that describes how to *run* the system
rather than call it. It is step 1 of the quickstart. It stays at the level a self-hoster needs
(clone, keys, compose, env var names, make targets, creating an account) and points to each repo's README for
anything deeper.

The `/v1/oauth/*` and `/.well-known/*` routes are documented in
`oauth.mdx` because third-party MCP clients genuinely complete that flow.

## Keeping this honest

`npm run audit:docs` diffs the router against the docs and fails on any
endpoint that is neither documented nor listed above. Run it after
touching routes. It catches the two failure modes that matter: an endpoint
documented with the wrong method, and a new endpoint that silently became
public without anyone deciding.
