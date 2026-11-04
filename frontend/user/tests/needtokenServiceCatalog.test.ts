import test from 'node:test'
import assert from 'node:assert/strict'
import {
  activeServiceSkus, defaultServiceChild, filterServiceProducts, resolveServiceScope, serviceAvailability,
  serviceCategoryIds, serviceDescriptionText, serviceSkuPrice,
} from '../src/templates/needtoken/lib/serviceCatalog.ts'

const categories = [
  { id: 1, slug: 'recharge', parent_id: 0 },
  { id: 2, slug: 'chatgpt-recharge', parent_id: 1 },
  { id: 3, slug: 'claude-recharge', parent_id: 1 },
  { id: 4, slug: 'rental', parent_id: 0 },
  { id: 5, slug: 'chatgpt-account', parent_id: 4 },
]

test('ALL remains scoped to the current service and root-category products', () => {
  const scope = resolveServiceScope(categories, 'recharge')!
  assert.deepEqual(serviceCategoryIds(scope), [1, 2, 3])
  assert.deepEqual(filterServiceProducts([
    { id: 1, slug: 'direct', category_id: 1 },
    { id: 2, slug: 'child', category_id: 2 },
    { id: 3, slug: 'account', category_id: 5 },
    { id: 4, slug: 'unclassified', category_id: 0 },
  ], scope).map(row => row.slug), ['direct', 'child'])
})

test('root visits prefer the ChatGPT child and keep other deep links unchanged', () => {
  const rows = [
    { id: 1, slug: 'recharge', parent_id: 0 },
    { id: 2, slug: 'chatgpt-recharge', parent_id: 1 },
    { id: 3, slug: 'claude-recharge', parent_id: 1 },
    { id: 4, slug: 'rental', parent_id: 0 },
    { id: 5, slug: 'chatgpt-accounts', parent_id: 4 },
    { id: 6, slug: 'claude-accounts', parent_id: 4 },
  ]
  assert.equal(defaultServiceChild(resolveServiceScope(rows, 'recharge'))?.slug, 'chatgpt-recharge')
  assert.equal(defaultServiceChild(resolveServiceScope(rows, 'rental'))?.slug, 'chatgpt-accounts')
  assert.equal(defaultServiceChild(resolveServiceScope(rows, 'claude-recharge')), null)
  assert.equal(defaultServiceChild(resolveServiceScope(rows, 'chatgpt-recharge')), null)
  assert.equal(defaultServiceChild(resolveServiceScope(rows, 'claude-accounts')), null)
  assert.equal(defaultServiceChild(resolveServiceScope(rows, 'recharge'), 'recharge'), null)
  assert.equal(defaultServiceChild(resolveServiceScope(rows, 'rental'), 'recharge')?.slug, 'chatgpt-accounts')
})

test('root without the ChatGPT child stays on ALL', () => {
  const rows = [
    { id: 1, slug: 'recharge', parent_id: 0 },
    { id: 3, slug: 'claude-recharge', parent_id: 1 },
    { id: 4, slug: 'rental', parent_id: 0 },
    { id: 5, slug: 'chatgpt-account', parent_id: 4 },
  ]
  assert.equal(defaultServiceChild(resolveServiceScope(rows, 'recharge')), null)
  assert.equal(defaultServiceChild(resolveServiceScope(rows, 'rental')), null)
})

test('direct child category retains its service identity and sibling filters', () => {
  const scope = resolveServiceScope(categories, 'chatgpt-account')!
  assert.equal(scope.kind, 'rental')
  assert.equal(scope.root?.id, 4)
  assert.deepEqual(serviceCategoryIds(scope), [5])
  assert.deepEqual(scope.categories.map(row => row.id), [5])
})

test('missing configured service does not silently become unscoped ALL', () => {
  const scope = resolveServiceScope([], 'recharge')!
  assert.equal(scope.kind, 'recharge')
  assert.equal(scope.root, null)
  assert.deepEqual(serviceCategoryIds(scope), [])
  assert.deepEqual(filterServiceProducts([{ id: 2, slug: 'other', category_id: 2 }], scope), [])
  assert.equal(resolveServiceScope(categories, 'ordinary-category'), null)
})

test('cycles in category data are bounded and do not create a service', () => {
  assert.equal(resolveServiceScope([{ id: 1, slug: 'a', parent_id: 2 }, { id: 2, slug: 'b', parent_id: 1 }], 'a'), null)
})

test('inactive, sold-out and unknown-stock options cannot create a purchase CTA', () => {
  const available = { id: 2, is_active: true, price_amount: '79.00', stock_status: 'in_stock' }
  const product = { id: 1, slug: 'chatgpt', category_id: 2, skus: [available, { ...available, id: 3, is_active: false }] }
  assert.deepEqual(activeServiceSkus(product).map(row => row.id), [2])
  assert.equal(serviceAvailability(product, available), 'available')
  assert.equal(serviceAvailability(product, { ...available, stock_status: 'out_of_stock' }), 'sold_out')
  assert.equal(serviceAvailability(product, { ...available, stock_status: undefined }), 'unknown')
  assert.equal(serviceAvailability(product, { ...available, price_amount: 'broken' }), 'unknown')
  assert.equal(serviceAvailability(product, { ...available, is_active: false }), 'unknown')
  assert.equal(serviceAvailability({ ...product, is_sold_out: true }, available), 'sold_out')
})

test('selected-SKU price uses actual valid promotion/member amounts without inventing a discount', () => {
  assert.deepEqual(serviceSkuPrice({ id: 1, price_amount: '200', promotion_price_amount: '179' }), { amount: '179.00', originalAmount: '200.00', kind: 'promotion' })
  assert.deepEqual(serviceSkuPrice({ id: 1, price_amount: '200', promotion_price_amount: '179', member_price_amount: '170' }), { amount: '170.00', originalAmount: '200.00', kind: 'member' })
  assert.deepEqual(serviceSkuPrice({ id: 1, price_amount: '200', promotion_price_amount: '300' }), { amount: '200.00', originalAmount: null, kind: 'standard' })
  assert.equal(serviceSkuPrice({ id: 1, price_amount: undefined }), null)
  assert.equal(serviceSkuPrice({ id: 1, price_amount: '-1' }), null)
  assert.equal(serviceSkuPrice({ id: 1, price_amount: '0.001' }), null)
  assert.deepEqual(serviceSkuPrice({ id: 1, price_amount: '0.00' }), { amount: '0.00', originalAmount: null, kind: 'standard' })
  assert.deepEqual(serviceSkuPrice({ id: 1, price_amount: '179.00', promotion_price_amount: '0.001' }), { amount: '179.00', originalAmount: null, kind: 'standard' })
})

test('product descriptions are rendered as plain text rather than backend HTML', () => {
  assert.equal(serviceDescriptionText('<p>月度&nbsp;订阅</p><p>按需 &amp; 续期</p>'), '月度 订阅 按需 & 续期')
})
