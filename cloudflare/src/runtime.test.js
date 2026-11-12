import assert from "node:assert/strict";
import { existsSync, readFileSync } from "node:fs";
import { dirname, join } from "node:path";
import test from "node:test";
import { fileURLToPath } from "node:url";

import {
  APP_PORT,
  ContainerNotReady,
  INSTANCE_TYPE,
  INACTIVITY_TIMEOUT_MS,
  MAX_INSTANCES,
  NOT_READY_RETRY_AFTER_SECONDS,
  READY_DEADLINE_MS,
  RESIDENT_CRON,
  RESIDENT_INSTANCE,
  START_COMMAND,
  containerEnv,
  forwardRequest,
  keepResident,
  notReadyResponse,
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

test("the worker source uses the runtime values and does not restrict hosts", () => {
  const source = readFileSync(join(root, "src/index.js"), "utf8");
  assert.equal(source.includes("container.images"), false);
  assert.equal(source.includes("memoryMib"), false);
  assert.match(source, /entrypoint: START_COMMAND/);
  assert.match(source, /env: containerEnv\(this\.env\)/);
  assert.match(source, /getTcpPort\(APP_PORT\)/);
  assert.match(source, /forwardRequest\(request\)/);
  assert.match(source, /enableInternet: true/);
  assert.match(source, /getByName\(RESIDENT_INSTANCE\)/);
  assert.match(source, /async scheduled\(_controller, env\)/);
  assert.match(source, /keepResident\(env\.STOREFRONT\)/);
  assert.match(source, /error instanceof ContainerNotReady/);
  assert.match(source, /return notReadyResponse\(\)/);
  assert.match(source, /waitForReady\(/);
  assert.equal(source.includes("instanceName"), false);
  assert.equal(source.includes("instance-1"), false);
  assert.equal(source.includes("needtoken.ai"), false);
  assert.equal(source.includes("staging"), false);
  assert.equal(source.includes("dockerfile"), false);
  assert.equal(source.includes("REDIS_PREFIX"), false);
  assert.equal(source.includes("OPS_WRITES_ENABLED"), false);
});
