import assert from "node:assert/strict";
import test from "node:test";

import { classifyRequest, drain, handleTraffic, observeExit, prepareDurableObject, recoveryDecision, schedulerWait, singletonName, watchdog } from "./lifecycle.js";
import { IMAGE_NAME, LABELS, PREVIOUS_IMAGE_NAME, SINGLETON_NAME } from "./policy.js";

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

test("ECONNREFUSED then ready, start failure, non-zero exit, rebuild, and zero traffic", async () => {
  assert.equal(LABELS.mock, true);
  assert.equal(LABELS.notOnARealAccount, true);
  assert.equal(LABELS.notASixHourRun, true);
  assert.equal(LABELS.durableObjectBeta, true);
  assert.equal(LABELS.rollback, false);
  assert.equal(LABELS.stopsOldCloudRunWriter, false);

  const alarms = [];
  const storage = {
    alarm: 0,
    async getAlarm() {
      return this.alarm || null;
    },
    async setAlarm(at) {
      this.alarm = at;
      alarms.push(at);
    },
  };
  const armed = await prepareDurableObject({
    storage,
    now: () => 1_000,
    intervalMs: 60_000,
    container: { running: false },
  });
  assert.equal(armed.action, "armed");
  assert.deepEqual(alarms, [61_000]);

  let started = 0;
  const failed = await watchdog({
    httpPercent: 0,
    startedFlag: true,
    container: {
      running: false,
      start() {
        started += 1;
        throw new Error("start failed");
      },
      signal() {
        throw new Error("start failure must not signal");
      },
    },
    scheduleNext: async () => {},
    plan: async () => ({ ok: true, options: { entrypoint: ["./dujiao-next", "-mode", "all"] } }),
  });
  assert.equal(failed.action, "start-failed");
  assert.equal(failed.rollback, false);
  assert.equal(started, 1);

  const images = {
    [IMAGE_NAME]: "sha256:current",
    [PREVIOUS_IMAGE_NAME]: "sha256:previous",
  };
  let startedImage;
  const recovered = await watchdog({
    httpPercent: 0,
    startedFlag: true,
    exitCode: 1,
    images,
    container: {
      running: false,
      start(options) {
        startedImage = options.image;
      },
      signal() {
        throw new Error("recovery start must not signal first");
      },
    },
    scheduleNext: async () => {},
    plan: async () => ({ ok: true, options: {} }),
  });
  assert.equal(recovered.action, "started");
  assert.equal(recovered.role, "previous");
  assert.equal(recovered.rollback, false);
  assert.equal(startedImage, "sha256:previous");
  assert.equal(recoveryDecision({ containerRunning: false, exitCode: 1, images }).rollback, false);

  const events = [];
  await drain({
    events,
    container: {
      signal() {},
      async monitor() {
        throw Object.assign(new Error("exit 1"), { exitCode: 1 });
      },
    },
    bumpFence: async () => "fence-after-exit",
  });
  assert.deepEqual(events, ["signal", "monitor-refused", "fence"]);

  let reattached = 0;
  const rebuilt = await prepareDurableObject({
    storage,
    now: () => 2_000,
    intervalMs: 60_000,
    container: { running: true },
    setInactivityTimeout: async () => {},
    observeExit() {
      reattached += 1;
    },
  });
  assert.equal(rebuilt.action, "reattach");
  assert.equal(reattached, 1);
  assert.equal(alarms.length, 1);

  let exit;
  await observeExit(
    {
      monitor() {
        return Promise.reject(Object.assign(new Error("exit 1"), { exitCode: 1 }));
      },
    },
    (result) => {
      exit = result;
    },
  );
  assert.equal(exit.code, 1);

  const fakeScheduler = { waits: [], async wait(ms) { this.waits.push(ms); } };
  await schedulerWait(200, fakeScheduler);
  assert.deepEqual(fakeScheduler.waits, [200]);
  await assert.rejects(() => schedulerWait(1, {}), /scheduler\.wait/);
});

test("every user request addresses the same durable object name", () => {
  assert.equal(singletonName("customer-a"), SINGLETON_NAME);
  assert.equal(singletonName("customer-b"), SINGLETON_NAME);
  assert.equal(classifyRequest("/__cloudflare/traffic", 0), "control");
  assert.equal(classifyRequest("/health", 100), "proxy");
});
