// Durable Object scheduling policy, read 2026-11-12:
// public beta, immutable, and no max_instances. Omitting `instance` selects
// lite. The runtime rejects `basic`. `standard-1` is the documented named
// size (1/2 vCPU, 4 GiB memory, 8 GB disk). It is unverified against this
// process's live memory. Disk is ephemeral. Snapshots are not a live volume.
export const SINGLETON_NAME = "dujiao-storefront";
export const CONTAINER_PORT = 8080;
export const IMAGE_NAME = "dujiao";
export const INSTANCE = "standard-1";
export const INACTIVITY_TIMEOUT_MS = 6 * 60 * 60 * 1000;
export const SIGTERM = 15;

export const FORCED_ENV = Object.freeze({
  CLOUDFLARE_ENABLED: "true",
  DATABASE_DRIVER: "postgres",
  QUEUE_ENABLED: "true",
  SERVER_HOST: "0.0.0.0",
  SERVER_PORT: String(CONTAINER_PORT),
});

export const REQUIRED_BINDINGS = Object.freeze([
  "APP_SECRET_KEY",
  "JWT_SECRET",
  "USER_JWT_SECRET",
  "DATABASE_DSN",
  "REDIS_HOST",
  "QUEUE_HOST",
  "CLOUDFLARE_OBJECT_STORAGE_ENDPOINT",
  "CLOUDFLARE_OBJECT_STORAGE_BUCKET",
  "CLOUDFLARE_OBJECT_STORAGE_ACCESS_KEY_ID",
  "CLOUDFLARE_OBJECT_STORAGE_SECRET_ACCESS_KEY",
]);

export const OPTIONAL_BINDINGS = Object.freeze([
  "REDIS_PASSWORD",
  "REDIS_PORT",
  "REDIS_DB",
  "REDIS_PREFIX",
  "QUEUE_PASSWORD",
  "QUEUE_PORT",
  "QUEUE_DB",
  "QUEUE_CONCURRENCY",
  "BOOTSTRAP_DEFAULT_ADMIN_PASSWORD",
  "TELEGRAM_AUTH_BOT_TOKEN",
  "TELEGRAM_AUTH_CLIENT_SECRET",
  "EMAIL_PASSWORD",
  "CAPTCHA_TURNSTILE_SECRET_KEY",
  "CLOUDFLARE_OBJECT_STORAGE_REGION",
  "CLOUDFLARE_OBJECT_STORAGE_PREFIX",
]);

export const WORKER_ONLY_BINDINGS = Object.freeze(["CLOUDFLARE_CONTROL_TOKEN"]);

const ENTRYPOINT = Object.freeze(["./dujiao-next", "-mode", "all"]);

function present(value) {
  return value != null && String(value).trim() !== "";
}

function sqliteDSN(value) {
  const dsn = String(value).trim().toLowerCase();
  return dsn.includes("sqlite") || dsn.startsWith("file:") || dsn.endsWith(".db") || dsn.includes(".db?");
}

// startPlan returns the container start options, or a refusal. A refusal must
// not be passed to start(). The image reference is attached by the Durable
// Object from ctx.container.images.dujiao.
export function startPlan(env, fence) {
  const source = env ?? {};
  const missing = REQUIRED_BINDINGS.filter((name) => !present(source[name]));
  if (missing.length > 0) {
    return { ok: false, reason: "required binding missing", missing };
  }
  if (sqliteDSN(source.DATABASE_DSN)) {
    return { ok: false, reason: "sqlite database is refused" };
  }
  if (!present(fence)) {
    return { ok: false, reason: "writer fence is missing" };
  }

  const containerEnv = { CLOUDFLARE_WRITER_FENCE: String(fence).trim() };
  for (const name of [...REQUIRED_BINDINGS, ...OPTIONAL_BINDINGS]) {
    if (present(source[name])) {
      containerEnv[name] = String(source[name]);
    }
  }
  Object.assign(containerEnv, FORCED_ENV);
  for (const name of WORKER_ONLY_BINDINGS) {
    delete containerEnv[name];
  }

  return {
    ok: true,
    options: {
      entrypoint: [...ENTRYPOINT],
      enableInternet: true,
      instance: INSTANCE,
      env: containerEnv,
    },
  };
}
