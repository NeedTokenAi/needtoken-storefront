import test from 'node:test'
import assert from 'node:assert/strict'
import { collectLocalDiagnostics, probeCookie, probeStorage, probeStorefrontConnection, redactedDiagnosticReport } from '../src/templates/needtoken/lib/browserDiagnostics.ts'

const fetchResult = (response: Response): typeof fetch => async () => response
const jsonResponse = () => new Response(JSON.stringify({ status: 'ok' }), { headers: { 'Content-Type': 'application/json; charset=utf-8' } })

test('storage probe removes its temporary key without disturbing pre-existing data', () => {
  const values = new Map<string, string>([['locale', 'zh-CN'], ['auth-token', 'secret']])
  const storage = { setItem: (key: string, value: string) => { values.set(key, value) }, getItem: (key: string) => values.get(key) ?? null, removeItem: (key: string) => { values.delete(key) } }
  assert.equal(probeStorage(() => storage), true)
  assert.deepEqual([...values], [['locale', 'zh-CN'], ['auth-token', 'secret']])
  assert.equal(probeStorage(() => { throw new Error('blocked') }), false)
  assert.equal(probeStorage(() => ({ ...storage, setItem: () => { throw new Error('quota') } })), false)
  assert.equal(values.size, 2)
})

test('cookie probe cleans its temporary cookie while preserving existing cookies', () => {
  const values = new Map([['existing', 'keep']])
  const document = {
    get cookie() { return [...values].map(([key, value]) => `${key}=${value}`).join('; ') },
    set cookie(value: string) {
      const pair = value.split(';')[0]!.split('=')
      if (value.includes('Max-Age=0')) values.delete(pair[0]!)
      else values.set(pair[0]!, pair[1]!)
    },
  }
  assert.equal(probeCookie(document, true), true)
  assert.deepEqual([...values], [['existing', 'keep']])
  const blocked = { get cookie() { return '' }, set cookie(_value: string) {} }
  assert.equal(probeCookie(blocked, true), false)
})

test('browser online hint does not imply real storefront connectivity', async () => {
  const local = collectLocalDiagnostics({ secure: true, online: true, cookiesEnabled: false, cookieDocument: { cookie: '' }, storage: () => { throw new Error('blocked') }, cryptoAvailable: true, wasmAvailable: true, language: 'en-US', timezone: () => 'America/Chicago' })
  assert.equal(local.find((row) => row.id === 'online')?.status, 'ok')
  assert.equal(local.find((row) => row.id === 'cookies')?.status, 'warning')
  assert.equal(local.find((row) => row.id === 'storage')?.status, 'warning')
  const connection = await probeStorefrontConnection({ fetcher: async () => { throw new TypeError('Failed to fetch') } })
  assert.deepEqual(connection, { status: 'unknown', detail: 'networkError' })
})

test('health check uses fixed same-origin path and validates JSON before reporting success', async () => {
  let calls = 0
  const times = [50, 87]
  const fetcher: typeof fetch = async (url, init) => {
    calls++
    assert.equal(url, '/health')
    assert.equal(init?.credentials, 'omit')
    assert.equal(init?.mode, 'same-origin')
    assert.equal(init?.redirect, 'error')
    assert.equal(init?.cache, 'no-store')
    return jsonResponse()
  }
  assert.deepEqual(await probeStorefrontConnection({ fetcher, now: () => times.shift()! }), { status: 'ok', detail: 'reachable', latencyMs: 37 })
  assert.equal(calls, 1)
  const html = new Response('<!doctype html><html>SPA</html>', { headers: { 'content-type': 'text/html' } })
  assert.deepEqual(await probeStorefrontConnection({ fetcher: fetchResult(html) }), { status: 'unknown', detail: 'invalidResponse' })
  const wrongJson = new Response('{"status":"starting"}', { headers: { 'content-type': 'application/json' } })
  assert.deepEqual(await probeStorefrontConnection({ fetcher: fetchResult(wrongJson) }), { status: 'unknown', detail: 'invalidResponse' })
  const malformedJson = new Response('{', { headers: { 'content-type': 'application/json' } })
  assert.deepEqual(await probeStorefrontConnection({ fetcher: fetchResult(malformedJson) }), { status: 'unknown', detail: 'invalidResponse' })
  assert.deepEqual(await probeStorefrontConnection({ fetcher: fetchResult(new Response('unavailable', { status: 503 })) }), { status: 'warning', detail: 'httpError', httpStatus: 503 })
})

test('timeout settles and aborts even if a fetch implementation ignores cancellation', async () => {
  let requestSignal: AbortSignal | null | undefined
  const result = await probeStorefrontConnection({ fetcher: async (_url, init) => { requestSignal = init?.signal; return new Promise<Response>(() => {}) }, timeoutMs: 5 })
  assert.deepEqual(result, { status: 'unknown', detail: 'timeout' })
  assert.equal(requestSignal?.aborted, true)
})

test('caller cancellation throws and propagates abort to request, including already cancelled calls', async () => {
  const controller = new AbortController()
  let requestSignal: AbortSignal | null | undefined
  const promise = probeStorefrontConnection({ signal: controller.signal, fetcher: async (_url, init) => { requestSignal = init?.signal; return new Promise<Response>(() => {}) } })
  controller.abort()
  await assert.rejects(promise, { name: 'AbortError' })
  assert.equal(requestSignal?.aborted, true)
  let called = false
  await assert.rejects(probeStorefrontConnection({ signal: controller.signal, fetcher: async () => { called = true; return jsonResponse() } }), { name: 'AbortError' })
  assert.equal(called, false)
})

test('redacted report omits local timezone/language and makes untested claims explicit', () => {
  const report = redactedDiagnosticReport([
    { id: 'language', status: 'ok', detail: 'localOnly', value: 'secret-language' },
    { id: 'timezone', status: 'ok', detail: 'localOnly', value: 'secret-timezone' },
  ], null, '2026-11-04T20:00:00.000Z')
  assert.equal(report.includes('secret-'), false)
  const parsed = JSON.parse(report)
  assert.equal(parsed.storefront_connection, null)
  assert.equal(parsed.account_eligibility, 'not_tested')
  assert.equal(parsed.ip_reputation, 'not_tested')
  assert.equal(parsed.external_service_connectivity, 'not_tested')
})

test('timeout also covers a health response body that never completes', async () => {
  const body = new ReadableStream({ start() { /* Deliberately never delivers JSON or closes. */ } })
  const response = new Response(body, { headers: { 'content-type': 'application/json' } })
  assert.deepEqual(await probeStorefrontConnection({ fetcher: fetchResult(response), timeoutMs: 5 }), { status: 'unknown', detail: 'timeout' })
})
