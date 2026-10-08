#!/usr/bin/env node
/**
 * Diffs the Go router against the published docs.
 *
 * Catches the two failure modes that actually cause damage:
 *
 *   1. An endpoint documented with a method or path the router does not
 *      serve — readers write code against a 404.
 *   2. A new route that became public without anyone deciding whether it
 *      should be documented.
 *
 * (2) is why UNDOCUMENTED_BY_DESIGN is an explicit list rather than an
 * ignore pattern: a route must be either documented or deliberately
 * excluded, and adding one to the list is a decision with a reason
 * recorded next to it in CONTENT-SCOPE.md.
 *
 * The API lives in the same monorepo (`server/`), so the router is always
 * at ../server/internal/server/server.go relative to this docs folder. If it
 * is missing the audit FAILS: a silent skip would let docs drift unnoticed.
 */

import { readFileSync, readdirSync, statSync, existsSync } from "node:fs";
import { join, dirname, relative } from "node:path";
import { fileURLToPath } from "node:url";

const here = dirname(fileURLToPath(import.meta.url));
const docsRoot = join(here, "..");
const ROUTER = join(docsRoot, "../server/internal/server/server.go");
const CONTENT = join(docsRoot, "content/docs");

/** Routes that exist but must NOT be documented. See CONTENT-SCOPE.md. */
const UNDOCUMENTED_BY_DESIGN = new Set([
  "POST /v1/vnytros/webhook",
  "GET /v1/dashboard/domains",
  "GET /v1/dashboard/domain",
  "POST /v1/dashboard/domain/verify",
]);

if (!existsSync(ROUTER)) {
  console.error(`audit failed: router not found at ${ROUTER}`);
  console.error(
    "      Run this from docs/ inside a full checkout of the vnytros monorepo\n" +
      "      (the router is server/internal/server/server.go). If the router\n" +
      "      moved, update ROUTER in scripts/audit-api-docs.mjs.",
  );
  process.exit(1);
}

// ── real routes ────────────────────────────────────────────────────────
// Matches `r.Get("/v1/...")`, `tools.Get("/v1/...")`, any receiver name.
const router = readFileSync(ROUTER, "utf8");
const real = new Set();
for (const m of router.matchAll(
  /\b\w+\.(Get|Post|Patch|Put|Delete)\(\s*"(\/v1[^"]*)"/g,
)) {
  real.add(`${m[1].toUpperCase()} ${m[2]}`);
}

// ── documented routes ──────────────────────────────────────────────────
function mdxFiles(dir) {
  return readdirSync(dir).flatMap((name) => {
    const p = join(dir, name);
    return statSync(p).isDirectory()
      ? mdxFiles(p)
      : name.endsWith(".mdx")
        ? [p]
        : [];
  });
}

const documented = new Map();
for (const file of mdxFiles(CONTENT)) {
  const body = readFileSync(file, "utf8");
  for (const m of body.matchAll(
    /\b(GET|POST|PATCH|PUT|DELETE) (\/v1[a-z0-9/{}._-]*)/g,
  )) {
    const key = `${m[1]} ${m[2]}`;
    if (!documented.has(key)) documented.set(key, new Set());
    documented.get(key).add(relative(CONTENT, file));
  }
}

// ── report ─────────────────────────────────────────────────────────────
const wrong = [...documented.keys()].filter((k) => !real.has(k)).sort();
const unclassified = [...real]
  .filter((r) => !documented.has(r) && !UNDOCUMENTED_BY_DESIGN.has(r))
  .sort();
const staleAllowlist = [...UNDOCUMENTED_BY_DESIGN]
  .filter((r) => !real.has(r))
  .sort();

console.log(
  `router: ${real.size} routes   documented: ${documented.size}   ` +
    `excluded by design: ${UNDOCUMENTED_BY_DESIGN.size}`,
);

let failed = false;

if (wrong.length) {
  failed = true;
  console.error("\nDOCUMENTED BUT NOT SERVED — readers will hit a 404:");
  for (const k of wrong) {
    const path = k.split(" ")[1];
    const alt = [...real].find((r) => r.endsWith(` ${path}`));
    const hint = alt ? `router serves ${alt}` : "path absent from router";
    console.error(`  ${k}  ->  ${hint}`);
    console.error(`      in: ${[...documented.get(k)].sort().join(", ")}`);
  }
}

if (unclassified.length) {
  failed = true;
  console.error("\nNEITHER DOCUMENTED NOR EXCLUDED — decide, then act:");
  for (const r of unclassified) console.error(`  ${r}`);
  console.error(
    "\n  Document it, or add it to UNDOCUMENTED_BY_DESIGN here and to the\n" +
      "  table in CONTENT-SCOPE.md with the reason.",
  );
}

if (staleAllowlist.length) {
  // Not a failure: a removed route is fine, the stale entry is just litter.
  console.warn("\nstale exclusions (route no longer exists, drop these):");
  for (const r of staleAllowlist) console.warn(`  ${r}`);
}

if (failed) {
  console.error("\naudit failed");
  process.exit(1);
}
console.log("\naudit passed — every route is documented or deliberately excluded");
