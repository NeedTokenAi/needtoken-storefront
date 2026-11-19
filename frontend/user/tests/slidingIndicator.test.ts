import assert from 'node:assert/strict'
import test from 'node:test'
import { computeSlidingIndicator } from '../src/templates/needtoken/lib/slidingIndicatorGeometry.ts'

test('returns hidden zero rect when container or active item is missing or collapsed', () => {
  assert.deepEqual(computeSlidingIndicator(null, { left: 10, top: 4, width: 80, height: 32 }), {
    x: 0, y: 0, width: 0, height: 0, visible: false,
  })
  assert.deepEqual(computeSlidingIndicator({ left: 0, top: 0, width: 300, height: 40 }, null), {
    x: 0, y: 0, width: 0, height: 0, visible: false,
  })
  assert.deepEqual(computeSlidingIndicator(
    { left: 0, top: 0, width: 300, height: 40 },
    { left: 10, top: 4, width: 0, height: 32 },
  ), {
    x: 0, y: 0, width: 0, height: 0, visible: false,
  })
})

test('computes relative coordinates and rounds to subpixel precision', () => {
  const rect = computeSlidingIndicator(
    { left: 100.25, top: 20.5, width: 420, height: 44 },
    { left: 184.756, top: 24.5, width: 96.334, height: 36 },
  )
  assert.deepEqual(rect, {
    x: 84.51,
    y: 4,
    width: 96.33,
    height: 36,
    visible: true,
  })
})

test('accounts for horizontal and vertical container scroll offsets', () => {
  const rect = computeSlidingIndicator(
    { left: 50, top: 10, width: 200, height: 40 },
    { left: 90, top: 14, width: 70, height: 32 },
    64,
    8,
  )
  assert.deepEqual(rect, {
    x: 104,
    y: 12,
    width: 70,
    height: 32,
    visible: true,
  })
})
