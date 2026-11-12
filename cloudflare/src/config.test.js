import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { dirname, join } from "node:path";
import test from "node:test";
import { fileURLToPath } from "node:url";

import { OPTIONAL_BINDINGS, REQUIRED_BINDINGS, WORKER_ONLY_BINDINGS } from "./policy.js";

const root = dirname(dirname(fileURLToPath(import.meta.url)));

test("wrangler uses the durable object policy without an instance cap", () => {
  const wrangler = JSON.parse(readFileSync(join(root, "wrangler.jsonc"), "utf8"));
  const container = wrangler.containers[0];
  assert.equal(container.scheduling_policy, "durable_object");
  assert.equal(container.class_name, "DujiaoStorefront");
  assert.equal(Object.hasOwn(container, "max_instances"), false);
  assert.equal(Object.hasOwn(container, "instance_type"), false);
  assert.equal(Object.hasOwn(container, "image"), false);
  assert.equal(container.images.dujiao.dockerfile, "../Dockerfile");
  assert.equal(container.images["dujiao-previous"].dockerfile, "../Dockerfile");
  assert.equal(wrangler.durable_objects.bindings[0].name, "STOREFRONT");
  assert.equal(wrangler.exports.DujiaoStorefront.storage, "sqlite");
  assert.equal(JSON.stringify(wrangler).includes("max_instances"), false);
  assert.equal(wrangler.workers_dev, false);
});

test("secret bindings are names only and match the start allowlist", () => {
  const document = JSON.parse(readFileSync(join(root, "secret-bindings.json"), "utf8"));
  assert.equal(document.version, "unverified");
  const containerEnv = new Set();
  const workerOnly = new Set();
  for (const binding of document.bindings) {
    assert.equal(binding.version, "unverified");
    assert.equal(Object.hasOwn(binding, "value"), false);
    assert.equal(JSON.stringify(binding).includes("postgres://"), false);
    if (binding.generated_by === "durable_object") {
      assert.equal(binding.secret, "CLOUDFLARE_WRITER_FENCE");
      assert.equal(binding.binding, null);
      assert.equal(binding.container_env, "CLOUDFLARE_WRITER_FENCE");
      continue;
    }
    assert.equal(binding.binding, binding.secret);
    if (binding.container_env == null) {
      workerOnly.add(binding.secret);
      continue;
    }
    assert.equal(binding.container_env, binding.secret);
    containerEnv.add(binding.secret);
  }
  for (const name of REQUIRED_BINDINGS) {
    assert.equal(containerEnv.has(name), true, name);
  }
  for (const name of OPTIONAL_BINDINGS) {
    if (name.endsWith("_PASSWORD") || name.endsWith("_SECRET") || name.endsWith("_TOKEN") || name.endsWith("_KEY") || name.includes("SECRET") || name.endsWith("_DSN")) {
      assert.equal(containerEnv.has(name), true, name);
    }
  }
  assert.deepEqual([...workerOnly], [...WORKER_ONLY_BINDINGS]);
});

test("the worker source stays a singleton and does not choose the deprecated policy", () => {
  const source = readFileSync(join(root, "src/index.js"), "utf8");
  assert.match(source, /from "cloudflare:workers"/);
  assert.match(source, /getByName\(SINGLETON_NAME\)/);
  assert.match(source, /schedulerWait/);
  assert.match(source, /containerRequest/);
  assert.equal(source.includes("setTimeout"), false);
  assert.equal(source.includes("max_instances"), false);
  assert.equal(source.includes("scheduling_policy"), false);
  assert.equal(source.includes("basic"), false);
  assert.match(source, /setInactivityTimeout\(INACTIVITY_TIMEOUT_MS\)/);
  assert.match(source, /rollback is false/);
});
