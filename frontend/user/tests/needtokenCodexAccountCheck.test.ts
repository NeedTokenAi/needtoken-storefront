import assert from 'node:assert/strict'
import test from 'node:test'
import { checkCodexAccount, CodexCheckError, MAX_CODEX_SESSION_INPUT, parseCodexAccountResult, parseCodexSessionInput } from '../src/templates/needtoken/lib/codexAccountCheck.ts'

const segment = (value: unknown) => Buffer.from(JSON.stringify(value)).toString('base64url')
// Deliberately unsigned test fixture: parsing checks shape, never treats claims as account evidence.
const token = `${segment({ alg: 'RS256', typ: 'JWT' })}.${segment({ sub: 'fixture-subject', exp: 1, plan: 'pro' })}.${'x'.repeat(32)}`
const result = {
  checked_at: '2026-11-14T10:00:00Z',
  accounts: [{ id: 'account-1', plan: 'plus', is_default: true }, { id: 'account-2', plan: 'team', is_default: false, expires_at: '2026-12-01T00:00:00Z' }],
  usage_status: 'not_requested',
}

test('session input extracts only the access token and never returns identity or JWT claims', () => {
  assert.deepEqual(parseCodexSessionInput(JSON.stringify({ accessToken: token, user: { email: 'fixture@example.test' }, expires: '2099-01-01', refreshToken: 'do-not-retain' })), { ok: true, accessToken: token })
  assert.deepEqual(parseCodexSessionInput(`  Bearer ${token}\n`), { ok: true, accessToken: token })
  assert.deepEqual(parseCodexSessionInput(JSON.stringify({ access_token: token })), { ok: true, accessToken: token })
  // An expired claim is not a verified server result and is deliberately not used for subscription decisions.
  assert.equal(parseCodexSessionInput(token).ok, true)
})

test('malformed, ambiguous, cookie, nested, opaque and oversized credentials fail closed', () => {
  assert.deepEqual(parseCodexSessionInput(''), { ok: false, error: 'empty' })
  assert.deepEqual(parseCodexSessionInput('x'.repeat(MAX_CODEX_SESSION_INPUT + 1)), { ok: false, error: 'tooLarge' })
  assert.deepEqual(parseCodexSessionInput('{broken'), { ok: false, error: 'invalidJson' })
  for (const value of [JSON.stringify([]), JSON.stringify({ user: { accessToken: token } })]) assert.deepEqual(parseCodexSessionInput(value), { ok: false, error: 'missingToken' })
  for (const value of [
    JSON.stringify({ accessToken: token, access_token: token + 'x' }),
    JSON.stringify({ accessToken: { token } }), '__Secure-next-auth.session-token=fixture-cookie',
    'at-fixture-personal-access-token', 'sk-proj-fixture-key', 'abc.def.ghi',
    `${segment({ alg: 'none' })}.${segment({ sub: 'fixture' })}.${'x'.repeat(32)}`,
    `${segment({ alg: 'RS256' })}.${segment(['not-an-object'])}.${'x'.repeat(32)}`,
    `${segment({ alg: 'RS256' })}.${segment({})}.${'x'.repeat(16384)}`,
  ]) assert.equal(parseCodexSessionInput(value).ok, false)
})

test('account results preserve workspace identity and discard private and unsupported fields', () => {
  const parsed = parseCodexAccountResult({ ...result, access_token: token, email: 'fixture@example.test', can_upgrade: true })!
  assert.equal(parsed.accounts[0]?.plan, 'plus')
  assert.equal(parsed.accounts[0]?.expiresAt, null)
  assert.equal(parsed.accounts[1]?.plan, 'team')
  assert.equal(parsed.selectedAccountId, null)
  assert.equal(parsed.usageStatus, 'not_requested')
  assert.equal(parsed.usage, null)
  assert.equal('access_token' in parsed, false)
  assert.equal('can_upgrade' in parsed, false)
  assert.equal(JSON.stringify(parsed).includes('fixture@example.test'), false)
  assert.equal(parseCodexAccountResult({ ...result, accounts: [{ id: 'a', is_default: false }] })?.accounts[0]?.plan, 'unknown')
})

test('usage belongs only to an explicitly selected returned account and valid measured windows', () => {
  const usage = { primary: { used_percent: 25.5, window_seconds: 18000, reset_at: '2026-11-14T15:00:00Z' }, secondary: { used_percent: 80, window_seconds: 604800 } }
  const parsed = parseCodexAccountResult({ ...result, selected_account_id: 'account-2', usage_status: 'ready', usage })!
  assert.equal(parsed.selectedAccountId, 'account-2')
  assert.equal(parsed.usage?.primary?.usedPercent, 25.5)
  assert.equal(parsed.usage?.secondary?.resetAt, null)
  for (const data of [
    { ...result, selected_account_id: 'other-account', usage_status: 'ready', usage },
    { ...result, usage_status: 'ready', usage },
    { ...result, selected_account_id: 'account-1', usage_status: 'ready', usage: { primary: { used_percent: -1, window_seconds: 18000 } } },
    { ...result, accounts: [result.accounts[0], result.accounts[0]] },
    { ...result, checked_at: 'not-a-date' },
  ]) assert.equal(parseCodexAccountResult(data), null)
  assert.equal(parseCodexAccountResult({ ...result, selected_account_id: 'account-1', usage_status: 'unavailable' })?.usage, null)
})

test('request sends only selected token and account to the fixed diagnostic route without cookies or caching', async () => {
  const value = await checkCodexAccount({ accessToken: token, accountId: 'account-1', locale: 'zh-CN', fetcher: (async (url, options) => {
    assert.equal(url, '/api/v1/public/tools/codex-account')
    assert.equal(options?.method, 'POST')
    assert.equal(options?.credentials, 'omit')
    assert.equal(options?.cache, 'no-store')
    assert.equal(options?.redirect, 'error')
    assert.equal(options?.referrerPolicy, 'no-referrer')
    assert.deepEqual(JSON.parse(String(options?.body)), { access_token: token, account_id: 'account-1' })
    assert.equal('Authorization' in (options?.headers as Record<string, string>), false)
    return Response.json({ status_code: 0, msg: 'ok', data: result })
  }) as typeof fetch })
  assert.equal(value.accounts.length, 2)
})

test('upstream error bodies never become UI messages or leak credentials', async () => {
  await assert.rejects(checkCodexAccount({ accessToken: token, fetcher: (async () => Response.json({ status_code: 401, msg: `unsafe ${token}`, data: { error_code: 'unauthorized', access_token: token } })) as typeof fetch }), error => {
    assert.ok(error instanceof CodexCheckError)
    assert.equal(error.code, 'unauthorized')
    assert.equal(String(error).includes(token), false)
    return true
  })
  await assert.rejects(checkCodexAccount({ accessToken: token, fetcher: (async () => new Response('<html>challenge</html>', { status: 429 })) as typeof fetch }), { code: 'rate_limited' })
  await assert.rejects(checkCodexAccount({ accessToken: token, fetcher: (async () => Response.json({ status_code: 0, data: { ...result, accounts: [] } })) as typeof fetch }), { code: 'invalid_response' })
})

test('deadline covers stalled response body and cancellation aborts in-flight checks', async () => {
  await assert.rejects(checkCodexAccount({ accessToken: token, timeoutMs: 4, fetcher: (async () => ({ ok: true, headers: new Headers(), text: () => new Promise(() => {}) })) as typeof fetch }), { code: 'timeout' })
  const controller = new AbortController()
  let aborted = false
  const pending = checkCodexAccount({ accessToken: token, signal: controller.signal, fetcher: (async (_url, options) => {
    options?.signal?.addEventListener('abort', () => { aborted = true })
    return await new Promise(() => {})
  }) as typeof fetch })
  controller.abort()
  await assert.rejects(pending, { name: 'AbortError' })
  assert.equal(aborted, true)
})
