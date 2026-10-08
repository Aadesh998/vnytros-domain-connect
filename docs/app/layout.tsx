import "./global.css";
import { RootProvider } from "fumadocs-ui/provider/next";
import { Geist, Geist_Mono } from "next/font/google";
import type { ReactNode } from "react";
import type { Metadata } from "next";
import { DOCS_URL } from "@/lib/config";
import { docsSiteJsonLd } from "@/lib/seo/jsonld";

// Same faces as the marketing site. The CSS variables are consumed by the
// --font-sans / --font-mono overrides in global.css.
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

export const metadata: Metadata = {
  // Required for the per-page canonicals in app/docs/[[...slug]]/page.tsx to
  // resolve to absolute URLs.
  metadataBase: new URL(DOCS_URL),
  title: {
    default: "Vnytros Docs",
    template: "%s — Vnytros Docs",
  },
  description:
    "Documentation for the Vnytros Domain Connect API — detect DNS providers, connect custom domains, and manage them programmatically.",
  applicationName: "Vnytros Docs",
  openGraph: {
    siteName: "Vnytros Docs",
    type: "website",
    locale: "en_US",
  },
  robots: {
    index: true,
    follow: true,
    googleBot: {
      index: true,
      follow: true,
      "max-image-preview": "large",
      "max-snippet": -1,
    },
  },
};

export default function Layout({ children }: { children: ReactNode }) {
  return (
    <html
      lang="en"
      suppressHydrationWarning
      className={`${geist.variable} ${geistMono.variable}`}
    >
      <body className="flex flex-col min-h-screen">
        <script
          type="application/ld+json"
          dangerouslySetInnerHTML={{ __html: JSON.stringify(docsSiteJsonLd) }}
        />
        <RootProvider>{children}</RootProvider>
      </body>
    </html>
  );
}
