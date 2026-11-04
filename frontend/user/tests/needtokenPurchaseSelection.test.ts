import test from 'node:test'
import assert from 'node:assert/strict'
import { isKnownPurchasePrice, resolveRequestedSku } from '../src/templates/needtoken/lib/purchaseSelection.ts'

const skus = [{ id: 21, is_active: true }, { id: 22, is_active: false }, { id: 23, is_active: true }]

test('a catalog link selects only an active SKU belonging to the loaded product', () => {
  assert.deepEqual(resolveRequestedSku(undefined, skus), { kind: 'absent' })
  assert.deepEqual(resolveRequestedSku('21', skus), { kind: 'valid', id: 21 })
  assert.deepEqual(resolveRequestedSku('23', skus), { kind: 'valid', id: 23 })
  assert.deepEqual(resolveRequestedSku('22', skus), { kind: 'invalid' })
  assert.deepEqual(resolveRequestedSku('24', skus), { kind: 'invalid' })
})

test('stale, ambiguous or lossy query values require a new manual choice', () => {
  for (const raw of [null, '', '0', '-1', '21.9', '2.1e1', ' 21', '021', ['21', '23'], '9007199254740993']) {
    assert.deepEqual(resolveRequestedSku(raw, skus), { kind: 'invalid' }, `invalid query: ${String(raw)}`)
  }
  assert.deepEqual(resolveRequestedSku('21', []), { kind: 'invalid' })
})

test('unknown prices cannot be offered as free while explicit zero remains valid', () => {
  for (const raw of [0, '0', '0.00', '179.00', 179.5]) assert.equal(isKnownPurchasePrice(raw), true)
  for (const raw of [undefined, null, '', ' ', false, NaN, Infinity, -1, '-0.001', '0.001', '1e2', '9007199254740993']) {
    assert.equal(isKnownPurchasePrice(raw), false, `unknown price: ${String(raw)}`)
  }
})
