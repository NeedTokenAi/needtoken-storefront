import { SIGTERM, SINGLETON_NAME } from "./policy.js";

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

export async function drain(state) {
  state.events?.push("signal");
  state.container.signal(SIGTERM);
  await state.container.monitor();
  state.events?.push("monitor");
  const fence = await state.bumpFence();
  state.events?.push("fence");
  return fence;
}

// watchdog keeps the background container running. HTTP percent 0 holds user
// traffic; it does not signal the container to stop.
export async function watchdog(state) {
  await state.scheduleNext();
  const percent = state.httpPercent === 100 ? 100 : 0;
  if (!state.container.running) {
    const plan = await state.plan();
    if (!plan.ok) {
      return { action: "refused", signaled: false, httpPercent: percent, reason: plan.reason };
    }
    state.container.start(plan.options);
    if (state.setInactivityTimeout) {
      await state.setInactivityTimeout();
    }
    return { action: "started", signaled: false, httpPercent: percent };
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
      return { action: "starting", signaled: false, httpPercent: percent };
    }
    await drain(state);
    return { action: "drained", signaled: true, httpPercent: percent };
  }
  return { action: "kept", signaled: false, httpPercent: percent };
}
