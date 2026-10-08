import { defineCloudflareConfig } from "@opennextjs/cloudflare";
import r2IncrementalCache from "@opennextjs/cloudflare/overrides/incremental-cache/r2-incremental-cache";

/*
 * OpenNext adapter config for Cloudflare Workers.
 *
 * The incremental cache lives in R2 (bucket `web-opennext-cache`, bound below
 * in wrangler.jsonc as NEXT_INC_CACHE_R2_BUCKET). Every route on this site is
 * statically prerendered today, so the cache is mostly a prewarmed copy of the
 * build output — but keeping it means adding an ISR/`revalidate` page later
 * does not require a config change plus a bucket creation under time pressure.
 */
/*
 * `buildCommand` points at `build:next`, not the default `npm run build`,
 * because `build` itself is now `opennextjs-cloudflare build` — the Cloudflare
 * pipeline's build step is hardwired to `npm run build`, and running plain
 * `next build` there left no `.open-next` directory for the deploy step, which
 * then failed with "Could not find compiled Open Next config". Leaving the
 * default here would make that script recurse into itself.
 */
const openNextConfig = {
  ...defineCloudflareConfig({
    incrementalCache: r2IncrementalCache,
  }),
  buildCommand: "npm run build:next",
};

export default openNextConfig;
