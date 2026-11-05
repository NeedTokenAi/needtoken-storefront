import assert from 'node:assert/strict'
import test from 'node:test'
import { CLAUDE_PROBES, classifyWebRtc, claudeNetworkReport, claudeRuleCategoryIds, isPublicIp, lookupExitNetwork, normalizeIp, parseCliReport, probeAnthropic, probeWebRtc, timezoneDiffers } from '../src/templates/needtoken/lib/claudeNetwork.ts'

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
  const result = await probeAnthropic({ fetcher: (async (url, options) => {
    assert.equal(url, CLAUDE_PROBES.anthropic); assert.equal(options?.mode, 'no-cors'); assert.equal(options?.redirect, 'error'); assert.equal(options?.credentials, 'omit')
    return { type: 'opaque', status: 0 }
  }) as typeof fetch, now: () => 10 })
  assert.equal(result.status, 'unknown'); assert.equal(result.detail, 'opaqueResponse'); assert.equal(result.httpStatus, undefined)
})

test('readable HTTP restrictions and failed requests remain distinct', async () => {
  const restricted = await probeAnthropic({ fetcher: (async () => new Response(null, { status: 403 })) as typeof fetch })
  assert.equal(restricted.status, 'warning'); assert.equal(restricted.httpStatus, 403)
  const network = await probeAnthropic({ timeoutMs: 3, fetcher: (async () => await new Promise(() => {})) as typeof fetch })
  assert.equal(network.status, 'unknown'); assert.equal(network.detail, 'requestUnconfirmed')
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
