export interface HelpCategory {
  id: number
  parent_id: number
  slug: string
  name: Record<string, unknown>
  sort_order?: number
}

export interface PublicAppleResource {
  id: string
  title: Record<string, unknown>
  description: Record<string, unknown>
  region: string
  url: string
  updated_at: string
}

/** Missing FAQ configuration must never turn into an unscoped blog query. */
export function faqCategories(value: unknown): { root?: HelpCategory; children: HelpCategory[] } {
  if (!Array.isArray(value)) return { children: [] }
  const categories = value.filter((item): item is HelpCategory => !!item && Number.isInteger(item.id) && item.id > 0 && typeof item.slug === 'string')
  const root = categories.find((item) => item.slug === 'faq' && !item.parent_id)
  if (!root) return { children: [] }
  return {
    root,
    children: categories.filter((item) => item.parent_id === root.id)
      .sort((left, right) => (left.sort_order || 0) - (right.sort_order || 0) || left.id - right.id),
  }
}

export function publicResourceUrl(value: unknown): string {
  if (typeof value !== 'string') return ''
  try {
    const url = new URL(value.trim())
    if (!['https:', 'http:'].includes(url.protocol) || url.username || url.password) return ''
    return url.href
  } catch {
    return ''
  }
}

/** Public resources are an explicit publication list, separate from the private account pool. */
export function publicAppleResources(value: unknown): PublicAppleResource[] {
  if (!Array.isArray(value)) return []
  const seen = new Set<string>()
  const result: PublicAppleResource[] = []
  for (const item of value) {
    if (!item || item.enabled !== true || typeof item.id !== 'string' || !item.id.trim()) continue
    const url = publicResourceUrl(item.url)
    if (!url || seen.has(item.id)) continue
    seen.add(item.id)
    result.push({
      id: item.id,
      title: item.title && typeof item.title === 'object' ? item.title : {},
      description: item.description && typeof item.description === 'object' ? item.description : {},
      region: typeof item.region === 'string' ? item.region : '',
      url,
      updated_at: typeof item.updated_at === 'string' ? item.updated_at : '',
    })
  }
  return result
}
