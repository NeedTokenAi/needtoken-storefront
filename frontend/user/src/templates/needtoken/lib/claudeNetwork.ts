/** Original network diagnostics, informed by LinXiaoTao/FuckClaude's separation
 * of local observations and IP reputation. No account eligibility score is inferred. */
import { probeCookie, probeStorage, probeStorefrontConnection, type ConnectionDiagnostic } from './browserDiagnostics.ts'

export const CLAUDE_PROBES = {
  geo: 'https://ipapi.co/json/',
  ip: 'https://api.ipify.org?format=json',
  anthropic: 'https://api.anthropic.com/',
  stun: 'stun:stun.l.google.com:19302',
} as const

export type NetworkStatus = 'ok' | 'warning' | 'unknown'
export interface ExitNetwork {
  status: NetworkStatus
  detail: 'geoAvailable' | 'ipOnly' | 'lookupUnavailable'
  source: 'ipapi.co' | 'api.ipify.org' | ''
  ip?: string
  country?: string
  countryCode?: string
  region?: string
  city?: string
  asn?: string
  organization?: string
  timezone?: string
  measuredAt: string
}
export interface AnthropicConnection {
  status: NetworkStatus
  detail: 'readableResponse' | 'httpWarning' | 'opaqueResponse' | 'requestUnconfirmed'
  latencyMs?: number
  httpStatus?: number
  measuredAt: string
}
export interface WebRtcObservation {
  status: NetworkStatus
  detail: 'differentExit' | 'sameExit' | 'publicOnly' | 'localOnly' | 'noCandidates' | 'unsupported' | 'notRequested'
  publicAddresses: string[]
  localCount: number
  mdnsCount: number
}
export interface CliReport {
  schema: 'needtoken.claude-network.v1'
  measured_at: string
  runtime: { node_major: number; timezone: string }
  proxy: { http: boolean; https: boolean; all: boolean; no_proxy: boolean; custom_base_url: boolean }
  dns: { status: 'ok' | 'error'; address_count: number }
  curl: { status: 'ok' | 'error' | 'unsupported'; exit_code: number; http_status?: number; ssl_verify_result?: number; time_ms?: number; peer_ip?: string }
  scope: 'curl_inherits_proxy_env'
}

export type LocalCheckGroup = 'connection' | 'features' | 'system'
export type LocalCheckId = 'secure' | 'online' | 'cookies' | 'storage' | 'crypto' | 'wasm' | 'fetch' | 'abort' | 'intl' | 'language' | 'timezone' | 'locale' | 'clock' | 'platform' | 'browser'
export interface ClaudeLocalCheck {
  id: LocalCheckId
  group: LocalCheckGroup
  status: NetworkStatus
  detail: string
  value?: string
}
export type ClaudeLocalEnvironment = Partial<Record<LocalCheckId, () => unknown>>
export type NetworkCheckId = 'storefront' | 'exit' | 'anthropic' | 'rtc'
export type NetworkCheckEvent =
  | { id: 'storefront'; result: ConnectionDiagnostic }
  | { id: 'exit'; result: ExitNetwork }
  | { id: 'anthropic'; result: AnthropicConnection }
  | { id: 'rtc'; result: WebRtcObservation }

/** Each getter is isolated: a restricted browser feature cannot prevent a run. */
export function collectClaudeLocalChecks(environment?: ClaudeLocalEnvironment): ClaudeLocalCheck[] {
  const defaults: ClaudeLocalEnvironment = {
    secure: () => globalThis.isSecureContext,
    online: () => navigator.onLine,
    cookies: () => navigator.cookieEnabled && probeCookie(document, globalThis.isSecureContext),
    storage: () => probeStorage(() => localStorage),
    crypto: () => !!globalThis.crypto?.subtle,
    wasm: () => typeof WebAssembly !== 'undefined',
    fetch: () => typeof fetch === 'function',
    abort: () => typeof AbortController === 'function',
    intl: () => typeof Intl.DateTimeFormat === 'function',
    language: () => (navigator.languages?.length ? navigator.languages : [navigator.language]).slice(0, 5).join(', '),
    timezone: () => Intl.DateTimeFormat().resolvedOptions().timeZone,
    locale: () => Intl.DateTimeFormat().resolvedOptions().locale,
    clock: () => new Date().toISOString(),
    platform: () => {
      const agent = navigator.userAgent
      return /Windows/i.test(agent) ? 'Windows' : /Android/i.test(agent) ? 'Android' : /iPhone|iPad|iPod/i.test(agent) ? 'iOS' : /Macintosh|Mac OS X/i.test(agent) ? 'macOS' : /Linux/i.test(agent) ? 'Linux' : ''
    },
    browser: () => {
      const agent = navigator.userAgent
      return /Edg\//.test(agent) ? 'Microsoft Edge' : /Firefox\//.test(agent) ? 'Firefox' : /Chrome\//.test(agent) ? 'Chrome / Chromium' : /Safari\//.test(agent) ? 'Safari' : ''
    },
  }
  const readers = environment ?? defaults
  const read = (id: LocalCheckId) => { try { return readers[id]?.() } catch { return undefined } }
  const capabilities: { id: LocalCheckId; group: LocalCheckGroup; yes: string; no: string }[] = [
    { id: 'secure', group: 'connection', yes: 'secureReady', no: 'secureLimited' },
    { id: 'online', group: 'connection', yes: 'onlineHint', no: 'offlineHint' },
    { id: 'cookies', group: 'features', yes: 'cookiesReady', no: 'cookiesLimited' },
    { id: 'storage', group: 'features', yes: 'storageReady', no: 'storageLimited' },
    { id: 'crypto', group: 'features', yes: 'cryptoReady', no: 'cryptoLimited' },
    { id: 'wasm', group: 'features', yes: 'wasmReady', no: 'wasmLimited' },
    { id: 'fetch', group: 'features', yes: 'featureReady', no: 'featureLimited' },
    { id: 'abort', group: 'features', yes: 'featureReady', no: 'featureLimited' },
    { id: 'intl', group: 'features', yes: 'featureReady', no: 'featureLimited' },
  ]
  const checks: ClaudeLocalCheck[] = capabilities.map(({ id, group, yes, no }) => {
    const value = read(id)
    return { id, group, status: typeof value !== 'boolean' ? 'unknown' : value ? 'ok' : 'warning', detail: typeof value !== 'boolean' ? 'featureUnknown' : value ? yes : no }
  })
  for (const id of ['language', 'timezone', 'locale', 'clock', 'platform', 'browser'] as const) {
    const raw = read(id), value = typeof raw === 'string' ? raw.trim().slice(0, 160) : ''
    checks.push({ id, group: 'system', status: value && id !== 'clock' ? 'ok' : 'unknown', detail: id === 'clock' ? 'clockLocalOnly' : id === 'platform' || id === 'browser' ? 'browserReported' : 'localOnly', ...(value ? { value } : {}) })
  }
  return checks
}

/** Every fixed probe settles independently, including when fetch ignores abort. */
export async function runClaudeNetworkChecks(options: {
  onResult: (event: NetworkCheckEvent) => void
  includeWebRtc?: boolean
  signal?: AbortSignal
  fetcher?: typeof fetch
  timeoutMs?: number
  rtcFactory?: () => RTCPeerConnection
}): Promise<void> {
  if (options.signal?.aborted) throw abortError()
  const timeoutMs = options.timeoutMs ?? 2500
  let expectedIp: string | undefined
  let rtcRaw: { addresses: string[]; mdnsCount: number; supported: boolean } | undefined
  const notify = (event: NetworkCheckEvent) => { if (!options.signal?.aborted) options.onResult(event) }
  const publishRtc = () => {
    if (!rtcRaw) return
    notify({ id: 'rtc', result: rtcRaw.supported ? classifyWebRtc(rtcRaw.addresses, rtcRaw.mdnsCount, expectedIp) : { status: 'unknown', detail: 'unsupported', publicAddresses: [], localCount: 0, mdnsCount: 0 } })
  }
  const tasks = [
    Promise.resolve().then(() => probeStorefrontConnection({ ...options, timeoutMs })).then(result => notify({ id: 'storefront', result })).catch(() => notify({ id: 'storefront', result: { status: 'unknown', detail: 'networkError' } })),
    Promise.resolve().then(() => lookupExitNetwork({ ...options, timeoutMs })).then(result => { expectedIp = result.ip; notify({ id: 'exit', result }); publishRtc() }).catch(() => notify({ id: 'exit', result: { status: 'unknown', detail: 'lookupUnavailable', source: '', measuredAt: new Date().toISOString() } })),
    Promise.resolve().then(() => probeAnthropic({ ...options, timeoutMs })).then(result => notify({ id: 'anthropic', result })).catch(() => notify({ id: 'anthropic', result: { status: 'unknown', detail: 'requestUnconfirmed', measuredAt: new Date().toISOString() } })),
  ]
  if (options.includeWebRtc) tasks.push(Promise.resolve().then(() => probeWebRtc({ signal: options.signal, timeoutMs, factory: options.rtcFactory })).then(result => { rtcRaw = result; publishRtc() }).catch(() => notify({ id: 'rtc', result: { status: 'unknown', detail: 'unsupported', publicAddresses: [], localCount: 0, mdnsCount: 0 } })))
  await Promise.allSettled(tasks)
  if (options.signal?.aborted) throw abortError()
}

const abortError = () => new DOMException('Aborted', 'AbortError')
const text = (value: unknown, limit = 100) => typeof value === 'string' ? value.trim().slice(0, limit) : ''
const record = (value: unknown): Record<string, unknown> => value && typeof value === 'object' && !Array.isArray(value) ? value as Record<string, unknown> : {}

export function normalizeIp(value: unknown): string | null {
  const candidate = text(value, 64).toLowerCase()
  if (/^\d{1,3}(?:\.\d{1,3}){3}$/.test(candidate)) {
    const parts = candidate.split('.')
    if (parts.every(part => Number(part) <= 255 && String(Number(part)) === part)) return parts.join('.')
  }
  if (!candidate.includes(':') || !/^[a-f0-9:.]+$/.test(candidate)) return null
  try { return new URL(`http://[${candidate}]/`).hostname.slice(1, -1) } catch { return null }
}

export function isPublicIp(value: string): boolean {
  const ip = normalizeIp(value)
  if (!ip) return false
  if (ip.includes(':')) {
    if (ip.startsWith('::ffff:')) {
      const mapped = ip.slice(7)
      if (mapped.includes('.')) return isPublicIp(mapped)
      return false
    }
    return /^[23]/.test(ip) && !ip.startsWith('2001:db8:')
  }
  const [a = 0, b = 0, c = 0] = ip.split('.').map(Number)
  return !(a === 0 || a === 10 || a === 127 || a >= 224 || (a === 100 && b >= 64 && b <= 127) ||
    (a === 169 && b === 254) || (a === 172 && b >= 16 && b <= 31) || (a === 192 && b === 168) ||
    (a === 192 && b === 0) || (a === 198 && (b === 18 || b === 19 || (b === 51 && c === 100))) ||
    (a === 203 && b === 0 && c === 113))
}

/** A deadline races the entire fetch/body operation, even if a mock ignores abort. */
async function deadline<T>(task: (signal: AbortSignal) => Promise<T>, signal?: AbortSignal, timeoutMs = 6000): Promise<T> {
  if (signal?.aborted) throw abortError()
  const controller = new AbortController()
  let timer: ReturnType<typeof setTimeout> | undefined
  let onAbort: (() => void) | undefined
  const stop = new Promise<never>((_resolve, reject) => {
    onAbort = () => { controller.abort(); reject(abortError()) }
    signal?.addEventListener('abort', onAbort, { once: true })
    timer = setTimeout(() => { controller.abort(); reject(new Error('timeout')) }, timeoutMs)
  })
  try { return await Promise.race([task(controller.signal), stop]) }
  finally { if (timer) clearTimeout(timer); if (onAbort) signal?.removeEventListener('abort', onAbort) }
}

export async function lookupExitNetwork(options: { fetcher?: typeof fetch; signal?: AbortSignal; timeoutMs?: number } = {}): Promise<ExitNetwork> {
  const fetcher = options.fetcher ?? fetch
  const request = (url: string) => deadline(async signal => {
    const response = await fetcher(url, { signal, credentials: 'omit', referrerPolicy: 'no-referrer', cache: 'no-store', mode: 'cors', redirect: 'error' })
    if (!response.ok) throw new Error('unavailable')
    return record(await response.json())
  }, options.signal, options.timeoutMs ?? 6000)
  try {
    const data = await request(CLAUDE_PROBES.geo)
    const ip = normalizeIp(data.ip)
    if (data.error || !ip || !isPublicIp(ip)) throw new Error('invalid IP response')
    return {
      status: 'ok', detail: 'geoAvailable', source: 'ipapi.co', ip,
      country: text(data.country_name), countryCode: /^[A-Z]{2}$/.test(text(data.country_code)) ? text(data.country_code) : '',
      region: text(data.region), city: text(data.city), asn: text(data.asn, 40), organization: text(data.org), timezone: text(data.timezone),
      measuredAt: new Date().toISOString(),
    }
  } catch {
    if (options.signal?.aborted) throw abortError()
  }
  try {
    const data = await request(CLAUDE_PROBES.ip)
    const ip = normalizeIp(data.ip)
    if (!ip || !isPublicIp(ip)) throw new Error('invalid IP response')
    return { status: 'unknown', detail: 'ipOnly', source: 'api.ipify.org', ip, measuredAt: new Date().toISOString() }
  } catch {
    if (options.signal?.aborted) throw abortError()
    return { status: 'unknown', detail: 'lookupUnavailable', source: '', measuredAt: new Date().toISOString() }
  }
}

export async function probeAnthropic(options: { fetcher?: typeof fetch; signal?: AbortSignal; timeoutMs?: number; now?: () => number } = {}): Promise<AnthropicConnection> {
  const fetcher = options.fetcher ?? fetch
  const now = options.now ?? (() => performance.now())
  const start = now()
  try {
    const response = await deadline(signal => fetcher(CLAUDE_PROBES.anthropic, {
      signal, mode: 'no-cors', credentials: 'omit', referrerPolicy: 'no-referrer', cache: 'no-store', redirect: 'error',
    }), options.signal, options.timeoutMs ?? 8000)
    const latencyMs = Math.max(0, Math.round(now() - start))
    if (response.type === 'opaque' || response.type === 'opaqueredirect' || response.status === 0) {
      return { status: 'unknown', detail: 'opaqueResponse', latencyMs, measuredAt: new Date().toISOString() }
    }
    const warning = response.status === 403 || response.status === 429 || response.status >= 500
    return { status: warning ? 'warning' : 'ok', detail: warning ? 'httpWarning' : 'readableResponse', httpStatus: response.status, latencyMs, measuredAt: new Date().toISOString() }
  } catch {
    if (options.signal?.aborted) throw abortError()
    return { status: 'unknown', detail: 'requestUnconfirmed', measuredAt: new Date().toISOString() }
  }
}

export function classifyWebRtc(addresses: string[], mdnsCount: number, expectedIp?: string): WebRtcObservation {
  const normalized = [...new Set(addresses.map(normalizeIp).filter((ip): ip is string => !!ip))]
  const publicAddresses = normalized.filter(isPublicIp).slice(0, 6)
  const localCount = normalized.length - publicAddresses.length
  const expected = normalizeIp(expectedIp)
  const different = !!expected && publicAddresses.some(ip => ip !== expected)
  return {
    status: different ? 'warning' : publicAddresses.length && expected ? 'ok' : 'unknown',
    detail: different ? 'differentExit' : publicAddresses.length ? expected ? 'sameExit' : 'publicOnly' : localCount || mdnsCount ? 'localOnly' : 'noCandidates',
    publicAddresses, localCount, mdnsCount,
  }
}

export async function probeWebRtc(options: { signal?: AbortSignal; timeoutMs?: number; factory?: () => RTCPeerConnection } = {}): Promise<{ addresses: string[]; mdnsCount: number; supported: boolean }> {
  if (options.signal?.aborted) throw abortError()
  if (!options.factory && typeof RTCPeerConnection === 'undefined') return { addresses: [], mdnsCount: 0, supported: false }
  return new Promise((resolve, reject) => {
    let pc: RTCPeerConnection | undefined
    const addresses: string[] = []
    let mdnsCount = 0
    let timer: ReturnType<typeof setTimeout> | undefined
    let settled = false
    const finish = (aborted = false) => {
      if (settled) return
      settled = true
      if (timer) clearTimeout(timer)
      options.signal?.removeEventListener('abort', onAbort)
      if (pc) { pc.onicecandidate = null; pc.onicegatheringstatechange = null; pc.close() }
      if (aborted) reject(abortError())
      else resolve({ addresses, mdnsCount, supported: !!pc })
    }
    const onAbort = () => finish(true)
    options.signal?.addEventListener('abort', onAbort, { once: true })
    timer = setTimeout(() => finish(), options.timeoutMs ?? 6000)
    try {
      pc = options.factory ? options.factory() : new RTCPeerConnection({ iceServers: [{ urls: CLAUDE_PROBES.stun }], iceCandidatePoolSize: 0 })
      pc.onicecandidate = event => {
        const candidate = event.candidate?.candidate
        if (!candidate) return
        const value = candidate.split(/\s+/)[4] || ''
        if (value.endsWith('.local')) mdnsCount++
        else if (normalizeIp(value)) addresses.push(value)
      }
      pc.onicegatheringstatechange = () => { if (pc?.iceGatheringState === 'complete') finish() }
      pc.createDataChannel('network-check')
      void pc.createOffer().then(offer => pc?.setLocalDescription(offer)).catch(() => finish())
    } catch { finish() }
  })
}

export function timezoneDiffers(browserTimezone: string, exit: ExitNetwork | null): boolean {
  return !!browserTimezone && !!exit?.timezone && browserTimezone !== exit.timezone
}

/** A missing CMS category cannot turn a specialist notice panel into all blogs. */
export function claudeRuleCategoryIds(raw: unknown): number[] {
  if (!Array.isArray(raw)) return []
  const rows = raw.map(record).filter(row => Number.isSafeInteger(row.id) && Number(row.id) > 0)
  const root = rows.find(row => row.slug === 'claude-code-rules' && !row.parent_id)
  if (!root) return []
  return rows.filter(row => row.id === root.id || row.parent_id === root.id).map(row => Number(row.id))
}

export function parseCliReport(raw: string): CliReport | null {
  if (raw.length > 65536) return null
  try {
    const data = record(JSON.parse(raw))
    const runtime = record(data.runtime), proxy = record(data.proxy), dns = record(data.dns), curl = record(data.curl)
    if (data.schema !== 'needtoken.claude-network.v1' || data.scope !== 'curl_inherits_proxy_env' || !/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d{1,3})?(?:Z|[+-]\d{2}:\d{2})$/.test(text(data.measured_at)) || !Number.isFinite(Date.parse(text(data.measured_at)))) return null
    if (!['ok', 'error'].includes(String(dns.status)) || !['ok', 'error', 'unsupported'].includes(String(curl.status))) return null
    const nodeMajor = Number(runtime.node_major), count = Number(dns.address_count)
    if (typeof runtime.node_major !== 'number' || typeof dns.address_count !== 'number' || !Number.isInteger(nodeMajor) || nodeMajor < 18 || nodeMajor > 100 || !Number.isInteger(count) || count < 0 || count > 100 || (dns.status === 'ok' && count === 0)) return null
    if (['http', 'https', 'all', 'no_proxy', 'custom_base_url'].some(key => typeof proxy[key] !== 'boolean')) return null
    const status = Number(curl.http_status), exit = Number(curl.exit_code), ssl = Number(curl.ssl_verify_result), time = Number(curl.time_ms)
    if (typeof curl.exit_code !== 'number' || !Number.isInteger(exit) || exit < -1 || exit > 255) return null
    if (curl.status === 'ok' && (exit !== 0 || typeof curl.http_status !== 'number' || status < 100 || status > 599 || !Number.isInteger(status) || curl.ssl_verify_result !== 0)) return null
    return {
      schema: 'needtoken.claude-network.v1', scope: 'curl_inherits_proxy_env', measured_at: text(data.measured_at),
      runtime: { node_major: nodeMajor, timezone: text(runtime.timezone) },
      proxy: { http: proxy.http === true, https: proxy.https === true, all: proxy.all === true, no_proxy: proxy.no_proxy === true, custom_base_url: proxy.custom_base_url === true },
      dns: { status: dns.status as 'ok' | 'error', address_count: count },
      curl: { status: curl.status as 'ok' | 'error' | 'unsupported', exit_code: exit,
        ...(Number.isInteger(status) && status >= 100 && status <= 599 ? { http_status: status } : {}),
        ...(Number.isInteger(ssl) && ssl >= 0 && ssl <= 1000 ? { ssl_verify_result: ssl } : {}),
        ...(Number.isFinite(time) && time >= 0 && time <= 30000 ? { time_ms: Math.round(time) } : {}),
        ...(normalizeIp(curl.peer_ip) ? { peer_ip: normalizeIp(curl.peer_ip)! } : {}) },
    }
  } catch { return null }
}

export function claudeNetworkReport(data: { exit: ExitNetwork | null; anthropic: AnthropicConnection | null; rtc: WebRtcObservation | null; cli: CliReport | null; measuredAt: string; local?: ClaudeLocalCheck[]; storefront?: ConnectionDiagnostic | null }): string {
  return JSON.stringify({
    schema: 'needtoken.claude-network.browser.v1', measured_at: data.measuredAt,
    local_checks: (data.local ?? []).map(({ id, group, status }) => ({ id, group, status })),
    storefront_connection: data.storefront ?? null,
    exit: data.exit ? { status: data.exit.status, detail: data.exit.detail, source: data.exit.source, measured_at: data.exit.measuredAt } : null,
    anthropic_browser: data.anthropic,
    webrtc: data.rtc ? { status: data.rtc.status, detail: data.rtc.detail, public_address_count: data.rtc.publicAddresses.length, local_count: data.rtc.localCount, mdns_count: data.rtc.mdnsCount } : null,
    cli: data.cli ? { scope: data.cli.scope, measured_at: data.cli.measured_at, dns: data.cli.dns, curl: { status: data.cli.curl.status, exit_code: data.cli.curl.exit_code, http_status: data.cli.curl.http_status, ssl_verify_result: data.cli.curl.ssl_verify_result, time_ms: data.cli.curl.time_ms }, proxy_configured: data.cli.proxy.http || data.cli.proxy.https || data.cli.proxy.all } : null,
    ip_reputation: 'not_tested', account_eligibility: 'not_tested', shared_data: 'statuses_only_no_ip_timezone_language_or_secrets',
  }, null, 2)
}
