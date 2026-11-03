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
    assert.equal(formatUsd(0.0000001), '<$0.000001')
    assert.equal(formatUsd(-1), '-')
})

test('expression, plugin, video and conditional prices never use legacy token numbers', () => {
    const base = { model_name: 'conditional', quota_type: 0, model_ratio: 0.5, model_price: 1, completion_ratio: 4, enable_groups: ['pool'] }
    const variants = [
        { ...base, model_name: 'expr', billing_mode: 'tiered_expr', billing_expr: 'p > 100 ? p * 3 : p' },
        { ...base, model_name: 'plugin', billing_plugin_variants: [{ plugin_key: 'vendor-a', billing_expr: 'u("seconds") * 2' }] },
        { ...base, model_name: 'video', supported_endpoint_types: ['openai-video'] },
        { ...base, model_name: 'task', billing_usage_schema: { seconds: { type: 'number', unit: 'seconds' } } },
        { ...base, model_name: 'audio', audio_ratio: 20 },
        { ...base, model_name: 'cache', cache_ratio: 0.1 },
        { ...base, model_name: 'future-mode', quota_type: 2 },
    ]
    const offers = buildOffers({ data: variants, group_ratio: { pool: 1 }, pricing_version: 'source-marker' }, options)
    assert.equal(offers.length, variants.length)
    for (const offer of offers) {
        assert.equal(offer.unit, 'conditional', offer.modelId)
        assert.equal(offer.inputPrice, null, offer.modelId)
        assert.equal(offer.outputPrice, null, offer.modelId)
        assert.equal(offer.sourceVersion, 'source-marker')
    }
    assert.deepEqual(offers.find(o => o.modelId === 'task')?.usageUnits, ['seconds (seconds)'])
})

test('conditional offers sort last without entering numeric price comparison', () => {
    const offers = buildOffers({ ...response, data: [...response.data!, { model_name: 'conditional', quota_type: 0, model_ratio: 0.001, model_price: 0, completion_ratio: 1, billing_expr: 'dynamic()', enable_groups: ['pool'] }] }, options)
    for (const sort of ['price_asc', 'price_desc'] as const) assert.equal(sortOffers(offers, sort, options.defaultModels).at(-1)?.modelId, 'conditional')
})

test('only authoritative finite nonnegative group prices are shown and all groups expand safely', () => {
    const model = { model_name: 'known', quota_type: 0, model_ratio: 1, model_price: 0, completion_ratio: 1, enable_groups: ['all'] }
    const offers = buildOffers({ data: [model], group_ratio: { pool: 1, free: 0, negative: -1, invalid: Number.NaN }, pricing_version: 'v-source' }, options)
    assert.deepEqual(offers.map(o => o.group), ['pool', 'free'])
    assert.equal(offers[1]?.inputPrice, 0)
    assert.equal(offers[0]?.sourceVersion, 'v-source')
    const noVersion = buildOffers({ data: [model], group_ratio: { pool: 1 } }, options)
    assert.equal(noVersion[0]?.sourceVersion, null)
    const overflow = buildOffers({ data: [{ ...model, model_ratio: Number.MAX_VALUE }], group_ratio: { pool: 10 } }, options)
    assert.equal(overflow[0]?.unit, 'conditional')
    assert.equal(overflow[0]?.inputPrice, null)
})
