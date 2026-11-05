/** Browser-only diagnostics. Never sends local capability values to a server. */
export type DiagnosticStatus = 'ok' | 'warning' | 'unknown'
export type DiagnosticId = 'secure' | 'online' | 'cookies' | 'storage' | 'crypto' | 'wasm' | 'language' | 'timezone'

export interface LocalDiagnostic {
  id: DiagnosticId
  status: DiagnosticStatus
  detail: string
  value?: string
}

interface StorageProbe {
  setItem(key: string, value: string): void
  getItem(key: string): string | null
  removeItem(key: string): void
}

export interface DiagnosticEnvironment {
  secure: boolean
  online: boolean
  cookiesEnabled: boolean
  cookieDocument: { cookie: string }
  storage: () => StorageProbe
  cryptoAvailable: boolean
  wasmAvailable: boolean
  language: string
  timezone: () => string
}

export function probeStorage(storage: () => StorageProbe): boolean {
  const key = `nt-diagnostics-${Date.now()}-${Math.random().toString(36).slice(2)}`
  let target: StorageProbe | undefined
  try {
    target = storage()
    target.setItem(key, '1')
    return target.getItem(key) === '1'
  } catch {
    return false
  } finally {
    try { target?.removeItem(key) } catch { /* Restricted storage is a diagnostic result. */ }
  }
}

export function probeCookie(document: { cookie: string }, secure: boolean): boolean {
  const key = `nt_diagnostics_${Date.now()}_${Math.random().toString(36).slice(2)}`
  const attributes = `; Path=/; SameSite=Lax${secure ? '; Secure' : ''}`
  try {
    document.cookie = `${key}=1; Max-Age=60${attributes}`
    return document.cookie.split(';').some((part) => part.trim() === `${key}=1`)
  } catch {
    return false
  } finally {
    try { document.cookie = `${key}=; Max-Age=0${attributes}` } catch { /* No persistent probe cookie. */ }
  }
}

export function collectLocalDiagnostics(environment: DiagnosticEnvironment): LocalDiagnostic[] {
  let timezone = ''
  try { timezone = environment.timezone() } catch { /* Browser may limit timezone access. */ }
  const cookies = environment.cookiesEnabled && probeCookie(environment.cookieDocument, environment.secure)
  const storage = probeStorage(environment.storage)
  return [
    { id: 'secure', status: environment.secure ? 'ok' : 'warning', detail: environment.secure ? 'secureReady' : 'secureLimited' },
    { id: 'online', status: environment.online ? 'ok' : 'warning', detail: environment.online ? 'onlineHint' : 'offlineHint' },
    { id: 'cookies', status: cookies ? 'ok' : 'warning', detail: cookies ? 'cookiesReady' : 'cookiesLimited' },
    { id: 'storage', status: storage ? 'ok' : 'warning', detail: storage ? 'storageReady' : 'storageLimited' },
    { id: 'crypto', status: environment.cryptoAvailable ? 'ok' : 'warning', detail: environment.cryptoAvailable ? 'cryptoReady' : 'cryptoLimited' },
    { id: 'wasm', status: environment.wasmAvailable ? 'ok' : 'warning', detail: environment.wasmAvailable ? 'wasmReady' : 'wasmLimited' },
    { id: 'language', status: environment.language ? 'ok' : 'unknown', detail: 'localOnly', value: environment.language.slice(0, 40) },
    { id: 'timezone', status: timezone ? 'ok' : 'unknown', detail: 'localOnly', value: timezone.slice(0, 80) },
  ]
}

export type ConnectionCode = 'reachable' | 'httpError' | 'invalidResponse' | 'timeout' | 'networkError'
export interface ConnectionDiagnostic {
  status: DiagnosticStatus
  detail: ConnectionCode
  latencyMs?: number
  httpStatus?: number
}

const abortedError = () => new DOMException('Aborted', 'AbortError')

/** Tests only this application's fixed public health endpoint; no user-provided URL. */
export async function probeStorefrontConnection(options: {
  fetcher?: typeof fetch
  signal?: AbortSignal
  timeoutMs?: number
  now?: () => number
} = {}): Promise<ConnectionDiagnostic> {
  const { fetcher = fetch, signal, timeoutMs = 5000, now = () => performance.now() } = options
  if (signal?.aborted) throw abortedError()
  const controller = new AbortController()
  let timer: ReturnType<typeof setTimeout> | undefined
  let onAbort: (() => void) | undefined
  const started = now()
  const deadline = new Promise<ConnectionDiagnostic>((resolve, reject) => {
    onAbort = () => {
      controller.abort()
      reject(abortedError())
    }
    signal?.addEventListener('abort', onAbort, { once: true })
    timer = setTimeout(() => {
      controller.abort()
      resolve({ status: 'unknown', detail: 'timeout' })
    }, Math.max(1, timeoutMs))
  })
  const request = (async (): Promise<ConnectionDiagnostic> => {
    try {
      const response = await fetcher('/health', {
        cache: 'no-store', credentials: 'omit', redirect: 'error', mode: 'same-origin', signal: controller.signal,
        headers: { Accept: 'application/json' },
      })
      if (!response.ok) return { status: 'warning', detail: 'httpError', httpStatus: response.status }
      if (!response.headers.get('content-type')?.toLowerCase().includes('application/json')) {
        return { status: 'unknown', detail: 'invalidResponse' }
      }
      let data: unknown
      try { data = await response.json() } catch { return { status: 'unknown', detail: 'invalidResponse' } }
      if (!data || typeof data !== 'object' || !('status' in data) || data.status !== 'ok') {
        return { status: 'unknown', detail: 'invalidResponse' }
      }
      return { status: 'ok', detail: 'reachable', latencyMs: Math.max(0, Math.round(now() - started)) }
    } catch {
      if (signal?.aborted) throw abortedError()
      return { status: 'unknown', detail: controller.signal.aborted ? 'timeout' : 'networkError' }
    }
  })()
  try {
    return await Promise.race([deadline, request])
  } finally {
    if (timer !== undefined) clearTimeout(timer)
    if (onAbort) signal?.removeEventListener('abort', onAbort)
  }
}

/** Shared reports omit local language/timezone, cookies, keys, URLs, and exact user agent. */
export function redactedDiagnosticReport(local: LocalDiagnostic[], connection: ConnectionDiagnostic | null, measuredAt: string): string {
  return JSON.stringify({
    tool: 'NeedToken.AI browser diagnostics',
    measured_at: measuredAt,
    scope: 'local browser capabilities and same-origin storefront health only',
    local_checks: local.map(({ id, status }) => ({ id, status })),
    storefront_connection: connection,
    account_eligibility: 'not_tested',
    ip_reputation: 'not_tested',
    external_service_connectivity: 'not_tested',
  }, null, 2)
}
