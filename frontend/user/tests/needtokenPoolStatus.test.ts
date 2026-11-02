import test from 'node:test'
import assert from 'node:assert/strict'
import { levelFor, parsePoolStatus } from '../src/templates/needtoken/lib/poolStatus.ts'

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
})

test('levelFor shows nothing without trustworthy data', () => {
    const view = parsePoolStatus({ enabled: true, groups: { pool: { status: 'unavailable' }, x: { status: 'unknown' } } })
    assert.equal(levelFor(view, 'pool'), 'unavailable')
    assert.equal(levelFor(view, 'x'), null)
    assert.equal(levelFor(view, 'missing'), null)
    assert.equal(levelFor(null, 'pool'), null)
    assert.equal(levelFor(parsePoolStatus({ enabled: false, groups: { pool: { status: 'available' } } }), 'pool'), null)
    assert.equal(parsePoolStatus(undefined).updatedAt, null)
    assert.equal(parsePoolStatus({ updated_at: 'not a date' }).updatedAt, null)
})
