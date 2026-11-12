import assert from "node:assert/strict";
import test from "node:test";

import { FORCED_ENV, INSTANCE, OPTIONAL_BINDINGS, REQUIRED_BINDINGS, startPlan, WORKER_ONLY_BINDINGS } from "./policy.js";

function env(overrides = {}) {
  return {
    CLOUDFLARE_IMAGE_DIGEST: `sha256:${"ab".repeat(32)}`,
    APP_SECRET_KEY: "app",
    JWT_SECRET: "jwt",
    USER_JWT_SECRET: "user-jwt",
    DATABASE_DSN: "postgres://store@db.example/storefront",
    REDIS_HOST: "redis.example",
    QUEUE_HOST: "redis.example",
    CLOUDFLARE_OBJECT_STORAGE_ENDPOINT: "https://example.r2.cloudflarestorage.com",
    CLOUDFLARE_OBJECT_STORAGE_BUCKET: "storefront-uploads",
    CLOUDFLARE_OBJECT_STORAGE_ACCESS_KEY_ID: "access",
    CLOUDFLARE_OBJECT_STORAGE_SECRET_ACCESS_KEY: "secret",
    CLOUDFLARE_CONTROL_TOKEN: "worker-only",
    DATABASE_DRIVER: "sqlite",
    QUEUE_ENABLED: "false",
    ...overrides,
  };
}

test("start plan runs all mode and refuses sqlite or a missing target", () => {
  const plan = startPlan(env(), "fence-1");
  assert.equal(plan.ok, true);
  assert.deepEqual(plan.options.entrypoint, ["./dujiao-next", "-mode", "all"]);
  assert.equal(plan.options.enableInternet, true);
  assert.equal(plan.options.instance, INSTANCE);
  assert.equal(plan.options.instance, "standard-1");
  assert.equal(plan.options.env.DATABASE_DRIVER, "postgres");
  assert.equal(plan.options.env.QUEUE_ENABLED, "true");
  assert.equal(plan.options.env.CLOUDFLARE_ENABLED, "true");
  assert.equal(plan.options.env.CLOUDFLARE_ISOLATION, "isolated");
  assert.equal(plan.options.env.SERVER_HOST, "0.0.0.0");
  assert.equal(plan.options.env.SERVER_PORT, "8080");
  assert.equal(plan.options.env.CLOUDFLARE_WRITER_FENCE, "fence-1");
  assert.equal(plan.options.env.CLOUDFLARE_CONTROL_TOKEN, undefined);
  assert.deepEqual(plan.options.env, {
    ...Object.fromEntries(REQUIRED_BINDINGS.map((name) => [name, env()[name]])),
    CLOUDFLARE_WRITER_FENCE: "fence-1",
    ...FORCED_ENV,
  });

  const sqlite = startPlan(env({ DATABASE_DSN: "file:./db/dujiao.db" }), "fence-1");
  assert.equal(sqlite.ok, false);
  assert.equal(sqlite.options, undefined);

  const missing = startPlan(env({ CLOUDFLARE_OBJECT_STORAGE_BUCKET: " " }), "fence-1");
  assert.equal(missing.ok, false);
  assert.deepEqual(missing.missing, ["CLOUDFLARE_OBJECT_STORAGE_BUCKET"]);

  const noFence = startPlan(env(), "");
  assert.equal(noFence.ok, false);

  const placeholder = startPlan(env({ CLOUDFLARE_IMAGE_DIGEST: `sha256:${"0".repeat(64)}` }), "fence-1");
  assert.equal(placeholder.ok, false);
  assert.equal(placeholder.options, undefined);
});

test("optional bindings are copied and worker-only bindings stay out", () => {
  const plan = startPlan(env({ REDIS_PASSWORD: "redis-secret", REDIS_PORT: "6379" }), "fence-2");
  assert.equal(plan.options.env.REDIS_PASSWORD, "redis-secret");
  assert.equal(plan.options.env.REDIS_PORT, "6379");
  for (const name of WORKER_ONLY_BINDINGS) {
    assert.equal(plan.options.env[name], undefined);
  }
  assert.ok(OPTIONAL_BINDINGS.includes("REDIS_PASSWORD"));
});
