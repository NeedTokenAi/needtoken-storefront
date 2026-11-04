import { describe, expect, it } from 'vitest'
import { normalizeNeedtokenPricing, pricingPatchForSave, pricingPatchForSettlementCurrencyChange } from './needtokenPricing'
import { mergeSiteSettingPatch } from './needtokenSettings'

describe('NeedToken display currency settings', () => {
  it('keeps native currency without inventing a default rate', () => {
    expect(normalizeNeedtokenPricing(undefined)).toEqual({ default_display_currency: 'CNY', usd_cny_rate: '', rate_updated_at: '' })
    expect(normalizeNeedtokenPricing(undefined, 'USD').default_display_currency).toBe('USD')
    expect(normalizeNeedtokenPricing({ usd_cny_rate: '7.123456' }).usd_cny_rate).toBe('7.123456')
  })
  it('changes the rate timestamp only for a changed rate and clears it when disabled', () => {
    const previous = { default_display_currency: 'CNY' as const, usd_cny_rate: '7.25', rate_updated_at: '2026-01-01T00:00:00Z' }
    expect(pricingPatchForSave({ ...previous, default_display_currency: 'USD' }, previous, 'now').rate_updated_at).toBe(previous.rate_updated_at)
    expect(pricingPatchForSave({ ...previous, usd_cny_rate: '0007.250000' }, previous, 'now').rate_updated_at).toBe(previous.rate_updated_at)
    expect(pricingPatchForSave({ ...previous, usd_cny_rate: '7.3' }, previous, 'now').rate_updated_at).toBe('now')
    expect(pricingPatchForSave({ ...previous, usd_cny_rate: '' }, previous, 'now').rate_updated_at).toBe('')
  })
  it('does not change settlement or unrelated settings while saving display config', () => {
    const latest = { currency: 'CNY', brand: { site_name: 'NeedToken.AI' }, legal: { terms: 'keep' }, needtoken_support: { enabled: false } }
    const pricing = normalizeNeedtokenPricing({ default_display_currency: 'USD', usd_cny_rate: '7.25' })
    const result = mergeSiteSettingPatch(latest, { needtoken_pricing: pricing })
    expect(result).toEqual({ ...latest, needtoken_pricing: pricing })
    expect(result.currency).toBe('CNY')
    expect(latest).not.toHaveProperty('needtoken_pricing')
  })
})

describe('settlement currency changes in Basic settings', () => {
  it('aligns an inherited rate-free default in both directions without changing rates or timestamps', () => {
    for (const [previous, next] of [['CNY', 'USD'], ['USD', 'CNY']] as const) {
      const latest = { currency: previous, needtoken_pricing: { default_display_currency: previous, usd_cny_rate: '', rate_updated_at: '' } }
      const patch = pricingPatchForSettlementCurrencyChange(latest, next)
      expect(patch).toEqual({ needtoken_pricing: { ...latest.needtoken_pricing, default_display_currency: next } })
      expect(latest.needtoken_pricing.default_display_currency).toBe(previous)
      const merged = mergeSiteSettingPatch(latest, { ...patch, currency: next })
      expect(merged.currency).toBe(next)
    }
  })
  it('preserves explicit display preferences and configured rates', () => {
    expect(pricingPatchForSettlementCurrencyChange({ currency: 'CNY', needtoken_pricing: { default_display_currency: 'USD', usd_cny_rate: '' } }, 'USD')).toEqual({})
    expect(pricingPatchForSettlementCurrencyChange({ currency: 'CNY', needtoken_pricing: { default_display_currency: 'CNY', usd_cny_rate: '7.25', rate_updated_at: 'keep' } }, 'USD')).toEqual({})
    expect(pricingPatchForSettlementCurrencyChange({ currency: 'CNY', needtoken_pricing: { default_display_currency: 'CNY', usd_cny_rate: 7.25 } }, 'USD')).toEqual({})
  })
  it('does not inject pricing for missing config, unchanged currency or other settlement currencies', () => {
    expect(pricingPatchForSettlementCurrencyChange({ currency: 'CNY' }, 'USD')).toEqual({})
    expect(pricingPatchForSettlementCurrencyChange({ currency: 'CNY', needtoken_pricing: { default_display_currency: 'CNY', usd_cny_rate: '' } }, 'CNY')).toEqual({})
    expect(pricingPatchForSettlementCurrencyChange({ currency: 'CNY', needtoken_pricing: { default_display_currency: 'CNY', usd_cny_rate: '' } }, 'EUR')).toEqual({})
  })
})
