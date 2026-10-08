import { test } from "node:test";
import assert from "node:assert/strict";
import {
  Vnytros,
  VnytrosAuthError,
  VnytrosNotFoundError,
  VnytrosRateLimitError,
  VnytrosServerError,
  VnytrosTimeoutError,
  VnytrosValidationError,
} from "../dist/index.js";

/** Records every call and replays queued responses. */
function stubFetch(responses) {
  const calls = [];
  const queue = [...responses];
  const fn = async (url, init) => {
    calls.push({ url, init, body: init?.body ? JSON.parse(init.body) : undefined });
    const next = queue.length > 1 ? queue.shift() : queue[0];
    if (typeof next === "function") return next(url, init);
    const { status = 200, body = {}, headers = {} } = next;
    return new Response(typeof body === "string" ? body : JSON.stringify(body), {
      status,
      headers: { "content-type": "application/json", ...headers },
    });
  };
  fn.calls = calls;
  return fn;
}

function client(fetchImpl, options = {}) {
  return new Vnytros({
    apiKey: "sk_test_123",
    baseUrl: "https://api.example.test",
    fetch: fetchImpl,
    ...options,
  });
}

test("sends the bearer header and the documented status body", async () => {
  const fetchImpl = stubFetch([{ body: { domain: "shop.acme.com", is_configured: false, records: [] } }]);
  await client(fetchImpl).domains.status({ domain: "shop.acme.com", target: "cname.platform.test" });

  const [call] = fetchImpl.calls;
  assert.equal(call.url, "https://api.example.test/v1/status");
  assert.equal(call.init.method, "POST");
  assert.equal(call.init.headers.Authorization, "Bearer sk_test_123");
  // The API reads snake_case and treats absent fields as empty strings.
  assert.deepEqual(call.body, { domain: "shop.acme.com", ip: "", target: "cname.platform.test", txt: "" });
});

test("email tools hit their endpoints and join DKIM selectors", async () => {
  const fetchImpl = stubFetch([{ body: { tool: "x", findings: [] } }]);
  const vny = client(fetchImpl);
  await vny.tools.emailAuth("a.com");
  await vny.tools.spf("a.com");
  await vny.tools.dkim("a.com", ["google", "s1"]);
  await vny.tools.dmarc("a.com");
  assert.deepEqual(
    fetchImpl.calls.map((c) => c.url),
    [
      "https://api.example.test/v1/tools/email/auth?domain=a.com",
      "https://api.example.test/v1/tools/email/spf?domain=a.com",
      "https://api.example.test/v1/tools/email/dkim?domain=a.com&selector=google%2Cs1",
      "https://api.example.test/v1/tools/email/dmarc?domain=a.com",
    ],
  );
});

test("baseUrl without a trailing slash does not swallow the path", async () => {
  const fetchImpl = stubFetch([{ body: { domain: "a.com", nameservers: [], provider: null } }]);
  await client(fetchImpl, { baseUrl: "https://api.example.test/" }).detect("a.com");
  assert.equal(fetchImpl.calls[0].url, "https://api.example.test/v1/detect?domain=a.com");
});

test("maps lowercase error bodies onto typed errors", async () => {
  const cases = [
    [401, "UNAUTHORIZED_TOKEN", VnytrosAuthError],
    [404, "DOMAIN_NOT_FOUND", VnytrosNotFoundError],
    [400, "BAD_REQUEST", VnytrosValidationError],
  ];
  for (const [status, code, Expected] of cases) {
    const fetchImpl = stubFetch([
      { status, body: { code, message: `boom ${code}`, status_code: status } },
    ]);
    await assert.rejects(
      () => client(fetchImpl).domains.list(),
      (err) => {
        assert.ok(err instanceof Expected, `${status} should be ${Expected.name}`);
        assert.equal(err.code, code);
        assert.equal(err.statusCode, status);
        assert.equal(err.message, `boom ${code}`);
        return true;
      },
    );
  }
});

test("still reads the legacy capitalised error keys", async () => {
  // The API sends both shapes today; the SDK must survive either being dropped.
  const fetchImpl = stubFetch([
    { status: 403, body: { Code: "EMAIL_UNVERIFIED", Message: "verify first", StatusCode: 403 } },
  ]);
  await assert.rejects(() => client(fetchImpl).domains.list(), (err) => {
    assert.ok(err instanceof VnytrosAuthError);
    assert.equal(err.code, "EMAIL_UNVERIFIED");
    assert.equal(err.message, "verify first");
    return true;
  });
});

test("survives an HTML error page from a proxy", async () => {
  const fetchImpl = stubFetch([
    { status: 502, body: "<html>bad gateway</html>", headers: { "content-type": "text/html" } },
  ]);
  await assert.rejects(
    () => client(fetchImpl, { maxRetries: 0 }).domains.list(),
    (err) => {
      assert.ok(err instanceof VnytrosServerError);
      assert.equal(err.raw, "<html>bad gateway</html>");
      return true;
    },
  );
});

test("retries an idempotent GET and returns the eventual success", async () => {
  const fetchImpl = stubFetch([
    { status: 500, body: { code: "INTERNAL_SERVER_ERROR", message: "nope" } },
    { body: [{ domain_name: "a.com" }] },
  ]);
  const domains = await client(fetchImpl).domains.list();
  assert.equal(domains.length, 1);
  assert.equal(fetchImpl.calls.length, 2);
});

test("never retries verify(), which persists state and can fire a webhook", async () => {
  const fetchImpl = stubFetch([{ status: 500, body: { code: "INTERNAL_SERVER_ERROR", message: "nope" } }]);
  await assert.rejects(() => client(fetchImpl).domains.verify("a.com"), VnytrosServerError);
  assert.equal(fetchImpl.calls.length, 1, "verify must be attempted exactly once");
});

test("status() is retried, because reading DNS has no side effects", async () => {
  const fetchImpl = stubFetch([
    { status: 503, body: { code: "INTERNAL_SERVER_ERROR", message: "nope" } },
    { body: { domain: "a.com", is_configured: true, records: [] } },
  ]);
  const status = await client(fetchImpl).domains.status({ domain: "a.com" });
  assert.equal(status.is_configured, true);
  assert.equal(fetchImpl.calls.length, 2);
});

test("gives up immediately when Retry-After exceeds the backoff ceiling", async () => {
  // The API sends Retry-After: 60. Blocking a request for a minute is worse
  // than handing the caller a typed error they can schedule around.
  const fetchImpl = stubFetch([
    { status: 429, body: { code: "RATE_LIMITED", message: "slow down" }, headers: { "retry-after": "60" } },
  ]);
  await assert.rejects(() => client(fetchImpl).tools.nameservers("a.com"), (err) => {
    assert.ok(err instanceof VnytrosRateLimitError);
    assert.equal(err.retryAfterMs, 60_000);
    return true;
  });
  assert.equal(fetchImpl.calls.length, 1);
});

test("times out a hanging request and reports the budget", async () => {
  const hang = (_url, init) =>
    new Promise((_resolve, reject) => {
      init.signal.addEventListener("abort", () => reject(new Error("aborted")), { once: true });
    });
  await assert.rejects(
    () => client(stubFetch([hang]), { timeoutMs: 40, maxRetries: 0 }).domains.list(),
    (err) => {
      assert.ok(err instanceof VnytrosTimeoutError);
      assert.match(err.message, /timed out after 40ms/);
      return true;
    },
  );
});

test("a caller's abort signal propagates and is not swallowed as a timeout", async () => {
  const controller = new AbortController();
  const hang = (_url, init) =>
    new Promise((_resolve, reject) => {
      init.signal.addEventListener("abort", () => reject(new Error("aborted")), { once: true });
    });
  const promise = client(stubFetch([hang])).domains.list(controller.signal);
  controller.abort();
  await assert.rejects(() => promise, /aborted/);
});

test("waitUntilConnected polls until the records resolve", async () => {
  let n = 0;
  const fetchImpl = stubFetch([
    () =>
      new Response(
        JSON.stringify({ domain: "a.com", is_configured: ++n === 3, records: [] }),
        { status: 200, headers: { "content-type": "application/json" } },
      ),
  ]);
  const seen = [];
  const result = await client(fetchImpl).domains.waitUntilConnected(
    { domain: "a.com", txt: "vny-1" },
    { intervalMs: 1, timeoutMs: 5_000, onPoll: (s, attempt) => seen.push(attempt) },
  );
  assert.equal(result.is_configured, true);
  assert.deepEqual(seen, [1, 2, 3]);
});

test("waitUntilConnected reports a timeout rather than a false negative", async () => {
  const fetchImpl = stubFetch([{ body: { domain: "a.com", is_configured: false, records: [] } }]);
  await assert.rejects(
    () =>
      client(fetchImpl).domains.waitUntilConnected(
        { domain: "a.com" },
        { intervalMs: 1, timeoutMs: 30 },
      ),
    (err) => {
      assert.ok(err instanceof VnytrosTimeoutError);
      assert.match(err.message, /still not configured/);
      return true;
    },
  );
});

test("refuses to construct without an apiKey", () => {
  assert.throws(() => new Vnytros({}), /apiKey is required/);
});

test("refuses to construct without a baseUrl", () => {
  assert.throws(() => new Vnytros({ apiKey: "k" }), /baseUrl is required/);
  assert.throws(() => new Vnytros({ apiKey: "k", baseUrl: "" }), /baseUrl is required/);
});

test("rejects a baseUrl that is not an absolute http(s) URL", () => {
  assert.throws(() => new Vnytros({ apiKey: "k", baseUrl: "api.example.test" }), /Invalid Vnytros baseUrl/);
  assert.throws(() => new Vnytros({ apiKey: "k", baseUrl: "ftp://api.example.test" }), /must start with http/);
});

test("refuses to construct in a browser, where the key would leak", () => {
  globalThis.window = {};
  try {
    assert.throws(() => client(stubFetch([{}])), /would expose your secret API key/);
    // ...unless the caller has explicitly accepted the risk.
    assert.ok(client(stubFetch([{}]), { dangerouslyAllowBrowser: true }));
  } finally {
    delete globalThis.window;
  }
});
