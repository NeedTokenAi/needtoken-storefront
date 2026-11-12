import { waitUntilReady } from "./lifecycle.js";

// Official container fetch: keep the original Request so upgrade and the
// cancellation signal survive, then point the URL at the container host.
export function containerRequest(request) {
  const url = new URL(request.url);
  url.protocol = "http:";
  url.host = "container";
  const forwarded = new Request(url, request);
  forwarded.headers.delete("host");
  return forwarded;
}

export async function proxyIfReady(request, deps) {
  const ready = await waitUntilReady(deps.probe, {
    attempts: deps.attempts ?? 30,
    delayMs: deps.delayMs ?? 200,
    wait: deps.wait,
  });
  if (!ready.ready) {
    return new Response("container is not ready", { status: 503 });
  }
  return deps.fetch(containerRequest(request));
}
