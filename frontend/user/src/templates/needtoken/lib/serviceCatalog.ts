import type { PublicCategory } from '../../../utils/category.ts'
import { amountToCents, centsToAmount } from '../../../utils/money.ts'
import { isKnownPurchasePrice } from './purchaseSelection.ts'

export type ServiceKind = 'recharge' | 'rental'

export interface ServiceSku {
  id: number
  sku_code?: string
  spec_values?: unknown
  display_meta?: unknown
  is_active?: boolean
  price_amount?: unknown
  promotion_price_amount?: unknown
  member_price_amount?: unknown
  stock_status?: string
  is_sold_out?: boolean
}

export interface ServiceProduct {
  id: number
  slug: string
  category_id: number
  category?: PublicCategory
  title?: unknown
  description?: unknown
  images?: unknown
  tags?: string[]
  fulfillment_type?: string
  stock_status?: string
  is_sold_out?: boolean
  skus?: ServiceSku[]
}

/** A view of one native purchase option; IDs, prices and fulfillment stay native. */
export interface ServiceCardProduct extends ServiceProduct {
  cardSkuId?: number
  cardHasVariants?: boolean
}

/** Recharge plans are independent cards even while an older import keeps them
 * under one product. Publishing separate products needs no frontend tier list. */
export const serviceCardProducts = (products: readonly ServiceProduct[], kind: ServiceKind): ServiceCardProduct[] =>
  products.flatMap(product => {
    if (kind !== 'recharge') return [product]
    const skus = activeServiceSkus(product)
    return skus.length > 1
      ? skus.map(sku => ({ ...product, skus: [sku], cardSkuId: sku.id, cardHasVariants: true }))
      : [product]
  })

export const serviceCardKey = (product: ServiceCardProduct): string =>
  `${product.id}:${product.cardSkuId || 'product'}`

export interface ServiceScope {
  kind: ServiceKind
  root: PublicCategory | null
  selected: PublicCategory | null
  categories: PublicCategory[]
}

const categoryId = (value: unknown): number => {
  const id = Number(value)
  return Number.isSafeInteger(id) && id > 0 ? id : 0
}

export const serviceKindFromSlug = (slug: string): ServiceKind | null =>
  slug === 'recharge' || slug === 'rental' ? slug : null

/** Resolve only the current service branch. A missing service root never means ALL products. */
export const resolveServiceScope = (categories: PublicCategory[], slug: string): ServiceScope | null => {
  const byId = new Map(categories.filter(row => categoryId(row.id)).map(row => [row.id, row]))
  const selected = categories.find(row => row.slug === slug) || null
  let current = selected
  const visited = new Set<number>()
  while (current && !visited.has(current.id)) {
    visited.add(current.id)
    const kind = serviceKindFromSlug(String(current.slug || ''))
    if (kind) {
      return {
        kind,
        root: current,
        selected,
        categories: categories.filter(row => categoryId(row.id) && row.parent_id === current!.id),
      }
    }
    current = byId.get(categoryId(current.parent_id)) || null
  }
  const kind = serviceKindFromSlug(slug)
  return kind ? { kind, root: null, selected: null, categories: [] } : null
}

const defaultChildSlug: Record<ServiceKind, string> = {
  recharge: 'chatgpt-recharge',
  rental: 'chatgpt-accounts',
}

/** Root visits open the ChatGPT child when it exists. Passing the root slug keeps ALL selected. */
export const defaultServiceChild = (
  scope: ServiceScope | null,
  explicitRootSlug: string | null = null,
): PublicCategory | null => {
  if (!scope?.root || scope.selected?.id !== scope.root.id) return null
  if (explicitRootSlug === scope.root.slug) return null
  return scope.categories.find(category => category.slug === defaultChildSlug[scope.kind]) || null
}

/** Refreshing identity, language or configuration keeps the visitor's category
 * whenever it still belongs to this service. A new service starts at its default. */
export const homeServiceSelectionId = (scope: ServiceScope | null, previousId = 0): number => {
  if (!scope?.root) return 0
  if (previousId === scope.root.id || scope.categories.some(category => category.id === previousId)) return previousId
  const child = defaultServiceChild({ ...scope, selected: scope.root }) || scope.categories[0]
  return child?.id || scope.root.id
}

export const serviceCategoryIds = (scope: ServiceScope): number[] => {
  if (!scope.root || !scope.selected) return []
  if (scope.selected.id !== scope.root.id) return [scope.selected.id]
  return [scope.root.id, ...scope.categories.map(row => row.id)]
}

/** Defensive response filtering also prevents stale/unscoped responses crossing the service tabs. */
export const filterServiceProducts = (rows: ServiceProduct[], scope: ServiceScope): ServiceProduct[] => {
  const allowed = new Set(serviceCategoryIds(scope))
  return rows.filter(row => allowed.has(categoryId(row.category_id)))
}

export const activeServiceSkus = (product: ServiceProduct): ServiceSku[] =>
  Array.isArray(product.skus)
    ? product.skus.filter(sku => categoryId(sku.id) && sku.is_active === true)
    : []

export interface ServiceSkuPrice {
  amount: string
  originalAmount: string | null
  kind: 'standard' | 'promotion' | 'member'
}

export const serviceSkuPrice = (sku: ServiceSku | null): ServiceSkuPrice | null => {
  if (!isKnownPurchasePrice(sku?.price_amount)) return null
  const base = amountToCents(sku?.price_amount)
  if (base === null || base < 0) return null
  let final = base
  let kind: ServiceSkuPrice['kind'] = 'standard'
  const promotion = isKnownPurchasePrice(sku?.promotion_price_amount) ? amountToCents(sku?.promotion_price_amount) : null
  if (promotion !== null && promotion >= 0 && promotion < final) {
    final = promotion
    kind = 'promotion'
  }
  const member = isKnownPurchasePrice(sku?.member_price_amount) ? amountToCents(sku?.member_price_amount) : null
  if (member !== null && member >= 0 && member < final) {
    final = member
    kind = 'member'
  }
  return {
    amount: centsToAmount(final),
    originalAmount: final < base ? centsToAmount(base) : null,
    kind,
  }
}

export type ServiceAvailability = 'available' | 'sold_out' | 'unknown'

export const serviceAvailability = (product: ServiceProduct, sku: ServiceSku | null): ServiceAvailability => {
  if (product.is_sold_out === true || product.stock_status === 'out_of_stock') return 'sold_out'
  if (!sku || sku.is_active !== true) return 'unknown'
  if (sku.is_sold_out === true || sku.stock_status === 'out_of_stock') return 'sold_out'
  if (!serviceSkuPrice(sku)) return 'unknown'
  return ['in_stock', 'low_stock', 'unlimited'].includes(String(sku.stock_status)) ? 'available' : 'unknown'
}

/** This value is rendered as text, never as HTML. */
export const serviceDescriptionText = (value: string): string => value
  .replace(/<[^>]*>/g, ' ')
  .replace(/&(?:nbsp|amp|lt|gt|quot|#39);/g, entity => ({
    '&nbsp;': ' ', '&amp;': '&', '&lt;': '<', '&gt;': '>', '&quot;': '"', '&#39;': "'",
  })[entity] || entity)
  .replace(/\s+/g, ' ')
  .trim()
