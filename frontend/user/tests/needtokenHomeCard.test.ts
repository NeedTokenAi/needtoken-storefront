import test from 'node:test'
import assert from 'node:assert/strict'
import {
  homeCardBrand, homeCardPlan, homeCardPriceSuffix, homeCardSku, homeCardTitle, sortHomeProductsByPrice,
} from '../src/templates/needtoken/lib/homeCardPresentation.ts'
import type { ServiceProduct, ServiceSku } from '../src/templates/needtoken/lib/serviceCatalog.ts'

const option = (id: number, price: unknown, extra: Partial<ServiceSku> = {}): ServiceSku => ({
  id, is_active: true, stock_status: 'in_stock', price_amount: price, ...extra,
})
const product = (id: number, skus: ServiceSku[], extra: Partial<ServiceProduct> = {}): ServiceProduct => ({
  id, slug: `product-${id}`, category_id: 2, skus, ...extra,
})

test('homepage order uses the exact displayed SKU final amount in cents, with stable ties and unknown last', () => {
  const rows = [
    product(1, [option(11, '200', { promotion_price_amount: '150', member_price_amount: '120' })]),
    product(2, [option(21, '100')]),
    product(3, [option(31, '100')], { is_sold_out: true }),
    product(4, [option(41, 'bad')]),
    product(5, []),
    product(6, [option(61, '0.00')]),
    product(7, [option(71, '100.01')]),
    product(8, [option(81, '90', { is_active: false }), option(82, '160')]),
  ]
  assert.deepEqual(sortHomeProductsByPrice(rows).map(row => row.id), [6, 2, 3, 7, 1, 8, 4, 5])
  assert.deepEqual(rows.map(row => row.id), [1, 2, 3, 4, 5, 6, 7, 8], 'does not reorder API data in place')
  assert.equal(homeCardSku(rows[7]!)?.id, 82)
})

test('sorting does not substitute a cheaper unavailable SKU for the linked purchase option', () => {
  const first = product(1, [option(10, '1', { stock_status: 'out_of_stock' }), option(11, '300'), option(12, '20')])
  const second = product(2, [option(20, '200')])
  assert.equal(homeCardSku(first)?.id, 11, 'preserves first available native SKU')
  assert.deepEqual(sortHomeProductsByPrice([first, second]).map(row => row.id), [2, 1])
  assert.equal(homeCardSku({ ...first, is_sold_out: true })?.id, 10, 'sold-out card still presents a real active SKU')
})

test('brand comes from explicit SKU copy, exact category slug or actual product title', () => {
  const base = product(1, [], { category: { id: 2, slug: 'chatgpt-recharge' }, title: { 'zh-CN': 'NeedToken Pro 5x' } })
  assert.equal(homeCardBrand(base, null, 'zh-CN'), 'gpt')
  assert.equal(homeCardBrand(base, option(1, '20', { display_meta: { title: { 'en-US': 'Claude Pro' } } }), 'en-US'), 'claude')
  assert.equal(homeCardBrand(base, option(1, '20', { display_meta: { label: { 'en-US': 'Claude monthly' } } }), 'en-US'), 'claude')
  assert.equal(homeCardBrand({ ...base, category: { id: 2, slug: 'claude-accounts' } }, null, 'zh-TW'), 'claude')
  assert.equal(homeCardBrand({ ...base, category: undefined, title: { 'en-US': 'OpenAI GPT-4 subscription' } }, null, 'en-US'), 'gpt')
  assert.equal(homeCardBrand({ ...base, category: undefined, title: { 'zh-CN': 'Claude 原号充值' } }, null, 'zh-CN'), 'claude')
})

test('unknown, ambiguous and substring lookalike brands stay neutral regardless of price or unsupported fields', () => {
  for (const title of ['NeedToken Pro 50x', 'MyChatGPTish', 'Claudeish', 'ChatGPT + Claude bundle', 'Pro 500']) {
    assert.equal(homeCardBrand(product(1, [], { title }), option(1, '500', { display_meta: { brand: 'gpt', provider: 'claude' } }), 'en-US'), 'unknown')
  }
  assert.equal(homeCardBrand(product(1, [], { category: { id: 2, slug: 'not-chatgpt-recharge' } }), null, 'en-US'), 'unknown')
  assert.equal(homeCardBrand(product(1, [], { category: { id: 2, slug: 'chatgpt-recharge' } }), option(1, '100', { display_meta: { title: 'ChatGPT Claude' } }), 'en-US'), 'unknown')
})

test('plan typography preserves all title text and recognizes explicit multipliers without price guesses', () => {
  for (const [title, tier, plan] of [
    ['ChatGPT Pro 5x', '5x', '5x'], ['ChatGPT Pro20x 月度', '20x', '20x'],
    ['Claude Pro 50× annual', '50×', '50x'], ['NeedToken Pro 5 X', '5 X', '5x'],
  ] as const) {
    const parts = homeCardTitle(title)
    assert.equal(parts.before + parts.tier + parts.after, title)
    assert.equal(parts.tier, tier)
    assert.equal(parts.plan, plan)
  }
  for (const title of ['Pro 500', 'Pro 50xlarge', 'Pro 205x', 'MyPlusish']) assert.equal(homeCardTitle(title).plan, 'neutral')
  assert.equal(homeCardTitle('ChatGPT Plus').plan, 'plus')
  assert.equal(homeCardPlan('ChatGPT Plus', option(1, '500', { display_meta: { label: { 'en-US': 'Pro 20x' } } }), 'en-US'), '20x')
  assert.equal(homeCardPlan('Product', option(1, '500', { display_meta: { tier: '50x', illustration: 'book' } }), 'en-US'), 'neutral')
})

test('price suffix accepts editor slash prefixes without duplicating the separator', () => {
  for (const [raw, clean] of [['/月', '月'], [' / / 月 ', '月'], ['／ 年', '年'], ['month', 'month'], [' / ', ''], ['', '']]) {
    assert.equal(homeCardPriceSuffix(raw!), clean)
  }
})
