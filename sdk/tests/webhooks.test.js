import { test } from "node:test";
import assert from "node:assert/strict";
import { createHmac } from "node:crypto";
import { readFileSync } from "node:fs";
import { Vnytros, VnytrosSignatureVerificationError } from "../dist/index.js";

const fixture = JSON.parse(
  readFileSync(new URL("./fixtures/go-signature.json", import.meta.url), "utf8"),
);

const client = new Vnytros({ apiKey: "test-key", baseUrl: "http://localhost:8000", webhookSecret: fixture.secret });

function sign(secret, timestamp, body) {
  const digest = createHmac("sha256", secret).update(`${timestamp}.${body}`).digest("hex");
  return `t=${timestamp},v1=${digest}`;
}

function headers(signature, extra = {}) {
  return { "x-vnytros-signature": signature, "x-vnytros-event": "domain.connected", ...extra };
}

test("accepts a signature produced by the Go worker", () => {
  // The whole point: the Go and JS halves must agree byte for byte.
  const event = client.webhooks.constructEvent(
    fixture.payload,
    headers(fixture.header),
    { toleranceSec: 0 },
  );
  assert.equal(event.event, "domain.connected");
  assert.equal(event.data.domain, "shop.acme.com");
  assert.equal(event.data.user_id, 42);
});

test("accepts a fresh signature within the tolerance window", () => {
  const body = JSON.stringify({ event: "domain.error", data: { domain: "a.com" }, timestamp: "x" });
  const ts = Math.floor(Date.now() / 1000);
  const event = client.webhooks.constructEvent(body, headers(sign(fixture.secret, ts, body)));
  assert.equal(event.event, "domain.error");
});

test("rejects a tampered body", () => {
  const tampered = fixture.payload.replace("shop.acme.com", "evil.example");
  assert.throws(
    () => client.webhooks.constructEvent(tampered, headers(fixture.header), { toleranceSec: 0 }),
    VnytrosSignatureVerificationError,
  );
});

test("rejects the wrong secret", () => {
  assert.throws(
    () =>
      client.webhooks.constructEvent(fixture.payload, headers(fixture.header), {
        secret: "not-the-secret",
        toleranceSec: 0,
      }),
    VnytrosSignatureVerificationError,
  );
});

test("rejects a replayed delivery outside the tolerance window", () => {
  // Correctly signed, but old: exactly what a captured-and-resent request is.
  assert.throws(
    () => client.webhooks.constructEvent(fixture.payload, headers(fixture.header)),
    /Possible replay/,
  );
});

test("rejects a body re-signed under a fresh timestamp with a stale digest", () => {
  const ts = Math.floor(Date.now() / 1000);
  const forged = `t=${ts},v1=${fixture.header.split("v1=")[1]}`;
  assert.throws(
    () => client.webhooks.constructEvent(fixture.payload, headers(forged)),
    VnytrosSignatureVerificationError,
  );
});

test("rejects a missing signature header", () => {
  assert.throws(
    () => client.webhooks.constructEvent(fixture.payload, { "x-vnytros-event": "domain.connected" }),
    /Missing x-vnytros-signature/,
  );
});

test("reads headers from a Headers instance and a raw Buffer body", () => {
  const h = new Headers({ "x-vnytros-signature": fixture.header });
  const event = client.webhooks.constructEvent(Buffer.from(fixture.payload), h, {
    toleranceSec: 0,
  });
  assert.equal(event.data.domain, "shop.acme.com");
});

test("accepts a separate x-vnytros-timestamp header when the sig omits t=", () => {
  const digest = fixture.header.split("v1=")[1];
  const event = client.webhooks.constructEvent(
    fixture.payload,
    headers(`v1=${digest}`, { "x-vnytros-timestamp": String(fixture.timestamp) }),
    { toleranceSec: 0 },
  );
  assert.equal(event.event, "domain.connected");
});

test("supports two digests during a secret rotation", () => {
  const body = fixture.payload;
  const ts = Math.floor(Date.now() / 1000);
  const oldDigest = createHmac("sha256", "previous").update(`${ts}.${body}`).digest("hex");
  const newDigest = createHmac("sha256", fixture.secret).update(`${ts}.${body}`).digest("hex");
  const event = client.webhooks.constructEvent(
    body,
    headers(`t=${ts},v1=${oldDigest},v1=${newDigest}`),
  );
  assert.equal(event.event, "domain.connected");
});

test("explains itself when no secret is configured", () => {
  const bare = new Vnytros({ apiKey: "k", baseUrl: "http://localhost:8000" });
  assert.throws(
    () => bare.webhooks.constructEvent(fixture.payload, headers(fixture.header)),
    /No webhook signing secret configured/,
  );
});
