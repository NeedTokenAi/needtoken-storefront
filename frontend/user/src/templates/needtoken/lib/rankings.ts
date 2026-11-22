/**
 * API 消耗排行榜的数据规范化、格式化与模型定价联动（纯函数，便于单测）。
 *
 * 数据源是 new-api 的 GET /api/rankings?period=today|week|month|year
 * （经店面同源缓存 /api/v1/public/api-rankings 代理）。
 */

import type { ModelOffer } from './pricing'

export type RankingPeriod = 'today' | 'week' | 'month' | 'year'

export const RANKING_PERIODS: readonly RankingPeriod[] = ['today', 'week', 'month', 'year'] as const

export interface RankedModel {
  rank: number
  previous_rank?: number
  model_name: string
  vendor: string
  vendor_icon?: string
  category?: string
  total_tokens: number
  share: number
  growth_pct: number
}

export interface RankedVendor {
  rank: number
  vendor: string
  vendor_icon?: string
  total_tokens: number
  share: number
  growth_pct: number
  models_count: number
  top_model: string
}

export interface RankingMover {
  model_name: string
  vendor: string
  vendor_icon?: string
  rank_delta: number
  current_rank: number
  growth_pct: number
}

export interface ModelHistoryPoint {
  ts: string
  label: string
  model: string
  vendor: string
  tokens: number
}

export interface ModelHistorySeries {
  points: ModelHistoryPoint[]
  models: Array<{ name: string; vendor: string; total: number }>
  buckets: number
}

export interface VendorSharePoint {
  ts: string
  label: string
  vendor: string
  share: number
  tokens: number
}

export interface VendorShareSeries {
  points: VendorSharePoint[]
  vendors: Array<{ name: string; total: number; share: number }>
  buckets: number
}

export interface RankingsSnapshot {
  models: RankedModel[]
  vendors: RankedVendor[]
  top_movers: RankingMover[]
  top_droppers: RankingMover[]
  models_history: ModelHistorySeries
  vendor_share_history: VendorShareSeries
}

export interface RankingsApiResponse {
  success?: boolean
  data?: Partial<RankingsSnapshot>
  error_code?: string
  message?: string
}

export type RankChange =
  | { kind: 'new'; delta: 0 }
  | { kind: 'up'; delta: number }
  | { kind: 'down'; delta: number }
  | { kind: 'same'; delta: 0 }

export const EMPTY_RANKINGS_SNAPSHOT: RankingsSnapshot = {
  models: [],
  vendors: [],
  top_movers: [],
  top_droppers: [],
  models_history: { points: [], models: [], buckets: 0 },
  vendor_share_history: { points: [], vendors: [], buckets: 0 },
}

export function normalizeRankingPeriod(raw?: unknown): RankingPeriod {
  const value = typeof raw === 'string' ? raw.trim().toLowerCase() : ''
  if (value === 'today' || value === 'week' || value === 'month' || value === 'year') {
    return value
  }
  return 'week'
}

const finiteNumber = (value: unknown, fallback = 0): number => {
  const num = typeof value === 'number' ? value : Number(value)
  return Number.isFinite(num) ? num : fallback
}

const optionalPositiveInt = (value: unknown): number | undefined => {
  if (value === undefined || value === null || value === '') return undefined
  const num = Number(value)
  if (!Number.isFinite(num) || num <= 0) return undefined
  return Math.trunc(num)
}

export function normalizeRankingsResponse(body: unknown): RankingsSnapshot {
  if (!body || typeof body !== 'object') {
    return { ...EMPTY_RANKINGS_SNAPSHOT }
  }
  const envelope = body as RankingsApiResponse
  const raw = envelope.data && typeof envelope.data === 'object' ? envelope.data : (body as Partial<RankingsSnapshot>)

  const models: RankedModel[] = Array.isArray(raw.models)
    ? raw.models
        .filter((item): item is RankedModel => Boolean(item && typeof item === 'object' && typeof item.model_name === 'string' && item.model_name.trim()))
        .map((item, idx) => ({
          rank: optionalPositiveInt(item.rank) ?? idx + 1,
          previous_rank: optionalPositiveInt(item.previous_rank),
          model_name: item.model_name.trim(),
          vendor: String(item.vendor || 'Unknown').trim() || 'Unknown',
          vendor_icon: typeof item.vendor_icon === 'string' && item.vendor_icon.trim() ? item.vendor_icon.trim() : undefined,
          category: typeof item.category === 'string' ? item.category.trim() : undefined,
          total_tokens: Math.max(0, finiteNumber(item.total_tokens, 0)),
          share: Math.max(0, Math.min(1, finiteNumber(item.share, 0))),
          growth_pct: finiteNumber(item.growth_pct, 0),
        }))
        .slice(0, 20)
    : []

  const vendors: RankedVendor[] = Array.isArray(raw.vendors)
    ? raw.vendors
        .filter((item): item is RankedVendor => Boolean(item && typeof item === 'object' && typeof item.vendor === 'string' && item.vendor.trim()))
        .map((item, idx) => ({
          rank: optionalPositiveInt(item.rank) ?? idx + 1,
          vendor: item.vendor.trim(),
          vendor_icon: typeof item.vendor_icon === 'string' && item.vendor_icon.trim() ? item.vendor_icon.trim() : undefined,
          total_tokens: Math.max(0, finiteNumber(item.total_tokens, 0)),
          share: Math.max(0, Math.min(1, finiteNumber(item.share, 0))),
          growth_pct: finiteNumber(item.growth_pct, 0),
          models_count: Math.max(0, Math.trunc(finiteNumber(item.models_count, 0))),
          top_model: String(item.top_model || '').trim(),
        }))
    : []

  const normalizeMovers = (items: unknown): RankingMover[] =>
    Array.isArray(items)
      ? items
          .filter((item): item is RankingMover => Boolean(item && typeof item === 'object' && typeof item.model_name === 'string' && item.model_name.trim()))
          .map(item => ({
            model_name: item.model_name.trim(),
            vendor: String(item.vendor || 'Unknown').trim() || 'Unknown',
            vendor_icon: typeof item.vendor_icon === 'string' && item.vendor_icon.trim() ? item.vendor_icon.trim() : undefined,
            rank_delta: Math.trunc(finiteNumber(item.rank_delta, 0)),
            current_rank: optionalPositiveInt(item.current_rank) ?? 0,
            growth_pct: finiteNumber(item.growth_pct, 0),
          }))
      : []

  const modelsHistory = raw.models_history && typeof raw.models_history === 'object'
    ? {
        points: Array.isArray(raw.models_history.points) ? raw.models_history.points : [],
        models: Array.isArray(raw.models_history.models) ? raw.models_history.models : [],
        buckets: Math.max(0, Math.trunc(finiteNumber(raw.models_history.buckets, 0))),
      }
    : { points: [], models: [], buckets: 0 }

  const vendorShareHistory = raw.vendor_share_history && typeof raw.vendor_share_history === 'object'
    ? {
        points: Array.isArray(raw.vendor_share_history.points) ? raw.vendor_share_history.points : [],
        vendors: Array.isArray(raw.vendor_share_history.vendors) ? raw.vendor_share_history.vendors : [],
        buckets: Math.max(0, Math.trunc(finiteNumber(raw.vendor_share_history.buckets, 0))),
      }
    : { points: [], vendors: [], buckets: 0 }

  return {
    models,
    vendors,
    top_movers: normalizeMovers(raw.top_movers),
    top_droppers: normalizeMovers(raw.top_droppers),
    models_history: modelsHistory,
    vendor_share_history: vendorShareHistory,
  }
}

export function resolveRankChange(rank: number, previousRank?: number | null): RankChange {
  if (!Number.isFinite(previousRank) || !previousRank || previousRank <= 0) {
    return { kind: 'new', delta: 0 }
  }
  const diff = Math.trunc(previousRank) - Math.trunc(rank)
  if (diff > 0) return { kind: 'up', delta: diff }
  if (diff < 0) return { kind: 'down', delta: Math.abs(diff) }
  return { kind: 'same', delta: 0 }
}

export function formatCompactTokens(tokens: number): string {
  if (!Number.isFinite(tokens) || tokens <= 0) return '0'
  if (tokens >= 1e12) return `${(tokens / 1e12).toFixed(2).replace(/\.?0+$/, '')}T`
  if (tokens >= 1e9) return `${(tokens / 1e9).toFixed(2).replace(/\.?0+$/, '')}B`
  if (tokens >= 1e6) return `${(tokens / 1e6).toFixed(1).replace(/\.0$/, '')}M`
  if (tokens >= 1e3) return `${(tokens / 1e3).toFixed(1).replace(/\.0$/, '')}K`
  return Math.round(tokens).toLocaleString('en-US')
}

export function formatSharePercent(share: number): string {
  if (!Number.isFinite(share) || share <= 0) return '0%'
  const pct = share * 100
  if (pct < 0.1) return '<0.1%'
  return `${pct.toFixed(1)}%`
}

export function formatGrowthPercent(pct: number): string {
  if (!Number.isFinite(pct) || Math.abs(pct) < 0.05) return '0.0%'
  const clamped = Math.max(-999.9, Math.min(999.9, pct))
  const sign = clamped > 0 ? '+' : ''
  return `${sign}${clamped.toFixed(1)}%`
}

/**
 * 从已加载的模型广场报价列表中匹配排行榜模型的首选报价（优先 default 分组，其次最低倍率）。
 */
export function findBestOfferForModel(offers: readonly ModelOffer[], modelName: string): ModelOffer | null {
  const target = String(modelName || '').trim()
  if (!target || offers.length === 0) return null
  let best: ModelOffer | null = null
  for (const offer of offers) {
    if (offer.modelId !== target) continue
    if (!best) {
      best = offer
      continue
    }
    const bestIsDefault = best.group === 'default'
    const candidateIsDefault = offer.group === 'default'
    if (candidateIsDefault && !bestIsDefault) {
      best = offer
      continue
    }
    if (!bestIsDefault && offer.groupRatio < best.groupRatio) {
      best = offer
    }
  }
  return best
}
