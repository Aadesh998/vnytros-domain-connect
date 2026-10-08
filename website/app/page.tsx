import Navbar from "@/components/Navbar";
import Footer from "@/components/Footer";
import { DOCS_ENTRY, REPO_URL, REPOS, repoTree } from "@/lib/config";

const FEATURES = [
  {
    title: "Provider detection",
    body: "Find out which DNS host serves any domain. Read-only, so it works whatever the provider is.",
  },
  {
    title: "Direct DNS record creation",
    body: "Create records through the AWS Route 53 and Hostinger APIs, using credentials you supply. Creating records is the only write supported so far.",
  },
  {
    title: "Domain status and verification",
    body: "Check that the expected A, CNAME and TXT records are live and verify a domain. Results go out as webhooks, HMAC-signed with a timestamp.",
  },
  {
    title: "15 diagnostic checks",
    body: "DNS propagation across 25 public resolvers, nameservers, CNAME chains, CAA, SOA, DNSSEC, SPF, DKIM, DMARC, combined email authentication, RDAP registration, TLS certificates, HTTP security headers, redirects and IP info.",
  },
  {
    title: "REST API and MCP server",
    body: "Use the domain tools and diagnostics over the REST API, or connect an AI client to the built-in Model Context Protocol server.",
  },
  {
    title: "Vnytros Mail",
    body: "Campaigns, templates and your own SMTP senders, with open tracking. All of it is in the dashboard.",
  },
  {
    title: "JavaScript SDK",
    body: "@vnytros/sdk is a typed client for Node 18+ with no runtime dependencies. It covers detection, record writes, status checks and webhook verification. MIT licensed.",
  },
];

const TEMPLATE_PRS = [
  {
    href: "https://github.com/Domain-Connect/Templates/pull/1064",
    label: "PR #1064",
    body: "vnytros.dev.custom-domain.json template, merged 1 May 2026",
  },
  {
    href: "https://github.com/Domain-Connect/Templates/pull/1166",
    label: "PR #1166",
    body: "Adding more records, merged 31 May 2026",
  },
];

const TEMPLATE_FILE_URL =
  "https://github.com/Domain-Connect/Templates/blob/master/vnytros.dev.custom-domain.json";

const SELF_HOST_STEPS = `git clone ${REPO_URL}
cd vnytros/server
cp .env.example .env.production   # set JWT_SECRET, APIKEY_SIGNING_SECRET, WEBHOOK_SIGNING_SECRET
make keys                         # Domain Connect keypair into keys/
docker compose up -d db rabbitmq  # PostgreSQL and RabbitMQ
make migration                    # create the schema
make run                          # REST API on :8000
make run-worker                   # background jobs (separate terminal)
make run-mcp                      # MCP server on :5000 (separate terminal)`;

const h2 =
  "mb-4 text-[26px] sm:text-[30px] font-[540] leading-[1.15] tracking-[-0.03em] text-vn-text";
const link = "text-vn-accent hover:underline underline-offset-2";

export default function Home() {
  return (
    <main id="main-content" className="min-h-screen">
      <Navbar />

      <div className="mx-auto max-w-5xl px-4 sm:px-6">
        {/* What it is */}
        <section className="pt-16 pb-14 sm:pt-24">
          <span className="vn-eyebrow">open source · AGPL-3.0</span>
          <h1 className="mt-5 max-w-3xl text-[38px] sm:text-[52px] font-[540] leading-[1.05] tracking-[-0.04em] text-vn-text">
            Self-hosted DNS and email-authentication toolkit
          </h1>
          <p className="mt-6 max-w-2xl text-[17px] leading-[1.6] text-vn-text-2">
            Vnytros is an open-source toolkit for DNS and email
            authentication, with a mail platform built in. You run it on your
            own infrastructure. There is no hosted service and no paid plan.
            The server, dashboard and docs are licensed under AGPL-3.0, and
            the SDK under MIT.
          </p>
          <div className="mt-8 flex flex-wrap gap-2.5">
            <a
              href={repoTree("server")}
              className="inline-flex h-11 items-center justify-center rounded-[10px] bg-vn-accent px-5 text-[14.5px] font-medium text-vn-on-accent shadow-vn-glow transition-colors hover:bg-vn-accent-2"
            >
              Get the server on GitHub
            </a>
            <a
              href={DOCS_ENTRY}
              className="inline-flex h-11 items-center justify-center rounded-[10px] border border-vn-hairline-2 bg-vn-surface px-5 text-[14.5px] font-medium text-vn-text transition-colors hover:bg-vn-surface-2"
            >
              Read the docs
            </a>
          </div>
        </section>

        {/* What it does */}
        <section aria-labelledby="features" className="py-12">
          <h2 id="features" className={h2}>
            What it does
          </h2>
          <ul className="grid grid-cols-1 gap-px overflow-hidden rounded-2xl border border-vn-hairline bg-vn-hairline sm:grid-cols-2">
            {FEATURES.map((f) => (
              <li key={f.title} className="bg-vn-surface p-6">
                <h3 className="text-[15.5px] font-[540] tracking-[-0.01em] text-vn-text">
                  {f.title}
                </h3>
                <p className="mt-2 text-[14px] leading-[1.6] text-vn-text-2">
                  {f.body}
                </p>
              </li>
            ))}
          </ul>
        </section>

        {/* Domain Connect */}
        <section aria-labelledby="domain-connect" className="py-12">
          <h2 id="domain-connect" className={h2}>
            Accepted Domain Connect template
          </h2>
          <p className="max-w-2xl text-[15.5px] leading-[1.6] text-vn-text-2">
            The official{" "}
            <a href="https://github.com/Domain-Connect/Templates" className={link}>
              Domain-Connect/Templates
            </a>{" "}
            repository has accepted the Vnytros Domain Connect template. You
            can read it there as{" "}
            <a href={TEMPLATE_FILE_URL} className={`${link} font-mono text-[14px]`}>
              vnytros.dev.custom-domain.json
            </a>
            .
          </p>
          <ul className="mt-5 flex flex-col gap-2.5">
            {TEMPLATE_PRS.map((pr) => (
              <li key={pr.href} className="text-[14.5px] text-vn-text-2">
                <a href={pr.href} className={`${link} font-mono`}>
                  {pr.label}
                </a>{" "}
                — {pr.body}
              </li>
            ))}
          </ul>
        </section>

        {/* Self-host */}
        <section aria-labelledby="self-host" className="py-12">
          <h2 id="self-host" className={h2}>
            Self-host in a few steps
          </h2>
          <p className="max-w-2xl text-[15.5px] leading-[1.6] text-vn-text-2">
            You need Go, Docker with Compose, <code className="font-mono text-[14px]">make</code>{" "}
            and <code className="font-mono text-[14px]">openssl</code>. These
            steps run the server locally:
          </p>
          <pre className="mt-5 overflow-x-auto rounded-xl border border-vn-hairline bg-vn-surface p-5 font-mono text-[13px] leading-[1.7] text-vn-text">
            <code>{SELF_HOST_STEPS}</code>
          </pre>
          <p className="mt-5 max-w-2xl text-[15.5px] leading-[1.6] text-vn-text-2">
            For the web UI, run the{" "}
            <a href={repoTree("dashboard")} className={link}>
              dashboard
            </a>{" "}
            with <code className="font-mono text-[14px]">VITE_API_BASE_URL</code>{" "}
            set to your API. To deploy to a real server with nginx and TLS,
            follow <span className="font-mono text-[14px]">server/deploy/README.md</span>{" "}
            in the repository. The full guide is on{" "}
            <a href={DOCS_ENTRY} className={link}>
              docs.vnytros.dev
            </a>
            .
          </p>
        </section>

        {/* Repositories */}
        <section aria-labelledby="repos" className="py-12">
          <h2 id="repos" className={h2}>
            One repository, five folders
          </h2>
          <ul className="divide-y divide-vn-hairline overflow-hidden rounded-2xl border border-vn-hairline bg-vn-surface">
            {REPOS.map((r) => (
              <li key={r.name}>
                <a
                  href={r.href}
                  className="flex items-center justify-between gap-4 px-5 py-4 transition-colors hover:bg-vn-surface-2"
                >
                  <span className="min-w-0">
                    <span className="block truncate font-mono text-[13.5px] text-vn-text">
                      {r.name}
                    </span>
                    <span className="block text-[13px] text-vn-text-3">
                      {r.body}
                    </span>
                  </span>
                  <span className="shrink-0 font-mono text-[11px] uppercase tracking-[0.06em] text-vn-text-4">
                    {r.license}
                  </span>
                </a>
              </li>
            ))}
          </ul>
        </section>
      </div>

      <Footer />
    </main>
  );
}
