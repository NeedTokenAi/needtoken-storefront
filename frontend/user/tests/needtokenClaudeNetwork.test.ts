import assert from 'node:assert/strict'
import test from 'node:test'
import { CLAUDE_PROBES, CLAUDE_SERVICES, classifyWebRtc, claudeNetworkReport, claudeRuleCategoryIds, collectClaudeLocalChecks, isPublicIp, lookupExitNetwork, normalizeIp, parseCliReport, probeAnthropic, probeClaudeService, probeWebRtc, runClaudeNetworkChecks, timezoneDiffers, type NetworkCheckEvent } from '../src/templates/needtoken/lib/claudeNetwork.ts'

const exit = { status: 'ok' as const, detail: 'geoAvailable' as const, source: 'ipapi.co' as const, ip: '8.8.8.8', country: 'United States', countryCode: 'US', timezone: 'America/Los_Angeles', measuredAt: '2026-11-05T00:00:00Z' }
const cli = {
  schema: 'needtoken.claude-network.v1', scope: 'curl_inherits_proxy_env', measured_at: '2026-11-05T00:00:00Z',
  runtime: { node_major: 24, timezone: 'Asia/Shanghai' },
  proxy: { http: true, https: true, all: false, no_proxy: false, custom_base_url: false },
  dns: { status: 'ok', address_count: 2 }, curl: { status: 'ok', exit_code: 0, http_status: 404, ssl_verify_result: 0, time_ms: 42, peer_ip: '8.8.4.4' },
}

test('exit IP parsing distinguishes public, local, documentation and mDNS observations', () => {
  assert.equal(normalizeIp('8.8.8.8'), '8.8.8.8')
  assert.equal(normalizeIp('999.1.2.3'), null)
  assert.equal(normalizeIp('abc.local'), null)
  assert.equal(isPublicIp('2001:4860:4860::8888'), true)
  for (const value of ['127.0.0.1', '10.1.1.2', '192.168.1.1', '100.64.1.1', '203.0.113.4', '198.51.100.7', '2001:db8::1', 'fe80::1', '::1']) assert.equal(isPublicIp(value), false)
})

test('exit lookup uses fixed CORS providers without credentials and preserves provenance', async () => {
  const calls: string[] = []
  const result = await lookupExitNetwork({ fetcher: (async (url, options) => {
    calls.push(String(url)); assert.equal(options?.credentials, 'omit'); assert.equal(options?.referrerPolicy, 'no-referrer')
    return Response.json({ ip: '8.8.8.8', country_name: 'United States', country_code: 'US', asn: 'AS15169', org: 'Example', timezone: 'America/Los_Angeles' })
  }) as typeof fetch })
  assert.deepEqual(calls, [CLAUDE_PROBES.geo])
  assert.equal(result.ip, '8.8.8.8'); assert.equal(result.asn, 'AS15169'); assert.equal(result.source, 'ipapi.co')
  assert.equal(result.status, 'ok')
})

test('failed geo lookup falls back to IP-only instead of inventing ASN or reputation', async () => {
  const result = await lookupExitNetwork({ fetcher: (async url => String(url) === CLAUDE_PROBES.geo ? Response.json({ error: true }) : Response.json({ ip: '1.1.1.1' })) as typeof fetch })
  assert.equal(result.detail, 'ipOnly'); assert.equal(result.status, 'unknown'); assert.equal(result.source, 'api.ipify.org')
  assert.equal(result.asn, undefined)
  const failed = await lookupExitNetwork({ fetcher: (async () => Response.json({ ip: '127.0.0.1' })) as typeof fetch })
  assert.equal(failed.detail, 'lookupUnavailable'); assert.equal(failed.ip, undefined)
})

test('lookup deadlines include stalled response bodies and cancellation stops fallback', async () => {
  const result = await lookupExitNetwork({ timeoutMs: 4, fetcher: (async () => ({ ok: true, json: () => new Promise(() => {}) })) as typeof fetch })
  assert.equal(result.detail, 'lookupUnavailable')
  const controller = new AbortController(); let calls = 0
  const pending = lookupExitNetwork({ signal: controller.signal, fetcher: (async () => { calls++; return await new Promise(() => {}) }) as typeof fetch })
  controller.abort()
  await assert.rejects(pending, { name: 'AbortError' }); assert.equal(calls, 1)
})

test('opaque Anthropic response remains unknown and never becomes CLI or TLS proof', async () => {
  const modes: Array<RequestMode | undefined> = []
  const result = await probeAnthropic({ fetcher: (async (url, options) => {
    modes.push(options?.mode)
    assert.equal(url, CLAUDE_PROBES.anthropic); assert.equal(options?.redirect, options?.mode === 'cors' ? 'error' : 'follow'); assert.equal(options?.credentials, 'omit')
    if (options?.mode === 'cors') throw new TypeError('CORS blocked')
    return { type: 'opaque', status: 0 }
  }) as typeof fetch, now: () => 10 })
  assert.deepEqual(modes, ['cors', 'no-cors'])
  assert.equal(result.status, 'unknown'); assert.equal(result.detail, 'opaqueResponse'); assert.equal(result.httpStatus, undefined)
})

test('readable HTTP restrictions and failed requests remain distinct', async () => {
  const restricted = await probeAnthropic({ fetcher: (async () => new Response(null, { status: 403 })) as typeof fetch })
  assert.equal(restricted.status, 'warning'); assert.equal(restricted.httpStatus, 403)
  const network = await probeAnthropic({ timeoutMs: 3, fetcher: (async () => await new Promise(() => {})) as typeof fetch })
  assert.equal(network.status, 'unknown'); assert.equal(network.detail, 'timeout')
})

test('WebRTC compares only observed public candidates and mDNS is not a proven public leak', () => {
  assert.equal(classifyWebRtc(['8.8.8.8'], 0, '8.8.8.8').detail, 'sameExit')
  assert.equal(classifyWebRtc(['1.1.1.1', '10.0.0.1'], 0, '8.8.8.8').detail, 'differentExit')
  const mdns = classifyWebRtc([], 1, '8.8.8.8'); assert.equal(mdns.status, 'unknown'); assert.equal(mdns.detail, 'localOnly')
  assert.equal(classifyWebRtc([], 0, '8.8.8.8').detail, 'noCandidates')
  assert.equal(classifyWebRtc(['8.8.8.8'], 0).detail, 'publicOnly')
})

test('WebRTC cancellation closes the peer and removes active work', async () => {
  const controller = new AbortController(); let closed = false
  const peer = { createDataChannel: () => {}, createOffer: () => new Promise(() => {}), close: () => { closed = true }, onicecandidate: null, onicegatheringstatechange: null }
  const result = probeWebRtc({ signal: controller.signal, factory: () => peer as unknown as RTCPeerConnection })
  controller.abort(); await assert.rejects(result, { name: 'AbortError' }); assert.equal(closed, true)
})

test('CLI imports have a fixed schema, size limits and consistent successful metrics', () => {
  const valid = parseCliReport(JSON.stringify({ ...cli, secret: 'never copy', curl: { ...cli.curl, proxy_url: 'secret' } }))
  assert.equal(valid?.curl.http_status, 404); assert.equal(valid?.scope, 'curl_inherits_proxy_env')
  assert.equal('secret' in (valid || {}), false); assert.equal('proxy_url' in (valid?.curl || {}), false)
  assert.equal(parseCliReport(JSON.stringify({ ...cli, curl: { status: 'ok', exit_code: 0 } })), null)
  assert.equal(parseCliReport(JSON.stringify({ ...cli, schema: 'arbitrary' })), null)
  assert.equal(parseCliReport('x'.repeat(65537)), null)
  assert.equal(parseCliReport(JSON.stringify({ ...cli, dns: { status: 'ok', address_count: 0 } })), null)
})

test('timezone differences are observations only and missing geo cannot be treated as a mismatch', () => {
  assert.equal(timezoneDiffers('Asia/Shanghai', exit), true)
  assert.equal(timezoneDiffers('Asia/Shanghai', null), false)
})

test('shared report omits IPs, timezone, location, language and imported peer IP', () => {
  const report = claudeNetworkReport({ exit, anthropic: null, rtc: classifyWebRtc(['1.1.1.1'], 0, exit.ip), cli: parseCliReport(JSON.stringify(cli)), measuredAt: '2026-11-05T00:00:00Z' })
  assert.equal(report.includes('8.8.8.8'), false); assert.equal(report.includes('1.1.1.1'), false); assert.equal(report.includes('8.8.4.4'), false)
  assert.equal(report.includes('Asia/Shanghai'), false); assert.equal(report.includes('America/Los_Angeles'), false); assert.equal(report.includes('United States'), false)
  assert.equal(JSON.parse(report).account_eligibility, 'not_tested'); assert.equal(JSON.parse(report).ip_reputation, 'not_tested')
})

test('operator notes require their dedicated published category scope', () => {
  assert.deepEqual(claudeRuleCategoryIds([{ id: 1, slug: 'other' }, { id: 2, slug: 'child', parent_id: 1 }]), [])
  assert.deepEqual(claudeRuleCategoryIds([{ id: 4, slug: 'claude-code-rules', parent_id: 0 }, { id: 5, slug: 'policy', parent_id: 4 }, { id: 8, slug: 'unrelated' }]), [4, 5])
  assert.deepEqual(claudeRuleCategoryIds(null), [])
})

test('local checks finish synchronously even when every browser getter is restricted', () => {
  const unavailable = () => { throw new Error('restricted') }
  const checks = collectClaudeLocalChecks({ secure: unavailable, online: unavailable, cookies: unavailable, storage: unavailable, crypto: unavailable, wasm: unavailable, fetch: unavailable, abort: unavailable, intl: unavailable, language: unavailable, timezone: unavailable, locale: unavailable, clock: unavailable, platform: unavailable, browser: unavailable })
  assert.equal(checks.length, 15)
  assert.ok(checks.every(check => check.status === 'unknown'))
  assert.equal(checks.filter(check => check.group === 'system').length, 6)
})

test('local clock and browser-reported system are observations without account risk scoring', () => {
  const checks = collectClaudeLocalChecks({ secure: () => true, online: () => false, language: () => 'zh-CN', timezone: () => 'Asia/Shanghai', clock: () => '2026-11-05T00:00:00Z', platform: () => 'Windows' })
  assert.equal(checks.find(check => check.id === 'online')?.status, 'warning')
  assert.equal(checks.find(check => check.id === 'clock')?.status, 'unknown')
  assert.equal(checks.find(check => check.id === 'platform')?.value, 'Windows')
  assert.equal(checks.find(check => check.id === 'language')?.status, 'ok')
})

test('same-origin health completes independently while all third-party probes stall', async () => {
  const events: NetworkCheckEvent[] = []
  let finished = false
  const pending = runClaudeNetworkChecks({ timeoutMs: 8, onResult: event => events.push(event), fetcher: (async url => String(url) === '/health' ? Response.json({ status: 'ok' }) : new Promise(() => {})) as typeof fetch }).then(() => { finished = true })
  await new Promise(resolve => setTimeout(resolve, 1))
  assert.equal(events.find(event => event.id === 'storefront')?.result.status, 'ok')
  assert.equal(finished, false)
  await pending
  assert.equal(events.length, 5)
  assert.equal(events.find(event => event.id === 'exit')?.result.status, 'unknown')
  assert.equal(events.find(event => event.id === 'anthropic')?.result.status, 'unknown')
})

test('blocked, synchronously throwing and uncancellable fetches all settle without fake success', async () => {
  for (const fetcher of [() => { throw new TypeError('blocked') }, () => new Promise(() => {})]) {
    const events: NetworkCheckEvent[] = []
    await runClaudeNetworkChecks({ timeoutMs: 3, onResult: event => events.push(event), fetcher: fetcher as typeof fetch })
    assert.equal(events.length, 5)
    assert.ok(events.every(event => event.result.status === 'unknown'))
  }
})

test('cancelled network batches suppress late events and stop even when fetch ignores abort', async () => {
  const controller = new AbortController(), events: NetworkCheckEvent[] = []
  const pending = runClaudeNetworkChecks({ signal: controller.signal, onResult: event => events.push(event), fetcher: (() => new Promise(() => {})) as typeof fetch })
  controller.abort()
  await assert.rejects(pending, { name: 'AbortError' })
  assert.deepEqual(events, [])
})

test('expanded reports redact local system, clock, locale and exact browser values', () => {
  const local = collectClaudeLocalChecks({ language: () => 'secret-language', timezone: () => 'secret-timezone', platform: () => 'secret-os', browser: () => 'secret-browser', clock: () => 'secret-clock', locale: () => 'secret-locale' })
  const report = claudeNetworkReport({ exit: null, anthropic: null, rtc: null, cli: null, measuredAt: '', local, storefront: { status: 'unknown', detail: 'networkError' } })
  assert.equal(report.includes('secret-'), false)
  assert.equal(JSON.parse(report).local_checks.length, 15)
  assert.equal(JSON.parse(report).storefront_connection.status, 'unknown')
})

test('three fixed service probes distinguish opaque completion, readable restrictions and deadlines', async () => {
  const calls: string[] = []
  const fetcher = (async (url, options) => {
    calls.push(String(url))
    assert.equal(options?.credentials, 'omit')
    assert.equal(options?.referrerPolicy, 'no-referrer')
    assert.equal(options?.redirect, options?.mode === 'cors' && url === CLAUDE_PROBES.anthropic ? 'error' : 'follow')
    if (url === CLAUDE_PROBES.claudeWeb) {
      if (options?.mode === 'cors') throw new TypeError('CORS blocked')
      return { type: 'opaque', status: 0 }
    }
    if (url === CLAUDE_PROBES.anthropicWeb) return new Response(null, { status: 403 })
    return new Promise(() => {})
  }) as typeof fetch
  const results = await Promise.all(CLAUDE_SERVICES.map(id => probeClaudeService(id, { fetcher, timeoutMs: 5 })))
  assert.deepEqual(calls, [CLAUDE_PROBES.claudeWeb, CLAUDE_PROBES.anthropicWeb, CLAUDE_PROBES.anthropic, CLAUDE_PROBES.claudeWeb])
  assert.equal(results[0]?.detail, 'opaqueResponse')
  assert.equal(results[0]?.status, 'unknown')
  assert.equal(results[0]?.httpStatus, undefined)
  assert.equal(results[1]?.status, 'warning')
  assert.equal(results[1]?.httpStatus, 403)
  assert.equal(results[2]?.detail, 'timeout')
  await assert.rejects(probeClaudeService('custom' as never, { fetcher }), /unsupported service target/)
})

test('service observations in shared reports allow only fixed target metrics', () => {
  const result = { status: 'unknown' as const, detail: 'opaqueResponse' as const, latencyMs: 24, measuredAt: '2026-11-05T00:00:00Z', secret: 'secret-value', ip: '8.8.8.8' }
  const report = claudeNetworkReport({ exit: null, anthropic: null, rtc: null, cli: null, measuredAt: '', services: { claudeWeb: result, custom: result } as never })
  assert.equal(report.includes('secret-value'), false)
  assert.equal(report.includes('8.8.8.8'), false)
  assert.equal('custom' in JSON.parse(report).services_browser, false)
  assert.equal(JSON.parse(report).services_browser.claudeWeb.latencyMs, 24)
})


test('abort-aware fetches keep deadline timeouts distinct from caller cancellation', async () => {
  let observedSignal: AbortSignal | null | undefined
  const modes: Array<RequestMode | undefined> = []
  const fetcher = ((_url, init) => new Promise((_resolve, reject) => {
    modes.push(init?.mode)
    observedSignal = init?.signal
    observedSignal?.addEventListener('abort', () => reject(new DOMException('Fetch aborted', 'AbortError')), { once: true })
  })) as typeof fetch
  const timedOut = await probeClaudeService('anthropic', { fetcher, timeoutMs: 5 })
  assert.equal(timedOut.detail, 'timeout')
  assert.equal(timedOut.status, 'unknown')
  assert.equal(observedSignal?.aborted, true)
  assert.deepEqual(modes, ['cors'])
  const caller = new AbortController()
  const cancelled = probeClaudeService('claudeWeb', { fetcher, signal: caller.signal, timeoutMs: 1000 })
  caller.abort()
  await assert.rejects(cancelled, { name: 'AbortError' })
  assert.equal(observedSignal?.aborted, true)
  assert.deepEqual(modes, ['cors', 'cors'])
})

test('readable CORS responses retain real HTTP outcomes without an opaque retry', async () => {
  for (const status of [200, 401, 404, 403, 429, 503]) {
    const calls: Array<RequestMode | undefined> = []
    const result = await probeClaudeService('anthropic', { fetcher: (async (url, init) => {
      calls.push(init?.mode)
      assert.equal(url, CLAUDE_PROBES.anthropic)
      assert.equal(init?.credentials, 'omit')
      assert.equal(init?.referrerPolicy, 'no-referrer')
      assert.equal(init?.headers, undefined)
      return new Response(null, { status })
    }) as typeof fetch })
    assert.deepEqual(calls, ['cors'])
    assert.equal(result.httpStatus, status)
    assert.equal(result.detail, [403, 429, 503].includes(status) ? 'httpWarning' : 'readableResponse')
  }
})

test('failed CORS and opaque fallback requests remain unconfirmed', async () => {
  const modes: Array<RequestMode | undefined> = []
  const result = await probeClaudeService('claudeWeb', { fetcher: (async (_url, init) => {
    modes.push(init?.mode)
    throw new TypeError('Network unavailable')
  }) as typeof fetch })
  assert.deepEqual(modes, ['cors', 'no-cors'])
  assert.equal(result.detail, 'requestUnconfirmed')
  assert.equal(result.status, 'unknown')
  assert.equal(result.httpStatus, undefined)
})

test('every opaque fallback uses Fetch-compatible redirect handling without credentials', async () => {
  for (const target of CLAUDE_SERVICES) {
    const calls: Array<{ mode: RequestMode | undefined; redirect: RequestRedirect | undefined }> = []
    const result = await probeClaudeService(target, { fetcher: (async (url, init) => {
      assert.equal(url, CLAUDE_PROBES[target])
      assert.equal(init?.credentials, 'omit')
      assert.equal(init?.referrerPolicy, 'no-referrer')
      calls.push({ mode: init?.mode, redirect: init?.redirect })
      if (init?.mode === 'cors') throw new TypeError('CORS blocked')
      assert.equal(init?.mode, 'no-cors')
      assert.equal(init?.redirect, 'follow')
      return { type: 'opaque', status: 0 }
    }) as typeof fetch })
    assert.deepEqual(calls, [
      { mode: 'cors', redirect: target === 'anthropic' ? 'error' : 'follow' },
      { mode: 'no-cors', redirect: 'follow' },
    ])
    assert.equal(result.detail, 'opaqueResponse')
    assert.equal(result.status, 'unknown')
    assert.equal(result.httpStatus, undefined)
  }
})

test('CORS and fallback attempts share one total deadline', async context => {
  context.mock.timers.enable({ apis: ['setTimeout'] })
  const modes: Array<RequestMode | undefined> = []
  const signals: Array<AbortSignal | null | undefined> = []
  let result: Awaited<ReturnType<typeof probeClaudeService>> | undefined
  const pending = probeClaudeService('claudeWeb', { timeoutMs: 100, fetcher: ((_url, init) => {
    modes.push(init?.mode)
    signals.push(init?.signal)
    return new Promise((_resolve, reject) => {
      if (init?.mode === 'cors') setTimeout(() => reject(new TypeError('CORS blocked')), 60)
      init?.signal?.addEventListener('abort', () => reject(new DOMException('Aborted', 'AbortError')), { once: true })
    })
  }) as typeof fetch }).then(value => { result = value })
  context.mock.timers.tick(60)
  await Promise.resolve()
  assert.deepEqual(modes, ['cors', 'no-cors'])
  assert.equal(signals[0], signals[1])
  context.mock.timers.tick(39)
  await Promise.resolve()
  assert.equal(result, undefined)
  context.mock.timers.tick(1)
  await pending
  assert.equal(result?.detail, 'timeout')
  assert.equal(signals[0]?.aborted, true)
})
