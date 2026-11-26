import { API_QUOTA_PER_USD, computeSubscriptionRemainingQuota } from './apiAccountLinkPolicy.ts'

export type NeedtokenMemberSection = 'overview' | 'keys' | 'models' | 'playground' | 'usage' | 'plans' | 'orders' | 'assets' | 'affiliate' | 'reseller' | 'security'

export type ResellerWorkspaceMainTab = 'dashboard' | 'products' | 'orders' | 'finance' | 'site'
export type ResellerFinanceSubTab = 'finance' | 'ledger' | 'withdraws'
export type ResellerSiteSubTab = 'domains' | 'site' | 'apply'

export interface ResolvedResellerView {
  mainTab: ResellerWorkspaceMainTab
  financeSubTab: ResellerFinanceSubTab
  siteSubTab: ResellerSiteSubTab
  isOrderDetail: boolean
}

export function normalizeNeedtokenMemberSection(raw?: string | null): NeedtokenMemberSection {
  switch (String(raw || '').trim()) {
    case 'keys':
    case 'api':
      return 'keys'
    case 'models':
      return 'models'
    case 'playground':
      return 'playground'
    case 'usage':
      return 'usage'
    case 'plans':
    case 'subscriptions':
      return 'plans'
    case 'orders':
      return 'orders'
    case 'assets':
    case 'wallet':
    case 'giftCard':
      return 'assets'
    case 'affiliate':
      return 'affiliate'
    case 'reseller':
      return 'reseller'
    case 'security':
    case 'profile':
      return 'security'
    default:
      return 'overview'
  }
}

export function mapLegacyResellerPath(rawPath: string): string {
  const clean = String(rawPath || '').trim()
  if (clean === '/reseller' || clean === '/reseller/') {
    return '/me/reseller'
  }
  if (clean.startsWith('/reseller/')) {
    return `/me${clean}`
  }
  return '/me/reseller'
}

export function resolveResellerViewFromPath(rawPath: string): ResolvedResellerView {
  const normalized = String(rawPath || '').replace(/\/+$/, '')
  if (normalized.startsWith('/me/reseller/orders/')) {
    return { mainTab: 'orders', financeSubTab: 'finance', siteSubTab: 'domains', isOrderDetail: true }
  }
  const suffix = normalized.startsWith('/me/reseller/')
    ? normalized.slice('/me/reseller/'.length)
    : ''
  switch (suffix) {
    case 'products':
      return { mainTab: 'products', financeSubTab: 'finance', siteSubTab: 'domains', isOrderDetail: false }
    case 'orders':
      return { mainTab: 'orders', financeSubTab: 'finance', siteSubTab: 'domains', isOrderDetail: false }
    case 'finance':
      return { mainTab: 'finance', financeSubTab: 'finance', siteSubTab: 'domains', isOrderDetail: false }
    case 'ledger':
      return { mainTab: 'finance', financeSubTab: 'ledger', siteSubTab: 'domains', isOrderDetail: false }
    case 'withdraws':
      return { mainTab: 'finance', financeSubTab: 'withdraws', siteSubTab: 'domains', isOrderDetail: false }
    case 'domains':
      return { mainTab: 'site', financeSubTab: 'finance', siteSubTab: 'domains', isOrderDetail: false }
    case 'site':
      return { mainTab: 'site', financeSubTab: 'finance', siteSubTab: 'site', isOrderDetail: false }
    case 'apply':
      return { mainTab: 'site', financeSubTab: 'finance', siteSubTab: 'apply', isOrderDetail: false }
    default:
      return { mainTab: 'dashboard', financeSubTab: 'finance', siteSubTab: 'domains', isOrderDetail: false }
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

export function isNeedtokenMemberConsolePath(rawPath?: string | null): boolean {
  const clean = String(rawPath || '').trim()
  return clean === '/me' || clean.startsWith('/me/')
}

export type WalletRecordCategoryFilter = 'all' | 'recharge' | 'gift_card' | 'spend'

export function matchesWalletRecordCategory(type?: string | null, filter: WalletRecordCategoryFilter = 'all'): boolean {
  if (filter === 'all') return true
  const norm = String(type || '').trim().toLowerCase()
  if (filter === 'gift_card') return norm === 'gift_card_redeem'
  if (filter === 'recharge') return norm === 'recharge' || norm === 'admin_adjust' || norm === 'order_underpaid_credit'
  if (filter === 'spend') return norm === 'order_pay' || norm === 'order_refund' || norm === 'admin_refund'
  return true
}

export function resolveNeedtokenRouteTransitionKey(rawPath?: string | null): string {
  const clean = String(rawPath || '').trim()
  if (isNeedtokenMemberConsolePath(clean)) {
    return '/me'
  }
  return clean
}
