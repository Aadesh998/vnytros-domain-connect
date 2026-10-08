import type { MetadataRoute } from "next";
import { SITE_URL } from "@/lib/config";

/** Serves /robots.txt. */
export default function robots(): MetadataRoute.Robots {
  return {
    rules: [{ userAgent: "*", allow: "/" }],
    sitemap: `${SITE_URL}/sitemap.xml`,
  };
}
