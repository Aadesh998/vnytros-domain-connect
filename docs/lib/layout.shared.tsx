import type { BaseLayoutProps } from "fumadocs-ui/layouts/shared";
import { REPO_URL } from "@/lib/config";

/**
 * Shared layout options (nav title, links) used by the docs layout.
 */
export function baseOptions(): BaseLayoutProps {
  return {
    nav: {
      title: (
        <span style={{ fontWeight: 600, letterSpacing: "-0.01em" }}>
          Vnytros&nbsp;<span style={{ opacity: 0.6 }}>Docs</span>
        </span>
      ),
      url: "/docs",
    },
    links: [
      {
        text: "API Reference",
        url: "/docs/api/detect-provider",
        active: "nested-url",
      },
    ],
    // Vnytros is self-hosted: the GitHub repository is the project's home
    // (there is no hosted product to link to).
    githubUrl: REPO_URL,
  };
}
