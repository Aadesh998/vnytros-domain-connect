"use client";

import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useSyncExternalStore,
} from "react";

export type ThemePreference = "light" | "dark" | "system";
export type ResolvedTheme = "light" | "dark";

/** localStorage key. Kept in sync with `themeInitScript` below. */
const THEME_STORAGE_KEY = "vn-theme";

/** Fired on the window so same-tab writes reach every subscriber. */
const THEME_EVENT = "vn-theme-change";

/**
 * Runs before React hydrates, as the first child of <body>, so the correct
 * class is on <html> by the time the browser paints. Without it the page
 * renders light for one frame and then snaps to dark — the classic FOUC.
 *
 * It is deliberately duplicated logic (not imported from this module) because
 * it must be a self-contained synchronous string with no bundler runtime.
 * If you change the storage key or the class name, change it in BOTH places.
 */
export const themeInitScript = `
(function () {
  try {
    var stored = localStorage.getItem(${JSON.stringify(THEME_STORAGE_KEY)});
    var isDark =
      stored === "dark" ||
      ((stored === null || stored === "system") &&
        window.matchMedia("(prefers-color-scheme: dark)").matches);
    document.documentElement.classList.toggle("dark", isDark);
    document.documentElement.style.colorScheme = isDark ? "dark" : "light";
    // Marks that scripting is live. Scroll-reveal styles key off this so a
    // no-JS client (or any renderer that never fires IntersectionObserver)
    // gets fully visible content instead of a page of opacity:0 sections.
    document.documentElement.classList.add("js");
  } catch (e) {
    /* private mode / storage disabled — fall through to the light default */
  }
})();
`;

/* ─────────────────────────────────────────────────────────────
   External store

   The source of truth is localStorage + the OS media query, both of
   which live outside React. `useSyncExternalStore` is the primitive
   built for exactly that: it avoids the setState-in-effect cascade,
   and its separate server snapshot doubles as the "have we hydrated
   yet" signal we need before we can trust either source.
───────────────────────────────────────────────────────────── */

function readPreference(): ThemePreference {
  try {
    const raw = localStorage.getItem(THEME_STORAGE_KEY);
    if (raw === "light" || raw === "dark" || raw === "system") return raw;
  } catch {
    /* storage unavailable */
  }
  return "system";
}

function systemTheme(): ResolvedTheme {
  return window.matchMedia("(prefers-color-scheme: dark)").matches
    ? "dark"
    : "light";
}

function subscribe(onChange: () => void) {
  const mql = window.matchMedia("(prefers-color-scheme: dark)");
  // `storage` covers other tabs; THEME_EVENT covers this one.
  mql.addEventListener("change", onChange);
  window.addEventListener("storage", onChange);
  window.addEventListener(THEME_EVENT, onChange);
  return () => {
    mql.removeEventListener("change", onChange);
    window.removeEventListener("storage", onChange);
    window.removeEventListener(THEME_EVENT, onChange);
  };
}

// Both snapshots return primitives, so React's Object.is check settles them
// without any caching layer.
const getPreferenceSnapshot = (): ThemePreference => readPreference();

const getResolvedSnapshot = (): ResolvedTheme => {
  const pref = readPreference();
  return pref === "system" ? systemTheme() : pref;
};

// `null` on the server AND for the hydration render. React swaps to the client
// snapshot only once hydration finishes, which is precisely when it becomes
// safe to read the class the init script already put on <html>.
const getServerSnapshot = () => null;

type ThemeContextValue = {
  /** What the user chose, including "system". */
  theme: ThemePreference;
  /** What is actually on screen. `null` until hydrated. */
  resolvedTheme: ResolvedTheme | null;
  setTheme: (theme: ThemePreference) => void;
  /** Flips between light and dark, resolving "system" first. */
  toggleTheme: () => void;
  /** False during SSR and the hydration render. */
  mounted: boolean;
};

const ThemeContext = createContext<ThemeContextValue | null>(null);

export function ThemeProvider({ children }: { children: React.ReactNode }) {
  const theme =
    useSyncExternalStore(
      subscribe,
      getPreferenceSnapshot,
      getServerSnapshot,
    ) ?? "system";

  const resolvedTheme = useSyncExternalStore(
    subscribe,
    getResolvedSnapshot,
    getServerSnapshot,
  );

  // Push React's view back onto <html>. This is a pure external-system sync —
  // no setState — so it stays out of the render cascade.
  useEffect(() => {
    if (!resolvedTheme) return;
    document.documentElement.classList.toggle("dark", resolvedTheme === "dark");
    document.documentElement.style.colorScheme = resolvedTheme;
  }, [resolvedTheme]);

  const setTheme = useCallback((next: ThemePreference) => {
    try {
      localStorage.setItem(THEME_STORAGE_KEY, next);
    } catch {
      /* storage unavailable — the choice just won't survive a reload */
    }
    // Storage events do not fire in the tab that performed the write, so the
    // store has to be nudged explicitly here.
    window.dispatchEvent(new Event(THEME_EVENT));
  }, []);

  const toggleTheme = useCallback(() => {
    const current = resolvedTheme ?? systemTheme();
    setTheme(current === "dark" ? "light" : "dark");
  }, [resolvedTheme, setTheme]);

  const value = useMemo<ThemeContextValue>(
    () => ({
      theme,
      resolvedTheme,
      setTheme,
      toggleTheme,
      mounted: resolvedTheme !== null,
    }),
    [theme, resolvedTheme, setTheme, toggleTheme],
  );

  return (
    <ThemeContext.Provider value={value}>{children}</ThemeContext.Provider>
  );
}

export function useTheme(): ThemeContextValue {
  const ctx = useContext(ThemeContext);
  if (!ctx) throw new Error("useTheme must be used inside <ThemeProvider>");
  return ctx;
}
