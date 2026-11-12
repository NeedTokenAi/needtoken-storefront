import { DurableObject } from "cloudflare:workers";

import {
  APP_PORT,
  ContainerNotReady,
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
    const container = ctx.container;
    if (container?.running) {
      void ctx.blockConcurrencyWhile(() => container.setInactivityTimeout(INACTIVITY_TIMEOUT_MS));
    }
  }

  async fetch(request) {
    try {
      this.starting ??= this.startAndWait().finally(() => {
        this.starting = undefined;
      });
      await this.starting;
    } catch (error) {
      if (error instanceof ContainerNotReady) return notReadyResponse();
      throw error;
    }
    return this.ctx.container.getTcpPort(APP_PORT).fetch(forwardRequest(request));
  }

  async startAndWait() {
    const container = this.ctx.container;
    if (!container) {
      throw new Error("container binding is missing");
    }
    if (!container.running) {
      try {
        // Image and instance_type come from wrangler.jsonc. basic is 1/4 vCPU and 1 GiB.
        container.start({
          enableInternet: true,
          entrypoint: START_COMMAND,
          env: containerEnv(this.env),
        });
        await container.setInactivityTimeout(INACTIVITY_TIMEOUT_MS);
      } catch (error) {
        throw new ContainerNotReady(error);
      }
    }
    const port = container.getTcpPort(APP_PORT);
    await waitForReady(
      () => port.fetch("http://container/health", { signal: AbortSignal.timeout(1000) }),
      (ms) => scheduler.wait(ms),
    );
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
