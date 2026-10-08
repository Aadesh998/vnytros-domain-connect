import { DOCS_ENTRY, REPO_URL } from "@/lib/config";
import Logo from "./Logo";
import ThemeToggle from "./ThemeToggle";

const linkClass =
  "text-[13.5px] font-medium tracking-[-0.005em] text-vn-text-2 hover:text-vn-text transition-colors";

/**
 * Three links and a theme switch — few enough to fit a phone without a
 * menu, so there is no mobile panel to manage.
 */
export default function Navbar() {
  return (
    <header className="border-b border-vn-hairline">
      <nav
        aria-label="Main"
        className="mx-auto flex max-w-5xl items-center justify-between gap-4 px-4 sm:px-6 py-4"
      >
        <Logo priority />

        <div className="flex items-center gap-4 sm:gap-6">
          <a href={REPO_URL} className={linkClass}>
            GitHub
          </a>
          <a href={DOCS_ENTRY} className={linkClass}>
            Docs
          </a>
          <ThemeToggle />
        </div>
      </nav>
    </header>
  );
}
