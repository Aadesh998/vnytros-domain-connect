# Domain Connect templates

Put your own service template here as `<DC_PROVIDER_DOMAIN>.custom-domain.json`.
The server serves it at `/.well-known/domainconnect/<DC_PROVIDER_DOMAIN>/custom-domain`
when `DC_PROVIDER_DOMAIN` is set; with it unset the route is disabled.

The Vnytros project's own template (providerId `vnytros.dev`) is accepted in the
official Domain-Connect/Templates repository:
[`vnytros.dev.custom-domain.json`](https://github.com/Domain-Connect/Templates/blob/master/vnytros.dev.custom-domain.json),
added in [PR #1064](https://github.com/Domain-Connect/Templates/pull/1064) and
extended in [PR #1166](https://github.com/Domain-Connect/Templates/pull/1166).
It belongs to that providerId, so on your own domain, write your own template
and submit it upstream.
