import assert from "node:assert/strict";
import { spawnSync } from "node:child_process";
import { chmodSync, existsSync, mkdtempSync, readFileSync, rmSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { dirname, join } from "node:path";
import test from "node:test";
import { fileURLToPath } from "node:url";

import {
  GLUE_BINDING,
  GLUE_SERVICE,
  InternalChannelUnavailable,
  failedResponse,
  glueInterceptPlan,
  prepareGlueContainer,
  routeInternal,
  unavailableResponse,
} from "./internal.js";
import {
  APP_PORT,
  CLOUDFLARE_CONTAINER_CA,
  ContainerNotReady,
  INSTANCE_TYPE,
  INACTIVITY_TIMEOUT_MS,
  MAX_INSTANCES,
  NOT_READY_RETRY_AFTER_SECONDS,
  READY_DEADLINE_MS,
  RESIDENT_CRON,
  RESIDENT_INSTANCE,
  START_COMMAND,
  SYSTEM_CA_BUNDLE,
  SYSTEM_CA_FALLBACK,
  TRUSTED_CA_FILE,
  containerEnv,
  forwardRequest,
  keepResident,
  notReadyResponse,
  trustEntrypoint,
  trustScript,
  waitForReady,
} from "./runtime.js";

const root = dirname(dirname(fileURLToPath(import.meta.url)));

test("production worker builds the repo Dockerfile and publishes no vars", () => {
  const wrangler = JSON.parse(readFileSync(join(root, "wrangler.jsonc"), "utf8"));
  const container = wrangler.containers[0];
  assert.equal(wrangler.name, "needtoken-storefront");
  assert.equal(wrangler.workers_dev, false);
  assert.equal(wrangler.preview_urls, false);
  assert.deepEqual(wrangler.triggers.crons, [RESIDENT_CRON]);
  assert.equal(RESIDENT_CRON, "* * * * *");
  assert.equal(wrangler.routes, undefined);
  assert.equal(JSON.stringify(wrangler).includes("needtoken.ai"), false);
  assert.equal(JSON.stringify(wrangler).toLowerCase().includes("staging"), false);
  assert.equal(container.scheduling_policy, "default");
  assert.equal(container.instance_type, "basic");
  assert.equal(container.class_name, "Storefront");
  assert.equal(container.max_instances, 2);
  assert.equal(container.rollout_step_percentage, 100);
  assert.equal(container.rollout_active_grace_period, undefined);
  assert.equal(wrangler.migrations, undefined);
  assert.equal(container.image, "../Dockerfile");
  assert.equal(container.image_build_context, "..");
  assert.equal(existsSync(join(root, container.image)), true);
  assert.equal(container.image.includes("registry.cloudflare.com"), false);
  assert.equal(container.image.includes("sha256:"), false);
  assert.deepEqual(container.constraints.regions, ["APAC"]);
  assert.deepEqual(wrangler.observability, {
    enabled: true,
    logs: { enabled: true },
    head_sampling_rate: 1,
  });
  assert.equal(Object.hasOwn(container, "images"), false);
  assert.equal(Object.hasOwn(container, "dockerfile"), false);
  assert.equal(wrangler.vars, undefined);
  assert.deepEqual(wrangler.services, [{ binding: GLUE_BINDING, service: GLUE_SERVICE }]);
  assert.equal(wrangler.services[0].entrypoint, undefined);
  assert.equal(JSON.stringify(wrangler).includes("INTERNAL_GLUE_HOST"), false);
  assert.equal(JSON.stringify(wrangler).includes("sha256:"), false);
  assert.equal(JSON.stringify(wrangler).includes("REDIS_PREFIX"), false);
  assert.equal(JSON.stringify(wrangler).includes("OPS_WRITES_ENABLED"), false);
});

test("start command, port, and instance size stay fixed", () => {
  assert.equal(APP_PORT, 8080);
  assert.deepEqual(START_COMMAND, ["/app/dujiao-next", "-mode=api"]);
  assert.equal(START_COMMAND.join(" "), "/app/dujiao-next -mode=api");
  assert.equal(INSTANCE_TYPE, "basic");
  assert.equal(MAX_INSTANCES, 2);
  assert.equal(RESIDENT_INSTANCE, "instance-0");
  assert.equal(INACTIVITY_TIMEOUT_MS, 6 * 60 * 60 * 1000);
  assert.equal(READY_DEADLINE_MS, 5 * 60 * 1000);
});

test("string env and secrets pass through unchanged", () => {
  const binding = { fetch() {} };
  const env = {
    STOREFRONT: binding,
    DATABASE_URL: "postgres://app@db.example.test:5432/store",
    REDIS_PASSWORD: "kept-secret",
    EMPTY: "",
  };
  const passed = containerEnv(env);
  assert.deepEqual(passed, {
    DATABASE_URL: "postgres://app@db.example.test:5432/store",
    REDIS_PASSWORD: "kept-secret",
    EMPTY: "",
    PORT: "8080",
  });
  assert.equal(containerEnv({ PORT: "3000" }).PORT, "8080");
  assert.equal(Object.hasOwn(passed, "REDIS_PREFIX"), false);
  assert.equal(Object.hasOwn(passed, "STOREFRONT"), false);
  assert.equal(passed.DATABASE_URL, env.DATABASE_URL);
});

test("requests are forwarded to the container port", async () => {
  const body = new Request("https://needtoken-storefront.example.workers.dev/api/v1/orders?x=1", {
    method: "POST",
    body: '{"ok":true}',
    headers: { "content-type": "application/json", host: "needtoken-storefront.example.workers.dev" },
  });
  const forwarded = forwardRequest(body);
  assert.equal(forwarded.url, "http://container/api/v1/orders?x=1");
  assert.equal(forwarded.method, "POST");
  assert.equal(await forwarded.text(), '{"ok":true}');
  assert.equal(forwarded.headers.get("host"), null);
  assert.equal(forwarded.headers.get("content-type"), "application/json");
});

test("readiness waits through a cold start, then answers 503", async () => {
  let probes = 0;
  await waitForReady(
    async () => {
      probes += 1;
      if (probes < 3) throw new Error("down");
      return new Response("ok");
    },
    async () => {},
    10_000,
    () => 0,
  );
  assert.equal(probes, 3);

  let clock = 0;
  await assert.rejects(
    () =>
      waitForReady(
        async () => {
          throw new Error("down");
        },
        async (ms) => {
          clock += ms;
        },
        1_000,
        () => clock,
      ),
    (error) => error instanceof ContainerNotReady && /port 8080/.test(error.message),
  );
  assert.ok(clock >= 1_000);

  const response = notReadyResponse();
  assert.equal(response.status, 503);
  assert.equal(response.headers.get("retry-after"), NOT_READY_RETRY_AFTER_SECONDS);
  assert.equal(await response.text(), "container is starting");
});

test("the minute cron wakes only instance-0", async () => {
  const names = [];
  const namespace = {
    getByName(name) {
      names.push(name);
      return {
        fetch() {
          return new Response("ok", { status: 200 });
        },
      };
    },
  };
  await keepResident(namespace);
  await keepResident(namespace);
  assert.deepEqual(names, ["instance-0", "instance-0"]);

  const failing = {
    getByName(name) {
      names.push(name);
      return {
        fetch() {
          return new Response("down", { status: 503 });
        },
      };
    },
  };
  await assert.rejects(() => keepResident(failing), /resident health returned 503/);
  assert.deepEqual(names, ["instance-0", "instance-0", "instance-0"]);
});

test("the worker source keeps internet, instance-0, and the trust entrypoint", () => {
  const source = readFileSync(join(root, "src/index.js"), "utf8");
  const internal = readFileSync(join(root, "src/internal.js"), "utf8");
  const worker = source.slice(source.indexOf("export default"));
  assert.equal(source.includes("container.images"), false);
  assert.equal(source.includes("memoryMib"), false);
  assert.match(source, /entrypoint: trustEntrypoint\(\)/);
  assert.match(source, /env: containerEnv\(this\.env\)/);
  assert.match(source, /getTcpPort\(APP_PORT\)/);
  assert.match(source, /forwardRequest\(request\)/);
  assert.match(source, /enableInternet: true/);
  assert.match(source, /getByName\(RESIDENT_INSTANCE\)/);
  assert.match(source, /async scheduled\(_controller, env\)/);
  assert.match(source, /keepResident\(env\.STOREFRONT\)/);
  assert.match(source, /error instanceof ContainerNotReady/);
  assert.match(source, /error instanceof InternalChannelUnavailable/);
  assert.match(source, /return notReadyResponse\(\)/);
  assert.match(source, /return unavailableResponse\(\)/);
  assert.match(source, /waitForReady\(/);
  assert.match(source, /prepareGlueContainer\(/);
  assert.equal(source.includes("instanceName"), false);
  assert.equal(source.includes("instance-1"), false);
  assert.equal(source.includes("needtoken.ai"), false);
  assert.equal(source.includes("staging"), false);
  assert.equal(source.includes("dockerfile"), false);
  assert.equal(source.includes("REDIS_PREFIX"), false);
  assert.equal(source.includes("OPS_WRITES_ENABLED"), false);
  assert.equal(source.includes("START_COMMAND"), false);
  assert.equal(source.includes("BaseURL"), false);
  assert.equal(source.includes("site_connections"), false);
  assert.equal(internal.includes("BaseURL"), false);
  assert.equal(internal.includes("site_connections"), false);
  assert.equal(source.includes('interceptOutboundHttps("*")'), false);
  assert.equal(internal.includes('interceptOutboundHttps("*")'), false);
  assert.equal(internal.includes("interceptAllOutboundHttp"), false);
  assert.equal(source.includes("x-internal-trust"), false);
  assert.equal(internal.includes("x-internal-trust"), false);
  assert.equal(worker.includes("INTERNAL_"), false);
  assert.equal(worker.includes("X-Internal-Target"), false);
});

const GLUE_HOST = "glue.example";

function glueEnv(extra = {}) {
  return { INTERNAL_GLUE_HOST: GLUE_HOST, ...extra };
}

async function withoutGlobalFetch(run) {
  const original = globalThis.fetch;
  let calls = 0;
  globalThis.fetch = () => {
    calls += 1;
    throw new Error("global fetch");
  };
  try {
    const result = await run();
    assert.equal(calls, 0);
    return result;
  } finally {
    globalThis.fetch = original;
  }
}

function bindingFetch(response) {
  const seen = [];
  const binding = {
    async fetch(request, init) {
      seen.push({ request, init });
      if (typeof response === "function") return response(request, init);
      return response;
    },
  };
  return { binding, seen };
}

test("missing or illegal INTERNAL_GLUE_HOST refuses start and intercepts one host", async () => {
  const http = [];
  const https = [];
  let started = 0;
  const container = {
    running: false,
    async interceptOutboundHttp(host) {
      http.push(host);
    },
    async interceptOutboundHttps(host) {
      https.push(host);
    },
  };
  const plan = await prepareGlueContainer(
    container,
    glueEnv({ HTTP_PROXY: "http://proxy.example:8080", HTTPS_PROXY: "http://proxy.example:8080" }),
    (next) => ({ deny: { host: next.host }, hop: { binding: next.binding } }),
    async () => {
      started += 1;
    },
  );
  assert.deepEqual(plan, { host: GLUE_HOST, binding: GLUE_BINDING });
  assert.deepEqual(http, [GLUE_HOST]);
  assert.deepEqual(https, [GLUE_HOST]);
  assert.equal(started, 1);
  assert.equal(http.includes("proxy.example"), false);
  assert.equal(http.includes("*"), false);

  started = 0;
  for (const env of [{}, { INTERNAL_GLUE_HOST: "" }, { INTERNAL_GLUE_HOST: "https://glue.example" }, { INTERNAL_GLUE_HOST: "glue.example:443" }, { INTERNAL_GLUE_HOST: "glue.example." }, { INTERNAL_GLUE_HOST: "*.glue.example" }, { INTERNAL_GLUE_HOST: "10.0.0.1" }, { INTERNAL_GLUE_HOST: "user@glue.example" }]) {
    await assert.rejects(
      () => prepareGlueContainer(container, env, () => ({}), async () => {
        started += 1;
      }),
      (error) => error instanceof InternalChannelUnavailable,
    );
  }
  assert.equal(started, 0);

  const running = { ...container, running: true };
  await prepareGlueContainer(running, glueEnv(), () => ({ deny: {}, hop: {} }), async () => {
    started += 1;
  });
  assert.equal(started, 0);

  const broken = {
    running: false,
    interceptOutboundHttp() {
      throw new Error("intercept down");
    },
  };
  await assert.rejects(
    () => prepareGlueContainer(broken, glueEnv(), () => ({ deny: {}, hop: {} }), async () => {
      started += 1;
    }),
    (error) => error instanceof InternalChannelUnavailable,
  );
  assert.equal(started, 0);
  assert.throws(() => glueInterceptPlan({ INTERNAL_GLUE_HOST: "glue.example/orders" }), InternalChannelUnavailable);
});

test("internal hop uses the binding, manual redirect, and never global fetch", async () => {
  const { binding, seen } = bindingFetch(new Response("ok", { status: 200 }));
  const response = await withoutGlobalFetch(() => routeInternal(
    new Request(`https://${GLUE_HOST}/api/v1/upstream/orders?x=1`, {
      method: "POST",
      body: '{"sku_id":1}',
      headers: {
        "content-type": "application/json",
        "dujiao-next-api-key": "kept",
        "dujiao-next-signature": "kept",
        "x-internal-target": "https://evil.example",
        "cloudflare-workers-version-overrides": "forged",
      },
    }),
    { [GLUE_BINDING]: binding, GLUE: { fetch() { throw new Error("wrong binding"); } } },
    { host: GLUE_HOST, binding: GLUE_BINDING },
    (target, request, init) => target.fetch(request, init),
  ));
  assert.equal(response.status, 200);
  assert.equal(seen.length, 1);
  assert.equal(seen[0].init.redirect, "manual");
  assert.equal(seen[0].request.url, `https://${GLUE_HOST}/api/v1/upstream/orders?x=1`);
  assert.equal(seen[0].request.headers.get("dujiao-next-api-key"), "kept");
  assert.equal(seen[0].request.headers.get("dujiao-next-signature"), "kept");
  assert.equal(seen[0].request.headers.get("cloudflare-workers-version-overrides"), null);
  assert.equal(seen[0].request.headers.get("x-internal-trust"), null);
  assert.equal(await seen[0].request.text(), '{"sku_id":1}');

  const redirect = await withoutGlobalFetch(() => routeInternal(
    new Request(`https://${GLUE_HOST}/api/v1/upstream/ping`),
    { [GLUE_BINDING]: { fetch: () => new Response(null, { status: 302, headers: { location: "https://evil.example/collect" } }) } },
    { host: GLUE_HOST, binding: GLUE_BINDING },
    (target, request, init) => target.fetch(request, init),
  ));
  assert.equal(redirect.status, 302);
  assert.equal(redirect.headers.get("location"), "https://evil.example/collect");

  const tooLarge = await withoutGlobalFetch(() => routeInternal(
    new Request(`https://${GLUE_HOST}/api/v1/upstream/orders`, { method: "POST", body: "{}" }),
    { [GLUE_BINDING]: { fetch: () => new Response("too large", { status: 413 }) } },
    { host: GLUE_HOST, binding: GLUE_BINDING },
    (target, request, init) => target.fetch(request, init),
  ));
  assert.equal(tooLarge.status, 413);
  assert.equal(await tooLarge.text(), "too large");
});

test("internal failures stay off global fetch", async () => {
  const cases = [
    [{ host: "other.example", binding: GLUE_BINDING }, glueEnvBinding(), 403],
    [{ host: GLUE_HOST, binding: "GLUE" }, glueEnvBinding(), 403],
    [{ host: GLUE_HOST, binding: GLUE_BINDING }, {}, 502],
    [{ host: GLUE_HOST, binding: GLUE_BINDING }, { [GLUE_BINDING]: { fetch() { throw new Error("down"); } } }, 502],
  ];
  for (const [props, env, status] of cases) {
    const response = await withoutGlobalFetch(() => routeInternal(
      new Request(`https://${GLUE_HOST}/api/v1/upstream/ping`, { headers: { "x-internal-target": "https://evil.example" } }),
      env,
      props,
      (target, request, init) => target.fetch(request, init),
    ));
    assert.equal(response.status, status);
    assert.equal(response.headers.get("cache-control"), "no-store");
    if (status === 502) assert.equal(await response.text(), "internal channel failed");
  }
  const unavailable = unavailableResponse();
  assert.equal(unavailable.status, 503);
  assert.equal(unavailable.headers.get("cache-control"), "no-store");
  assert.equal(await unavailable.text(), "internal channel unavailable");
  const failed = failedResponse();
  assert.equal(failed.status, 502);
});

function glueEnvBinding() {
  return { [GLUE_BINDING]: { fetch: () => new Response("ok") } };
}

test("a streamed body is forwarded unconsumed and cancellation is visible", async () => {
  const body = new ReadableStream({
    start(controller) {
      controller.enqueue(new TextEncoder().encode("order-json"));
      controller.close();
    },
  });
  let forwarded;
  await routeInternal(
    new Request(`https://${GLUE_HOST}/api/v1/upstream/orders`, { method: "POST", body, duplex: "half" }),
    glueEnvBinding(),
    { host: GLUE_HOST, binding: GLUE_BINDING },
    (target, request, init) => {
      forwarded = request;
      return target.fetch(request, init);
    },
  );
  assert.equal(await forwarded.text(), "order-json");

  const controller = new AbortController();
  const request = new Request(`https://${GLUE_HOST}/api/v1/upstream/orders`, { method: "POST", body: "{}", signal: controller.signal });
  controller.abort();
  let signal;
  await routeInternal(
    request,
    glueEnvBinding(),
    { host: GLUE_HOST, binding: GLUE_BINDING },
    (target, forwardedRequest, init) => {
      signal = forwardedRequest.signal;
      return target.fetch(forwardedRequest, init);
    },
  );
  assert.equal(signal.aborted, true);

  const stream = new ReadableStream({
    start(controller) {
      controller.enqueue(new TextEncoder().encode("data: hi\n\n"));
      controller.close();
    },
  });
  const response = await routeInternal(
    new Request(`https://${GLUE_HOST}/api/v1/upstream/orders`),
    { [GLUE_BINDING]: { fetch: () => new Response(stream, { headers: { "content-type": "text/event-stream" } }) } },
    { host: GLUE_HOST, binding: GLUE_BINDING },
    (target, forwardedRequest, init) => target.fetch(forwardedRequest, init),
  );
  assert.equal(response.headers.get("content-type"), "text/event-stream");
  assert.equal(await response.text(), "data: hi\n\n");
});

test("the trust entry concatenates CAs and does not exec when a CA is unreadable", () => {
  const script = trustScript();
  assert.match(script, new RegExp(SYSTEM_CA_BUNDLE));
  assert.match(script, new RegExp(SYSTEM_CA_FALLBACK));
  assert.match(script, new RegExp(CLOUDFLARE_CONTAINER_CA));
  assert.match(script, new RegExp(`SSL_CERT_FILE=${TRUSTED_CA_FILE.replaceAll("/", "\\/")}|SSL_CERT_FILE='${TRUSTED_CA_FILE}'`));
  assert.match(script, /exec '\/app\/dujiao-next' '-mode=api'/);
  assert.equal(script.includes("update-ca-certificates"), false);
  assert.equal(script.includes("unset SSL_CERT_DIR"), false);
  assert.deepEqual(trustEntrypoint().slice(0, 2), ["sh", "-c"]);
  assert.equal(trustEntrypoint()[2], script);
  assert.deepEqual(START_COMMAND, ["/app/dujiao-next", "-mode=api"]);

  const dir = mkdtempSync(join(tmpdir(), "needtoken-ca-"));
  try {
    const bundle = join(dir, "ca-certificates.crt");
    const fallback = join(dir, "cert.pem");
    const cloudflare = join(dir, "cloudflare-containers-ca.crt");
    const out = join(dir, "needtoken-ca.pem");
    const marker = join(dir, "started");
    const bin = join(dir, "dujiao-next");
    writeFileSync(bundle, "SYSTEM\n");
    writeFileSync(fallback, "FALLBACK\n");
    writeFileSync(cloudflare, "CLOUDFLARE\n");
    writeFileSync(bin, `#!/bin/sh\nprintf '%s %s\\n' "$SSL_CERT_FILE" "$1" > ${JSON.stringify(marker)}\n`);
    chmodSync(bin, 0o755);

    const ok = spawnSync("sh", ["-c", trustScript({ bundle, fallback, cloudflare, out, argv: [bin, "-mode=api"] })], { encoding: "utf8" });
    assert.equal(ok.status, 0, ok.stderr);
    assert.equal(readFileSync(out, "utf8"), "SYSTEM\nCLOUDFLARE\n");
    assert.equal(readFileSync(marker, "utf8"), `${out} -mode=api\n`);

    rmSync(bundle);
    rmSync(out);
    rmSync(marker);
    const usedFallback = spawnSync("sh", ["-c", trustScript({ bundle, fallback, cloudflare, out, argv: [bin, "-mode=api"] })], { encoding: "utf8" });
    assert.equal(usedFallback.status, 0, usedFallback.stderr);
    assert.equal(readFileSync(out, "utf8"), "FALLBACK\nCLOUDFLARE\n");

    rmSync(marker);
    const missing = spawnSync("sh", ["-c", trustScript({
      bundle,
      fallback: join(dir, "missing-fallback.pem"),
      cloudflare: join(dir, "missing-cloudflare.crt"),
      out,
      argv: [bin, "-mode=api"],
    })], { encoding: "utf8" });
    assert.notEqual(missing.status, 0);
    assert.equal(existsSync(marker), false);
  } finally {
    rmSync(dir, { recursive: true, force: true });
  }
});
