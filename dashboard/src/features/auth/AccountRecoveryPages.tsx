import { useEffect, useRef, useState, type ReactNode } from "react"
import { Link, useNavigate, useSearchParams } from "react-router-dom"
import { authApi } from "@/lib/auth/api"
import { ApiClientError } from "@/lib/api/http"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"

/**
 * Email verification and password reset. The API builds these links from its
 * FRONTEND_URL, so a self-hosted deployment points FRONTEND_URL at this
 * dashboard and needs no separate website for account recovery.
 */

function AuthShell({
  title,
  subtitle,
  children,
}: {
  title: string
  subtitle?: string
  children: ReactNode
}) {
  return (
    <div className="dark min-h-screen bg-vn-bg text-vn-text flex items-center justify-center px-6">
      <div className="w-full max-w-sm">
        <div className="mb-8 text-center">
          <img src="/logo-mark.png" alt="vnytros" className="mx-auto h-8 w-auto" />
          <h1 className="mt-6 text-2xl font-[540] tracking-[-0.02em]">{title}</h1>
          {subtitle && <p className="mt-1 text-sm text-vn-text-3">{subtitle}</p>}
        </div>
        {children}
        <p className="mt-6 text-center text-sm text-vn-text-3">
          <Link to="/login" className="text-vn-accent hover:text-vn-accent-2">
            Back to sign in
          </Link>
        </p>
      </div>
    </div>
  )
}

function messageOf(err: unknown): string {
  return err instanceof ApiClientError
    ? err.message || "Something went wrong. Please try again."
    : "Network error. Please try again."
}

export function VerifyEmailPage() {
  const [params] = useSearchParams()
  const navigate = useNavigate()
  const token = params.get("token")
  const [state, setState] = useState<"pending" | "done" | "error">(
    token ? "pending" : "error",
  )
  const [error, setError] = useState<string | null>(
    token ? null : "The verification token is missing from the link.",
  )
  // StrictMode runs effects twice in dev; a token is single-use.
  const started = useRef(false)

  useEffect(() => {
    if (!token || started.current) return
    started.current = true
    authApi
      .verify(token)
      .then(() => {
        setState("done")
        setTimeout(() => navigate("/", { replace: true }), 1200)
      })
      .catch((err) => {
        setState("error")
        setError(messageOf(err))
      })
  }, [token, navigate])

  return (
    <AuthShell
      title={
        state === "pending"
          ? "Verifying your email…"
          : state === "done"
            ? "Email verified"
            : "Verification failed"
      }
      subtitle={state === "done" ? "Signing you in…" : undefined}
    >
      {error && (
        <p className="text-center text-sm text-vn-danger" role="alert">
          {error}
        </p>
      )}
    </AuthShell>
  )
}

export function ForgotPasswordPage() {
  const [email, setEmail] = useState("")
  const [sent, setSent] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [submitting, setSubmitting] = useState(false)

  const submit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (submitting) return
    setError(null)
    setSubmitting(true)
    try {
      await authApi.forgetPassword(email.trim().toLowerCase())
      setSent(true)
    } catch (err) {
      setError(messageOf(err))
    } finally {
      setSubmitting(false)
    }
  }

  if (sent) {
    return (
      <AuthShell
        title="Check your email"
        subtitle={`If an account exists for ${email}, a reset link is on its way.`}
      >
        {null}
      </AuthShell>
    )
  }

  return (
    <AuthShell title="Reset your password" subtitle="We'll email you a reset link.">
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
        {error && (
          <p className="text-sm text-vn-danger" role="alert">
            {error}
          </p>
        )}
        <Button type="submit" className="w-full" disabled={submitting}>
          {submitting ? "Sending…" : "Send reset link"}
        </Button>
      </form>
    </AuthShell>
  )
}

export function ResetPasswordPage() {
  const [params] = useSearchParams()
  const token = params.get("token")
  const [password, setPassword] = useState("")
  const [confirm, setConfirm] = useState("")
  const [done, setDone] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [submitting, setSubmitting] = useState(false)

  if (!token) {
    return (
      <AuthShell title="Invalid reset link">
        <p className="text-center text-sm text-vn-text-3">
          The reset token is missing.{" "}
          <Link to="/forgot-password" className="text-vn-accent hover:text-vn-accent-2">
            Request a new link
          </Link>
          .
        </p>
      </AuthShell>
    )
  }

  if (done) {
    return (
      <AuthShell title="Password updated" subtitle="You can now sign in with your new password.">
        {null}
      </AuthShell>
    )
  }

  const submit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (submitting) return
    if (password !== confirm) return setError("Passwords do not match.")
    if (password.length < 8) return setError("Password must be at least 8 characters.")
    setError(null)
    setSubmitting(true)
    try {
      await authApi.resetPassword({
        token,
        new_password: password,
        confirm_password: confirm,
      })
      setDone(true)
    } catch (err) {
      setError(messageOf(err))
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <AuthShell title="Choose a new password">
      <form onSubmit={submit} className="space-y-4">
        <div>
          <label className="block text-xs font-medium text-vn-text-2 mb-1.5">
            New password
          </label>
          <Input
            type="password"
            autoComplete="new-password"
            required
            value={password}
            onChange={(e) => setPassword(e.target.value)}
          />
        </div>
        <div>
          <label className="block text-xs font-medium text-vn-text-2 mb-1.5">
            Confirm password
          </label>
          <Input
            type="password"
            autoComplete="new-password"
            required
            value={confirm}
            onChange={(e) => setConfirm(e.target.value)}
          />
        </div>
        {error && (
          <p className="text-sm text-vn-danger" role="alert">
            {error}
          </p>
        )}
        <Button type="submit" className="w-full" disabled={submitting}>
          {submitting ? "Updating…" : "Update password"}
        </Button>
      </form>
    </AuthShell>
  )
}
