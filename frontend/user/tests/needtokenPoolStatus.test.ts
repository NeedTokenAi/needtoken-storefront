import test from 'node:test'
import assert from 'node:assert/strict'
import { historyFor, levelFor, parsePoolStatus, summarizePoolStatus, uptimeFor } from '../src/templates/needtoken/lib/poolStatus.ts'

test('parsePoolStatus keeps known levels and coerces the rest to unknown', () => {
    const view = parsePoolStatus({
        enabled: true,
        stale: true,
        updated_at: '2026-11-02T06:00:00Z',
        groups: { pool: { status: 'available' }, aws: { status: 'busy' }, odd: { status: 'great' }, none: {} },
    })
    assert.equal(view.enabled, true)
    assert.equal(view.stale, true)
    assert.equal(view.updatedAt?.toISOString(), '2026-11-02T06:00:00.000Z')
    assert.deepEqual(view.levels, { pool: 'available', aws: 'busy', odd: 'unknown', none: 'unknown' })
    assert.deepEqual(summarizePoolStatus(view), {
        enabled: true,
        totalGroups: 2,
        healthyGroups: 2,
        overall: 'available',
    })
})

test('parsePoolStatus handles unconfigured array fallback safely and levelFor shows nothing without trustworthy data', () => {
    const emptyFallback = parsePoolStatus({ groups: [] })
    assert.equal(emptyFallback.enabled, false)
    assert.deepEqual(emptyFallback.levels, {})
    assert.equal(summarizePoolStatus(emptyFallback).enabled, false)

    const view = parsePoolStatus({ enabled: true, groups: { pool: { status: 'unavailable' }, x: { status: 'unknown' } } })
    assert.equal(levelFor(view, 'pool'), 'unavailable')
    assert.equal(levelFor(view, 'x'), null)
    assert.equal(levelFor(view, 'missing'), null)
    assert.equal(levelFor(null, 'pool'), null)
    assert.equal(levelFor(parsePoolStatus({ enabled: false, groups: { pool: { status: 'available' } } }), 'pool'), null)
    assert.equal(parsePoolStatus(undefined).updatedAt, null)
    assert.equal(parsePoolStatus({ updated_at: 'not a date' }).updatedAt, null)
})

test('parsePoolStatus parses uptime_pct and history slots with graceful fallback', () => {
    const view = parsePoolStatus({
        enabled: true,
        groups: {
            pool: { status: 'available', uptime_pct: 99.667, history: ['available', 'busy', 'weird'] },
            solo: { status: 'busy' },
            outOfRange: { status: 'available', uptime_pct: 120 },
        },
    })
    assert.equal(uptimeFor(view, 'pool'), 99.67)
    assert.deepEqual(historyFor(view, 'pool'), ['available', 'busy', 'unknown'])
    assert.equal(uptimeFor(view, 'solo'), null)
    assert.deepEqual(historyFor(view, 'solo'), ['busy'])
    assert.equal(uptimeFor(view, 'outOfRange'), null)
    assert.equal(uptimeFor(null, 'pool'), null)
    assert.deepEqual(historyFor(null, 'pool'), [])
})
