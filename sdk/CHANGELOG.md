# Changelog

All notable changes to `@vnytros/sdk` are documented here. The
project follows [Semantic Versioning](https://semver.org/); while it is below
1.0, breaking changes bump the minor version.

## Unreleased

- Source moved to the `sdk/` folder of the
  [vnytros/vnytros](https://github.com/vnytros/vnytros) monorepo; package
  metadata (`repository.directory`, `homepage`, `bugs`) updated to match.
- Added a `prepack` script that runs the build, so `npm pack` always produces
  a tarball containing `dist/`. The README documents installing the SDK
  without the npm registry (tarball or local folder).

## 0.2.0

Vnytros is now a free, open-source project. This release aligns the SDK with
the open-source API.

### Renamed (breaking)

- The package is now published as `@vnytros/sdk` (previously
  `@vnytros/domain-connect`). Update your install and imports:
  `npm uninstall @vnytros/domain-connect && npm install @vnytros/sdk`.

### Required `baseUrl` (breaking)

- `baseUrl` is now required; there is no hosted Vnytros API. Pass the URL of
  your own deployment, e.g. `new Vnytros({ apiKey, baseUrl:
  "https://api.your-domain.com" })` (or `http://localhost:8000` locally). The
  constructor throws a `VnytrosError` (`INVALID_INPUT`) when it is missing or
  not an absolute `http(s)` URL. The `DEFAULT_BASE_URL` export was removed.
- The playground reads the API URL from `--base-url=URL` or `VNYTROS_BASE_URL`
  and defaults to `http://localhost:8000`.

### Removed (breaking)

- `vny.domains.connect()` and the `ConnectParams` / `ConnectResult` types. The
  Domain Connect apply flow (`POST /v1/connect/auto`) and its callback
  (`GET /v1/callback`) were removed from the API.
- The `@vnytros/domain-connect/browser` entry point (`createDomainConnect`,
  `connect`, `connectInPopup`, `redirectToProvider`, `DomainConnectClientError`)
  and the standalone `dist/cdn/domain-connect.global.js` build. Both existed
  only to drive the apply flow.

- Name.com (`"namecom"`) is no longer a `connectDirect()` provider, and the
  `NameComProviderConfig` type was removed: the server's Name.com adapter was a
  stub that wrote nothing. `DirectProviderSlug` is now `"aws" | "hostinger"`.

### Added

- Email authentication tools: `vny.tools.emailAuth()`, `spf()`, `dkim()` and
  `dmarc()`, so the tools resource now covers all 15 `/v1/tools/*` endpoints.

### Changed

- `connectDirect()` documents its supported write providers explicitly: AWS
  Route 53 (`"aws"`) and Hostinger (`"hostinger"`). Only record creation is
  implemented server-side; delete, list and test are not yet.
- The playground is now a detect, add records, and verify demo with no apply
  step.
- Docs no longer mention plans, quotas or pricing. Added `LICENSE` (MIT),
  `CONTRIBUTING.md` and repository metadata.

## 0.1.0

- Initial release.
