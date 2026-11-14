export type CheckoutSource = 'cart' | 'buynow'

/** An embedded single-product checkout is explicit; existing routes retain their original mode. */
export function resolveCheckoutSource(routeMode: unknown, source?: 'buynow'): CheckoutSource {
  return source === 'buynow' || routeMode === 'buynow' ? 'buynow' : 'cart'
}

/** A missing single-product draft must never turn into an unrelated cart purchase. */
export function checkoutSourceItems<T>(source: CheckoutSource, cart: T[], draft: T | null): T[] {
  return source === 'buynow' ? draft ? [draft] : [] : cart
}
