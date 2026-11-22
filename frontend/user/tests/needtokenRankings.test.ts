import test from 'node:test'
import assert from 'node:assert/strict'
import {
  findBestOfferForModel,
  formatCompactTokens,
  formatGrowthPercent,
  formatSharePercent,
  normalizeRankingPeriod,
  normalizeRankingsResponse,
  resolveRankChange,
} from '../src/templates/needtoken/lib/rankings.ts'
import type { ModelOffer } from '../src/templates/needtoken/lib/pricing.ts'

test('normalizeRankingPeriod enforces period allowlist and defaults to week', () => {
  assert.equal(normalizeRankingPeriod('today'), 'today')
  assert.equal(normalizeRankingPeriod('WEEK'), 'week')
  assert.equal(normalizeRankingPeriod(' month '), 'month')
  assert.equal(normalizeRankingPeriod('year'), 'year')
  assert.equal(normalizeRankingPeriod('decade'), 'week')
  assert.equal(normalizeRankingPeriod(''), 'week')
  assert.equal(normalizeRankingPeriod(undefined), 'week')
})

test('normalizeRankingsResponse safely normalizes payload and caps models at 20', () => {
  const models = Array.from({ length: 25 }, (_, i) => ({
    rank: i + 1,
    previous_rank: i === 0 ? undefined : i + 2,
    model_name: `model-${i + 1}`,
    vendor: 'OpenAI',
    total_tokens: 1_000_000 * (25 - i),
    share: 0.05,
    growth_pct: 12.34,
  }))
  const normalized = normalizeRankingsResponse({
    success: true,
    data: {
      models,
      vendors: [
        { rank: 1, vendor: 'OpenAI', total_tokens: 5_000_000, share: 0.65, growth_pct: 8.5, models_count: 5, top_model: 'model-1' },
      ],
      top_movers: [
        { model_name: 'model-2', vendor: 'OpenAI', rank_delta: 3, current_rank: 2, growth_pct: 45.2 },
      ],
      top_droppers: [
        { model_name: 'model-5', vendor: 'Anthropic', rank_delta: -2, current_rank: 5, growth_pct: -18.1 },
      ],
    },
  })
  assert.equal(normalized.models.length, 20)
  assert.equal(normalized.models[0]?.model_name, 'model-1')
  assert.equal(normalized.models[0]?.previous_rank, undefined)
  assert.equal(normalized.vendors.length, 1)
  assert.equal(normalized.vendors[0]?.top_model, 'model-1')
  assert.equal(normalized.top_movers[0]?.rank_delta, 3)
  assert.equal(normalized.top_droppers[0]?.rank_delta, -2)

  const empty = normalizeRankingsResponse(null)
  assert.deepEqual(empty.models, [])
  assert.deepEqual(empty.vendors, [])
})

test('resolveRankChange distinguishes new, up, down, and unchanged ranks', () => {
  assert.deepEqual(resolveRankChange(1, undefined), { kind: 'new', delta: 0 })
  assert.deepEqual(resolveRankChange(2, 5), { kind: 'up', delta: 3 })
  assert.deepEqual(resolveRankChange(6, 4), { kind: 'down', delta: 2 })
  assert.deepEqual(resolveRankChange(3, 3), { kind: 'same', delta: 0 })
})

test('formatting helpers render compact tokens, share, and growth clearly', () => {
  assert.equal(formatCompactTokens(0), '0')
  assert.equal(formatCompactTokens(950), '950')
  assert.equal(formatCompactTokens(12_500), '12.5K')
  assert.equal(formatCompactTokens(45_600_000), '45.6M')
  assert.equal(formatCompactTokens(1_250_000_000), '1.25B')
  assert.equal(formatCompactTokens(2_000_000_000_000), '2T')

  assert.equal(formatSharePercent(0), '0%')
  assert.equal(formatSharePercent(0.0004), '<0.1%')
  assert.equal(formatSharePercent(0.324), '32.4%')

  assert.equal(formatGrowthPercent(24.56), '+24.6%')
  assert.equal(formatGrowthPercent(-8.21), '-8.2%')
  assert.equal(formatGrowthPercent(0.01), '0.0%')
})

test('findBestOfferForModel prefers default group, then lowest group ratio', () => {
  const baseOffer: Omit<ModelOffer, 'key' | 'modelId' | 'group' | 'groupRatio' | 'inputPrice' | 'outputPrice'> = {
    vendor: 'OpenAI',
    groupDescription: '',
    supplyLine: null,
    quotaType: 0,
    unit: 'per_million_tokens',
    cachePrice: null,
    isDefault: true,
    condition: null,
    sourceVersion: null,
    usageUnits: [],
    endpoints: [],
    tags: [],
    description: '',
  }
  const offers: ModelOffer[] = [
    { ...baseOffer, key: 'gpt-5.4@vip', modelId: 'gpt-5.4', group: 'vip', groupRatio: 0.8, inputPrice: 2, outputPrice: 8 },
    { ...baseOffer, key: 'gpt-5.4@default', modelId: 'gpt-5.4', group: 'default', groupRatio: 1.0, inputPrice: 2.5, outputPrice: 10 },
    { ...baseOffer, key: 'claude-sonnet-4-6@pro', modelId: 'claude-sonnet-4-6', group: 'pro', groupRatio: 1.5, inputPrice: 4.5, outputPrice: 22.5 },
    { ...baseOffer, key: 'claude-sonnet-4-6@eco', modelId: 'claude-sonnet-4-6', group: 'eco', groupRatio: 0.7, inputPrice: 2.1, outputPrice: 10.5 },
  ]
  assert.equal(findBestOfferForModel(offers, 'gpt-5.4')?.group, 'default')
  assert.equal(findBestOfferForModel(offers, 'claude-sonnet-4-6')?.group, 'eco')
  assert.equal(findBestOfferForModel(offers, 'nonexistent'), null)
})
