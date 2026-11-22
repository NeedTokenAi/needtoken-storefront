import { API_QUOTA_PER_USD, computeSubscriptionRemainingQuota } from './apiAccountLinkPolicy.ts'

export type NeedtokenMemberSection = 'overview' | 'keys' | 'usage' | 'orders' | 'assets' | 'affiliate' | 'security'

export function normalizeNeedtokenMemberSection(raw?: string | null): NeedtokenMemberSection {
  switch (String(raw || '').trim()) {
    case 'keys':
    case 'api':
      return 'keys'
    case 'usage':
      return 'usage'
    case 'orders':
      return 'orders'
    case 'assets':
    case 'wallet':
    case 'giftCard':
      return 'assets'
    case 'affiliate':
    case 'reseller':
      return 'affiliate'
    case 'security':
    case 'profile':
      return 'security'
    default:
      return 'overview'
  }
}

export function usdToRawQuota(usd: number | string): number {
  const num = typeof usd === 'number' ? usd : Number(String(usd).trim())
  if (!Number.isFinite(num) || num <= 0) return 0
  return Math.round(num * API_QUOTA_PER_USD)
}

export function rawQuotaToUsdNumber(rawQuota?: number | null): number {
  const value = typeof rawQuota === 'number' && Number.isFinite(rawQuota) ? rawQuota : 0
  return Math.max(0, value) / API_QUOTA_PER_USD
}

export function sumSubscriptionRemainingQuota(subs?: Array<{ status?: string; amount_total?: number; amount_used?: number }> | null): number {
  if (!Array.isArray(subs) || subs.length === 0) return 0
  let total = 0
  for (const sub of subs) {
    if (sub?.status && String(sub.status).toLowerCase() !== 'active') continue
    total += computeSubscriptionRemainingQuota(sub?.amount_total, sub?.amount_used)
  }
  return total
}

export function computeCombinedApiQuota(
  walletQuota?: number | null,
  subs?: Array<{ status?: string; amount_total?: number; amount_used?: number }> | null,
): number {
  const base = typeof walletQuota === 'number' && Number.isFinite(walletQuota) ? Math.max(0, walletQuota) : 0
  return base + sumSubscriptionRemainingQuota(subs)
}
