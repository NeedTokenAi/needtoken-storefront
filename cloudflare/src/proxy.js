export async function proxyIfReady(request, deps) {
  const attempts = deps.attempts ?? 1;
  for (let attempt = 0; attempt < attempts; attempt += 1) {
    const ready = await deps.probe();
    if (ready) {
      return deps.fetch(request);
    }
    if (attempt + 1 < attempts && deps.sleep) {
      await deps.sleep();
    }
  }
  return new Response("container is not ready", { status: 503 });
}
