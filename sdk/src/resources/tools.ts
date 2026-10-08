import type { Transport } from "../transport.js";
import type { Report } from "../types.js";

/**
 * The diagnostic tools under `/v1/tools`. Every one returns the same {@link
 * Report} envelope, so a single UI component can render all of them.
 *
 * These endpoints are public but rate limited to 60 requests/minute per IP.
 */
export class ToolsResource {
  readonly #transport: Transport;

  constructor(transport: Transport) {
    this.#transport = transport;
  }

  private check(path: string, query: Record<string, string | number | undefined>, signal?: AbortSignal) {
    return this.#transport.request<Report>({ method: "GET", path, query, signal });
  }

  /** Which tools the API exposes, for building a menu dynamically. */
  catalogue(signal?: AbortSignal): Promise<unknown> {
    return this.#transport.request<unknown>({ method: "GET", path: "/v1/tools", signal });
  }

  /** Whether a record has propagated across public resolvers. */
  dnsPropagation(domain: string, recordType = "A", signal?: AbortSignal) {
    return this.check("/v1/tools/dns/propagation", { domain, type: recordType }, signal);
  }

  nameservers(domain: string, signal?: AbortSignal) {
    return this.check("/v1/tools/dns/nameservers", { domain }, signal);
  }

  /** Follows the CNAME chain, flagging loops and dead ends. */
  cnameChain(domain: string, signal?: AbortSignal) {
    return this.check("/v1/tools/dns/cname", { domain }, signal);
  }

  caa(domain: string, signal?: AbortSignal) {
    return this.check("/v1/tools/dns/caa", { domain }, signal);
  }

  soa(domain: string, signal?: AbortSignal) {
    return this.check("/v1/tools/dns/soa", { domain }, signal);
  }

  dnssec(domain: string, signal?: AbortSignal) {
    return this.check("/v1/tools/dns/dnssec", { domain }, signal);
  }

  /**
   * SPF, DKIM and DMARC in one report. `selectors` are optional DKIM selectors
   * to probe; omit them and the API tries common ones.
   */
  emailAuth(domain: string, selectors?: string[], signal?: AbortSignal) {
    return this.check("/v1/tools/email/auth", { domain, selector: selectors?.join(",") || undefined }, signal);
  }

  spf(domain: string, signal?: AbortSignal) {
    return this.check("/v1/tools/email/spf", { domain }, signal);
  }

  /** DKIM public keys for the given selectors, or common ones when omitted. */
  dkim(domain: string, selectors?: string[], signal?: AbortSignal) {
    return this.check("/v1/tools/email/dkim", { domain, selector: selectors?.join(",") || undefined }, signal);
  }

  dmarc(domain: string, signal?: AbortSignal) {
    return this.check("/v1/tools/email/dmarc", { domain }, signal);
  }

  /** Registration and RDAP data for the domain. */
  domainInfo(domain: string, signal?: AbortSignal) {
    return this.check("/v1/tools/domain/info", { domain }, signal);
  }

  tlsCertificate(domain: string, port?: number, signal?: AbortSignal) {
    return this.check("/v1/tools/tls/certificate", { domain, port }, signal);
  }

  securityHeaders(url: string, signal?: AbortSignal) {
    return this.check("/v1/tools/http/headers", { url }, signal);
  }

  redirectChain(url: string, signal?: AbortSignal) {
    return this.check("/v1/tools/http/redirects", { url }, signal);
  }

  /** Geo and network ownership for an IP or hostname. */
  ipInfo(target: string, signal?: AbortSignal) {
    return this.check("/v1/tools/ip/info", { target }, signal);
  }
}
