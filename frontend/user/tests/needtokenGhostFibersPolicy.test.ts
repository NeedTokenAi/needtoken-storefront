import assert from 'node:assert/strict'
import test from 'node:test'
import { ghostFibersPolicy } from '../src/templates/needtoken/lib/ghostFibersPolicy.ts'

test('reduced motion, data saving and constrained devices use the static artwork', () => {
  for (const hints of [{ reducedMotion: true }, { saveData: true }, { hardwareConcurrency: 4 }, { deviceMemory: 4 }]) {
    assert.equal(ghostFibersPolicy(hints).animate, false)
  }
})

test('render resolution and frame rate remain bounded for full and subtle backgrounds', () => {
  assert.deepEqual(ghostFibersPolicy({ hardwareConcurrency: 16, deviceMemory: 16 }), { animate: true, maxDimension: 1600, maxFps: 30, dpr: 1.25 })
  assert.deepEqual(ghostFibersPolicy({ coarsePointer: true }), { animate: true, maxDimension: 1200, maxFps: 24, dpr: 1 })
  assert.deepEqual(ghostFibersPolicy({ subtle: true }), { animate: true, maxDimension: 1100, maxFps: 18, dpr: 1 })
})
