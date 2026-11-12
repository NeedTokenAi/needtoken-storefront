import { IMAGE_NAME, PREVIOUS_IMAGE_NAME, SIGTERM, SINGLETON_NAME } from "./policy.js";

export const TRAFFIC_PATH = "/__cloudflare/traffic";

export function singletonName(_requested) {
  return SINGLETON_NAME;
}

export function classifyRequest(pathname, percent) {
  if (pathname === TRAFFIC_PATH) {
    return "control";
  }
  if (percent === 100) {
    return "proxy";
  }
  return "hold";
}

export function handleTraffic(body, authorization, expectedToken) {
  if (!expectedToken) {
    return { status: 403, error: "control token is not configured" };
  }
  if (authorization !== `Bearer ${expectedToken}`) {
    return { status: 401, error: "control token rejected" };
  }
  const percent = body && body.percent;
  if (percent !== 0 && percent !== 100) {
    return { status: 400, error: "percent must be 0 or 100" };
  }
  return { status: 200, percent };
}

// scheduler.wait is the request-context timer from the current Workers
// web-standards page and the Containers get-started example. It is not a
// substitute for an alarm: alarms are what run when no request is in flight.
// setTimeout does not replace it. Tests inject a fake scheduler.
export function schedulerWait(ms, schedulerImpl = globalThis.scheduler) {
  if (!schedulerImpl || typeof schedulerImpl.wait !== "function") {
    return Promise.reject(new Error("scheduler.wait is required and setTimeout is not a Durable Object sleep"));
  }
  return schedulerImpl.wait(ms);
}

// waitUntilReady retries a closed port. A thrown ECONNREFUSED is one attempt,
// not a terminal failure, until the attempt budget is spent.
export async function waitUntilReady(probe, options = {}) {
  const attempts = options.attempts ?? 5;
  if (typeof options.wait !== "function") {
    throw new Error("readiness wait must be scheduler.wait");
  }
  let lastError;
  let probes = 0;
  for (let attempt = 0; attempt < attempts; attempt += 1) {
    probes += 1;
    try {
      if (await probe()) {
        return { ready: true, probes, error: lastError };
      }
    } catch (error) {
      lastError = error;
    }
    if (attempt + 1 < attempts) {
      await options.wait(options.delayMs ?? 200);
    }
  }
  return { ready: false, probes, error: lastError };
}

export async function armAlarm(storage, now, intervalMs) {
  const existing = await storage.getAlarm();
  if (existing) {
    return existing;
  }
  const at = now + intervalMs;
  await storage.setAlarm(at);
  return at;
}

// A rebuilt Durable Object must arm the alarm even when no request arrived.
// A running container is reattached. A started flag is not consulted.
export async function prepareDurableObject(state) {
  const alarmAt = await armAlarm(state.storage, state.now(), state.intervalMs);
  if (state.container?.running) {
    if (state.setInactivityTimeout) {
      await state.setInactivityTimeout();
    }
    state.observeExit?.();
    return { action: "reattach", alarmAt, rollback: false };
  }
  return { action: "armed", alarmAt, rollback: false };
}

// Two configured digests are a recovery target only. rollback stays false:
// this skeleton has no proven previous production image and no real account.
export function recoveryDecision(input) {
  if (input?.containerRunning) {
    return { action: "keep", rollback: false };
  }
  if (!input?.images) {
    return { action: "start", role: "current", rollback: false };
  }
  const current = input.images[IMAGE_NAME];
  const previous = input.images[PREVIOUS_IMAGE_NAME];
  const distinctPrevious = previous && previous !== current;
  if (input.exitCode && input.exitCode !== 0 && distinctPrevious) {
    return {
      action: "start",
      image: previous,
      role: "previous",
      rollback: false,
      reason: "previous digest is a recovery target, not a production rollback",
    };
  }
  if (!current) {
    return { action: "refused", rollback: false, reason: "current image missing" };
  }
  return { action: "start", image: current, role: "current", rollback: false };
}

// The lifecycle doc rejects monitor() with error.exitCode on a non-zero exit.
// error.code remains a fallback for older shapes. A rejection must be handled.
export function observeExit(container, onExit) {
  return container.monitor().then(
    () => {
      onExit?.({ code: 0 });
    },
    (error) => {
      const code =
        typeof error?.exitCode === "number" ? error.exitCode : typeof error?.code === "number" ? error.code : 1;
      onExit?.({ code, error });
    },
  );
}

export async function drain(state) {
  state.events?.push("signal");
  state.container.signal(SIGTERM);
  try {
    await state.container.monitor();
    state.events?.push("monitor");
  } catch (error) {
    state.events?.push("monitor-refused");
    state.monitorError = error;
  }
  const fence = await state.bumpFence();
  state.events?.push("fence");
  return fence;
}

// watchdog keeps the background container running. HTTP percent 0 holds user
// traffic; it does not signal the container to stop. A started flag cannot
// skip a restart after the container is no longer running.
export async function watchdog(state) {
  await state.scheduleNext();
  const percent = state.httpPercent === 100 ? 100 : 0;
  if (!state.container.running) {
    const decision = recoveryDecision({
      containerRunning: false,
      startedFlag: state.startedFlag,
      exitCode: state.exitCode,
      images: state.images,
    });
    if (decision.action !== "start") {
      return {
        action: "refused",
        signaled: false,
        httpPercent: percent,
        rollback: false,
        reason: decision.reason,
      };
    }
    try {
      const plan = await state.plan(decision);
      if (!plan.ok) {
        return { action: "refused", signaled: false, httpPercent: percent, rollback: false, reason: plan.reason };
      }
      await state.container.start({ ...plan.options, image: decision.image ?? plan.options.image });
    } catch (error) {
      return { action: "start-failed", signaled: false, httpPercent: percent, rollback: false, error };
    }
    if (state.setInactivityTimeout) {
      await state.setInactivityTimeout();
    }
    return { action: "started", signaled: false, httpPercent: percent, rollback: false, image: decision.image, role: decision.role };
  }

  let healthy = true;
  if (state.probe) {
    try {
      healthy = Boolean(await state.probe());
    } catch {
      healthy = false;
    }
  }
  if (!healthy) {
    const now = state.now ? state.now() : 0;
    const age = now - (state.startedAt ?? 0);
    if (age < (state.graceMs ?? 0)) {
      return { action: "starting", signaled: false, httpPercent: percent, rollback: false };
    }
    await drain(state);
    return { action: "drained", signaled: true, httpPercent: percent, rollback: false };
  }
  return { action: "kept", signaled: false, httpPercent: percent, rollback: false };
}
