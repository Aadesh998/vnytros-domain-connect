import { permanentRedirect } from "next/navigation";

/**
 * The docs root is not a page — everything lives under /docs.
 *
 * `permanentRedirect` (308), not `redirect` (307). The distinction matters to
 * crawlers rather than to browsers: a 307 says "this moved for now, keep
 * asking for the old URL", so search engines keep the bare origin in their
 * crawl rotation and pass no signal through to the target. This redirect is
 * structural and is not going to change, which is what 308 means. It also
 * matches the marketing site's own convention — next.config.ts there uses 308
 * for both the www→apex and trailing-slash redirects.
 */
export default function RootPage() {
  permanentRedirect("/docs");
}
