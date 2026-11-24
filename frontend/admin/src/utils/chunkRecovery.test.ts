import { describe, expect, it } from 'vitest'
import { isChunkLoadError } from './chunkRecovery'

describe('admin chunkRecovery', () => {
  it('recognizes dynamic module and chunk load errors', () => {
    expect(
      isChunkLoadError(new TypeError('Failed to fetch dynamically imported module: https://needtoken.ai/admin/assets/Dashboard-abc.js')),
    ).toBe(true)
    expect(isChunkLoadError(new Error('Importing a module script failed.'))).toBe(true)
    expect(isChunkLoadError(Object.assign(new Error('chunk'), { name: 'ChunkLoadError' }))).toBe(true)
    expect(isChunkLoadError(new Error('Validation failed'))).toBe(false)
  })
})
