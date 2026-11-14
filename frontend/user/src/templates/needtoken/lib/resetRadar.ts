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
