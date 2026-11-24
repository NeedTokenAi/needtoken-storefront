export type StartupPhase = 'pending' | 'ready' | 'error'

export const NEEDTOKEN_MAIN_BRAND_NAME = 'NeedToken.AI'
export const NEEDTOKEN_MAIN_SITE_ICON = '/needtoken-icon.svg'
export const NEEDTOKEN_MAIN_FALLBACK_ICON = '/needtoken.ico'
export const NEEDTOKEN_MAIN_TOUCH_ICON = '/needtoken-touch.png'

export interface StorefrontHeadBrandInput {
  brandOverride?: {
    siteName?: string | null
    siteIcon?: string | null
    touchIcon?: string | null
  } | null
  config?: {
    storefront_template?: unknown
    tenant?: { mode?: unknown } | null
    brand?: { site_icon?: unknown; site_name?: unknown } | null
  } | null
  templateHint?: unknown
  verifiedMainHint?: boolean
  hostname?: string
  resolveImageUrl?: (raw: string) => string
}

export interface StorefrontHeadBrand {
  siteIcon: string
  fallbackIcon?: string
  touchIcon?: string
  brandTitle?: string
}

/** Refreshing existing configuration must never hide an already usable page. */
export function storefrontStartupPhase(input: { hasConfig: boolean; routeReady: boolean; routeFailed: boolean }): StartupPhase {
  if (input.routeFailed) return 'error'
  if (input.hasConfig && input.routeReady) return 'ready'
  if (input.routeReady && !input.hasConfig) return 'error'
  return 'pending'
}

/** Only the main storefront or its previously verified main-site hint gets this brand. */
export function shouldBrandNeedtokenStartup(templateHint: unknown, verifiedMainHint = false, hostname = ''): boolean {
  const mainHost = hostname === 'needtoken.ai' || hostname === 'www.needtoken.ai'
  return mainHost || (verifiedMainHint && templateHint === 'needtoken')
}

/** Upstream Dujiao-Next default icon paths must never surface in the public storefront. */
export function isLegacyDefaultSiteIcon(icon: unknown): boolean {
  const raw = String(icon || '').trim()
  if (!raw) return false
  const withoutQuery = raw.split(/[?#]/, 1)[0] || ''
  return /(?:^|\/)dj\.svg$/i.test(withoutQuery)
}

/** Resolve favicon and brand title across boot, config load, and layout mount without flashing upstream defaults. */
export function resolveStorefrontHeadBrand(input: StorefrontHeadBrandInput): StorefrontHeadBrand {
  const resolveImage = input.resolveImageUrl ?? ((raw: string) => raw)
  const overrideIcon = String(input.brandOverride?.siteIcon || '').trim()
  const overrideName = String(input.brandOverride?.siteName || '').trim()
  const overrideTouch = String(input.brandOverride?.touchIcon || '').trim()

  if (input.brandOverride && overrideIcon && !isLegacyDefaultSiteIcon(overrideIcon)) {
    return {
      siteIcon: overrideIcon,
      fallbackIcon: overrideIcon === NEEDTOKEN_MAIN_SITE_ICON ? NEEDTOKEN_MAIN_FALLBACK_ICON : undefined,
      touchIcon: overrideTouch || undefined,
      brandTitle: overrideName || undefined,
    }
  }

  if (input.config) {
    const isReseller = String(input.config.tenant?.mode || '').trim().toLowerCase() === 'reseller'
    const isMainNeedtoken = !isReseller && input.config.storefront_template === 'needtoken'
    if (isMainNeedtoken) {
      return {
        siteIcon: NEEDTOKEN_MAIN_SITE_ICON,
        fallbackIcon: NEEDTOKEN_MAIN_FALLBACK_ICON,
        touchIcon: NEEDTOKEN_MAIN_TOUCH_ICON,
        brandTitle: NEEDTOKEN_MAIN_BRAND_NAME,
      }
    }
    const rawIcon = String(input.config.brand?.site_icon || '').trim()
    const siteIcon = rawIcon && !isLegacyDefaultSiteIcon(rawIcon) ? resolveImage(rawIcon) : ''
    return { siteIcon }
  }

  if (shouldBrandNeedtokenStartup(input.templateHint, input.verifiedMainHint, input.hostname)) {
    return {
      siteIcon: NEEDTOKEN_MAIN_SITE_ICON,
      fallbackIcon: NEEDTOKEN_MAIN_FALLBACK_ICON,
      touchIcon: NEEDTOKEN_MAIN_TOUCH_ICON,
      brandTitle: NEEDTOKEN_MAIN_BRAND_NAME,
    }
  }

  return { siteIcon: '' }
}

/** Resolve display site name without falling back to upstream brand strings. */
export function resolveFallbackSiteName(
  siteName: unknown,
  options: {
    brandOverrideName?: string | null
    fallbackLabel?: string
    hostname?: string
    isReseller?: boolean
  } = {},
): string {
  const overrideName = String(options.brandOverrideName || '').trim()
  if (overrideName) return overrideName
  const configured = String(siteName || '').trim()
  if (configured) return configured
  const host = String(options.hostname || '').trim()
  if (!options.isReseller && (host === 'needtoken.ai' || host === 'www.needtoken.ai')) {
    return NEEDTOKEN_MAIN_BRAND_NAME
  }
  const fallback = String(options.fallbackLabel || '').trim()
  if (fallback) return fallback
  return host
}
