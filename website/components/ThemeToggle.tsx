"use client";

import { useTheme } from "@/lib/theme-provider/ThemeProvider";

/**
 * Light/dark switch.
 *
 * Until `mounted` is true we have no idea which theme is live — the class on
 * <html> was set by an inline script the server never saw. Rendering a guessed
 * icon would flash the wrong one, so the button renders inert-but-sized first
 * and fills in on mount. Reserving the box keeps the navbar from reflowing.
 */
export default function ThemeToggle({
  className = "",
}: {
  className?: string;
}) {
  const { resolvedTheme, toggleTheme, mounted } = useTheme();

  const isDark = resolvedTheme === "dark";
  const label = mounted
    ? `Switch to ${isDark ? "light" : "dark"} theme`
    : "Switch theme";

  return (
    <button
      type="button"
      onClick={toggleTheme}
      aria-label={label}
      title={label}
      className={`inline-flex size-9 items-center justify-center rounded-lg text-vn-text-2 transition-colors hover:bg-vn-surface-2 hover:text-vn-text focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-vn-accent ${className}`}
    >
      {/* Both icons are always in the DOM; only opacity/rotation changes, so
          the swap animates instead of popping. */}
      <span className="relative block size-[18px]">
        <SunIcon
          className={`absolute inset-0 transition-all duration-300 ${
            mounted && !isDark
              ? "rotate-0 scale-100 opacity-100"
              : "-rotate-90 scale-50 opacity-0"
          }`}
        />
        <MoonIcon
          className={`absolute inset-0 transition-all duration-300 ${
            mounted && isDark
              ? "rotate-0 scale-100 opacity-100"
              : "rotate-90 scale-50 opacity-0"
          }`}
        />
      </span>
    </button>
  );
}

function SunIcon({ className = "" }: { className?: string }) {
  return (
    <svg
      className={className}
      width="18"
      height="18"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="1.8"
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden="true"
    >
      <circle cx="12" cy="12" r="4" />
      <path d="M12 2v2M12 20v2M4.93 4.93l1.41 1.41M17.66 17.66l1.41 1.41M2 12h2M20 12h2M6.34 17.66l-1.41 1.41M19.07 4.93l-1.41 1.41" />
    </svg>
  );
}

function MoonIcon({ className = "" }: { className?: string }) {
  return (
    <svg
      className={className}
      width="18"
      height="18"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="1.8"
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden="true"
    >
      <path d="M12 3a6.4 6.4 0 0 0 9 9 9 9 0 1 1-9-9Z" />
    </svg>
  );
}
