import { test } from "node:test";
import assert from "node:assert/strict";
import { Vnytros, VnytrosValidationError } from "../dist/index.js";

function stubFetch(capture = []) {
  const fn = async (url, init) => {
    capture.push({ url, body: JSON.parse(init.body) });
    return new Response(JSON.stringify({ message: "ok", domain: "a.com" }), {
      status: 200,
      headers: { "content-type": "application/json" },
    });
  };
  fn.calls = capture;
  return fn;
}

const client = (fetchImpl) =>
  new Vnytros({ apiKey: "k", baseUrl: "https://api.example.test", fetch: fetchImpl });

test("sends the exact provider_config keys the Go structs bind to", async () => {
  const fetchImpl = stubFetch();
  await client(fetchImpl).domains.connectDirect({
    domain: "a.com",
    provider: "aws",
    ip: "203.0.113.10",
    providerConfig: { accessKey: "AKIA", secretKey: "shh" },
  });

  const [call] = fetchImpl.calls;
  assert.equal(call.url, "https://api.example.test/v1/connect/direct");
  assert.equal(call.body.provider, "aws");
  // Go matches JSON keys case-insensitively but not across underscores.
  assert.deepEqual(call.body.provider_config, { accessKey: "AKIA", secretKey: "shh" });
});

test("rejects route53, name.com and namecom, which the factory does not support", async () => {
  for (const provider of ["route53", "name.com", "namecom"]) {
    await assert.rejects(
      () => client(stubFetch()).domains.connectDirect({ domain: "a.com", provider, providerConfig: {} }),
      (err) => {
        assert.ok(err instanceof VnytrosValidationError);
        assert.match(err.message, /Supported: aws, hostinger\./);
        return true;
      },
    );
  }
});

test("catches snake_case config keys before the server swallows them", async () => {
  const fetchImpl = stubFetch();
  await assert.rejects(
    () =>
      client(fetchImpl).domains.connectDirect({
        domain: "a.com",
        provider: "aws",
        providerConfig: { access_key: "AKIA", secret_key: "shh" },
      }),
    (err) => {
      assert.match(err.message, /missing "accessKey", "secretKey"/);
      assert.match(err.message, /not across underscores/);
      return true;
    },
  );
  assert.equal(fetchImpl.calls.length, 0, "must not spend a request on a known-bad config");
});

test("names the missing credential per provider", async () => {
  await assert.rejects(
    () =>
      client(stubFetch()).domains.connectDirect({
        domain: "a.com",
        provider: "hostinger",
        providerConfig: {},
      }),
    /missing "apiToken"/,
  );
});

test("accepts hostinger with correct credentials", async () => {
  const fetchImpl = stubFetch();
  const vny = client(fetchImpl);
  await vny.domains.connectDirect({
    domain: "b.com",
    provider: "hostinger",
    providerConfig: { apiToken: "t" },
  });
  assert.equal(fetchImpl.calls.length, 1);
});

test("connectDirect is never retried — the records are already written", async () => {
  let attempts = 0;
  const failing = async () => {
    attempts++;
    return new Response(JSON.stringify({ code: "INTERNAL_SERVER_ERROR", message: "boom" }), {
      status: 500,
      headers: { "content-type": "application/json" },
    });
  };
  await assert.rejects(() =>
    client(failing).domains.connectDirect({
      domain: "a.com",
      provider: "aws",
      providerConfig: { accessKey: "AKIA", secretKey: "s" },
    }),
  );
  assert.equal(attempts, 1);
});
