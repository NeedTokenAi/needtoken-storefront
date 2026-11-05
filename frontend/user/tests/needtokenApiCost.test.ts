import test from 'node:test'
import assert from 'node:assert/strict'
import { buildApiCostRates, compareApiCost, convertApiCost, estimateApiCost, parseApiRate, parseApiUsage, type ApiCostRate } from '../src/templates/needtoken/lib/apiCost.ts'
import type { PricingResponse } from '../src/templates/needtoken/lib/pricing.ts'

const options = { defaultModels: [], blockedModels: ['blocked'], supplyLineByGroup: {} }
const model = { model_name: 'text-model', quota_type: 0, model_ratio: 1.25, model_price: 0, completion_ratio: 4, cache_ratio: 0.1, create_cache_ratio: 1.25, enable_groups: ['public'], supported_endpoint_types: ['openai'] }
const response: PricingResponse = { data: [model], group_ratio: { public: 1.5 }, pricing_version: 'native-marker' }
const usage = { calls: '1000', input: '2000', output: '500', cacheRead: '500', cacheWrite: '100' }
const rate = buildApiCostRates(response, options)[0]!

test('native ratio and public group multiplier are applied once, with explicit cache rates', () => {
  assert.equal(rate.inputUsd, 3.75)
  assert.equal(rate.outputUsd, 15)
  assert.equal(rate.cacheReadUsd, 0.375)
  assert.equal(rate.cacheWriteUsd, 4.6875)
  assert.equal(rate.groupRatio, 1.5)
  assert.equal(rate.sourceVersion, 'native-marker')
  assert.deepEqual(estimateApiCost(rate, usage), { monthlyUsd: 13.40625, perCallUsd: 0.01340625, monthlyInput: 2000000, monthlyOutput: 500000 })
})

test('missing/invalid group or unconfigured native prices never become a public quote', () => {
  for (const group of [{}, { public: -1 }, { public: Number.NaN }]) assert.deepEqual(buildApiCostRates({ ...response, group_ratio: group }, options), [])
  for (const patch of [{ model_ratio: 37.5 }, { model_ratio: 0 }, { completion_ratio: -1 }, { model_name: 'blocked' }]) {
    assert.deepEqual(buildApiCostRates({ ...response, data: [{ ...model, ...patch }] }, options), [])
  }
  const free = buildApiCostRates({ ...response, group_ratio: { public: 0 } }, options)[0]!
  assert.equal(estimateApiCost(free, usage)?.monthlyUsd, 0)
})

test('malformed public rows cannot become calculator prices or break the page', () => {
  for (const data of [[null], [{ ...model, model_name: {} }], [{ ...model, enable_groups: 'all' }], [{ ...model, supported_endpoint_types: 'openai' }]]) {
    assert.deepEqual(buildApiCostRates({ ...response, data } as unknown as PricingResponse, options), [])
  }
  assert.deepEqual(buildApiCostRates({ ...response, group_ratio: [1] } as unknown as PricingResponse, options), [])
})

test('expression, plugin, image/audio, tasks and unfamiliar endpoints fail closed', () => {
  for (const patch of [{ billing_expr: 'dynamic()' }, { billing_mode: 'tiered_expr' }, { billing_plugin_variants: [{ plugin_key: 'vendor' }] }, { image_ratio: 1 }, { audio_ratio: 1 }, { billing_usage_schema: { seconds: { unit: 'seconds' } } }, { supported_endpoint_types: ['future-tool'] }, { supported_endpoint_types: ['openai-video'] }]) {
    assert.deepEqual(buildApiCostRates({ ...response, data: [{ ...model, ...patch }] }, options), [], JSON.stringify(patch))
  }
})

test('cached tokens are subsets of input and cannot receive an invented missing rate', () => {
  assert.equal(estimateApiCost(rate, { ...usage, cacheRead: '1950', cacheWrite: '100' }), null)
  assert.equal(estimateApiCost({ ...rate, cacheReadUsd: null }, usage), null)
  assert.equal(estimateApiCost({ ...rate, cacheWriteUsd: null }, usage), null)
  assert.notEqual(estimateApiCost({ ...rate, cacheReadUsd: null, cacheWriteUsd: null }, { ...usage, cacheRead: 0, cacheWrite: 0 }), null)
})

test('per-call prices ignore token guesses and retain the native fixed unit', () => {
  const fixed = buildApiCostRates({ ...response, data: [{ ...model, quota_type: 1, model_price: 0.04 }] }, options)[0]!
  assert.equal(fixed.unit, 'per_call')
  assert.equal(estimateApiCost(fixed, { ...usage, input: '', output: '', cacheRead: '', cacheWrite: '' })?.monthlyUsd, 60)
})

test('manual rates and usage have no unknown zero fallback or unsafe totals', () => {
  for (const value of ['', null, undefined, 'NaN', 'Infinity', '-1', '1e3']) assert.equal(parseApiRate(value), null)
  assert.equal(parseApiRate('0'), 0)
  assert.equal(parseApiRate('0.000000001'), 0.000000001)
  for (const value of ['', null, '-1', '2.5', '1000000001']) assert.equal(parseApiUsage(value), null)
  assert.equal(estimateApiCost(null, usage), null)
  assert.equal(estimateApiCost(rate, { ...usage, calls: '1000000000', input: '1000000000' }), null)
  assert.equal(estimateApiCost({ ...rate, inputUsd: Number.POSITIVE_INFINITY } as ApiCostRate, usage), null)
})

test('FX is explicit and conversion happens before cent rounding', () => {
  assert.ok(Math.abs(convertApiCost(0.004, 'CNY', '7.2')! - 0.0288) < 1e-12)
  assert.equal(compareApiCost(convertApiCost(0.004, 'CNY', '7.2'), '1')?.difference, 0.97)
  assert.equal(convertApiCost(1, 'CNY', ''), null)
  assert.equal(convertApiCost(1, 'CNY', '0'), null)
  assert.equal(convertApiCost(1, 'USD', ''), 1)
})

test('comparison accepts only buyer-provided amounts and can favor either service', () => {
  assert.equal(compareApiCost(20, ''), null)
  assert.equal(compareApiCost(null, '100'), null)
  assert.deepEqual(compareApiCost(20, '50'), { amount: 50, difference: 30, kind: 'lower' })
  assert.deepEqual(compareApiCost(60, '50'), { amount: 50, difference: 10, kind: 'higher' })
  assert.deepEqual(compareApiCost(50, '50'), { amount: 50, difference: 0, kind: 'equal' })
})
