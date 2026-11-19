/** Public announcement records, never a personal account quota or delivery feed. */
export const RESET_FEED_PATH = '/api/v1/public/tools/codex-resets'
export const RESET_FEED_SOURCE = 'https://codex-resets.com/'
export const RESET_FEED_POLL_MS = 60_000
export const RESET_FEED_MAX_AGE_MS = 12 * 60 * 60 * 1000

export interface ResetRecord {
  id: string
  type: 'regular' | 'banked'
  announced_at: string
  text: string
  source: { type: 'x_post' | 'observed'; author: string; url: string }
}

export interface ResetFeed {
  source_url: string
  /** Time our server successfully retrieved the source; not the source's scan time. */
  fetched_at: string
  stale: boolean
  has_more: boolean
  records: ResetRecord[]
}

const object = (value: unknown): Record<string, unknown> | null =>
  value !== null && typeof value === 'object' && !Array.isArray(value) ? value as Record<string, unknown> : null

function timestamp(value: unknown): string {
  if (typeof value !== 'string' || !/^\d{4}-\d{2}-\d{2}T.+(?:Z|[+-]\d{2}:\d{2})$/.test(value)) return ''
  return Number.isFinite(Date.parse(value)) ? new Date(value).toISOString() : ''
}

export function resetSourceUrl(value: unknown): string {
  if (typeof value !== 'string') return ''
  try {
    const url = new URL(value)
    if (url.protocol !== 'https:' || !['x.com', 'twitter.com'].includes(url.hostname) || url.username || url.password || url.port || url.search || url.hash) return ''
    return /^\/[a-zA-Z0-9_]{1,15}\/status\/\d+$/.test(url.pathname) ? url.toString() : ''
  } catch { return '' }
}

export function parseResetFeed(value: unknown): ResetFeed {
  const data = object(value)
  const fetchedAt = timestamp(data?.fetched_at)
  if (!data || ![RESET_FEED_SOURCE, RESET_FEED_SOURCE.slice(0, -1)].includes(String(data.source_url)) || !fetchedAt || typeof data.stale !== 'boolean' || typeof data.has_more !== 'boolean' || !Array.isArray(data.records)) throw new Error('invalid reset feed')
  const records: ResetRecord[] = []
  const ids = new Set<string>()
  for (const value of data.records) {
    const row = object(value)
    const source = object(row?.source)
    const announcedAt = timestamp(row?.announced_at)
    if (!row || !source || typeof row.id !== 'string' || !row.id.trim() || row.id.length > 128 || ids.has(row.id) || !['regular', 'banked'].includes(String(row.type)) || !['x_post', 'observed'].includes(String(source.type)) || !announcedAt || typeof row.text !== 'string' || !row.text.trim() || row.text.length > 20_000) continue
    const sourceUrl = resetSourceUrl(source.url)
    if (source.type === 'x_post' && !sourceUrl) continue
    ids.add(row.id)
    records.push({
      id: row.id,
      type: row.type as ResetRecord['type'],
      announced_at: announcedAt,
      text: row.text.trim(),
      source: {
        type: source.type as ResetRecord['source']['type'],
        author: typeof source.author === 'string' && /^[a-zA-Z0-9_]{1,15}$/.test(source.author) ? source.author : '',
        url: sourceUrl,
      },
    })
  }
  if (data.records.length && !records.length) throw new Error('invalid reset records')
  records.sort((a, b) => Date.parse(b.announced_at) - Date.parse(a.announced_at))
  return { source_url: RESET_FEED_SOURCE, fetched_at: fetchedAt, stale: data.stale, has_more: data.has_more || records.length > 100, records: records.slice(0, 100) }
}

export function resetFeedIsExpired(feed: Pick<ResetFeed, 'fetched_at'>, now = Date.now()): boolean {
  const fetched = Date.parse(feed.fetched_at)
  return !Number.isFinite(fetched) || now - fetched >= RESET_FEED_MAX_AGE_MS
}

export function resetElapsedTime(announcedAt: string, now = Date.now()) {
  const minutes = Math.max(0, Math.floor((now - Date.parse(announcedAt)) / 60_000)) || 0
  return { days: Math.floor(minutes / 1440), hours: Math.floor(minutes % 1440 / 60), minutes: minutes % 60 }
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
