import test from 'node:test'
import assert from 'node:assert/strict'
import { matchesTerms, operationalCount, parsePublicIncidents, parsePublicStatus } from '../src/templates/needtoken/lib/publicStatus.ts'

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
