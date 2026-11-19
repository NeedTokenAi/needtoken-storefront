import { test } from 'node:test'
import assert from 'node:assert/strict'
import { fetchResetFeed, parseResetFeed, RESET_FEED_MAX_AGE_MS, RESET_FEED_PATH, RESET_FEED_SOURCE, resetActivityDays, resetCalendar, resetCalendarWithRecords, resetElapsedTime, resetFeedIsExpired, resetGapDays, resetRecordPage, resetSourceUrl, resetStatistics, type ResetRecord } from '../src/templates/needtoken/lib/resetRadar.ts'

// Synthetic contract fixtures. These do not verify a live upstream or account delivery.
const fixtureRecord = (overrides: Record<string, unknown> = {}) => ({
  id: 'synthetic-public-post', type: 'regular', announced_at: '2026-11-14T08:00:00Z', text: 'Synthetic reset announcement',
  source: { type: 'x_post', author: 'fixture_author', url: 'https://x.com/fixture_author/status/100000001' }, ...overrides,
})
const fixtureFeed = (records: unknown[] = [fixtureRecord()], overrides: Record<string, unknown> = {}) => ({
  source_url: RESET_FEED_SOURCE, fetched_at: '2026-11-14T09:00:00Z', stale: false, has_more: false, records, ...overrides,
})
const response = (value: unknown, status = 200, contentType = 'application/json') => new Response(JSON.stringify(value), { status, headers: { 'content-type': contentType } })

test('announcement snapshots preserve source timing, observations, and the server successful-fetch timestamp', () => {
  const snapshot = parseResetFeed(fixtureFeed([
    fixtureRecord(),
    fixtureRecord({ id: 'synthetic-observation', type: 'banked', announced_at: '2026-11-14T08:30:00Z', source: { type: 'observed', author: null, url: null } }),
  ], { stale: true, has_more: true, generated_at: '2099-01-01T00:00:00Z' }))
  assert.equal(snapshot.fetched_at, '2026-11-14T09:00:00.000Z')
  assert.equal(snapshot.stale, true)
  assert.equal(snapshot.has_more, true)
  assert.equal(snapshot.records[0]?.id, 'synthetic-observation')
  assert.deepEqual(snapshot.records[0]?.source, { type: 'observed', author: '', url: '' })
  assert.equal(snapshot.records[0]?.announced_at, '2026-11-14T08:30:00.000Z')
})

test('source links accept only canonical HTTPS X/Twitter posts', () => {
  for (const url of ['javascript:alert(1)', '//x.com/fixture/status/1', 'http://x.com/fixture/status/1', 'https://user:pass@x.com/fixture/status/1', 'https://x.com.evil.test/fixture/status/1', 'https://x.com:8443/fixture/status/1', 'https://x.com/fixture/status/1?redirect=evil', 'https://x.com/fixture/status/1#fragment', 'https://x.com/fixture', 'https://x.com/fixture/status/not-a-number', 'https://x.com/abcdefghijklmnop/status/1']) assert.equal(resetSourceUrl(url), '')
  assert.equal(resetSourceUrl('https://x.com/fixture_author/status/100000001'), 'https://x.com/fixture_author/status/100000001')
  assert.equal(resetSourceUrl('https://twitter.com/fixture_author/status/100000001'), 'https://twitter.com/fixture_author/status/100000001')
})

test('invalid post links remove public-post records but retain clearly typed observations without links', () => {
  const snapshot = parseResetFeed(fixtureFeed([
    fixtureRecord({ source: { type: 'x_post', url: 'javascript:alert(1)' } }),
    fixtureRecord({ id: 'synthetic-observation', source: { type: 'observed', url: 'https://evil.test/unsafe' } }),
  ]))
  assert.deepEqual(snapshot.records.map(row => row.id), ['synthetic-observation'])
  assert.equal(snapshot.records[0]?.source.url, '')
})

test('malformed dates, unknown classifications, blank text and duplicate IDs cannot become displayed records', () => {
  const snapshot = parseResetFeed(fixtureFeed([
    fixtureRecord(), fixtureRecord(), fixtureRecord({ id: 'bad-date', announced_at: 'not a date' }),
    fixtureRecord({ id: 'local-date', announced_at: '2026-11-14T09:00:00' }),
    fixtureRecord({ id: 'unknown-type', type: 'scheduled' }), fixtureRecord({ id: 'blank', text: '  ' }),
    fixtureRecord({ id: 'unknown-source', source: { type: 'official', url: 'https://x.com/fixture/status/1' } }),
  ]))
  assert.equal(snapshot.records.length, 1)
  assert.throws(() => parseResetFeed(fixtureFeed([fixtureRecord({ announced_at: 'broken' })])))
})

test('incomplete schemas do not produce a fresh empty snapshot or trust an alternate source', () => {
  for (const value of [null, [], {}, fixtureFeed([], { source_url: 'https://codex-resets.com.evil.test/' }), fixtureFeed([], { fetched_at: '' }), fixtureFeed([], { stale: 'false' }), fixtureFeed([], { has_more: undefined }), fixtureFeed([], { records: null })]) assert.throws(() => parseResetFeed(value))
  assert.deepEqual(parseResetFeed(fixtureFeed([])).records, [])
})

test('a bounded recent sample is sorted and capped without inventing a full-history count', () => {
  const rows = Array.from({ length: 105 }, (_, index) => fixtureRecord({ id: `synthetic-${index}`, announced_at: new Date(Date.UTC(2026, 10, 1) + index * 60_000).toISOString() }))
  const snapshot = parseResetFeed(fixtureFeed(rows))
  assert.equal(snapshot.records.length, 100)
  assert.equal(snapshot.records[0]?.id, 'synthetic-104')
  assert.equal(snapshot.has_more, true)
  assert.equal('total' in snapshot, false)
  assert.equal('avg_interval_days' in snapshot, false)
})

test('the twelve-hour display cutoff uses the last successful source fetch, including the exact boundary', () => {
  const feed = parseResetFeed(fixtureFeed())
  const fetchedAt = Date.parse(feed.fetched_at)
  assert.equal(resetFeedIsExpired(feed, fetchedAt + RESET_FEED_MAX_AGE_MS - 1), false)
  assert.equal(resetFeedIsExpired(feed, fetchedAt + RESET_FEED_MAX_AGE_MS), true)
  assert.equal(resetFeedIsExpired(feed, fetchedAt + RESET_FEED_MAX_AGE_MS + 60_000), true)
  assert.equal(resetFeedIsExpired({ fetched_at: 'invalid' }, fetchedAt), true)
})

test('elapsed display is since the record, clamps future timestamps, and does not predict a next reset', () => {
  const announced = '2026-11-14T08:00:00Z'
  assert.deepEqual(resetElapsedTime(announced, Date.parse(announced) + (1440 + 62) * 60_000), { days: 1, hours: 1, minutes: 2 })
  assert.deepEqual(resetElapsedTime(announced, Date.parse(announced) - 60_000), { days: 0, hours: 0, minutes: 0 })
  assert.deepEqual(resetElapsedTime('invalid'), { days: 0, hours: 0, minutes: 0 })
})

test('a newer reset card anchors the latest record clock while quota-only averages remain separately scoped', () => {
  const now = Date.parse('2026-11-16T02:00:00Z')
  const records = parseResetFeed(fixtureFeed([
    fixtureRecord({ id: 'regular-oct28', announced_at: '2026-10-28T06:29:38Z' }),
    fixtureRecord({ id: 'banked-nov13', type: 'banked', announced_at: '2026-11-12T21:59:39Z', text: 'Synthetic November reset card' }),
  ])).records
  const latest = records[0]!
  assert.equal(latest.id, 'banked-nov13')
  assert.equal(latest.text, 'Synthetic November reset card')
  assert.equal(latest.announced_at, '2026-11-12T21:59:39.000Z')
  assert.deepEqual(resetElapsedTime(latest.announced_at, now), { days: 3, hours: 4, minutes: 0 })
  const stats = resetStatistics(records, now)
  assert.deepEqual([stats.total, stats.regular, stats.banked, stats.recent30], [2, 1, 1, 2])
  assert.equal(stats.lastResetAt, Date.parse('2026-10-28T06:29:38Z'))
})

test('public feed requests stay same-origin, omit credentials, reject redirects and pass cancellation', async () => {
  const controller = new AbortController()
  const fetcher: typeof fetch = async (url, options) => {
    assert.equal(url, RESET_FEED_PATH)
    assert.equal(options?.method, 'GET')
    assert.equal(options?.mode, 'same-origin')
    assert.equal(options?.credentials, 'omit')
    assert.equal(options?.redirect, 'error')
    assert.equal(options?.cache, 'no-store')
    assert.equal(options?.signal, controller.signal)
    return response({ status_code: 0, data: fixtureFeed() })
  }
  const result = await fetchResetFeed(controller.signal, fetcher)
  assert.equal(result.records.length, 1)
})

test('HTTP or business failures and SPA HTML responses never count as successful fetches', async () => {
  const controller = new AbortController()
  for (const result of [response({ status_code: 503, data: { error_code: 'unavailable' } }), response({ status_code: 0, data: fixtureFeed() }, 503), response({ status_code: 0, data: fixtureFeed() }, 200, 'text/html'), response({ status_code: 0, data: fixtureFeed() }, 200, 'application/jsonp'), response({ data: fixtureFeed() })]) {
    await assert.rejects(fetchResetFeed(controller.signal, async () => result))
  }
  await assert.rejects(fetchResetFeed(controller.signal, async () => { throw new TypeError('Synthetic offline failure') }))
})


test('historical intervals and elapsed time use actual usage resets, not newer reset-credit announcements', () => {
  const feed = parseResetFeed(fixtureFeed([
    fixtureRecord({ id: 'regular-older', announced_at: '2026-11-01T00:00:00Z' }),
    fixtureRecord({ id: 'regular-latest', announced_at: '2026-11-07T00:00:00Z' }),
    fixtureRecord({ id: 'credit-newer', type: 'banked', announced_at: '2026-11-08T00:00:00Z' }),
    fixtureRecord({ id: 'future-reset', announced_at: '2026-11-20T00:00:00Z' }),
  ]))
  const stats = resetStatistics(feed.records, Date.parse('2026-11-09T00:00:00Z'))
  assert.equal(stats.lastResetAt, Date.parse('2026-11-07T00:00:00Z'))
  assert.equal(stats.averageIntervalDays, 6)
  assert.equal(stats.elapsedDays, 2)
  assert.equal(stats.averageRemainingDays, 4)
  assert.deepEqual([stats.total, stats.regular, stats.banked], [3, 2, 1])
  const creditOnly = resetStatistics(feed.records.filter(record => record.type === 'banked'))
  assert.equal(creditOnly.averageIntervalDays, null)
  assert.equal(creditOnly.elapsedDays, null)
})

test('rolling thirty-day counts keep the previous interval separate without counting future records', () => {
  const now = Date.parse('2026-11-15T00:00:00Z')
  const rows = [0, 29, 30, 59, 60, -1].map((days, index) => fixtureRecord({ id: `window-${index}`, announced_at: new Date(now - days * 86400000).toISOString() }))
  const stats = resetStatistics(parseResetFeed(fixtureFeed(rows)).records, now)
  assert.equal(stats.recent30, 2)
  assert.equal(stats.previous30, 2)
})

test('reset calendar groups UTC+8 days, keeps mixed reset types, and marks uncovered dates honestly', () => {
  const rows = parseResetFeed(fixtureFeed([
    fixtureRecord({ id: 'regular-day', announced_at: '2026-11-13T17:00:00Z' }),
    fixtureRecord({ id: 'credit-same-day', type: 'banked', announced_at: '2026-11-14T15:00:00Z' }),
    fixtureRecord({ id: 'next-local-day', announced_at: '2026-11-14T16:00:00Z' }),
  ])).records
  const weeks = resetCalendar(rows, Date.parse('2026-11-15T04:00:00Z'))
  assert.equal(new Date(`${weeks[0]![0]!.date}T00:00:00Z`).getUTCDay(), 1)
  const days = weeks.flat()
  const mixedDay = days.find(day => day.date === '2026-11-14')!
  assert.deepEqual([mixedDay.regular, mixedDay.banked, mixedDay.inCoverage], [1, 1, true])
  assert.equal(days.find(day => day.date === '2026-11-15')?.regular, 1)
  assert.equal(days.find(day => day.date === '2026-11-13')?.inCoverage, false)
  assert.equal(resetCalendar(rows, Date.parse('2026-11-14T04:00:00Z')).flat().find(day => day.date === '2026-11-15')?.future, true)
})

test('each reset-history filter starts with ten records and expands in ten-record pages', () => {
  const rows = parseResetFeed(fixtureFeed(Array.from({ length: 45 }, (_, index) => fixtureRecord({ id: `page-${index}`, type: index % 2 ? 'regular' : 'banked' })))).records
  for (const type of ['all', 'regular', 'banked'] as const) {
    const first = resetRecordPage(rows, type)
    const second = resetRecordPage(rows, type, 20)
    assert.equal(first.records.length, 10)
    assert.equal(second.records.length, 20)
    assert.equal(first.remaining - second.remaining, 10)
    assert.ok(type === 'all' || first.records.every(record => record.type === type))
  }
})

test('activity days follow Beijing midnight rather than UTC midnight and include today', () => {
  const days = resetActivityDays([
    { type: 'regular', announced_at: '2026-11-14T15:59:59.999Z' },
    { type: 'regular', announced_at: '2026-11-14T16:00:00Z' },
    { type: 'banked', announced_at: '2026-11-15T00:01:00+08:00' },
  ], Date.parse('2026-11-14T16:05:00Z'))
  assert.equal(days.length, 30)
  assert.deepEqual(days[0], { date: '2026-10-16T16:00:00.000Z', regular: 0, banked: 0 })
  assert.deepEqual(days.at(-2), { date: '2026-11-13T16:00:00.000Z', regular: 1, banked: 0 })
  assert.deepEqual(days.at(-1), { date: '2026-11-14T16:00:00.000Z', regular: 1, banked: 1 })
})

test('activity includes the exact 30-day boundary, ignores invalid and future times, and is order independent', () => {
  const rows: Pick<ResetRecord, 'type' | 'announced_at'>[] = [
    { type: 'banked', announced_at: '2026-10-16T16:00:00Z' },
    { type: 'regular', announced_at: '2026-10-16T15:59:59.999Z' },
    { type: 'regular', announced_at: '2026-11-14T16:05:00Z' },
    { type: 'banked', announced_at: '2026-11-14T16:05:00.001Z' },
    { type: 'regular', announced_at: 'invalid' },
    { type: 'regular', announced_at: '2026-11-14T00:00:00' },
  ]
  const now = Date.parse('2026-11-14T16:05:00Z')
  const days = resetActivityDays(rows, now)
  assert.deepEqual(days, resetActivityDays([...rows].reverse(), now))
  assert.deepEqual(days[0], { date: '2026-10-16T16:00:00.000Z', regular: 0, banked: 1 })
  assert.deepEqual(days.at(-1), { date: '2026-11-14T16:00:00.000Z', regular: 1, banked: 0 })
  assert.equal(days.reduce((total, day) => total + day.regular + day.banked, 0), 2)
  assert.equal(rows[0]?.announced_at, '2026-10-16T16:00:00Z', 'input remains in its original order')
})

test('rich reset calendar attaches day records, marks today, and calculates inter-record day gaps', () => {
  const rows = parseResetFeed(fixtureFeed([
    fixtureRecord({ id: 'latest', type: 'banked', announced_at: '2026-11-12T21:59:00Z', text: 'Banked reset activated' }),
    fixtureRecord({ id: 'prev', type: 'regular', announced_at: '2026-11-01T23:27:00Z', text: 'Full reset' }),
  ])).records
  const now = Date.parse('2026-11-19T12:00:00Z')
  const weeks = resetCalendarWithRecords(rows, now)
  const flat = weeks.flat()
  const nov13 = flat.find(day => day.date === '2026-11-13')!
  assert.equal(nov13.banked, 1)
  assert.equal(nov13.records[0]?.text, 'Banked reset activated')
  assert.equal(flat.find(day => day.date === '2026-11-19')?.isToday, true)
  assert.equal(resetGapDays(rows[0]!.announced_at, rows[1]!.announced_at), 11)
  assert.equal(resetGapDays(rows[0]!.announced_at, undefined), null)
})
