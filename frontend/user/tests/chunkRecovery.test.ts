import assert from 'node:assert/strict'
import test from 'node:test'
import { isChunkLoadError } from '../src/utils/chunkRecovery.ts'

test('isChunkLoadError identifies browser dynamic import failures', () => {
  assert.equal(
    isChunkLoadError(new TypeError('Failed to fetch dynamically imported module: https://needtoken.ai/assets/ApiCostCalculator-CHU0lyHb.js')),
    true,
  )
  assert.equal(isChunkLoadError(new Error('Importing a module script failed.')), true)
  assert.equal(isChunkLoadError(Object.assign(new Error('boom'), { name: 'ChunkLoadError' })), true)
  assert.equal(isChunkLoadError(new Error('Unable to preload CSS for /assets/app.css')), true)
  assert.equal(isChunkLoadError(new Error('Network timeout')), false)
  assert.equal(isChunkLoadError(null), false)
})
