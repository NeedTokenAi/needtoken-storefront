import assert from "node:assert/strict";
import test from "node:test";

import { containerRequest, proxyIfReady } from "./proxy.js";
import { waitUntilReady } from "./lifecycle.js";

function paymentBody() {
  let reads = 0;
  const raw = new Uint8Array([1, 2, 3, 4]);
  let streamReads = 0;
  const stream = new ReadableStream({
    pull(controller) {
      streamReads += 1;
      controller.enqueue(raw);
      controller.close();
    },
  });
  const request = new Request("https://shop.example/api/v1/payments/callback?provider=alipay", {
    method: "POST",
    headers: { "content-type": "application/x-www-form-urlencoded", host: "shop.example" },
    body: stream,
    duplex: "half",
  });
  const guarded = new Proxy(request, {
    get(target, prop) {
      if (prop === "text" || prop === "json" || prop === "arrayBuffer" || prop === "blob") {
        reads += 1;
      }
      // Node 24 Request getters use a private field. The receiver must stay
      // the real Request; the proxy only counts body readers.
      const value = Reflect.get(target, prop, target);
      return typeof value === "function" ? value.bind(target) : value;
    },
  });
  return { request: guarded, reads: () => reads, streamReads: () => streamReads };
}

test("a closed port is retried with scheduler.wait until it is ready", async () => {
  const waits = [];
  let probes = 0;
  const result = await waitUntilReady(
    async () => {
      probes += 1;
      if (probes < 3) {
        throw Object.assign(new Error("connect ECONNREFUSED"), { code: "ECONNREFUSED" });
      }
      return true;
    },
    {
      attempts: 5,
      delayMs: 200,
      wait: async (ms) => {
        waits.push(ms);
      },
    },
  );
  assert.equal(result.ready, true);
  assert.equal(result.probes, 3);
  assert.deepEqual(waits, [200, 200]);
  await assert.rejects(
    () => waitUntilReady(async () => true, { attempts: 1, wait: undefined }),
    /scheduler\.wait/,
  );
});

test("readiness does not read the payment body before the container fetch", async () => {
  const pending = paymentBody();
  let fetched = 0;
  const response = await proxyIfReady(pending.request, {
    attempts: 3,
    wait: async () => {
      assert.equal(pending.reads(), 0);
    },
    probe: async () => {
      assert.equal(pending.reads(), 0);
      return true;
    },
    fetch: async (forwarded) => {
      fetched += 1;
      assert.equal(pending.reads(), 0);
      assert.equal(new URL(forwarded.url).host, "container");
      assert.equal(forwarded.headers.get("host"), null);
      assert.equal(forwarded.method, "POST");
      const body = new Uint8Array(await forwarded.arrayBuffer());
      assert.deepEqual(body, new Uint8Array([1, 2, 3, 4]));
      return new Response("proxied", { status: 200 });
    },
  });
  assert.equal(response.status, 200);
  assert.equal(fetched, 1);
  assert.equal(pending.reads(), 0);
});

test("an unready container does not read the payment body", async () => {
  const pending = paymentBody();
  let fetched = 0;
  const response = await proxyIfReady(pending.request, {
    attempts: 2,
    wait: async () => {},
    probe: async () => {
      throw Object.assign(new Error("connect ECONNREFUSED"), { code: "ECONNREFUSED" });
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

test("forwarding keeps the websocket upgrade and the cancellation signal", () => {
  const controller = new AbortController();
  const request = new Request("https://shop.example/api/v1/chat", {
    headers: {
      host: "shop.example",
      upgrade: "websocket",
      connection: "Upgrade",
      "sec-websocket-key": "dGhlIHNhbXBsZSBub25jZQ==",
      "sec-websocket-version": "13",
    },
    signal: controller.signal,
  });
  const forwarded = containerRequest(request);
  assert.equal(new URL(forwarded.url).href, "http://container/api/v1/chat");
  assert.equal(forwarded.headers.get("upgrade"), "websocket");
  assert.equal(forwarded.headers.get("connection"), "Upgrade");
  assert.equal(forwarded.headers.get("sec-websocket-key"), "dGhlIHNhbXBsZSBub25jZQ==");
  assert.equal(forwarded.headers.get("host"), null);
  // new Request copies the signal. The official forward uses that constructor;
  // abort still reaches the forwarded request.
  controller.abort();
  assert.equal(forwarded.signal.aborted, true);
});

test("an aborted SSE client cancels the container request", async () => {
  const controller = new AbortController();
  const request = new Request("https://shop.example/api/v1/chat", {
    headers: { accept: "text/event-stream", host: "shop.example" },
    signal: controller.signal,
  });
  let sawAbort = false;
  const response = await proxyIfReady(request, {
    attempts: 1,
    wait: async () => {},
    probe: async () => true,
    fetch: async (forwarded) => {
      assert.equal(forwarded.headers.get("accept"), "text/event-stream");
      const body = new ReadableStream({
        start(streamController) {
          forwarded.signal.addEventListener("abort", () => {
            sawAbort = true;
            streamController.close();
          });
        },
      });
      return new Response(body, { headers: { "content-type": "text/event-stream" } });
    },
  });
  assert.equal(response.headers.get("content-type"), "text/event-stream");
  controller.abort();
  assert.equal(sawAbort, true);
  const text = await response.text();
  assert.equal(text, "");
});
