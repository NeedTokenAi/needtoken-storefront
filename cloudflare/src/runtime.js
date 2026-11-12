export const APP_PORT = 8080;

// argv, not a shell string. Joined, this is /app/dujiao-next -mode=api.
export const START_COMMAND = ["/app/dujiao-next", "-mode=api"];

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
  return out;
}

export function forwardRequest(request) {
  const url = new URL(request.url);
  url.protocol = "http:";
  url.host = "container";
  const forwarded = new Request(url, request);
  forwarded.headers.delete("host");
  return forwarded;
}
