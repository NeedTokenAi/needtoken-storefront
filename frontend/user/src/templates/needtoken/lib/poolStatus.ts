/**
 * 号池可用状态（来自店面同源缓存 GET /api/v1/public/pool-status -> Glue /api/public/pool-status）。
 * 胶水只公开状态等级，不公开账号数；未登记或拿不到数据的分组一律按“待接入”展示。
 */

export type PoolLevel = 'available' | 'busy' | 'unavailable' | 'unknown'

export interface PoolStatusResponse {
    ok?: boolean
    enabled?: boolean
    stale?: boolean
    updated_at?: string
    groups?: Record<string, { status?: string }> | unknown[]
}

export interface PoolStatusView {
    enabled: boolean
    stale: boolean
    updatedAt: Date | null
    levels: Record<string, PoolLevel>
}

export interface PoolStatusSummary {
    enabled: boolean
    totalGroups: number
    healthyGroups: number
    overall: PoolLevel
}

const LEVELS: PoolLevel[] = ['available', 'busy', 'unavailable', 'unknown']

export const parsePoolStatus = (body: PoolStatusResponse | null | undefined): PoolStatusView => {
    const levels: Record<string, PoolLevel> = {}
    if (body?.groups && typeof body.groups === 'object' && !Array.isArray(body.groups)) {
        for (const [group, value] of Object.entries(body.groups)) {
            const level = value?.status as PoolLevel
            levels[group] = LEVELS.includes(level) ? level : 'unknown'
        }
    }
    const updatedAt = body?.updated_at ? new Date(body.updated_at) : null
    return {
        enabled: Boolean(body?.enabled),
        stale: Boolean(body?.stale),
        updatedAt: updatedAt && !Number.isNaN(updatedAt.getTime()) ? updatedAt : null,
        levels,
    }
}

/** 某分组要展示的状态；没有可信数据时返回 null（界面显示“状态待接入”） */
export const levelFor = (view: PoolStatusView | null, group: string): PoolLevel | null => {
    if (!view || !view.enabled) return null
    const level = view.levels[group]
    return level && level !== 'unknown' ? level : null
}

/** 聚合号池整体健康快照，供 API 服务页指标条展示 */
export const summarizePoolStatus = (view: PoolStatusView | null): PoolStatusSummary => {
    if (!view || !view.enabled) {
        return { enabled: false, totalGroups: 0, healthyGroups: 0, overall: 'unknown' }
    }
    const entries = Object.values(view.levels).filter(level => level !== 'unknown')
    const availableCount = entries.filter(level => level === 'available').length
    const busyCount = entries.filter(level => level === 'busy').length
    const healthyGroups = availableCount + busyCount
    const overall: PoolLevel = availableCount > 0
        ? 'available'
        : busyCount > 0
            ? 'busy'
            : entries.length > 0
                ? 'unavailable'
                : 'unknown'
    return {
        enabled: true,
        totalGroups: entries.length,
        healthyGroups,
        overall,
    }
}
