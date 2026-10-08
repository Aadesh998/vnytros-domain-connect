/**
 * Shown instead of the app when VITE_API_BASE_URL is not set. Production
 * builds already refuse to run without it (vite.config.ts); this covers
 * `npm run dev` with a missing .env.local.
 */
export function ConfigError() {
  return (
    <div className="dark min-h-screen bg-vn-bg text-vn-text flex items-center justify-center px-6">
      <div className="w-full max-w-md">
        <h1 className="text-xl font-[540] tracking-[-0.02em]">
          Dashboard not configured
        </h1>
        <p className="mt-3 text-sm text-vn-text-2">
          <code className="font-mono">VITE_API_BASE_URL</code> is not set, so
          the dashboard doesn&apos;t know which Vnytros API to talk to. Vnytros
          is self-hosted: point it at your own deployment.
        </p>
        <pre className="mt-4 overflow-x-auto rounded-md border border-vn-hairline bg-vn-surface-2 p-3 text-xs text-vn-text-2">
          {`cp .env.example .env.local
# edit .env.local:
VITE_API_BASE_URL=http://localhost:8000
# then restart: npm run dev`}
        </pre>
        <p className="mt-3 text-xs text-vn-text-3">
          See the README for every variable.
        </p>
      </div>
    </div>
  )
}
