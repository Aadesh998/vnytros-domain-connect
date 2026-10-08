import { statSync } from "node:fs";
import path from "node:path";
import type { MetadataRoute } from "next";
import { source } from "@/lib/source";
import { DOCS_URL } from "@/lib/config";

/**
 * Sitemap for the docs subdomain.
 *
 * Pages come from the Fumadocs loader rather than a hand-written list, so a new
 * .mdx file is included the moment it is added.
 *
 * `lastmod` is the file's real last-commit date, injected by the
 * `lastModified` plugin in source.config.ts. It replaces the build timestamp
 * that every entry used to share — a value that claimed all 30 pages changed
 * on every deploy and so carried no information at all.
 *
 * `priority` and `changefreq` are deliberately absent: Google ignores both, and
 * a hand-tuned ladder over 30 reference pages is upkeep with no payoff.
 */
export const dynamic = "force-static";

const CONTENT_DIR = path.join(process.cwd(), "content", "docs");

/**
 * Fallback for when git history is unavailable — a shallow CI clone, or a build
 * without `VERCEL_DEEP_CLONE=true`. File mtime is less meaningful than a commit
 * date but still beats "now", which would re-stamp every page on every deploy.
 */
function mtimeFor(page: { path?: string }): Date | undefined {
  if (!page.path) return undefined;
  try {
    return statSync(path.join(CONTENT_DIR, page.path)).mtime;
  } catch {
    return undefined;
  }
}

export default function sitemap(): MetadataRoute.Sitemap {
  return source.getPages().map((page) => {
    const gitTime = (page.data as { lastModified?: Date | string })
      .lastModified;

    return {
      url: new URL(page.url, DOCS_URL).toString(),
      lastModified: gitTime ? new Date(gitTime) : mtimeFor(page),
    };
  });
}
