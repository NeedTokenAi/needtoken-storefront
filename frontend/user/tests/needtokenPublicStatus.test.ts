import test from 'node:test'
import assert from 'node:assert/strict'
import { matchesTerms, operationalCount, parsePublicIncidents, parsePublicStatus, publicStatusFresh, publicStatusTone, serviceStatusComponents } from '../src/templates/needtoken/lib/publicStatus.ts'

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
  for (const value of [null, 'invalid', '2026-11-14T15:57:59Z', '2026-11-14T16:10:00Z']) assert.equal(publicStatusFresh(value, now), false)
  assert.equal(publicStatusFresh('2026-11-14T15:59:30Z', now), true)
  assert.equal(publicStatusTone([{ name: 'ChatGPT', status: 'operational' }], false), 'unknown')
  assert.equal(publicStatusTone([], true), 'unknown')
  assert.equal(publicStatusTone([{ name: 'ChatGPT', status: 'unknown' }], true), 'unknown')
  assert.equal(publicStatusTone([{ name: 'ChatGPT', status: 'operational' }], true), 'operational')
})
