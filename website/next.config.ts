import type { NextConfig } from "next";
import { initOpenNextCloudflareForDev } from "@opennextjs/cloudflare";

/*
 * Content Security Policy.
 *
 * `script-src` keeps 'unsafe-inline' because every page is statically
 * prerendered: a per-request nonce with 'strict-dynamic' cannot reach HTML
 * generated at build time, and 'strict-dynamic' would then block every
 * script. The hand-written theme-init script in app/layout.tsx also has to
 * run before first paint, so a nonce migration would need to thread the
 * nonce into it by hand.
 *
 * What this policy does still buy, none of which depends on script-src:
 *   - `object-src 'none'`   kills <object>/<embed> plugin injection outright
 *   - `base-uri 'none'`     stops an injected <base> silently repointing every
 *                           relative script URL on the page
 *   - `frame-ancestors`     the clickjacking control the audit asked for
 *   - `form-action 'self'`  an injected form cannot post credentials offsite
 *   - the allowlists below  an injected script still cannot exfiltrate to an
 *                           attacker-controlled origin, because connect-src
 *                           names the only places this site talks to
 *
 * Nothing on this site embeds a third-party widget, and it calls no API, so
 * connect-src only allows analytics.
 */
/*
 * Every public path the site used to serve before it became a two-page
 * project page. Each one now redirects permanently to the home page so
 * inbound links and search results keep landing somewhere useful.
 */
const RETIRED_PATHS = [
  "/support",
  "/tools",
  "/tools/:path*",
  "/guides",
  "/guides/:path*",
  "/blog",
  "/blog/:path*",
  "/learn",
  "/learn/:path*",
  "/mcp",
  "/providers",
  "/overview",
  "/about",
  "/contact",
  "/terms",
  "/privacy",
  "/mail",
  "/self-host",
  "/login",
  "/signup",
  "/verify",
  "/forgot-password",
  "/reset-password",
  "/auth/:path*",
  "/pricing",
  "/billing",
  "/llms.txt",
  "/llms-full.txt",
];

const CSP_DIRECTIVES: Record<string, string[]> = {
  "default-src": ["'self'"],
  "script-src": [
    "'self'",
    "'unsafe-inline'",
    "https://www.googletagmanager.com",
  ],
  // Tailwind and next/font are self-hosted out of /_next, but React still sets
  // style attributes inline during hydration.
  "style-src": ["'self'", "'unsafe-inline'"],
  "img-src": [
    "'self'",
    "data:",
    "blob:",
    "https://www.googletagmanager.com",
    "https://www.google-analytics.com",
  ],
  // next/font downloads Google Fonts at build time and serves them from our
  // own origin, so no fonts.gstatic.com entry is needed here.
  "font-src": ["'self'", "data:"],
  "connect-src": [
    "'self'",
    "https://www.google-analytics.com",
    "https://*.google-analytics.com",
    "https://*.analytics.google.com",
    "https://*.googletagmanager.com",
  ],
  // This site frames nothing.
  "frame-src": ["'none'"],
  "worker-src": ["'self'", "blob:"],
  "manifest-src": ["'self'"],
  "object-src": ["'none'"],
  "base-uri": ["'none'"],
  "form-action": ["'self'"],
  "frame-ancestors": ["'none'"],
};

const serialize = (directives: Record<string, string[]>) =>
  Object.entries(directives)
    .map(([key, values]) => `${key} ${values.join(" ")}`)
    .join("; ");

const CSP = `${serialize(CSP_DIRECTIVES)}; upgrade-insecure-requests`;

/*
 * Report-Only twin, carrying the one directive that cannot be enforced blind.
 *
 * `require-trusted-types-for 'script'` makes the browser reject any assignment
 * to a DOM XSS sink that is not a TrustedHTML object. React 19 writes
 * `dangerouslySetInnerHTML` through `node.innerHTML` on client-side
 * navigation, and this site uses it for JSON-LD, so enforcing it risks
 * throwing on ordinary in-app navigation.
 *
 * Report-Only answers the question instead of assuming it: violations surface
 * in the DevTools console and nothing is blocked. Click between the two pages
 * and, if the console stays quiet, move this directive into
 * CSP_DIRECTIVES. Lighthouse will keep flagging trusted types until you do —
 * that is the honest state of it, not a fix.
 *
 * There is no report-uri: no collection endpoint exists yet, so reports go to
 * the console only. Add one before relying on this in production.
 */
const CSP_REPORT_ONLY = `${serialize({
  ...CSP_DIRECTIVES,
  "require-trusted-types-for": ["'script'"],
})}; upgrade-insecure-requests`;

const nextConfig: NextConfig = {
  devIndicators: false,
  compress: true,
  // Pinned, not left to the default: canonicals, the sitemap and JSON-LD all
  // use the no-trailing-slash form.
  trailingSlash: false,
  poweredByHeader: false,
  reactStrictMode: true,
  images: {
    formats: ["image/avif", "image/webp"],
  },
  /*
   * Security headers.
   *
   * HSTS: `includeSubDomains` assumes every subdomain of the origin you
   * deploy this site to speaks HTTPS. If that is not true for your domain,
   * drop `includeSubDomains` (and `preload`) before deploying.
   *
   * `preload` is inert until the domain is submitted at hstspreload.org;
   * browsers ignore it otherwise. Do not submit unless you are certain every
   * future subdomain will speak HTTPS, because removal from the list takes
   * months and bricks any plain-HTTP host in the meantime.
   */
  async headers() {
    return [
      {
        source: "/:path*",
        headers: [
          {
            key: "Strict-Transport-Security",
            value: "max-age=63072000; includeSubDomains; preload",
          },

          // See CSP and CSP_REPORT_ONLY above the config for what these carry
          // and why the script-src is shaped the way it is.
          { key: "Content-Security-Policy", value: CSP },
          {
            key: "Content-Security-Policy-Report-Only",
            value: CSP_REPORT_ONLY,
          },
          /*
           * Clickjacking. Both headers are sent on purpose, and they are not
           * redundant: `frame-ancestors` is the modern directive and the one
           * Lighthouse looks for, but it is ignored inside a <meta> CSP and is
           * unsupported by older embedders, so XFO stays as the floor. Where
           * both are understood, `frame-ancestors` wins.
           *
           * `none`, not `sameorigin` — nothing on this site is embedded in a
           * frame by anything, including itself.
           */
          { key: "X-Frame-Options", value: "DENY" },

          /*
           * Origin isolation. Severs `window.opener` between this page and any
           * document it opens or is opened by, which removes the cross-window
           * reference a malicious opener would otherwise hold.
           *
           * `same-origin` is safe here because this site opens no popups and
           * exchanges no `postMessage` traffic.
           */
          { key: "Cross-Origin-Opener-Policy", value: "same-origin" },

          /*
           * Stops the browser second-guessing a declared Content-Type. Without
           * it, a response we serve as text or JSON can be sniffed into script
           * and executed.
           */
          { key: "X-Content-Type-Options", value: "nosniff" },

          /*
           * Full URL to our own origin, bare origin to third parties.
           */
          {
            key: "Referrer-Policy",
            value: "strict-origin-when-cross-origin",
          },

          /*
           * Nothing here uses these, so they are switched off rather than left
           * at the default of "allowed until asked". This is the header that
           * makes an injected script unable to silently request a camera or
           * geolocation prompt.
           */
          {
            key: "Permissions-Policy",
            value:
              "camera=(), microphone=(), geolocation=(), payment=(), usb=(), interest-cohort=()",
          },
        ],
      },
      {
        // Files in public/ are served with `Cache-Control: public, max-age=0`
        // by default, so the logo was refetched on every navigation. These two
        // are content-addressed by hand — the artwork changes under a new
        // filename, never in place — which is what makes `immutable` safe.
        source: "/:file(logo-mark\\.png|logo-512\\.png)",
        headers: [
          {
            key: "Cache-Control",
            value: "public, max-age=31536000, immutable",
          },
        ],
      },
    ];
  },
  /*
   * Paths the old site served all redirect permanently to the single page.
   * No host-specific (www → apex) redirect is configured: set one up at your
   * own host or proxy if you need it.
   */
  async redirects() {
    return [
      ...RETIRED_PATHS.map((source) => ({
        source,
        destination: "/",
        permanent: true,
      })),
    ];
  },
};

export default nextConfig;

/*
 * Makes the Cloudflare bindings declared in wrangler.jsonc (ASSETS, IMAGES,
 * the R2 cache bucket) available to `next dev`, so local development sees the
 * same env the deployed Worker does. No-op in a production build.
 */
initOpenNextCloudflareForDev();
