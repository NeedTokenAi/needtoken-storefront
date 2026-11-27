import test from 'node:test'
import assert from 'node:assert/strict'
import {
    buildDynamicPlanMultiplierOptions,
    buildOffers,
    buildPricingGroups,
    effectiveDiscountPercent,
    effectivePlanOfferPrice,
    filterOffers,
    formatUsd,
    inferVendorFromModelName,
    resolveEndpointBadges,
    selectOffersByGroup,
    sortOffers,
    toggleCompare,
    type PricingResponse,
} from '../src/templates/needtoken/lib/pricing.ts'

const options = {
    defaultModels: ['gpt-5.4', 'claude-sonnet-4-6'],
    blockedModels: ['gpt-5.2'],
    supplyLineByGroup: { pool: 'pool' },
}

const response: PricingResponse = {
    data: [
        {
            model_name: 'claude-sonnet-4-6',
            quota_type: 0,
            model_ratio: 1.5,
            model_price: 0,
            completion_ratio: 5,
            cache_ratio: 0.1,
            vendor_id: 2,
            enable_groups: ['pool', 'aws'],
            supported_endpoint_types: ['anthropic', 'openai'],
            tags: 'coding, agent',
            description: 'Flagship coding and reasoning model',
        },
        {
            model_name: 'gpt-5.4',
            quota_type: 0,
            model_ratio: 1.25,
            model_price: 0,
            completion_ratio: 4,
            cache_ratio: 0.25,
            vendor_id: 1,
            enable_groups: ['pool'],
            supported_endpoint_types: ['openai-response', 'openai'],
        },
        // 未配价：new-api 兜底倍率 37.5
        { model_name: 'gpt-unpriced', quota_type: 0, model_ratio: 37.5, model_price: 0, completion_ratio: 1, enable_groups: ['pool'] },
        // 黑名单模型
        { model_name: 'gpt-5.2', quota_type: 0, model_ratio: 1, model_price: 0, completion_ratio: 1, enable_groups: ['pool'] },
        // 按次计费
        { model_name: 'image-x', quota_type: 1, model_ratio: 0, model_price: 0.04, completion_ratio: 0, owner_by: 'acme', enable_groups: ['aws'] },
    ],
    vendors: [{ id: 1, name: 'OpenAI' }, { id: 2, name: 'Anthropic' }],
    group_ratio: { pool: 1, aws: 1.5 },
    usable_group: { pool: '标准号池分组', aws: 'AWS 企业专线' },
    supported_endpoint: {
        'openai-response': { path: '/v1/responses', method: 'POST' },
        anthropic: { path: '/v1/messages', method: 'POST' },
    },
}

test('buildOffers splits each model by group, computes cache price, and preserves group descriptions', () => {
    const offers = buildOffers(response, options)
    const claudeAws = offers.find(o => o.key === 'claude-sonnet-4-6@aws')
    assert.ok(claudeAws)
    assert.equal(claudeAws.inputPrice, 1.5 * 2 * 1.5)
    assert.equal(claudeAws.outputPrice, 1.5 * 2 * 1.5 * 5)
    assert.ok(Math.abs((claudeAws.cachePrice ?? 0) - 0.45) < 1e-9)
    assert.equal(claudeAws.vendor, 'Anthropic')
    assert.equal(claudeAws.groupDescription, 'AWS 企业专线')
    assert.equal(claudeAws.supplyLine, null)
    assert.deepEqual(claudeAws.tags, ['coding', 'agent'])
    assert.deepEqual(claudeAws.endpoints.map(e => e.label), ['Messages', 'Chat'])

    const claudePool = offers.find(o => o.key === 'claude-sonnet-4-6@pool')
    assert.equal(claudePool?.supplyLine, 'pool')
    assert.equal(claudePool?.inputPrice, 3)
    assert.ok(Math.abs((claudePool?.cachePrice ?? 0) - 0.3) < 1e-9)
})

test('buildPricingGroups and selectOffersByGroup support live group switching and deduplication', () => {
    const groups = buildPricingGroups(response, options.supplyLineByGroup)
    assert.deepEqual(groups.map(g => ({ name: g.name, ratio: g.ratio, desc: g.description })), [
        { name: 'pool', ratio: 1, desc: '标准号池分组' },
        { name: 'aws', ratio: 1.5, desc: 'AWS 企业专线' },
    ])

    const offers = buildOffers(response, options)
    const recommended = selectOffersByGroup(offers, '')
    assert.equal(recommended.filter(o => o.modelId === 'claude-sonnet-4-6').length, 1)
    assert.equal(recommended.find(o => o.modelId === 'claude-sonnet-4-6')?.group, 'pool')

    const awsScoped = selectOffersByGroup(offers, 'aws')
    assert.deepEqual(awsScoped.map(o => o.key), ['claude-sonnet-4-6@aws', 'image-x@aws'])
})

test('inferVendorFromModelName and resolveEndpointBadges classify common gateway models', () => {
    assert.equal(inferVendorFromModelName('gpt-5.4'), 'OpenAI')
    assert.equal(inferVendorFromModelName('claude-opus-4-6'), 'Anthropic')
    assert.equal(inferVendorFromModelName('gemini-2.5-pro'), 'Google')
    assert.equal(inferVendorFromModelName('deepseek-v3'), 'DeepSeek')
    assert.equal(inferVendorFromModelName('grok-4'), 'xAI')
    assert.deepEqual(
        resolveEndpointBadges(['openai-response', 'openai-response-compact', 'gemini']).map(b => b.label),
        ['Responses', 'Gemini'],
    )
})

test('buildOffers drops unpriced and blocked models', () => {
    const ids = buildOffers(response, options).map(o => o.modelId)
    assert.equal(ids.includes('gpt-unpriced'), false)
    assert.equal(ids.includes('gpt-5.2'), false)
})

test('buildOffers handles per-call pricing and empty responses', () => {
    const image = buildOffers(response, options).find(o => o.modelId === 'image-x')
    assert.equal(image?.unit, 'per_call')
    assert.equal(image?.inputPrice, 0.04 * 1.5)
    assert.equal(image?.outputPrice, null)
    assert.equal(image?.cachePrice, null)
    assert.equal(image?.vendor, 'acme')
    assert.deepEqual(buildOffers({}, options), [])
})

test('sortOffers puts default models first and sorts token prices before per-call', () => {
    const offers = buildOffers(response, options)
    assert.deepEqual(sortOffers(offers, 'default', options.defaultModels).map(o => o.key), [
        'gpt-5.4@pool', 'claude-sonnet-4-6@aws', 'claude-sonnet-4-6@pool', 'image-x@aws',
    ])
    assert.deepEqual(sortOffers(offers, 'price_asc', options.defaultModels).map(o => o.key), [
        'gpt-5.4@pool', 'claude-sonnet-4-6@pool', 'claude-sonnet-4-6@aws', 'image-x@aws',
    ])
    assert.deepEqual(sortOffers(offers, 'price_desc', options.defaultModels).map(o => o.key), [
        'claude-sonnet-4-6@aws', 'claude-sonnet-4-6@pool', 'gpt-5.4@pool', 'image-x@aws',
    ])
})

test('filterOffers matches vendor, supply line, billingType and keyword', () => {
    const offers = buildOffers(response, options)
    assert.deepEqual(filterOffers(offers, { supply: 'pool' }).map(o => o.key), ['claude-sonnet-4-6@pool', 'gpt-5.4@pool'])
    assert.deepEqual(filterOffers(offers, { supply: 'group:aws', vendor: 'Anthropic' }).map(o => o.key), ['claude-sonnet-4-6@aws'])
    assert.deepEqual(filterOffers(offers, { billingType: 'per_call' }).map(o => o.key), ['image-x@aws'])
    assert.deepEqual(filterOffers(offers, { keyword: 'reasoning' }).map(o => o.key), ['claude-sonnet-4-6@pool', 'claude-sonnet-4-6@aws'])
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

test('expression, plugin, video and audio/image multimodal prices never use legacy token numbers', () => {
    const base = { model_name: 'conditional', quota_type: 0, model_ratio: 0.5, model_price: 1, completion_ratio: 4, enable_groups: ['pool'] }
    const variants = [
        { ...base, model_name: 'expr', billing_mode: 'tiered_expr', billing_expr: 'p > 100 ? p * 3 : p' },
        { ...base, model_name: 'plugin', billing_plugin_variants: [{ plugin_key: 'vendor-a', billing_expr: 'u("seconds") * 2' }] },
        { ...base, model_name: 'video', supported_endpoint_types: ['openai-video'] },
        { ...base, model_name: 'task', billing_usage_schema: { seconds: { type: 'number', unit: 'seconds' } } },
        { ...base, model_name: 'audio', audio_ratio: 20 },
        { ...base, model_name: 'image-mm', image_ratio: 5 },
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

test('buildOffers computes official pricing, ops overrides, discount percent, and plan multiplier savings', () => {
    const priced = buildOffers({
        data: [
            {
                model_name: 'claude-sonnet-4-6',
                quota_type: 0,
                model_ratio: 1.5,
                model_price: 0,
                completion_ratio: 5,
                cache_ratio: 0.1,
                official_model_ratio: 2.5,
                official_completion_ratio: 5,
                enable_groups: ['pool'],
            },
            {
                model_name: 'gpt-5.4',
                quota_type: 0,
                model_ratio: 1.0,
                model_price: 0,
                completion_ratio: 4,
                enable_groups: ['pool'],
            },
        ],
        group_ratio: { pool: 0.6 },
        needtoken_meta: {
            official_overrides: {
                'gpt-5.4': { input_usd: 5, output_usd: 20, cache_usd: 0.5, context_length: '2M' },
            },
        },
    }, options)

    const claude = priced.find(o => o.modelId === 'claude-sonnet-4-6')!
    // Actual input = 1.5 * 2 * 0.6 = 1.8; official input = 2.5 * 2 = 5 -> 64% off
    assert.ok(Math.abs((claude.inputPrice ?? 0) - 1.8) < 1e-9)
    assert.equal(claude.officialInputPrice, 5)
    assert.equal(claude.officialOutputPrice, 25)
    assert.equal(claude.discountPercent, 64)

    const gpt = priced.find(o => o.modelId === 'gpt-5.4')!
    // Actual input = 1.0 * 2 * 0.6 = 1.2; ops override official = 5 -> 76% off, context = 2M
    assert.ok(Math.abs((gpt.inputPrice ?? 0) - 1.2) < 1e-9)
    assert.equal(gpt.officialInputPrice, 5)
    assert.equal(gpt.officialOutputPrice, 20)
    assert.equal(gpt.officialCachePrice, 0.5)
    assert.equal(gpt.contextLength, '2M')
    assert.equal(gpt.discountPercent, 76)

    // With 2x subscription plan multiplier: 1.2 / 2 = 0.6 vs $5 official -> 88% off
    const planPrice = effectivePlanOfferPrice(gpt.inputPrice, 2)
    assert.ok(Math.abs((planPrice ?? 0) - 0.6) < 1e-9)
    assert.equal(effectiveDiscountPercent(gpt.officialInputPrice, planPrice), 88)
    assert.equal(effectiveDiscountPercent(5, 5), null)
    assert.equal(effectivePlanOfferPrice(1.2, 0.5), 1.2)
})

test('buildDynamicPlanMultiplierOptions derives sorted unique multipliers without hardcoded 2x/3x fallback', () => {
    assert.deepEqual(buildDynamicPlanMultiplierOptions([]), [])
    assert.deepEqual(
        buildDynamicPlanMultiplierOptions([
            { id: 1, title: 'Base 1x', multiplier: 1, price: '10', currency: 'CNY', total_quota_usd: 10, cycle_days: 30, group: 'pool' } as any,
        ]),
        [],
    )
    const opts = buildDynamicPlanMultiplierOptions([
        { id: 10, title: 'Pro 3x', multiplier: 3, price: '99', currency: 'CNY', total_quota_usd: 40, cycle_days: 30, group: 'pool' } as any,
        { id: 11, title: 'Plus 2x', multiplier: 2, price: '49', currency: 'CNY', total_quota_usd: 14, cycle_days: 30, group: 'pool' } as any,
        { id: 12, title: 'Dup 2x', multiplier: 2, price: '59', currency: 'CNY', total_quota_usd: 17, cycle_days: 30, group: 'pool' } as any,
        { id: 13, title: 'Ultra 5x', multiplier: 5, price: '199', currency: 'CNY', total_quota_usd: 140, cycle_days: 30, group: 'pool' } as any,
    ])
    assert.deepEqual(
        opts.map((o) => o.multiplier),
        [2, 3, 5],
    )
})
