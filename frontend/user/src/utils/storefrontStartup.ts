export type StartupPhase = 'pending' | 'ready' | 'error'

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
