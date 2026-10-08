import { Navigate, Outlet, useLocation } from "react-router-dom"
import { useUser } from "@/hooks/useUser"

export function RequireAuth() {
  const { user, isLoading, unauthenticated } = useUser()
  const location = useLocation()

  if (isLoading) {
    return (
      <div className="dark min-h-screen bg-vn-bg text-vn-text flex items-center justify-center">
        <p className="text-sm text-vn-text-3">Loading…</p>
      </div>
    )
  }

  if (unauthenticated || !user) {
    return (
      <Navigate
        to="/login"
        replace
        state={{ from: location.pathname + location.search }}
      />
    )
  }

  return <Outlet />
}
