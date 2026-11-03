/**
 * 模型广场的价格换算与筛选（纯函数，不依赖 Vue / 网络，便于单测）。
 *
 * 数据源是 new-api 的 GET /api/pricing。换算公式与 new-api 控制台一致
 * （web/src/features/pricing/lib/price.ts）：
 *   输入价（美元 / 百万 tokens）= model_ratio × 2 × 分组倍率
 *   输出价 = 输入价 × completion_ratio
 *   按次计费（quota_type = 1）= model_price × 分组倍率（美元 / 次）
 */

export interface PricingModel {
    model_name: string
    quota_type: number
    model_ratio: number
    model_price: number
    completion_ratio: number
    vendor_id?: number
    owner_by?: string
    enable_groups?: string[]
    tags?: string
    description?: string
    billing_mode?: string
    billing_expr?: string
    billing_plugin_variants?: { plugin_key: string; plugin_name?: string; billing_mode?: string; billing_expr?: string }[]
    billing_usage_schema?: Record<string, { type?: string; unit?: string }>
    supported_endpoint_types?: string[]
    pricing_version?: string
    cache_ratio?: number | null
    create_cache_ratio?: number | null
    image_ratio?: number | null
    audio_ratio?: number | null
    audio_completion_ratio?: number | null
}

export interface PricingVendor {
    id: number
    name: string
}

export interface PricingResponse {
    success?: boolean
    data?: PricingModel[]
    vendors?: PricingVendor[]
    group_ratio?: Record<string, number>
    pricing_version?: string
}

/**
 * new-api 对未配置倍率的模型返回 37.5（GetModelRatio 的兜底值，自用模式下照样放行）。
 * 接口本身分不出“配了 37.5”和“没配”，这里一律按未配价处理，宁可不展示也不写假价。
 */
export const UNCONFIGURED_MODEL_RATIO = 37.5

export type PriceUnit = 'per_million_tokens' | 'per_call' | 'conditional'
export type PricingCondition = 'expression' | 'plugin' | 'task' | 'multimodal' | 'unknown'

export interface ModelOffer {
    /** 唯一键：模型 + 分组 */
    key: string
    modelId: string
    vendor: string
    group: string
    /** 供应线：已登记的分组用登记值，否则为 null（界面上显示分组名） */
    supplyLine: string | null
    unit: PriceUnit
    /** 美元；per_call 时 output 为 null */
    inputPrice: number | null
    outputPrice: number | null
    isDefault: boolean
    condition: PricingCondition | null
    /** Backend cache/schema marker; never treated as an immutable checkout quote. */
    sourceVersion: string | null
    usageUnits: string[]
}

export interface BuildOffersOptions {
    defaultModels: readonly string[]
    blockedModels: readonly string[]
    supplyLineByGroup: Record<string, string>
}

export const pricingCondition = (model: PricingModel): PricingCondition | null => {
    if (model.billing_plugin_variants?.length) return 'plugin'
    if (String(model.billing_expr || '').trim() || (model.billing_mode && model.billing_mode !== 'ratio')) return 'expression'
    if (Object.keys(model.billing_usage_schema || {}).length || model.supported_endpoint_types?.some(type => ['openai-video', 'image-generation'].includes(type))) return 'task'
    if ([model.cache_ratio, model.create_cache_ratio, model.image_ratio, model.audio_ratio, model.audio_completion_ratio].some(value => value !== undefined && value !== null)) return 'multimodal'
    if (model.quota_type !== 0 && model.quota_type !== 1) return 'unknown'
    return null
}

export const isModelPriced = (model: PricingModel): boolean => {
    // Never interpret expression/plugin/task prices through the legacy token formula.
    if (pricingCondition(model)) return true
    if (model.quota_type === 1) {
        return Number.isFinite(model.model_price) && model.model_price > 0
    }
    return (
        Number.isFinite(model.model_ratio)
        && model.model_ratio > 0
        && model.model_ratio !== UNCONFIGURED_MODEL_RATIO
        && Number.isFinite(model.completion_ratio) && model.completion_ratio >= 0
    )
}

const resolveVendor = (model: PricingModel, vendors: Map<number, string>): string => {
    if (model.vendor_id && vendors.has(model.vendor_id)) return vendors.get(model.vendor_id) as string
    const owner = String(model.owner_by || '').trim()
    return owner || 'other'
}

/**
 * 把 new-api 定价表展开成“模型 × 分组”的报价列表。
 * 同一模型在不同分组（供应线）价格不同，分开列出、不混算。
 */
export const buildOffers = (response: PricingResponse, options: BuildOffersOptions): ModelOffer[] => {
    const models = Array.isArray(response.data) ? response.data : []
    const groupRatio = response.group_ratio || {}
    const vendors = new Map<number, string>()
    for (const v of response.vendors || []) {
        if (v && typeof v.id === 'number') vendors.set(v.id, String(v.name || '').trim())
    }
    const blocked = new Set(options.blockedModels)
    const defaults = new Set(options.defaultModels)

    const offers: ModelOffer[] = []
    for (const model of models) {
        if (!model || !model.model_name || blocked.has(model.model_name)) continue
        if (!isModelPriced(model)) continue
        const condition = pricingCondition(model)
        const enabled = model.enable_groups?.includes('all') ? Object.keys(groupRatio) : (model.enable_groups || [])
        const groups = [...new Set(enabled)].filter(g => Number.isFinite(groupRatio[g]) && groupRatio[g]! >= 0)
        for (const group of groups) {
            const ratio = groupRatio[group] as number
            const perCall = model.quota_type === 1
            const input = condition ? null : perCall ? model.model_price * ratio : model.model_ratio * 2 * ratio
            const output = condition || perCall || input === null ? null : input * model.completion_ratio
            const valid = input !== null && Number.isFinite(input) && (output === null || Number.isFinite(output))
            offers.push({
                key: `${model.model_name}@${group}`,
                modelId: model.model_name,
                vendor: resolveVendor(model, vendors),
                group,
                supplyLine: options.supplyLineByGroup[group] ?? null,
                unit: valid ? perCall ? 'per_call' : 'per_million_tokens' : 'conditional',
                inputPrice: valid ? input : null,
                outputPrice: valid ? output : null,
                isDefault: defaults.has(model.model_name),
                condition: condition ?? (valid ? null : 'unknown'),
                sourceVersion: model.pricing_version || response.pricing_version || null,
                usageUnits: [...new Set(Object.entries(model.billing_usage_schema || {}).map(([key, field]) => `${key}${field.unit ? ` (${field.unit})` : ''}`))],
            })
        }
    }
    return offers
}

export interface OfferFilter {
    vendor?: string
    /** 供应线值；未登记分组用 `group:<分组名>` 表示 */
    supply?: string
    keyword?: string
}

export const supplyKeyOf = (offer: ModelOffer): string => offer.supplyLine ?? `group:${offer.group}`

export const filterOffers = (offers: ModelOffer[], filter: OfferFilter): ModelOffer[] => {
    const keyword = String(filter.keyword || '').trim().toLowerCase()
    return offers.filter(o =>
        (!filter.vendor || o.vendor === filter.vendor)
        && (!filter.supply || supplyKeyOf(o) === filter.supply)
        && (!keyword || o.modelId.toLowerCase().includes(keyword)),
    )
}

export type OfferSort = 'default' | 'price_asc' | 'price_desc'

/** 默认排序：默认模型按白名单顺序在前，其余按模型名；价格排序只比较同计价单位，按次计费排在最后 */
export const sortOffers = (offers: ModelOffer[], sort: OfferSort, defaultModels: readonly string[]): ModelOffer[] => {
    const list = [...offers]
    if (sort === 'default') {
        const rank = (o: ModelOffer) => {
            const i = defaultModels.indexOf(o.modelId)
            return i === -1 ? defaultModels.length : i
        }
        return list.sort((a, b) => rank(a) - rank(b) || a.modelId.localeCompare(b.modelId) || a.group.localeCompare(b.group))
    }
    const dir = sort === 'price_asc' ? 1 : -1
    return list.sort((a, b) => {
        const units: Record<PriceUnit, number> = { per_million_tokens: 0, per_call: 1, conditional: 2 }
        if (a.unit !== b.unit) return units[a.unit] - units[b.unit]
        if (a.inputPrice === null || b.inputPrice === null) return a.modelId.localeCompare(b.modelId)
        return (a.inputPrice - b.inputPrice) * dir || a.modelId.localeCompare(b.modelId)
    })
}

export const MAX_COMPARE = 3

/** 比价选择：已选则取消；未选且未满则加入；满了不变 */
export const toggleCompare = (selected: string[], key: string): string[] => {
    if (selected.includes(key)) return selected.filter(k => k !== key)
    if (selected.length >= MAX_COMPARE) return selected
    return [...selected, key]
}

/** 美元价格显示：去掉多余的 0，最多 6 位小数 */
export const formatUsd = (value: number | null): string => {
    if (value === null || !Number.isFinite(value) || value < 0) return '-'
    if (value > 0 && value < 0.000001) return '<$0.000001'
    const fixed = value.toFixed(6).replace(/\.?0+$/, '')
    return `$${fixed}`
}
