import assert from "node:assert/strict";
import test from "node:test";

import { proxyIfReady } from "./proxy.js";

function guardedRequest() {
  let reads = 0;
  const raw = new Uint8Array([7, 7, 7]);
  const request = new Proxy(
    {
      url: "https://shop.example/api/v1/payments/callback",
      method: "POST",
    },
    {
      get(target, prop, receiver) {
        if (prop === "body" || prop === "arrayBuffer" || prop === "text" || prop === "json") {
          reads += 1;
          if (prop === "body") {
            return raw;
          }
        }
        const value = Reflect.get(target, prop, receiver);
        return typeof value === "function" ? value.bind(target) : value;
      },
    },
  );
  return { request, reads: () => reads };
}

test("readiness finishes before the user body is proxied", async () => {
  const pending = guardedRequest();
  let probes = 0;
  let fetched = 0;
  const response = await proxyIfReady(pending.request, {
    attempts: 3,
    sleep: async () => {
      assert.equal(pending.reads(), 0);
    },
    probe: async () => {
      probes += 1;
      assert.equal(pending.reads(), 0);
      return probes >= 2;
    },
    fetch: async (incoming) => {
      fetched += 1;
      assert.equal(pending.reads(), 0);
      assert.equal(incoming, pending.request);
      assert.equal(incoming.body.length, 3);
      return new Response("proxied", { status: 200 });
    },
  });
  assert.equal(response.status, 200);
  assert.equal(probes, 2);
  assert.equal(fetched, 1);
  assert.equal(pending.reads(), 1);
});

test("an unready container does not read the user body", async () => {
  const pending = guardedRequest();
  let fetched = 0;
  const response = await proxyIfReady(pending.request, {
    attempts: 2,
    sleep: async () => {
      assert.equal(pending.reads(), 0);
    },
    probe: async () => {
      assert.equal(pending.reads(), 0);
      return false;
    },
    fetch: async () => {
      fetched += 1;
      return new Response("nope");
    },
  });
  assert.equal(response.status, 503);
  assert.equal(fetched, 0);
  assert.equal(pending.reads(), 0);
});
