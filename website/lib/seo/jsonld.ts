import { GITHUB_ORG_URL, REPO_URL, SITE_URL } from "@/lib/config";

/**
 * Site-wide structured data: the project as an Organization, and the server
 * as SoftwareSourceCode. Vnytros is self-host only, so there is no hosted
 * application to describe and nothing sold, hence no Offer or rating nodes.
 */

const ORG_ID = `${SITE_URL}/#organization`;

const organization = {
  "@type": "Organization",
  "@id": ORG_ID,
  name: "Vnytros",
  alternateName: ["vnytros"],
  url: SITE_URL,
  logo: `${SITE_URL}/logo-512.png`,
  description:
    "Open-source, self-hosted DNS and email-authentication toolkit with a built-in mail platform.",
  sameAs: [GITHUB_ORG_URL, REPO_URL],
};

const serverSourceCode = {
  "@type": "SoftwareSourceCode",
  "@id": `${SITE_URL}/#server`,
  name: "Vnytros server",
  description:
    "Self-hosted Vnytros server: DNS provider detection, DNS record creation on AWS Route 53 and Hostinger, domain verification with signed webhooks, 15 DNS/email/TLS/HTTP diagnostic checks, a REST API, an MCP server and Vnytros Mail.",
  codeRepository: REPO_URL,
  programmingLanguage: "Go",
  license: "https://spdx.org/licenses/AGPL-3.0-only.html",
  isAccessibleForFree: true,
  author: { "@id": ORG_ID },
};

/** Root-level graph, emitted once from the root layout. */
export const siteJsonLd = {
  "@context": "https://schema.org",
  "@graph": [organization, serverSourceCode],
};
