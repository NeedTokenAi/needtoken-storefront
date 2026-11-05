import type { PublicCategory } from '../../../utils/category.ts'
import type { ServiceProduct } from './serviceCatalog.ts'

export type APIPlanAudience = 'personal' | 'enterprise'
export interface APIPlanScope {
  root: PublicCategory | null
  category: PublicCategory | null
  categoryIds: number[]
}

const validId = (value: unknown): number => {
  const id = Number(value)
  return Number.isSafeInteger(id) && id > 0 ? id : 0
}

export const apiPlanAudience = (value: unknown): APIPlanAudience => value === 'enterprise' ? 'enterprise' : 'personal'

/** API plan categories are explicitly assigned, never guessed from product titles. */
export function resolveAPIPlanScope(categories: PublicCategory[], audience: APIPlanAudience, rootSlug = 'plans'): APIPlanScope {
  const root = categories.find(row => row.slug === rootSlug && !validId(row.parent_id)) || null
  if (!root) return { root: null, category: null, categoryIds: [] }
  const slugs = [`${rootSlug}-${audience}`, `api-${audience}`, audience]
  const category = slugs.map(slug => categories.find(row => row.slug === slug && row.parent_id === root.id)).find(Boolean) || null
  if (!category) return { root, category: null, categoryIds: [] }
  const ids = new Set([category.id])
  let changed = true
  while (changed) {
    changed = false
    for (const row of categories) {
      if (validId(row.id) && validId(row.parent_id) && ids.has(Number(row.parent_id)) && !ids.has(row.id)) {
        ids.add(row.id)
        changed = true
      }
    }
  }
  return { root, category, categoryIds: [...ids] }
}

export function filterAPIPlanProducts(products: ServiceProduct[], scope: APIPlanScope): ServiceProduct[] {
  const ids = new Set(scope.categoryIds)
  return products.filter(product => ids.has(validId(product.category_id)))
}

/** The return path is always internal and keeps the selected real SKU. */
export function apiPlanMemberLoginPath(slug: string, skuId: number): string {
  const path = `/products/${encodeURIComponent(slug)}?sku=${encodeURIComponent(String(skuId))}`
  return `/auth/login?redirect=${encodeURIComponent(path)}`
}
