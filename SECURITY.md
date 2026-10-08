# Security policy

## Reporting a vulnerability

Please report security issues privately through GitHub's private
vulnerability reporting:
<https://github.com/vnytros/vnytros/security/advisories/new>. Do not
open a public issue or pull request for a vulnerability.

Include what you can of:

- the affected part of the repository (`server/`: API, OAuth, MCP server,
  mailforge, worker; `dashboard/`; `website/`; `docs/`; `sdk/`) and the
  version or commit;
- steps to reproduce, or a proof of concept;
- the impact as you understand it.

We will acknowledge your report, keep you informed while we work on a fix, and
credit you in the release notes if you would like.

## Scope

This policy covers all the code in this repository: `server/`, `dashboard/`,
`website/`, `docs/` and `sdk/`. The project does not operate
any hosted instance: every deployment is run by its own operator, so report
problems with a specific deployment to whoever runs it, and problems in the code
itself here.

## Operator notes

- Generate your own signing keys and secrets (`make keys` and `.env.example`
  in `server/`).
  Never reuse values from another deployment or from git history.
- `keys/` and `.env.*` files are gitignored and must never be committed.
