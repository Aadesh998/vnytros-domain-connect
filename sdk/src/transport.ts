import {
  VnytrosConnectionError,
  VnytrosRateLimitError,
  VnytrosTimeoutError,
  errorFromResponse,
} from "./errors.js";

export type FetchLike = (input: string, init?: RequestInit) => Promise<Response>;

interface TransportConfig {
  baseUrl: string;
  apiKey?: string;
  timeoutMs: number;
  maxRetries: number;
  fetch: FetchLike;
}

interface RequestOptions {
  method: "GET" | "POST" | "PATCH" | "DELETE";
  path: string;
  query?: Record<string, string | number | undefined>;
  body?: unknown;
  /**
   * Whether replaying this request is harmless. Only idempotent requests are
   * retried: `POST /v1/connect/direct` writes records and `POST
   * /v1/connect/verify` persists state, so a blind retry after an ambiguous
   * failure would do that twice.
   */
  idempotent?: boolean;
  /** Caller-supplied cancellation, combined with the per-request timeout. */
  signal?: AbortSignal;
  timeoutMs?: number;
}

const RETRY_BASE_MS = 250;
const RETRY_MAX_MS = 8_000;

function sleep(ms: number, signal?: AbortSignal): Promise<void> {
  return new Promise((resolve, reject) => {
    if (signal?.aborted) return reject(signal.reason);
    const timer = setTimeout(() => {
      signal?.removeEventListener("abort", onAbort);
      resolve();
    }, ms);
    const onAbort = () => {
      clearTimeout(timer);
      reject(signal?.reason);
    };
    signal?.addEventListener("abort", onAbort, { once: true });
  });
}

/**
 * Links an external signal to a timeout signal. `AbortSignal.any` would do this
 * in one line but only lands in Node 20, and this SDK supports Node 18.
 */
function withTimeout(timeoutMs: number, external?: AbortSignal) {
  const controller = new AbortController();
  let timedOut = false;

  const timer = setTimeout(() => {
    timedOut = true;
    controller.abort();
  }, timeoutMs);

  const onExternalAbort = () => controller.abort(external?.reason);
  if (external) {
    if (external.aborted) onExternalAbort();
    else external.addEventListener("abort", onExternalAbort, { once: true });
  }

  return {
    signal: controller.signal,
    didTimeOut: () => timedOut,
    dispose: () => {
      clearTimeout(timer);
      external?.removeEventListener("abort", onExternalAbort);
    },
  };
}

function parseRetryAfterMs(header: string | null): number | undefined {
  if (!header) return undefined;
  const seconds = Number(header);
  if (Number.isFinite(seconds)) return Math.max(0, seconds * 1000);
  const date = Date.parse(header);
  return Number.isNaN(date) ? undefined : Math.max(0, date - Date.now());
}

/** Full jitter, so a fleet of clients retrying together spreads out. */
function backoffMs(attempt: number): number {
  const ceiling = Math.min(RETRY_BASE_MS * 2 ** attempt, RETRY_MAX_MS);
  return Math.random() * ceiling;
}

export class Transport {
  // A真 private field: `private` alone is erased at runtime, which would let
  // JSON.stringify(client) or a logged error dump the API key.
  readonly #config: TransportConfig;

  constructor(config: TransportConfig) {
    this.#config = config;
  }

  private url(path: string, query?: RequestOptions["query"]): string {
    const url = new URL(path, this.#config.baseUrl);
    for (const [key, value] of Object.entries(query ?? {})) {
      if (value !== undefined && value !== "") url.searchParams.set(key, String(value));
    }
    return url.toString();
  }

  async request<T>(options: RequestOptions): Promise<T> {
    const { method, path, query, body, idempotent = method === "GET", signal } = options;
    const url = this.url(path, query);
    const timeoutMs = options.timeoutMs ?? this.#config.timeoutMs;
    const maxAttempts = idempotent ? this.#config.maxRetries + 1 : 1;

    let lastError: unknown;

    for (let attempt = 0; attempt < maxAttempts; attempt++) {
      if (attempt > 0) await sleep(backoffMs(attempt - 1), signal);

      const timeout = withTimeout(timeoutMs, signal);
      let response: Response;

      try {
        const headers: Record<string, string> = { Accept: "application/json" };
        if (this.#config.apiKey) headers.Authorization = `Bearer ${this.#config.apiKey}`;
        if (body !== undefined) headers["Content-Type"] = "application/json";

        response = await this.#config.fetch(url, {
          method,
          headers,
          body: body === undefined ? undefined : JSON.stringify(body),
          signal: timeout.signal,
        });
      } catch (cause) {
        if (timeout.didTimeOut()) {
          lastError = new VnytrosTimeoutError(
            `Request to ${method} ${path} timed out after ${timeoutMs}ms`,
            { code: "TIMEOUT", cause },
          );
        } else if (signal?.aborted) {
          throw cause;
        } else {
          lastError = new VnytrosConnectionError(`Could not reach the Vnytros API at ${url}`, {
            code: "CONNECTION_ERROR",
            cause,
          });
        }
        continue;
      } finally {
        timeout.dispose();
      }

      const requestId = response.headers.get("x-request-id") ?? undefined;

      if (response.ok) {
        if (response.status === 204) return undefined as T;
        const text = await response.text();
        if (text.trim() === "") return undefined as T;
        try {
          return JSON.parse(text) as T;
        } catch (cause) {
          throw new VnytrosConnectionError(
            `Vnytros API returned a non-JSON body for ${method} ${path}`,
            { statusCode: response.status, requestId, raw: text, cause },
          );
        }
      }

      const retryAfterMs = parseRetryAfterMs(response.headers.get("retry-after"));
      const payload = await response.text();
      let parsed: unknown = payload;
      try {
        parsed = payload === "" ? undefined : JSON.parse(payload);
      } catch {
        /* keep the raw text; some proxies return HTML error pages */
      }

      const error = errorFromResponse(response.status, parsed, requestId, retryAfterMs);
      const retryable =
        response.status === 408 || response.status === 429 || response.status >= 500;
      if (!retryable) throw error;

      lastError = error;

      // The API sets `Retry-After: 60` on 429. Honour it rather than hammering,
      // but do not block a caller for a minute if that outlasts their patience.
      if (error instanceof VnytrosRateLimitError && retryAfterMs !== undefined) {
        if (retryAfterMs > RETRY_MAX_MS) throw error;
        if (attempt < maxAttempts - 1) await sleep(retryAfterMs, signal);
      }
    }

    throw lastError;
  }
}
