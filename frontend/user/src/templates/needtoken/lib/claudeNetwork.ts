/** Original network diagnostics, informed by LinXiaoTao/FuckClaude's separation
 * of local observations and IP reputation. No account eligibility score is inferred. */
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

export function claudeNetworkReport(data: { exit: ExitNetwork | null; anthropic: AnthropicConnection | null; rtc: WebRtcObservation | null; cli: CliReport | null; measuredAt: string }): string {
  return JSON.stringify({
    schema: 'needtoken.claude-network.browser.v1', measured_at: data.measuredAt,
    exit: data.exit ? { status: data.exit.status, detail: data.exit.detail, source: data.exit.source, measured_at: data.exit.measuredAt } : null,
    anthropic_browser: data.anthropic,
    webrtc: data.rtc ? { status: data.rtc.status, detail: data.rtc.detail, public_address_count: data.rtc.publicAddresses.length, local_count: data.rtc.localCount, mdns_count: data.rtc.mdnsCount } : null,
    cli: data.cli ? { scope: data.cli.scope, measured_at: data.cli.measured_at, dns: data.cli.dns, curl: { status: data.cli.curl.status, exit_code: data.cli.curl.exit_code, http_status: data.cli.curl.http_status, ssl_verify_result: data.cli.curl.ssl_verify_result, time_ms: data.cli.curl.time_ms }, proxy_configured: data.cli.proxy.http || data.cli.proxy.https || data.cli.proxy.all } : null,
    ip_reputation: 'not_tested', account_eligibility: 'not_tested', shared_data: 'statuses_only_no_ip_timezone_language_or_secrets',
  }, null, 2)
}
