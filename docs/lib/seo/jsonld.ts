import { DOCS_URL, SITE_URL } from "@/lib/config";

/**
 * Structured data for the docs site.
 *
 * The docs shipped with no JSON-LD at all, which left the two properties
 * looking like unrelated hosts to anything reading structured data: the
 * marketing site declared an Organization, and docs.vnytros.dev declared
 * nothing.
 *
 * The fix is not to redeclare the company here. `ORG_ID` is the *same* `@id`
 * the marketing site publishes, so both origins describe one entity and the
 * docs nodes reference it rather than competing with it. vnytros.dev stays
 * canonical for the Organization's own fields (name, logo, description); this
 * file only points at them.
 *
 * Deliberately no `TechArticle`/`Article`: those need real `datePublished` /
 * `dateModified` / `headline` values to be valid, and the MDX frontmatter in
 * content/docs carries no dates. Inventing one to satisfy a schema validator
 * is exactly the kind of unverifiable claim the marketing site was just
 * cleaned of. Add `Article` once the frontmatter tracks real dates.
 */
const ORG_ID = `${SITE_URL}/#organization`;
const DOCS_SITE_ID = `${DOCS_URL}/#website`;

/** Site-wide graph — emitted once, from the root layout. */
export const docsSiteJsonLd = {
  "@context": "https://schema.org",
  "@graph": [
    {
      "@type": "WebSite",
      "@id": DOCS_SITE_ID,
      name: "Vnytros Docs",
      url: DOCS_URL,
      publisher: { "@id": ORG_ID },
      // Ties this host to the marketing site's WebSite node, so the docs read
      // as part of one property rather than a separate site that happens to
      // share a brand name.
      isPartOf: { "@id": `${SITE_URL}/#website` },
      inLanguage: "en",
    },
  ],
};

/**
 * Per-page graph.
 *
 * The breadcrumb is intentionally two levels (Docs → this page) rather than
 * mirroring the full folder nesting. A deeper trail would mean resolving each
 * ancestor's display title out of the page tree, and a breadcrumb naming a
 * folder that has no page behind it is worse than a shallow one that is true.
 */
export function docsPageJsonLd(opts: {
  /** Route path beginning with a slash, e.g. "/docs/api-keys". */
  path: string;
  title: string;
  description?: string;
}) {
  const url = `${DOCS_URL}${opts.path}`;

  return {
    "@context": "https://schema.org",
    "@graph": [
      {
        "@type": "WebPage",
        "@id": `${url}#page`,
        name: opts.title,
        url,
        ...(opts.description ? { description: opts.description } : {}),
        isPartOf: { "@id": DOCS_SITE_ID },
        about: { "@id": ORG_ID },
        inLanguage: "en",
      },
      {
        "@type": "BreadcrumbList",
        itemListElement: [
          {
            "@type": "ListItem",
            position: 1,
            name: "Docs",
            item: `${DOCS_URL}/docs`,
          },
          {
            "@type": "ListItem",
            position: 2,
            name: opts.title,
            item: url,
          },
        ],
      },
    ],
  };
}
