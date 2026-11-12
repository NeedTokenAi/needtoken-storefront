import { DurableObject } from "cloudflare:workers";

import { classifyRequest, handleTraffic, TRAFFIC_PATH, watchdog } from "./lifecycle.js";
import { CONTAINER_PORT, IMAGE_NAME, INACTIVITY_TIMEOUT_MS, SINGLETON_NAME, startPlan } from "./policy.js";
import { proxyIfReady } from "./proxy.js";

const ALARM_MS = 60 * 1000;
const START_GRACE_MS = 60 * 1000;

// Dujiao stays a Linux container. This Durable Object is the only instance:
// the Worker always addresses getByName("dujiao-storefront"). SQLite here is
// the control record (fence, traffic percent, alarm), not the application
// database. Uploads stay durable only when the container's Cloudflare profile
// has an object-storage target.
export class DujiaoStorefront extends DurableObject {
  constructor(ctx, env) {
    super(ctx, env);
    this.ctx = ctx;
    this.env = env;
    const container = ctx.container;
    if (container?.running) {
      void ctx.blockConcurrencyWhile(async () => {
        await container.setInactivityTimeout(INACTIVITY_TIMEOUT_MS);
        await ctx.storage.setAlarm(Date.now() + ALARM_MS);
      });
    }
  }

  async alarm() {
    const percent = (await this.ctx.storage.get("httpPercent")) ?? 0;
    const startedAt = (await this.ctx.storage.get("startedAt")) ?? 0;
    await watchdog({
      httpPercent: percent,
      startedAt,
      graceMs: START_GRACE_MS,
      now: () => Date.now(),
      container: this.ctx.container,
      scheduleNext: () => this.ctx.storage.setAlarm(Date.now() + ALARM_MS),
      probe: () => this.probeHealth(),
      plan: () => this.containerPlan(),
      setInactivityTimeout: () => this.ctx.container.setInactivityTimeout(INACTIVITY_TIMEOUT_MS),
      bumpFence: () => this.bumpFence(),
    });
  }

  async fetch(request) {
    const url = new URL(request.url);
    const percent = (await this.ctx.storage.get("httpPercent")) ?? 0;
    const action = classifyRequest(url.pathname, percent);
    if (action === "control") {
      return this.traffic(request);
    }
    await this.ensureRunning();
    if (action !== "proxy") {
      return new Response("storefront traffic is held at 0%", { status: 503 });
    }
    return proxyIfReady(request, {
      attempts: 30,
      sleep: () => new Promise((resolve) => setTimeout(resolve, 1000)),
      probe: () => this.probeHealth(),
      fetch: (readyRequest) => this.ctx.container.getTcpPort(CONTAINER_PORT).fetch(readyRequest),
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
    const plan = await this.containerPlan();
    if (!plan.ok) {
      throw new Error(plan.reason);
    }
    container.start(plan.options);
    await container.setInactivityTimeout(INACTIVITY_TIMEOUT_MS);
    await this.ctx.storage.put("startedAt", Date.now());
    await this.ctx.storage.setAlarm(Date.now() + ALARM_MS);
  }

  async containerPlan() {
    const image = this.ctx.container?.images?.[IMAGE_NAME];
    if (!image) {
      return { ok: false, reason: "container image dujiao is missing" };
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
    const response = await this.ctx.container.getTcpPort(CONTAINER_PORT).fetch("http://container/health/worker");
    return response.ok;
  }
}

export default {
  async fetch(request, env) {
    return env.STOREFRONT.getByName(SINGLETON_NAME).fetch(request);
  },
};
