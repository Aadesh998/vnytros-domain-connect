import { DomainsResource } from "./resources/domains.js";
import { ToolsResource } from "./resources/tools.js";
import { Transport, type FetchLike } from "./transport.js";
import { Webhooks } from "./webhooks.js";
import { VnytrosError } from "./errors.js";
import type { DetectResult, DnsLookupResult } from "./types.js";

export interface VnytrosOptions {
  /**
   * Your secret API key. This decodes to your account identity, so treat it
   * like a password: keep it in an environment variable and never ship it to a
   * browser bundle.
   */
  apiKey: string;
  /**
   * Base URL of your Vnytros API deployment, e.g. `https://api.your-domain.com`
   * or `http://localhost:8000`. Required: Vnytros is self-hosted and there is
   * no hosted Vnytros API to fall back to.
   */
  baseUrl: string;
  /** Per-request timeout in ms. Default 30000. */
  timeoutMs?: number;
  /** Extra attempts for idempotent requests. Default 2. */
  maxRetries?: number;
  /** Secret for verifying inbound webhooks (`WEBHOOK_SIGNING_SECRET`). */
  webhookSecret?: string;
  /** Inject a fetch implementation, e.g. for tests or a proxy agent. */
  fetch?: FetchLike;
  /**
   * Escape hatch for the browser guard. Only set this if you have deliberately
   * scoped the key and accept that anyone can read it from your bundle.
   */
  dangerouslyAllowBrowser?: boolean;
}

/**
 * The Vnytros Domain Connect API client.
 *
 * Server-side only. Construct it once and reuse it — it holds no per-request
 * state and the underlying `fetch` pools its own connections.
 *
 * ```ts
 * const vny = new Vnytros({
 *   apiKey: process.env.VNYTROS_API_KEY!,
 *   baseUrl: process.env.VNYTROS_BASE_URL!, // e.g. https://api.your-domain.com
 * });
 * const { provider } = await vny.detect("shop.acme.com");
 * const { is_configured } = await vny.domains.status({
 *   domain: "shop.acme.com",
 *   target: "cname.myplatform.com",
 * });
 * ```
 */
export class Vnytros {
  readonly domains: DomainsResource;
  readonly tools: ToolsResource;
  readonly webhooks: Webhooks;
  readonly #transport: Transport;

  constructor(options: VnytrosOptions) {
    if (!options?.apiKey || typeof options.apiKey !== "string") {
      throw new VnytrosError(
        "A Vnytros apiKey is required. Create one on your deployment's dashboard and read it from an environment variable.",
        { code: "INVALID_INPUT" },
      );
    }

    if (!options.baseUrl || typeof options.baseUrl !== "string") {
      throw new VnytrosError(
        "A Vnytros baseUrl is required: the URL of your own Vnytros API deployment, e.g. " +
          '"https://api.your-domain.com" or "http://localhost:8000". There is no hosted Vnytros API.',
        { code: "INVALID_INPUT" },
      );
    }
    let parsedBase: URL;
    try {
      parsedBase = new URL(options.baseUrl);
    } catch {
      throw new VnytrosError(
        `Invalid Vnytros baseUrl "${options.baseUrl}". Use an absolute URL such as "https://api.your-domain.com".`,
        { code: "INVALID_INPUT" },
      );
    }
    if (parsedBase.protocol !== "http:" && parsedBase.protocol !== "https:") {
      throw new VnytrosError(
        `Invalid Vnytros baseUrl "${options.baseUrl}": it must start with http:// or https://.`,
        { code: "INVALID_INPUT" },
      );
    }

    // Fail loudly at construction rather than leaking the key to every visitor.
    if (typeof window !== "undefined" && !options.dangerouslyAllowBrowser) {
      throw new VnytrosError(
        "The Vnytros client was constructed in a browser, which would expose your secret API key to anyone who opens devtools. " +
          "Call it from your server (a Next.js route handler, an Express route) and have your page talk to that route instead.",
        { code: "FORBIDDEN" },
      );
    }

    const fetchImpl = options.fetch ?? (globalThis.fetch as FetchLike | undefined);
    if (!fetchImpl) {
      throw new VnytrosError(
        "No global fetch found. Use Node 18 or newer, or pass a `fetch` implementation.",
        { code: "INVALID_INPUT" },
      );
    }

    this.#transport = new Transport({
      baseUrl: options.baseUrl.replace(/\/+$/, "") + "/",
      apiKey: options.apiKey,
      timeoutMs: options.timeoutMs ?? 30_000,
      maxRetries: options.maxRetries ?? 2,
      fetch: fetchImpl,
    });

    this.domains = new DomainsResource(this.#transport);
    this.tools = new ToolsResource(this.#transport);
    this.webhooks = new Webhooks(options.webhookSecret);
  }

  /**
   * Identifies the DNS provider behind a domain from its nameservers. Read-only
   * and works for any provider; use it to show the right setup instructions or
   * to pick a provider for `domains.connectDirect()`.
   */
  detect(domain: string, signal?: AbortSignal): Promise<DetectResult> {
    return this.#transport.request<DetectResult>({
      method: "GET",
      path: "/v1/detect",
      query: { domain },
      signal,
    });
  }

  /** Current A, AAAA, CNAME, MX, TXT and NS records for a domain. */
  dnsLookup(domain: string, signal?: AbortSignal): Promise<DnsLookupResult> {
    return this.#transport.request<DnsLookupResult>({
      method: "GET",
      path: "/v1/dns/lookup",
      query: { domain },
      signal,
    });
  }
}

export {
  VnytrosError,
  VnytrosAuthError,
  VnytrosValidationError,
  VnytrosNotFoundError,
  VnytrosServerError,
  VnytrosConnectionError,
  VnytrosTimeoutError,
  VnytrosRateLimitError,
  VnytrosSignatureVerificationError,
} from "./errors.js";
export type { VnytrosErrorCode } from "./errors.js";
export { SIGNATURE_HEADER, TIMESTAMP_HEADER, EVENT_HEADER } from "./webhooks.js";
export type { ConstructEventOptions, HeaderSource } from "./webhooks.js";
export type { WaitOptions } from "./resources/domains.js";
export type { FetchLike } from "./transport.js";
export type * from "./types.js";
export default Vnytros;
