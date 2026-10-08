import type { Transport } from "../transport.js";
import { VnytrosTimeoutError, VnytrosValidationError } from "../errors.js";
import type {
  ConnectDirectParams,
  ConnectDirectResult,
  Domain,
  DomainStatusResult,
  DomainVerifyResult,
  StatusParams,
} from "../types.js";

export interface WaitOptions {
  /** Give up after this long. Default 120000 (2 minutes). */
  timeoutMs?: number;
  /** Gap between polls. Default 3000. DNS rarely propagates faster. */
  intervalMs?: number;
  signal?: AbortSignal;
  /** Called after each poll, for progress UI. */
  onPoll?: (status: DomainStatusResult, attempt: number) => void;
}

export class DomainsResource {
  readonly #transport: Transport;

  constructor(transport: Transport) {
    this.#transport = transport;
  }

  /**
   * Applies records straight to the provider using credentials you supply.
   * Supported providers: AWS Route 53 (`"aws"`) and Hostinger (`"hostinger"`).
   * Only record creation is implemented; the API cannot yet delete or list
   * records through a provider.
   *
   * Note the API places records at fixed hosts: the A record at `@`, the CNAME
   * at `www`, and the TXT at `@`. It applies immediately, so a success here
   * means the records are already written.
   *
   * `providerConfig` carries live credentials, so this must only run
   * server-side, and never with credentials belonging to your end user's
   * account unless they handed them to you deliberately.
   */
  // `async` so a validation failure arrives as a rejection. Throwing
  // synchronously from a Promise-returning method escapes `.catch()` and
  // becomes an uncaught exception in the caller's request handler.
  async connectDirect(
    params: ConnectDirectParams,
    signal?: AbortSignal,
  ): Promise<ConnectDirectResult> {
    assertProviderConfig(params);
    return this.#transport.request<ConnectDirectResult>({
      method: "POST",
      path: "/v1/connect/direct",
      body: {
        domain: params.domain,
        provider: params.provider,
        ip: params.ip ?? "",
        target: params.target ?? "",
        txt: params.txt ?? "",
        provider_config: params.providerConfig ?? null,
      },
      signal,
    });
  }

  /**
   * Live DNS check of the expected records. Read-only and safe to poll, and it
   * reflects what is actually resolving rather than what we last stored.
   */
  status(params: StatusParams, signal?: AbortSignal): Promise<DomainStatusResult> {
    return this.#transport.request<DomainStatusResult>({
      method: "POST",
      path: "/v1/status",
      body: {
        domain: params.domain,
        ip: params.ip ?? "",
        target: params.target ?? "",
        txt: params.txt ?? "",
      },
      // A status read has no side effects, so a retry is free.
      idempotent: true,
      signal,
    });
  }

  /**
   * Re-runs verification and persists the result. On the transition to
   * `completed` this fires the `domain.connected` webhook, so prefer
   * {@link status} for polling.
   */
  verify(domain: string, signal?: AbortSignal): Promise<DomainVerifyResult> {
    return this.#transport.request<DomainVerifyResult>({
      method: "POST",
      path: "/v1/connect/verify",
      body: { domain },
      signal,
    });
  }

  /** Every domain registered under the API key's account. */
  list(signal?: AbortSignal): Promise<Domain[]> {
    return this.#transport.request<Domain[]>({
      method: "GET",
      path: "/v1/connect/domains",
      signal,
    });
  }

  /** One domain by name. Throws `VnytrosNotFoundError` if it is not yours. */
  get(domain: string, signal?: AbortSignal): Promise<Domain> {
    return this.#transport.request<Domain>({
      method: "GET",
      path: "/v1/connect/domain",
      query: { domain },
      signal,
    });
  }

  /**
   * Polls {@link status} until the records resolve, then returns. Use this when
   * you cannot receive webhooks — a CLI, a script, or a page waiting on a
   * spinner. Throws `VnytrosTimeoutError` if the deadline passes first, which
   * means "not propagated yet", not "failed".
   */
  async waitUntilConnected(
    params: StatusParams,
    options: WaitOptions = {},
  ): Promise<DomainStatusResult> {
    const { timeoutMs = 120_000, intervalMs = 3_000, signal, onPoll } = options;
    const deadline = Date.now() + timeoutMs;
    let attempt = 0;
    let last: DomainStatusResult | undefined;

    while (true) {
      const result = await this.status(params, signal);
      last = result;
      onPoll?.(result, ++attempt);
      if (result.is_configured) return result;

      const remaining = deadline - Date.now();
      if (remaining <= 0) {
        throw new VnytrosTimeoutError(
          `${params.domain} was still not configured after ${timeoutMs}ms (${attempt} checks)`,
          { code: "TIMEOUT", raw: last },
        );
      }
      await new Promise((r) => setTimeout(r, Math.min(intervalMs, remaining)));
    }
  }
}

/**
 * The server unmarshals `provider_config` into Go structs with no JSON tags, so
 * an unrecognised key is dropped silently and surfaces later as a confusing
 * "AWS Access Key is empty". Catching it here names the key that is actually
 * wrong, which is the difference between a five-second and a one-hour fix.
 */
const REQUIRED_CONFIG_KEYS: Record<string, readonly string[]> = {
  aws: ["accessKey", "secretKey"],
  hostinger: ["apiToken"],
};

function assertProviderConfig(params: ConnectDirectParams): void {
  const expected = REQUIRED_CONFIG_KEYS[params.provider];
  if (!expected) {
    throw new VnytrosValidationError(
      `Unknown provider "${params.provider}". Supported: ${Object.keys(REQUIRED_CONFIG_KEYS).join(", ")}. ` +
        `Note the slug is "aws" (not "route53"). Name.com is not supported.`,
      { code: "INVALID_INPUT" },
    );
  }

  const config = (params.providerConfig ?? {}) as unknown as Record<string, unknown>;
  const missing = expected.filter((key) => typeof config[key] !== "string" || config[key] === "");

  if (missing.length > 0) {
    const supplied = Object.keys(config);
    // snake_case is the near-miss developers reach for, and it fails silently.
    const nearMiss = missing.filter((key) =>
      supplied.some((given) => given.replace(/_/g, "").toLowerCase() === key.toLowerCase()),
    );
    const hint =
      nearMiss.length > 0
        ? ` The API matches keys case-insensitively but not across underscores, so write ${nearMiss
            .map((k) => `"${k}"`)
            .join(", ")} exactly.`
        : "";
    throw new VnytrosValidationError(
      `providerConfig for "${params.provider}" is missing ${missing.map((k) => `"${k}"`).join(", ")}` +
        `${supplied.length > 0 ? ` (received: ${supplied.join(", ")})` : ""}.${hint}`,
      { code: "INVALID_INPUT" },
    );
  }
}
