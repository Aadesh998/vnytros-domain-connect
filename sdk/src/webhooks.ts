import { createHmac, timingSafeEqual } from "node:crypto";
import { VnytrosSignatureVerificationError } from "./errors.js";
import type { VnytrosEvent } from "./types.js";

export const SIGNATURE_HEADER = "x-vnytros-signature";
export const TIMESTAMP_HEADER = "x-vnytros-timestamp";
export const EVENT_HEADER = "x-vnytros-event";

/** Anything with a case-insensitive header lookup: `Headers`, or a plain object. */
export type HeaderSource =
  | Headers
  | Record<string, string | string[] | undefined>
  | { get(name: string): string | null };

function readHeader(source: HeaderSource, name: string): string | undefined {
  if (typeof (source as Headers).get === "function") {
    return (source as Headers).get(name) ?? undefined;
  }
  const bag = source as Record<string, string | string[] | undefined>;
  const key = Object.keys(bag).find((k) => k.toLowerCase() === name);
  const value = key === undefined ? undefined : bag[key];
  return Array.isArray(value) ? value[0] : value;
}

/** Parses `t=<unix>,v1=<hex>`. */
function parseSignatureHeader(header: string): { timestamp?: number; v1: string[] } {
  let timestamp: number | undefined;
  const v1: string[] = [];

  for (const part of header.split(",")) {
    const index = part.indexOf("=");
    if (index === -1) continue;
    const key = part.slice(0, index).trim();
    const value = part.slice(index + 1).trim();
    if (key === "t") {
      const parsed = Number(value);
      if (Number.isFinite(parsed)) timestamp = parsed;
    } else if (key === "v1") {
      v1.push(value);
    }
  }
  return { timestamp, v1 };
}

function constantTimeEquals(a: string, b: string): boolean {
  const left = Buffer.from(a, "hex");
  const right = Buffer.from(b, "hex");
  // timingSafeEqual throws on a length mismatch, which would itself leak.
  if (left.length === 0 || left.length !== right.length) return false;
  return timingSafeEqual(left, right);
}

export interface ConstructEventOptions {
  /** Overrides the client-level secret. */
  secret?: string;
  /**
   * How much clock skew to tolerate, in seconds. Default 300. Set to 0 to skip
   * the timestamp check entirely (not recommended — it re-opens replays).
   */
  toleranceSec?: number;
}

export class Webhooks {
  readonly #defaultSecret?: string;

  constructor(defaultSecret?: string) {
    this.#defaultSecret = defaultSecret;
  }

  /**
   * Verifies a webhook delivery and returns the parsed event.
   *
   * Pass the **raw request body**, not a re-serialised object: the signature
   * covers exact bytes, and `JSON.stringify(JSON.parse(body))` can reorder keys
   * or change number formatting, which fails verification for the wrong reason.
   * In Express that means `express.raw({ type: "application/json" })`; in Next
   * route handlers, `await req.text()`.
   *
   * @throws {VnytrosSignatureVerificationError} if the signature, timestamp, or
   * body does not check out. Return 400 and do not process the payload.
   */
  constructEvent(
    payload: string | Buffer | Uint8Array,
    headers: HeaderSource,
    options: ConstructEventOptions = {},
  ): VnytrosEvent {
    const secret = options.secret ?? this.#defaultSecret;
    if (!secret) {
      throw new VnytrosSignatureVerificationError(
        "No webhook signing secret configured. Pass `webhookSecret` to the Vnytros constructor or `secret` here.",
        { code: "SIGNATURE_VERIFICATION_FAILED" },
      );
    }

    const header = readHeader(headers, SIGNATURE_HEADER);
    if (!header) {
      throw new VnytrosSignatureVerificationError(
        `Missing ${SIGNATURE_HEADER} header. If this account predates webhook signing, set WEBHOOK_SIGNING_SECRET on the API and redeploy the worker.`,
        { code: "SIGNATURE_VERIFICATION_FAILED" },
      );
    }

    const body = typeof payload === "string" ? payload : Buffer.from(payload).toString("utf8");
    const { timestamp, v1 } = parseSignatureHeader(header);
    const effectiveTimestamp = timestamp ?? Number(readHeader(headers, TIMESTAMP_HEADER));

    if (!Number.isFinite(effectiveTimestamp)) {
      throw new VnytrosSignatureVerificationError("Webhook signature carried no usable timestamp.", {
        code: "SIGNATURE_VERIFICATION_FAILED",
      });
    }
    if (v1.length === 0) {
      throw new VnytrosSignatureVerificationError("Webhook signature carried no v1 digest.", {
        code: "SIGNATURE_VERIFICATION_FAILED",
      });
    }

    const tolerance = options.toleranceSec ?? 300;
    if (tolerance > 0) {
      const ageSec = Math.abs(Date.now() / 1000 - effectiveTimestamp);
      if (ageSec > tolerance) {
        throw new VnytrosSignatureVerificationError(
          `Webhook timestamp is ${Math.round(ageSec)}s away from now, outside the ${tolerance}s tolerance. Possible replay.`,
          { code: "SIGNATURE_VERIFICATION_FAILED" },
        );
      }
    }

    const expected = createHmac("sha256", secret)
      .update(`${effectiveTimestamp}.${body}`)
      .digest("hex");

    // Any digest matching is enough, so a secret rotation can send two.
    if (!v1.some((candidate) => constantTimeEquals(expected, candidate))) {
      throw new VnytrosSignatureVerificationError(
        "Webhook signature did not match. Check the secret matches WEBHOOK_SIGNING_SECRET, and that you passed the raw unparsed body.",
        { code: "SIGNATURE_VERIFICATION_FAILED" },
      );
    }

    try {
      return JSON.parse(body) as VnytrosEvent;
    } catch (cause) {
      throw new VnytrosSignatureVerificationError(
        "Webhook body verified but was not valid JSON.",
        { code: "SIGNATURE_VERIFICATION_FAILED", cause },
      );
    }
  }
}
