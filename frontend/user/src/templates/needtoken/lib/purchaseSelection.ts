import { amountToCents } from '../../../utils/money.ts'

export type RequestedSkuSelection =
  | { kind: 'absent' }
  | { kind: 'invalid' }
  | { kind: 'valid'; id: number }

/** A link can suggest a SKU; only an active SKU from the loaded product can be selected. */
export function resolveRequestedSku(raw: unknown, skus: readonly { id?: unknown; is_active?: unknown }[]): RequestedSkuSelection {
  if (raw === undefined) return { kind: 'absent' }
  if (typeof raw !== 'string' || !/^[1-9]\d*$/.test(raw)) return { kind: 'invalid' }
  const id = Number(raw)
  if (!Number.isSafeInteger(id)) return { kind: 'invalid' }
  const exists = skus.some((sku) => sku.is_active === true && Number(sku.id) === id)
  return exists ? { kind: 'valid', id } : { kind: 'invalid' }
}
/** Zero is a valid price; absent, malformed, negative or lossy amounts are not. */
export function isKnownPurchasePrice(raw: unknown): boolean {
  if (typeof raw !== 'string' && typeof raw !== 'number') return false
  const text = String(raw).trim()
  if (!/^\d+(?:\.\d{1,2})?$/.test(text)) return false
  const cents = amountToCents(text)
  return cents !== null && cents >= 0
}
