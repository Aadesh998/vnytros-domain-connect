import { useState } from "react"
import { useNavigate, Link, useLocation } from "react-router-dom"
import { authApi } from "@/lib/auth/api"
import { ApiClientError } from "@/lib/api/http"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { AuthDivider, OAuthButtons } from "@/features/auth/OAuthButtons"

export function LoginPage() {
  const navigate = useNavigate()
  const location = useLocation()
  const from = (location.state as { from?: string } | null)?.from ?? "/"
  const [email, setEmail] = useState("")
  const [password, setPassword] = useState("")
  const [error, setError] = useState<string | null>(null)
  const [submitting, setSubmitting] = useState(false)

  const submit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (submitting) return
    setError(null)
    setSubmitting(true)
    try {
      await authApi.login(email.trim().toLowerCase(), password)
      navigate(from, { replace: true })
    } catch (err) {
      if (err instanceof ApiClientError) {
        if (err.code === "INVALID_CREDENTIALS")
          setError("Incorrect email or password.")
        else if (err.code === "EMAIL_UNVERIFIED")
          setError("Please verify your email before signing in.")
        else setError(err.message)
      } else {
        setError("Network error. Please try again.")
      }
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <div className="dark min-h-screen bg-vn-bg text-vn-text flex items-center justify-center px-6">
      <div className="w-full max-w-sm">
        <div className="mb-8 text-center">
          <img
            src="/logo-mark.png"
            alt="vnytros"
            className="mx-auto h-8 w-auto"
          />
          <h1 className="mt-6 text-2xl font-[540] tracking-[-0.02em]">Sign in</h1>
          <p className="mt-1 text-sm text-vn-text-3">
            Welcome back. Enter your credentials.
          </p>
        </div>

        <OAuthButtons next={from} disabled={submitting} />
        <AuthDivider label="or continue with email" />

        <form onSubmit={submit} className="space-y-4">
          <div>
            <label className="block text-xs font-medium text-vn-text-2 mb-1.5">
              Email
            </label>
            <Input
              type="email"
              autoComplete="email"
              required
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              placeholder="you@company.com"
            />
          </div>
          <div>
            <div className="mb-1.5 flex items-center justify-between">
              <label className="block text-xs font-medium text-vn-text-2">
                Password
              </label>
              <Link
                to="/forgot-password"
                className="text-xs text-vn-text-3 hover:text-vn-accent"
              >
                Forgot password?
              </Link>
            </div>
            <Input
              type="password"
              autoComplete="current-password"
              required
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              placeholder="••••••••"
            />
          </div>

          {error && (
            <p className="text-sm text-vn-danger" role="alert">
              {error}
            </p>
          )}

          <Button type="submit" className="w-full" disabled={submitting}>
            {submitting ? "Signing in…" : "Sign in"}
          </Button>
        </form>

        <p className="mt-6 text-center text-sm text-vn-text-3">
          New here?{" "}
          <Link to="/signup" className="text-vn-accent hover:text-vn-accent-2">
            Create an account
          </Link>
        </p>
      </div>
    </div>
  )
}
