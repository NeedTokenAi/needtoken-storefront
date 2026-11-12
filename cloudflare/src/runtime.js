export const APP_PORT = 8080;

// Cloud Run injects PORT. The production env copied into this Worker does not.
export const CONTAINER_PORT = "8080";

// argv, not a shell string. Joined, this is /app/dujiao-next -mode=api.
export const START_COMMAND = ["/app/dujiao-next", "-mode=api"];

// Predefined basic is 1/4 vCPU and 1 GiB. Wrangler instance_type carries it.
export const INSTANCE_TYPE = "basic";

export const MAX_INSTANCES = 2;

// One container runs the in-process paid-fulfillment loop. max_instances 2
// would start a second loop if another name were addressed.
export const RESIDENT_INSTANCE = "instance-0";

// UTC, once a minute. The in-process ticker does not count as container activity.
export const RESIDENT_CRON = "* * * * *";

// sleepAfter. setInactivityTimeout rejects any value above 6 hours, so 24h
// cannot be applied. This is the platform maximum. The minute cron resets it.
export const INACTIVITY_TIMEOUT_MS = 6 * 60 * 60 * 1000;

// The published cold start took 2–3 minutes. Wrangler 4 stops the old container
// before it starts the new one, so this wait has to cover that gap.
export const READY_DEADLINE_MS = 5 * 60 * 1000;
export const READY_POLL_MS = 500;
export const NOT_READY_RETRY_AFTER_SECONDS = "10";

export class ContainerNotReady extends Error {
  constructor(cause) {
    super("container did not become ready on port 8080", { cause });
    this.name = "ContainerNotReady";
  }
}

export function containerEnv(env) {
  const out = {};
  if (env == null || typeof env !== "object") {
    return out;
  }
  for (const [key, value] of Object.entries(env)) {
    if (typeof value === "string") {
      out[key] = value;
    }
  }
  out.PORT = CONTAINER_PORT;
  return out;
}

export function notReadyResponse() {
  return new Response("container is starting", {
    status: 503,
    headers: {
      "retry-after": NOT_READY_RETRY_AFTER_SECONDS,
      "cache-control": "no-store",
    },
  });
}

export async function waitForReady(probe, wait, deadlineMs = READY_DEADLINE_MS, now = Date.now) {
  const deadline = now() + deadlineMs;
  let lastError;
  while (now() <= deadline) {
    try {
      const response = await probe();
      await response.body?.cancel();
      if (response.ok) return;
      lastError = new Error(`health check returned ${response.status}`);
    } catch (error) {
      lastError = error;
    }
    if (now() >= deadline) break;
    await wait(Math.min(READY_POLL_MS, deadline - now()));
  }
  throw new ContainerNotReady(lastError);
}

export async function keepResident(namespace) {
  const response = await namespace.getByName(RESIDENT_INSTANCE).fetch(
    new Request("https://internal/health"),
  );
  await response.body?.cancel();
  if (!response.ok) {
    throw new Error(`resident health returned ${response.status}`);
  }
}

export function forwardRequest(request) {
  const url = new URL(request.url);
  url.protocol = "http:";
  url.host = "container";
  const forwarded = new Request(url, request);
  forwarded.headers.delete("host");
  return forwarded;
}
