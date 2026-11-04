import { amountToCents } from '../../../utils/money.ts'

/** Display only: checkout and payment amounts remain owned by the native flow. */
export function formatServiceMoney(amount: unknown, currency: string, locale: string): string | null {
  const cents = amountToCents(amount)
  if (cents === null || cents < 0) return null
  try {
    return new Intl.NumberFormat(locale, {
      style: 'currency',
      currency,
      minimumFractionDigits: cents % 100 === 0 ? 0 : 2,
      maximumFractionDigits: 2,
    }).format(cents / 100)
  } catch {
    return null
  }
}
