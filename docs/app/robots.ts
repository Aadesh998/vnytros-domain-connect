import type { MetadataRoute } from "next";
import { DOCS_URL } from "@/lib/config";

/**
 * robots.txt for the docs site, pointing crawlers at the sitemap. The
 * origin comes from NEXT_PUBLIC_DOCS_URL (see lib/config.ts). NOTE: if your
 * host (e.g. Cloudflare's managed robots.txt) serves its own robots.txt it
 * may shadow this route.
 */
export const dynamic = "force-static";

export default function robots(): MetadataRoute.Robots {
  return {
    rules: {
      userAgent: "*",
      allow: "/",
    },
    sitemap: `${DOCS_URL}/sitemap.xml`,
    host: DOCS_URL,
  };
}
