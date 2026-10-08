/**
 * Error codes the API returns in the `code` field. The union is open (`string &
 * {}`) so a new server-side code does not break type-checking for consumers.
 */
export type VnytrosErrorCode =
  | "INTERNAL_SERVER_ERROR"
  | "INVALID_INPUT"
  | "UNAUTHORIZED"
  | "FORBIDDEN"
  | "NOT_FOUND"
  | "CONFLICT"
  | "BAD_REQUEST"
  | "DOMAIN_NOT_FOUND"
  | "INVALID_TOKEN"
  | "TOKEN_EXPIRED"
  | "UNAUTHORIZED_TOKEN"
  | "INVALID_API_KEY"
  | "EMAIL_UNVERIFIED"
  | "RATE_LIMITED"
  | "NOT_IMPLEMENTED"
  // Client-side codes, never sent by the server.
  | "CONNECTION_ERROR"
  | "TIMEOUT"
  | "SIGNATURE_VERIFICATION_FAILED"
  | (string & {});

interface VnytrosErrorOptions {
  code?: VnytrosErrorCode;
  statusCode?: number;
  requestId?: string;
  raw?: unknown;
  cause?: unknown;
}

/** Base class for every error this SDK throws. */
export class VnytrosError extends Error {
  readonly code: VnytrosErrorCode;
  readonly statusCode?: number;
  /** Value of the `X-Request-Id` response header, when the API sends one. */
  readonly requestId?: string;
  /** The parsed response body, for codes this SDK does not model yet. */
  readonly raw?: unknown;

  constructor(message: string, options: VnytrosErrorOptions = {}) {
    super(message, options.cause !== undefined ? { cause: options.cause } : undefined);
    this.name = new.target.name;
    this.code = options.code ?? "INTERNAL_SERVER_ERROR";
    this.statusCode = options.statusCode;
    this.requestId = options.requestId;
    this.raw = options.raw;
  }
}

/** 401/403: the API key is missing, malformed, or not permitted. */
export class VnytrosAuthError extends VnytrosError {}
/** 400/409/422: the request was rejected before any work was done. */
export class VnytrosValidationError extends VnytrosError {}
/** 404: the domain or resource does not exist. */
export class VnytrosNotFoundError extends VnytrosError {}
/** 5xx: the API failed. Safe to retry idempotent calls. */
export class VnytrosServerError extends VnytrosError {}
/** The request never reached the API (DNS failure, connection reset, offline). */
export class VnytrosConnectionError extends VnytrosError {}
/** The request was aborted because it exceeded `timeoutMs`. */
export class VnytrosTimeoutError extends VnytrosError {}
/** A webhook payload's signature did not verify. Treat the delivery as hostile. */
export class VnytrosSignatureVerificationError extends VnytrosError {}

/** 429: rate limited. `retryAfterMs` mirrors the `Retry-After` header. */
export class VnytrosRateLimitError extends VnytrosError {
  readonly retryAfterMs?: number;
  constructor(message: string, options: VnytrosErrorOptions & { retryAfterMs?: number } = {}) {
    super(message, options);
    this.retryAfterMs = options.retryAfterMs;
  }
}

/**
 * The API sends each error field twice: lowercase (`code`) for this SDK, and
 * capitalised (`Code`) for the older dashboard client. Read lowercase first and
 * fall back, so the SDK keeps working whichever half is eventually removed.
 */
function readErrorBody(body: unknown): { code?: string; message?: string } {
  if (typeof body !== "object" || body === null) return {};
  const b = body as Record<string, unknown>;
  const code = typeof b.code === "string" ? b.code : typeof b.Code === "string" ? b.Code : undefined;
  const message =
    typeof b.message === "string" ? b.message : typeof b.Message === "string" ? b.Message : undefined;
  return { code, message };
}

/** Maps an HTTP error response onto the most specific error class available. */
export function errorFromResponse(
  status: number,
  body: unknown,
  requestId?: string,
  retryAfterMs?: number,
): VnytrosError {
  const { code, message } = readErrorBody(body);
  const options: VnytrosErrorOptions = {
    code: code ?? `HTTP_${status}`,
    statusCode: status,
    requestId,
    raw: body,
  };
  const text = message ?? `Vnytros API responded with HTTP ${status}`;

  if (status === 429) return new VnytrosRateLimitError(text, { ...options, retryAfterMs });
  if (status === 401 || status === 403) return new VnytrosAuthError(text, options);
  if (status === 404) return new VnytrosNotFoundError(text, options);
  if (status >= 500) return new VnytrosServerError(text, options);
  if (status >= 400) return new VnytrosValidationError(text, options);
  return new VnytrosError(text, options);
}
