import path from "path"
import { defineConfig, loadEnv } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'
import { cloudflare } from "@cloudflare/vite-plugin"

// https://vite.dev/config/
export default defineConfig(({ command, mode }) => {
  // Vnytros is self-hosted, so there is no default API to fall back to. Fail
  // the production build loudly instead of shipping a dashboard that calls
  // nothing (or someone else's server). `npm run dev` shows a config error
  // screen instead (src/components/ConfigError.tsx).
  if (command === "build") {
    const env = loadEnv(mode, process.cwd(), "VITE_")
    const apiBase = (process.env.VITE_API_BASE_URL ?? env.VITE_API_BASE_URL ?? "").trim()
    if (!apiBase) {
      throw new Error(
        "VITE_API_BASE_URL is not set. Set it to your own Vnytros API URL " +
          "(e.g. https://api.your-domain.com) in .env.production.local, .env.local, " +
          "or the build environment. See README.md.",
      )
    }
  }

  return {
    plugins: [react(), tailwindcss(), cloudflare()],
    resolve: {
      alias: {
        "@": path.resolve(__dirname, "./src"),
      },
    },
  }
})
