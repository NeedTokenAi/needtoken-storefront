import test from 'node:test'
import assert from 'node:assert/strict'
import {
    buildOffers,
    filterOffers,
    formatUsd,
    sortOffers,
    toggleCompare,
    type PricingResponse,
} from '../src/templates/needtoken/lib/pricing.ts'

const options = {
    defaultModels: ['gpt-6.1-sol', 'gpt-6-astra'],
    blockedModels: ['gpt-5.2'],
    supplyLineByGroup: { pool: 'pool' },
}

const response: PricingResponse = {
    data: [
        { model_name: 'gpt-6-astra', quota_type: 0, model_ratio: 1.25, model_price: 0, completion_ratio: 8, vendor_id: 1, enable_groups: ['pool', 'aws'] },
        { model_name: 'gpt-6.1-sol', quota_type: 0, model_ratio: 0.5, model_price: 0, completion_ratio: 4, vendor_id: 1, enable_groups: ['pool'] },
        // 未配价：new-api 兜底倍率 37.5
        { model_name: 'gpt-6-luna', quota_type: 0, model_ratio: 37.5, model_price: 0, completion_ratio: 1, enable_groups: ['pool'] },
        // 黑名单模型
        { model_name: 'gpt-5.2', quota_type: 0, model_ratio: 1, model_price: 0, completion_ratio: 1, enable_groups: ['pool'] },
        // 按次计费
        { model_name: 'image-x', quota_type: 1, model_ratio: 0, model_price: 0.04, completion_ratio: 0, owner_by: 'acme', enable_groups: ['aws'] },
    ],
    vendors: [{ id: 1, name: 'OpenAI' }],
    group_ratio: { pool: 1, aws: 1.5 },
}

test('buildOffers splits each model by group and uses new-api price formula', () => {
    const offers = buildOffers(response, options)
    const astraAws = offers.find(o => o.key === 'gpt-6-astra@aws')
    assert.ok(astraAws)
    assert.equal(astraAws.inputPrice, 1.25 * 2 * 1.5)
    assert.equal(astraAws.outputPrice, 1.25 * 2 * 1.5 * 8)
    assert.equal(astraAws.vendor, 'OpenAI')
    assert.equal(astraAws.supplyLine, null)
    const astraPool = offers.find(o => o.key === 'gpt-6-astra@pool')
    assert.equal(astraPool?.supplyLine, 'pool')
    assert.equal(astraPool?.inputPrice, 2.5)
})

test('buildOffers drops unpriced and blocked models', () => {
    const ids = buildOffers(response, options).map(o => o.modelId)
    assert.equal(ids.includes('gpt-6-luna'), false)
    assert.equal(ids.includes('gpt-5.2'), false)
})

test('buildOffers handles per-call pricing and empty responses', () => {
    const image = buildOffers(response, options).find(o => o.modelId === 'image-x')
    assert.equal(image?.unit, 'per_call')
    assert.equal(image?.inputPrice, 0.04 * 1.5)
    assert.equal(image?.outputPrice, null)
    assert.equal(image?.vendor, 'acme')
    assert.deepEqual(buildOffers({}, options), [])
})

test('sortOffers puts default models first and sorts token prices before per-call', () => {
    const offers = buildOffers(response, options)
    assert.deepEqual(sortOffers(offers, 'default', options.defaultModels).map(o => o.key), [
        'gpt-6.1-sol@pool', 'gpt-6-astra@aws', 'gpt-6-astra@pool', 'image-x@aws',
    ])
    assert.deepEqual(sortOffers(offers, 'price_asc', options.defaultModels).map(o => o.key), [
        'gpt-6.1-sol@pool', 'gpt-6-astra@pool', 'gpt-6-astra@aws', 'image-x@aws',
    ])
    assert.deepEqual(sortOffers(offers, 'price_desc', options.defaultModels).map(o => o.key), [
        'gpt-6-astra@aws', 'gpt-6-astra@pool', 'gpt-6.1-sol@pool', 'image-x@aws',
    ])
})

test('filterOffers matches vendor, supply line and keyword', () => {
    const offers = buildOffers(response, options)
    assert.deepEqual(filterOffers(offers, { supply: 'pool' }).map(o => o.key), ['gpt-6-astra@pool', 'gpt-6.1-sol@pool'])
    assert.deepEqual(filterOffers(offers, { supply: 'group:aws', vendor: 'OpenAI' }).map(o => o.key), ['gpt-6-astra@aws'])
    assert.deepEqual(filterOffers(offers, { keyword: 'SOL' }).map(o => o.key), ['gpt-6.1-sol@pool'])
})

test('toggleCompare keeps at most three selections', () => {
    let selected: string[] = []
    for (const key of ['a', 'b', 'c', 'd']) selected = toggleCompare(selected, key)
    assert.deepEqual(selected, ['a', 'b', 'c'])
    assert.deepEqual(toggleCompare(selected, 'b'), ['a', 'c'])
})

test('formatUsd trims trailing zeros', () => {
    assert.equal(formatUsd(2.5), '$2.5')
    assert.equal(formatUsd(0.0006), '$0.0006')
    assert.equal(formatUsd(null), '-')
})
