import { amountToCents, centsToAmount } from '../../../utils/money.ts'

export type DisplayCurrency = 'CNY' | 'USD'
const RATE_SCALE = 1_000_000n

/** Operator-published CNY per 1 USD, with no binary floating point conversion. */
export function normalizeUsdCnyRate(value: unknown): string | null {
  if (typeof value !== 'string' && typeof value !== 'number') return null
  const raw = String(value).trim()
  if (!/^\d{1,4}(?:\.\d{1,6})?$/.test(raw)) return null
  const [major = '0', fraction = ''] = raw.split('.')
  const scaled = BigInt(major) * RATE_SCALE + BigInt(fraction.padEnd(6, '0'))
  if (scaled <= 0n || scaled > 1000n * RATE_SCALE) return null
  return `${BigInt(major)}${fraction.replace(/0+$/, '') ? `.${fraction.replace(/0+$/, '')}` : ''}`
}

export function isDisplayCurrency(value: unknown): value is DisplayCurrency {
  return value === 'CNY' || value === 'USD'
}

/** Display quote only. Native order/payment amounts must never use this result. */
export function convertDisplayAmount(amount: unknown, from: string, to: string, rate: unknown): string | null {
  const cents = amountToCents(amount)
  if (cents === null || cents < 0) return null
  if (from === to) return centsToAmount(cents)
  if (!isDisplayCurrency(from) || !isDisplayCurrency(to)) return null
  const normalized = normalizeUsdCnyRate(rate)
  if (!normalized) return null
  const [major = '0', fraction = ''] = normalized.split('.')
  const scaled = BigInt(major) * RATE_SCALE + BigInt(fraction.padEnd(6, '0'))
  const numerator = BigInt(cents) * (from === 'USD' ? scaled : RATE_SCALE)
  const denominator = from === 'USD' ? RATE_SCALE : scaled
  const converted = (numerator * 2n + denominator) / (denominator * 2n)
  if (converted > BigInt(Number.MAX_SAFE_INTEGER)) return null
  return centsToAmount(Number(converted))
}

export function displayCurrencyOptions(settlement: string, rate: unknown): string[] {
  return isDisplayCurrency(settlement) && normalizeUsdCnyRate(rate) ? ['CNY', 'USD'] : [settlement]
}

export function resolveDisplayCurrency(settlement: string, rate: unknown, configured: unknown, preferred: unknown): string {
  const available = displayCurrencyOptions(settlement, rate)
  if (typeof preferred === 'string' && available.includes(preferred)) return preferred
  if (typeof configured === 'string' && available.includes(configured)) return configured
  return settlement
}
