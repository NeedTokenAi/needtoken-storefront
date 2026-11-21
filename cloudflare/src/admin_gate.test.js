import assert from "node:assert/strict";
import test from "node:test";

import {
  EDGE_ADMIN_COOKIE,
  EDGE_ADMIN_HEADER,
  handleEdgeAdminGate,
} from "./admin_gate.js";
import { containerEnv, forwardRequest } from "./runtime.js";

const TEST_SECRET = "s".repeat(48);

function createMockSql() {
  const tickets = new Map();
  const sessions = new Map();
  return {
    tickets,
    sessions,
    exec(query, ...params) {
      const q = query.replace(/\s+/g, " ").trim();
      if (q.startsWith("CREATE TABLE IF NOT EXISTS")) {
        return { toArray: () => [] };
      }
      if (q.startsWith("DELETE FROM nt_edge_admin_tickets WHERE expires_at <= ?")) {
        const now = params[0];
        for (const [k, v] of tickets) {
          if (v.expires_at <= now) tickets.delete(k);
        }
        return { toArray: () => [] };
      }
      if (q.startsWith("DELETE FROM nt_edge_admin_sessions WHERE expires_at <= ?")) {
        const now = params[0];
        for (const [k, v] of sessions) {
          if (v.expires_at <= now) sessions.delete(k);
        }
        return { toArray: () => [] };
      }
      if (q.startsWith("SELECT session_id, actor, reason, created_at, expires_at FROM nt_edge_admin_sessions")) {
        const now = params[0];
        const rows = [...sessions.values()]
          .filter((r) => r.expires_at > now)
          .sort((a, b) => b.created_at - a.created_at);
        return { toArray: () => rows };
      }
      if (q.startsWith("SELECT COUNT(*) AS cnt FROM nt_edge_admin_tickets")) {
        const now = params[0];
        const cnt = [...tickets.values()].filter((r) => r.expires_at > now).length;
        return { toArray: () => [{ cnt }] };
      }
      if (q.startsWith("SELECT session_id FROM nt_edge_admin_sessions WHERE session_id = ?")) {
        const [sid, now] = params;
        const row = sessions.get(sid);
        return { toArray: () => (row && row.expires_at > now ? [{ session_id: sid }] : []) };
      }
      if (q.startsWith("INSERT INTO nt_edge_admin_tickets")) {
        const [token_hash, actor, reason, ttl_seconds, created_at, expires_at] = params;
        tickets.set(token_hash, { token_hash, actor, reason, ttl_seconds, created_at, expires_at });
        return { toArray: () => [] };
      }
      if (q.startsWith("SELECT token_hash, actor, reason, ttl_seconds, expires_at FROM nt_edge_admin_tickets")) {
        const [hash, now] = params;
        const row = tickets.get(hash);
        return { toArray: () => (row && row.expires_at > now ? [row] : []) };
      }
      if (q.startsWith("DELETE FROM nt_edge_admin_tickets WHERE token_hash = ?")) {
        tickets.delete(params[0]);
        return { toArray: () => [] };
      }
      if (q.startsWith("DELETE FROM nt_edge_admin_tickets WHERE token_hash IN")) {
        const limit = params[0];
        const oldest = [...tickets.values()].sort((a, b) => a.created_at - b.created_at).slice(0, limit);
        for (const r of oldest) tickets.delete(r.token_hash);
        return { toArray: () => [] };
      }
      if (q.startsWith("INSERT INTO nt_edge_admin_sessions")) {
        const [session_id, actor, reason, created_at, expires_at] = params;
        sessions.set(session_id, { session_id, actor, reason, created_at, expires_at });
        return { toArray: () => [] };
      }
      if (q.startsWith("DELETE FROM nt_edge_admin_sessions WHERE session_id IN")) {
        const limit = params[0];
        const oldest = [...sessions.values()].sort((a, b) => a.created_at - b.created_at).slice(0, limit);
        for (const r of oldest) sessions.delete(r.session_id);
        return { toArray: () => [] };
      }
      if (q === "DELETE FROM nt_edge_admin_sessions") {
        sessions.clear();
        return { toArray: () => [] };
      }
      if (q === "DELETE FROM nt_edge_admin_tickets") {
        tickets.clear();
        return { toArray: () => [] };
      }
      throw new Error(`unexpected SQL: ${q}`);
    },
  };
}

test("public admin UI and /api/v1/admin/* return 404 by default while C-side and ops channel stay open", async () => {
  const sql = createMockSql();
  const env = { EDGE_ADMIN_SECRET: TEST_SECRET, WEB_ADMIN_PATH: "/secret-console" };

  for (const blocked of [
    "/admin",
    "/admin/",
    "/admin/orders",
    "/ADMIN/orders",
    "//admin/orders",
    "/%61dmin",
    "/api/v1/admin/products",
    "/api/v1/ADMIN/login",
    "/api/v1/admin%2fproducts",
    "/api/v1/public/../admin/products",
    "/secret-console",
    "/secret-console/dashboard",
  ]) {
    const res = await handleEdgeAdminGate(new Request(`https://shop.example${blocked}`), env, sql);
    assert.equal(res.response?.status, 404, `expected 404 for ${blocked}`);
  }

  for (const allowed of [
    "/",
    "/products/chatgpt-plus",
    "/api/v1/products",
    "/api/v1/channel/needtoken-ops/orders",
    "/api/v1/upstream/orders",
    "/health",
  ]) {
    const res = await handleEdgeAdminGate(new Request(`https://shop.example${allowed}`), env, sql);
    assert.equal(res.allow, true, `expected allow for ${allowed}`);
    assert.equal(res.response, undefined);
  }
});

test("break-glass ticket is single-use, scoped to cookie holder, expires, and revokes immediately", async () => {
  const sql = createMockSql();
  let now = 1_700_000_000_000;
  const env = { EDGE_ADMIN_SECRET: TEST_SECRET };

  // Unauthenticated control endpoints hide existence with 404.
  for (const path of [
    "/__needtoken_edge/admin-gate/status",
    "/__needtoken_edge/admin-gate/ticket",
    "/__needtoken_edge/admin-gate/revoke",
    "/__needtoken_edge/unknown",
  ]) {
    const r = await handleEdgeAdminGate(new Request(`https://shop.example${path}`), env, sql, () => now);
    assert.equal(r.response?.status, 404);
  }

  // Issue ticket with valid internal secret.
  const issueRes = await handleEdgeAdminGate(
    new Request("https://shop.example/__needtoken_edge/admin-gate/ticket", {
      method: "POST",
      headers: {
        "content-type": "application/json",
        [EDGE_ADMIN_HEADER]: TEST_SECRET,
      },
      body: JSON.stringify({ actor: "owner-1", reason: "verify theme layout", ttl_minutes: 15 }),
    }),
    env,
    sql,
    () => now,
  );
  assert.equal(issueRes.response?.status, 201);
  const issued = await issueRes.response.json();
  assert.equal(issued.ok, true);
  assert.equal(issued.session_ttl_seconds, 900);
  assert.match(issued.unlock_path, /^\/__needtoken_edge\/unlock\?ticket=/);

  // Raw ticket must not be stored in plaintext inside SQLite.
  const rawTicket = new URL(`https://shop.example${issued.unlock_path}`).searchParams.get("ticket");
  assert.equal(sql.tickets.has(rawTicket), false);
  assert.equal(sql.tickets.size, 1);

  // Redeem ticket once -> 302 + __Host cookie.
  const unlockRes = await handleEdgeAdminGate(
    new Request(`https://shop.example${issued.unlock_path}`),
    env,
    sql,
    () => now,
  );
  assert.equal(unlockRes.response?.status, 302);
  assert.equal(unlockRes.response.headers.get("location"), "/admin");
  assert.equal(unlockRes.response.headers.get("referrer-policy"), "no-referrer");
  const setCookie = unlockRes.response.headers.get("set-cookie");
  assert.match(setCookie, new RegExp(`^${EDGE_ADMIN_COOKIE}=[^;]+; Path=/; HttpOnly; Secure; SameSite=Lax; Max-Age=900$`));
  const cookiePair = setCookie.split(";")[0];

  // Replay of the same ticket must fail with 404.
  const replayRes = await handleEdgeAdminGate(
    new Request(`https://shop.example${issued.unlock_path}`),
    env,
    sql,
    () => now,
  );
  assert.equal(replayRes.response?.status, 404);

  // Holder with valid cookie accesses /admin and /api/v1/admin/*; anonymous still gets 404.
  const withCookie = await handleEdgeAdminGate(
    new Request("https://shop.example/api/v1/admin/products", { headers: { cookie: cookiePair } }),
    env,
    sql,
    () => now,
  );
  assert.equal(withCookie.allow, true);

  const withoutCookie = await handleEdgeAdminGate(
    new Request("https://shop.example/api/v1/admin/products"),
    env,
    sql,
    () => now,
  );
  assert.equal(withoutCookie.response?.status, 404);

  // Tampered cookie signature fails with 404.
  const tampered = await handleEdgeAdminGate(
    new Request("https://shop.example/admin", { headers: { cookie: `${cookiePair}x` } }),
    env,
    sql,
    () => now,
  );
  assert.equal(tampered.response?.status, 404);

  // Status reports active session without exposing raw session_id.
  const statusRes = await handleEdgeAdminGate(
    new Request("https://shop.example/__needtoken_edge/admin-gate/status", {
      headers: { [EDGE_ADMIN_HEADER]: TEST_SECRET },
    }),
    env,
    sql,
    () => now,
  );
  const statusBody = await statusRes.response.json();
  assert.equal(statusBody.active_sessions, 1);
  assert.equal(statusBody.api_lock_mode, "strict_404");
  assert.equal(statusBody.sessions[0].actor, "owner-1");
  assert.equal(statusBody.sessions[0].session_id_prefix.length, 8);
  assert.equal(JSON.stringify(statusBody).includes(TEST_SECRET), false);

  // Revoke clears sessions immediately -> existing cookie immediately returns 404.
  const revokeRes = await handleEdgeAdminGate(
    new Request("https://shop.example/__needtoken_edge/admin-gate/revoke", {
      method: "POST",
      headers: { [EDGE_ADMIN_HEADER]: TEST_SECRET },
    }),
    env,
    sql,
    () => now,
  );
  assert.equal((await revokeRes.response.json()).revoked_sessions, 1);

  const afterRevoke = await handleEdgeAdminGate(
    new Request("https://shop.example/admin", { headers: { cookie: cookiePair } }),
    env,
    sql,
    () => now,
  );
  assert.equal(afterRevoke.response?.status, 404);

  // Expired ticket cannot be redeemed.
  const issue2 = await handleEdgeAdminGate(
    new Request("https://shop.example/__needtoken_edge/admin-gate/ticket", {
      method: "POST",
      headers: { "content-type": "application/json", [EDGE_ADMIN_HEADER]: TEST_SECRET },
      body: JSON.stringify({ actor: "owner-1", reason: "expiry test", ttl_minutes: 10 }),
    }),
    env,
    sql,
    () => now,
  );
  const issued2 = await issue2.response.json();
  now += 61_000;
  const expiredUnlock = await handleEdgeAdminGate(
    new Request(`https://shop.example${issued2.unlock_path}`),
    env,
    sql,
    () => now,
  );
  assert.equal(expiredUnlock.response?.status, 404);
});

test("edge secrets never leak into container env or forwarded headers", () => {
  const passed = containerEnv({
    DATABASE_URL: "postgres://db/store",
    EDGE_ADMIN_SECRET: TEST_SECRET,
    DISABLE_PUBLIC_ADMIN: "1",
    EDGE_ADMIN_ENFORCE_API: "1",
  });
  assert.equal(Object.hasOwn(passed, "EDGE_ADMIN_SECRET"), false);
  assert.equal(Object.hasOwn(passed, "DISABLE_PUBLIC_ADMIN"), false);
  assert.equal(Object.hasOwn(passed, "EDGE_ADMIN_ENFORCE_API"), false);

  const req = new Request("https://shop.example/api/v1/products", {
    headers: { [EDGE_ADMIN_HEADER]: TEST_SECRET, "x-keep": "yes" },
  });
  const forwarded = forwardRequest(req);
  assert.equal(forwarded.headers.get(EDGE_ADMIN_HEADER), null);
  assert.equal(forwarded.headers.get("x-keep"), "yes");
});
