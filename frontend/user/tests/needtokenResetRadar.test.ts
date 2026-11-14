import { test } from 'node:test'
import assert from 'node:assert/strict'
import { fetchResetFeed, parseResetFeed, RESET_FEED_MAX_AGE_MS, RESET_FEED_PATH, RESET_FEED_SOURCE, resetElapsedTime, resetFeedIsExpired, resetSourceUrl } from '../src/templates/needtoken/lib/resetRadar.ts'

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
