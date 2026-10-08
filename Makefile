# Convenience targets for the Vnytros monorepo. Every target delegates into
# one folder; each folder also works on its own (server/ has its own Makefile
# with the run, migration, keys and remote-deploy targets).
#
#   make help        list targets
#   make install     npm ci in every JS folder
#   make test        every folder's checks (what CI runs)

NPM ?= npm
GO  ?= go

# Placeholder API URL for check builds only. Real dashboard builds take
# VITE_API_BASE_URL from your environment or dashboard/.env.production.local.
CHECK_API_BASE_URL ?= http://localhost:8000

.PHONY: help install test \
        server-build server-vet server-test server-check server-run \
        dashboard-install dashboard-dev dashboard-lint dashboard-build dashboard-check \
        website-install website-dev website-lint website-build website-build-cloudflare website-check \
        docs-install docs-dev docs-build docs-audit docs-check \
        sdk-install sdk-build sdk-typecheck sdk-test sdk-pack sdk-check

help:
	@grep -E '^[a-z-]+:.*## ' $(MAKEFILE_LIST) | sort | awk -F':.*## ' '{printf "  %-26s %s\n", $$1, $$2}'

install: dashboard-install website-install docs-install sdk-install ## npm ci in every JS folder

test: server-check dashboard-check website-check docs-check sdk-check ## Run every folder's checks (same as CI)

# ── server/ (Go) ────────────────────────────────────────────────────────
server-build: ## Build the three binaries into server/bin/
	$(MAKE) -C server build-all

server-vet: ## go vet ./... in server/
	cd server && $(GO) vet ./...

server-test: ## go test ./... in server/
	cd server && $(GO) test ./...

server-check: ## go build + vet + test in server/
	cd server && $(GO) build ./... && $(GO) vet ./... && $(GO) test ./...

server-run: ## Run the API on :8000 (needs server/.env.production, see server/README.md)
	$(MAKE) -C server run

# ── dashboard/ (Vite + React) ───────────────────────────────────────────
dashboard-install: ## npm ci in dashboard/
	cd dashboard && $(NPM) ci

dashboard-dev: ## Vite dev server on :5173
	cd dashboard && $(NPM) run dev

dashboard-lint: ## ESLint in dashboard/
	cd dashboard && $(NPM) run lint

dashboard-build: ## Production build to dashboard/dist (set VITE_API_BASE_URL)
	cd dashboard && $(NPM) run build

dashboard-check: ## tsc -b, lint and build with a placeholder API URL
	cd dashboard && npx tsc -b && $(NPM) run lint && VITE_API_BASE_URL=$(CHECK_API_BASE_URL) $(NPM) run build

# ── website/ (Next.js on Cloudflare Workers via OpenNext) ───────────────
website-install: ## npm ci in website/
	cd website && $(NPM) ci

website-dev: ## Next.js dev server on :3000
	cd website && $(NPM) run dev

website-lint: ## ESLint in website/
	cd website && $(NPM) run lint

website-build: ## Plain next build (no Cloudflare credentials needed)
	cd website && $(NPM) run build:next

website-build-cloudflare: ## OpenNext build for Cloudflare Workers (.open-next/)
	cd website && $(NPM) run build

website-check: ## tsc, lint and next build
	cd website && npx tsc --noEmit && $(NPM) run lint && $(NPM) run build:next

# ── docs/ (Fumadocs on Next.js) ─────────────────────────────────────────
docs-install: ## npm ci in docs/
	cd docs && $(NPM) ci

docs-dev: ## Docs dev server on :3000/docs
	cd docs && $(NPM) run dev

docs-build: ## next build in docs/
	cd docs && $(NPM) run build

docs-audit: ## Diff server/internal/server/server.go routes against the docs
	cd docs && $(NPM) run audit:docs

docs-check: ## build and audit:docs
	cd docs && $(NPM) run build && $(NPM) run audit:docs

# ── sdk/ (@vnytros/sdk) ─────────────────────────────────────────────────
sdk-install: ## npm ci in sdk/
	cd sdk && $(NPM) ci

sdk-build: ## Build sdk/dist
	cd sdk && $(NPM) run build

sdk-typecheck: ## tsc --noEmit in sdk/
	cd sdk && $(NPM) run typecheck

sdk-test: ## Build, then run the SDK tests
	cd sdk && $(NPM) test

sdk-pack: ## npm pack -> sdk/vnytros-sdk-<version>.tgz (installable without npm registry)
	cd sdk && $(NPM) pack

sdk-check: ## typecheck and test
	cd sdk && $(NPM) run typecheck && $(NPM) test
