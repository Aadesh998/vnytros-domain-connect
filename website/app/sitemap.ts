import type { MetadataRoute } from "next";
import { SITE_URL } from "@/lib/config";

/** Serves /sitemap.xml: the site has exactly two pages. */
export default function sitemap(): MetadataRoute.Sitemap {
  return [{ url: SITE_URL }];
}
