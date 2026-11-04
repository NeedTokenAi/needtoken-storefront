import type { AdminSKUDisplayMeta, LocalizedText } from '@/api/types'

const languages = ['zh-CN', 'zh-TW', 'en-US'] as const
const record = (raw: unknown): Record<string, unknown> => raw && typeof raw === 'object' && !Array.isArray(raw) ? raw as Record<string, unknown> : {}
const localized = (raw: unknown): LocalizedText => {
  const source = record(raw)
  return Object.fromEntries(languages.map(language => [language, typeof source[language] === 'string' ? source[language] : '']))
}

/** Presentation copy is independent from SKU stock, price, and specification identifiers. */
export function normalizeSKUDisplayMeta(raw: unknown, legacySpecs?: unknown): AdminSKUDisplayMeta {
  const source = record(raw)
  const legacyPrice = record(legacySpecs).official_usd
  const legacyText = typeof legacyPrice === 'string' || typeof legacyPrice === 'number' ? String(legacyPrice).trim() : ''
  const reference = raw == null && legacyText && isValidOfficialUSDPrice(legacyText) ? legacyText : ''
  return {
    label: localized(source.label),
    title: localized(source.title),
    description: localized(source.description),
    feature_lines: localized(source.feature_lines),
    price_suffix: localized(source.price_suffix),
    official_price_usd: typeof source.official_price_usd === 'string' ? source.official_price_usd : reference,
  }
}

export function isValidOfficialUSDPrice(raw: string): boolean {
  const text = raw.trim()
  return !text || (/^(?:0|[1-9]\d{0,7})(?:\.\d{1,2})?$/.test(text) && Number(text) > 0)
}

/** Preserve unchanged DEFAULT copy; a variant-to-single transition explicitly replaces inherited variant copy. */
export function changedDefaultSKUDisplayMeta(current: AdminSKUDisplayMeta, original: AdminSKUDisplayMeta, resetAfterVariantCollapse = false): AdminSKUDisplayMeta | undefined {
  const value = normalizeSKUDisplayMeta(current)
  return !resetAfterVariantCollapse && JSON.stringify(value) === JSON.stringify(normalizeSKUDisplayMeta(original)) ? undefined : value
}

export function isValidSKUFeatureLines(lines: LocalizedText): boolean {
  return Object.values(lines).every(value => {
    const entries = value.split(/\r?\n/).map(line => line.trim()).filter(Boolean)
    return entries.length <= 8 && entries.every(line => Array.from(line).length <= 160)
  })
}
