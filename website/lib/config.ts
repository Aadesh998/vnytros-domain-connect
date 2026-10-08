/**
 * Central site config. Vnytros is self-host only: this site is a static
 * project page and talks to no API.
 */

/** Canonical public site — must match what actually serves traffic. */
export const SITE_URL = "https://vnytros.dev";

/** The GitHub organisation (the project's identity on GitHub). */
export const GITHUB_ORG_URL = "https://github.com/vnytros";

/** All source code lives in this one monorepo, organised by folder. */
export const REPO_URL = "https://github.com/vnytros/vnytros";

/** Link to a folder (or file) on the monorepo's default branch. */
export const repoTree = (path: string) => `${REPO_URL}/tree/main/${path}`;
export const repoBlob = (path: string) => `${REPO_URL}/blob/main/${path}`;

/**
 * Where to send a reader who clicks "Docs". The bare docs origin 301s to
 * `/docs`, so link here to skip the hop.
 */
export const DOCS_ENTRY = "https://docs.vnytros.dev/docs";

/** The folders of the monorepo, in the order the site lists them. */
export const REPOS = [
  { name: "server/", href: repoTree("server"), body: "Go REST API, MCP server, worker and Vnytros Mail", license: "AGPL-3.0" },
  { name: "dashboard/", href: repoTree("dashboard"), body: "Web dashboard for your server", license: "AGPL-3.0" },
  { name: "sdk/", href: repoTree("sdk"), body: "JavaScript SDK, @vnytros/sdk", license: "MIT" },
  { name: "docs/", href: repoTree("docs"), body: "Documentation (docs.vnytros.dev)", license: "AGPL-3.0" },
  { name: "website/", href: repoTree("website"), body: "This project page", license: "AGPL-3.0" },
] as const;
