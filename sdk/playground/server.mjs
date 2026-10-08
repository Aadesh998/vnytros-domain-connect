// Custom-domain playground: a page where anyone can check a real domain, see
// the records to add, and watch them go live. It is also the reference
// integration for an app adding "custom domain" support.
//
// The API key lives here, on the server. The page only ever talks to the
// /api/* routes below, never to the Vnytros API directly.
//
//   VNYTROS_API_KEY=... PLAYGROUND_IP=203.0.113.10 node playground/server.mjs [--base-url=URL]
//
// The API URL is your own Vnytros deployment: --base-url, else VNYTROS_BASE_URL,
// else http://localhost:8000 (the API's local dev port).
//
// Zero dependencies: it imports the SDK from ../dist, so run `npm run build`
// first.

import { createServer } from "node:http";
import { readFile } from "node:fs/promises";
import { fileURLToPath } from "node:url";
import { Vnytros, VnytrosError } from "../dist/index.js";

const here = (p) => fileURLToPath(new URL(p, import.meta.url));

const env = process.env;
if (!env.VNYTROS_API_KEY) {
  console.error("VNYTROS_API_KEY is not set. Create a key on your deployment's dashboard and export it.");
  process.exit(1);
}

const baseUrlArg = process.argv.find((a) => a.startsWith("--base-url="))?.slice("--base-url=".length);
const BASE_URL = baseUrlArg || env.VNYTROS_BASE_URL || "http://localhost:8000";
const PORT = Number(env.PORT || 5173);
// Prefilled record values. Visitors can edit them; these are just where your
// service actually lives.
const DEFAULT_IP = env.PLAYGROUND_IP || "";
const DEFAULT_TARGET = env.PLAYGROUND_TARGET || "";
const vny = new Vnytros({ apiKey: env.VNYTROS_API_KEY, baseUrl: BASE_URL });

const DOMAIN_RE = /^(?=.{1,253}$)(?!-)([a-z0-9-]{1,63}(?<!-)\.)+[a-z]{2,63}$/;
const IPV4_RE = /^(25[0-5]|2[0-4]\d|1?\d?\d)(\.(25[0-5]|2[0-4]\d|1?\d?\d)){3}$/;

function cleanDomain(value) {
  const d = String(value ?? "").trim().toLowerCase().replace(/^https?:\/\//, "").replace(/[/.]+$/, "");
  if (!DOMAIN_RE.test(d)) throw new HttpError(400, "Enter a domain like example.com.");
  return d;
}

function cleanRecords(body) {
  const ip = String(body.ip ?? "").trim();
  const target = String(body.target ?? "").trim().replace(/\.$/, "").toLowerCase();
  if (ip && !IPV4_RE.test(ip)) throw new HttpError(400, "The A record must be an IPv4 address.");
  if (target && !DOMAIN_RE.test(target)) throw new HttpError(400, "The CNAME target must be a hostname.");
  if (!ip && !target) throw new HttpError(400, "Add at least an A record or a CNAME target.");
  return { ip, target };
}

class HttpError extends Error {
  constructor(status, message) {
    super(message);
    this.status = status;
  }
}

async function readJson(req) {
  let raw = "";
  for await (const chunk of req) {
    raw += chunk;
    if (raw.length > 16_384) throw new HttpError(413, "Request too large.");
  }
  try {
    return raw ? JSON.parse(raw) : {};
  } catch {
    throw new HttpError(400, "Body must be JSON.");
  }
}

function send(res, status, body, type = "application/json") {
  res.writeHead(status, { "Content-Type": type, "Cache-Control": "no-store" });
  res.end(type === "application/json" ? JSON.stringify(body) : body);
}

const routes = {
  "GET /": async (req, res) => send(res, 200, await readFile(here("./index.html")), "text/html; charset=utf-8"),

  "GET /api/config": async (req, res) => send(res, 200, { ip: DEFAULT_IP, target: DEFAULT_TARGET }),

  // Who hosts the DNS, and what currently resolves.
  "GET /api/detect": async (req, res, url) => {
    const domain = cleanDomain(url.searchParams.get("domain"));
    const [detected, lookup] = await Promise.all([vny.detect(domain), vny.dnsLookup(domain).catch(() => null)]);
    send(res, 200, { ...detected, records: lookup?.records ?? [] });
  },

  // Live DNS check, safe to poll.
  "POST /api/status": async (req, res) => {
    const body = await readJson(req);
    const domain = cleanDomain(body.domain);
    const { ip, target } = cleanRecords(body);
    send(res, 200, await vny.domains.status({ domain, ip, target, txt: String(body.txt ?? "") }));
  },
};

createServer(async (req, res) => {
  const url = new URL(req.url ?? "/", "http://localhost");
  const route = routes[`${req.method} ${url.pathname}`];
  if (!route) return send(res, 404, { message: "Not found" });
  try {
    await route(req, res, url);
  } catch (err) {
    if (err instanceof HttpError) return send(res, err.status, { message: err.message });
    if (err instanceof VnytrosError) {
      const status = err.statusCode && err.statusCode < 500 ? err.statusCode : 502;
      return send(res, status, { message: err.message });
    }
    console.error(err);
    send(res, 500, { message: "Something went wrong." });
  }
}).listen(PORT, () => {
  console.log(`Custom-domain playground on http://localhost:${PORT} (API ${BASE_URL})`);
});
