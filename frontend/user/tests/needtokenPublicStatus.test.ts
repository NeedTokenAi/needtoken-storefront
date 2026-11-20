import test from 'node:test'
import assert from 'node:assert/strict'
import { fallbackSeriesFromSummary, incidentElapsedMinutes, incidentPhase, mergePublicStatusIncidents, matchesTerms, operationalCount, parseCachedPublicStatus, parsePublicIncidents, parsePublicStatus, parsePublicStatusHistory, parsePublicStatusSeries, parseRetryAfterMs, publicHistoryDays, publicHistoryIncidents, publicIncidentTone, publicStatusColor, publicStatusFresh, publicStatusTone, serializeCachedPublicStatus, seriesOverallTone, serviceStatusComponents, serviceStatusIncidents, serviceStatusTone, shouldFetchStatus, statusRefreshDelayMs } from '../src/templates/needtoken/lib/publicStatus.ts'

test('parsePublicStatus keeps component states and does not invent uptime', () => {
  const view = parsePublicStatus({
    page: { name: 'OpenAI', url: 'https://status.openai.com/', updated_at: '2026-11-12T00:00:00Z' },
    status: { indicator: 'minor', description: 'Partial System Outage' },
    components: [
      { name: 'ChatGPT Work', status: 'operational' },
      { name: 'Codex in ChatGPT Desktop', status: 'degraded_performance' },
      { name: 'Mystery', status: 'invented' },
      { name: '   ', status: 'operational' },
    ],
    incidents: [{ name: 'Codex quota note', status: 'resolved', impact: 'minor', updated_at: 'not-a-date' }],
    uptime: 99.26,
  }, 'https://status.openai.com/')

  assert.equal(view.ok, true)
  if (!view.ok) return
  assert.equal(view.description, 'Partial System Outage')
  assert.equal(view.indicator, 'minor')
  assert.deepEqual(view.components.map(row => row.status), ['operational', 'degraded_performance', 'unknown'])
  assert.equal(view.incidents[0]?.updatedAt, null)
  assert.equal('uptime' in view, false)
  assert.equal('percent' in view, false)
  assert.deepEqual(operationalCount(view.components), { ok: 1, total: 3 })
})

test('parsePublicIncidents reads an incident feed that has no status summary', () => {
  const rows = parsePublicIncidents({
    page: { name: 'OpenAI', url: 'https://status.openai.com/' },
    incidents: [
      { name: 'Codex quota reset', status: 'resolved', impact: 'none', updated_at: '2026-11-01T00:00:00Z' },
      { name: ' ', status: 'resolved' },
    ],
  })
  assert.equal(rows?.length, 1)
  assert.equal(rows?.[0]?.name, 'Codex quota reset')
  assert.equal(parsePublicIncidents({ page: {} }), null)
})

test('parsePublicStatus rejects a payload that is not a status summary', () => {
  assert.deepEqual(parsePublicStatus(null, 'https://status.claude.com'), { ok: false })
  assert.deepEqual(parsePublicStatus({ status: { description: 'ok' } }, 'https://status.claude.com'), { ok: false })
  const view = parsePublicStatus({
    page: { name: 'Claude', url: 'http://status.claude.com' },
    status: { indicator: 'surprise', description: 'All Systems Operational' },
    components: [],
  }, 'https://status.claude.com')
  assert.equal(view.ok, true)
  if (!view.ok) return
  assert.equal(view.url, 'https://status.claude.com')
  assert.equal(view.indicator, 'unknown')
  assert.equal(matchesTerms('Codex in ChatGPT Desktop', ['codex']), true)
  assert.equal(matchesTerms('Images', ['codex', 'reset']), false)
})

test('GPT uses verified ChatGPT components, not similarly named API login or Codex', () => {
  const components = [
    { id: '01JMXBNJXGV1T5GT2M9XA83XNG', name: 'Conversations', status: 'operational' as const },
    { id: '01JMXBNJXG1S2D9V65P1ZZTD94', name: 'Login', status: 'operational' as const },
    { id: '01JSM5RTJWHRWDTS6Q604VEW3B', name: 'Login', status: 'major_outage' as const },
    { name: 'Codex in ChatGPT Desktop', status: 'major_outage' as const },
    { name: 'Chat Completions', status: 'major_outage' as const },
    { name: 'ChatGPT Work', status: 'degraded_performance' as const },
    { id: 'changed-id', name: 'Search', status: 'major_outage' as const },
  ]
  const scoped = serviceStatusComponents(components, 'gpt')
  assert.deepEqual(scoped.map(component => component.name), ['Conversations', 'Login', 'ChatGPT Work'])
  assert.equal(publicStatusTone(scoped, true), 'degraded_performance')
  assert.deepEqual(serviceStatusComponents([components[3]!], 'gpt'), [])
})

test('native ChatGPT groups support new children while excluding Codex', () => {
  const view = parsePublicStatus({
    page: { name: 'OpenAI' }, status: { indicator: 'none', description: 'Operational' },
    components: [
      { id: 'chatgpt-group', name: 'ChatGPT', status: 'operational', group: true, components: ['new-child', 'codex-child'] },
      { id: 'new-child', name: 'New Chat Feature', status: 'operational', group_id: 'chatgpt-group' },
      { id: 'codex-child', name: 'Codex in ChatGPT Desktop', status: 'operational', group_id: 'chatgpt-group' },
      { name: 'API Child', status: 'major_outage', group_id: 'api-group' },
    ],
  }, 'https://status.openai.com/')
  assert.equal(view.ok, true)
  if (!view.ok) return
  assert.deepEqual(serviceStatusComponents(view.components, 'gpt').map(component => component.name), ['New Chat Feature'])
})

test('Claude consumer pill covers claude.ai only, not API or Claude Code', () => {
  const scoped = serviceStatusComponents([
    { name: 'claude.ai', status: 'degraded_performance' },
    { name: 'Claude Code', status: 'operational' },
    { name: 'Claude API (api.anthropic.com)', status: 'operational' },
    { name: 'Claude on Cloud Platforms', status: 'operational', isGroup: true },
  ], 'claude')
  assert.deepEqual(scoped, [{ name: 'claude.ai', status: 'degraded_performance' }])
  assert.deepEqual(serviceStatusComponents([{ name: 'Claude API', status: 'operational' }], 'claude'), [])
})

test('unknown, stale, invalid, future and absent retrievals never become healthy', () => {
  const now = Date.parse('2026-11-14T16:00:00Z')
  for (const value of [null, 'invalid', '2026-11-14T15:44:59Z', '2026-11-14T16:10:00Z']) assert.equal(publicStatusFresh(value, now), false)
  assert.equal(publicStatusFresh('2026-11-14T15:52:00Z', now), true)
  assert.equal(publicStatusTone([{ name: 'ChatGPT', status: 'operational' }], false), 'unknown')
  assert.equal(publicStatusTone([], true), 'unknown')
  assert.equal(publicStatusTone([{ name: 'ChatGPT', status: 'unknown' }], true), 'unknown')
  assert.equal(publicStatusTone([{ name: 'ChatGPT', status: 'operational' }], true), 'operational')
})

test('status polling throttles rapid tab switches, backs off on 429 and shares valid cache entries', () => {
  const now = Date.parse('2026-11-20T06:00:00Z')
  assert.equal(shouldFetchStatus('2026-11-20T05:58:00Z', 0, now), false, 'skips fetch within minimum gap')
  assert.equal(shouldFetchStatus('2026-11-20T05:56:00Z', 0, now), true, 'fetches after minimum gap elapses')
  assert.equal(shouldFetchStatus(null, now + 30_000, now), false, 'respects active backoff window')
  assert.equal(parseRetryAfterMs('45', now), 45_000)
  assert.equal(parseRetryAfterMs('2', now), 10_000)
  assert.equal(parseRetryAfterMs('3600', now), 600_000)
  assert.equal(statusRefreshDelayMs({ fetchedAt: '2026-11-20T05:58:00Z', failures: 0, now }), 180_000)
  assert.equal(statusRefreshDelayMs({ fetchedAt: null, failures: 1, retryAfterMs: 180_000, now }), 180_000)
  assert.equal(statusRefreshDelayMs({ fetchedAt: null, failures: 2, now }), 240_000)

  const raw = serializeCachedPublicStatus({
    provider: 'openai',
    fetchedAt: '2026-11-20T05:57:00Z',
    stale: false,
    summary: { page: { name: 'OpenAI' }, status: { indicator: 'none', description: 'Operational' }, components: [{ name: 'ChatGPT', status: 'operational' }] },
    series: [{ key: 'chatgpt', label: 'ChatGPT', component_count: 16, status: 'operational', uptime_percent: 99.5, days: [{ date: '2026-11-20', status: 'operational' }] }],
  })
  const cached = parseCachedPublicStatus(raw, 'openai', 'https://status.openai.com/', now)
  assert.ok(cached)
  assert.equal(cached.series.length, 1)
  assert.equal(parseCachedPublicStatus(raw, 'claude', 'https://status.claude.com/', now), null)
  assert.equal(parseCachedPublicStatus(raw, 'openai', 'https://status.openai.com/', now + 20 * 60_000), null)
})

test('incident summaries preserve latest official update, association and valid times', () => {
  const rows = parsePublicIncidents({
    page: { url: 'https://status.claude.com/' },
    incidents: [{
      id: 'incident-1', name: 'Sign-in issue', status: 'monitoring', impact: 'minor',
      created_at: '2026-11-14T12:00:00Z', started_at: '2026-11-14T11:58:00Z',
      updated_at: '2026-11-14T12:30:00Z', resolved_at: 'invalid', shortlink: 'https://unrelated.example/incident',
      incident_updates: [
        { status: 'investigating', body: 'Earlier update', display_at: '2026-11-14T12:00:00Z' },
        { status: 'monitoring', body: ' Recovery is being monitored. ', display_at: '2026-11-14T12:30:00Z', affected_components: [{ code: 'claude-web', name: 'claude.ai' }] },
      ],
    }],
  })
  assert.equal(rows?.length, 1)
  const incident = rows![0]!
  assert.equal(incident.latestUpdate?.body, 'Recovery is being monitored.')
  assert.equal(incident.latestUpdate?.status, 'monitoring')
  assert.equal(incident.startedAt, '2026-11-14T11:58:00.000Z')
  assert.equal(incident.resolvedAt, null)
  assert.equal(incident.url, null)
  assert.deepEqual(incident.components, [{ id: 'claude-web', name: 'claude.ai' }])
  assert.equal(incidentPhase(incident), 2)
  assert.equal(incidentElapsedMinutes(incident, Date.parse('2026-11-14T12:38:00Z')), 40)
})

test('unlinked provider announcements and API-only incidents cannot become GPT incidents', () => {
  const incidents = parsePublicIncidents({ incidents: [
    { name: 'ChatGPT mentioned in provider-wide note', status: 'investigating', impact: 'major' },
    { name: 'API outage', status: 'investigating', impact: 'major', components: [{ id: 'api-login', name: 'Login' }] },
    { name: 'ChatGPT login issue', status: 'identified', impact: 'minor', components: [{ id: '01JMXBNJXG1S2D9V65P1ZZTD94', name: 'Login' }] },
  ] })!
  const components = [
    { id: '01JMXBNJXG1S2D9V65P1ZZTD94', name: 'Login', status: 'operational' as const },
    { id: 'api-login', name: 'Login', status: 'major_outage' as const },
  ]
  const scoped = serviceStatusIncidents(incidents, components, 'gpt')
  assert.deepEqual(scoped.map(incident => incident.name), ['ChatGPT login issue'])
  assert.equal(serviceStatusTone(serviceStatusComponents(components, 'gpt'), scoped, true), 'degraded_performance')
  assert.equal(serviceStatusTone(serviceStatusComponents(components, 'gpt'), scoped, false), 'unknown')
})

test('active scoped incidents precede resolved reports without inventing a phase or duration', () => {
  const rows = parsePublicIncidents({ incidents: [
    { name: 'Recently resolved', status: 'resolved', impact: 'minor', created_at: '2026-11-14T12:00:00Z', updated_at: '2026-11-14T14:00:00Z', components: [{ name: 'claude.ai' }] },
    { name: 'Still under investigation', status: 'investigating', impact: 'major', created_at: '2026-11-14T10:00:00Z', updated_at: '2026-11-14T11:00:00Z', components: [{ name: 'claude.ai' }] },
    { name: 'Unknown phase', status: 'unexpected', impact: 'none', created_at: 'invalid', updated_at: '2026-11-14T09:00:00Z', components: [{ name: 'claude.ai' }] },
  ] })!
  const scoped = serviceStatusIncidents(rows, [{ name: 'claude.ai', status: 'operational' }], 'claude')
  assert.equal(scoped[0]?.name, 'Still under investigation')
  assert.equal(incidentPhase(rows[0]!), 3)
  assert.equal(incidentElapsedMinutes(rows[0]!, Date.parse('2026-11-14T15:00:00Z')), null)
  assert.equal(incidentPhase(rows[2]!), null)
  assert.equal(incidentElapsedMinutes(rows[2]!, Date.parse('2026-11-14T15:00:00Z')), null)
})

test('partial history empty days stay unknown and do not produce uptime', () => {
  const now = Date.parse('2026-11-14T15:00:00Z')
  const history = parsePublicStatusHistory({ state: 'ready', coverage: 'partial', source_url: 'https://status.claude.com/history', incidents: [] }, 'https://status.claude.com/')
  assert.equal(history.state, 'ready')
  assert.equal(history.coverage, 'partial')
  const days = publicHistoryDays(history.incidents, now)
  assert.equal(days.length, 90)
  assert.equal(days[0]?.date, '2026-08-17')
  assert.equal(days[89]?.date, '2026-11-14')
  assert.equal(days.every(day => day.incidents.length === 0), true)
  assert.equal('uptime' in history, false)
  assert.equal('percentage' in history, false)
  assert.equal('operational' in days[0]!, false)
  assert.equal(parsePublicStatusHistory({ state: 'ready', coverage: 'complete', incidents: [] }, 'https://status.claude.com/').state, 'unavailable')
  assert.equal(parsePublicStatusHistory({ state: 'ready', coverage: 'partial' }, 'https://status.claude.com/').state, 'unavailable')
})

test('history marks only scoped documented intervals and does not fill missing resolution', () => {
  const now = Date.parse('2026-11-14T15:00:00Z')
  const history = parsePublicStatusHistory({ state: 'ready', coverage: 'partial', incidents: [
    { id: 'known-period', name: 'Known incident', status: 'resolved', started_at: '2026-11-12T23:00:00Z', resolved_at: '2026-11-13T01:00:00Z', components: [{ name: 'claude.ai' }] },
    { id: 'no-end', name: 'Unknown resolution time', status: 'resolved', created_at: '2026-11-10T10:00:00Z', components: [{ name: 'claude.ai' }] },
    { id: 'unlinked', name: 'Claude mentioned but not associated', status: 'investigating', created_at: '2026-11-01T10:00:00Z' },
    { id: 'too-old', name: 'Outside range', status: 'resolved', started_at: '2026-01-01T10:00:00Z', resolved_at: '2026-01-01T12:00:00Z', components: [{ name: 'claude.ai' }] },
  ] }, 'https://status.claude.com/')
  const incidents = publicHistoryIncidents(serviceStatusIncidents(history.incidents, [{ name: 'claude.ai', status: 'operational' }], 'claude'), now)
  assert.equal(incidents.length, 2)
  const days = publicHistoryDays(incidents, now)
  assert.deepEqual(days.filter(day => day.incidents.length).map(day => day.date), ['2026-11-10', '2026-11-12', '2026-11-13'])
  assert.equal(days.filter(day => !day.incidents.length).length, 87)
  assert.equal(days.some(day => 'uptime' in day || 'percent' in day), false)
})

test('an unknown incident phase never proves ongoing impact or fills subsequent days', () => {
  const now = Date.parse('2026-11-14T15:00:00Z')
  const rows = parsePublicIncidents({ incidents: [{
    id: 'unknown-old-record', name: 'Unknown incident stage', status: 'unexpected', impact: 'major',
    created_at: '2026-11-01T08:00:00Z', started_at: '2026-11-01T07:00:00Z', components: [{ name: 'claude.ai' }],
  }] })!
  const components = [{ name: 'claude.ai', status: 'operational' as const }]
  const incidents = serviceStatusIncidents(rows, components, 'claude')
  assert.equal(incidentPhase(incidents[0]!), null)
  assert.equal(incidentElapsedMinutes(incidents[0]!, now), null)
  assert.equal(serviceStatusTone(components, incidents, true), 'operational')
  assert.deepEqual(publicHistoryDays(incidents, now).filter(day => day.incidents.length).map(day => day.date), ['2026-11-01'])
})

test('native groups override compatibility identities even when names or IDs are familiar', () => {
  const components = [
    { id: 'gpt-group', name: 'ChatGPT', isGroup: true, childIds: ['real-child'], status: 'unknown' as const },
    { id: 'real-child', name: 'New official ChatGPT feature', groupId: 'gpt-group', status: 'operational' as const },
    { id: '01JMXBNJXG1S2D9V65P1ZZTD94', name: 'Login', groupId: 'other-group', status: 'major_outage' as const },
    { id: 'other-child', name: 'ChatGPT misleading name', groupId: 'other-group', status: 'major_outage' as const },
  ]
  assert.deepEqual(serviceStatusComponents(components, 'gpt').map(component => component.id), ['real-child'])
})

test('full incident titles and all updates retain context beyond the latest generic message', () => {
  const longTitle = 'Official incident '.repeat(20)
  const fullBody = 'Detailed official scope. '.repeat(100) + '\nChatGPT usage is not affected.'
  const rows = parsePublicIncidents({ page: { url: 'https://status.openai.com/' }, incidents: [{
    id: 'incident-with-context', name: longTitle, status: 'monitoring', impact: 'minor',
    incident_updates: [
      { status: 'identified', body: fullBody, display_at: '2026-11-14T12:00:00Z' },
      { status: 'monitoring', body: 'Monitoring recovery.', display_at: '2026-11-14T13:00:00Z' },
    ],
  }] })!
  assert.equal(rows[0]!.name, longTitle.trim())
  assert.equal(rows[0]!.updates.length, 2)
  assert.equal(rows[0]!.updates[1]!.body, fullBody)
  assert.equal(rows[0]!.latestUpdate?.body, 'Monitoring recovery.')
})

test('an optional older history response cannot overwrite newer incident progress', () => {
  const current = parsePublicIncidents({ incidents: [{ id: 'same-incident', name: 'Current progress', status: 'monitoring', impact: 'minor', updated_at: '2026-11-14T13:00:00Z' }] })!
  const history = parsePublicIncidents({ incidents: [{ id: 'same-incident', name: 'Earlier progress', status: 'investigating', impact: 'minor', updated_at: '2026-11-14T12:00:00Z' }] })!
  const combined = mergePublicStatusIncidents(current, history)
  assert.equal(combined.length, 1)
  assert.equal(combined[0]!.status, 'monitoring')
  assert.equal(mergePublicStatusIncidents(current, [...current])[0], current[0])
})

test('an older historical active record missing from the current summary cannot extend impact through today', () => {
  const history = parsePublicIncidents({ incidents: [{ id: 'old-active', name: 'Old progress', status: 'investigating', impact: 'minor', started_at: '2026-11-12T13:00:00Z', updated_at: '2026-11-12T13:00:00Z' }] })!
  const combined = mergePublicStatusIncidents([], history)
  assert.equal(combined[0]!.status, 'unknown')
  const days = publicHistoryDays(combined, Date.parse('2026-11-15T12:00:00Z'))
  assert.deepEqual(days.filter(day => day.incidents.length).map(day => day.date), ['2026-11-12'])
})


test('native incident linkage cannot fall back to a similarly named component from another product', () => {
  const components = [
    { id: 'gpt-group', name: 'ChatGPT', isGroup: true, childIds: ['gpt-login'], status: 'unknown' as const },
    { id: 'gpt-login', name: 'Login', groupId: 'gpt-group', status: 'operational' as const },
  ]
  const rows = parsePublicIncidents({ incidents: [
    { id: 'api-only', name: 'API incident', status: 'investigating', components: [{ id: 'api-id', name: 'ChatGPT Work' }] },
    { id: 'gpt-login-incident', name: 'Login incident', status: 'investigating', components: [{ id: 'gpt-login', name: 'Login' }] },
  ] })!
  assert.deepEqual(serviceStatusIncidents(rows, components, 'gpt').map(row => row.id), ['gpt-login-incident'])
})

test('current status colors distinguish normal, maintenance or degradation, failures, and unknown', () => {
  assert.equal(publicStatusColor('operational'), 'green')
  for (const status of ['under_maintenance', 'degraded_performance'] as const) assert.equal(publicStatusColor(status), 'yellow')
  for (const status of ['partial_outage', 'major_outage'] as const) assert.equal(publicStatusColor(status), 'red')
  assert.equal(publicStatusColor('unknown'), 'gray')
  assert.equal(publicStatusColor(publicStatusTone([{ name: 'ChatGPT', status: 'operational' }], false)), 'gray')
  assert.equal(publicStatusColor(publicIncidentTone({ impact: 'none' })), 'gray')
})

test('daily records keep real incident details and severity while uncovered days remain gray', () => {
  const now = Date.parse('2026-11-16T12:00:00Z')
  const incidents = parsePublicIncidents({ incidents: [
    { id: 'minor-issue', name: 'Degraded service', status: 'resolved', impact: 'minor', started_at: '2026-11-13T10:00:00Z', resolved_at: '2026-11-13T11:00:00Z' },
    { id: 'major-issue', name: 'Service outage', status: 'resolved', impact: 'major', started_at: '2026-11-14T10:00:00Z', resolved_at: '2026-11-14T11:00:00Z' },
    { id: 'maintenance', name: 'Scheduled work', status: 'resolved', impact: 'maintenance', started_at: '2026-11-15T10:00:00Z', resolved_at: '2026-11-15T11:00:00Z' },
  ] })!
  const days = publicHistoryDays(incidents, now)
  assert.equal(publicStatusColor(days.find(day => day.date === '2026-11-13')!.tone), 'yellow')
  const failedDay = days.find(day => day.date === '2026-11-14')!
  assert.equal(publicStatusColor(failedDay.tone), 'red')
  assert.equal(failedDay.records[0], incidents[1])
  assert.equal(failedDay.records[0]!.name, 'Service outage')
  assert.equal(publicStatusColor(days.find(day => day.date === '2026-11-15')!.tone), 'yellow')
  assert.equal(publicStatusColor(days.find(day => day.date === '2026-11-16')!.tone), 'gray')
  assert.equal(days.filter(day => !day.records.length).every(day => day.tone === 'unknown'), true)
})

test('parsePublicStatusSeries validates 90-day uptime series for GPT (2 groups) and Claude (3 groups)', () => {
  const gptSeries = parsePublicStatusSeries([
    {
      key: 'chatgpt',
      label: 'ChatGPT',
      component_count: 16,
      status: 'operational',
      uptime_percent: 99.26,
      days: [
        { date: '2026-11-18', status: 'operational' },
        {
          date: '2026-11-19',
          status: 'degraded_performance',
          events: [
            { id: 'inc-1', name: 'Elevated errors', url: 'https://status.openai.com/incidents/inc-1' },
            { id: 'bad', name: 'Untrusted', url: 'https://evil.example/phish' },
          ],
        },
        { date: 'invalid-date', status: 'operational' },
      ],
    },
    {
      key: 'codex',
      label: 'Codex',
      component_count: 4,
      status: 'operational',
      uptime_percent: 99.94,
      days: [{ date: '2026-11-19', status: 'operational' }],
    },
    { key: 'empty', label: 'Empty', days: [] },
  ], 'https://status.openai.com/')

  assert.equal(gptSeries.length, 2)
  assert.equal(gptSeries[0]!.componentCount, 16)
  assert.equal(gptSeries[0]!.uptimePercent, 99.26)
  assert.equal(gptSeries[0]!.days.length, 2)
  assert.equal(publicStatusColor(gptSeries[0]!.days[0]!.status), 'green')
  assert.equal(publicStatusColor(gptSeries[0]!.days[1]!.status), 'yellow')
  assert.equal(gptSeries[0]!.days[1]!.events[0]!.url, 'https://status.openai.com/incidents/inc-1')
  assert.equal(gptSeries[0]!.days[1]!.events[1]!.url, null)
  assert.equal(seriesOverallTone(gptSeries, true), 'operational')
  assert.equal(seriesOverallTone(gptSeries, false), 'unknown')

  const claudeSeries = parsePublicStatusSeries([
    { key: 'claude', label: 'Claude', component_count: 1, status: 'operational', uptime_percent: 99.56, days: [{ date: '2026-11-19', status: 'operational' }] },
    { key: 'claude_code', label: 'Claude Code', component_count: 1, status: 'partial_outage', uptime_percent: 99.7, days: [{ date: '2026-11-19', status: 'major_outage' }] },
    { key: 'claude_api', label: 'Claude API', component_count: 1, status: 'operational', uptime_percent: 150, days: [{ date: '2026-11-19', status: 'operational' }] },
  ], 'https://status.claude.com/')
  assert.equal(claudeSeries.length, 3)
  assert.equal(claudeSeries[2]!.uptimePercent, null)
  assert.equal(seriesOverallTone(claudeSeries, true), 'partial_outage')
})

test('fallbackSeriesFromSummary preserves official groups without fabricating uptime or green history', () => {
  const now = Date.parse('2026-11-19T12:00:00Z')
  const claudeSummary = parsePublicStatus({
    page: { name: 'Claude' },
    status: { indicator: 'none', description: 'Operational' },
    components: [
      { id: 'rwppv331jlwc', name: 'claude.ai', status: 'operational' },
      { id: 'yyzkbfz2thpt', name: 'Claude Code', status: 'degraded_performance' },
      { id: 'k8w3r06qmzrp', name: 'Claude API (api.anthropic.com)', status: 'operational' },
    ],
  }, 'https://status.claude.com/')
  assert.equal(claudeSummary.ok, true)
  if (!claudeSummary.ok) return
  const fallback = fallbackSeriesFromSummary(claudeSummary, 'claude', now)
  assert.deepEqual(fallback.map(item => item.key), ['claude', 'claude_code', 'claude_api'])
  assert.equal(fallback[1]!.status, 'degraded_performance')
  assert.equal(fallback.every(item => item.uptimePercent === null), true)
  assert.equal(fallback.every(item => item.days.length === 90 && item.days.every(d => d.status === 'unknown')), true)
})
