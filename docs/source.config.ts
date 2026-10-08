import { defineDocs, defineConfig } from "fumadocs-mdx/config";
import lastModified from "fumadocs-mdx/plugins/last-modified";

export const docs = defineDocs({
  dir: "content/docs",
});

/**
 * `lastModified` reads each .mdx file's real last-commit date out of git and
 * exposes it as `page.data.lastModified`.
 *
 * This exists for the sitemap. Without it every entry carried the build
 * timestamp, which told crawlers all 30 documentation pages changed on every
 * deploy — indistinguishable from telling them nothing, and useless as a
 * change-monitoring signal for us.
 *
 * On Vercel this needs `VERCEL_DEEP_CLONE=true`, otherwise the shallow clone
 * has no history to read and every page comes back `undefined`. The sitemap
 * falls back to file mtime in that case rather than emitting a wrong date.
 */
export default defineConfig({
  plugins: [lastModified()],
});
