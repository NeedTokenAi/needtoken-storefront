/** User-supplied credentials stay in the caller's memory; this module never persists or logs them. */
export const MAX_CODEX_SESSION_INPUT = 65_536
const MAX_TOKEN_LENGTH = 16_384
const MAX_RESPONSE_LENGTH = 65_536
const ACCOUNT_ID = /^[A-Za-z0-9_-]{1,128}$/

export type SessionInputError = 'empty' | 'tooLarge' | 'invalidJson' | 'missingToken' | 'invalidToken'
export type SessionInput = { ok: true; accessToken: string } | { ok: false; error: SessionInputError }

const record = (value: unknown): Record<string, unknown> | null =>
  value !== null && typeof value === 'object' && !Array.isArray(value) ? value as Record<string, unknown> : null

const jwtObject = (segment: string): Record<string, unknown> | null => {
  try {
    const bytes = Uint8Array.from(atob(segment.replace(/-/g, '+').replace(/_/g, '/')), char => char.charCodeAt(0))
    return record(JSON.parse(new TextDecoder('utf-8', { fatal: true }).decode(bytes)))
  } catch { return null }
}

/** Structural validation only. JWT claims are never used as verified account/subscription evidence. */
export function parseCodexSessionInput(value: string): SessionInput {
  if (value.length > MAX_CODEX_SESSION_INPUT) return { ok: false, error: 'tooLarge' }
  let token = value.trim()
  if (!token) return { ok: false, error: 'empty' }
  if (token.startsWith('{') || token.startsWith('[')) {
    let parsed: unknown
    try { parsed = JSON.parse(token) } catch { return { ok: false, error: 'invalidJson' } }
    const session = record(parsed)
    if (!session) return { ok: false, error: 'missingToken' }
    const aliases = [session.accessToken, session.access_token].filter(item => item !== undefined)
    if (!aliases.length) return { ok: false, error: 'missingToken' }
    if (aliases.some(item => typeof item !== 'string') || new Set(aliases).size !== 1) return { ok: false, error: 'invalidToken' }
    token = String(aliases[0]).trim()
  }
  token = token.replace(/^Bearer[ \t]+/i, '')
  if (token.length > MAX_TOKEN_LENGTH || !/^[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+$/.test(token)) return { ok: false, error: 'invalidToken' }
  const [headerPart = '', payloadPart = '', signaturePart = ''] = token.split('.')
  const header = jwtObject(headerPart)
  if (!header || typeof header.alg !== 'string' || !/^[A-Za-z0-9_-]{1,20}$/.test(header.alg) || header.alg.toLowerCase() === 'none' || !jwtObject(payloadPart) || signaturePart.length < 16) return { ok: false, error: 'invalidToken' }
  return { ok: true, accessToken: token }
}

export interface CodexAccount {
  id: string
  plan: string
  isDefault: boolean
  expiresAt: string | null
}
export interface CodexUsageWindow {
  usedPercent: number
  windowSeconds: number
  resetAt: string | null
}
export interface CodexAccountResult {
  checkedAt: string
  accounts: CodexAccount[]
  selectedAccountId: string | null
  usageStatus: 'not_requested' | 'ready' | 'unavailable'
  usage: { primary: CodexUsageWindow | null; secondary: CodexUsageWindow | null } | null
}

export type CodexCheckErrorCode = 'invalid_input' | 'unauthorized' | 'challenge' | 'unavailable' | 'invalid_response' | 'account_not_found' | 'rate_limited' | 'timeout'
const ERROR_CODES = new Set<CodexCheckErrorCode>(['invalid_input', 'unauthorized', 'challenge', 'unavailable', 'invalid_response', 'account_not_found', 'rate_limited', 'timeout'])
export class CodexCheckError extends Error {
  code: CodexCheckErrorCode
  constructor(code: CodexCheckErrorCode) { super(code); this.name = 'CodexCheckError'; this.code = code }
}

const timestamp = (value: unknown): string | null => {
  if (typeof value !== 'string' || value.length > 64 || !value.trim()) return null
  const date = new Date(value)
  return Number.isFinite(date.getTime()) ? date.toISOString() : null
}
const usageWindow = (value: unknown): CodexUsageWindow | null => {
  const row = record(value)
  if (!row || typeof row.used_percent !== 'number' || !Number.isFinite(row.used_percent) || row.used_percent < 0 || row.used_percent > 100 ||
    typeof row.window_seconds !== 'number' || !Number.isSafeInteger(row.window_seconds) || row.window_seconds <= 0 || row.window_seconds > 366 * 86_400) return null
  return { usedPercent: row.used_percent, windowSeconds: row.window_seconds, resetAt: timestamp(row.reset_at) }
}

/** Explicit allowlist: unexpected upstream fields (including credential or identity fields) are discarded. */
export function parseCodexAccountResult(value: unknown): CodexAccountResult | null {
  const data = record(value)
  const checkedAt = timestamp(data?.checked_at)
  if (!data || !checkedAt || !Array.isArray(data.accounts) || data.accounts.length < 1 || data.accounts.length > 100) return null
  const accounts: CodexAccount[] = []
  const ids = new Set<string>()
  for (const candidate of data.accounts) {
    const account = record(candidate)
    if (!account || typeof account.id !== 'string' || !ACCOUNT_ID.test(account.id) || ids.has(account.id)) return null
    ids.add(account.id)
    accounts.push({ id: account.id, plan: typeof account.plan === 'string' && /^[a-z][a-z0-9_-]{0,39}$/.test(account.plan) ? account.plan : 'unknown', isDefault: account.is_default === true, expiresAt: timestamp(account.expires_at) })
  }
  const selectedAccountId = typeof data.selected_account_id === 'string' && ids.has(data.selected_account_id) ? data.selected_account_id : null
  if (data.selected_account_id !== undefined && !selectedAccountId) return null
  if (!['not_requested', 'ready', 'unavailable'].includes(String(data.usage_status))) return null
  const usage = record(data.usage)
  const primary = usageWindow(usage?.primary)
  const secondary = usageWindow(usage?.secondary)
  const usageStatus = data.usage_status as CodexAccountResult['usageStatus']
  if (usageStatus !== 'not_requested' && !selectedAccountId) return null
  if (usageStatus === 'ready' && (!usage || (!primary && !secondary))) return null
  return { checkedAt, accounts, selectedAccountId, usageStatus, usage: usageStatus === 'ready' ? { primary, secondary } : null }
}

const abortError = () => new DOMException('Aborted', 'AbortError')
export async function checkCodexAccount(options: {
  accessToken: string
  accountId?: string
  locale?: string
  baseUrl?: string
  signal?: AbortSignal
  timeoutMs?: number
  fetcher?: typeof fetch
}): Promise<CodexAccountResult> {
  const credential = parseCodexSessionInput(options.accessToken)
  if (!credential.ok || (options.accountId !== undefined && !ACCOUNT_ID.test(options.accountId))) throw new CodexCheckError('invalid_input')
  if (options.signal?.aborted) throw abortError()
  const controller = new AbortController()
  let timer: ReturnType<typeof setTimeout> | undefined
  let onAbort: (() => void) | undefined
  const stop = new Promise<never>((_resolve, reject) => {
    onAbort = () => { controller.abort(); reject(abortError()) }
    options.signal?.addEventListener('abort', onAbort, { once: true })
    timer = setTimeout(() => { reject(new CodexCheckError('timeout')); controller.abort() }, options.timeoutMs ?? 18_000)
  })
  const request = async () => {
    const response = await (options.fetcher ?? fetch)(`${(options.baseUrl ?? '').replace(/\/$/, '')}/api/v1/public/tools/codex-account`, {
      method: 'POST', signal: controller.signal, credentials: 'omit', cache: 'no-store', redirect: 'error', referrerPolicy: 'no-referrer',
      headers: { 'content-type': 'application/json', accept: 'application/json', ...(options.locale ? { 'X-Lang': options.locale } : {}) },
      body: JSON.stringify({ access_token: credential.accessToken, ...(options.accountId ? { account_id: options.accountId } : {}) }),
    })
    if (Number(response.headers.get('content-length')) > MAX_RESPONSE_LENGTH) throw new CodexCheckError('invalid_response')
    const raw = await response.text()
    if (raw.length > MAX_RESPONSE_LENGTH) throw new CodexCheckError('invalid_response')
    let payload: Record<string, unknown> | null
    try { payload = record(JSON.parse(raw)) } catch { throw new CodexCheckError(response.status === 429 ? 'rate_limited' : 'unavailable') }
    if (!response.ok || payload?.status_code !== 0) {
      const code = record(payload?.data)?.error_code
      const fallback: CodexCheckErrorCode = response.status === 429 ? 'rate_limited' : response.status === 401 ? 'unauthorized' : 'unavailable'
      throw new CodexCheckError(ERROR_CODES.has(code as CodexCheckErrorCode) ? code as CodexCheckErrorCode : fallback)
    }
    const result = parseCodexAccountResult(payload.data)
    if (!result) throw new CodexCheckError('invalid_response')
    return result
  }
  try { return await Promise.race([request(), stop]) }
  catch (error) {
    if (options.signal?.aborted) throw abortError()
    if (error instanceof CodexCheckError) throw error
    throw new CodexCheckError('unavailable')
  } finally {
    if (timer) clearTimeout(timer)
    if (onAbort) options.signal?.removeEventListener('abort', onAbort)
  }
}
