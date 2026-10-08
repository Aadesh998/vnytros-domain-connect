import { authApi, type OAuthProvider } from "@/lib/auth/api"

/**
 * Social sign-in. Both providers hand the browser back to
 * /auth/:provider/callback on this origin (see OAuthCallbackPage).
 */
export function OAuthButtons({
  next,
  disabled,
}: {
  next?: string | null
  disabled?: boolean
}) {
  return (
    <div className="grid grid-cols-2 gap-2">
      <OAuthButton provider="google" label="Google" next={next} disabled={disabled} />
      <OAuthButton provider="github" label="GitHub" next={next} disabled={disabled} />
    </div>
  )
}

function OAuthButton({
  provider,
  label,
  next,
  disabled,
}: {
  provider: OAuthProvider
  label: string
  next?: string | null
  disabled?: boolean
}) {
  return (
    <button
      type="button"
      disabled={disabled}
      onClick={() => authApi.startOAuth(provider, next)}
      className="inline-flex h-9 items-center justify-center gap-2 rounded-md border border-vn-hairline-2 bg-vn-surface text-sm font-medium text-vn-text transition-colors hover:bg-vn-surface-2 hover:border-vn-hairline-3 disabled:cursor-not-allowed disabled:opacity-60"
    >
      {provider === "google" ? <GoogleIcon /> : <GithubIcon />}
      {label}
    </button>
  )
}

function GoogleIcon() {
  return (
    <svg width="15" height="15" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
      <path d="M21.6 12.2c0-.7-.1-1.3-.2-1.9H12v3.6h5.4c-.2 1.2-.9 2.2-2 2.9v2.4h3.2c1.9-1.7 3-4.3 3-7Z" />
      <path d="M12 22c2.7 0 5-.9 6.6-2.4l-3.2-2.4c-.9.6-2 .9-3.4.9-2.6 0-4.8-1.7-5.6-4.1H3.1v2.5A10 10 0 0 0 12 22Z" />
      <path d="M6.4 14c-.2-.6-.3-1.3-.3-2s.1-1.4.3-2V7.5H3.1A10 10 0 0 0 2 12c0 1.6.4 3.1 1.1 4.5L6.4 14Z" />
      <path d="M12 5.9c1.5 0 2.8.5 3.8 1.5l2.8-2.8C16.9 3 14.7 2 12 2 8.1 2 4.7 4.3 3.1 7.5L6.4 10C7.2 7.6 9.4 5.9 12 5.9Z" />
    </svg>
  )
}

function GithubIcon() {
  return (
    <svg width="15" height="15" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
      <path d="M12 2C6.5 2 2 6.5 2 12c0 4.4 2.9 8.2 6.8 9.5.5.1.7-.2.7-.5v-1.7c-2.8.6-3.4-1.3-3.4-1.3-.5-1.2-1.1-1.5-1.1-1.5-.9-.6.1-.6.1-.6 1 .1 1.5 1 1.5 1 .9 1.5 2.4 1.1 3 .8.1-.7.4-1.1.6-1.4-2.2-.3-4.6-1.1-4.6-5 0-1.1.4-2 1-2.7-.1-.3-.4-1.3.1-2.7 0 0 .8-.3 2.8 1a9.6 9.6 0 0 1 5 0c2-1.3 2.8-1 2.8-1 .5 1.4.2 2.4.1 2.7.6.7 1 1.6 1 2.7 0 3.9-2.4 4.7-4.6 5 .4.3.7.9.7 1.8v2.7c0 .3.2.6.7.5C19.1 20.2 22 16.4 22 12c0-5.5-4.5-10-10-10Z" />
    </svg>
  )
}

export function AuthDivider({ label }: { label: string }) {
  return (
    <div className="my-5 flex items-center gap-3">
      <span className="h-px flex-1 bg-vn-hairline-2" />
      <span className="text-[11px] uppercase tracking-wide text-vn-text-3">{label}</span>
      <span className="h-px flex-1 bg-vn-hairline-2" />
    </div>
  )
}
