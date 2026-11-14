import test from 'node:test'
import assert from 'node:assert/strict'
import { checkoutSourceItems, resolveCheckoutSource } from '../src/utils/checkoutSource.ts'

test('existing checkout routes keep cart and native direct-purchase modes unchanged', () => {
  assert.equal(resolveCheckoutSource('buynow'), 'buynow')
  for (const mode of [undefined, null, '', 'cart', 'BUYNOW', ['buynow'], { mode: 'buynow' }]) assert.equal(resolveCheckoutSource(mode), 'cart')
})

test('the explicit embedded mode reads only its single-product draft on product routes', () => {
  const cart = [{ productId: 1, skuId: 10, quantity: 3, priceAmount: '99.00' }]
  const draft = { productId: 2, skuId: 20, quantity: 2, priceAmount: '199.00' }
  const source = resolveCheckoutSource(undefined, 'buynow')
  assert.equal(source, 'buynow')
  assert.deepEqual(checkoutSourceItems(source, cart, draft), [draft])
  assert.equal(checkoutSourceItems(source, cart, draft)[0], draft)
  assert.deepEqual(cart, [{ productId: 1, skuId: 10, quantity: 3, priceAmount: '99.00' }])
})

test('a cleared or missing direct draft cannot silently submit shopping cart items', () => {
  const cart = [{ productId: 8, quantity: 1 }]
  assert.deepEqual(checkoutSourceItems('buynow', cart, null), [])
  assert.equal(checkoutSourceItems('cart', cart, null), cart)
  assert.equal(checkoutSourceItems('cart', cart, { productId: 9, quantity: 5 }), cart)
})

test('updating a single-product draft preserves the native SKU, quantity and other field values', () => {
  const initial = { productId: 2, skuId: 20, quantity: 1, priceAmount: '199.00', manualFormSchema: { fields: ['account'] } }
  const revised = { ...initial, skuId: 21, quantity: 4, priceAmount: '399.00' }
  assert.deepEqual(checkoutSourceItems('buynow', [], revised), [revised])
  assert.equal(initial.skuId, 20)
  assert.equal(initial.quantity, 1)
  assert.equal(checkoutSourceItems('buynow', [], revised)[0]?.manualFormSchema, initial.manualFormSchema)
})
