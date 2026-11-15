import test from 'node:test'
import assert from 'node:assert/strict'
import { createCatalogRequestCache } from '../src/templates/needtoken/lib/catalogRequestCache.ts'

test('catalogue memory deduplicates requests and expires prices after a short TTL', async () => {
  let now = 100
  let calls = 0
  const cache = createCatalogRequestCache(30, () => now)
  const load = async () => ({ amount: String(++calls) })
  const [a, b] = await Promise.all([cache.read('guest:tenant1:en', 'products:1', load), cache.read('guest:tenant1:en', 'products:1', load)])
  assert.equal(calls, 1)
  assert.deepEqual(a, b)
  assert.deepEqual(cache.peek('guest:tenant1:en', 'products:1'), a)
  now += 31
  assert.equal(cache.peek('guest:tenant1:en', 'products:1'), undefined)
  assert.deepEqual(await cache.read('guest:tenant1:en', 'products:1', load), { amount: '2' })
  assert.deepEqual(await cache.read('guest:tenant1:en', 'products:1', load, true), { amount: '3' })
})

test('identity, locale and tenant scope changes discard old data and pending results cannot repopulate it', async () => {
  const cache = createCatalogRequestCache()
  let finish!: (value: string) => void
  const old = cache.read('guest:tenant1:en', 'products:1', () => new Promise<string>(resolve => { finish = resolve }))
  assert.equal(cache.peek('member:tenant2:zh', 'products:1'), undefined)
  await cache.read('member:tenant2:zh', 'products:1', async () => 'new member price')
  finish('old guest price')
  await old
  assert.equal(cache.peek('member:tenant2:zh', 'products:1'), 'new member price')
  assert.equal(cache.peek('guest:tenant1:en', 'products:1'), undefined)
})

test('failed catalogue requests are retryable rather than cached', async () => {
  const cache = createCatalogRequestCache()
  await assert.rejects(cache.read('guest', 'products:1', async () => { throw new Error('offline') }))
  assert.equal(await cache.read('guest', 'products:1', async () => 'recovered'), 'recovered')
})
