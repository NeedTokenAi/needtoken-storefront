import { DurableObject } from "cloudflare:workers";

import {
  classifyRequest,
  handleTraffic,
  observeExit,
  prepareDurableObject,
  recoveryDecision,
  schedulerWait,
  TRAFFIC_PATH,
  watchdog,
} from "./lifecycle.js";
import { CONTAINER_PORT, IMAGE_NAME, INACTIVITY_TIMEOUT_MS, PREVIOUS_IMAGE_NAME, SINGLETON_NAME, startPlan } from "./policy.js";
import { containerRequest, proxyIfReady } from "./proxy.js";

const ALARM_MS = 60 * 1000;
const START_GRACE_MS = 60 * 1000;

// Mock skeleton. Not on a real account. Not a 6 hour run. durable_object is
// public beta. rollback is false. The Redis lease does not stop Cloud Run.
export class DujiaoStorefront extends DurableObject {
  constructor(ctx, env) {
    super(ctx, env);
    this.ctx = ctx;
    this.env = env;
    this.exitCode = 0;
    void ctx.blockConcurrencyWhile(() => this.prepare());
  }

  async prepare() {
    await prepareDurableObject({
      storage: this.ctx.storage,
      now: () => Date.now(),
      intervalMs: ALARM_MS,
      container: this.ctx.container,
      setInactivityTimeout: () => this.ctx.container.setInactivityTimeout(INACTIVITY_TIMEOUT_MS),
      observeExit: () => this.observeExit(),
    });
  }

  observeExit() {
    if (this.monitoring || !this.ctx.container) {
      return;
    }
    this.monitoring = observeExit(this.ctx.container, (result) => {
      this.exitCode = result.code;
      void this.ctx.storage.put("lastExitCode", result.code);
    }).finally(() => {
      this.monitoring = undefined;
    });
    this.ctx.waitUntil?.(this.monitoring);
  }

  async alarm() {
    const percent = (await this.ctx.storage.get("httpPercent")) ?? 0;
    const startedAt = (await this.ctx.storage.get("startedAt")) ?? 0;
    const exitCode = (await this.ctx.storage.get("lastExitCode")) ?? this.exitCode ?? 0;
    await watchdog({
      httpPercent: percent,
      startedAt,
      exitCode,
      startedFlag: true,
      graceMs: START_GRACE_MS,
      now: () => Date.now(),
      container: this.ctx.container,
      images: this.imageMap(),
      scheduleNext: () => this.ctx.storage.setAlarm(Date.now() + ALARM_MS),
      probe: () => this.probeHealth(),
      plan: (decision) => this.containerPlan(decision),
      setInactivityTimeout: () => this.ctx.container.setInactivityTimeout(INACTIVITY_TIMEOUT_MS),
      bumpFence: () => this.bumpFence(),
    });
    this.observeExit();
  }

  async fetch(request) {
    const url = new URL(request.url);
    if (url.pathname === TRAFFIC_PATH) {
      return this.traffic(request);
    }
    const percent = (await this.ctx.storage.get("httpPercent")) ?? 0;
    const action = classifyRequest(url.pathname, percent);
    await this.ensureRunning();
    if (action !== "proxy") {
      return new Response("storefront traffic is held at 0%", { status: 503 });
    }
    return proxyIfReady(request, {
      attempts: 30,
      delayMs: 200,
      wait: (ms) => schedulerWait(ms),
      probe: () => this.probeHealth(),
      fetch: (forwarded) => this.ctx.container.getTcpPort(CONTAINER_PORT).fetch(forwarded),
    });
  }

  async traffic(request) {
    let body = null;
    try {
      body = await request.json();
    } catch {
      body = null;
    }
    const decision = handleTraffic(body, request.headers.get("authorization") ?? "", this.env.CLOUDFLARE_CONTROL_TOKEN);
    if (decision.status === 200) {
      await this.ctx.storage.put("httpPercent", decision.percent);
    }
    return Response.json(decision, { status: decision.status });
  }

  async ensureRunning() {
    const container = this.ctx.container;
    if (!container) {
      throw new Error("container binding is missing");
    }
    if (container.running) {
      return;
    }
    const exitCode = (await this.ctx.storage.get("lastExitCode")) ?? this.exitCode ?? 0;
    const decision = recoveryDecision({
      containerRunning: false,
      exitCode,
      images: this.imageMap(),
    });
    if (decision.action !== "start") {
      throw new Error(decision.reason ?? "container recovery refused");
    }
    const plan = await this.containerPlan(decision);
    if (!plan.ok) {
      throw new Error(plan.reason);
    }
    await container.start(plan.options);
    await container.setInactivityTimeout(INACTIVITY_TIMEOUT_MS);
    await this.ctx.storage.put("startedAt", Date.now());
    await this.ctx.storage.setAlarm(Date.now() + ALARM_MS);
    this.observeExit();
  }

  imageMap() {
    const images = this.ctx.container?.images;
    const current = images?.[IMAGE_NAME];
    const previous = images?.[PREVIOUS_IMAGE_NAME];
    if (!current && !previous) {
      return undefined;
    }
    return { [IMAGE_NAME]: current, [PREVIOUS_IMAGE_NAME]: previous };
  }

  async containerPlan(decision) {
    const image = decision?.image ?? this.ctx.container?.images?.[IMAGE_NAME];
    if (!image) {
      return { ok: false, reason: "container image is missing" };
    }
    const plan = startPlan(this.env, await this.currentFence());
    if (!plan.ok) {
      return plan;
    }
    return { ok: true, options: { ...plan.options, image } };
  }

  async currentFence() {
    const existing = await this.ctx.storage.get("fenceToken");
    if (existing) {
      return existing;
    }
    return this.bumpFence();
  }

  async bumpFence() {
    const current = Number(await this.ctx.storage.get("fence")) || 0;
    const next = current + 1;
    const token = `${next}-${crypto.randomUUID()}`;
    await this.ctx.storage.put("fence", next);
    await this.ctx.storage.put("fenceToken", token);
    return token;
  }

  async probeHealth() {
    if (!this.ctx.container?.running) {
      return false;
    }
    const response = await this.ctx.container.getTcpPort(CONTAINER_PORT).fetch("http://container/health/worker", {
      signal: AbortSignal.timeout(1000),
    });
    await response.body?.cancel();
    return response.ok;
  }
}

export default {
  async fetch(request, env) {
    return env.STOREFRONT.getByName(SINGLETON_NAME).fetch(request);
  },
};

export { containerRequest };
