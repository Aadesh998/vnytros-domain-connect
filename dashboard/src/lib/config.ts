/**
 * Deployment configuration, read from Vite env vars at build time.
 *
 * Vnytros is self-hosted: there is no hosted API to fall back to, so a missing
 * VITE_API_BASE_URL is a configuration error, not a reason to call someone
 * else's server. `vite build` refuses to run without it (see vite.config.ts),
 * and in dev `main.tsx` renders a config error screen instead of the app.
 */

function readUrl(value: unknown): string {
  return typeof value === "string" ? value.trim().replace(/\/+$/, "") : ""
}

/** Base URL of your Vnytros API (`/v1/*`). Empty when not configured. */
export const API_BASE_URL = readUrl(import.meta.env.VITE_API_BASE_URL)

/**
 * Base URL for the mail routes (`/api/*`). Mail is served by the same API, so
 * this defaults to API_BASE_URL.
 */
export const MAILFORGE_BASE_URL =
  readUrl(import.meta.env.VITE_MAILFORGE_BASE_URL) || API_BASE_URL

export const isConfigured = API_BASE_URL !== ""
