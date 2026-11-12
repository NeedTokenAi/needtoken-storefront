import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { dirname, join } from "node:path";
import test from "node:test";
import { fileURLToPath } from "node:url";

import {
  APP_PORT,
  INSTANCE,
  START_COMMAND,
  containerEnv,
  forwardRequest,
} from "./runtime.js";

const root = dirname(dirname(fileURLToPath(import.meta.url)));

test("production worker serves workers.dev and does not build an image", () => {
  const wrangler = JSON.parse(readFileSync(join(root, "wrangler.jsonc"), "utf8"));
  const container = wrangler.containers[0];
  assert.equal(wrangler.name, "needtoken-storefront");
  assert.equal(wrangler.workers_dev, true);
  assert.equal(wrangler.preview_urls, false);
  assert.equal(wrangler.routes, undefined);
  assert.equal(JSON.stringify(wrangler).includes("needtoken.ai"), false);
  assert.equal(JSON.stringify(wrangler).toLowerCase().includes("staging"), false);
  assert.equal(container.max_instances, 2);
  assert.equal(container.scheduling_policy, "durable_object");
  assert.equal(container.images.app.image, "${CLOUDFLARE_IMAGE_DIGEST}");
  assert.equal(Object.hasOwn(container.images.app, "dockerfile"), false);
  assert.equal(Object.hasOwn(container.images.app, "build_context"), false);
  assert.equal(JSON.stringify(wrangler).includes("Dockerfile"), false);
  assert.equal(wrangler.vars, undefined);
  assert.equal(JSON.stringify(wrangler).includes("REDIS_PREFIX"), false);
  assert.equal(JSON.stringify(wrangler).includes("OPS_WRITES_ENABLED"), false);
});

test("start command, port, and instance size stay fixed", () => {
  assert.equal(APP_PORT, 8080);
  assert.deepEqual(START_COMMAND, ["/app/dujiao-next", "-mode=api"]);
  assert.equal(START_COMMAND.join(" "), "/app/dujiao-next -mode=api");
  assert.equal(INSTANCE.vcpu, 1);
  assert.equal(INSTANCE.memoryMib, 512);
  assert.equal(INSTANCE.diskMb, 2000);
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
  });
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

test("the worker source uses the runtime values and does not restrict hosts", () => {
  const source = readFileSync(join(root, "src/index.js"), "utf8");
  assert.match(source, /image: container\.images\.app/);
  assert.match(source, /instance: INSTANCE/);
  assert.match(source, /entrypoint: START_COMMAND/);
  assert.match(source, /env: containerEnv\(this\.env\)/);
  assert.match(source, /getTcpPort\(APP_PORT\)/);
  assert.match(source, /forwardRequest\(request\)/);
  assert.match(source, /enableInternet: true/);
  assert.match(source, /getByName\(SINGLETON_NAME\)/);
  assert.equal(source.includes("needtoken.ai"), false);
  assert.equal(source.includes("staging"), false);
  assert.equal(source.includes("dockerfile"), false);
  assert.equal(source.includes("REDIS_PREFIX"), false);
  assert.equal(source.includes("OPS_WRITES_ENABLED"), false);
});
