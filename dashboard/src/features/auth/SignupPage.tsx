import { useState } from "react"
import { Link } from "react-router-dom"
import { authApi } from "@/lib/auth/api"
import { ApiClientError } from "@/lib/api/http"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { AuthDivider, OAuthButtons } from "@/features/auth/OAuthButtons"

export function SignupPage() {
  const [name, setName] = useState("")
  const [email, setEmail] = useState("")
  const [password, setPassword] = useState("")
  const [error, setError] = useState<string | null>(null)
  const [submitting, setSubmitting] = useState(false)
  const [sentTo, setSentTo] = useState<string | null>(null)

  const submit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (submitting) return
    setError(null)
    setSubmitting(true)
    try {
      const res = await authApi.signup(
        name.trim(),
        email.trim().toLowerCase(),
        password,
      )
      setSentTo(res.email)
    } catch (err) {
      if (err instanceof ApiClientError) setError(err.message)
      else setError("Network error. Please try again.")
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
          <h1 className="mt-6 text-2xl font-[540] tracking-[-0.02em]">
            Create your account
          </h1>
          <p className="mt-1 text-sm text-vn-text-3">
            Start managing domains in minutes.
          </p>
        </div>

        {sentTo ? (
          <div className="rounded-xl border border-vn-hairline-2 bg-vn-surface p-6 text-center">
            <p className="text-sm text-vn-text">
              Check <span className="font-medium">{sentTo}</span> for a
              verification link to activate your account.
            </p>
            <Link
              to="/login"
              className="mt-4 inline-block text-sm text-vn-accent hover:text-vn-accent-2"
            >
              Back to sign in
            </Link>
          </div>
        ) : (
          <>
            <OAuthButtons disabled={submitting} />
            <AuthDivider label="or sign up with email" />

            <form onSubmit={submit} className="space-y-4">
            <div>
              <label className="block text-xs font-medium text-vn-text-2 mb-1.5">
                Name
              </label>
              <Input
                autoComplete="name"
                required
                value={name}
                onChange={(e) => setName(e.target.value)}
              />
            </div>
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
              />
            </div>
            <div>
              <label className="block text-xs font-medium text-vn-text-2 mb-1.5">
                Password
              </label>
              <Input
                type="password"
                autoComplete="new-password"
                required
                minLength={8}
                value={password}
                onChange={(e) => setPassword(e.target.value)}
              />
            </div>

            {error && (
              <p className="text-sm text-vn-danger" role="alert">
                {error}
              </p>
            )}

            <Button type="submit" className="w-full" disabled={submitting}>
              {submitting ? "Creating…" : "Create account"}
            </Button>

            <p className="text-center text-sm text-vn-text-3">
              Already have an account?{" "}
              <Link to="/login" className="text-vn-accent hover:text-vn-accent-2">
                Sign in
              </Link>
            </p>
            </form>
          </>
        )}
      </div>
    </div>
  )
}
