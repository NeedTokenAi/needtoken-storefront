import assert from "node:assert/strict";
import test from "node:test";

import { classifyRequest, drain, handleTraffic, singletonName, watchdog } from "./lifecycle.js";
import { SINGLETON_NAME } from "./policy.js";

test("http percent 0 keeps a running worker and does not signal it", async () => {
  let scheduled = 0;
  const result = await watchdog({
    httpPercent: 0,
    container: {
      running: true,
      signal() {
        throw new Error("0% HTTP must not stop the worker");
      },
      start() {
        throw new Error("running worker must not be restarted");
      },
    },
    scheduleNext: async () => {
      scheduled += 1;
    },
    probe: async () => true,
    plan() {
      throw new Error("running worker does not need a start plan");
    },
  });
  assert.equal(result.action, "kept");
  assert.equal(result.signaled, false);
  assert.equal(result.httpPercent, 0);
  assert.equal(scheduled, 1);
  assert.equal(classifyRequest("/api/v1/orders", 0), "hold");
});

test("http percent 0 still starts a stopped worker", async () => {
  let started = 0;
  const result = await watchdog({
    httpPercent: 0,
    container: {
      running: false,
      start(options) {
        started += 1;
        assert.deepEqual(options.entrypoint, ["./dujiao-next", "-mode", "all"]);
      },
      signal() {
        throw new Error("starting at 0% must not signal");
      },
    },
    scheduleNext: async () => {},
    plan: async () => ({ ok: true, options: { entrypoint: ["./dujiao-next", "-mode", "all"] } }),
    setInactivityTimeout: async () => {},
  });
  assert.equal(result.action, "started");
  assert.equal(result.signaled, false);
  assert.equal(started, 1);
});

test("a failed probe drains with SIGTERM before the fence changes", async () => {
  const events = [];
  const signals = [];
  let monitored = false;
  const fence = await drain({
    events,
    container: {
      signal(number) {
        signals.push(number);
        assert.equal(monitored, false);
      },
      async monitor() {
        assert.deepEqual(signals, [15]);
        monitored = true;
      },
    },
    bumpFence: async () => {
      assert.equal(monitored, true);
      return "fence-2";
    },
  });
  assert.equal(fence, "fence-2");
  assert.deepEqual(events, ["signal", "monitor", "fence"]);
  assert.deepEqual(signals, [15]);
});

test("startup grace does not treat an unread health check as a dead worker", async () => {
  const result = await watchdog({
    httpPercent: 0,
    startedAt: 1_000,
    now: () => 1_500,
    graceMs: 60_000,
    container: {
      running: true,
      signal() {
        throw new Error("grace must not signal");
      },
    },
    scheduleNext: async () => {},
    probe: async () => false,
    plan() {
      throw new Error("grace must not restart");
    },
  });
  assert.equal(result.action, "starting");
  assert.equal(result.signaled, false);
});

test("traffic control fails closed without a token and only accepts 0 or 100", () => {
  assert.equal(handleTraffic({ percent: 100 }, "Bearer secret", "").status, 403);
  assert.equal(handleTraffic({ percent: 100 }, "Bearer no", "secret").status, 401);
  assert.equal(handleTraffic({ percent: 50 }, "Bearer secret", "secret").status, 400);
  assert.deepEqual(handleTraffic({ percent: 100 }, "Bearer secret", "secret"), { status: 200, percent: 100 });
  assert.deepEqual(handleTraffic({ percent: 0 }, "Bearer secret", "secret"), { status: 200, percent: 0 });
});

test("every user request addresses the same durable object name", () => {
  assert.equal(singletonName("customer-a"), SINGLETON_NAME);
  assert.equal(singletonName("customer-b"), SINGLETON_NAME);
  assert.equal(classifyRequest("/__cloudflare/traffic", 0), "control");
  assert.equal(classifyRequest("/health", 100), "proxy");
});
