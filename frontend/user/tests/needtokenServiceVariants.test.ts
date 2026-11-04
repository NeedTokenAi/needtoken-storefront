import test from 'node:test'
import assert from 'node:assert/strict'
import { serviceSkuPresentation, serviceVariantLabel } from '../src/templates/needtoken/lib/serviceVariants.ts'
import { serviceAvailability, serviceSkuPrice } from '../src/templates/needtoken/lib/serviceCatalog.ts'

test('legacy import labels prefer the actual option and hide source and reference-price fields', () => {
  const sku = { id: 1, spec_values: { official_usd: '500', period: '1 month', plan: 'Pro 500', source_product_id: 'x'.repeat(500) } }
  assert.equal(serviceVariantLabel(sku, 'zh-CN', '标准方案'), 'Pro 500 · 1 month')
  assert.equal(serviceSkuPresentation(sku, 'zh-CN').officialPriceUsd, '500.00')
  assert.equal(serviceVariantLabel({ id: 2, spec_values: { official_usd: '20', source: 'private', currency: 'USD' }, sku_code: 'SKU_SECRET_987654321' }, 'en-US', 'Option 2'), 'Option 2')
  assert.equal(serviceVariantLabel({ id: 3, spec_values: { period: '年度' } }, 'zh-CN', '标准方案'), '年度')
})

test('native multilingual SKU specifications remain buyer-facing labels', () => {
  const sku = { id: 1, spec_values: { 'zh-CN': '月度', 'zh-TW': '每月', 'en-US': 'Monthly', source: 'internal' } }
  assert.equal(serviceVariantLabel(sku, 'zh-CN', '方案 1'), '月度')
  assert.equal(serviceVariantLabel(sku, 'zh-TW', '方案 1'), '每月')
  assert.equal(serviceVariantLabel(sku, 'en-US', 'Option 1'), 'Monthly')
  assert.notEqual(serviceVariantLabel(sku, 'en-US', 'Option 1'), 'internal')
})

test('each selected SKU supplies localized copy and reference data without changing sale price or availability', () => {
  const monthly = { id: 1, price_amount: '168', is_active: true, stock_status: 'in_stock', display_meta: {
    label: { 'zh-CN': '月度', 'en-US': 'Monthly' }, title: { 'zh-CN': 'ChatGPT Plus 月度' }, description: { 'zh-CN': '按月订阅。' },
    feature_lines: { 'zh-CN': '原号充值\n按月生效\n查看订单进度' }, price_suffix: { 'zh-CN': '/月' }, official_price_usd: '20.00',
  } }
  const annual = { ...monthly, id: 2, price_amount: '1988', stock_status: 'out_of_stock', display_meta: {
    ...monthly.display_meta, label: { 'zh-CN': '年度', 'en-US': 'Annual' }, title: { 'zh-CN': 'ChatGPT Plus 年度' },
    description: { 'zh-CN': '全年订阅。' }, official_price_usd: '200.00',
  } }
  assert.equal(serviceVariantLabel(monthly, 'en-US', 'Default'), 'Monthly')
  assert.deepEqual(serviceSkuPresentation(monthly, 'zh-TW').features, ['原号充值', '按月生效', '查看订单进度'])
  assert.equal(serviceSkuPresentation(annual, 'zh-CN').description, '全年订阅。')
  assert.equal(serviceSkuPresentation(annual, 'zh-CN').title, 'ChatGPT Plus 年度')
  assert.equal(serviceSkuPrice(monthly)?.amount, '168.00')
  assert.equal(serviceSkuPrice(annual)?.amount, '1988.00')
  assert.equal(serviceAvailability({ id: 1, slug: 'plus', category_id: 1 }, annual), 'sold_out')
})

test('missing display metadata falls back cleanly and invalid official references are not displayed', () => {
  for (const reference of ['-20', 'NaN', '20.123', '1e3', '0', {}, null]) {
    assert.equal(serviceSkuPresentation({ id: 1, display_meta: { official_price_usd: reference } }, 'en-US').officialPriceUsd, null)
  }
  assert.deepEqual(serviceSkuPresentation({ id: 1 }, 'en-US'), { label: '', title: '', description: '', features: [], priceSuffix: '', officialPriceUsd: null })
  assert.equal(serviceSkuPresentation({ id: 1, spec_values: { official_usd: '500' }, display_meta: { official_price_usd: '' } }, 'zh-CN').officialPriceUsd, null)
})
