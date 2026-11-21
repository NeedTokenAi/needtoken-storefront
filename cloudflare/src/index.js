import { DurableObject } from "cloudflare:workers";

import { handleEdgeAdminGate } from "./admin_gate.js";
import {
  APP_PORT,
  ContainerNotReady,
  INACTIVITY_REFRESH_MS,
  INACTIVITY_TIMEOUT_MS,
  RESIDENT_INSTANCE,
  START_COMMAND,
  containerEnv,
  forwardRequest,
  keepResident,
  notReadyResponse,
  waitForReady,
} from "./runtime.js";

export class Storefront extends DurableObject {
  constructor(ctx, env) {
    super(ctx, env);
    this.env = env;
    this.ctx = ctx;
    this.starting = undefined;
    this.portReady = false;
    this.lastActivityRefreshAt = 0;
    const container = ctx.container;
    if (container?.running) {
      void ctx.blockConcurrencyWhile(() => container.setInactivityTimeout(INACTIVITY_TIMEOUT_MS));
      this.lastActivityRefreshAt = Date.now();
    }
  }

  async fetch(request) {
    const gate = await handleEdgeAdminGate(request, this.env, this.ctx.storage?.sql);
    if (gate.response) return gate.response;
    try {
      await this.ensureStarted();
      return await this.ctx.container.getTcpPort(APP_PORT).fetch(forwardRequest(request));
    } catch (error) {
      this.portReady = false;
      if (error instanceof ContainerNotReady) return notReadyResponse();
      throw error;
    }
  }

  async ensureStarted() {
    const container = this.ctx.container;
    if (this.portReady && container?.running) {
      const now = Date.now();
      if (now - this.lastActivityRefreshAt >= INACTIVITY_REFRESH_MS) {
        this.lastActivityRefreshAt = now;
        void container.setInactivityTimeout(INACTIVITY_TIMEOUT_MS).catch(() => {});
      }
      return;
    }
    this.starting ??= this.startAndWait().finally(() => {
      this.starting = undefined;
    });
    await this.starting;
  }

  async startAndWait() {
    const container = this.ctx.container;
    if (!container) {
      throw new Error("container binding is missing");
    }
    if (!container.running) {
      try {
        // Image and instance_type come from wrangler.jsonc. basic is 1/4 vCPU and 1 GiB.
        this.portReady = false;
        container.start({
          enableInternet: true,
          entrypoint: START_COMMAND,
          env: containerEnv(this.env),
        });
        await container.setInactivityTimeout(INACTIVITY_TIMEOUT_MS);
        this.lastActivityRefreshAt = Date.now();
      } catch (error) {
        throw new ContainerNotReady(error);
      }
    }
    const port = container.getTcpPort(APP_PORT);
    await waitForReady(
      () => port.fetch("http://container/health", { signal: AbortSignal.timeout(1000) }),
      (ms) => scheduler.wait(ms),
    );
    this.portReady = true;
  }
}

export default {
  fetch(request, env) {
    return env.STOREFRONT.getByName(RESIDENT_INSTANCE).fetch(request);
  },

  async scheduled(_controller, env) {
    await keepResident(env.STOREFRONT);
  },
};
