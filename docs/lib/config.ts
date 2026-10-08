/**
 * Absolute origin for this docs site.
 *
 * Next needs this to turn the relative canonical/OG paths in `generateMetadata`
 * into absolute URLs. Set NEXT_PUBLIC_DOCS_URL to the origin you deploy the
 * docs to (it is inlined at build time). The default matches the local dev
 * server (`npm run dev -- -p 3001`).
 */
export const DOCS_URL = (
  process.env.NEXT_PUBLIC_DOCS_URL || "http://localhost:3001"
).replace(/\/+$/, "");

/**
 * The project landing page (website/). Used only for structured-data ids.
 * Set NEXT_PUBLIC_SITE_URL to where you deploy it; defaults to its local dev
 * server.
 */
export const SITE_URL = (
  process.env.NEXT_PUBLIC_SITE_URL || "http://localhost:3000"
).replace(/\/+$/, "");

/** The project's home: the GitHub monorepo. */
export const REPO_URL = "https://github.com/vnytros/vnytros";
