import { Outlet, NavLink, useLocation, useNavigate } from "react-router-dom"
import { Home, Globe, Settings, Menu, X, BookOpen, Code2, LogOut, Mail } from "lucide-react"
import { useState } from "react"
import { Button } from "@/components/ui/button"
import { cn } from "@/lib/utils"
import { useUser } from "@/hooks/useUser"
import { authApi } from "@/lib/auth/api"

export function AppLayout() {
  const [isSidebarOpen, setIsSidebarOpen] = useState(false)
  const { user, isLoading: isUserLoading } = useUser()
  const navigate = useNavigate()
  const location = useLocation()

  const initial = user?.email?.charAt(0).toUpperCase() ?? "?"
  const primaryLabel = user?.email ?? (isUserLoading ? "Loading…" : "Guest")
  const secondaryLabel = user
    ? user.user_type.charAt(0).toUpperCase() + user.user_type.slice(1) + " account"
    : isUserLoading
      ? "—"
      : "Not signed in"

  const navItems = [
    { name: "Overview", href: "/", icon: Home },
    { name: "Domains", href: "/domains", icon: Globe },
    {
      name: "Mail",
      href: "/mail",
      icon: Mail,
      // Mail is the one section with pages of its own. Without these the only
      // way between them is a link buried in page content, or the back button.
      children: [
        { name: "Overview", href: "/mail", end: true },
        { name: "Campaigns", href: "/mail/campaigns" },
        { name: "Templates", href: "/mail/templates" },
        { name: "Mail settings", href: "/mail/settings" },
      ],
    },
    { name: "Developer", href: "/developer", icon: Code2 },
    { name: "Settings", href: "/settings", icon: Settings },
  ]

  const handleLogout = async () => {
    await authApi.logout()
    navigate("/login", { replace: true })
  }

  return (
    <div className="dark relative isolate min-h-screen bg-vn-bg text-vn-text">
      <div className="pointer-events-none fixed inset-0 z-0 overflow-hidden">
        <div className="absolute inset-0 vn-dashboard-grid" />
      </div>

      {isSidebarOpen && (
        <div
          className="fixed inset-0 z-40 bg-vn-text/30 backdrop-blur-sm lg:hidden"
          onClick={() => setIsSidebarOpen(false)}
          aria-hidden
        />
      )}

      <aside
        className={cn(
          "fixed inset-y-0 left-0 z-50 flex w-64 flex-col border-r border-vn-hairline bg-vn-surface transition-transform duration-300",
          isSidebarOpen ? "translate-x-0" : "-translate-x-full",
          "lg:translate-x-0",
        )}
      >
        <div className="flex h-16 shrink-0 items-center justify-between border-b border-vn-hairline px-5">
          <NavLink
            to="/"
            className="flex items-center gap-2 select-none"
            onClick={() => setIsSidebarOpen(false)}
          >
            <img
              src="/logo-mark.png"
              alt="vnytros logo"
              className="h-6 w-auto object-contain"
            />
            <span className="text-base font-semibold text-vn-text">
              Vny<span className="text-vn-accent italic">tros</span>
            </span>
          </NavLink>
          <button
            type="button"
            onClick={() => setIsSidebarOpen(false)}
            className="flex h-8 w-8 items-center justify-center rounded-md text-vn-text-3 transition-colors hover:bg-vn-surface-2 hover:text-vn-text lg:hidden"
            aria-label="Close menu"
          >
            <X className="size-4" />
          </button>
        </div>

        <nav className="flex-1 overflow-y-auto px-3 py-5">
          <p className="mb-2 px-3 font-mono text-[10px] uppercase tracking-widest text-vn-text-4">
            Workspace
          </p>
          <ul className="space-y-0.5">
          {navItems.map((item) => {
            const Icon = item.icon
            // Keep a section's pages listed while the reader is inside it.
            const inSection =
              location.pathname === item.href ||
              location.pathname.startsWith(`${item.href}/`)
            return (
              <li key={item.name}>
              <NavLink
                to={item.href}
                onClick={() => setIsSidebarOpen(false)}
                end={item.href === "/"}
                className={({ isActive }) =>
                  cn(
                    "flex items-center gap-3 rounded-lg border px-3 py-2 text-sm font-medium transition-colors",
                    isActive
                      ? "border-vn-accent/20 bg-vn-accent-soft text-vn-text"
                      : "border-transparent text-vn-text-2 hover:bg-vn-surface-2 hover:text-vn-text",
                  )
                }
              >
                {({ isActive }) => (
                  <>
                    <Icon
                      className={cn(
                        "size-[18px]",
                        isActive ? "text-vn-accent" : "text-vn-text-3",
                      )}
                    />
                    <span className="flex-1">{item.name}</span>
                  </>
                )}
              </NavLink>

              {item.children && inSection && (
                <ul className="mt-0.5 mb-1 ml-[26px] space-y-0.5 border-l border-vn-hairline-2 pl-3">
                  {item.children.map((child) => (
                    <li key={child.href}>
                      <NavLink
                        to={child.href}
                        end={child.end}
                        onClick={() => setIsSidebarOpen(false)}
                        className={({ isActive }) =>
                          cn(
                            "block rounded-md px-2.5 py-1.5 text-[13px] transition-colors",
                            isActive
                              ? "bg-vn-surface-2 font-medium text-vn-text"
                              : "text-vn-text-3 hover:bg-vn-surface-2 hover:text-vn-text",
                          )
                        }
                      >
                        {child.name}
                      </NavLink>
                    </li>
                  ))}
                </ul>
              )}
              </li>
            )
          })}
          </ul>
        </nav>

        <div className="space-y-2 border-t border-vn-hairline p-3">
          <NavLink
            to="/profile"
            onClick={() => setIsSidebarOpen(false)}
            className={({ isActive }) =>
              cn(
                "flex items-center gap-3 rounded-lg border px-2 py-2 transition-colors",
                isActive
                  ? "border-vn-accent/20 bg-vn-accent-soft"
                  : "border-transparent hover:bg-vn-surface-2",
              )
            }
            aria-label="Open profile"
          >
            <div className="flex size-8 shrink-0 items-center justify-center rounded-full border border-vn-accent/30 bg-vn-accent-soft text-xs font-semibold text-vn-accent">
              {initial}
            </div>
            <div className="min-w-0 flex-1">
              <p className="truncate text-sm font-medium text-vn-text">{primaryLabel}</p>
              <p className="truncate text-xs text-vn-text-3">{secondaryLabel}</p>
            </div>
          </NavLink>
          <button
            type="button"
            onClick={handleLogout}
            className="flex w-full items-center gap-2 rounded-lg border border-transparent px-2 py-2 text-sm text-vn-text-2 transition-colors hover:bg-vn-surface-2 hover:text-vn-text"
          >
            <LogOut className="size-4" />
            Sign out
          </button>
          <div className="flex items-center gap-3 px-2 pt-1 text-xs text-vn-text-3">
            <a
              href="https://github.com/Aadesh998/vnytros-domain-connect"
              target="_blank"
              rel="noreferrer"
              className="flex items-center gap-1.5 transition-colors hover:text-vn-text"
            >
              <GithubMark />
              GitHub
            </a>
          </div>
        </div>
      </aside>

      <div className="relative z-10 lg:pl-64">
        <header className="sticky top-0 z-30 border-b border-vn-hairline bg-vn-bg/80 backdrop-blur-md lg:hidden">
          <div className="flex items-center justify-between px-5 py-4">
            <NavLink
              to="/"
              className="flex items-center gap-2 select-none"
            >
              <img
                src="/logo-mark.png"
                alt="vnytros logo"
                className="h-6 w-auto object-contain"
              />
              <span className="text-base font-semibold text-vn-text">
                Vny<span className="italic text-vn-accent">tros</span>
              </span>
            </NavLink>
            <Button
              variant="outline"
              size="icon"
              onClick={() => setIsSidebarOpen(true)}
              aria-label="Open menu"
            >
              <Menu className="size-5" />
            </Button>
          </div>
        </header>

        <main className="mx-auto max-w-5xl px-5 py-5 md:px-6 md:py-7">
          <div className="mb-4 hidden justify-end lg:flex">
            <Button variant="outline" size="sm" asChild>
              <a
                href="https://github.com/Aadesh998/vnytros-domain-connect/tree/main/docs/content/docs"
                target="_blank"
                rel="noreferrer"
              >
                <BookOpen className="size-4" />
                Documentation
              </a>
            </Button>
          </div>
          <div>
            <Outlet />
          </div>
        </main>
      </div>
    </div>
  )
}

function GithubMark() {
  return (
    <svg className="size-3.5" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
      <path d="M12 2C6.5 2 2 6.5 2 12c0 4.4 2.9 8.2 6.8 9.5.5.1.7-.2.7-.5v-1.7c-2.8.6-3.4-1.3-3.4-1.3-.5-1.2-1.1-1.5-1.1-1.5-.9-.6.1-.6.1-.6 1 .1 1.5 1 1.5 1 .9 1.5 2.4 1.1 3 .8.1-.7.4-1.1.6-1.4-2.2-.3-4.6-1.1-4.6-5 0-1.1.4-2 1-2.7-.1-.3-.4-1.3.1-2.7 0 0 .8-.3 2.8 1a9.6 9.6 0 0 1 5 0c2-1.3 2.8-1 2.8-1 .5 1.4.2 2.4.1 2.7.6.7 1 1.6 1 2.7 0 3.9-2.4 4.7-4.6 5 .4.3.7.9.7 1.8v2.7c0 .3.2.6.7.5C19.1 20.2 22 16.4 22 12c0-5.5-4.5-10-10-10Z" />
    </svg>
  )
}
