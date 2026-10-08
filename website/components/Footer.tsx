import Logo from "./Logo";
import { DOCS_ENTRY, REPOS, repoBlob } from "@/lib/config";

export default function Footer() {
  return (
    <footer className="mt-24 border-t border-vn-hairline">
      <div className="mx-auto grid max-w-5xl grid-cols-1 gap-10 px-4 sm:px-6 py-12 sm:grid-cols-[1.3fr_1fr_1fr]">
        <div>
          <Logo size="md" />
          <p className="mt-4 max-w-70 text-[13.5px] leading-[1.55] text-vn-text-3">
            Open-source, self-hosted DNS and email-authentication toolkit.
          </p>
        </div>

        <div>
          <h2
            id="footer-repos"
            className="mb-3.5 font-mono font-medium text-[12px] uppercase tracking-[0.08em] text-vn-text-3"
          >
            Source
          </h2>
          <ul aria-labelledby="footer-repos" className="flex flex-col gap-2.5">
            {REPOS.map((r) => (
              <li key={r.name}>
                <a
                  href={r.href}
                  className="font-mono text-[13px] text-vn-text-2 hover:text-vn-text transition-colors"
                >
                  {r.name}
                </a>
              </li>
            ))}
          </ul>
        </div>

        <div>
          <h2
            id="footer-project"
            className="mb-3.5 font-mono font-medium text-[12px] uppercase tracking-[0.08em] text-vn-text-3"
          >
            Project
          </h2>
          <ul aria-labelledby="footer-project" className="flex flex-col gap-2.5">
            <li>
              <a
                href={DOCS_ENTRY}
                className="text-[13.5px] text-vn-text-2 hover:text-vn-text transition-colors"
              >
                Documentation
              </a>
            </li>
          </ul>
        </div>
      </div>

      <div className="border-t border-vn-hairline px-4 py-6 text-center text-[12px] text-vn-text-3">
        Vnytros is open source under{" "}
        <a
          href={repoBlob("LICENSE")}
          className="hover:text-vn-text transition-colors underline-offset-2 hover:underline"
        >
          AGPL-3.0-only
        </a>
        ; the SDK is{" "}
        <a
          href={repoBlob("sdk/LICENSE")}
          className="hover:text-vn-text transition-colors underline-offset-2 hover:underline"
        >
          MIT
        </a>
        .
      </div>
    </footer>
  );
}
