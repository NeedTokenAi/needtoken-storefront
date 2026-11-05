export interface NeedtokenPricingSettings {
  default_display_currency: 'CNY' | 'USD'
  usd_cny_rate: string
  rate_updated_at: string
}

const asRecord = (raw: unknown): Record<string, unknown> => raw && typeof raw === 'object' && !Array.isArray(raw) ? raw as Record<string, unknown> : {}
const asText = (raw: unknown) => typeof raw === 'string' ? raw.trim() : ''
const canonicalRateText = (raw: unknown): string => {
  const text = asText(raw)
  if (!/^[0-9]{1,4}(?:\.[0-9]{1,6})?$/.test(text)) return text
  const [integer = '', fraction = ''] = text.split('.')
  const whole = integer.replace(/^0+(?=\d)/, '')
  const decimal = fraction.replace(/0+$/, '')
  return decimal ? `${whole}.${decimal}` : whole
}

/** No default rate is invented, and no monetary amount is converted here. */
export function normalizeNeedtokenPricing(raw: unknown, settlementCurrency = 'CNY'): NeedtokenPricingSettings {
  const value = asRecord(raw)
  const fallback = settlementCurrency === 'USD' ? 'USD' : 'CNY'
  const currency = asText(value.default_display_currency).toUpperCase()
  const rate = canonicalRateText(value.usd_cny_rate)
  return {
    default_display_currency: currency === 'CNY' || currency === 'USD' ? currency : fallback,
    usd_cny_rate: rate,
    rate_updated_at: asText(value.rate_updated_at),
  }
}

/** Timestamp changes only when the stored exchange rate changes, never on a display-only switch. */
export function pricingPatchForSave(draft: NeedtokenPricingSettings, previous: NeedtokenPricingSettings, now: string): NeedtokenPricingSettings {
  const rate = canonicalRateText(draft.usd_cny_rate)
  return {
    default_display_currency: draft.default_display_currency,
    usd_cny_rate: rate,
    rate_updated_at: !rate ? '' : rate !== previous.usd_cny_rate ? now : previous.rate_updated_at,
  }
}

/** Keep a rate-free display default following the native currency when Basic settings changes it. */
export function pricingPatchForSettlementCurrencyChange(latest: unknown, nextCurrency: string): Record<string, unknown> {
  const site = asRecord(latest)
  const previousCurrency = asText(site.currency).toUpperCase() || 'CNY'
  const next = nextCurrency.trim().toUpperCase()
  if (previousCurrency === next || !['CNY', 'USD'].includes(previousCurrency) || !['CNY', 'USD'].includes(next)) return {}
  const previousPricing = asRecord(site.needtoken_pricing)
  const rate = previousPricing.usd_cny_rate
  const hasNoRate = rate === undefined || (typeof rate === 'string' && rate.trim() === '')
  if (asText(previousPricing.default_display_currency).toUpperCase() !== previousCurrency || !hasNoRate) return {}
  return { needtoken_pricing: { ...previousPricing, default_display_currency: next } }
}
