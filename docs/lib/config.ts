/**
 * Absolute origin for this docs site.
 *
 * Next needs this to turn the relative canonical/OG paths in `generateMetadata`
 * into absolute URLs. Without a `metadataBase` those resolve against
 * `localhost` at build time, which is why the deployed pages carried no usable
 * canonical at all.
 */
export const DOCS_URL = "https://docs.vnytros.dev";

/** The marketing site. Docs link back to it; keep the two in sync. */
export const SITE_URL = "https://vnytros.dev";
