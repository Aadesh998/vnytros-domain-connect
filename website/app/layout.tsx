import type { Metadata, Viewport } from "next";
import { Geist, Geist_Mono } from "next/font/google";
import Script from "next/script";
import "./globals.css";
import {
  ThemeProvider,
  themeInitScript,
} from "@/lib/theme-provider/ThemeProvider";
import { cn } from "@/lib/utils";
import { SITE_URL } from "@/lib/config";
import { siteJsonLd } from "@/lib/seo/jsonld";

// Optional. Unset (the default for forks and local builds) loads no analytics.
const GA_MEASUREMENT_ID = process.env.NEXT_PUBLIC_GA_MEASUREMENT_ID;

// The CSS variable names here are consumed by `--font-sans` / `--font-mono`
// in theme.css. They are deliberately NOT named `--font-sans`
// directly: theme.css needs to compose them with a fallback stack, and a
// variable cannot reference itself.
const geist = Geist({
  subsets: ["latin"],
  variable: "--font-geist",
  display: "swap",
});

const geistMono = Geist_Mono({
  subsets: ["latin"],
  variable: "--font-geist-mono",
  display: "swap",
});

export const viewport: Viewport = {
  // Must track the real canvas in each theme, or mobile browser chrome renders
  // a band in the wrong color above the page.
  themeColor: [
    { media: "(prefers-color-scheme: light)", color: "#ffffff" },
    { media: "(prefers-color-scheme: dark)", color: "#08090b" },
  ],
  width: "device-width",
  initialScale: 1,
};

const TITLE = "Vnytros — open-source DNS & email-authentication toolkit";
const DESCRIPTION =
  "Vnytros is an open-source, self-hosted DNS and email-authentication toolkit with a built-in mail platform: provider detection, DNS record creation, domain verification and 15 DNS, email, TLS and HTTP checks via REST API and MCP.";

export const metadata: Metadata = {
  metadataBase: new URL(SITE_URL),
  title: TITLE,
  description: DESCRIPTION,
  applicationName: "Vnytros",
  alternates: {
    canonical: "/",
  },
  authors: [{ name: "Vnytros contributors" }],
  creator: "vnytros",
  publisher: "vnytros",
  openGraph: {
    title: TITLE,
    description: DESCRIPTION,
    url: SITE_URL,
    siteName: "Vnytros",
    locale: "en_US",
    type: "website",
    // `images` is omitted on purpose: app/opengraph-image.tsx generates the
    // card and Next injects it here.
  },
  twitter: {
    card: "summary_large_image",
    title: TITLE,
    description: DESCRIPTION,
  },
  robots: {
    index: true,
    follow: true,
  },
};

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    // `suppressHydrationWarning` is required: themeInitScript mutates this
    // element's class and style before React hydrates, so the client markup
    // legitimately differs from what the server sent.
    <html
      lang="en"
      suppressHydrationWarning
      className={cn(
        "antialiased font-sans",
        geist.variable,
        geistMono.variable,
      )}
    >
      <body>
        {/* Must be the first thing in <body> so the theme class lands before
            the browser paints any content. */}
        <script dangerouslySetInnerHTML={{ __html: themeInitScript }} />
        <script
          type="application/ld+json"
          dangerouslySetInnerHTML={{ __html: JSON.stringify(siteJsonLd) }}
        />
        {/* `lazyOnload`, not `afterInteractive`: Lighthouse attributed ~250 ms
            of blocking time and 170 kB of transfer to GTM while the hero was
            still painting. Deferring to the load event keeps page-view and
            event collection intact (gtag queues calls into dataLayer before the
            script arrives) but takes it off the critical path entirely.
            No `preconnect` for this origin on purpose — pre-warming a
            connection we have deliberately deferred would put the cost back. */}
        {GA_MEASUREMENT_ID && (
          <>
            <Script
              src={`https://www.googletagmanager.com/gtag/js?id=${GA_MEASUREMENT_ID}`}
              strategy="lazyOnload"
            />
            <Script id="google-analytics" strategy="lazyOnload">
              {`
            window.dataLayer = window.dataLayer || [];
            function gtag(){dataLayer.push(arguments);}
            gtag('js', new Date());
            gtag('config', '${GA_MEASUREMENT_ID}');
          `}
            </Script>
          </>
        )}
        {/* First focusable element in the document, so a keyboard user's
            first Tab offers a jump past the nav into the page content.
            Every route's <main> carries id="main-content" as the target. */}
        <a href="#main-content" className="vn-skip-link">
          Skip to content
        </a>
        <ThemeProvider>{children}</ThemeProvider>
      </body>
    </html>
  );
}
