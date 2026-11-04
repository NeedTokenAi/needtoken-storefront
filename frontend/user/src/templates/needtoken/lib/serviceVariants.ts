import { amountToCents, centsToAmount } from '../../../utils/money.ts'
import { getLocalizedText } from '../../../utils/resellerSiteConfig.ts'
import type { ServiceSku } from './serviceCatalog.ts'
import { serviceDescriptionText } from './serviceCatalog.ts'

const record = (value: unknown): Record<string, unknown> => {
  if (typeof value === 'string' && value.length < 16000) {
    try { value = JSON.parse(value) } catch { return {} }
  }
  return value && typeof value === 'object' && !Array.isArray(value) ? value as Record<string, unknown> : {}
}
const localized = (value: unknown, locale: string): string => {
  const values = record(value)
  return getLocalizedText(Object.fromEntries(['zh-CN', 'zh-TW', 'en-US'].map(code => [code, values[code]])), locale)
}
const text = (value: unknown, locale: string): string => {
  if (typeof value === 'number' && Number.isFinite(value)) return String(value)
  if (typeof value === 'string') {
    const parsed = record(value)
    if (Object.keys(parsed).some(key => ['zh-CN', 'zh-TW', 'en-US'].includes(key))) return text(parsed, locale)
    return serviceDescriptionText(value).trim()
  }
  return serviceDescriptionText(localized(value, locale))
}
const key = (value: string) => value.toLowerCase().replace(/[\s_-]/g, '')

/** Only buyer-facing option names are considered; import/executor metadata is never a label. */
const optionNameKeys = ['plan', 'tier', 'variant', 'option', 'spec', '档位', '方案', '套餐', '规格', '版本', '等级', '額度', '额度']
const durationKeys = ['period', 'billingperiod', 'duration', 'term', 'cycle', '周期', '週期', '时长', '時長', '订阅周期', '訂閱週期']
const firstSpecText = (values: Record<string, unknown>, names: string[], locale: string): string => {
  const entries = Object.entries(values)
  for (const name of names) {
    const entry = entries.find(([entryKey]) => key(entryKey) === key(name))
    const value = entry ? text(entry[1], locale) : ''
    if (value) return value
  }
  return ''
}

export interface ServiceSkuPresentation {
  label: string
  title: string
  description: string
  features: string[]
  priceSuffix: string
  officialPriceUsd: string | null
}

const officialUsdPrice = (value: unknown): string | null => {
  const raw = typeof value === 'string' ? value.trim() : typeof value === 'number' ? String(value) : ''
  if (!/^\d+(?:\.\d{1,2})?$/.test(raw)) return null
  const cents = amountToCents(raw)
  return cents !== null && cents > 0 ? centsToAmount(cents) : null
}

/** Presentation is independent of the native sale price, stock and selected SKU identity. */
export function serviceSkuPresentation(sku: ServiceSku | null | undefined, locale: string): ServiceSkuPresentation {
  const meta = record(sku?.display_meta)
  const legacy = record(sku?.spec_values)
  const featureText = typeof meta.feature_lines === 'string' ? meta.feature_lines : localized(meta.feature_lines, locale)
  return {
    label: text(meta.label, locale),
    title: text(meta.title, locale),
    description: text(meta.description, locale),
    features: featureText.split(/\r?\n/).map(line => serviceDescriptionText(line)).filter(Boolean).slice(0, 8),
    priceSuffix: text(meta.price_suffix, locale),
    // `official_usd` was used by the first import. It is an explicit USD reference,
    // not a source for deriving or relabelling the native sale amount.
    officialPriceUsd: Object.prototype.hasOwnProperty.call(meta, 'official_price_usd')
      ? officialUsdPrice(meta.official_price_usd) : officialUsdPrice(legacy.official_usd),
  }
}

export function serviceVariantLabel(sku: ServiceSku, locale: string, fallback: string): string {
  const meta = serviceSkuPresentation(sku, locale)
  if (meta.label) return meta.label
  const values = record(sku.spec_values)
  const translated = localized(values, locale)
  if (translated) return serviceDescriptionText(translated)
  const named = firstSpecText(values, optionNameKeys, locale)
  const duration = firstSpecText(values, durationKeys, locale)
  if (named && duration && named !== duration) return `${named} · ${duration}`
  if (named) return named
  if (duration) return duration
  const code = typeof sku.sku_code === 'string' ? sku.sku_code.trim() : ''
  // Raw inventory keys are often opaque. Only a short phrase can be a final label.
  if (code && code.length <= 48 && /[\u3400-\u9fff\s]/.test(code) && !/[_:/\\]/.test(code) && !/[a-f0-9]{16}/i.test(code)) return code
  return fallback
}
