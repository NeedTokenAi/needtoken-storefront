// Durable Object scheduling policy, read 2026-11-12:
// public beta, immutable, and no max_instances. Omitting `instance` selects
// lite. The runtime rejects `basic`. `standard-1` is the documented named
// size (1/2 vCPU, 4 GiB memory, 8 GB disk). It is unverified against this
// process's live memory. Disk is ephemeral. Snapshots are not a live volume.
// These labels describe this skeleton. They are not a live Cloudflare run.
export const LABELS = Object.freeze({
  mock: true,
  notOnARealAccount: true,
  notASixHourRun: true,
  durableObjectBeta: true,
  rollback: false,
  stopsOldCloudRunWriter: false,
});

export const ISOLATED_OBJECT_BUCKET = "needtoken-staging";

export const SINGLETON_NAME = "dujiao-storefront";
export const CONTAINER_PORT = 8080;
export const IMAGE_NAME = "dujiao";
export const PREVIOUS_IMAGE_NAME = "dujiao-previous";
export const INSTANCE = "standard-1";
export const INACTIVITY_TIMEOUT_MS = 6 * 60 * 60 * 1000;
export const SIGTERM = 15;

export const FORCED_ENV = Object.freeze({
  CLOUDFLARE_ENABLED: "true",
  // isolated marks this process as a candidate. It does not stop Cloud Run.
  CLOUDFLARE_ISOLATION: "isolated",
  DATABASE_DRIVER: "postgres",
  QUEUE_ENABLED: "true",
  SERVER_HOST: "0.0.0.0",
  SERVER_PORT: String(CONTAINER_PORT),
});

export const REQUIRED_BINDINGS = Object.freeze([
  "CLOUDFLARE_IMAGE_DIGEST",
  "CLOUDFLARE_BUILD_REPOSITORY",
  "CLOUDFLARE_BUILD_SHA",
  "CLOUDFLARE_BUILD_RUN_ID",
  "CLOUDFLARE_BUILD_SOURCE",
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
  "REDIS_KEY_PREFIX",
  "CLOUDFLARE_BACKGROUND_CONSUMERS",
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

export function fullRegistryRef(value) {
  const ref = String(value ?? "").trim();
  return /^[a-z0-9][a-z0-9._:-]*(?:\/[a-z0-9][a-z0-9._:-]*)+@sha256:[0-9a-f]{64}$/.test(ref);
}

function redisKeyPrefix(value) {
  return String(value ?? "").trim().replace(/:$/, "");
}

// A build record is the workflow's repository, commit, run id, and full
// registry ref. A bare sha256 or a JSON document is not that record.
export function buildRecordFromEnv(env) {
  const source = env ?? {};
  if (present(source.CLOUDFLARE_BUILD_RECORD_JSON) || typeof source === "string") {
    return { ok: false, reason: "hand-filled JSON is not a build record" };
  }
  if (String(source.CLOUDFLARE_BUILD_SOURCE ?? "").trim() !== "github-actions") {
    return { ok: false, reason: "hand-filled SHA or JSON is not a build record" };
  }
  const repository = String(source.CLOUDFLARE_BUILD_REPOSITORY ?? "").trim();
  const sha = String(source.CLOUDFLARE_BUILD_SHA ?? "").trim();
  const runId = String(source.CLOUDFLARE_BUILD_RUN_ID ?? "").trim();
  const digest = String(source.CLOUDFLARE_IMAGE_DIGEST ?? "").trim();
  if (!/^[^/\s]+\/[^/\s]+$/.test(repository)) {
    return { ok: false, reason: "build repository is missing" };
  }
  if (!/^[0-9a-f]{40}$/.test(sha)) {
    return { ok: false, reason: "build sha is missing" };
  }
  if (!/^[1-9][0-9]*$/.test(runId)) {
    return { ok: false, reason: "build run id is missing" };
  }
  if (!fullRegistryRef(digest)) {
    return { ok: false, reason: "digest must be the full registry ref of ctx.container.images.dujiao" };
  }
  return { ok: true, record: { repository, sha, runId, digest, source: "github-actions" } };
}

export function acceptBuildRecord(record) {
  if (typeof record === "string" || record?.format === "json" || record?.source === "repository-json" || record?.source === "manual") {
    return { ok: false, reason: "hand-filled JSON is not a build record" };
  }
  if (!record || record.builder !== "github-actions") {
    return { ok: false, reason: "hand-filled SHA or JSON is not a build record" };
  }
  return buildRecordFromEnv({
    CLOUDFLARE_BUILD_SOURCE: "github-actions",
    CLOUDFLARE_BUILD_REPOSITORY: record.repository,
    CLOUDFLARE_BUILD_SHA: record.sha,
    CLOUDFLARE_BUILD_RUN_ID: record.runId,
    CLOUDFLARE_IMAGE_DIGEST: record.digest,
  });
}

export function acceptImage(containerImage, env) {
  const built = buildRecordFromEnv(env);
  if (!built.ok) {
    return built;
  }
  if (String(containerImage ?? "").trim() !== built.record.digest) {
    return { ok: false, reason: "digest must equal ctx.container.images.dujiao" };
  }
  return { ok: true, image: built.record.digest };
}

function sqliteDSN(value) {
  const dsn = String(value).trim().toLowerCase();
  return dsn.includes("sqlite") || dsn.startsWith("file:") || dsn.endsWith(".db") || dsn.includes(".db?");
}

// startPlan returns the container start options, or a refusal. A refusal must
// not be passed to start(). imageRef must be ctx.container.images.dujiao.
export function startPlan(env, fence, imageRef) {
  const source = env ?? {};
  const missing = REQUIRED_BINDINGS.filter((name) => !present(source[name]));
  if (missing.length > 0) {
    return { ok: false, reason: "required binding missing", missing };
  }
  const image = acceptImage(imageRef, source);
  if (!image.ok) {
    return image;
  }
  if (sqliteDSN(source.DATABASE_DSN)) {
    return { ok: false, reason: "sqlite database is refused" };
  }
  if (String(source.CLOUDFLARE_OBJECT_STORAGE_BUCKET ?? "").trim() !== ISOLATED_OBJECT_BUCKET) {
    return { ok: false, reason: "object bucket is outside the isolated allowlist" };
  }
  if (redisKeyPrefix(source.REDIS_KEY_PREFIX) !== "staging") {
    return { ok: false, reason: "REDIS_KEY_PREFIX must be staging:" };
  }
  if (String(source.CLOUDFLARE_BACKGROUND_CONSUMERS ?? "").trim().toLowerCase() !== "off") {
    return { ok: false, reason: "background consumers must stay off so this process does not consume production queues or schedules" };
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
      image: image.image,
      entrypoint: [...ENTRYPOINT],
      enableInternet: true,
      instance: INSTANCE,
      env: containerEnv,
    },
  };
}
