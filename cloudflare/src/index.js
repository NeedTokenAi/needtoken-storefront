import { DurableObject, WorkerEntrypoint } from "cloudflare:workers";

import {
  InternalChannelUnavailable,
  denyResponse,
  prepareGlueContainer,
  routeInternal,
  unavailableResponse,
} from "./internal.js";
import {
  APP_PORT,
  ContainerNotReady,
  INACTIVITY_TIMEOUT_MS,
  RESIDENT_INSTANCE,
  containerEnv,
  forwardRequest,
  keepResident,
  notReadyResponse,
  trustEntrypoint,
  waitForReady,
} from "./runtime.js";

export class InternalDeny extends WorkerEntrypoint {
  fetch() {
    return denyResponse();
  }
}

export class InternalHop extends WorkerEntrypoint {
  fetch(request) {
    return routeInternal(request, this.env, this.ctx.props, (binding, forwarded, init) => binding.fetch(forwarded, init));
  }
}

export class Storefront extends DurableObject {
  constructor(ctx, env) {
    super(ctx, env);
    this.env = env;
    this.ctx = ctx;
    const container = ctx.container;
    if (container?.running) {
      void ctx.blockConcurrencyWhile(async () => {
        try {
          await this.prepare(container);
          await container.setInactivityTimeout(INACTIVITY_TIMEOUT_MS);
        } catch (error) {
          this.channelError = error;
        }
      });
    }
  }

  async fetch(request) {
    if (this.channelError) return unavailableResponse();
    try {
      this.starting ??= this.startAndWait().finally(() => {
        this.starting = undefined;
      });
      await this.starting;
    } catch (error) {
      if (error instanceof InternalChannelUnavailable) return unavailableResponse();
      if (error instanceof ContainerNotReady) return notReadyResponse();
      throw error;
    }
    return this.ctx.container.getTcpPort(APP_PORT).fetch(forwardRequest(request));
  }

  prepare(container) {
    return prepareGlueContainer(
      container,
      this.env,
      (plan) => ({
        deny: this.ctx.exports.InternalDeny({ props: { host: plan.host } }),
        hop: this.ctx.exports.InternalHop({ props: { host: plan.host, binding: plan.binding } }),
      }),
      async () => {
        // Image and instance_type come from wrangler.jsonc. basic is 1/4 vCPU and 1 GiB.
        container.start({
          enableInternet: true,
          entrypoint: trustEntrypoint(),
          env: containerEnv(this.env),
        });
        await container.setInactivityTimeout(INACTIVITY_TIMEOUT_MS);
      },
    );
  }

  async startAndWait() {
    const container = this.ctx.container;
    if (!container) {
      throw new Error("container binding is missing");
    }
    try {
      await this.prepare(container);
    } catch (error) {
      if (error instanceof InternalChannelUnavailable) throw error;
      throw new ContainerNotReady(error);
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
