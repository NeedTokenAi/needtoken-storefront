import { amountToCents } from '../../../utils/money.ts'
import { getLocalizedText } from '../../../utils/resellerSiteConfig.ts'
import {
  activeServiceSkus,
  serviceAvailability,
  serviceDescriptionText,
  serviceSkuPrice,
  type ServiceProduct,
  type ServiceSku,
} from './serviceCatalog.ts'
import { serviceSkuPresentation } from './serviceVariants.ts'

export type ProductBrand = 'gpt' | 'claude' | 'unknown'
export type ProductPlan = 'plus' | '5x' | '20x' | '50x' | 'neutral'

/** Share the exact displayed/linked SKU with the price sorter. Backend SKU order
 * stays authoritative; sorting cards must not choose another purchase option. */
export function homeCardSku(product: ServiceProduct): ServiceSku | null {
  const skus = activeServiceSkus(product)
  return skus.find(sku => serviceAvailability(product, sku) === 'available') || skus[0] || null
}

/** Sort only the supplied page, by the final native amount shown on each card.
 * Equal and unknown prices retain the server's order. This is not a global
 * catalogue price query: the public products endpoint has no price-sort option. */
export function sortHomeProductsByPrice(products: readonly ServiceProduct[]): ServiceProduct[] {
  return products.map((product, index) => {
    const price = serviceSkuPrice(homeCardSku(product))
    return { product, index, cents: price ? amountToCents(price.amount) : null }
  }).sort((a, b) => {
    if (a.cents === null || b.cents === null) {
      return a.cents === b.cents ? a.index - b.index : a.cents === null ? 1 : -1
    }
    return a.cents - b.cents || a.index - b.index
  }).map(row => row.product)
}

const categoryBrands: Readonly<Record<string, ProductBrand>> = {
  chatgpt: 'gpt', 'chatgpt-recharge': 'gpt', 'chatgpt-account': 'gpt', 'chatgpt-accounts': 'gpt',
  claude: 'claude', 'claude-recharge': 'claude', 'claude-account': 'claude', 'claude-accounts': 'claude',
}

const titleBrand = (title: string): ProductBrand | null => {
  const gpt = /(?:^|[^a-z0-9])(?:chatgpt|openai|gpt(?:[- ]?\d+(?:\.\d+)?)?)(?=$|[^a-z0-9])/i.test(title)
  const claude = /(?:^|[^a-z0-9])(?:claude|anthropic)(?=$|[^a-z0-9])/i.test(title)
  return gpt && claude ? 'unknown' : gpt ? 'gpt' : claude ? 'claude' : null
}

/** Only existing SKU display fields, exact public category slugs and explicit
 * brand names are used. Price, product IDs and image filenames imply no brand. */
export function homeCardBrand(product: ServiceProduct, sku: ServiceSku | null, locale: string): ProductBrand {
  const meta = serviceSkuPresentation(sku, locale)
  const explicit = titleBrand(meta.title) || titleBrand(meta.label)
  if (explicit) return explicit
  const category = String(product.category?.slug || '').trim().toLowerCase()
  if (Object.prototype.hasOwnProperty.call(categoryBrands, category)) return categoryBrands[category]!
  const nativeTitle = typeof product.title === 'string'
    ? product.title : getLocalizedText(product.title as Record<string, unknown> | null, locale)
  return titleBrand(serviceDescriptionText(nativeTitle)) || 'unknown'
}

export interface HomeCardTitle {
  before: string
  tier: string
  after: string
  plan: ProductPlan
}

/** Keep every backend character. Only explicit multiplier tokens get a color;
 * e.g. a price of 500 or an opaque plan code never becomes a 50x label. */
export function homeCardTitle(title: string): HomeCardTitle {
  const match = /(^|\bPro\s*|[\s·/（(])(\d+(?:\.\d+)?\s*[x×])(?=$|[^a-z0-9])/i.exec(title)
  if (match) {
    const start = match.index + match[1]!.length
    const tier = match[2]!
    const normalized = tier.replace(/\s/g, '').replace('×', 'x').toLowerCase()
    const plan = ['5x', '20x', '50x'].includes(normalized) ? normalized as ProductPlan : 'neutral'
    return { before: title.slice(0, start), tier, after: title.slice(start + tier.length), plan }
  }
  return { before: title, tier: '', after: '', plan: /(?:^|[^a-z0-9])plus(?=$|[^a-z0-9])/i.test(title) ? 'plus' : 'neutral' }
}

/** Metadata label can identify an illustration when the displayed title is
 * generic. No unsupported display_meta fields or numeric allowance guesses. */
export function homeCardPlan(title: string, sku: ServiceSku | null, locale: string): ProductPlan {
  const meta = serviceSkuPresentation(sku, locale)
  for (const candidate of [meta.title, meta.label, title]) {
    const plan = homeCardTitle(candidate).plan
    if (plan !== 'neutral') return plan
  }
  return 'neutral'
}

/** Editors accept '/月' as well as '月'; render one separator in either case. */
export function homeCardPriceSuffix(suffix: string): string {
  return suffix.trim().replace(/^[\s/／]+/, '').trim()
}
