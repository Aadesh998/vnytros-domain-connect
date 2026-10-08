import { useEffect, useRef, useState } from "react"
import { Link, useNavigate, useParams } from "react-router-dom"
import { authApi, takeOAuthNext, type OAuthProvider } from "@/lib/auth/api"
import { ApiClientError } from "@/lib/api/http"

const PROVIDERS = ["google", "github"] as const

function isProvider(value: unknown): value is OAuthProvider {
  return typeof value === "string" && (PROVIDERS as readonly string[]).includes(value)
}

/**
 * Landing spot for Google/GitHub. They send `code` + `state` here; we hand both
 * to the API, which exchanges them for our token pair.
 */
/**
 * Everything that can be judged from the URL alone, resolved before the first
 * render so the effect only ever performs the exchange.
 */
function readCallback(provider: string | undefined) {
  if (!isProvider(provider)) return { error: "Unknown sign-in provider." }

  const params = new URLSearchParams(window.location.search)
  if (params.get("error")) {
    return {
      error:
        params.get("error_description") ?? "Sign-in was cancelled or failed.",
    }
  }

  const code = params.get("code")
  const state = params.get("state")
  if (!code || !state) {
    return {
      error: "Missing sign-in details from the provider. Please try again.",
    }
  }
  return { provider, code, state }
}

export function OAuthCallbackPage() {
  const navigate = useNavigate()
  const { provider: providerParam } = useParams<{ provider: string }>()
  const [request] = useState(() => readCallback(providerParam))
  const [error, setError] = useState<string | null>(request.error ?? null)
  // StrictMode mounts effects twice; an authorization code is single-use.
  const ranRef = useRef(false)

  useEffect(() => {
    if (ranRef.current || !request.code) return
    ranRef.current = true

    const { provider, code, state } = request
    authApi
      .oauthExchange(provider, code, state)
      .then(() => navigate(takeOAuthNext() ?? "/", { replace: true }))
      .catch((err) =>
        setError(
          err instanceof ApiClientError
            ? err.message
            : "We couldn't complete sign-in. Please try again.",
        ),
      )
  }, [request, navigate])

  return (
    <div className="dark min-h-screen bg-vn-bg text-vn-text flex items-center justify-center px-6">
      <div className="w-full max-w-sm rounded-xl border border-vn-hairline-2 bg-vn-surface p-8 text-center">
        {error ? (
          <>
            <h1 className="text-lg font-[540] tracking-[-0.02em]">Sign-in failed</h1>
            <p className="mt-2 text-sm text-vn-text-3" role="alert">
              {error}
            </p>
            <Link
              to="/login"
              className="mt-6 inline-block text-sm text-vn-accent hover:text-vn-accent-2"
            >
              Back to sign in
            </Link>
          </>
        ) : (
          <>
            <div className="mx-auto h-8 w-8 animate-spin rounded-full border-2 border-vn-hairline-2 border-t-vn-accent" />
            <p className="mt-4 text-sm text-vn-text-3">Signing you in…</p>
          </>
        )}
      </div>
    </div>
  )
}
