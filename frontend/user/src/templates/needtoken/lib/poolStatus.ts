/**
 * 号池可用状态（来自胶水 GET /api/public/pool-status）。
 * 胶水只公开状态等级，不公开账号数；未登记或拿不到数据的分组一律按“待接入”展示。
 */

export type PoolLevel = 'available' | 'busy' | 'unavailable' | 'unknown'

export interface PoolStatusResponse {
    ok?: boolean
    enabled?: boolean
    stale?: boolean
    updated_at?: string
    groups?: Record<string, { status?: string }>
}

export interface PoolStatusView {
    enabled: boolean
    stale: boolean
    updatedAt: Date | null
    levels: Record<string, PoolLevel>
}

const LEVELS: PoolLevel[] = ['available', 'busy', 'unavailable', 'unknown']

export const parsePoolStatus = (body: PoolStatusResponse | null | undefined): PoolStatusView => {
    const levels: Record<string, PoolLevel> = {}
    for (const [group, value] of Object.entries(body?.groups ?? {})) {
        const level = value?.status as PoolLevel
        levels[group] = LEVELS.includes(level) ? level : 'unknown'
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
