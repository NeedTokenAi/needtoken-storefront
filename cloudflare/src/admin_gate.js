const MODULE_NAME = "storefront";

export const EDGE_ADMIN_HEADER = "x-needtoken-internal-admin-secret";
export const EDGE_ADMIN_COOKIE = "__Host-nt-edge-admin";
export const EDGE_CONTROL_PREFIX = "/__needtoken_edge";
export const MIN_SECRET_LENGTH = 32;
export const TICKET_TTL_SECONDS = 60;
export const MIN_SESSION_TTL_MINUTES = 5;
export const DEFAULT_SESSION_TTL_MINUTES = 30;
export const MAX_SESSION_TTL_MINUTES = 120;
export const MAX_ACTIVE_SESSIONS = 20;
export const MAX_PENDING_TICKETS = 20;

const EDGE_ONLY_ENV_KEYS = new Set([
  "EDGE_ADMIN_SECRET",
  "DISABLE_PUBLIC_ADMIN",
  "EDGE_ADMIN_ENFORCE_API",
]);

const textEncoder = new TextEncoder();

export function isEdgeOnlyEnvKey(key) {
  return EDGE_ONLY_ENV_KEYS.has(key);
}

export function stripEdgeGateHeaders(headers) {
  headers.delete(EDGE_ADMIN_HEADER);
}

function notFoundResponse() {
  return new Response("Not Found", {
    status: 404,
    headers: {
      "content-type": "text/plain; charset=utf-8",
      "cache-control": "no-store",
      "referrer-policy": "no-referrer",
      "x-robots-tag": "noindex, nofollow",
    },
  });
}

function jsonResponse(status, body) {
  return new Response(JSON.stringify(body), {
    status,
    headers: {
      "content-type": "application/json; charset=utf-8",
      "cache-control": "no-store",
      "referrer-policy": "no-referrer",
    },
  });
}

function toBase64Url(bytes) {
  let binary = "";
  for (let i = 0; i < bytes.length; i += 1) {
    binary += String.fromCharCode(bytes[i]);
  }
  return btoa(binary).replace(/\+/g, "-").replace(/\//g, "_").replace(/=+$/g, "");
}

function toHex(bytes) {
  let out = "";
  for (let i = 0; i < bytes.length; i += 1) {
    out += bytes[i].toString(16).padStart(2, "0");
  }
  return out;
}

function randomToken(byteLength = 32) {
  const bytes = new Uint8Array(byteLength);
  crypto.getRandomValues(bytes);
  return toBase64Url(bytes);
}

async function sha256Bytes(value) {
  const digest = await crypto.subtle.digest("SHA-256", textEncoder.encode(value));
  return new Uint8Array(digest);
}

async function sha256Hex(value) {
  return toHex(await sha256Bytes(value));
}

async function timingSafeStringEqual(a, b) {
  if (typeof a !== "string" || typeof b !== "string") return false;
  const [da, db] = await Promise.all([sha256Bytes(a), sha256Bytes(b)]);
  let diff = 0;
  for (let i = 0; i < da.length; i += 1) {
    diff |= da[i] ^ db[i];
  }
  return diff === 0 && a.length === b.length;
}

async function signSessionId(secret, sessionId) {
  const key = await crypto.subtle.importKey(
    "raw",
    textEncoder.encode(secret),
    { name: "HMAC", hash: "SHA-256" },
    false,
    ["sign"],
  );
  const sig = await crypto.subtle.sign(
    "HMAC",
    key,
    textEncoder.encode(`nt-edge-admin-v1:${MODULE_NAME}:${sessionId}`),
  );
  return toBase64Url(new Uint8Array(sig));
}

function getConfiguredSecret(env) {
  const raw = typeof env?.EDGE_ADMIN_SECRET === "string" ? env.EDGE_ADMIN_SECRET.trim() : "";
  return raw.length >= MIN_SECRET_LENGTH ? raw : "";
}

export function isPublicAdminDisabled(env) {
  return env?.DISABLE_PUBLIC_ADMIN !== "0";
}

export function resolveAdminEntryPath(env) {
  const raw = typeof env?.WEB_ADMIN_PATH === "string" ? env.WEB_ADMIN_PATH.trim() : "";
  if (!raw || !raw.startsWith("/") || raw === "/" || raw.endsWith("/") || raw.includes("..") || raw.includes("\\")) {
    return "/admin";
  }
  if (!/^\/[A-Za-z0-9._~@/-]+$/.test(raw)) {
    return "/admin";
  }
  return raw.replace(/\/{2,}/g, "/");
}

// Fail-closed URL path normalization to prevent encoding, case, double-slash,
// backslash, null-byte, or dot-segment bypasses.
export function normalizeRequestPath(rawPathname) {
  if (typeof rawPathname !== "string" || !rawPathname.startsWith("/")) {
    return { ok: false, canonical: "", lower: "", hasTrailingSlash: false };
  }
  if (/[\u0000\\]/.test(rawPathname) || /%(?:00|2f|5c)/i.test(rawPathname)) {
    return { ok: false, canonical: "", lower: "", hasTrailingSlash: false };
  }
  let decoded;
  try {
    decoded = decodeURIComponent(rawPathname);
  } catch {
    return { ok: false, canonical: "", lower: "", hasTrailingSlash: false };
  }
  if (/[\u0000\\]/.test(decoded)) {
    return { ok: false, canonical: "", lower: "", hasTrailingSlash: false };
  }
  const collapsed = decoded.replace(/\/{2,}/g, "/");
  const segments = collapsed.split("/");
  for (const seg of segments) {
    if (seg === "." || seg === "..") {
      return { ok: false, canonical: "", lower: "", hasTrailingSlash: false };
    }
  }
  const hasTrailingSlash = collapsed.length > 1 && collapsed.endsWith("/");
  const canonical = hasTrailingSlash ? collapsed.slice(0, -1) : collapsed;
  return {
    ok: true,
    canonical,
    lower: canonical.toLowerCase(),
    hasTrailingSlash,
  };
}

function matchesPrefix(lowerPath, prefix) {
  const normalizedPrefix = prefix.toLowerCase();
  return lowerPath === normalizedPrefix || lowerPath.startsWith(`${normalizedPrefix}/`);
}

export function classifyStorefrontRequest(method, normalized, env) {
  if (!normalized.ok) return "invalid";
  const path = normalized.lower;
  if (matchesPrefix(path, EDGE_CONTROL_PREFIX)) return "control";
  const entry = resolveAdminEntryPath(env);
  if (matchesPrefix(path, "/admin") || matchesPrefix(path, entry) || matchesPrefix(path, "/api/v1/admin")) {
    return "admin_only";
  }
  return "public";
}

function parseCookieValue(cookieHeader, cookieName) {
  if (typeof cookieHeader !== "string" || cookieHeader.length === 0) return "";
  const parts = cookieHeader.split(";");
  for (const part of parts) {
    const idx = part.indexOf("=");
    if (idx <= 0) continue;
    const name = part.slice(0, idx).trim();
    if (name === cookieName) {
      return part.slice(idx + 1).trim();
    }
  }
  return "";
}

class GateStore {
  constructor(sql) {
    this.sql = sql;
    this.initialized = false;
  }

  ensureSchema() {
    if (this.initialized || !this.sql) return;
    this.sql.exec(
      `CREATE TABLE IF NOT EXISTS nt_edge_admin_tickets (
        token_hash TEXT PRIMARY KEY,
        actor TEXT NOT NULL,
        reason TEXT NOT NULL,
        ttl_seconds INTEGER NOT NULL,
        created_at INTEGER NOT NULL,
        expires_at INTEGER NOT NULL
      )`,
    );
    this.sql.exec(
      `CREATE TABLE IF NOT EXISTS nt_edge_admin_sessions (
        session_id TEXT PRIMARY KEY,
        actor TEXT NOT NULL,
        reason TEXT NOT NULL,
        created_at INTEGER NOT NULL,
        expires_at INTEGER NOT NULL
      )`,
    );
    this.initialized = true;
  }

  prune(nowMs) {
    if (!this.sql) return;
    this.ensureSchema();
    this.sql.exec("DELETE FROM nt_edge_admin_tickets WHERE expires_at <= ?", nowMs);
    this.sql.exec("DELETE FROM nt_edge_admin_sessions WHERE expires_at <= ?", nowMs);
  }

  listSessions(nowMs) {
    if (!this.sql) return [];
    this.prune(nowMs);
    return this.sql
      .exec(
        "SELECT session_id, actor, reason, created_at, expires_at FROM nt_edge_admin_sessions WHERE expires_at > ? ORDER BY created_at DESC",
        nowMs,
      )
      .toArray();
  }

  countTickets(nowMs) {
    if (!this.sql) return 0;
    this.prune(nowMs);
    const rows = this.sql
      .exec("SELECT COUNT(*) AS cnt FROM nt_edge_admin_tickets WHERE expires_at > ?", nowMs)
      .toArray();
    return Number(rows[0]?.cnt ?? 0);
  }

  hasSession(sessionId, nowMs) {
    if (!this.sql) return false;
    this.prune(nowMs);
    const rows = this.sql
      .exec("SELECT session_id FROM nt_edge_admin_sessions WHERE session_id = ? AND expires_at > ?", sessionId, nowMs)
      .toArray();
    return rows.length > 0;
  }

  insertTicket({ tokenHash, actor, reason, ttlSeconds, createdAt, expiresAt }) {
    if (!this.sql) throw new Error("gate_storage_unavailable");
    this.prune(createdAt);
    const existing = this.countTickets(createdAt);
    if (existing >= MAX_PENDING_TICKETS) {
      this.sql.exec(
        "DELETE FROM nt_edge_admin_tickets WHERE token_hash IN (SELECT token_hash FROM nt_edge_admin_tickets ORDER BY created_at ASC LIMIT ?)",
        existing - MAX_PENDING_TICKETS + 1,
      );
    }
    this.sql.exec(
      "INSERT INTO nt_edge_admin_tickets (token_hash, actor, reason, ttl_seconds, created_at, expires_at) VALUES (?, ?, ?, ?, ?, ?)",
      tokenHash,
      actor,
      reason,
      ttlSeconds,
      createdAt,
      expiresAt,
    );
  }

  // Synchronous SELECT + immediate DELETE guarantees single-use anti-replay inside the DO.
  consumeTicket(tokenHash, nowMs) {
    if (!this.sql) return null;
    this.prune(nowMs);
    const rows = this.sql
      .exec(
        "SELECT token_hash, actor, reason, ttl_seconds, expires_at FROM nt_edge_admin_tickets WHERE token_hash = ? AND expires_at > ?",
        tokenHash,
        nowMs,
      )
      .toArray();
    if (rows.length === 0) return null;
    this.sql.exec("DELETE FROM nt_edge_admin_tickets WHERE token_hash = ?", tokenHash);
    return rows[0];
  }

  insertSession({ sessionId, actor, reason, createdAt, expiresAt }) {
    if (!this.sql) throw new Error("gate_storage_unavailable");
    this.prune(createdAt);
    const sessions = this.listSessions(createdAt);
    if (sessions.length >= MAX_ACTIVE_SESSIONS) {
      this.sql.exec(
        "DELETE FROM nt_edge_admin_sessions WHERE session_id IN (SELECT session_id FROM nt_edge_admin_sessions ORDER BY created_at ASC LIMIT ?)",
        sessions.length - MAX_ACTIVE_SESSIONS + 1,
      );
    }
    this.sql.exec(
      "INSERT INTO nt_edge_admin_sessions (session_id, actor, reason, created_at, expires_at) VALUES (?, ?, ?, ?, ?)",
      sessionId,
      actor,
      reason,
      createdAt,
      expiresAt,
    );
  }

  revokeAll(nowMs) {
    if (!this.sql) return { revokedSessions: 0, revokedTickets: 0 };
    this.prune(nowMs);
    const sessions = this.listSessions(nowMs).length;
    const tickets = this.countTickets(nowMs);
    this.sql.exec("DELETE FROM nt_edge_admin_sessions");
    this.sql.exec("DELETE FROM nt_edge_admin_tickets");
    return { revokedSessions: sessions, revokedTickets: tickets };
  }
}

async function hasValidInternalSecret(request, secret) {
  if (!secret) return false;
  const provided = request.headers.get(EDGE_ADMIN_HEADER);
  if (!provided) return false;
  return timingSafeStringEqual(provided.trim(), secret);
}

async function hasValidBreakGlassCookie(request, secret, store, nowMs) {
  if (!secret || !store?.sql) return false;
  const raw = parseCookieValue(request.headers.get("cookie") || "", EDGE_ADMIN_COOKIE);
  if (!raw || raw.length > 256) return false;
  const dot = raw.indexOf(".");
  if (dot <= 16 || dot === raw.length - 1) return false;
  const sessionId = raw.slice(0, dot);
  const sig = raw.slice(dot + 1);
  if (!/^[A-Za-z0-9_-]{24,96}$/.test(sessionId) || !/^[A-Za-z0-9_-]{32,96}$/.test(sig)) {
    return false;
  }
  const expectedSig = await signSessionId(secret, sessionId);
  if (!(await timingSafeStringEqual(sig, expectedSig))) return false;
  return store.hasSession(sessionId, nowMs);
}

async function handleControlRequest(request, normalized, env, store, nowMs) {
  const secret = getConfiguredSecret(env);
  const entryPath = resolveAdminEntryPath(env);
  const path = normalized.lower;
  const method = request.method.toUpperCase();

  // Browser one-time ticket redemption endpoint
  if (path === `${EDGE_CONTROL_PREFIX}/unlock`) {
    if (method !== "GET" || !secret || !store?.sql) {
      return notFoundResponse();
    }
    const url = new URL(request.url);
    const rawTicket = (url.searchParams.get("ticket") || "").trim();
    if (!/^[A-Za-z0-9_-]{32,128}$/.test(rawTicket)) {
      return notFoundResponse();
    }
    const tokenHash = await sha256Hex(rawTicket);
    const ticket = store.consumeTicket(tokenHash, nowMs);
    if (!ticket) {
      return notFoundResponse();
    }
    const ttlSeconds = Math.max(
      MIN_SESSION_TTL_MINUTES * 60,
      Math.min(MAX_SESSION_TTL_MINUTES * 60, Number(ticket.ttl_seconds) || DEFAULT_SESSION_TTL_MINUTES * 60),
    );
    const sessionId = randomToken(32);
    const sig = await signSessionId(secret, sessionId);
    store.insertSession({
      sessionId,
      actor: String(ticket.actor || ""),
      reason: String(ticket.reason || ""),
      createdAt: nowMs,
      expiresAt: nowMs + ttlSeconds * 1000,
    });
    const cookie = `${EDGE_ADMIN_COOKIE}=${sessionId}.${sig}; Path=/; HttpOnly; Secure; SameSite=Lax; Max-Age=${ttlSeconds}`;
    return new Response(null, {
      status: 302,
      headers: {
        location: entryPath,
        "set-cookie": cookie,
        "cache-control": "no-store",
        "referrer-policy": "no-referrer",
      },
    });
  }

  // All remaining control endpoints require a valid internal admin secret; hide existence with 404.
  if (!(await hasValidInternalSecret(request, secret))) {
    return notFoundResponse();
  }

  if (path === `${EDGE_CONTROL_PREFIX}/admin-gate/status` && method === "GET") {
    const sessions = store.listSessions(nowMs);
    const pendingTickets = store.countTickets(nowMs);
    const publicAdminDisabled = isPublicAdminDisabled(env);
    return jsonResponse(200, {
      ok: true,
      module: MODULE_NAME,
      public_admin_disabled: publicAdminDisabled,
      secret_configured: true,
      api_lock_mode: publicAdminDisabled ? "strict_404" : "public_open",
      admin_entry_path: entryPath,
      active_sessions: sessions.length,
      pending_tickets: pendingTickets,
      sessions: sessions.map((row) => ({
        session_id_prefix: String(row.session_id).slice(0, 8),
        actor: String(row.actor),
        reason: String(row.reason),
        created_at: new Date(Number(row.created_at)).toISOString(),
        expires_at: new Date(Number(row.expires_at)).toISOString(),
      })),
      now: new Date(nowMs).toISOString(),
    });
  }

  if (path === `${EDGE_CONTROL_PREFIX}/admin-gate/ticket` && method === "POST") {
    let payload;
    try {
      payload = await request.json();
    } catch {
      return jsonResponse(400, { ok: false, error: "invalid_json" });
    }
    const actor = typeof payload?.actor === "string" ? payload.actor.trim().slice(0, 120) : "";
    const reason = typeof payload?.reason === "string" ? payload.reason.trim().slice(0, 200) : "";
    const rawTtl = payload?.ttl_minutes == null ? DEFAULT_SESSION_TTL_MINUTES : Number(payload.ttl_minutes);
    if (!actor || !reason || !Number.isInteger(rawTtl) || rawTtl < MIN_SESSION_TTL_MINUTES || rawTtl > MAX_SESSION_TTL_MINUTES) {
      return jsonResponse(400, { ok: false, error: "invalid_ticket_request" });
    }
    const rawTicket = randomToken(32);
    const tokenHash = await sha256Hex(rawTicket);
    const ticketExpiresAt = nowMs + TICKET_TTL_SECONDS * 1000;
    const ttlSeconds = rawTtl * 60;
    store.insertTicket({
      tokenHash,
      actor,
      reason,
      ttlSeconds,
      createdAt: nowMs,
      expiresAt: ticketExpiresAt,
    });
    return jsonResponse(201, {
      ok: true,
      module: MODULE_NAME,
      unlock_path: `${EDGE_CONTROL_PREFIX}/unlock?ticket=${encodeURIComponent(rawTicket)}`,
      ticket_expires_at: new Date(ticketExpiresAt).toISOString(),
      session_ttl_seconds: ttlSeconds,
      admin_entry_path: entryPath,
    });
  }

  if (path === `${EDGE_CONTROL_PREFIX}/admin-gate/revoke` && method === "POST") {
    const result = store.revokeAll(nowMs);
    return jsonResponse(200, {
      ok: true,
      module: MODULE_NAME,
      revoked_sessions: result.revokedSessions,
      revoked_tickets: result.revokedTickets,
      now: new Date(nowMs).toISOString(),
    });
  }

  return notFoundResponse();
}

export async function handleEdgeAdminGate(request, env, storageSql, nowFn = Date.now) {
  let pathname;
  try {
    pathname = new URL(request.url).pathname;
  } catch {
    return { response: notFoundResponse() };
  }
  const normalized = normalizeRequestPath(pathname);
  const category = classifyStorefrontRequest(request.method.toUpperCase(), normalized, env);

  if (category === "public") {
    return { allow: true };
  }
  if (category === "invalid") {
    return { response: notFoundResponse() };
  }

  const nowMs = nowFn();
  const store = new GateStore(storageSql);

  if (category === "control") {
    return { response: await handleControlRequest(request, normalized, env, store, nowMs) };
  }

  // category === "admin_only"
  if (!isPublicAdminDisabled(env)) {
    return { allow: true };
  }

  const secret = getConfiguredSecret(env);
  if (await hasValidBreakGlassCookie(request, secret, store, nowMs)) {
    return { allow: true };
  }

  return { response: notFoundResponse() };
}
