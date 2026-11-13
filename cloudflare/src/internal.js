// Outbound hop from the storefront container to needtoken-glue.
// The Worker cannot see the database site connection. It intercepts one hostname.

export const GLUE_BINDING = "INTERNAL_GLUE";
export const GLUE_SERVICE = "needtoken-glue";

const HOSTNAME = /^(?=.{1,253}$)[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?(?:\.[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?)*$/i;

export class InternalChannelUnavailable extends Error {
  constructor(cause) {
    super("internal channel unavailable", { cause });
    this.name = "InternalChannelUnavailable";
  }
}

export function isSingleHostname(value) {
  return typeof value === "string" && HOSTNAME.test(value) && !/^\d{1,3}(?:\.\d{1,3}){3}$/.test(value);
}

// Missing or illegal INTERNAL_GLUE_HOST refuses container start.
export function glueInterceptPlan(env) {
  const host = env?.INTERNAL_GLUE_HOST;
  if (!isSingleHostname(host)) {
    throw new InternalChannelUnavailable(new Error("INTERNAL_GLUE_HOST is missing or not a single hostname"));
  }
  return { host, binding: GLUE_BINDING };
}

export async function registerGlueIntercept(container, target, plan) {
  await container.interceptOutboundHttp(plan.host, target.deny);
  await container.interceptOutboundHttps(plan.host, target.hop);
}

// Register the one hostname before start. A plan or intercept error never calls start.
export async function prepareGlueContainer(container, env, target, startContainer) {
  const plan = glueInterceptPlan(env);
  try {
    await registerGlueIntercept(container, target(plan), plan);
  } catch (error) {
    throw new InternalChannelUnavailable(error);
  }
  if (!container.running) {
    await startContainer();
  }
  return plan;
}

export function unavailableResponse() {
  return new Response("internal channel unavailable", {
    status: 503,
    headers: { "cache-control": "no-store" },
  });
}

export function failedResponse() {
  return new Response("internal channel failed", {
    status: 502,
    headers: { "cache-control": "no-store" },
  });
}

export function denyResponse() {
  return new Response("internal channel denied", {
    status: 403,
    headers: { "cache-control": "no-store" },
  });
}

// props.binding is the only target. A forged Host or header cannot select another one.
export async function routeInternal(request, env, props, fetchBinding) {
  let url;
  try {
    url = new URL(request.url);
  } catch {
    return denyResponse();
  }
  if (
    props?.host !== url.hostname ||
    url.protocol !== "https:" ||
    url.port !== "" ||
    url.username !== "" ||
    url.password !== "" ||
    props?.binding !== GLUE_BINDING
  ) {
    return denyResponse();
  }
  const binding = env?.[props.binding];
  if (binding == null || typeof binding.fetch !== "function") {
    return failedResponse();
  }
  const forwarded = new Request(url, request);
  forwarded.headers.delete("Cloudflare-Workers-Version-Overrides");
  try {
    return await fetchBinding(binding, forwarded, { redirect: "manual" });
  } catch {
    return failedResponse();
  }
}
