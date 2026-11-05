import { buildOffers, type BuildOffersOptions, type PricingResponse } from './pricing.ts'
import { normalizeUsdCnyRate, type DisplayCurrency } from './currencyDisplay.ts'
import { amountToCents } from '../../../utils/money.ts'

export interface ApiCostRate {
  key: string
  modelId: string
  group: string
  groupRatio: number
  unit: 'per_million_tokens' | 'per_call'
  inputUsd: number
  outputUsd: number | null
  cacheReadUsd: number | null
  cacheWriteUsd: number | null
  sourceVersion: string | null
}

const textEndpoints = new Set(['openai', 'openai-response', 'openai-response-compact', 'anthropic', 'gemini', 'embeddings'])
const nonnegative = (value: unknown): value is number => typeof value === 'number' && Number.isFinite(value) && value >= 0

/** The same public source and group expansion as ModelSquare. Cache-only text
 * rates have explicit buckets; expression/plugin/task/multimodal rates remain
 * unknown instead of being evaluated with a legacy ratio formula.
 * Native reference: new-api web/src/features/pricing/lib/price.ts.
 */
export function buildApiCostRates(response: PricingResponse, options: BuildOffersOptions): ApiCostRate[] {
  const rates: ApiCostRate[] = []
  if (!response.group_ratio || typeof response.group_ratio !== 'object' || Array.isArray(response.group_ratio)) return rates
  const vendors = Array.isArray(response.vendors) ? response.vendors : []
  for (const model of Array.isArray(response.data) ? response.data : []) {
    if (!model || typeof model.model_name !== 'string' || !model.model_name.trim() || !Array.isArray(model.enable_groups) || model.enable_groups.some(group => typeof group !== 'string')) continue
    if (model.supported_endpoint_types !== undefined && (!Array.isArray(model.supported_endpoint_types) || model.supported_endpoint_types.some(type => !textEndpoints.has(type)))) continue
    const scalar = { ...model, cache_ratio: undefined, create_cache_ratio: undefined }
    for (const offer of buildOffers({ ...response, vendors, data: [scalar] }, options)) {
      if (offer.unit === 'conditional' || !nonnegative(offer.inputPrice)) continue
      const groupRatio = response.group_ratio?.[offer.group]
      if (!nonnegative(groupRatio)) continue
      const cachePrice = (ratio: unknown) => {
        if (offer.unit !== 'per_million_tokens' || !nonnegative(ratio)) return null
        const price = offer.inputPrice! * ratio
        return nonnegative(price) ? price : null
      }
      rates.push({
        key: offer.key, modelId: offer.modelId, group: offer.group, groupRatio,
        unit: offer.unit, inputUsd: offer.inputPrice, outputUsd: offer.outputPrice,
        cacheReadUsd: cachePrice(model.cache_ratio), cacheWriteUsd: cachePrice(model.create_cache_ratio),
        sourceVersion: typeof offer.sourceVersion === 'string' ? offer.sourceVersion : null,
      })
    }
  }
  return rates
}

/** Empty values stay unknown; zero is accepted only when explicitly entered. */
export function parseApiRate(value: unknown): number | null {
  if (typeof value !== 'string' && typeof value !== 'number') return null
  const raw = String(value).trim()
  if (!/^\d+(?:\.\d{1,12})?$/.test(raw)) return null
  const amount = Number(raw)
  return nonnegative(amount) && amount <= 1_000_000_000 ? amount : null
}

export function parseApiUsage(value: unknown): number | null {
  if (typeof value !== 'string' && typeof value !== 'number') return null
  const raw = String(value).trim()
  if (!/^\d+$/.test(raw)) return null
  const count = Number(raw)
  return Number.isSafeInteger(count) && count >= 0 && count <= 1_000_000_000 ? count : null
}

export interface ApiCostUsage {
  calls: unknown
  input: unknown
  output: unknown
  cacheRead: unknown
  cacheWrite: unknown
}

export interface ApiCostEstimate {
  monthlyUsd: number
  perCallUsd: number
  monthlyInput: number
  monthlyOutput: number
}

export function estimateApiCost(rate: ApiCostRate | null, usage: ApiCostUsage): ApiCostEstimate | null {
  const calls = parseApiUsage(usage.calls)
  if (!rate || calls === null || !nonnegative(rate.inputUsd)) return null
  if (rate.unit === 'per_call') {
    const monthlyUsd = calls * rate.inputUsd
    return Number.isFinite(monthlyUsd) && monthlyUsd <= Number.MAX_SAFE_INTEGER / 100
      ? { monthlyUsd, perCallUsd: rate.inputUsd, monthlyInput: 0, monthlyOutput: 0 } : null
  }
  const input = parseApiUsage(usage.input), output = parseApiUsage(usage.output)
  const cacheRead = parseApiUsage(usage.cacheRead), cacheWrite = parseApiUsage(usage.cacheWrite)
  if (input === null || output === null || cacheRead === null || cacheWrite === null || cacheRead + cacheWrite > input || !nonnegative(rate.outputUsd)) return null
  if ((cacheRead > 0 && !nonnegative(rate.cacheReadUsd)) || (cacheWrite > 0 && !nonnegative(rate.cacheWriteUsd))) return null
  const monthlyInput = calls * input, monthlyOutput = calls * output
  if (!Number.isSafeInteger(monthlyInput) || !Number.isSafeInteger(monthlyOutput)) return null
  const perCallUsd = ((input - cacheRead - cacheWrite) * rate.inputUsd + output * rate.outputUsd
    + cacheRead * (rate.cacheReadUsd ?? 0) + cacheWrite * (rate.cacheWriteUsd ?? 0)) / 1_000_000
  const monthlyUsd = calls * perCallUsd
  return nonnegative(monthlyUsd) && monthlyUsd <= Number.MAX_SAFE_INTEGER / 100
    ? { monthlyUsd, perCallUsd, monthlyInput, monthlyOutput } : null
}

/** Convert only after aggregating fractional USD token costs. Product money
 * helpers round at cents, which would otherwise erase sub-cent USD estimates
 * before converting to CNY. This remains a display estimate, never billing.
 */
export function convertApiCost(usd: number, currency: DisplayCurrency, fx: unknown): number | null {
  if (!nonnegative(usd)) return null
  const rate = currency === 'USD' ? 1 : Number(normalizeUsdCnyRate(fx))
  if (!Number.isFinite(rate) || rate <= 0) return null
  const amount = usd * rate
  return nonnegative(amount) && amount <= Number.MAX_SAFE_INTEGER / 100 ? amount : null
}

export function compareApiCost(estimate: number | null, comparison: unknown): { amount: number; difference: number; kind: 'lower' | 'higher' | 'equal' } | null {
  const cents = amountToCents(comparison)
  if (estimate === null || !nonnegative(estimate) || cents === null || cents < 0) return null
  const apiCents = Math.round(estimate * 100)
  if (!Number.isSafeInteger(apiCents)) return null
  const difference = (cents - apiCents) / 100
  return { amount: cents / 100, difference: Math.abs(difference), kind: difference > 0 ? 'lower' : difference < 0 ? 'higher' : 'equal' }
}
