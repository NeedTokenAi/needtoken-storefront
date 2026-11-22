/** Public announcement records, never a personal account quota or delivery feed. */
export const RESET_FEED_PATH = '/api/v1/public/tools/codex-resets'
export const RESET_FEED_SOURCE = 'https://codex-resets.com/'
export const CODEX_DIRECT_API_URL = 'https://codex-resets.com/api/v1/resets?limit=100'
export const TSALON_DIRECT_API_URL = 'https://www.tsalon.tech/api/reset/data'
export const TSALON_FEED_SOURCE = 'https://www.tsalon.tech/whenreset/'
export const ANTIGRAVITY_FEED_SOURCE = 'https://yidinghuiziyou.com/ai/antigravity-reset/'
export const CLAUDE_RESETS_REFERENCE = 'https://claude-resets.net/'
export const RESET_FEED_POLL_MS = 60_000
export const RESET_CLIENT_CACHE_TTL_MS = 5 * 60_000
export const RESET_FEED_MAX_AGE_MS = 12 * 60 * 60 * 1000

export type RadarChannelId = 'codex' | 'claude' | 'gemini' | 'grok'

export interface RadarChannelMeta {
  id: RadarChannelId
  brand: 'gpt' | 'claude' | 'gemini' | 'grok'
}

export const RADAR_CHANNELS: ReadonlyArray<RadarChannelMeta> = [
  { id: 'codex', brand: 'gpt' },
  { id: 'claude', brand: 'claude' },
  { id: 'gemini', brand: 'gemini' },
  { id: 'grok', brand: 'grok' },
]

export function channelResetFeedPath(channel: RadarChannelId): string {
  return `/api/v1/public/tools/resets/${channel}`
}

export interface ResetRecord {
  id: string
  type: 'regular' | 'banked'
  announced_at: string
  text: string
  text_zh?: string
  scope?: string
  scope_zh?: string
  source: { type: 'x_post' | 'observed'; author: string; url: string }
}

export interface ResetFeed {
  source_url: string
  /** Time the snapshot was retrieved (by browser direct fetch or server cache). */
  fetched_at: string
  stale: boolean
  has_more: boolean
  delivery?: 'direct' | 'cached_proxy'
  records: ResetRecord[]
}

const ALLOWED_SOURCE_URLS: Record<RadarChannelId, readonly string[]> = {
  codex: [RESET_FEED_SOURCE, RESET_FEED_SOURCE.slice(0, -1), TSALON_FEED_SOURCE, TSALON_FEED_SOURCE.slice(0, -1)],
  claude: [TSALON_FEED_SOURCE, TSALON_FEED_SOURCE.slice(0, -1), CLAUDE_RESETS_REFERENCE, CLAUDE_RESETS_REFERENCE.slice(0, -1)],
  gemini: [ANTIGRAVITY_FEED_SOURCE, ANTIGRAVITY_FEED_SOURCE.slice(0, -1), TSALON_FEED_SOURCE, TSALON_FEED_SOURCE.slice(0, -1)],
  grok: [TSALON_FEED_SOURCE, TSALON_FEED_SOURCE.slice(0, -1)],
}

const object = (value: unknown): Record<string, unknown> | null =>
  value !== null && typeof value === 'object' && !Array.isArray(value) ? value as Record<string, unknown> : null

function timestamp(value: unknown): string {
  if (typeof value !== 'string' || !/^\d{4}-\d{2}-\d{2}T.+(?:Z|[+-]\d{2}:\d{2})$/.test(value)) return ''
  return Number.isFinite(Date.parse(value)) ? new Date(value).toISOString() : ''
}

function parseBeijingTimestamp(value: unknown): string {
  if (typeof value !== 'string') return ''
  const trimmed = value.trim()
  const iso = timestamp(trimmed)
  if (iso) return iso
  const match = /^(\d{4}-\d{2}-\d{2})\s+(\d{2}:\d{2})(?::(\d{2}))?$/.exec(trimmed)
  if (!match) return ''
  const secs = match[3] || '00'
  const candidate = `${match[1]}T${match[2]}:${secs}+08:00`
  return Number.isFinite(Date.parse(candidate)) ? new Date(candidate).toISOString() : ''
}

function optionalText(value: unknown, placeholder?: string): string | undefined {
  if (typeof value !== 'string') return undefined
  const trimmed = value.trim()
  if (!trimmed || trimmed === '--' || (placeholder && trimmed === placeholder)) return undefined
  return trimmed.slice(0, 2000)
}

export function resetSourceUrl(value: unknown): string {
  if (typeof value !== 'string') return ''
  try {
    const url = new URL(value)
    if (url.protocol !== 'https:' || !['x.com', 'twitter.com'].includes(url.hostname) || url.username || url.password || url.port || url.search || url.hash) return ''
    return /^\/[a-zA-Z0-9_]{1,15}\/status\/\d+$/.test(url.pathname) ? url.toString() : ''
  } catch { return '' }
}

function sanitizePostUrlAndAuthor(value: unknown): { url: string; author: string } {
  if (typeof value !== 'string') return { url: '', author: '' }
  try {
    const u = new URL(value.trim())
    const host = u.hostname.toLowerCase()
    if (u.protocol !== 'https:' || !['x.com', 'twitter.com'].includes(host) || u.username || u.password || u.port) {
      return { url: '', author: '' }
    }
    const cleanPath = u.pathname.replace(/\/$/, '')
    const match = /^\/([a-zA-Z0-9_]{1,15})\/status\/(\d+)$/.exec(cleanPath)
    if (!match) return { url: '', author: '' }
    return { url: `https://${host}${cleanPath}`, author: match[1]! }
  } catch {
    return { url: '', author: '' }
  }
}

function parseRecordsArray(rawRecords: unknown[]): ResetRecord[] {
  const records: ResetRecord[] = []
  const ids = new Set<string>()
  for (const value of rawRecords) {
    const row = object(value)
    const source = object(row?.source)
    const announcedAt = timestamp(row?.announced_at)
    if (!row || !source || typeof row.id !== 'string' || !row.id.trim() || row.id.length > 128 || ids.has(row.id) || !['regular', 'banked'].includes(String(row.type)) || !['x_post', 'observed'].includes(String(source.type)) || !announcedAt || typeof row.text !== 'string' || !row.text.trim() || row.text.length > 20_000) continue
    const sourceUrl = resetSourceUrl(source.url)
    if (source.type === 'x_post' && !sourceUrl) continue
    ids.add(row.id)
    const rec: ResetRecord = {
      id: row.id,
      type: row.type as ResetRecord['type'],
      announced_at: announcedAt,
      text: row.text.trim(),
      source: {
        type: source.type as ResetRecord['source']['type'],
        author: typeof source.author === 'string' && /^[a-zA-Z0-9_]{1,15}$/.test(source.author) ? source.author : '',
        url: sourceUrl,
      },
    }
    const textZh = optionalText(row.text_zh)
    const scope = optionalText(row.scope)
    const scopeZh = optionalText(row.scope_zh)
    if (textZh) rec.text_zh = textZh
    if (scope) rec.scope = scope
    if (scopeZh) rec.scope_zh = scopeZh
    records.push(rec)
  }
  records.sort((a, b) => Date.parse(b.announced_at) - Date.parse(a.announced_at))
  return records
}

export function parseResetFeed(value: unknown): ResetFeed {
  return parseChannelResetFeed(value, 'codex', [RESET_FEED_SOURCE, RESET_FEED_SOURCE.slice(0, -1)])
}

export function parseChannelResetFeed(
  value: unknown,
  channel: RadarChannelId,
  allowedSources: readonly string[] = ALLOWED_SOURCE_URLS[channel],
): ResetFeed {
  const data = object(value)
  const fetchedAt = timestamp(data?.fetched_at)
  const rawSource = String(data?.source_url ?? '')
  if (!data || !allowedSources.includes(rawSource) || !fetchedAt || typeof data.stale !== 'boolean' || typeof data.has_more !== 'boolean' || !Array.isArray(data.records)) {
    throw new Error('invalid reset feed')
  }
  const records = parseRecordsArray(data.records)
  if (data.records.length && !records.length) throw new Error('invalid reset records')
  const normalizedSource = rawSource.endsWith('/') ? rawSource : `${rawSource}/`
  const delivery = data.delivery === 'direct' || data.delivery === 'cached_proxy' ? data.delivery : undefined
  return {
    source_url: normalizedSource,
    fetched_at: fetchedAt,
    stale: data.stale,
    has_more: data.has_more || records.length > 100,
    ...(delivery ? { delivery } : {}),
    records: records.slice(0, 100),
  }
}

/** Parse direct browser response from https://codex-resets.com/api/v1/resets?limit=100 */
export function parseCodexDirectPayload(value: unknown, now = Date.now()): ResetFeed {
  const data = object(value)
  const pagination = object(data?.pagination)
  if (!data || !Array.isArray(data.data) || typeof pagination?.has_more !== 'boolean') {
    throw new Error('invalid direct codex feed')
  }
  const normalizedRows = data.data.map(item => {
    const row = object(item)
    if (!row) return null
    return {
      id: row.id,
      type: row.reset_type,
      announced_at: row.announced_at,
      text: row.text,
      source: row.source,
    }
  })
  const records = parseRecordsArray(normalizedRows)
  if (data.data.length && !records.length) throw new Error('invalid direct codex records')
  return {
    source_url: RESET_FEED_SOURCE,
    fetched_at: new Date(now).toISOString(),
    stale: false,
    has_more: pagination.has_more || records.length > 100,
    delivery: 'direct',
    records: records.slice(0, 100),
  }
}

/** Parse direct browser response from https://www.tsalon.tech/api/reset/data (claude / grok / codex). */
export function parseTSalonPayload(value: unknown, now = Date.now()): Partial<Record<RadarChannelId, ResetFeed>> {
  const data = object(value)
  if (!data || (data.status && data.status !== 'ready')) {
    throw new Error('invalid tsalon feed')
  }
  const fetchedAt = new Date(now).toISOString()
  const out: Partial<Record<RadarChannelId, ResetFeed>> = {}
  for (const channel of ['codex', 'claude', 'grok'] as const) {
    const block = object(data[channel])
    if (!block || !Array.isArray(block.events)) continue
    const records: ResetRecord[] = []
    const ids = new Set<string>()
    for (const rawEv of block.events) {
      const ev = object(rawEv)
      if (!ev || typeof ev.id !== 'string' || !ev.id.trim() || ev.id.length > 128 || ids.has(ev.id)) continue
      const announcedAt = parseBeijingTimestamp(ev.landedAtBeijing)
      if (!announcedAt) continue
      const isCard = String(ev.type || '').toLowerCase() === 'card' || String(ev.category || '').toLowerCase().includes('banked')
      const kind: ResetRecord['type'] = isCard ? 'banked' : 'regular'
      const post = sanitizePostUrlAndAuthor(ev.postUrl)
      const reasonEn = optionalText(ev.reasonEn, 'Not specified in this record')
      const reasonZh = optionalText(ev.reasonZh, '该记录未说明')
      const scope = optionalText(ev.scopeEn, 'Not specified')
      const scopeZh = optionalText(ev.scopeZh, '未说明')
      const text = reasonEn || optionalText(ev.typeLabelEn) || (kind === 'banked' ? 'Reset card announcement' : 'Global quota reset')
      const textZh = reasonZh || optionalText(ev.typeLabelZh)
      ids.add(ev.id)
      const rec: ResetRecord = {
        id: ev.id.trim(),
        type: kind,
        announced_at: announcedAt,
        text,
        source: {
          type: post.url ? 'x_post' : 'observed',
          author: post.author,
          url: post.url,
        },
      }
      if (textZh) rec.text_zh = textZh
      if (scope) rec.scope = scope
      if (scopeZh) rec.scope_zh = scopeZh
      records.push(rec)
    }
    records.sort((a, b) => Date.parse(b.announced_at) - Date.parse(a.announced_at))
    out[channel] = {
      source_url: TSALON_FEED_SOURCE,
      fetched_at: fetchedAt,
      stale: false,
      has_more: records.length > 100,
      delivery: 'direct',
      records: records.slice(0, 100),
    }
  }
  return out
}

export function resetFeedIsExpired(feed: Pick<ResetFeed, 'fetched_at'>, now = Date.now()): boolean {
  const fetched = Date.parse(feed.fetched_at)
  return !Number.isFinite(fetched) || now - fetched >= RESET_FEED_MAX_AGE_MS
}

export function resetElapsedTime(announcedAt: string, now = Date.now()) {
  const minutes = Math.max(0, Math.floor((now - Date.parse(announcedAt)) / 60_000)) || 0
  return { days: Math.floor(minutes / 1440), hours: Math.floor(minutes % 1440 / 60), minutes: minutes % 60 }
}

export interface GeminiDailyResetSchedule {
  isPacificDst: boolean
  resetHourBeijing: string
  hoursLeft: number
  minutesLeft: number
  nextResetIso: string
}

/** Pure client-side Pacific midnight RPD countdown (0 network requests). */
export function geminiDailyResetSchedule(now = Date.now()): GeminiDailyResetSchedule {
  const safeNow = Number.isFinite(now) ? now : Date.now()
  const parts = new Intl.DateTimeFormat('en-US', {
    timeZone: 'America/Los_Angeles',
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
    hourCycle: 'h23',
    timeZoneName: 'shortOffset',
  }).formatToParts(new Date(safeNow))
  const part = (type: string) => parts.find(p => p.type === type)?.value || ''
  const tzOffsetRaw = part('timeZoneName') // e.g. "GMT-7" or "GMT-8"
  const isPacificDst = tzOffsetRaw.includes('-7')
  const offsetHours = isPacificDst ? -7 : -8
  const resetHourBeijing = isPacificDst ? '15:00' : '16:00'

  const hour = Number(part('hour')) || 0
  const minute = Number(part('minute')) || 0
  const second = Number(part('second')) || 0
  const elapsedSecToday = hour * 3600 + minute * 60 + second
  const remainingSec = Math.max(60, 86_400 - elapsedSecToday)
  const hoursLeft = Math.floor(remainingSec / 3600)
  const minutesLeft = Math.floor((remainingSec % 3600) / 60)

  const year = Number(part('year')) || 2026
  const month = Number(part('month')) || 1
  const day = Number(part('day')) || 1
  const nextMidnightUtcMs = Date.UTC(year, month - 1, day + 1, -offsetHours, 0, 0)
  return {
    isPacificDst,
    resetHourBeijing,
    hoursLeft,
    minutesLeft,
    nextResetIso: new Date(nextMidnightUtcMs).toISOString(),
  }
}

const DAY_MS = 86_400_000
const BEIJING_OFFSET_MS = 8 * 60 * 60_000
export const RESET_RECORD_PAGE_SIZE = 10

/** Statistics describe the fetched sample, never an account's next reset. */
export function resetStatistics(records: readonly ResetRecord[], now = Date.now()) {
  const known = records.filter(record => Date.parse(record.announced_at) <= now)
  const regularTimes = [...new Set(known.filter(record => record.type === 'regular').map(record => Date.parse(record.announced_at)))].sort((a, b) => b - a)
  const lastResetAt = regularTimes[0] ?? null
  const averageIntervalDays = regularTimes.length > 1 ? (regularTimes[0]! - regularTimes[regularTimes.length - 1]!) / (regularTimes.length - 1) / DAY_MS : null
  const elapsedDays = lastResetAt === null ? null : Math.max(0, (now - lastResetAt) / DAY_MS)
  return {
    total: known.length,
    regular: known.filter(record => record.type === 'regular').length,
    banked: known.filter(record => record.type === 'banked').length,
    recent30: known.filter(record => Date.parse(record.announced_at) > now - 30 * DAY_MS).length,
    previous30: known.filter(record => Date.parse(record.announced_at) > now - 60 * DAY_MS && Date.parse(record.announced_at) <= now - 30 * DAY_MS).length,
    lastResetAt,
    elapsedDays,
    averageIntervalDays,
    averageRemainingDays: averageIntervalDays === null || elapsedDays === null ? null : Math.max(0, averageIntervalDays - elapsedDays),
  }
}

export interface ResetCalendarDay {
  date: string
  regular: number
  banked: number
  inCoverage: boolean
  future: boolean
}

export interface ResetCalendarRichDay extends ResetCalendarDay {
  isToday: boolean
  records: ResetRecord[]
}

/** Align weeks Monday-first and group records by the displayed UTC+8 day. */
export function resetCalendar(records: readonly ResetRecord[], now = Date.now()): ResetCalendarDay[][] {
  return resetCalendarWithRecords(records, now).map(week =>
    week.map(({ date, regular, banked, inCoverage, future }) => ({ date, regular, banked, inCoverage, future })),
  )
}

/** Same UTC+8 Monday-first annual calendar, enriched with each day's matching records for rich hover cards. */
export function resetCalendarWithRecords(records: readonly ResetRecord[], now = Date.now()): ResetCalendarRichDay[][] {
  const today = Math.floor((now + BEIJING_OFFSET_MS) / DAY_MS) * DAY_MS
  const todayStr = new Date(today).toISOString().slice(0, 10)
  const yearStart = today - 364 * DAY_MS
  const weekday = (new Date(yearStart).getUTCDay() + 6) % 7
  const start = yearStart - weekday * DAY_MS
  const earliest = records.reduce((min, record) => Math.min(min, Math.floor((Date.parse(record.announced_at) + BEIJING_OFFSET_MS) / DAY_MS) * DAY_MS), today)
  const grouped = new Map<string, { regular: number; banked: number; records: ResetRecord[] }>()
  for (const record of records) {
    if (Date.parse(record.announced_at) > now) continue
    const date = new Date(Date.parse(record.announced_at) + BEIJING_OFFSET_MS).toISOString().slice(0, 10)
    const current = grouped.get(date) ?? { regular: 0, banked: 0, records: [] }
    current[record.type]++
    current.records.push(record)
    grouped.set(date, current)
  }
  return Array.from({ length: Math.ceil(((today - start) / DAY_MS + 1) / 7) }, (_, week) => Array.from({ length: 7 }, (_, day) => {
    const time = start + (week * 7 + day) * DAY_MS
    const date = new Date(time).toISOString().slice(0, 10)
    const bucket = grouped.get(date)
    return {
      date,
      regular: bucket?.regular ?? 0,
      banked: bucket?.banked ?? 0,
      records: bucket?.records ?? [],
      inCoverage: records.length > 0 && time >= earliest && time <= today,
      future: time > today,
      isToday: date === todayStr,
    }
  }))
}

/** Whole calendar days between two announcement timestamps (newer - older), or null if invalid. */
export function resetGapDays(newerAnnouncedAt: string, olderAnnouncedAt: string | undefined): number | null {
  if (!olderAnnouncedAt) return null
  const newer = Date.parse(newerAnnouncedAt)
  const older = Date.parse(olderAnnouncedAt)
  if (!Number.isFinite(newer) || !Number.isFinite(older) || newer < older) return null
  return Math.max(0, Math.round((newer - older) / DAY_MS))
}

export function resetRecordPage(records: readonly ResetRecord[], type: 'all' | ResetRecord['type'], count = RESET_RECORD_PAGE_SIZE) {
  const filtered = records.filter(record => type === 'all' || record.type === type)
  const limit = Math.max(RESET_RECORD_PAGE_SIZE, Math.floor(count / RESET_RECORD_PAGE_SIZE) * RESET_RECORD_PAGE_SIZE)
  return { records: filtered.slice(0, limit), remaining: Math.max(0, filtered.length - limit) }
}

export async function fetchResetFeed(signal: AbortSignal, fetcher: typeof fetch = fetch): Promise<ResetFeed> {
  const response = await fetcher(RESET_FEED_PATH, { method: 'GET', mode: 'same-origin', credentials: 'omit', redirect: 'error', cache: 'no-store', headers: { Accept: 'application/json' }, signal })
  if (!response.ok || !/^application\/json(?:\s*;|$)/i.test(response.headers.get('content-type') || '')) throw new Error('reset feed unavailable')
  const envelope = object(await response.json())
  if (!envelope || envelope.status_code !== 0) throw new Error('reset feed unavailable')
  return parseResetFeed(envelope.data)
}

const memoryFeedCache = new Map<RadarChannelId, ResetFeed>()

function cacheStorageKey(channel: RadarChannelId): string {
  return `nt_reset_radar_v3_${channel}`
}

function defaultStorage(): Pick<Storage, 'getItem' | 'setItem'> | null {
  try {
    return typeof sessionStorage !== 'undefined' ? sessionStorage : null
  } catch {
    return null
  }
}

export function readCachedChannelFeed(
  channel: RadarChannelId,
  now = Date.now(),
  storage: Pick<Storage, 'getItem' | 'setItem'> | null = defaultStorage(),
): ResetFeed | null {
  const mem = memoryFeedCache.get(channel)
  if (mem && now - Date.parse(mem.fetched_at) < RESET_CLIENT_CACHE_TTL_MS) {
    return mem
  }
  if (!storage) return null
  try {
    const raw = storage.getItem(cacheStorageKey(channel))
    if (!raw) return null
    const parsed = parseChannelResetFeed(JSON.parse(raw), channel)
    if (now - Date.parse(parsed.fetched_at) < RESET_CLIENT_CACHE_TTL_MS) {
      memoryFeedCache.set(channel, parsed)
      return parsed
    }
  } catch {
    // Ignore malformed client cache entries.
  }
  return null
}

function writeCachedChannelFeed(
  channel: RadarChannelId,
  feed: ResetFeed,
  storage: Pick<Storage, 'getItem' | 'setItem'> | null = defaultStorage(),
) {
  memoryFeedCache.set(channel, feed)
  if (!storage) return
  try {
    storage.setItem(cacheStorageKey(channel), JSON.stringify(feed))
  } catch {
    // Ignore storage quota or privacy-mode errors.
  }
}

export function clearResetRadarClientCache() {
  memoryFeedCache.clear()
}

export interface FetchChannelResetOptions {
  force?: boolean
  fetcher?: typeof fetch
  storage?: Pick<Storage, 'getItem' | 'setItem'> | null
  now?: number
  directTimeoutMs?: number
}

async function fetchWithTimeout(
  url: string,
  init: RequestInit,
  parentSignal: AbortSignal,
  timeoutMs: number,
  fetcher: typeof fetch,
): Promise<Response> {
  const ctrl = new AbortController()
  const onParentAbort = () => ctrl.abort()
  if (parentSignal.aborted) {
    ctrl.abort()
  } else {
    parentSignal.addEventListener('abort', onParentAbort, { once: true })
  }
  const timer = setTimeout(() => ctrl.abort(), timeoutMs)
  try {
    return await fetcher(url, { ...init, signal: ctrl.signal })
  } finally {
    clearTimeout(timer)
    parentSignal.removeEventListener('abort', onParentAbort)
  }
}

/**
 * Direct-first multi-channel feed loader that minimizes origin traffic:
 * 1) Serve fresh (<5m) client sessionStorage/memory cache with 0 network requests.
 * 2) For CORS-enabled public feeds (Codex -> codex-resets.com; Claude/Grok -> tsalon.tech),
 *    let the visitor's browser fetch upstream directly (0 origin hits).
 * 3) Fall back to our Cloudflare-edge-cached `/api/v1/public/tools/resets/:channel`
 *    only when direct fetch is blocked/unavailable or upstream has no CORS (Gemini Antigravity).
 */
export async function fetchChannelResetFeed(
  channel: RadarChannelId,
  signal: AbortSignal,
  options: FetchChannelResetOptions = {},
): Promise<ResetFeed> {
  const fetcher = options.fetcher ?? fetch
  const storage = options.storage === undefined ? defaultStorage() : options.storage
  const now = options.now ?? Date.now()
  const directTimeoutMs = options.directTimeoutMs ?? 6_000

  if (!options.force) {
    const cached = readCachedChannelFeed(channel, now, storage)
    if (cached) return cached
  }

  // Step 1: Browser direct fetch for CORS-enabled public sources (zero origin load).
  if (channel === 'codex') {
    try {
      const resp = await fetchWithTimeout(
        CODEX_DIRECT_API_URL,
        { method: 'GET', mode: 'cors', credentials: 'omit', cache: 'default', headers: { Accept: 'application/json' } },
        signal,
        directTimeoutMs,
        fetcher,
      )
      if (resp.ok && /^application\/json(?:\s*;|$)/i.test(resp.headers.get('content-type') || '')) {
        const directFeed = parseCodexDirectPayload(await resp.json(), now)
        writeCachedChannelFeed('codex', directFeed, storage)
        return directFeed
      }
    } catch {
      if (signal.aborted) throw new Error('aborted')
    }
  } else if (channel === 'claude' || channel === 'grok') {
    try {
      const resp = await fetchWithTimeout(
        TSALON_DIRECT_API_URL,
        { method: 'GET', mode: 'cors', credentials: 'omit', cache: 'default', headers: { Accept: 'application/json' } },
        signal,
        directTimeoutMs,
        fetcher,
      )
      if (resp.ok && /^application\/json(?:\s*;|$)/i.test(resp.headers.get('content-type') || '')) {
        const parsedMap = parseTSalonPayload(await resp.json(), now)
        // One direct request populates both Claude and Grok caches simultaneously.
        for (const key of ['claude', 'grok'] as const) {
          if (parsedMap[key]) writeCachedChannelFeed(key, parsedMap[key]!, storage)
        }
        if (parsedMap[channel]) return parsedMap[channel]!
      }
    } catch {
      if (signal.aborted) throw new Error('aborted')
    }
  }

  // Step 2: Same-origin Cloudflare-edge-cached fallback (or primary for non-CORS Gemini Antigravity).
  const response = await fetcher(channelResetFeedPath(channel), {
    method: 'GET',
    mode: 'same-origin',
    credentials: 'omit',
    redirect: 'error',
    cache: 'default',
    headers: { Accept: 'application/json' },
    signal,
  })
  if (!response.ok || !/^application\/json(?:\s*;|$)/i.test(response.headers.get('content-type') || '')) {
    throw new Error('reset feed unavailable')
  }
  const envelope = object(await response.json())
  if (!envelope || envelope.status_code !== 0) throw new Error('reset feed unavailable')
  const feed = { ...parseChannelResetFeed(envelope.data, channel), delivery: 'cached_proxy' as const }
  writeCachedChannelFeed(channel, feed, storage)
  return feed
}

interface ResetUpdate { id: number; slug: string; title: Record<string, string>; published_at: string }
export function resetUpdates(value: unknown): ResetUpdate[] {
  if (!Array.isArray(value)) throw new Error('invalid reset updates')
  return value.filter((row): row is ResetUpdate => {
    if (!row || typeof row !== 'object' || !Number.isSafeInteger(row.id) || row.id < 1 || typeof row.slug !== 'string' || !row.slug.trim()) return false
    if (!row.title || typeof row.title !== 'object' || Array.isArray(row.title)) return false
    return ['zh-CN', 'zh-TW', 'en-US'].some(locale => typeof row.title[locale] === 'string' && row.title[locale].trim())
  }).map(row => ({ id: row.id, slug: row.slug, title: row.title, published_at: typeof row.published_at === 'string' && Number.isFinite(Date.parse(row.published_at)) ? row.published_at : '' })).slice(0, 10)
}

/** Count only supplied public records in the last 30 Beijing calendar days.
 * Empty days are sample counts, not a forecast or a claim of complete history. */
export function resetActivityDays(records: Pick<ResetRecord, 'type' | 'announced_at'>[], now = Date.now()): { date: string; regular: number; banked: number }[] {
  if (!Number.isFinite(now)) return []
  const dayMs = 24 * 60 * 60 * 1000
  const beijingOffset = 8 * 60 * 60 * 1000
  const today = Math.floor((now + beijingOffset) / dayMs) * dayMs - beijingOffset
  const firstDay = today - 29 * dayMs
  const days = Array.from({ length: 30 }, (_, index) => ({ date: new Date(firstDay + index * dayMs).toISOString(), regular: 0, banked: 0 }))
  for (const record of records) {
    const announced = Date.parse(timestamp(record.announced_at))
    if (!Number.isFinite(announced) || announced < firstDay || announced > now || !['regular', 'banked'].includes(record.type)) continue
    const day = days[Math.floor((announced - firstDay) / dayMs)]
    if (day) day[record.type] += 1
  }
  return days
}
