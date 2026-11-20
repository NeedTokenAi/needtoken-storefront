export interface PublicWalletConfigLike {
  wallet_only_payment?: boolean
  wallet_recharge_channel_ids?: unknown
}

export function hasWalletRechargeConfig(config: PublicWalletConfigLike | null | undefined): boolean {
  if (!config || typeof config !== 'object') return false
  if (config.wallet_only_payment === true) return true
  return Array.isArray(config.wallet_recharge_channel_ids) && config.wallet_recharge_channel_ids.length > 0
}

export function shouldShowWalletSidebarTab(
  activeTemplate: string | null | undefined,
  config: PublicWalletConfigLike | null | undefined,
): boolean {
  if (String(activeTemplate || '').trim().toLowerCase() !== 'needtoken') return true
  return hasWalletRechargeConfig(config)
}

export function shouldShowCheckoutWalletBox(
  config: PublicWalletConfigLike | null | undefined,
  walletBalance: string | number | null | undefined,
): boolean {
  if (hasWalletRechargeConfig(config)) return true
  const raw = String(walletBalance ?? '').trim()
  if (!raw) return false
  const parsed = Number(raw)
  return Number.isFinite(parsed) && parsed > 0
}
