export const APP_PORT = 8080;

// Cloud Run injects PORT. The production env copied into this Worker does not.
export const CONTAINER_PORT = "8080";

// argv, not a shell string. Joined, this is /app/dujiao-next -mode=api.
export const START_COMMAND = ["/app/dujiao-next", "-mode=api"];

// durable_object cannot express max_instances. These two names are the cap.
export const MAX_INSTANCES = 2;

// No named type is 1 vCPU and 512 MiB. diskMb is the durable_object minimum.
export const INSTANCE = {
  vcpu: 1,
  memoryMib: 512,
  diskMb: 2000,
};

// Platform maximum. A shorter timeout would stop the process between requests.
export const INACTIVITY_TIMEOUT_MS = 6 * 60 * 60 * 1000;

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

export function instanceName(random = Math.random) {
  const index = Math.floor(Number(random()) * MAX_INSTANCES);
  if (index < 0 || index >= MAX_INSTANCES) {
    throw new Error("instance index out of range");
  }
  return `instance-${index}`;
}

export function forwardRequest(request) {
  const url = new URL(request.url);
  url.protocol = "http:";
  url.host = "container";
  const forwarded = new Request(url, request);
  forwarded.headers.delete("host");
  return forwarded;
}
