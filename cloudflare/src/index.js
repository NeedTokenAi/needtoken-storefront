import { DurableObject } from "cloudflare:workers";

import {
  APP_PORT,
  INACTIVITY_TIMEOUT_MS,
  INSTANCE,
  START_COMMAND,
  containerEnv,
  forwardRequest,
} from "./runtime.js";

const SINGLETON_NAME = "needtoken-storefront";

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
    this.starting ??= this.startAndWait().finally(() => {
      this.starting = undefined;
    });
    await this.starting;
    return this.ctx.container.getTcpPort(APP_PORT).fetch(forwardRequest(request));
  }

  async startAndWait() {
    const container = this.ctx.container;
    if (!container) {
      throw new Error("container binding is missing");
    }
    if (!container.running) {
      container.start({
        image: container.images.app,
        instance: INSTANCE,
        enableInternet: true,
        entrypoint: START_COMMAND,
        env: containerEnv(this.env),
      });
      await container.setInactivityTimeout(INACTIVITY_TIMEOUT_MS);
    }
    const port = container.getTcpPort(APP_PORT);
    let lastError;
    for (let attempt = 0; attempt < 100; attempt++) {
      try {
        const response = await port.fetch("http://container/health", {
          signal: AbortSignal.timeout(1000),
        });
        await response.body?.cancel();
        if (!response.ok) {
          throw new Error(`health check returned ${response.status}`);
        }
        return;
      } catch (error) {
        lastError = error;
        await scheduler.wait(200);
      }
    }
    throw new Error("container did not become ready on port 8080", { cause: lastError });
  }
}

export default {
  fetch(request, env) {
    return env.STOREFRONT.getByName(SINGLETON_NAME).fetch(request);
  },
};
