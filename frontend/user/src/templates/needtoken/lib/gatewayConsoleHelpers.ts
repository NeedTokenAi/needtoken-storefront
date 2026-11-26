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

export interface GroupMetaLike {
  code?: string
  description?: string
  channel_names?: string[]
  display_name?: string
  ratio?: number
  enabled_channel_count?: number
  models?: string[]
}

export interface SelectableGroupOption {
  code: string
  channelName: string
  description: string
  ratio: number
  enabledChannelCount: number
  hasActiveChannels: boolean
  models: string[]
  label: string
}

export type ConsoleModelVendorKey = 'openai' | 'claude' | 'gemini' | 'grok' | 'other'

export function classifyConsoleModelVendor(modelId: string): ConsoleModelVendorKey {
  const m = String(modelId || '').toLowerCase()
  if (m.startsWith('claude') || m.includes('anthropic')) return 'claude'
  if (m.startsWith('gemini') || m.startsWith('imagen') || m.startsWith('veo') || m.startsWith('gemma')) return 'gemini'
  if (m.startsWith('grok') || m.startsWith('xai')) return 'grok'
  if (
    m.startsWith('gpt')
    || m.startsWith('o1')
    || m.startsWith('o3')
    || m.startsWith('o4')
    || m.startsWith('codex')
    || m.startsWith('chatgpt')
    || m.startsWith('text-embedding')
    || m.startsWith('dall-e')
    || m.startsWith('whisper')
    || m.startsWith('tts')
  ) {
    return 'openai'
  }
  return 'other'
}

export function formatGroupRatioText(ratio?: number | null): string {
  const r = typeof ratio === 'number' && Number.isFinite(ratio) && ratio > 0 ? ratio : 1
  return `×${r.toFixed(2)}`
}

function isGenericGroupLabel(code: string, raw?: string | null): boolean {
  const trimmed = String(raw || '').trim()
  if (!trimmed || trimmed.toLowerCase() === code.trim().toLowerCase()) return true
  const low = trimmed.toLowerCase()
  return low === 'default group' || low === 'vip group' || low === 'user group' || low === 'auto group'
}

/**
 * Formats any member-facing group consistently as:
 * "渠道名称 - 分组代号 - ×倍率" (falling back cleanly when channel name or custom label is absent).
 */
export function formatConsoleGroupLabel(
  codeRaw?: string | null,
  groupMeta?: Record<string, GroupMetaLike> | null,
  usableGroups?: Record<string, string> | null,
  groupRatio?: Record<string, number> | null,
): string {
  const code = String(codeRaw || '').trim()
  if (!code) return ''
  if (code === 'auto') return 'auto'
  const meta = groupMeta?.[code]
  const rawDesc = String(meta?.description || usableGroups?.[code] || '').trim()
  const primaryChannel = String(meta?.display_name || meta?.channel_names?.[0] || '').trim()
  const lead = primaryChannel && !isGenericGroupLabel(code, primaryChannel)
    ? primaryChannel
    : (!isGenericGroupLabel(code, rawDesc) ? rawDesc : '')
  const ratioVal = typeof meta?.ratio === 'number' && Number.isFinite(meta.ratio)
    ? meta.ratio
    : (typeof groupRatio?.[code] === 'number' && Number.isFinite(groupRatio[code]) ? groupRatio[code] : 1)
  const ratioStr = formatGroupRatioText(ratioVal)
  return lead ? `${lead} - ${code} - ${ratioStr}` : `${code} - ${ratioStr}`
}

export function formatTokenGroupsSummary(
  token: { group?: string | null; auto_groups?: string[] | null },
  defaultGroup: string,
  groupMeta?: Record<string, GroupMetaLike> | null,
  usableGroups?: Record<string, string> | null,
  groupRatio?: Record<string, number> | null,
): string {
  const grp = String(token.group || '').trim()
  if (grp === 'auto') {
    const subs = Array.isArray(token.auto_groups) ? token.auto_groups.map(s => String(s || '').trim()).filter(Boolean) : []
    if (subs.length > 0) {
      return subs.map(g => formatConsoleGroupLabel(g, groupMeta, usableGroups, groupRatio)).join(' / ')
    }
    return 'auto'
  }
  const effective = grp || defaultGroup || 'default'
  return formatConsoleGroupLabel(effective, groupMeta, usableGroups, groupRatio)
}

export function buildSelectableGroupOptions(params: {
  usableGroups?: Record<string, string> | null
  groupMeta?: Record<string, GroupMetaLike> | null
  groupRatio?: Record<string, number> | null
  pricingModels?: Array<{ model_name?: string; enable_groups?: string[] }> | null
  defaultGroup?: string
  includeCodes?: string[]
}): SelectableGroupOption[] {
  const usable = params.usableGroups || {}
  const metaMap = params.groupMeta || {}
  const ratioMap = params.groupRatio || {}
  const includeSet = new Set((params.includeCodes || []).map(s => String(s || '').trim()).filter(Boolean))

  const modelCountByGroup = new Map<string, Set<string>>()
  for (const item of params.pricingModels || []) {
    const mName = String(item?.model_name || '').trim()
    if (!mName) continue
    for (const rawG of item?.enable_groups || []) {
      const g = String(rawG || '').trim()
      if (!g) continue
      if (!modelCountByGroup.has(g)) modelCountByGroup.set(g, new Set())
      modelCountByGroup.get(g)!.add(mName)
    }
  }

  const allCodes = new Set<string>()
  for (const k of Object.keys(usable)) {
    if (k && k !== 'auto') allCodes.add(k)
  }
  for (const k of Object.keys(metaMap)) {
    if (k && k !== 'auto') allCodes.add(k)
  }
  for (const k of includeSet) {
    if (k && k !== 'auto') allCodes.add(k)
  }
  if (!allCodes.size && params.defaultGroup && params.defaultGroup !== 'auto') {
    allCodes.add(params.defaultGroup)
  }

  const options: SelectableGroupOption[] = []
  for (const code of allCodes) {
    const meta = metaMap[code]
    const desc = String(meta?.description || usable[code] || '').trim()
    const channelName = String(meta?.display_name || meta?.channel_names?.[0] || '').trim()
    const ratio = typeof meta?.ratio === 'number' && Number.isFinite(meta.ratio)
      ? meta.ratio
      : (typeof ratioMap[code] === 'number' && Number.isFinite(ratioMap[code]) ? ratioMap[code] : 1)
    const modelSet = new Set<string>()
    for (const m of meta?.models || []) {
      if (String(m).trim()) modelSet.add(String(m).trim())
    }
    for (const m of modelCountByGroup.get(code) || []) {
      modelSet.add(m)
    }
    for (const m of modelCountByGroup.get('all') || []) {
      modelSet.add(m)
    }
    const models = [...modelSet].sort()
    const hasMetaCount = typeof meta?.enabled_channel_count === 'number'
    const enabledChannelCount = hasMetaCount ? meta!.enabled_channel_count! : (models.length > 0 ? 1 : 0)
    const hasActiveChannels = hasMetaCount ? enabledChannelCount > 0 : true
    options.push({
      code,
      channelName: !isGenericGroupLabel(code, channelName) ? channelName : (!isGenericGroupLabel(code, desc) ? desc : code),
      description: desc,
      ratio,
      enabledChannelCount,
      hasActiveChannels,
      models,
      label: formatConsoleGroupLabel(code, metaMap, usable, ratioMap),
    })
  }

  // If the user has at least one group backed by an active channel, hide ghost groups
  // that have zero enabled channels unless explicitly bound on the token being edited.
  const hasAnyActive = options.some(o => o.hasActiveChannels)
  const filtered = hasAnyActive
    ? options.filter(o => o.hasActiveChannels || includeSet.has(o.code))
    : options

  filtered.sort((a, b) => {
    if (a.hasActiveChannels !== b.hasActiveChannels) return a.hasActiveChannels ? -1 : 1
    if (a.code === params.defaultGroup && b.code !== params.defaultGroup) return -1
    if (b.code === params.defaultGroup && a.code !== params.defaultGroup) return 1
    return a.label.localeCompare(b.label)
  })
  return filtered
}

export function collectModelsForConsoleGroups(
  groupCodes: string[],
  groupMeta?: Record<string, GroupMetaLike> | null,
  pricingModels?: Array<{ model_name?: string; enable_groups?: string[] }> | null,
): Array<{ model: string; vendor: ConsoleModelVendorKey; groups: string[] }> {
  const selected = groupCodes.map(s => String(s || '').trim()).filter(g => g && g !== 'auto')
  const selectedSet = new Set(selected)
  const map = new Map<string, Set<string>>()

  for (const g of selected) {
    const metaModels = groupMeta?.[g]?.models || []
    for (const rawM of metaModels) {
      const m = String(rawM || '').trim()
      if (!m) continue
      if (!map.has(m)) map.set(m, new Set())
      map.get(m)!.add(g)
    }
  }

  for (const item of pricingModels || []) {
    const m = String(item?.model_name || '').trim()
    if (!m) continue
    const egs = Array.isArray(item?.enable_groups) ? item.enable_groups : []
    if (selectedSet.size === 0) {
      if (!map.has(m)) map.set(m, new Set())
      for (const g of egs) map.get(m)!.add(g)
      continue
    }
    if (egs.includes('all')) {
      if (!map.has(m)) map.set(m, new Set())
      for (const g of selected) map.get(m)!.add(g)
      continue
    }
    for (const g of egs) {
      if (selectedSet.has(g)) {
        if (!map.has(m)) map.set(m, new Set())
        map.get(m)!.add(g)
      }
    }
  }

  return [...map.entries()]
    .map(([model, grps]) => ({
      model,
      vendor: classifyConsoleModelVendor(model),
      groups: [...grps],
    }))
    .sort((a, b) => a.model.localeCompare(b.model))
}

export function resolveTokenAvailableModels(params: {
  token?: { group?: string | null; auto_groups?: string[] | null; model_limits_enabled?: boolean; model_limits?: string | null } | null
  defaultGroup?: string
  userAutoGroups?: string[]
  groupMeta?: Record<string, GroupMetaLike> | null
  pricingModels?: Array<{ model_name?: string; enable_groups?: string[] }> | null
  fallbackModels?: string[]
}): string[] {
  const tok = params.token
  const fallback = params.fallbackModels || []
  if (!tok) {
    const all = collectModelsForConsoleGroups([], params.groupMeta, params.pricingModels).map(i => i.model)
    return all.length ? all : fallback
  }

  let targetGroups: string[] = []
  const g = String(tok.group || '').trim()
  if (g === 'auto') {
    if (Array.isArray(tok.auto_groups) && tok.auto_groups.length > 0) {
      targetGroups = tok.auto_groups.map(s => String(s || '').trim()).filter(Boolean)
    } else if (Array.isArray(params.userAutoGroups) && params.userAutoGroups.length > 0) {
      targetGroups = params.userAutoGroups.filter(Boolean)
    }
  } else {
    const single = g || params.defaultGroup || 'default'
    targetGroups = [single]
  }

  const groupModels = collectModelsForConsoleGroups(targetGroups, params.groupMeta, params.pricingModels).map(i => i.model)

  if (tok.model_limits_enabled) {
    const allowed = String(tok.model_limits || '')
      .split(/[\n,]+/)
      .map(s => s.trim())
      .filter(Boolean)
    if (!allowed.length) return []
    if (!groupModels.length) return [...new Set(allowed)]
    const groupSet = new Set(groupModels)
    const intersected = allowed.filter(m => groupSet.has(m))
    return intersected.length ? [...new Set(intersected)] : [...new Set(allowed)]
  }

  if (groupModels.length) return groupModels
  const all = collectModelsForConsoleGroups([], params.groupMeta, params.pricingModels).map(i => i.model)
  return all.length ? all : fallback
}

export interface QuickToolRenderVars {
  apiKey: string
  baseUrl: string
  baseUrlV1: string
  model: string
  tokenName: string
  group: string
}

export function renderQuickToolTemplate(template: string, vars: QuickToolRenderVars, encodeForUrl = false): string {
  if (!template) return ''
  const v = (raw: string) => (encodeForUrl ? encodeURIComponent(raw) : raw)
  return template
    .replace(/\{\{API_KEY\}\}/g, v(vars.apiKey))
    .replace(/\{\{BASE_URL_V1\}\}/g, v(vars.baseUrlV1))
    .replace(/\{\{BASE_URL\}\}/g, v(vars.baseUrl))
    .replace(/\{\{MODEL\}\}/g, v(vars.model))
    .replace(/\{\{TOKEN_NAME\}\}/g, v(vars.tokenName))
    .replace(/\{\{GROUP\}\}/g, v(vars.group))
}
